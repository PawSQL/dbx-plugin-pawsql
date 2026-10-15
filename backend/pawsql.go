// Package main implements the PawSQL sidecar: a thin HTTPS client for the
// PawSQL server API (/api/v1/*) plus the orchestration that turns a pasted SQL
// statement into a structured optimization report. The API contract mirrors the
// PawSQL DBeaver/JetBrains plugin exactly.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

/* ------------------------------------------------------------------ consts -- */

const (
	defaultServerURL = "https://www.pawsql.com"
	// AnonymousUserKey is the shared key used when the plugin runs without a
	// login. It is a compile-time constant, never written to disk.
	anonymousUserKey = "23927924-010A7F59-69ABCF01-B1E1D303"

	apiVersion = "/api/v1"

	maxSQLLength      = 10000
	pollInterval      = 1 * time.Second
	analysisWaitLimit = 90 * time.Second // below the 120s Host Bridge invoke cap

	connectTimeout = 10 * time.Second
	readTimeout    = 120 * time.Second
)

// Max members tuning knobs (aligned with the DBeaver plugin defaults).
const (
	maxMembers4IndexOnly = 4
	maxMembers           = 6
	maxPerTable          = 5
)

/* ------------------------------------------------------------------ errors -- */

// PawSQLError is a user-facing error from the PawSQL API or the orchestration.
type PawSQLError struct {
	Message string
}

func (e *PawSQLError) Error() string { return e.Message }

func pawError(format string, args ...any) *PawSQLError {
	return &PawSQLError{Message: fmt.Sprintf(format, args...)}
}

/* ----------------------------------------------------------------- options -- */

// Options carries the user-configurable behaviour of an optimization run. It is
// persisted to config.json and shipped with every createAnalysis request.
type Options struct {
	ServerURL  string `json:"serverUrl"`
	Anonymous  bool   `json:"anonymous"`
	Email      string `json:"email,omitempty"`
	UserKey    string `json:"userKey,omitempty"`
	FrontendURL string `json:"frontendUrl,omitempty"`

	Validate    bool `json:"validate"`
	Analyze     bool `json:"analyze"`
	Rewrite     bool `json:"rewrite"`
	DedupIndex  bool `json:"dedupIndex"`
	IndexOnly   bool `json:"indexOnly"`
	UpdateStats bool `json:"updateStats"`
}

// DefaultOptions returns the plugin defaults, aligned with PawSQLConstants.
func DefaultOptions() Options {
	return Options{
		ServerURL:   defaultServerURL,
		Anonymous:   true,
		Validate:    true,
		Analyze:     true,
		Rewrite:     true,
		DedupIndex:  true,
		IndexOnly:   true,
		UpdateStats: true,
	}
}

// resolveKey returns the API key to use: the shared anonymous key when in
// anonymous mode, otherwise the stored account key (which may be empty, meaning
// "not signed in yet").
func (o Options) resolveKey() string {
	if o.Anonymous {
		return anonymousUserKey
	}
	return o.UserKey
}

/* ------------------------------------------------------------------ client -- */

type Client struct {
	baseURL     string
	userKey     string
	frontendURL string
	http        *http.Client
}

func NewClient(baseURL, userKey, frontendURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultServerURL
	}
	return &Client{
		baseURL:     baseURL,
		userKey:     userKey,
		frontendURL: frontendURL,
		http:        &http.Client{Timeout: readTimeout},
	}
}

// ping verifies only that the server is reachable over HTTP; any HTTP response
// counts as reachable. Used by the anonymous-mode test connection.
func (c *Client) ping() error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL, nil)
	if err != nil {
		return pawError("invalid server URL: %v", err)
	}
	client := &http.Client{Timeout: connectTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return pawError("cannot reach %s: %v", c.baseURL, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// connect posts /getUserKey to validate credentials and stores the returned key
// and frontend URL on the client.
func (c *Client) connect(email, password string) error {
	data, err := c.postObject("/getUserKey", map[string]any{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return err
	}
	key := getString(data, "apikey")
	if key == "" {
		return pawError("the server did not return an API key")
	}
	c.userKey = key
	c.frontendURL = getString(data, "frontendUrl")
	return nil
}

// Workspace is a PawSQL workspace as returned by /listWorkspaces.
type Workspace struct {
	WorkspaceID           string `json:"workspaceId"`
	WorkspaceName         string `json:"workspaceName"`
	WorkspaceDefinitionID string `json:"workspaceDefinitionId"`
}

// listWorkspaces posts /listWorkspaces and returns the account's workspaces.
// Selecting one of these makes createAnalysis run a schema-aware analysis.
func (c *Client) listWorkspaces() ([]Workspace, error) {
	data, err := c.postObject("/listWorkspaces", map[string]any{
		"userKey": c.userKey,
	})
	if err != nil {
		return nil, err
	}
	records, ok := data["records"].([]any)
	if !ok {
		return []Workspace{}, nil
	}
	out := make([]Workspace, 0, len(records))
	for _, item := range records {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, Workspace{
			WorkspaceID:           getString(obj, "workspaceId"),
			WorkspaceName:         getString(obj, "workspaceName"),
			WorkspaceDefinitionID: getString(obj, "workspaceDefinitionId"),
		})
	}
	return out, nil
}

// createAnalysis posts /createAnalysisAsync and returns the analysisId. A null
// or empty workspaceId runs a schema-only (workspace-less) analysis, which
// requires a valid dbType.
func (c *Client) createAnalysis(sqlText, workspaceID, dbType string, opts Options, validate bool) (string, error) {
	body := map[string]any{
		"userKey":                       c.userKey,
		"workload":                      sqlText,
		"singleQueryFlag":               false,
		"queryMode":                     "plain_sql",
		"validateFlag":                  validate,
		"analyzeFlag":                   opts.Analyze,
		"indexOnly":                     opts.IndexOnly,
		"closeRewrite":                  !opts.Rewrite,
		"deduplicateFlag":               opts.DedupIndex,
		"updateStatsBeforeValidationFlag": opts.UpdateStats,
		"maxMembersForIndexOnly":        maxMembers4IndexOnly,
		"maxMembers":                    maxMembers,
		"maxPerTable":                   maxPerTable,
	}
	if workspaceID != "" {
		body["workspace"] = workspaceID
	}
	if dbType != "" {
		body["dbType"] = dbType
	}
	data, err := c.postObject("/createAnalysisAsync", body)
	if err != nil {
		return "", err
	}
	analysisID := getString(data, "analysisId")
	if analysisID == "" {
		return "", pawError("the server did not return an analysis id")
	}
	return analysisID, nil
}

// getAnalysisSummary posts /getAnalysisSummary and returns the raw data object.
func (c *Client) getAnalysisSummary(analysisID string) (map[string]any, error) {
	return c.postObject("/getAnalysisSummary", map[string]any{
		"userKey":    c.userKey,
		"analysisId": analysisID,
	})
}

// getStatementDetails posts /getStatementDetails and returns the raw data object.
func (c *Client) getStatementDetails(statementID string) (map[string]any, error) {
	return c.postObject("/getStatementDetails", map[string]any{
		"userKey":         c.userKey,
		"analysisStmtId": statementID,
	})
}

/* ------------------------------------------------------------- post core ---- */

// postObject performs a JSON POST and unwraps the {code, data, message}
// envelope, returning the data object (or nil when data is absent/null).
func (c *Client) postObject(path string, body any) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, pawError("failed to encode request: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+apiVersion+path, bytes.NewReader(payload))
	if err != nil {
		return nil, pawError("invalid request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, pawError("cannot reach %s: %v", c.baseURL, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, pawError("failed to read response: %v", err)
	}
	if resp.StatusCode >= 400 {
		return nil, pawError("PawSQL server error (%d): %s", resp.StatusCode, truncate(string(raw)))
	}

	var root struct {
		Code    int            `json:"code"`
		Data    map[string]any `json:"data"`
		Message string         `json:"message"`
	}
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, pawError("unparsable server response")
	}
	if root.Code != 200 {
		return nil, pawError("%s", resolveError(root.Message))
	}
	return root.Data, nil
}

// resolveError maps the server envelope's `message` field to a stable error
// code. PawSQL returns an error code (not a message) in `message`; the UI maps
// it to human-readable text via `error.<code>`. The empty case is the generic
// unknown-error code.
func resolveError(message string) string {
	if message != "" {
		return message
	}
	return "unknown.error"
}

func truncate(s string) string {
	const max = 3000
	if len(s) > max {
		return s[:max] + "...[truncated]"
	}
	return s
}

/* ------------------------------------------------------------- optimize ---- */

// Progress is emitted by the sidecar while an optimization runs so the workbench
// can show live status instead of a spinner.
type Progress struct {
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

// progressFunc reports a stage change to the caller (typically via emitter.Event).
type progressFunc func(Progress)

// OptimizeResult is the fully-normalized report handed to the frontend.
type OptimizeResult struct {
	Detail         Detail    `json:"detail"`
	ImprovePercent *float64  `json:"improvePercent"`
	AnalysisName   string    `json:"analysisName"`
	StatementID    string    `json:"statementId"`
	DetailURL      string    `json:"detailUrl"`
	// Validated reports whether the improvement is a measured What-If result
	// rather than an estimate: the validate option was on AND the server
	// returned a performance-improve ratio (basicSummary.performanceImprove > 0)
	// AND both the before and after execution plans. Any one of these missing is
	// not proof that a real What-If run measured the improvement.
	Validated bool `json:"validated"`
}

// Optimize runs createAnalysis -> poll summary -> getStatementDetails and
// returns the normalized report. It is synchronous; the SDK dispatches it on a
// worker goroutine, so blocking is expected. Progress is reported via progress.
// A non-empty workspaceID selects schema-aware analysis; otherwise the dbType
// drives a schema-less (workspace-less) analysis.
func (c *Client) Optimize(sqlText, workspaceID, dbType string, opts Options, validate bool, progress progressFunc) (*OptimizeResult, error) {
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return nil, pawError("please paste the SQL to optimize")
	}
	if len(sqlText) > maxSQLLength {
		return nil, pawError("SQL is too long (%d chars; max %d)", len(sqlText), maxSQLLength)
	}

	if progress != nil {
		progress(Progress{Stage: "creating", Message: "submitting the analysis"})
	}
	analysisID, err := c.createAnalysis(sqlText, workspaceID, dbType, opts, validate)
	if err != nil {
		return nil, err
	}

	if progress != nil {
		progress(Progress{Stage: "running", Message: "waiting for the analysis to finish"})
	}
	summary, err := c.waitForSummary(analysisID, progress)
	if err != nil {
		return nil, err
	}

	analysisName := getString(summary, "analysisName")
	statementID := extractStatementID(summary)
	if statementID == "" {
		return nil, pawError("the analysis returned no statement details")
	}

	if progress != nil {
		progress(Progress{Stage: "details", Message: "fetching the optimization report"})
	}
	rawDetail, err := c.getStatementDetails(statementID)
	if err != nil {
		return nil, err
	}
	detail := parseDetail(rawDetail)
	// Index DDLs are reported in the summary (analysisIndexInfo), not in the
	// statement detail; merge them in when the detail has none.
	if len(detail.IndexRecommended) == 0 {
		detail.IndexRecommended = stringList(summary, "analysisIndexInfo")
	}

	improvePercent := extractImprovePercent(summary)

	frontend := strings.TrimRight(strings.TrimSpace(c.frontendURL), "/")
	if frontend == "" {
		frontend = c.baseURL
	}
	detailURL := frontend + "/statement/" + statementID

	return &OptimizeResult{
		Detail:         detail,
		ImprovePercent: improvePercent,
		AnalysisName:   analysisName,
		StatementID:    statementID,
		DetailURL:      detailURL,
		Validated:      validate && improvePercent != nil && detail.Validation.hasBothPlans(),
	}, nil
}

func (c *Client) waitForSummary(analysisID string, progress progressFunc) (map[string]any, error) {
	deadline := time.Now().Add(analysisWaitLimit)
	for {
		summary, err := c.getAnalysisSummary(analysisID)
		if err != nil {
			return nil, err
		}
		if isTerminal(summary) {
			return summary, nil
		}
		if time.Now().After(deadline) {
			return nil, pawError("analysis %s did not finish in time", analysisID)
		}
		time.Sleep(pollInterval)
	}
}

func isTerminal(summary map[string]any) bool {
	if summary == nil {
		return false
	}
	status := getString(summary, "status")
	if status == "" {
		return true
	}
	s := strings.ToLower(status)
	return strings.Contains(s, "success") || strings.Contains(s, "fail") ||
		strings.Contains(s, "error") || strings.Contains(s, "cancel")
}

func extractStatementID(summary map[string]any) string {
	items, ok := summary["summaryStatementInfo"].([]any)
	if !ok {
		return ""
	}
	for _, item := range items {
		if obj, ok := item.(map[string]any); ok {
			if id := getString(obj, "analysisStmtId"); id != "" {
				return id
			}
		}
	}
	return ""
}

func extractImprovePercent(summary map[string]any) *float64 {
	basic, ok := summary["basicSummary"].(map[string]any)
	if !ok {
		return nil
	}
	value, ok := basic["performanceImprove"].(float64)
	if !ok || value <= 0 {
		return nil
	}
	percent := value * 100
	return &percent
}

/* -------------------------------------------------------------- data model - */

// Violation mirrors a single rule violation from violationRule[].
type Violation struct {
	RuleCode  string   `json:"ruleCode"`
	NameZh    string   `json:"nameZh"`
	NameEn    string   `json:"nameEn"`
	Level     int      `json:"level"` // 0=Critical, 1=Warning, 2=Info
	Fragments []string `json:"fragments"`
}

// Rewrite mirrors a single rewrite from rewrittenQuery[].
type Rewrite struct {
	RuleCode  string   `json:"ruleCode"`
	NameZh    string   `json:"nameZh"`
	NameEn    string   `json:"nameEn"`
	Rewritten []string `json:"rewritten"`
	Violated  []string `json:"violated"`
}

// Validation mirrors validationDetails.
type Validation struct {
	BeforePlan     string `json:"beforePlan"`
	AfterPlan      string `json:"afterPlan"`
	BeforeCost     string `json:"beforeCost"`
	AfterCost      string `json:"afterCost"`
	ImprovePercent string `json:"improvePercent"`
}

// hasBothPlans reports whether both the before and after execution plans are
// present — the mark of a completed What-If run.
func (v *Validation) hasBothPlans() bool {
	return v != nil && v.BeforePlan != "" && v.AfterPlan != ""
}

// Detail is the normalized getStatementDetails response.
type Detail struct {
	StmtText         string      `json:"stmtText"`
	IndexRecommended []string    `json:"indexRecommended"`
	Violations       []Violation `json:"violations"`
	Rewrites         []Rewrite   `json:"rewrites"`
	Validation       *Validation `json:"validation,omitempty"`
}

func parseDetail(data map[string]any) Detail {
	detail := Detail{
		StmtText:         getString(data, "stmtText"),
		IndexRecommended: stringList(data, "indexRecommended"),
		Violations:       parseViolations(data["violationRule"]),
		Rewrites:         parseRewrites(data["rewrittenQuery"]),
		Validation:       parseValidation(data["validationDetails"]),
	}
	return detail
}

func parseViolations(value any) []Violation {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]Violation, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case map[string]any:
			v := Violation{
				RuleCode:  getString(typed, "ruleCode"),
				NameZh:    getString(typed, "ruleNameZh"),
				NameEn:    getString(typed, "ruleNameEn"),
				Level:     levelFrom(typed),
				Fragments: stringList(typed, "fragmentsStr"),
			}
			if v.RuleCode == "" && v.NameZh == "" && v.NameEn == "" {
				continue
			}
			out = append(out, v)
		case string:
			out = append(out, Violation{NameEn: typed, Level: 0})
		}
	}
	return out
}

func parseRewrites(value any) []Rewrite {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]Rewrite, 0, len(items))
	for _, item := range items {
		typed, ok := item.(map[string]any)
		if !ok {
			continue
		}
		r := Rewrite{
			RuleCode:  getString(typed, "ruleCode"),
			NameZh:    getString(typed, "ruleNameZh"),
			NameEn:    getString(typed, "ruleNameEn"),
			Rewritten: stringList(typed, "rewrittenQueriesStr"),
			Violated:  stringList(typed, "violatedQueriesStr"),
		}
		if r.RuleCode == "" && r.NameZh == "" && r.NameEn == "" &&
			len(r.Rewritten) == 0 && len(r.Violated) == 0 {
			continue
		}
		out = append(out, r)
	}
	return out
}

func parseValidation(value any) *Validation {
	typed, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	v := Validation{
		BeforePlan:     getString(typed, "beforePlan"),
		AfterPlan:      getString(typed, "afterPlan"),
		BeforeCost:     getString(typed, "beforeCost"),
		AfterCost:      getString(typed, "afterCost"),
		ImprovePercent: getString(typed, "performImprovePer"),
	}
	if v.BeforePlan == "" && v.AfterPlan == "" && v.BeforeCost == "" && v.ImprovePercent == "" {
		return nil
	}
	return &v
}

// levelFrom extracts a rule severity level; missing/unknown -> 0 (Critical).
func levelFrom(obj map[string]any) int {
	switch v := obj["level"].(type) {
	case float64:
		return int(v)
	case string:
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return 0
}

// stringList parses a field that is either a JSON array or a JSON-encoded string
// array (some PawSQL fields carry the array as a string, e.g. "[\"a\",\"b\"]").
func stringList(obj map[string]any, key string) []string {
	value, ok := obj[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case string:
		var arr []string
		if err := json.Unmarshal([]byte(typed), &arr); err == nil {
			return arr
		}
		return []string{typed}
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func getString(obj map[string]any, key string) string {
	if obj == nil {
		return ""
	}
	switch v := obj[key].(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case float64:
		return fmt.Sprintf("%g", v)
	}
	return ""
}
