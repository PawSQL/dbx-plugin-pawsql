package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestServer returns an httptest server that fakes the PawSQL API envelope
// ({code:200, data, message}) and records request paths for later assertions.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func envelope(data any) map[string]any {
	return map[string]any{"code": 200, "data": data, "message": "ok"}
}

func TestCreateAnalysisBody(t *testing.T) {
	var captured map[string]any
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/createAnalysisAsync" {
			_ = json.NewDecoder(r.Body).Decode(&captured)
			_ = json.NewEncoder(w).Encode(envelope(map[string]any{"analysisId": "a1"}))
			return
		}
		http.NotFound(w, r)
	})

	opts := DefaultOptions()
	client := NewClient(server.URL, opts.resolveKey(), "")
	_, err := client.createAnalysis("SELECT 1", "", "mysql", opts, true)
	if err != nil {
		t.Fatalf("createAnalysis: %v", err)
	}

	if captured["userKey"] != anonymousUserKey {
		t.Errorf("userKey = %v, want anonymous key", captured["userKey"])
	}
	if captured["workload"] != "SELECT 1" {
		t.Errorf("workload = %v", captured["workload"])
	}
	if captured["dbType"] != "mysql" {
		t.Errorf("dbType = %v", captured["dbType"])
	}
	if captured["queryMode"] != "plain_sql" {
		t.Errorf("queryMode = %v", captured["queryMode"])
	}
	if captured["validateFlag"] != true {
		t.Errorf("validateFlag = %v, want true", captured["validateFlag"])
	}
	if captured["closeRewrite"] != false {
		t.Errorf("closeRewrite = %v, want false (rewrite on)", captured["closeRewrite"])
	}
	if captured["maxMembersForIndexOnly"] != float64(4) {
		t.Errorf("maxMembersForIndexOnly = %v, want 4", captured["maxMembersForIndexOnly"])
	}
}

func TestOptimizeHappyPath(t *testing.T) {
	detail := map[string]any{
		"stmtText":         "SELECT * FROM t WHERE a = 1",
		"indexRecommended": []any{"CREATE INDEX idx_a ON t(a)"},
		"violationRule": []any{
			map[string]any{"ruleCode": "R1", "ruleNameEn": "full scan", "level": float64(0), "fragmentsStr": "SELECT * FROM t"},
		},
		"rewrittenQuery": []any{
			map[string]any{"ruleCode": "R2", "ruleNameEn": "rewrite", "rewrittenQueriesStr": "SELECT a FROM t WHERE a = 1"},
		},
		"validationDetails": map[string]any{
			"beforePlan": "Seq Scan", "afterPlan": "Index Scan", "performImprovePer": "50.0",
		},
	}

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/createAnalysisAsync":
			_ = json.NewEncoder(w).Encode(envelope(map[string]any{"analysisId": "a1"}))
		case "/api/v1/getAnalysisSummary":
			_ = json.NewEncoder(w).Encode(envelope(map[string]any{
				"status":       "success",
				"analysisName": "Test analysis",
				"basicSummary": map[string]any{"performanceImprove": 0.215},
				"summaryStatementInfo": []any{
					map[string]any{"analysisStmtId": "s1"},
				},
			}))
		case "/api/v1/getStatementDetails":
			_ = json.NewEncoder(w).Encode(envelope(detail))
		default:
			http.NotFound(w, r)
		}
	})

	opts := DefaultOptions()
	client := NewClient(server.URL, opts.resolveKey(), "")
	result, err := client.Optimize("SELECT * FROM t WHERE a = 1", "", "mysql", opts, true, nil)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}

	if result.StatementID != "s1" {
		t.Errorf("statementId = %q, want s1", result.StatementID)
	}
	if result.AnalysisName != "Test analysis" {
		t.Errorf("analysisName = %q", result.AnalysisName)
	}
	if result.ImprovePercent == nil || *result.ImprovePercent != 21.5 {
		t.Errorf("improvePercent = %v, want 21.5", result.ImprovePercent)
	}
	if len(result.Detail.Violations) != 1 || result.Detail.Violations[0].Level != 0 {
		t.Errorf("violations = %+v", result.Detail.Violations)
	}
	if len(result.Detail.Rewrites) != 1 {
		t.Errorf("rewrites = %+v", result.Detail.Rewrites)
	}
	if len(result.Detail.IndexRecommended) != 1 {
		t.Errorf("indexRecommended = %+v", result.Detail.IndexRecommended)
	}
	if result.Detail.Validation == nil || result.Detail.Validation.BeforePlan != "Seq Scan" {
		t.Errorf("validation = %+v", result.Detail.Validation)
	}
	if !strings.HasSuffix(result.DetailURL, "/statement/s1") {
		t.Errorf("detailURL = %q", result.DetailURL)
	}
	if !result.Validated {
		t.Errorf("validated = false, want true when validation details are present")
	}
}

func TestOptimizeBusinessError(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 500, "message": "boom"})
	})
	opts := DefaultOptions()
	client := NewClient(server.URL, opts.resolveKey(), "")
	_, err := client.Optimize("SELECT 1", "", "mysql", opts, true, nil)
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %q, want it to mention the server message", err.Error())
	}
}

func TestOptimizeRejectsEmptySQL(t *testing.T) {
	opts := DefaultOptions()
	client := NewClient(defaultServerURL, opts.resolveKey(), "")
	_, err := client.Optimize("   ", "", "mysql", opts, true, nil)
	if err == nil {
		t.Fatal("expected an error for blank SQL")
	}
}

func TestListWorkspaces(t *testing.T) {
	var captured map[string]any
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/listWorkspaces" {
			_ = json.NewDecoder(r.Body).Decode(&captured)
			_ = json.NewEncoder(w).Encode(envelope(map[string]any{
				"records": []any{
					map[string]any{"workspaceId": "ws-1", "workspaceName": "prod", "workspaceDefinitionId": "def-1"},
					map[string]any{"workspaceId": "ws-2", "workspaceName": "staging"},
				},
			}))
			return
		}
		http.NotFound(w, r)
	})

	opts := DefaultOptions()
	client := NewClient(server.URL, opts.resolveKey(), "")
	workspaces, err := client.listWorkspaces()
	if err != nil {
		t.Fatalf("listWorkspaces: %v", err)
	}
	if len(workspaces) != 2 {
		t.Fatalf("workspaces = %d, want 2", len(workspaces))
	}
	if workspaces[0].WorkspaceID != "ws-1" || workspaces[0].WorkspaceName != "prod" || workspaces[0].WorkspaceDefinitionID != "def-1" {
		t.Errorf("workspace[0] = %+v", workspaces[0])
	}
	if workspaces[1].WorkspaceID != "ws-2" || workspaces[1].WorkspaceName != "staging" {
		t.Errorf("workspace[1] = %+v", workspaces[1])
	}
	if captured["userKey"] != anonymousUserKey {
		t.Errorf("userKey = %v, want anonymous key", captured["userKey"])
	}
}

func TestOptimizeSendsWorkspace(t *testing.T) {
	var captured map[string]any
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/createAnalysisAsync":
			_ = json.NewDecoder(r.Body).Decode(&captured)
			_ = json.NewEncoder(w).Encode(envelope(map[string]any{"analysisId": "a1"}))
		case "/api/v1/getAnalysisSummary":
			_ = json.NewEncoder(w).Encode(envelope(map[string]any{
				"status": "success",
				"summaryStatementInfo": []any{map[string]any{"analysisStmtId": "s1"}},
			}))
		case "/api/v1/getStatementDetails":
			_ = json.NewEncoder(w).Encode(envelope(map[string]any{"stmtText": "SELECT 1"}))
		default:
			http.NotFound(w, r)
		}
	})

	opts := DefaultOptions()
	client := NewClient(server.URL, opts.resolveKey(), "")
	result, err := client.Optimize("SELECT 1", "ws-9", "mysql", opts, true, nil)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if captured["workspace"] != "ws-9" {
		t.Errorf("workspace = %v, want ws-9", captured["workspace"])
	}
	if result.Validated {
		t.Errorf("validated = true, want false when no validation details are returned")
	}
}

func TestResolveError(t *testing.T) {
	// Server error codes pass through verbatim (the UI maps them via error.<code>).
	if got := resolveError("analysis.over.limit"); got != "analysis.over.limit" {
		t.Errorf("resolveError(code) = %q, want passthrough", got)
	}
	// An empty message is the generic unknown-error code.
	if got := resolveError(""); got != "unknown.error" {
		t.Errorf("resolveError(\"\") = %q, want unknown.error", got)
	}
}

func TestOptimizeValidatedRequiresImproveAndPlan(t *testing.T) {
	// "Validated" needs a returned improvement ratio AND both before/after
	// execution plans; missing any of the three is not proof that a real
	// What-If run measured it.
	cases := []struct {
		name           string
		performanceImp any
		validation     map[string]any
		wantValidated  bool
	}{
		{
			name:           "improve with both plans is validated",
			performanceImp: 0.5,
			validation:     map[string]any{"beforePlan": "Seq Scan", "afterPlan": "Index Scan"},
			wantValidated:  true,
		},
		{name: "improve without plan details stays estimated", performanceImp: 0.5, wantValidated: false},
		{name: "plan details without an improve ratio stays estimated", validation: map[string]any{"beforePlan": "Seq Scan", "afterPlan": "Index Scan"}, wantValidated: false},
		{name: "improve with only the before plan stays estimated", performanceImp: 0.5, validation: map[string]any{"beforePlan": "Seq Scan"}, wantValidated: false},
		{name: "improve with only the after plan stays estimated", performanceImp: 0.5, validation: map[string]any{"afterPlan": "Index Scan"}, wantValidated: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			summary := map[string]any{
				"status":               "success",
				"summaryStatementInfo": []any{map[string]any{"analysisStmtId": "s1"}},
			}
			if tc.performanceImp != nil {
				summary["basicSummary"] = map[string]any{"performanceImprove": tc.performanceImp}
			}
			detail := map[string]any{"stmtText": "SELECT 1"}
			if tc.validation != nil {
				detail["validationDetails"] = tc.validation
			}
			server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/createAnalysisAsync":
					_ = json.NewEncoder(w).Encode(envelope(map[string]any{"analysisId": "a1"}))
				case "/api/v1/getAnalysisSummary":
					_ = json.NewEncoder(w).Encode(envelope(summary))
				case "/api/v1/getStatementDetails":
					_ = json.NewEncoder(w).Encode(envelope(detail))
				default:
					http.NotFound(w, r)
				}
			})
			opts := DefaultOptions()
			client := NewClient(server.URL, opts.resolveKey(), "")
			result, err := client.Optimize("SELECT 1", "", "mysql", opts, true, nil)
			if err != nil {
				t.Fatalf("Optimize: %v", err)
			}
			if result.Validated != tc.wantValidated {
				t.Errorf("validated = %v, want %v", result.Validated, tc.wantValidated)
			}
		})
	}
}

func TestStringList(t *testing.T) {
	asJSONArray := map[string]any{"v": []any{"a", "b"}}
	asEncodedString := map[string]any{"v": `["a","b"]`}
	asPlainString := map[string]any{"v": "plain"}

	if got := stringList(asJSONArray, "v"); len(got) != 2 || got[0] != "a" {
		t.Errorf("array case = %v", got)
	}
	if got := stringList(asEncodedString, "v"); len(got) != 2 || got[1] != "b" {
		t.Errorf("encoded-string case = %v", got)
	}
	if got := stringList(asPlainString, "v"); len(got) != 1 || got[0] != "plain" {
		t.Errorf("plain-string case = %v", got)
	}
	if got := stringList(map[string]any{}, "missing"); got != nil {
		t.Errorf("missing case = %v, want nil", got)
	}
}

func TestParseDetail(t *testing.T) {
	data := map[string]any{
		"stmtText":         "SELECT * FROM t",
		"indexRecommended": []any{"CREATE INDEX i ON t(a)"},
		"violationRule": []any{
			map[string]any{"ruleCode": "X", "ruleNameZh": "全表扫描", "ruleNameEn": "full scan", "level": float64(1), "fragmentsStr": `["SELECT * FROM t"]`},
		},
		"rewrittenQuery": []any{
			map[string]any{"ruleNameEn": "rewrite", "rewrittenQueriesStr": "SELECT a FROM t", "violatedQueriesStr": []any{"SELECT * FROM t"}},
		},
		"validationDetails": map[string]any{"beforePlan": "p1", "afterPlan": "p2"},
	}
	detail := parseDetail(data)
	if len(detail.Violations) != 1 || detail.Violations[0].Level != 1 {
		t.Errorf("violations = %+v", detail.Violations)
	}
	if detail.Violations[0].NameZh != "全表扫描" {
		t.Errorf("nameZh = %q", detail.Violations[0].NameZh)
	}
	if len(detail.Violations[0].Fragments) != 1 || detail.Violations[0].Fragments[0] != "SELECT * FROM t" {
		t.Errorf("fragments = %+v", detail.Violations[0].Fragments)
	}
	if len(detail.Rewrites) != 1 || detail.Rewrites[0].Rewritten[0] != "SELECT a FROM t" {
		t.Errorf("rewrites = %+v", detail.Rewrites)
	}
	if detail.Validation == nil || detail.Validation.AfterPlan != "p2" {
		t.Errorf("validation = %+v", detail.Validation)
	}
}
