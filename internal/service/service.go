package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/crypto/bcrypt"

	"ip_check/internal/config"
	"ip_check/internal/models"
	"ip_check/internal/redis"
	"ip_check/internal/repository"
)

var ErrUnauthorized = errors.New("unauthorized")
var ErrNotFound = errors.New("not found")

type AuthService struct {
	repo  *repository.Repository
	redis *redis.Client
	cfg   *config.Config
}

func NewAuthService(repo *repository.Repository, redis *redis.Client, cfg *config.Config) *AuthService {
	return &AuthService{repo: repo, redis: redis, cfg: cfg}
}

func (s *AuthService) HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *AuthService) GeneratePassword(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b), nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return "", ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrUnauthorized
	}
	return s.generateToken(user.Username)
}

func (s *AuthService) Logout(ctx context.Context, tokenString string) error {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return err
	}
	jti, _ := claims["jti"].(string)
	exp, _ := claims["exp"].(float64)
	if jti == "" {
		return nil
	}
	ttl := time.Until(time.Unix(int64(exp), 0))
	if ttl <= 0 {
		return nil
	}
	return s.redis.BlacklistJWT(ctx, jti, ttl)
}

func (s *AuthService) ValidateToken(ctx context.Context, tokenString string) (string, error) {
	claims, err := s.parseToken(tokenString)
	if err != nil {
		return "", err
	}
	jti, _ := claims["jti"].(string)
	if jti != "" {
		blacklisted, err := s.redis.IsJWTBlacklisted(ctx, jti)
		if err != nil {
			return "", err
		}
		if blacklisted {
			return "", ErrUnauthorized
		}
	}
	username, _ := claims["sub"].(string)
	if username == "" {
		return "", ErrUnauthorized
	}
	return username, nil
}

func (s *AuthService) generateToken(username string) (string, error) {
	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": username,
		"jti": base64.RawURLEncoding.EncodeToString(jti),
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(s.cfg.JWTExpireHours) * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *AuthService) parseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrUnauthorized
}

func (s *AuthService) SeedAdmin(password string) (string, error) {
	ctx := context.Background()
	count, err := s.repo.UserCount(ctx)
	if err != nil {
		return "", err
	}
	if count > 0 {
		return "", nil
	}
	pwd := password
	if pwd == "" {
		var err error
		pwd, err = s.GeneratePassword(16)
		if err != nil {
			return "", err
		}
	}
	hash, err := s.HashPassword(pwd)
	if err != nil {
		return "", err
	}
	if err := s.repo.CreateUser(ctx, &models.User{Username: s.cfg.AdminUsername, PasswordHash: hash}); err != nil {
		return "", err
	}
	return pwd, nil
}

// DomainService

type DomainService struct {
	repo *repository.Repository
}

func NewDomainService(repo *repository.Repository) *DomainService {
	return &DomainService{repo: repo}
}

func (s *DomainService) Create(ctx context.Context, d *models.Domain) error {
	if d.RealIPHeaders == "" {
		d.RealIPHeaders = "X-Real-IP,X-Forwarded-For,CF-Connecting-IP"
	}
	if d.ForwardIPHeader == "" {
		d.ForwardIPHeader = "X-Forwarded-For"
	}
	if d.RequestTransform == nil {
		d.RequestTransform = models.JSONB("[]")
	}
	if d.ResponseTransform == nil {
		d.ResponseTransform = models.JSONB("[]")
	}
	if d.RewriteMode == "" {
		d.RewriteMode = "full"
	}
	if d.IsDefault {
		if err := s.repo.ClearDefaultDomain(ctx, 0); err != nil {
			return err
		}
	}
	return s.repo.CreateDomain(ctx, d)
}

func (s *DomainService) Update(ctx context.Context, id uint64, d *models.Domain) error {
	if d.RequestTransform == nil {
		d.RequestTransform = models.JSONB("[]")
	}
	if d.ResponseTransform == nil {
		d.ResponseTransform = models.JSONB("[]")
	}
	if d.RewriteMode == "" {
		d.RewriteMode = "full"
	}
	if d.IsDefault {
		if err := s.repo.ClearDefaultDomain(ctx, id); err != nil {
			return err
		}
	}
	return s.repo.UpdateDomain(ctx, id, d)
}

func (s *DomainService) Delete(ctx context.Context, id uint64) error {
	return s.repo.DeleteDomain(ctx, id)
}

func (s *DomainService) GetByID(ctx context.Context, id uint64) (*models.Domain, error) {
	return s.repo.GetDomainByID(ctx, id)
}

func (s *DomainService) List(ctx context.Context) ([]models.Domain, error) {
	return s.repo.ListDomains(ctx)
}

// RuleService

type RuleService struct {
	repo  *repository.Repository
	redis *redis.Client
}

func NewRuleService(repo *repository.Repository, redis *redis.Client) *RuleService {
	return &RuleService{repo: repo, redis: redis}
}

func (s *RuleService) Create(ctx context.Context, r *models.Rule) error {
	r.PathPrefix, r.QueryMatch = normalizeRulePathQuery(r.PathPrefix, r.QueryMatch)
	r.Methods = strings.ToUpper(r.Methods)
	if r.Methods == "" {
		r.Methods = "ALL"
	}
	if strings.TrimSpace(r.SuccessStatuses) == "" {
		r.SuccessStatuses = "2xx"
	}
	return s.repo.CreateRule(ctx, r)
}

func (s *RuleService) Update(ctx context.Context, id uint64, r *models.Rule) error {
	r.PathPrefix, r.QueryMatch = normalizeRulePathQuery(r.PathPrefix, r.QueryMatch)
	r.Methods = strings.ToUpper(r.Methods)
	if r.Methods == "" {
		r.Methods = "ALL"
	}
	if strings.TrimSpace(r.SuccessStatuses) == "" {
		r.SuccessStatuses = "2xx"
	}
	return s.repo.UpdateRule(ctx, id, r)
}

func (s *RuleService) Delete(ctx context.Context, id uint64) error {
	return s.repo.DeleteRule(ctx, id)
}

func (s *RuleService) GetByID(ctx context.Context, id uint64) (*models.Rule, error) {
	return s.repo.GetRuleByID(ctx, id)
}

func (s *RuleService) ListByDomain(ctx context.Context, domainID uint64) ([]models.Rule, error) {
	return s.repo.ListRulesByDomain(ctx, domainID)
}

func (s *RuleService) ListFirewallBlacklist(ctx context.Context) ([]redis.AttemptEntry, error) {
	list, err := s.repo.ListFirewallAttempts(ctx)
	if err != nil {
		return nil, err
	}
	entries := make([]redis.AttemptEntry, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, item := range list {
		entry := firewallAttemptEntry(item)
		entries = append(entries, entry)
		seen[entry.Key] = struct{}{}
	}

	cachedEntries, err := s.redis.ListAttempts(ctx)
	if err != nil {
		return entries, nil
	}
	for _, entry := range cachedEntries {
		if _, ok := seen[entry.Key]; ok || entry.RuleID == 0 || entry.Identity == "" {
			continue
		}
		entries = append(entries, entry)
		seen[entry.Key] = struct{}{}
		_ = s.repo.SaveFirewallAttemptCount(ctx, entry.RuleID, entry.Identity, entry.Count, entry.TTLSeconds)
	}
	return entries, nil
}

func (s *RuleService) DeleteFirewallBlacklistEntry(ctx context.Context, key string) (bool, error) {
	redisDeleted, err := s.redis.DeleteAttempt(ctx, key)
	if err != nil {
		return false, err
	}
	dbDeleted, err := s.repo.DeleteFirewallAttemptByKey(ctx, key)
	if err != nil {
		return false, err
	}
	return redisDeleted || dbDeleted, nil
}

func (s *RuleService) ClearFirewallBlacklist(ctx context.Context) (int64, error) {
	redisDeleted, err := s.redis.ClearAttempts(ctx)
	if err != nil {
		return redisDeleted, err
	}
	dbDeleted, err := s.repo.ClearFirewallAttempts(ctx)
	if err != nil {
		return redisDeleted, err
	}
	if dbDeleted > redisDeleted {
		return dbDeleted, nil
	}
	return redisDeleted, nil
}

// ProxyService

type ProxyService struct {
	repo  *repository.Repository
	redis attemptStore
}

type attemptStore interface {
	IncrementAttempt(ctx context.Context, ruleID uint64, identity string, windowSeconds int) (int64, error)
	GetAttemptCount(ctx context.Context, ruleID uint64, identity string) (int64, error)
	SetAttemptCount(ctx context.Context, ruleID uint64, identity string, count int64, ttlSeconds int) error
}

func NewProxyService(repo *repository.Repository, redis attemptStore) *ProxyService {
	return &ProxyService{repo: repo, redis: redis}
}

func (s *ProxyService) GetDomainByHost(ctx context.Context, host string) (*models.Domain, error) {
	return s.repo.GetDomainByBind(ctx, host)
}

func (s *ProxyService) GetDefaultDomain(ctx context.Context) (*models.Domain, error) {
	return s.repo.GetDefaultDomain(ctx)
}

func (s *ProxyService) ListRulesByDomain(ctx context.Context, domainID uint64) ([]models.Rule, error) {
	return s.repo.ListRulesByDomain(ctx, domainID)
}

const maxBodySize = 10 << 20

func firewallAttemptEntry(item models.FirewallAttempt) redis.AttemptEntry {
	return redis.AttemptEntry{
		Key:        fmt.Sprintf("attempt:%d:%s", item.RuleID, item.Identity),
		RuleID:     item.RuleID,
		Identity:   item.Identity,
		Count:      item.Count,
		TTLSeconds: firewallAttemptTTL(item.ExpiresAt),
	}
}

func firewallAttemptTTL(expiresAt *time.Time) int64 {
	if expiresAt == nil {
		return -1
	}
	duration := time.Until(*expiresAt)
	if duration <= 0 {
		return -2
	}
	return int64((duration + time.Second - 1) / time.Second)
}

func (s *ProxyService) attemptCount(ctx context.Context, ruleID uint64, identity string) (int64, error) {
	count, err := s.redis.GetAttemptCount(ctx, ruleID, identity)
	if err != nil || count > 0 || s.repo == nil {
		return count, err
	}
	persistedCount, ttlSeconds, err := s.repo.GetFirewallAttemptCount(ctx, ruleID, identity)
	if err != nil || persistedCount == 0 {
		return persistedCount, err
	}
	_ = s.redis.SetAttemptCount(ctx, ruleID, identity, persistedCount, ttlSeconds)
	return persistedCount, nil
}

func (s *ProxyService) EvaluateRules(ctx context.Context, rules []models.Rule, r *http.Request, path string, body []byte, realIP string) (matched []models.Rule, blocked bool, status int, blockResp string, blockedRule *models.Rule, err error) {
	method := r.Method
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		pathPrefix, queryMatch := normalizeRulePathQuery(rule.PathPrefix, rule.QueryMatch)
		if !strings.HasPrefix(path, pathPrefix) {
			continue
		}
		if !queryMatches(queryMatch, r.URL.Query()) {
			continue
		}
		if rule.Methods != "ALL" && !containsMethod(rule.Methods, method) {
			continue
		}
		identity := buildIdentity(realIP, rule.IdentityFields, body, r.Header.Get("Content-Type"))
		count, err := s.attemptCount(ctx, rule.ID, identity)
		if err != nil {
			return nil, false, 0, "", nil, err
		}
		if int(count) >= rule.MaxAttempts {
			resp := string(rule.BlockResponse)
			if resp == "" || resp == "null" {
				resp = fmt.Sprintf(`{"code":%d,"message":"blocked"}`, rule.BlockStatus)
			}
			resp = injectRuleInfo(resp, rule)
			return nil, true, rule.BlockStatus, resp, &rule, nil
		}
		matched = append(matched, rule)
	}
	return matched, false, 0, "", nil, nil
}

func injectRuleInfo(resp string, rule models.Rule) string {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &data); err != nil {
		return resp
	}
	if _, ok := data["rule_name"]; !ok {
		data["rule_name"] = rule.Name
	}
	if _, ok := data["rule_type"]; !ok {
		data["rule_type"] = rule.RuleType
	}
	b, _ := json.Marshal(data)
	return string(b)
}

func (s *ProxyService) RecordAttempts(ctx context.Context, rules []models.Rule, domainID uint64, path, realIP string, body []byte, contentType string) {
	_ = ctx
	recordCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for _, rule := range rules {
		identity := buildIdentity(realIP, rule.IdentityFields, body, contentType)
		if s.repo != nil {
			persistedCount, err := s.repo.IncrementFirewallAttempt(recordCtx, rule.ID, identity, rule.WindowSeconds)
			if err == nil {
				_ = s.redis.SetAttemptCount(recordCtx, rule.ID, identity, persistedCount, rule.WindowSeconds)
				_ = domainID
				_ = path
				continue
			}
		}
		_, err := s.redis.IncrementAttempt(recordCtx, rule.ID, identity, rule.WindowSeconds)
		if err != nil {
			// log
		}
		_ = domainID
		_ = path
	}
}

func (s *ProxyService) Log(ctx context.Context, log *models.ProxyLog) error {
	return s.repo.CreateLog(ctx, log)
}

func (s *ProxyService) ListLogs(ctx context.Context, page, pageSize int) ([]models.ProxyLog, int64, error) {
	return s.repo.ListLogs(ctx, page, pageSize)
}

func (s *ProxyService) ListBlockedLogs(ctx context.Context, page, pageSize int) ([]models.ProxyLog, int64, error) {
	return s.repo.ListBlockedLogs(ctx, page, pageSize)
}

func (s *ProxyService) StatsBlocked(ctx context.Context) (*models.BlockedStats, error) {
	stats, err := s.repo.StatsBlocked(ctx)
	if err != nil {
		return nil, err
	}

	// enrich top rules with rule names
	if len(stats.TopRules) > 0 {
		ruleIDs := make([]uint64, 0, len(stats.TopRules))
		for _, r := range stats.TopRules {
			ruleIDs = append(ruleIDs, r.RuleID)
		}
		var rules []models.Rule
		if err := s.repo.ListRulesByIDs(ctx, ruleIDs, &rules); err == nil {
			nameMap := make(map[uint64]string, len(rules))
			for _, r := range rules {
				nameMap[r.ID] = r.Name
			}
			for i := range stats.TopRules {
				if name, ok := nameMap[stats.TopRules[i].RuleID]; ok {
					stats.TopRules[i].RuleName = name
				} else {
					stats.TopRules[i].RuleName = "未知规则"
				}
			}
		}
	}

	return stats, nil
}

func (s *ProxyService) ShouldRecordSuccess(rule models.Rule, status int, location string) bool {
	if !statusMatches(rule.SuccessStatuses, status) {
		return false
	}
	if strings.TrimSpace(rule.FailureLocationMatch) != "" && locationQueryMatches(rule.FailureLocationMatch, location) {
		return false
	}
	if strings.TrimSpace(rule.SuccessLocationMatch) != "" {
		return locationQueryMatches(rule.SuccessLocationMatch, location)
	}
	return true
}

func (s *ProxyService) TransformBody(body []byte, contentType string, transform models.JSONB) ([]byte, error) {
	if len(body) == 0 || len(transform) == 0 || string(transform) == "null" || string(transform) == "[]" {
		return body, nil
	}
	if !strings.Contains(strings.ToLower(contentType), "json") {
		return body, nil
	}
	if !gjson.ValidBytes(body) {
		return body, nil
	}
	var mappings []struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	}
	if err := json.Unmarshal(transform, &mappings); err != nil {
		return body, err
	}
	for _, m := range mappings {
		m.Src = strings.TrimSpace(m.Src)
		m.Dst = strings.TrimSpace(m.Dst)
		if m.Src == "" || m.Dst == "" {
			continue
		}
		v := gjson.GetBytes(body, m.Src)
		if !v.Exists() {
			continue
		}
		body, _ = sjson.SetRawBytes(body, m.Dst, []byte(v.Raw))
	}
	return body, nil
}

func buildIdentity(realIP, fields string, body []byte, contentType string) string {
	parts := []string{realIP}
	if strings.TrimSpace(fields) == "" || len(body) == 0 {
		return strings.Join(parts, "|")
	}
	formValues := parseFormValues(body, contentType)
	for _, f := range strings.Split(fields, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		parts = append(parts, f+"="+identityFieldValue(body, formValues, f))
	}
	return strings.Join(parts, "|")
}

func identityFieldValue(body []byte, formValues url.Values, field string) string {
	if len(formValues) > 0 {
		if v := formValues.Get(field); v != "" {
			return v
		}
		return formValues.Get(dotPathToBracketPath(field))
	}
	if gjson.ValidBytes(body) {
		return gjson.GetBytes(body, field).String()
	}
	return ""
}

func parseFormValues(body []byte, contentType string) url.Values {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	}
	switch strings.ToLower(mediaType) {
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil
		}
		return values
	case "multipart/form-data":
		boundary := params["boundary"]
		if boundary == "" {
			return nil
		}
		reader := multipart.NewReader(bytes.NewReader(body), boundary)
		form, err := reader.ReadForm(maxBodySize)
		if err != nil {
			return nil
		}
		defer form.RemoveAll()
		return form.Value
	default:
		return nil
	}
}

func dotPathToBracketPath(path string) string {
	parts := strings.Split(path, ".")
	if len(parts) <= 1 {
		return path
	}
	var b strings.Builder
	b.WriteString(parts[0])
	for _, part := range parts[1:] {
		b.WriteString("[")
		b.WriteString(part)
		b.WriteString("]")
	}
	return b.String()
}

func normalizeRulePathQuery(pathPrefix, queryMatch string) (string, string) {
	pathPrefix = strings.TrimSpace(pathPrefix)
	queryMatch = strings.TrimSpace(strings.TrimPrefix(queryMatch, "?"))
	if pathPrefix == "" {
		return "/", queryMatch
	}

	parsed, err := url.Parse(pathPrefix)
	if err != nil {
		parts := strings.SplitN(pathPrefix, "?", 2)
		pathPrefix = parts[0]
		if len(parts) == 2 {
			queryMatch = mergeQueryMatch(parts[1], queryMatch)
		}
		return normalizePathPrefix(pathPrefix), queryMatch
	}

	if parsed.Path != "" {
		pathPrefix = parsed.Path
	}
	if parsed.RawQuery != "" {
		queryMatch = mergeQueryMatch(parsed.RawQuery, queryMatch)
	}
	return normalizePathPrefix(pathPrefix), queryMatch
}

func normalizePathPrefix(pathPrefix string) string {
	pathPrefix = strings.TrimSpace(pathPrefix)
	if pathPrefix == "" {
		return "/"
	}
	if !strings.HasPrefix(pathPrefix, "/") {
		return "/" + pathPrefix
	}
	return pathPrefix
}

func mergeQueryMatch(fromPath, configured string) string {
	fromPath = strings.TrimSpace(strings.TrimPrefix(fromPath, "?"))
	configured = strings.TrimSpace(strings.TrimPrefix(configured, "?"))
	if fromPath == "" {
		return configured
	}
	if configured == "" || configured == fromPath {
		return fromPath
	}
	return fromPath + "&" + configured
}

func queryMatches(match string, values url.Values) bool {
	match = strings.TrimSpace(strings.TrimPrefix(match, "?"))
	if match == "" {
		return true
	}
	expected, err := url.ParseQuery(match)
	if err != nil || len(expected) == 0 {
		return false
	}
	for key, wantValues := range expected {
		if len(wantValues) == 0 {
			if _, ok := values[key]; !ok {
				return false
			}
			continue
		}
		gotValues, ok := values[key]
		if !ok {
			return false
		}
		for _, want := range wantValues {
			if !containsString(gotValues, want) {
				return false
			}
		}
	}
	return true
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func statusMatches(pattern string, status int) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		pattern = "2xx"
	}
	for _, token := range strings.Split(pattern, ",") {
		token = strings.ToLower(strings.TrimSpace(token))
		if token == "" {
			continue
		}
		if len(token) == 3 && token[1:] == "xx" {
			prefix, err := strconv.Atoi(token[:1])
			if err == nil && status/100 == prefix {
				return true
			}
			continue
		}
		if strings.Contains(token, "-") {
			parts := strings.SplitN(token, "-", 2)
			min, minErr := strconv.Atoi(strings.TrimSpace(parts[0]))
			max, maxErr := strconv.Atoi(strings.TrimSpace(parts[1]))
			if minErr == nil && maxErr == nil && status >= min && status <= max {
				return true
			}
			continue
		}
		code, err := strconv.Atoi(token)
		if err == nil && status == code {
			return true
		}
	}
	return false
}

func locationQueryMatches(match, location string) bool {
	location = strings.TrimSpace(location)
	if location == "" {
		return false
	}
	u, err := url.Parse(location)
	if err != nil {
		return false
	}
	return queryMatches(match, u.Query())
}

func containsMethod(list, method string) bool {
	for _, m := range strings.Split(list, ",") {
		if strings.EqualFold(strings.TrimSpace(m), method) {
			return true
		}
	}
	return false
}

func RealIP(r *http.Request, configuredHeaders string) string {
	headers := strings.Split(configuredHeaders, ",")
	if configuredHeaders == "" {
		headers = []string{"X-Real-IP", "X-Forwarded-For", "CF-Connecting-IP"}
	}
	for _, h := range headers {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		for _, v := range r.Header.Values(h) {
			if strings.EqualFold(h, "X-Forwarded-For") {
				for _, part := range strings.Split(v, ",") {
					ip := strings.TrimSpace(part)
					if isValidIP(ip) {
						return ip
					}
				}
			} else {
				ip := strings.TrimSpace(v)
				if isValidIP(ip) {
					return ip
				}
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isValidIP(s string) bool {
	if s == "" {
		return false
	}
	return net.ParseIP(s) != nil
}

func SetForwardedIP(r *http.Request, header, ip string) {
	if ip == "" {
		return
	}
	r.Header.Set("X-Real-IP", ip)
	if header == "" {
		return
	}
	if strings.EqualFold(header, "X-Forwarded-For") {
		existing := r.Header.Get("X-Forwarded-For")
		if existing == "" {
			r.Header.Set("X-Forwarded-For", ip)
		} else {
			r.Header.Set("X-Forwarded-For", existing+", "+ip)
		}
	} else {
		r.Header.Set(header, ip)
	}
}

func NewReverseProxy(target *url.URL, bindDomain, realIP, forwardHeader string, rewriteHost bool) *httputil.ReverseProxy {
	if forwardHeader == "" {
		forwardHeader = "X-Forwarded-For"
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if rewriteHost {
			req.Host = target.Host
		}
		if bindDomain != "" {
			req.Header.Set("X-Forwarded-Host", bindDomain)
		}
		SetForwardedIP(req, forwardHeader, realIP)
		// Defensive: remove any empty header keys that would be rejected by net/http.
		for k := range req.Header {
			if k == "" {
				req.Header.Del(k)
			}
		}
	}
	return proxy
}

func ReadBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, maxBodySize))
}

func (s *ProxyService) ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
