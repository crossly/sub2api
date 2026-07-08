package main

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestParseCodexTabLineBuildsOpenAIOAuthAccount(t *testing.T) {
	now := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	idToken := buildTestJWT(t, map[string]any{
		"email": "claims@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acct_123",
			"chatgpt_user_id":    "chatgpt-user-123",
			"user_id":            "user-123",
			"chatgpt_plan_type":  "team",
			"organizations": []map[string]any{
				{"id": "org_default", "is_default": true},
			},
		},
	})

	account, warning, err := parseCodexTabLine(
		1,
		"ac_test_token\towner@example.com\t"+idToken+"\trt_test_token|app_test_client",
		now,
	)
	if err != nil {
		t.Fatalf("parseCodexTabLine error = %v", err)
	}
	if warning != "" {
		t.Fatalf("warning = %q, want empty", warning)
	}
	if account.Name != "owner@example.com" {
		t.Fatalf("Name = %q, want owner@example.com", account.Name)
	}
	if account.Platform != "openai" {
		t.Fatalf("Platform = %q, want openai", account.Platform)
	}
	if account.Type != "oauth" {
		t.Fatalf("Type = %q, want oauth", account.Type)
	}
	if account.Concurrency != 3 {
		t.Fatalf("Concurrency = %d, want 3", account.Concurrency)
	}
	if account.Priority != 50 {
		t.Fatalf("Priority = %d, want 50", account.Priority)
	}
	if got := stringValue(account.Credentials["access_token"]); got != "ac_test_token" {
		t.Fatalf("access_token = %q, want ac_test_token", got)
	}
	if got := stringValue(account.Credentials["refresh_token"]); got != "rt_test_token" {
		t.Fatalf("refresh_token = %q, want rt_test_token", got)
	}
	if got := stringValue(account.Credentials["client_id"]); got != "app_test_client" {
		t.Fatalf("client_id = %q, want app_test_client", got)
	}
	if got := stringValue(account.Credentials["id_token"]); got != idToken {
		t.Fatalf("id_token mismatch")
	}
	if got := stringValue(account.Credentials["email"]); got != "owner@example.com" {
		t.Fatalf("email = %q, want owner@example.com", got)
	}
	if got := stringValue(account.Credentials["chatgpt_account_id"]); got != "acct_123" {
		t.Fatalf("chatgpt_account_id = %q, want acct_123", got)
	}
	if got := stringValue(account.Credentials["chatgpt_user_id"]); got != "chatgpt-user-123" {
		t.Fatalf("chatgpt_user_id = %q, want chatgpt-user-123", got)
	}
	if got := stringValue(account.Credentials["plan_type"]); got != "team" {
		t.Fatalf("plan_type = %q, want team", got)
	}
	if got := stringValue(account.Credentials["organization_id"]); got != "org_default" {
		t.Fatalf("organization_id = %q, want org_default", got)
	}
	expiresAt := stringValue(account.Credentials["expires_at"])
	if expiresAt == "" {
		t.Fatalf("expires_at should be set to force initial refresh")
	}
	parsedExpiresAt, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		t.Fatalf("expires_at parse error = %v", err)
	}
	if !parsedExpiresAt.Before(now) {
		t.Fatalf("expires_at = %q, want a past time before %q", expiresAt, now.Format(time.RFC3339))
	}
	if got := stringValue(account.Extra["import_source"]); got != "codex_tab_txt" {
		t.Fatalf("import_source = %q, want codex_tab_txt", got)
	}
	if got := stringValue(account.Extra["imported_at"]); got != now.Format(time.RFC3339) {
		t.Fatalf("imported_at = %q, want %q", got, now.Format(time.RFC3339))
	}
}

func TestConvertTSVToPayloadBuildsImportJSON(t *testing.T) {
	now := time.Date(2026, 6, 6, 12, 0, 0, 0, time.UTC)
	idToken := buildTestJWT(t, map[string]any{
		"email": "claims@example.com",
	})

	payload, warnings, err := convertTSVToPayload(strings.NewReader(
		"ac_1\tfirst@example.com\t"+idToken+"\trt_1|app_client\n"+
			"ac_2\tsecond@example.com\t"+idToken+"\trt_2|app_client\n",
	), now)
	if err != nil {
		t.Fatalf("convertTSVToPayload error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if payload.Type != "sub2api-data" {
		t.Fatalf("Type = %q, want sub2api-data", payload.Type)
	}
	if payload.Version != 1 {
		t.Fatalf("Version = %d, want 1", payload.Version)
	}
	if payload.ExportedAt != now.Format(time.RFC3339) {
		t.Fatalf("ExportedAt = %q, want %q", payload.ExportedAt, now.Format(time.RFC3339))
	}
	if len(payload.Proxies) != 0 {
		t.Fatalf("len(Proxies) = %d, want 0", len(payload.Proxies))
	}
	if len(payload.Accounts) != 2 {
		t.Fatalf("len(Accounts) = %d, want 2", len(payload.Accounts))
	}
	if payload.Accounts[0].Name != "first@example.com" {
		t.Fatalf("first account name = %q, want first@example.com", payload.Accounts[0].Name)
	}
	if payload.Accounts[1].Name != "second@example.com" {
		t.Fatalf("second account name = %q, want second@example.com", payload.Accounts[1].Name)
	}
}

func TestDefaultOutputPathReplacesExtension(t *testing.T) {
	got := defaultOutputPath("/tmp/accounts.txt")
	want := "/tmp/accounts.sub2api.json"
	if got != want {
		t.Fatalf("defaultOutputPath = %q, want %q", got, want)
	}
}

func buildTestJWT(t *testing.T, claims map[string]any) string {
	t.Helper()

	headerJSON, err := json.Marshal(map[string]any{
		"alg": "none",
		"typ": "JWT",
	})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(headerJSON) + "." +
		base64.RawURLEncoding.EncodeToString(payloadJSON) + "."
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}
