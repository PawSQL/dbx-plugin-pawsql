// Command dbx-plugin-pawsql is the native sidecar of the DBX "PawSQL SQL
// Optimizer" workbench plugin.
//
// The sandboxed plugin UI cannot open sockets on its own (the workbench iframe
// runs under a restrictive CSP), so every byte of PawSQL traffic travels over
// protocol v1 (JSON Lines on stdin/stdout) to this process, which performs the
// actual HTTPS request.
//
// stdout is reserved for protocol frames; all diagnostics go to stderr.
package main

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	dbxpluginsdk "github.com/t8y2/dbx/plugins/sdk/go/dbx-plugin-sdk"
)

const (
	pluginID      = "com.pawsql.dbx"
	pluginVersion = "1.0.0"
)

type plugin struct {
	mu  sync.Mutex
	cfg Config
}

func (p *plugin) Handle(
	_ dbxpluginsdk.RequestContext,
	method string,
	params json.RawMessage,
	emitter *dbxpluginsdk.Emitter,
) (any, *dbxpluginsdk.PluginError) {
	switch method {
	case "plugin/ping":
		return p.handlePing()
	case "pawsql/test":
		return p.handleTest(params)
	case "pawsql/connect":
		return p.handleConnect(params)
	case "pawsql/disconnect":
		return p.handleDisconnect()
	case "pawsql/listWorkspaces":
		return p.handleListWorkspaces()
	case "pawsql/getConfig":
		return p.handleGetConfig()
	case "pawsql/saveConfig":
		return p.handleSaveConfig(params)
	case "pawsql/optimize":
		return p.handleOptimize(params, emitter)
	case "pawsql/openExternal":
		return p.handleOpenExternal(params)
	default:
		return nil, dbxpluginsdk.MethodNotFound(method)
	}
}

// client builds a Client from the current config.
func (p *plugin) client() *Client {
	o := p.cfg.Options
	return NewClient(o.ServerURL, o.resolveKey(), o.FrontendURL)
}

func (p *plugin) handlePing() (any, *dbxpluginsdk.PluginError) {
	return map[string]any{
		"ok":      true,
		"plugin":  pluginID,
		"version": pluginVersion,
	}, nil
}

// handleTest verifies the server is reachable. In anonymous mode a plain ping is
// enough; in signed-in mode the account key is used (but no API call validates
// it beyond reachability, mirroring the DBeaver plugin).
func (p *plugin) handleTest(params json.RawMessage) (any, *dbxpluginsdk.PluginError) {
	var req struct {
		ServerURL string `json:"serverUrl"`
	}
	if len(params) > 0 {
		_ = json.Unmarshal(params, &req)
	}
	client := p.client()
	if req.ServerURL != "" {
		client = NewClient(req.ServerURL, p.cfg.Options.resolveKey(), p.cfg.Options.FrontendURL)
	}
	if err := client.ping(); err != nil {
		return nil, dbxpluginsdk.NewError(-32000, err.Error())
	}
	return map[string]any{
		"success":   true,
		"serverUrl": client.baseURL,
		"mode":      modeName(p.cfg.Options),
	}, nil
}

// handleConnect logs in with email/password, stores the returned key and marks
// the plugin signed-in.
func (p *plugin) handleConnect(params json.RawMessage) (any, *dbxpluginsdk.PluginError) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if pluginErr := decodeParams(params, &req); pluginErr != nil {
		return nil, pluginErr
	}
	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		return nil, dbxpluginsdk.NewError(-32602, "email and password are required")
	}

	client := p.client()
	if err := client.connect(strings.TrimSpace(req.Email), req.Password); err != nil {
		return nil, dbxpluginsdk.NewError(-32000, err.Error())
	}

	p.mu.Lock()
	p.cfg.Options.Anonymous = false
	p.cfg.Options.Email = strings.TrimSpace(req.Email)
	p.cfg.Options.UserKey = client.userKey
	p.cfg.Options.FrontendURL = client.frontendURL
	p.mu.Unlock()
	if err := saveConfig(p.cfg); err != nil {
		return nil, dbxpluginsdk.NewError(-32000, "failed to save configuration: "+err.Error())
	}
	return map[string]any{
		"success": true,
		"email":   p.cfg.Options.Email,
		"mode":    "signed-in",
	}, nil
}

// handleDisconnect reverts to anonymous mode and forgets the stored key.
func (p *plugin) handleDisconnect() (any, *dbxpluginsdk.PluginError) {
	p.mu.Lock()
	p.cfg.Options.Anonymous = true
	p.cfg.Options.UserKey = ""
	p.cfg.Options.FrontendURL = ""
	p.mu.Unlock()
	if err := saveConfig(p.cfg); err != nil {
		return nil, dbxpluginsdk.NewError(-32000, "failed to save configuration: "+err.Error())
	}
	return map[string]any{"success": true, "mode": "anonymous"}, nil
}

// handleListWorkspaces returns the signed-in account's PawSQL workspaces. It
// requires a login; anonymous mode has no account and no workspaces.
func (p *plugin) handleListWorkspaces() (any, *dbxpluginsdk.PluginError) {
	o := p.cfg.Options
	if o.Anonymous || o.UserKey == "" {
		return nil, dbxpluginsdk.NewError(-32000, "sign in to PawSQL to list workspaces")
	}
	workspaces, err := p.client().listWorkspaces()
	if err != nil {
		return nil, dbxpluginsdk.NewError(-32000, err.Error())
	}
	return map[string]any{"workspaces": workspaces}, nil
}

// handleGetConfig returns the current config without secrets. The userKey is
// reported only as a boolean so the frontend can render the auth state.
func (p *plugin) handleGetConfig() (any, *dbxpluginsdk.PluginError) {
	o := p.cfg.Options
	return map[string]any{
		"serverUrl":    o.ServerURL,
		"anonymous":    o.Anonymous,
		"email":        o.Email,
		"hasUserKey":   o.UserKey != "" && o.UserKey != anonymousUserKey,
		"validate":     o.Validate,
		"analyze":      o.Analyze,
		"rewrite":      o.Rewrite,
		"dedupIndex":   o.DedupIndex,
		"indexOnly":    o.IndexOnly,
		"updateStats":  o.UpdateStats,
	}, nil
}

// handleSaveConfig updates the server URL and optimization switches. It never
// accepts a userKey/password from the frontend; auth is changed via connect.
func (p *plugin) handleSaveConfig(params json.RawMessage) (any, *dbxpluginsdk.PluginError) {
	var req struct {
		ServerURL   string `json:"serverUrl"`
		Validate    *bool  `json:"validate"`
		Analyze     *bool  `json:"analyze"`
		Rewrite     *bool  `json:"rewrite"`
		DedupIndex  *bool  `json:"dedupIndex"`
		IndexOnly   *bool  `json:"indexOnly"`
		UpdateStats *bool  `json:"updateStats"`
	}
	if pluginErr := decodeParams(params, &req); pluginErr != nil {
		return nil, pluginErr
	}

	p.mu.Lock()
	if strings.TrimSpace(req.ServerURL) != "" {
		p.cfg.Options.ServerURL = strings.TrimRight(strings.TrimSpace(req.ServerURL), "/")
	}
	setBool(&p.cfg.Options.Validate, req.Validate)
	setBool(&p.cfg.Options.Analyze, req.Analyze)
	setBool(&p.cfg.Options.Rewrite, req.Rewrite)
	setBool(&p.cfg.Options.DedupIndex, req.DedupIndex)
	setBool(&p.cfg.Options.IndexOnly, req.IndexOnly)
	setBool(&p.cfg.Options.UpdateStats, req.UpdateStats)
	p.mu.Unlock()

	if err := saveConfig(p.cfg); err != nil {
		return nil, dbxpluginsdk.NewError(-32000, "failed to save configuration: "+err.Error())
	}
	return map[string]any{"success": true}, nil
}

// handleOptimize runs a full optimization and streams progress events on
// "pawsql/progress" before returning the normalized report.
func (p *plugin) handleOptimize(params json.RawMessage, emitter *dbxpluginsdk.Emitter) (any, *dbxpluginsdk.PluginError) {
	var req struct {
		SQL         string `json:"sql"`
		DBType      string `json:"dbType"`
		WorkspaceID string `json:"workspaceId"`
	}
	if pluginErr := decodeParams(params, &req); pluginErr != nil {
		return nil, pluginErr
	}
	if strings.TrimSpace(req.SQL) == "" {
		return nil, dbxpluginsdk.NewError(-32602, "sql is required")
	}

	progress := func(step Progress) {
		if emitter != nil {
			_ = emitter.Event("pawsql/progress", step)
		}
	}

	result, err := p.client().Optimize(req.SQL, strings.TrimSpace(req.WorkspaceID), strings.TrimSpace(req.DBType), p.cfg.Options, p.cfg.Options.Validate, progress)
	if err != nil {
		return nil, dbxpluginsdk.NewError(-32000, err.Error())
	}
	return result, nil
}

// handleOpenExternal opens a URL in the user's default browser.
func (p *plugin) handleOpenExternal(params json.RawMessage) (any, *dbxpluginsdk.PluginError) {
	var req struct {
		URL string `json:"url"`
	}
	if len(params) > 0 {
		_ = json.Unmarshal(params, &req)
	}
	if err := openURL(req.URL); err != nil {
		return nil, dbxpluginsdk.NewError(-32000, err.Error())
	}
	return map[string]any{"success": true}, nil
}

func modeName(o Options) string {
	if o.Anonymous {
		return "anonymous"
	}
	return "signed-in"
}

func setBool(dst *bool, src *bool) {
	if src != nil {
		*dst = *src
	}
}

func decodeParams(params json.RawMessage, target any) *dbxpluginsdk.PluginError {
	if len(params) == 0 || string(params) == "null" {
		return dbxpluginsdk.NewError(-32602, "Missing request parameters")
	}
	if err := json.Unmarshal(params, target); err != nil {
		return dbxpluginsdk.NewError(-32602, "Invalid request parameters: "+err.Error())
	}
	return nil
}

func main() {
	cfg := loadConfig()
	instance := &plugin{cfg: cfg}
	metadata := dbxpluginsdk.Metadata{
		ID:           pluginID,
		Version:      pluginVersion,
		Capabilities: []string{"events"},
	}
	server := dbxpluginsdk.NewServer(metadata, instance)
	if err := server.Serve(); err != nil {
		log.Fatal(err)
	}
}
