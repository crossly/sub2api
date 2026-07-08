package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

const (
	dataType    = "sub2api-data"
	dataVersion = 1

	defaultConcurrency = 3
	defaultPriority    = 50
)

type dataPayload struct {
	Type       string        `json:"type,omitempty"`
	Version    int           `json:"version,omitempty"`
	ExportedAt string        `json:"exported_at"`
	Proxies    []dataProxy   `json:"proxies"`
	Accounts   []dataAccount `json:"accounts"`
}

type dataProxy struct {
	ProxyKey string `json:"proxy_key"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Status   string `json:"status"`
}

type dataAccount struct {
	Name               string         `json:"name"`
	Notes              *string        `json:"notes,omitempty"`
	Platform           string         `json:"platform"`
	Type               string         `json:"type"`
	Credentials        map[string]any `json:"credentials"`
	Extra              map[string]any `json:"extra,omitempty"`
	ProxyKey           *string        `json:"proxy_key,omitempty"`
	Concurrency        int            `json:"concurrency"`
	Priority           int            `json:"priority"`
	RateMultiplier     *float64       `json:"rate_multiplier,omitempty"`
	ExpiresAt          *int64         `json:"expires_at,omitempty"`
	AutoPauseOnExpired *bool          `json:"auto_pause_on_expired,omitempty"`
}

func main() {
	var inputPath string
	var outputPath string

	flag.StringVar(&inputPath, "input", "", "Path to source .txt file")
	flag.StringVar(&outputPath, "output", "", "Path to output .json file")
	flag.Parse()

	if strings.TrimSpace(inputPath) == "" {
		fmt.Fprintln(os.Stderr, "missing required -input")
		os.Exit(2)
	}
	if strings.TrimSpace(outputPath) == "" {
		outputPath = defaultOutputPath(inputPath)
	}

	if err := run(inputPath, outputPath, time.Now().UTC()); err != nil {
		fmt.Fprintf(os.Stderr, "convert failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(outputPath)
}

func run(inputPath, outputPath string, now time.Time) error {
	in, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer in.Close()

	payload, warnings, err := convertTSVToPayload(in, now)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return err
	}

	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, warning)
	}

	return nil
}

func convertTSVToPayload(r io.Reader, now time.Time) (*dataPayload, []string, error) {
	scanner := bufio.NewScanner(r)
	// Allow large JWT/id_token lines.
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)

	accounts := make([]dataAccount, 0)
	warnings := make([]string, 0)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		account, warning, err := parseCodexTabLine(lineNo, line, now)
		if err != nil {
			return nil, warnings, err
		}
		accounts = append(accounts, account)
		if warning != "" {
			warnings = append(warnings, warning)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, warnings, err
	}
	if len(accounts) == 0 {
		return nil, warnings, errors.New("no valid rows found")
	}

	payload := &dataPayload{
		Type:       dataType,
		Version:    dataVersion,
		ExportedAt: now.Format(time.RFC3339),
		Proxies:    []dataProxy{},
		Accounts:   accounts,
	}
	return payload, warnings, nil
}

func parseCodexTabLine(index int, line string, now time.Time) (dataAccount, string, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != 4 {
		return dataAccount{}, "", fmt.Errorf("line %d: expected 4 tab-separated columns, got %d", index, len(fields))
	}

	accessToken := strings.TrimSpace(fields[0])
	email := strings.TrimSpace(fields[1])
	idToken := strings.TrimSpace(fields[2])
	refreshWithClient := strings.TrimSpace(fields[3])

	if accessToken == "" || email == "" || idToken == "" || refreshWithClient == "" {
		return dataAccount{}, "", fmt.Errorf("line %d: contains empty required column", index)
	}

	rtParts := strings.SplitN(refreshWithClient, "|", 2)
	if len(rtParts) != 2 {
		return dataAccount{}, "", fmt.Errorf("line %d: refresh/client column must be refresh_token|client_id", index)
	}
	refreshToken := strings.TrimSpace(rtParts[0])
	clientID := strings.TrimSpace(rtParts[1])
	if refreshToken == "" || clientID == "" {
		return dataAccount{}, "", fmt.Errorf("line %d: refresh_token or client_id is empty", index)
	}

	credentials := map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"id_token":      idToken,
		"client_id":     clientID,
		"email":         email,
		// Force sub2api to refresh immediately after import instead of trying
		// a stale imported access_token first.
		"expires_at": now.Add(-time.Minute).Format(time.RFC3339),
	}

	warning := ""
	claims, err := openai.DecodeIDToken(idToken)
	if err != nil {
		warning = fmt.Sprintf("line %d: id_token decode failed, imported without claim enrichment: %v", index, err)
	} else if claims != nil {
		if userInfo := claims.GetUserInfo(); userInfo != nil {
			setIfMissing(credentials, "email", strings.TrimSpace(userInfo.Email))
			setIfMissing(credentials, "chatgpt_account_id", strings.TrimSpace(userInfo.ChatGPTAccountID))
			setIfMissing(credentials, "chatgpt_user_id", strings.TrimSpace(userInfo.ChatGPTUserID))
			setIfMissing(credentials, "plan_type", strings.TrimSpace(userInfo.PlanType))
			setIfMissing(credentials, "organization_id", strings.TrimSpace(userInfo.OrganizationID))
		}
	}

	account := dataAccount{
		Name:        email,
		Platform:    "openai",
		Type:        "oauth",
		Credentials: credentials,
		Extra: map[string]any{
			"import_source": "codex_tab_txt",
			"imported_at":   now.Format(time.RFC3339),
		},
		Concurrency: defaultConcurrency,
		Priority:    defaultPriority,
	}

	return account, warning, nil
}

func setIfMissing(m map[string]any, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if existing, _ := m[key].(string); strings.TrimSpace(existing) == "" {
		m[key] = value
	}
}

func defaultOutputPath(inputPath string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	if base == "" {
		base = inputPath
	}
	return base + ".sub2api.json"
}
