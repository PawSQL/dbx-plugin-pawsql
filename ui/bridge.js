/* Thin wrapper around `window.dbxPlugin`. Every call the workbench makes over the
   DBX Host Bridge funnels through here, so the rest of the UI stays transport
   agnostic and degrades cleanly when no backend is attached. */

(function () {
  const plugin = typeof window.dbxPlugin !== "undefined" ? window.dbxPlugin : null;

  const MAX_INVOKE_TIMEOUT = 120000;
  const PROGRESS_LISTENERS = new Set();
  const CONTEXT_LISTENERS = new Set();

  /* DBX connection `db_type` -> PawSQL `dbType` code. Only the engines PawSQL
     understands are mapped; everything else stays manual (null). */
  const DBX_TO_PAWSQL = {
    mysql: "mysql",
    starrocks: "mysql",
    postgres: "postgres",
    redshift: "postgres",
    opengauss: "opengauss",
    oracle: "oracle",
    kingbase: "kingbase",
    hive: "hive",
    dameng: "dameng",
    gaussdb: "gaussdbx",
    sqlserver: "mssql",
    db2: "db2",
    "oceanbase-oracle": "oracle",
    jdbc: "universal"
  };

  let backendReady = false;
  let backendError = "";
  let locale = "en";
  let appearance = "light";
  let context = null;

  const ready = (async () => {
    if (!plugin) {
      backendError = "no-host";
      return false;
    }
    try {
      await plugin.ready;
      context = plugin.context || null;
      locale = HC.i18n.setLocale(plugin.locale);
      appearance = readAppearance(plugin.theme) || appearance;
      applyAppearance(appearance);
    } catch (error) {
      backendError = String((error && error.message) || error);
      return false;
    }
    if (plugin.onContext) {
      plugin.onContext((ctx) => {
        context = ctx || null;
        CONTEXT_LISTENERS.forEach((listener) => {
          try { listener(context); } catch (error) { /* listener errors must not break the stream */ }
        });
      });
    }
    if (plugin.onEvent) {
      plugin.onEvent((message) => {
        const method = message && (message.method || (message.event && message.event.method));
        const params = (message && message.params) || (message.event && message.event.params) || {};
        if (method === "pawsql/progress") {
          PROGRESS_LISTENERS.forEach((listener) => {
            try { listener(params); } catch (error) { /* listener errors must not break the stream */ }
          });
        }
      });
    }
    try {
      await plugin.invoke("plugin/ping", {}, { timeoutMs: 5000 });
      backendReady = true;
    } catch (error) {
      backendReady = false;
      backendError = String((error && error.message) || error);
    }
    return backendReady;
  })();

  function readAppearance(theme) {
    if (!theme) return null;
    if (typeof theme === "string") return theme;
    return theme.appearance || null;
  }

  function applyAppearance(value) {
    if (!value) return;
    appearance = value;
    document.documentElement.dataset.dbxTheme = value;
  }

  function onEnv(listener) {
    const handler = (event) => {
      const detail = (event && event.detail) || {};
      if (detail.locale) locale = HC.i18n.setLocale(detail.locale);
      const next = readAppearance(detail.theme);
      if (next) applyAppearance(next);
      listener({ locale, appearance });
    };
    document.addEventListener("dbx-plugin-env", handler);
    window.addEventListener("dbx-plugin-env", handler);
    return () => {
      document.removeEventListener("dbx-plugin-env", handler);
      window.removeEventListener("dbx-plugin-env", handler);
    };
  }

  function onProgress(listener) {
    PROGRESS_LISTENERS.add(listener);
    return () => PROGRESS_LISTENERS.delete(listener);
  }

  function mapDbType(dbxType) {
    if (!dbxType) return null;
    return DBX_TO_PAWSQL[String(dbxType).toLowerCase()] || null;
  }

  function detectDbType(ctx) {
    if (!ctx || typeof ctx !== "object") return null;
    return mapDbType(ctx.dbType);
  }

  function getContext() {
    return context;
  }

  function onContext(listener) {
    CONTEXT_LISTENERS.add(listener);
    return () => CONTEXT_LISTENERS.delete(listener);
  }

  function requireBackend() {
    if (!plugin) throw new Error(HC.i18n.t("err.backendHint"));
    return plugin;
  }

  async function invoke(method, params, timeoutMs) {
    const bridge = requireBackend();
    return bridge.invoke(method, params, { timeoutMs: Math.min(timeoutMs || 30000, MAX_INVOKE_TIMEOUT) });
  }

  /* ------------------------------------------------- PawSQL sidecar methods -- */

  function ping() {
    return invoke("plugin/ping", {}, 5000);
  }

  function test(serverUrl) {
    return invoke("pawsql/test", { serverUrl }, 15000);
  }

  function connect(email, password) {
    return invoke("pawsql/connect", { email, password }, 15000);
  }

  function disconnect() {
    return invoke("pawsql/disconnect", {}, 10000);
  }

  function listWorkspaces() {
    return invoke("pawsql/listWorkspaces", {}, 20000);
  }

  function getConfig() {
    return invoke("pawsql/getConfig", {}, 10000);
  }

  function saveConfig(config) {
    return invoke("pawsql/saveConfig", config, 10000);
  }

  /* Optimization runs longer than the default timeout: use the full invoke cap
     so the sidecar has headroom up to its 90s analysis window. A non-empty
     workspaceId selects schema-aware analysis. */
  function optimize(sql, dbType, workspaceId) {
    return invoke("pawsql/optimize", { sql, dbType, workspaceId: workspaceId || "" }, MAX_INVOKE_TIMEOUT);
  }

  /* Opens a URL in the default browser via the sidecar (the sandboxed iframe
     cannot navigate to an external page on its own). */
  function openExternal(url) {
    return invoke("pawsql/openExternal", { url }, 10000);
  }

  async function copy(text) {
    if (plugin && plugin.copy) {
      try {
        await plugin.copy(text);
        return true;
      } catch (error) { /* fall back to the DOM path in util.copyText */ }
    }
    return false;
  }

  window.HC.bridge = {
    get backendReady() { return backendReady; },
    get backendError() { return backendError; },
    get locale() { return locale; },
    get appearance() { return appearance; },
    get hasHost() { return !!plugin; },
    ready,
    onEnv,
    onProgress,
    onContext,
    getContext,
    mapDbType,
    detectDbType,
    invoke,
    ping,
    test,
    connect,
    disconnect,
    listWorkspaces,
    getConfig,
    saveConfig,
    optimize,
    openExternal,
    copy,
    MAX_INVOKE_TIMEOUT
  };
})();
