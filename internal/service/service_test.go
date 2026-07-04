package service

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"ip_check/internal/models"
)

func TestBuildIdentityFromJSON(t *testing.T) {
	body := []byte(`{"user":{"phone":"13800138000","email":"a@example.com"}}`)

	got := buildIdentity("1.2.3.4", "user.phone,user.email", body, "application/json")
	want := "1.2.3.4|user.phone=13800138000|user.email=a@example.com"

	if got != want {
		t.Fatalf("identity mismatch:\nwant %q\n got %q", want, got)
	}
}

func TestBuildIdentityFromURLEncodedForm(t *testing.T) {
	body := []byte("phone=13800138000&email=a%40example.com")

	got := buildIdentity("1.2.3.4", "phone,email", body, "application/x-www-form-urlencoded")
	want := "1.2.3.4|phone=13800138000|email=a@example.com"

	if got != want {
		t.Fatalf("identity mismatch:\nwant %q\n got %q", want, got)
	}
}

func TestBuildIdentityFromBracketFormPath(t *testing.T) {
	body := []byte("user%5Bphone%5D=13800138000&user%5Bemail%5D=a%40example.com")

	got := buildIdentity("1.2.3.4", "user.phone,user.email", body, "application/x-www-form-urlencoded")
	want := "1.2.3.4|user.phone=13800138000|user.email=a@example.com"

	if got != want {
		t.Fatalf("identity mismatch:\nwant %q\n got %q", want, got)
	}
}

func TestBuildIdentityFromMultipartForm(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("phone", "13800138000"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("email", "a@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	got := buildIdentity("1.2.3.4", "phone,email", body.Bytes(), writer.FormDataContentType())
	want := "1.2.3.4|phone=13800138000|email=a@example.com"

	if got != want {
		t.Fatalf("identity mismatch:\nwant %q\n got %q", want, got)
	}
}

func TestQueryMatches(t *testing.T) {
	values := url.Values{
		"e":    {"index.post_register"},
		"type": {"1"},
	}

	tests := []struct {
		name  string
		match string
		want  bool
	}{
		{name: "empty match", match: "", want: true},
		{name: "single query", match: "e=index.post_register", want: true},
		{name: "leading question mark", match: "?e=index.post_register", want: true},
		{name: "multiple query values", match: "e=index.post_register&type=1", want: true},
		{name: "missing key", match: "missing=1", want: false},
		{name: "wrong value", match: "e=index.login", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := queryMatches(tt.match, values); got != tt.want {
				t.Fatalf("queryMatches(%q) = %v, want %v", tt.match, got, tt.want)
			}
		})
	}
}

func TestNormalizeRulePathQuery(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		query     string
		wantPath  string
		wantQuery string
	}{
		{
			name:      "path contains query from legacy saved rule",
			path:      "/index.php?e=index.post_register",
			wantPath:  "/index.php",
			wantQuery: "e=index.post_register",
		},
		{
			name:      "path query merges with configured query",
			path:      "/index.php?e=index.post_register",
			query:     "type=1",
			wantPath:  "/index.php",
			wantQuery: "e=index.post_register&type=1",
		},
		{
			name:      "full url keeps only request path and query",
			path:      "http://localhost:18080/index.php?e=index.post_register",
			wantPath:  "/index.php",
			wantQuery: "e=index.post_register",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotQuery := normalizeRulePathQuery(tt.path, tt.query)
			if gotPath != tt.wantPath || gotQuery != tt.wantQuery {
				t.Fatalf("normalizeRulePathQuery() = (%q, %q), want (%q, %q)", gotPath, gotQuery, tt.wantPath, tt.wantQuery)
			}
		})
	}
}

func TestShouldRecordSuccess(t *testing.T) {
	svc := &ProxyService{}

	tests := []struct {
		name     string
		rule     models.Rule
		status   int
		location string
		want     bool
	}{
		{
			name:   "default only counts 2xx",
			rule:   models.Rule{},
			status: httpStatusOK,
			want:   true,
		},
		{
			name:   "default skips redirect",
			rule:   models.Rule{},
			status: httpStatusFound,
			want:   false,
		},
		{
			name:   "configured redirect counts",
			rule:   models.Rule{SuccessStatuses: "2xx,302"},
			status: httpStatusFound,
			want:   true,
		},
		{
			name:     "failure location skips redirect count",
			rule:     models.Rule{SuccessStatuses: "2xx,302", FailureLocationMatch: "key=username_repeat_register"},
			status:   httpStatusFound,
			location: "/index.php?e=index.msg&key=username_repeat_register",
			want:     false,
		},
		{
			name:     "success location requires matching redirect",
			rule:     models.Rule{SuccessStatuses: "302", SuccessLocationMatch: "key=register_success"},
			status:   httpStatusFound,
			location: "/index.php?e=index.msg&key=register_success",
			want:     true,
		},
		{
			name:     "success location rejects non matching redirect",
			rule:     models.Rule{SuccessStatuses: "302", SuccessLocationMatch: "key=register_success"},
			status:   httpStatusFound,
			location: "/index.php?e=index.msg&key=username_repeat_register",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.ShouldRecordSuccess(tt.rule, tt.status, tt.location); got != tt.want {
				t.Fatalf("ShouldRecordSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDuplicateIPRuleBlocksRepeatedRegisterFormAfterRedirectSuccess(t *testing.T) {
	body := []byte("type=1&username=xiaoyun&password=VPN12345678&passwordre=VPN12345678&contact=%E9%82%93%E4%B9%BE&mobile=15027195073&qq=&email=xiaoyun%40email.hbue.edu.cn&bankname=icbc&accountname=%E9%82%93%E4%B9%BE&bankbranch=&bankaccount=15027195073&regcode=GATM")
	contentType := "application/x-www-form-urlencoded"
	realIP := "127.0.0.1"
	rule := models.Rule{
		ID:                   99,
		Name:                 "多字段组合防重",
		PathPrefix:           "/index.php",
		QueryMatch:           "e=index.post_register",
		Methods:              http.MethodPost,
		RuleType:             "duplicate_ip",
		IdentityFields:       "username,mobile,email,bankaccount",
		SuccessStatuses:      "302",
		SuccessLocationMatch: "key=username_repeat_register",
		MaxAttempts:          1,
		WindowSeconds:        0,
		BlockStatus:          http.StatusForbidden,
		BlockResponse:        models.JSONB(`{"code":403,"message":"重复注册"}`),
		Enabled:              true,
	}
	svc := NewProxyService(nil, newFakeAttemptStore())

	identity := buildIdentity(realIP, rule.IdentityFields, body, contentType)
	wantIdentity := "127.0.0.1|username=xiaoyun|mobile=15027195073|email=xiaoyun@email.hbue.edu.cn|bankaccount=15027195073"
	if identity != wantIdentity {
		t.Fatalf("identity mismatch:\nwant %q\n got %q", wantIdentity, identity)
	}

	req := newRegisterFormRequest(t, body, contentType)
	matched, blocked, _, _, _, err := svc.EvaluateRules(context.Background(), []models.Rule{rule}, req, req.URL.Path, body, realIP)
	if err != nil {
		t.Fatal(err)
	}
	if blocked {
		t.Fatal("first register request should not be blocked before it is recorded")
	}
	if len(matched) != 1 || matched[0].ID != rule.ID {
		t.Fatalf("matched rules mismatch: got %+v", matched)
	}

	location := "/index.php?e=index.msg&key=username_repeat_register"
	if !svc.ShouldRecordSuccess(rule, http.StatusFound, location) {
		t.Fatal("302 redirect with username_repeat_register should be recorded by this test rule")
	}
	svc.RecordAttempts(context.Background(), matched, 0, req.URL.Path, realIP, body, contentType)

	req = newRegisterFormRequest(t, body, contentType)
	_, blocked, status, blockResp, blockedRule, err := svc.EvaluateRules(context.Background(), []models.Rule{rule}, req, req.URL.Path, body, realIP)
	if err != nil {
		t.Fatal(err)
	}
	if !blocked {
		t.Fatal("second identical register request should be blocked after the first redirect was recorded")
	}
	if status != http.StatusForbidden {
		t.Fatalf("block status = %d, want %d", status, http.StatusForbidden)
	}
	if blockedRule == nil || blockedRule.ID != rule.ID {
		t.Fatalf("blocked rule mismatch: got %+v", blockedRule)
	}
	if !strings.Contains(blockResp, `"message":"重复注册"`) {
		t.Fatalf("block response %q does not contain configured message", blockResp)
	}
}

func TestDuplicateIPRuleMatchesLegacyPathPrefixWithQuery(t *testing.T) {
	body := []byte("type=1&username=xiaoyun&password=VPN12345678&passwordre=VPN12345678&contact=%E9%82%93%E4%B9%BE&mobile=15027195073&qq=&email=xiaoyun%40email.hbue.edu.cn&bankname=icbc&accountname=%E9%82%93%E4%B9%BE&bankbranch=&bankaccount=15027195073&regcode=GATM")
	contentType := "application/x-www-form-urlencoded"
	realIP := "127.0.0.1"
	rule := models.Rule{
		ID:                   100,
		Name:                 "历史路径带 Query 的规则",
		PathPrefix:           "/index.php?e=index.post_register",
		QueryMatch:           "",
		Methods:              http.MethodPost,
		RuleType:             "duplicate_ip",
		IdentityFields:       "username,mobile,email,bankaccount",
		SuccessStatuses:      "302",
		SuccessLocationMatch: "e=index.msg&key=username_repeat_register",
		MaxAttempts:          1,
		WindowSeconds:        0,
		BlockStatus:          http.StatusForbidden,
		BlockResponse:        models.JSONB(`{"code":403,"message":"重复注册"}`),
		Enabled:              true,
	}
	svc := NewProxyService(nil, newFakeAttemptStore())

	req := newRegisterFormRequest(t, body, contentType)
	matched, blocked, _, _, _, err := svc.EvaluateRules(context.Background(), []models.Rule{rule}, req, req.URL.Path, body, realIP)
	if err != nil {
		t.Fatal(err)
	}
	if blocked {
		t.Fatal("first legacy-rule request should not be blocked before it is recorded")
	}
	if len(matched) != 1 || matched[0].ID != rule.ID {
		t.Fatalf("legacy path/query rule should match, got %+v", matched)
	}

	location := "/index.php?e=index.msg&key=username_repeat_register"
	if !svc.ShouldRecordSuccess(rule, http.StatusFound, location) {
		t.Fatal("302 redirect should be recorded for legacy path/query rule")
	}
	svc.RecordAttempts(context.Background(), matched, 0, req.URL.Path, realIP, body, contentType)

	req = newRegisterFormRequest(t, body, contentType)
	_, blocked, status, _, _, err := svc.EvaluateRules(context.Background(), []models.Rule{rule}, req, req.URL.Path, body, realIP)
	if err != nil {
		t.Fatal(err)
	}
	if !blocked || status != http.StatusForbidden {
		t.Fatalf("second legacy-rule request blocked=%v status=%d, want blocked with %d", blocked, status, http.StatusForbidden)
	}
}

func newRegisterFormRequest(t *testing.T, body []byte, contentType string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodPost, "http://localhost:18080/index.php?e=index.post_register", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", contentType)
	return req
}

type fakeAttemptStore struct {
	counts map[string]int64
}

func newFakeAttemptStore() *fakeAttemptStore {
	return &fakeAttemptStore{counts: make(map[string]int64)}
}

func (f *fakeAttemptStore) IncrementAttempt(_ context.Context, ruleID uint64, identity string, _ int) (int64, error) {
	key := f.key(ruleID, identity)
	f.counts[key]++
	return f.counts[key], nil
}

func (f *fakeAttemptStore) GetAttemptCount(_ context.Context, ruleID uint64, identity string) (int64, error) {
	return f.counts[f.key(ruleID, identity)], nil
}

func (f *fakeAttemptStore) SetAttemptCount(_ context.Context, ruleID uint64, identity string, count int64, _ int) error {
	f.counts[f.key(ruleID, identity)] = count
	return nil
}

func (f *fakeAttemptStore) key(ruleID uint64, identity string) string {
	return fmt.Sprintf("attempt:%d:%s", ruleID, identity)
}

const (
	httpStatusOK    = 200
	httpStatusFound = 302
)
