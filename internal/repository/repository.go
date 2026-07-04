package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"ip_check/internal/models"
)

type Repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// User

func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) UserCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}

// Domain

func (r *Repository) CreateDomain(ctx context.Context, d *models.Domain) error {
	d.BindDomain = strings.ToLower(strings.TrimSpace(d.BindDomain))
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *Repository) UpdateDomain(ctx context.Context, id uint64, d *models.Domain) error {
	d.BindDomain = strings.ToLower(strings.TrimSpace(d.BindDomain))
	return r.db.WithContext(ctx).Model(&models.Domain{}).Where("id = ?", id).Updates(map[string]interface{}{
		"bind_domain":        d.BindDomain,
		"target_url":         d.TargetURL,
		"real_ip_headers":    d.RealIPHeaders,
		"forward_ip_header":  d.ForwardIPHeader,
		"request_transform":  d.RequestTransform,
		"response_transform": d.ResponseTransform,
		"rewrite_host":       d.RewriteHost,
		"rewrite_mode":       d.RewriteMode,
		"is_default":         d.IsDefault,
	}).Error
}

func (r *Repository) GetDefaultDomain(ctx context.Context) (*models.Domain, error) {
	var d models.Domain
	err := r.db.WithContext(ctx).Where("is_default = ?", true).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) ClearDefaultDomain(ctx context.Context, excludeID uint64) error {
	q := r.db.WithContext(ctx).Model(&models.Domain{}).Where("is_default = ?", true)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	return q.Update("is_default", false).Error
}

func (r *Repository) DeleteDomain(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&models.Domain{}, id).Error
}

func (r *Repository) GetDomainByID(ctx context.Context, id uint64) (*models.Domain, error) {
	var d models.Domain
	err := r.db.WithContext(ctx).First(&d, id).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) GetDomainByBind(ctx context.Context, bind string) (*models.Domain, error) {
	var d models.Domain
	err := r.db.WithContext(ctx).Where("bind_domain = ?", strings.ToLower(bind)).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) ListDomains(ctx context.Context) ([]models.Domain, error) {
	var list []models.Domain
	err := r.db.WithContext(ctx).Order("id desc").Find(&list).Error
	return list, err
}

// Rule

func (r *Repository) CreateRule(ctx context.Context, rule *models.Rule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *Repository) UpdateRule(ctx context.Context, id uint64, rule *models.Rule) error {
	return r.db.WithContext(ctx).Model(&models.Rule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"domain_id":              rule.DomainID,
		"name":                   rule.Name,
		"path_prefix":            rule.PathPrefix,
		"query_match":            rule.QueryMatch,
		"methods":                strings.ToUpper(rule.Methods),
		"rule_type":              rule.RuleType,
		"identity_fields":        rule.IdentityFields,
		"success_statuses":       rule.SuccessStatuses,
		"success_location_match": rule.SuccessLocationMatch,
		"failure_location_match": rule.FailureLocationMatch,
		"max_attempts":           rule.MaxAttempts,
		"window_seconds":         rule.WindowSeconds,
		"block_seconds":          rule.BlockSeconds,
		"block_status":           rule.BlockStatus,
		"block_response":         rule.BlockResponse,
		"enabled":                rule.Enabled,
	}).Error
}

func (r *Repository) DeleteRule(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Delete(&models.Rule{}, id).Error
}

func (r *Repository) GetRuleByID(ctx context.Context, id uint64) (*models.Rule, error) {
	var rule models.Rule
	err := r.db.WithContext(ctx).First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

func (r *Repository) ListRulesByDomain(ctx context.Context, domainID uint64) ([]models.Rule, error) {
	var list []models.Rule
	err := r.db.WithContext(ctx).Where("domain_id = ?", domainID).Order("id desc").Find(&list).Error
	return list, err
}

func (r *Repository) ListRulesByIDs(ctx context.Context, ids []uint64, out *[]models.Rule) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Where("id IN ?", ids).Find(out).Error
}

// Log

func (r *Repository) CreateLog(ctx context.Context, log *models.ProxyLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *Repository) ListLogs(ctx context.Context, page, pageSize int) ([]models.ProxyLog, int64, error) {
	var list []models.ProxyLog
	var total int64
	offset := (page - 1) * pageSize
	db := r.db.WithContext(ctx).Model(&models.ProxyLog{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) ListBlockedLogs(ctx context.Context, page, pageSize int) ([]models.ProxyLog, int64, error) {
	var list []models.ProxyLog
	var total int64
	offset := (page - 1) * pageSize
	db := r.db.WithContext(ctx).Model(&models.ProxyLog{}).Where("blocked = ?", true)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *Repository) StatsBlocked(ctx context.Context) (*models.BlockedStats, error) {
	stats := &models.BlockedStats{
		TopIPs:     []models.TopIP{},
		TopRules:   []models.TopRule{},
		DailyTrend: []models.DailyTrend{},
	}

	// total blocked
	if err := r.db.WithContext(ctx).Model(&models.ProxyLog{}).Where("blocked = ?", true).Count(&stats.TotalBlocked).Error; err != nil {
		return nil, err
	}

	// today blocked
	today := time.Now().Truncate(24 * time.Hour)
	if err := r.db.WithContext(ctx).Model(&models.ProxyLog{}).Where("blocked = ? AND created_at >= ?", true, today).Count(&stats.TodayBlocked).Error; err != nil {
		return nil, err
	}

	// unique blocked ips
	if err := r.db.WithContext(ctx).Model(&models.ProxyLog{}).Where("blocked = ?", true).Select("COUNT(DISTINCT client_ip)").Scan(&stats.UniqueIPs).Error; err != nil {
		return nil, err
	}

	// active rules (rules that have blocked at least once)
	if err := r.db.WithContext(ctx).Model(&models.ProxyLog{}).Where("blocked = ? AND rule_id IS NOT NULL", true).Select("COUNT(DISTINCT rule_id)").Scan(&stats.ActiveRules).Error; err != nil {
		return nil, err
	}

	// top ips
	if err := r.db.WithContext(ctx).Raw(`
		SELECT client_ip, COUNT(*) as count
		FROM proxy_logs
		WHERE blocked = ?
		GROUP BY client_ip
		ORDER BY count DESC
		LIMIT 10
	`, true).Scan(&stats.TopIPs).Error; err != nil {
		return nil, err
	}

	// top rules
	if err := r.db.WithContext(ctx).Raw(`
		SELECT rule_id, COUNT(*) as count
		FROM proxy_logs
		WHERE blocked = ? AND rule_id IS NOT NULL
		GROUP BY rule_id
		ORDER BY count DESC
		LIMIT 10
	`, true).Scan(&stats.TopRules).Error; err != nil {
		return nil, err
	}

	// daily trend (last 7 days)
	if err := r.db.WithContext(ctx).Raw(`
		SELECT TO_CHAR(DATE(created_at), 'YYYY-MM-DD') as date, COUNT(*) as count
		FROM proxy_logs
		WHERE blocked = ? AND created_at >= NOW() - INTERVAL '7 days'
		GROUP BY DATE(created_at)
		ORDER BY DATE(created_at) ASC
	`, true).Scan(&stats.DailyTrend).Error; err != nil {
		return nil, err
	}

	return stats, nil
}
