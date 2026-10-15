/* Workbench bootstrap and wiring: loads config, drives the sidecar, and renders
   the optimization report. All transport goes through HC.bridge. */

(function () {
  const util = HC.util;
  const bridge = HC.bridge;
  const t = (key, params) => HC.i18n.t(key, params);

  /* Turn a bridge/API error into a display string, localising PawSQL server
     error codes (e.g. "analysis.over.limit") into human-readable text. */
  function serverMessage(error) {
    return HC.i18n.serverError(String((error && error.message) || error));
  }

  /* Canonical PawSQL database-type codes, ordered for the dropdown. Brand names
     are not localised; the code is what the server expects. */
  const DB_TYPES = [
    ["mysql", "MySQL"],
    ["postgres", "PostgreSQL"],
    ["opengauss", "openGauss"],
    ["oracle", "Oracle"],
    ["kingbase", "KingbaseES"],
    ["hive", "Hive"],
    ["dameng", "DM (Dameng)"],
    ["gaussdbx", "GaussDB"],
    ["oceanbase_mysql", "OceanBase (MySQL)"],
    ["mssql", "SQL Server"],
    ["db2", "DB2"],
    ["universal", "Universal"]
  ];

  let optimizing = false;
  let stopProgress = null;
  let workspaces = [];
  let selectedWorkspaceId = "";
  let workspaceAvailable = false;
  let anonymousMode = true;
  let lastCfg = { anonymous: true };

  function $id(id) { return document.getElementById(id); }

  function populateDbTypes() {
    const select = $id("dbtype-select");
    DB_TYPES.forEach(([code, label]) => {
      select.append(util.el("option", { value: code, text: label }));
    });
  }

  function dbTypeLabel(code) {
    const entry = DB_TYPES.find(([c]) => c === code);
    return entry ? entry[1] : code;
  }

  /* Auto-select the database type inferred from the workbench context (opened
     via a connection context menu). Manual selection stays untouched when the
     host provides no dbType. */
  function applyDetectedDbType(ctx) {
    if (optimizing) return;
    const code = bridge.detectDbType(ctx);
    const select = $id("dbtype-select");
    if (code && DB_TYPES.some(([c]) => c === code)) {
      select.value = code;
      $id("detected-dbtype").textContent = t("input.detectedDbType", { dbType: dbTypeLabel(code) });
    } else {
      $id("detected-dbtype").textContent = "";
    }
  }

  /* ---------------------------------------------------------- workspace ----- */

  function populateWorkspaces() {
    const select = $id("workspace-select");
    util.clear(select);
    select.append(util.el("option", { value: "", text: t("workspace.placeholder") }));
    workspaces.forEach((ws) => {
      select.append(util.el("option", {
        value: ws.workspaceId || "",
        text: ws.workspaceName || ws.workspaceId || ""
      }));
    });
    if (selectedWorkspaceId && workspaces.some((ws) => ws.workspaceId === selectedWorkspaceId)) {
      select.value = selectedWorkspaceId;
    }
  }

  function setWorkspaceHint(text) {
    $id("workspace-hint").textContent = text || "";
  }

  /* Load the signed-in account's workspaces. In anonymous mode the dropdown is
     hidden entirely (the database-type picker is shown instead). */
  async function refreshWorkspaces(cfg) {
    if (cfg.anonymous) {
      workspaces = [];
      selectedWorkspaceId = "";
      workspaceAvailable = false;
      populateWorkspaces();
      setWorkspaceHint("");
    } else {
      workspaceAvailable = true;
      setWorkspaceHint(t("workspace.loading"));
      try {
        const result = await bridge.listWorkspaces();
        workspaces = (result && result.workspaces) || [];
        if (selectedWorkspaceId && !workspaces.some((ws) => ws.workspaceId === selectedWorkspaceId)) {
          selectedWorkspaceId = "";
        }
        populateWorkspaces();
        setWorkspaceHint(workspaces.length === 0 ? t("workspace.emptyHint") : "");
      } catch (error) {
        setWorkspaceHint(t("workspace.loadFailed", { message: serverMessage(error) }));
      }
    }
    $id("workspace-select").disabled = !workspaceAvailable;
    $id("btn-refresh-workspaces").disabled = !workspaceAvailable;
  }

  function setOptimizing(value) {
    optimizing = value;
    const btn = $id("btn-optimize");
    btn.disabled = value;
    btn.textContent = value ? t("input.optimizing") : t("input.optimize");
    $id("sql-input").disabled = value;
    $id("dbtype-select").disabled = value;
    $id("workspace-select").disabled = value || !workspaceAvailable;
    $id("btn-refresh-workspaces").disabled = value || !workspaceAvailable;
    if (!value) setProgress("");
  }

  function setProgress(message) {
    $id("progress-line").textContent = message || "";
  }

  function progressStageLabel(stage) {
    switch (stage) {
      case "creating": return t("progress.creating");
      case "running": return t("progress.running");
      case "details": return t("progress.details");
      default: return "";
    }
  }

  function showReport(node) {
    const pane = $id("report-pane");
    util.clear(pane);
    pane.append(node);
    util.highlightSql(pane);
  }

  function showEmptyReport() {
    showReport(util.el("div", { class: "report-empty", text: t("report.empty") }));
  }

  function showError(message) {
    showReport(HC.report.renderError(message));
  }

  /* ------------------------------------------------------------ config ----- */

  function authControls() {
    return {
      anonymous: $id("cfg-anonymous"),
      email: $id("cfg-email"),
      password: $id("cfg-password"),
      signin: $id("btn-signin"),
      signout: $id("btn-signout"),
      authStatus: $id("auth-status")
    };
  }

  function refreshAuthUI(cfg) {
    const auth = authControls();
    const signedIn = !!(cfg && !cfg.anonymous);
    // The "Anonymous" checkbox is a real mode switch. Checked = anonymous
    // (login group hidden); unchecked = signed-in form (email/password +
    // Sign in), which becomes read-only once actually signed in.
    auth.anonymous.checked = anonymousMode;
    if (anonymousMode) {
      auth.email.disabled = true;
      auth.password.disabled = true;
      auth.signin.hidden = true;
      auth.signout.hidden = true;
      auth.authStatus.textContent = t("config.anonymousHint");
    } else if (signedIn) {
      auth.email.disabled = true;
      auth.password.disabled = true;
      auth.signin.hidden = true;
      auth.signout.hidden = false;
      auth.authStatus.textContent = t("config.signedInAs", { email: (cfg && cfg.email) || "" });
    } else {
      auth.email.disabled = false;
      auth.password.disabled = false;
      auth.signin.hidden = false;
      auth.signout.hidden = true;
      auth.authStatus.textContent = "";
    }
  }

  /* Show the database-type picker only in anonymous mode, and the workspace
     picker only when signed in (schema-aware analysis). */
  function applyModeUI(cfg) {
    const anonymous = !!cfg.anonymous;
    $id("dbtype-row").hidden = !anonymous;
    $id("detected-dbtype").hidden = !anonymous;
    $id("workspace-row").hidden = anonymous;
    $id("workspace-hint").hidden = anonymous;
  }

  async function loadConfig() {
    let cfg;
    try {
      cfg = await bridge.getConfig();
    } catch (error) {
      cfg = { anonymous: true };
    }
    lastCfg = cfg;
    anonymousMode = !!cfg.anonymous;
    $id("cfg-server-url").value = cfg.serverUrl || "";
    authControls().email.value = cfg.email || "";
    authControls().password.value = "";
    $id("cfg-validate").checked = cfg.validate !== false;
    $id("cfg-analyze").checked = cfg.analyze !== false;
    $id("cfg-rewrite").checked = cfg.rewrite !== false;
    $id("cfg-dedup").checked = cfg.dedupIndex !== false;
    $id("cfg-index-only").checked = cfg.indexOnly !== false;
    $id("cfg-update-stats").checked = cfg.updateStats !== false;
    refreshAuthUI(cfg);
    applyModeUI(cfg);
    await refreshWorkspaces(cfg);
  }

  function currentConfig() {
    return {
      serverUrl: $id("cfg-server-url").value.trim(),
      validate: $id("cfg-validate").checked,
      analyze: $id("cfg-analyze").checked,
      rewrite: $id("cfg-rewrite").checked,
      dedupIndex: $id("cfg-dedup").checked,
      indexOnly: $id("cfg-index-only").checked,
      updateStats: $id("cfg-update-stats").checked
    };
  }

  async function saveConfig() {
    try {
      await bridge.saveConfig(currentConfig());
      util.toast(t("config.saved"));
    } catch (error) {
      util.toast(t("config.saveFailed", { message: String((error && error.message) || error) }), "error");
    }
  }

  async function testConnection() {
    const serverUrl = $id("cfg-server-url").value.trim();
    if (!serverUrl) {
      util.toast(t("config.testRequiredAnon"), "error");
      return;
    }
    const btn = $id("btn-test");
    btn.disabled = true;
    btn.textContent = t("config.testing");
    try {
      await bridge.test(serverUrl);
      util.toast(t("config.testSuccess"));
    } catch (error) {
      util.toast(t("config.testFailed", { message: String((error && error.message) || error) }), "error");
    } finally {
      btn.disabled = false;
      btn.textContent = t("config.testConnection");
    }
  }

  async function signIn() {
    const auth = authControls();
    const email = auth.email.value.trim();
    const password = auth.password.value;
    if (!email || !password) {
      util.toast(t("config.testRequired"), "error");
      return;
    }
    auth.signin.disabled = true;
    try {
      const result = await bridge.connect(email, password);
      auth.password.value = "";
      await loadConfig();
      util.toast(t("config.testSuccess"));
    } catch (error) {
      util.toast(t("config.signinFailed", { message: serverMessage(error) }), "error");
    } finally {
      auth.signin.disabled = false;
    }
  }

  async function signOut() {
    try {
      await bridge.disconnect();
      await loadConfig();
    } catch (error) {
      util.toast(t("config.saveFailed", { message: String((error && error.message) || error) }), "error");
    }
  }

  /* ----------------------------------------------------------- optimize ----- */

  async function optimize() {
    if (optimizing) return;
    const sql = $id("sql-input").value;
    if (!sql || !sql.trim()) {
      util.toast(t("input.sqlEmpty"), "error");
      return;
    }
    const dbType = $id("dbtype-select").value;
    const workspaceId = $id("workspace-select").value || "";
    if (!$id("workspace-row").hidden && !workspaceId) {
      util.toast(t("workspace.required"), "error");
      return;
    }

    setOptimizing(true);
    stopProgress = bridge.onProgress((progress) => {
      setProgress(progressStageLabel(progress && progress.stage) || (progress && progress.message) || "");
    });

    try {
      const result = await bridge.optimize(sql.trim(), dbType, workspaceId);
      showReport(HC.report.render(result, { anonymous: !!(lastCfg && lastCfg.anonymous) }));
    } catch (error) {
      const message = serverMessage(error);
      if (message && message !== "Error") showError(message);
      else showError(t("err.invoke", { message }));
    } finally {
      if (stopProgress) { stopProgress(); stopProgress = null; }
      setOptimizing(false);
    }
  }

  /* ------------------------------------------------------------- wiring ----- */

  function wire() {
    $id("brand-name").textContent = t("brand");
    $id("label-sql").textContent = t("input.sqlLabel");
    $id("label-dbtype").textContent = t("input.dbType");
    $id("label-workspace").textContent = t("workspace.label");
    $id("btn-refresh-workspaces").textContent = t("workspace.refresh");
    $id("sql-input").placeholder = t("input.sqlPlaceholder");
    $id("btn-optimize").textContent = t("input.optimize");
    $id("config-title").textContent = t("config.title");
    $id("label-server-url").textContent = t("config.serverUrl");
    $id("label-anonymous").textContent = t("config.anonymous");
    $id("label-email").textContent = t("config.email");
    $id("label-password").textContent = t("config.password");
    $id("btn-signin").textContent = t("config.signin");
    $id("btn-signout").textContent = t("config.signout");
    $id("label-opt-group").textContent = t("config.optGroup");
    $id("label-validate").textContent = t("config.validate");
    $id("label-analyze").textContent = t("config.analyze");
    $id("label-rewrite").textContent = t("config.rewrite");
    $id("label-dedup").textContent = t("config.dedup");
    $id("label-index-only").textContent = t("config.indexOnly");
    $id("label-update-stats").textContent = t("config.updateStats");
    $id("btn-test").textContent = t("config.testConnection");
    $id("btn-save").textContent = t("config.save");
    $id("btn-toggle-config").textContent = t("config.expand");

    $id("btn-optimize").addEventListener("click", optimize);
    $id("btn-test").addEventListener("click", testConnection);
    $id("btn-save").addEventListener("click", saveConfig);
    $id("btn-signin").addEventListener("click", signIn);
    $id("btn-signout").addEventListener("click", signOut);
    $id("cfg-anonymous").addEventListener("change", async (event) => {
      anonymousMode = event.target.checked;
      if (anonymousMode && lastCfg && !lastCfg.anonymous) {
        await signOut();
      } else {
        refreshAuthUI(lastCfg);
      }
    });
    $id("workspace-select").addEventListener("change", (event) => {
      selectedWorkspaceId = event.target.value || "";
    });
    $id("btn-refresh-workspaces").addEventListener("click", async () => {
      try {
        const cfg = await bridge.getConfig();
        await refreshWorkspaces(cfg);
      } catch (error) {
        setWorkspaceHint(t("workspace.loadFailed", { message: serverMessage(error) }));
      }
    });

    $id("btn-toggle-config").addEventListener("click", () => {
      const body = $id("config-body");
      body.hidden = !body.hidden;
      $id("btn-toggle-config").textContent = body.hidden ? t("config.expand") : t("config.collapse");
    });

    bridge.onEnv(rewireLabels);
  }

  /* Re-apply translated labels after a locale change. */
  function rewireLabels() {
    $id("brand-name").textContent = t("brand");
    $id("label-sql").textContent = t("input.sqlLabel");
    $id("label-dbtype").textContent = t("input.dbType");
    $id("label-workspace").textContent = t("workspace.label");
    $id("btn-refresh-workspaces").textContent = t("workspace.refresh");
    $id("sql-input").placeholder = t("input.sqlPlaceholder");
    if (!optimizing) $id("btn-optimize").textContent = t("input.optimize");
    $id("config-title").textContent = t("config.title");
    $id("label-server-url").textContent = t("config.serverUrl");
    $id("label-anonymous").textContent = t("config.anonymous");
    $id("label-email").textContent = t("config.email");
    $id("label-password").textContent = t("config.password");
    $id("btn-signin").textContent = t("config.signin");
    $id("btn-signout").textContent = t("config.signout");
    $id("label-opt-group").textContent = t("config.optGroup");
    $id("label-validate").textContent = t("config.validate");
    $id("label-analyze").textContent = t("config.analyze");
    $id("label-rewrite").textContent = t("config.rewrite");
    $id("label-dedup").textContent = t("config.dedup");
    $id("label-index-only").textContent = t("config.indexOnly");
    $id("label-update-stats").textContent = t("config.updateStats");
    $id("btn-test").textContent = t("config.testConnection");
    $id("btn-save").textContent = t("config.save");
    $id("btn-toggle-config").textContent = $id("config-body").hidden ? t("config.expand") : t("config.collapse");
  }

  async function bootstrap() {
    wire();
    populateDbTypes();
    showEmptyReport();
    bridge.onContext(applyDetectedDbType);
    await bridge.ready;
    rewireLabels();
    applyDetectedDbType(bridge.getContext());
    if (bridge.backendReady) {
      await loadConfig();
    }
  }

  document.addEventListener("DOMContentLoaded", bootstrap);
})();
