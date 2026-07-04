package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ip_check/internal/models"
	"ip_check/internal/service"
)

type Handler struct {
	authService   *service.AuthService
	domainService *service.DomainService
	ruleService   *service.RuleService
	proxyService  *service.ProxyService
}

func New(authService *service.AuthService, domainService *service.DomainService, ruleService *service.RuleService, proxyService *service.ProxyService) *Handler {
	return &Handler{
		authService:   authService,
		domainService: domainService,
		ruleService:   ruleService,
		proxyService:  proxyService,
	}
}

func (h *Handler) RegisterAdmin(r *gin.Engine, authMiddleware gin.HandlerFunc) {
	admin := r.Group("/api/admin")
	{
		admin.POST("/login", h.login)
		admin.POST("/logout", authMiddleware, h.logout)
		admin.GET("/me", authMiddleware, h.me)

		admin.GET("/domains", authMiddleware, h.listDomains)
		admin.POST("/domains", authMiddleware, h.createDomain)
		admin.GET("/domains/:id", authMiddleware, h.getDomain)
		admin.PUT("/domains/:id", authMiddleware, h.updateDomain)
		admin.DELETE("/domains/:id", authMiddleware, h.deleteDomain)

		admin.GET("/rules", authMiddleware, h.listRules)
		admin.POST("/rules", authMiddleware, h.createRule)
		admin.GET("/rules/:id", authMiddleware, h.getRule)
		admin.PUT("/rules/:id", authMiddleware, h.updateRule)
		admin.DELETE("/rules/:id", authMiddleware, h.deleteRule)

		admin.GET("/logs", authMiddleware, h.listLogs)
		admin.GET("/logs/stats", authMiddleware, h.statsLogs)
		admin.GET("/blocks", authMiddleware, h.listBlockedLogs)
		admin.GET("/firewall/blacklist", authMiddleware, h.listFirewallBlacklist)
		admin.DELETE("/firewall/blacklist", authMiddleware, h.deleteFirewallBlacklistEntry)
		admin.POST("/firewall/blacklist/clear", authMiddleware, h.clearFirewallBlacklist)
	}
}

func (h *Handler) login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	token, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "invalid credentials"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"access_token": token, "token_type": "bearer"}})
}

func (h *Handler) logout(c *gin.Context) {
	token := extractToken(c)
	if token != "" {
		_ = h.authService.Logout(c.Request.Context(), token)
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

func (h *Handler) me(c *gin.Context) {
	username, _ := c.Get("username")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"username": username}})
}

func extractToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

// Domains

func (h *Handler) listDomains(c *gin.Context) {
	list, err := h.domainService.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

func (h *Handler) createDomain(c *gin.Context) {
	var d models.Domain
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.domainService.Create(c.Request.Context(), &d); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": d})
}

func (h *Handler) getDomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	d, err := h.domainService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": d})
}

func (h *Handler) updateDomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var d models.Domain
	if err := c.ShouldBindJSON(&d); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.domainService.Update(c.Request.Context(), id, &d); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

func (h *Handler) deleteDomain(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.domainService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

// Rules

func (h *Handler) listRules(c *gin.Context) {
	domainID, _ := strconv.ParseUint(c.Query("domain_id"), 10, 64)
	if domainID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "domain_id required"})
		return
	}
	list, err := h.ruleService.ListByDomain(c.Request.Context(), domainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

func (h *Handler) createRule(c *gin.Context) {
	var r models.Rule
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.ruleService.Create(c.Request.Context(), &r); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": r})
}

func (h *Handler) getRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	r, err := h.ruleService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": r})
}

func (h *Handler) updateRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var r models.Rule
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.ruleService.Update(c.Request.Context(), id, &r); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

func (h *Handler) deleteRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.ruleService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

// Logs

func (h *Handler) listLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := h.proxyService.ListLogs(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list, "total": total, "page": page, "page_size": pageSize}})
}

func (h *Handler) statsLogs(c *gin.Context) {
	stats, err := h.proxyService.StatsBlocked(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": stats})
}

func (h *Handler) listBlockedLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	list, total, err := h.proxyService.ListBlockedLogs(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": list, "total": total, "page": page, "page_size": pageSize}})
}

func (h *Handler) listFirewallBlacklist(c *gin.Context) {
	list, err := h.ruleService.ListFirewallBlacklist(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

func (h *Handler) deleteFirewallBlacklistEntry(c *gin.Context) {
	key := strings.TrimSpace(c.Query("key"))
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "key required"})
		return
	}
	deleted, err := h.ruleService.DeleteFirewallBlacklistEntry(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"deleted": deleted}, "message": "ok"})
}

func (h *Handler) clearFirewallBlacklist(c *gin.Context) {
	deleted, err := h.ruleService.ClearFirewallBlacklist(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"deleted": deleted}, "message": "ok"})
}

// Proxy

func stripPort(host string) string {
	if i := strings.Index(host, ":"); i >= 0 {
		return host[:i]
	}
	return host
}

func (h *Handler) Proxy() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("proxy panic: %v", r))
			}
		}()

		rawHost := strings.ToLower(c.Request.Host)
		host := strings.ToLower(stripPort(c.Request.Host))
		ctx := c.Request.Context()

		domain, err := h.proxyService.GetDomainByHost(ctx, rawHost)
		if err != nil && rawHost != host {
			domain, err = h.proxyService.GetDomainByHost(ctx, host)
		}
		isDefault := false
		if err != nil {
			domain, err = h.proxyService.GetDefaultDomain(ctx)
			if err != nil {
				renderErrorPage(c, http.StatusNotFound, "域名未配置", "当前访问的 Host 未匹配到任何域名映射，也未配置默认站点。")
				h.logProxy(&models.Domain{BindDomain: host}, c.Request, service.RealIP(c.Request, ""), "", http.StatusNotFound, false, nil, "domain not found", nil)
				return
			}
			isDefault = true
		}

		rewriteBindDomain := domain.BindDomain
		if isDefault {
			rewriteBindDomain = stripPort(c.Request.Host)
		}

		target, err := url.Parse(domain.TargetURL)
		if err != nil {
			renderErrorPage(c, http.StatusInternalServerError, "目标地址无效", "该域名配置的目标地址无法解析。")
			h.logProxy(domain, c.Request, service.RealIP(c.Request, domain.RealIPHeaders), "", http.StatusInternalServerError, false, nil, "invalid target", nil)
			return
		}

		realIP := service.RealIP(c.Request, domain.RealIPHeaders)
		rules, _ := h.proxyService.ListRulesByDomain(ctx, domain.ID)

		body, _ := service.ReadBody(c.Request)

		matched, blocked, status, blockResp, blockedRule, err := h.proxyService.EvaluateRules(ctx, rules, c.Request, c.Request.URL.Path, body, realIP)
		if err != nil {
			renderErrorPage(c, http.StatusInternalServerError, "规则引擎错误", "风控规则执行失败，请稍后重试。")
			h.logProxy(domain, c.Request, realIP, target.String(), http.StatusInternalServerError, false, nil, "rule engine error", nil)
			return
		}
		if blocked {
			accept := strings.ToLower(c.GetHeader("Accept"))
			if strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/html") {
				c.Header("Content-Type", "application/json; charset=utf-8")
				c.Status(status)
				c.Writer.Write([]byte(blockResp))
			} else {
				renderBlockPage(c, status, blockResp)
			}
			h.logProxy(domain, c.Request, realIP, target.String(), status, true, blockedRule, "blocked by rule", body)
			return
		}

		for _, rule := range matched {
			if rule.RuleType == "rate_limit" {
				h.proxyService.RecordAttempts(ctx, []models.Rule{rule}, domain.ID, c.Request.URL.Path, realIP, body, c.Request.Header.Get("Content-Type"))
			}
		}

		body, _ = h.proxyService.TransformBody(body, c.Request.Header.Get("Content-Type"), domain.RequestTransform)

		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		c.Request.ContentLength = int64(len(body))
		c.Request.Header.Set("Content-Length", strconv.Itoa(len(body)))

		proxy := service.NewReverseProxy(target, rewriteBindDomain, realIP, domain.ForwardIPHeader, domain.RewriteHost)
		rb := newResponseBuffer(c.Writer)
		proxy.ServeHTTP(rb, c.Request)

		clientScheme := c.GetHeader("X-Forwarded-Proto")
		if clientScheme == "" {
			if c.Request.TLS != nil {
				clientScheme = "https"
			} else {
				clientScheme = "http"
			}
		}
		rewriteResponse(rb, target, rewriteBindDomain, clientScheme, domain.RewriteMode)

		for _, rule := range matched {
			if rule.RuleType == "duplicate_ip" {
				if h.proxyService.ShouldRecordSuccess(rule, rb.statusCode, rb.header.Get("Location")) {
					h.proxyService.RecordAttempts(ctx, []models.Rule{rule}, domain.ID, c.Request.URL.Path, realIP, body, c.Request.Header.Get("Content-Type"))
				}
			}
		}
		rb.flush()

		h.logProxy(domain, c.Request, realIP, target.String(), rb.statusCode, false, nil, "", nil)
	}
}

func renderErrorPage(c *gin.Context, code int, title, message string) {
	html := strings.NewReplacer(
		"{{TITLE}}", title,
		"{{CODE}}", strconv.Itoa(code),
		"{{MESSAGE}}", message,
	).Replace(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{TITLE}}</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  background: radial-gradient(ellipse at center, #2a1818 0%, #1a1010 50%, #0a0505 100%);
  color: #e2e8f0;
  overflow: hidden;
}
.container {
  text-align: center;
  padding: 2rem;
  max-width: 520px;
  position: relative;
  z-index: 1;
}
.code {
  font-size: 5rem;
  font-weight: 800;
  line-height: 1;
  background: linear-gradient(90deg, #f87171, #fbbf24);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin-bottom: 1rem;
}
h1 {
  font-size: 1.75rem;
  font-weight: 700;
  margin-bottom: 0.75rem;
  color: #f8fafc;
}
.message {
  font-size: 1rem;
  line-height: 1.6;
  color: #94a3b8;
  margin-bottom: 2rem;
}
.brand {
  font-size: 0.75rem;
  color: #475569;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}
.brand strong { color: #f87171; }
.grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(239, 68, 68, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(239, 68, 68, 0.04) 1px, transparent 1px);
  background-size: 40px 40px;
  pointer-events: none;
}
</style>
</head>
<body>
  <div class="grid"></div>
  <div class="container">
    <div class="code">{{CODE}}</div>
    <h1>{{TITLE}}</h1>
    <p class="message">{{MESSAGE}}</p>
    <div class="brand">Protected by <strong>SafeGate Firewall</strong></div>
  </div>
</body>
</html>`)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(code)
	c.Writer.Write([]byte(html))
}

func marshalJSON(v interface{}) models.JSONB {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return models.JSONB(b)
}

func (h *Handler) logProxy(domain *models.Domain, r *http.Request, clientIP, target string, status int, blocked bool, rule *models.Rule, msg string, body []byte) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var ruleID *uint64
		var ruleName string
		if rule != nil {
			ruleID = &rule.ID
			ruleName = rule.Name
		}

		var queryParams, requestHeaders models.JSONB
		if blocked {
			queryParams = marshalJSON(r.URL.Query())
			requestHeaders = marshalJSON(r.Header)
		}

		_ = h.proxyService.Log(ctx, &models.ProxyLog{
			BindDomain:     domain.BindDomain,
			ClientIP:       clientIP,
			Method:         r.Method,
			Path:           r.URL.Path,
			QueryParams:    queryParams,
			RequestHeaders: requestHeaders,
			RequestBody:    string(body),
			UserAgent:      r.UserAgent(),
			TargetURL:      target,
			StatusCode:     &status,
			Blocked:        blocked,
			RuleID:         ruleID,
			RuleName:       ruleName,
			Message:        msg,
		})
	}()
}

func renderBlockPage(c *gin.Context, status int, blockResp string) {
	title := "访问已被拦截"
	message := ""
	detail := ""
	code := status
	ruleName := ""
	ruleType := ""

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(blockResp), &data); err == nil {
		if v, ok := data["title"].(string); ok && v != "" {
			title = v
		}
		if v, ok := data["message"].(string); ok && v != "" {
			message = v
		}
		if v, ok := data["detail"].(string); ok && v != "" {
			detail = v
		}
		if v, ok := data["code"].(float64); ok {
			code = int(v)
		}
		if v, ok := data["rule_name"].(string); ok {
			ruleName = v
		}
		if v, ok := data["rule_type"].(string); ok {
			ruleType = v
		}
	}

	// 如果用户没有配置 detail，自动拼接规则信息
	if detail == "" && ruleName != "" {
		typeText := "风控拦截"
		if ruleType == "duplicate_ip" {
			typeText = "重复 IP 拦截"
		} else if ruleType == "rate_limit" {
			typeText = "访问频率限制"
		}
		detail = fmt.Sprintf("触发规则：%s（%s）", ruleName, typeText)
	}

	messageDiv := ""
	if message != "" {
		messageDiv = fmt.Sprintf(`<p class="message">%s</p>`, message)
	}

	detailDiv := ""
	if detail != "" {
		detailDiv = fmt.Sprintf(`<div class="detail">%s</div>`, detail)
	}

	html := strings.NewReplacer(
		"{{TITLE}}", title,
		"{{CODE}}", strconv.Itoa(code),
		"{{MESSAGE_DIV}}", messageDiv,
		"{{DETAIL}}", detailDiv,
	).Replace(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{TITLE}}</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  background: radial-gradient(ellipse at center, #2a1818 0%, #1a1010 50%, #0a0505 100%);
  color: #e2e8f0;
  overflow: hidden;
}
.container {
  text-align: center;
  padding: 2rem;
  max-width: 560px;
  position: relative;
  z-index: 1;
}
.shield-wrap {
  position: relative;
  width: 140px;
  height: 140px;
  margin: 0 auto 2rem;
}
.pulse {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 2px solid rgba(239, 68, 68, 0.4);
  animation: pulse 2s ease-out infinite;
}
.pulse:nth-child(2) { animation-delay: 0.6s; }
.pulse:nth-child(3) { animation-delay: 1.2s; }
@keyframes pulse {
  0% { transform: scale(0.8); opacity: 0.6; }
  100% { transform: scale(1.6); opacity: 0; }
}
.shield {
  position: absolute;
  inset: 20px;
  border-radius: 50%;
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 0 40px rgba(239, 68, 68, 0.5);
}
.shield svg {
  width: 56px;
  height: 56px;
  fill: none;
  stroke: #fff;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.code {
  font-size: 5rem;
  font-weight: 800;
  line-height: 1;
  background: linear-gradient(90deg, #f87171, #fca5a5);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin-bottom: 1rem;
}
h1 {
  font-size: 1.75rem;
  font-weight: 700;
  margin-bottom: 0.75rem;
  color: #f8fafc;
}
.message {
  font-size: 1rem;
  line-height: 1.6;
  color: #94a3b8;
  margin-bottom: 1rem;
}
.detail {
  font-size: 0.875rem;
  color: #64748b;
  background: rgba(30, 41, 59, 0.6);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 0.75rem;
  padding: 1rem;
  margin-bottom: 2rem;
  word-break: break-word;
}
.brand {
  font-size: 0.75rem;
  color: #475569;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}
.brand strong { color: #f87171; }
.grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(239, 68, 68, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(239, 68, 68, 0.04) 1px, transparent 1px);
  background-size: 40px 40px;
  pointer-events: none;
}
</style>
</head>
<body>
  <div class="grid"></div>
  <div class="container">
    <div class="shield-wrap">
      <div class="pulse"></div>
      <div class="pulse"></div>
      <div class="pulse"></div>
      <div class="shield">
        <svg viewBox="0 0 24 24"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>
      </div>
    </div>
    <div class="code">{{CODE}}</div>
    <h1>{{TITLE}}</h1>
    {{MESSAGE_DIV}}
    {{DETAIL}}
    <div class="brand">Protected by <strong>SafeGate Firewall</strong></div>
  </div>
</body>
</html>`)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(status)
	c.Writer.Write([]byte(html))
}

type responseBuffer struct {
	http.ResponseWriter
	statusCode int
	header     http.Header
	body       *bytes.Buffer
}

func newResponseBuffer(w http.ResponseWriter) *responseBuffer {
	return &responseBuffer{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		header:         w.Header().Clone(),
		body:           &bytes.Buffer{},
	}
}

func (rb *responseBuffer) Header() http.Header { return rb.header }

func (rb *responseBuffer) Write(b []byte) (int, error) {
	return rb.body.Write(b)
}

func (rb *responseBuffer) WriteString(s string) (int, error) {
	return rb.body.WriteString(s)
}

func (rb *responseBuffer) WriteHeader(code int) {
	rb.statusCode = code
}

func (rb *responseBuffer) flush() {
	wh := rb.ResponseWriter.Header()
	for k, v := range rb.header {
		wh[k] = v
	}
	rb.ResponseWriter.WriteHeader(rb.statusCode)
	if rb.body.Len() > 0 {
		rb.ResponseWriter.Write(rb.body.Bytes())
	}
}

var cookieDomainRe = regexp.MustCompile(`(?i)\s*domain=[^;]+;?`)

func rewriteResponse(rb *responseBuffer, target *url.URL, bindDomain, clientScheme, rewriteMode string) {
	mode := strings.ToLower(rewriteMode)
	if mode == "" {
		mode = "full"
	}

	if mode == "headers" || mode == "full" {
		// Rewrite 3xx redirects
		if loc := rb.header.Get("Location"); loc != "" {
			loc = strings.ReplaceAll(loc, target.Scheme+"://"+target.Host, clientScheme+"://"+bindDomain)
			loc = strings.ReplaceAll(loc, target.Host, bindDomain)
			rb.header.Set("Location", loc)
		}

		// Rewrite Set-Cookie domain to bind domain (remove Domain=... makes it host-only)
		cookies := rb.header.Values("Set-Cookie")
		if len(cookies) > 0 {
			rb.header.Del("Set-Cookie")
			for _, c := range cookies {
				rb.header.Add("Set-Cookie", cookieDomainRe.ReplaceAllString(c, ""))
			}
		}
	}

	if mode == "full" {
		// Rewrite HTML body links
		contentType := rb.header.Get("Content-Type")
		if strings.Contains(contentType, "text/html") {
			oldSchemeHost := target.Scheme + "://" + target.Host
			newSchemeHost := clientScheme + "://" + bindDomain
			s := rb.body.String()
			s = strings.ReplaceAll(s, oldSchemeHost, newSchemeHost)
			s = strings.ReplaceAll(s, target.Host, bindDomain)
			rb.body = bytes.NewBufferString(s)
		}
	}

	rb.header.Del("Content-Length")
}
