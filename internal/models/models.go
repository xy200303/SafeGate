package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type JSONB json.RawMessage

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case string:
		*j = JSONB(v)
	case []byte:
		*j = JSONB(v)
	default:
		return errors.New("invalid scan source for JSONB")
	}
	return nil
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	*j = JSONB(data)
	return nil
}

type User struct {
	ID           uint64         `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:64;not null" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

type Domain struct {
	ID                uint64         `gorm:"primaryKey" json:"id"`
	BindDomain        string         `gorm:"uniqueIndex;size:255;not null" json:"bind_domain"`
	TargetURL         string         `gorm:"size:512;not null" json:"target_url"`
	RealIPHeaders     string         `gorm:"size:512;default:'CF-Connecting-IP,X-Forwarded-For,X-Real-IP'" json:"real_ip_headers"`
	ForwardIPHeader   string         `gorm:"size:128;default:'X-Forwarded-For'" json:"forward_ip_header"`
	RequestTransform  JSONB          `gorm:"type:jsonb;default:'[]'" json:"request_transform"`
	ResponseTransform JSONB          `gorm:"type:jsonb;default:'[]'" json:"response_transform"`
	RewriteHost       bool           `gorm:"default:true" json:"rewrite_host"`
	RewriteMode       string         `gorm:"size:32;default:'full'" json:"rewrite_mode"`
	IsDefault         bool           `gorm:"default:false;index" json:"is_default"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

type Rule struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	DomainID       uint64         `gorm:"not null;index" json:"domain_id"`
	Domain         Domain         `gorm:"foreignKey:DomainID" json:"-"`
	Name           string         `gorm:"size:128;not null" json:"name"`
	PathPrefix     string         `gorm:"size:255;not null" json:"path_prefix"`
	Methods        string         `gorm:"size:128;default:'ALL'" json:"methods"`
	RuleType       string         `gorm:"size:32;not null" json:"rule_type"`
	IdentityFields string         `gorm:"size:512" json:"identity_fields"`
	MaxAttempts    int            `gorm:"default:1" json:"max_attempts"`
	WindowSeconds  int            `gorm:"default:0" json:"window_seconds"`
	BlockSeconds   int            `gorm:"default:0" json:"block_seconds"`
	BlockStatus    int            `gorm:"default:403" json:"block_status"`
	BlockResponse  JSONB          `gorm:"type:jsonb" json:"block_response"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type ProxyLog struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	BindDomain     string    `gorm:"size:255;not null;index" json:"bind_domain"`
	ClientIP       string    `gorm:"size:64;not null" json:"client_ip"`
	Method         string    `gorm:"size:16;not null" json:"method"`
	Path           string    `gorm:"size:2048;not null" json:"path"`
	QueryParams    JSONB     `gorm:"type:jsonb" json:"query_params"`
	RequestHeaders JSONB     `gorm:"type:jsonb" json:"request_headers"`
	RequestBody    string    `gorm:"type:text" json:"request_body"`
	UserAgent      string    `gorm:"size:512" json:"user_agent"`
	TargetURL      string    `gorm:"size:512;not null" json:"target_url"`
	StatusCode     *int      `json:"status_code"`
	Blocked        bool      `gorm:"default:false" json:"blocked"`
	RuleID         *uint64   `json:"rule_id"`
	RuleName       string    `gorm:"size:128" json:"rule_name"`
	Message        string    `gorm:"size:512" json:"message"`
	CreatedAt      time.Time `json:"created_at"`
}

type TopIP struct {
	ClientIP string `json:"client_ip"`
	Count    int64  `json:"count"`
}

type TopRule struct {
	RuleID   uint64 `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Count    int64  `json:"count"`
}

type DailyTrend struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type BlockedStats struct {
	TotalBlocked int64        `json:"total_blocked"`
	TodayBlocked int64        `json:"today_blocked"`
	UniqueIPs    int64        `json:"unique_ips"`
	ActiveRules  int64        `json:"active_rules"`
	TopIPs       []TopIP      `json:"top_ips"`
	TopRules     []TopRule    `json:"top_rules"`
	DailyTrend   []DailyTrend `json:"daily_trend"`
}
