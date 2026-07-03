package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
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
	repo   *repository.Repository
	redis  *redis.Client
	cfg    *config.Config
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
	r.Methods = strings.ToUpper(r.Methods)
	if r.Methods == "" {
		r.Methods = "ALL"
	}
	return s.repo.CreateRule(ctx, r)
}

func (s *RuleService) Update(ctx context.Context, id uint64, r *models.Rule) error {
	r.Methods = strings.ToUpper(r.Methods)
	if r.Methods == "" {
		r.Methods = "ALL"
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

// ProxyService

type ProxyService struct {
	repo  *repository.Repository
	redis *redis.Client
}

func NewProxyService(repo *repository.Repository, redis *redis.Client) *ProxyService {
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

func (s *ProxyService) EvaluateRules(ctx context.Context, rules []models.Rule, r *http.Request, path string, body []byte, realIP string) (matched []models.Rule, blocked bool, status int, blockResp string, blockedRule *models.Rule, err error) {
	method := r.Method
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		if !strings.HasPrefix(path, rule.PathPrefix) {
			continue
		}
		if rule.Methods != "ALL" && !containsMethod(rule.Methods, method) {
			continue
		}
		identity := buildIdentity(realIP, rule.IdentityFields, body)
		count, err := s.redis.GetAttemptCount(ctx, rule.ID, identity)
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

func (s *ProxyService) RecordAttempts(ctx context.Context, rules []models.Rule, domainID uint64, path, realIP string, body []byte) {
	for _, rule := range rules {
		identity := buildIdentity(realIP, rule.IdentityFields, body)
		_, err := s.redis.IncrementAttempt(ctx, rule.ID, identity, rule.WindowSeconds)
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

func buildIdentity(realIP, fields string, body []byte) string {
	parts := []string{realIP}
	if strings.TrimSpace(fields) != "" && len(body) > 0 && gjson.ValidBytes(body) {
		for _, f := range strings.Split(fields, ",") {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			v := gjson.GetBytes(body, f)
			parts = append(parts, f+"="+v.String())
		}
	}
	return strings.Join(parts, "|")
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
