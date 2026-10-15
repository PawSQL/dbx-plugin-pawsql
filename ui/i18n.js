/* Localised copy for the PawSQL workbench UI. The host never translates plugin
   UI, so every rendered string lives here and follows `dbxPlugin.locale`. */

(function () {
  const dictionaries = {
    "zh-CN": {
      "brand": "PawSQL SQL 优化器",

      "config.title": "配置",
      "config.expand": "展开配置",
      "config.collapse": "收起配置",
      "config.serverUrl": "服务器地址",
      "config.serverUrlPlaceholder": "https://www.pawsql.com",
      "config.authGroup": "登录",
      "config.anonymous": "匿名",
      "config.anonymousHint": "使用共享密钥，无需登录。登录后可选择工作空间，进行基于真实 Schema 的优化分析。",
      "config.email": "邮箱",
      "config.password": "密码",
      "config.signin": "登录",
      "config.signout": "退出登录",
      "config.signedInAs": "已登录：{email}",
      "config.testConnection": "测试连接",
      "config.testing": "正在测试连接…",
      "config.testSuccess": "连接成功。",
      "config.testFailed": "连接失败：{message}",
      "config.testRequiredAnon": "请填写服务器地址。",
      "config.testRequired": "请填写邮箱和密码。",
      "config.optGroup": "优化选项",
      "config.validate": "验证性能（What-If）",
      "config.analyze": "explain analyze",
      "config.rewrite": "建议查询改写",
      "config.dedup": "索引建议去重",
      "config.indexOnly": "仅索引建议",
      "config.updateStats": "验证前更新统计信息",
      "config.save": "保存配置",
      "config.saved": "配置已保存。",
      "config.saveFailed": "保存失败：{message}",
      "config.signinFailed": "登录失败：{message}",

      "input.title": "优化 SQL",
      "input.sqlLabel": "SQL 文本",
      "input.sqlPlaceholder": "在此粘贴要优化的 SQL 语句…",
      "input.dbType": "数据库类型",
      "input.optimize": "优化",
      "input.optimizing": "优化中…",
      "input.sqlEmpty": "请先粘贴要优化的 SQL。",
      "input.detectedDbType": "已根据连接识别数据库类型：{dbType}",

      "workspace.label": "工作空间",
      "workspace.placeholder": "请选择工作空间",
      "workspace.refresh": "刷新",
      "workspace.required": "请先选择一个工作空间。",
      "workspace.loading": "正在加载工作空间…",
      "workspace.emptyHint": "该账号暂无工作空间；可在 PawSQL 控制台创建后再刷新。",
      "workspace.loadFailed": "加载工作空间失败：{message}",

      "progress.creating": "正在提交分析…",
      "progress.running": "正在等待分析完成…",
      "progress.details": "正在获取优化报告…",

      "report.empty": "粘贴 SQL 并点击「优化」后，此处将展示 PawSQL 优化报告。",
      "report.title": "优化结果",
      "report.overview.violations": "违规",
      "report.overview.rewrites": "重写优化",
      "report.overview.indexes": "索引推荐",
      "report.overview.improve": "预计提升",
      "report.overview.estimate": "预估值",
      "report.overview.improveValidated": "验证性能提升",
      "report.verified": "已验证",
      "report.violations.title": "审核违规",
      "report.violations.none": "未发现违规",
      "report.violations.colLevel": "严重程度",
      "report.violations.colName": "规则",
      "report.violations.colFragment": "违规 SQL 片段",
      "report.indexes.title": "推荐索引",
      "report.rewrites.title": "查询改写",
      "report.original.title": "优化后 SQL",
      "report.validation.title": "性能验证",
      "report.validation.before": "优化前",
      "report.validation.after": "优化后",
      "report.copy": "复制",
      "report.moreDetails": "在浏览器中查看更多优化详情",
      "level.critical": "严重",
      "level.warning": "警告",
      "level.info": "提示",

      "markdown.failedTitle": "PawSQL 优化失败",
      "toast.copied": "已复制",
      "toast.copyFailed": "复制失败，请手动选择文本",
      "err.backendHint": "该插件的优化请求由原生 Sidecar 执行；请在 DBX 中打开本插件工作台，或先构建后端。",
      "err.invoke": "调用失败：{message}",

      /* PawSQL server error codes (the envelope `message` field) -> human text. */
      "error.unknown.error": "发生了一个未知错误，请稍后重试。",
      "error.api.key.not.configured": "ApiKey 未配置。",
      "error.license.code.not.exist": "Apikey 不存在，请联系 PawSQL 管理员。",
      "error.license.code.not.valid": "许可证代码无效，请联系 PawSQL 管理员获取。",
      "error.beta.analysis.over.limit": "β 用户优化次数超过限制。",
      "error.analysis.over.limit": "优化次数超过限制。",
      "error.plan.user.exceeded.maximum.opt.times": "优化次数已超出当前计划限制。如需继续使用，请升级您的计划。",
      "error.workspace.not.exist": "工作空间不存在。",
      "error.workspace.has.no.analysis": "工作空间优化列表为空。",
      "error.analysis.not.exist": "优化不存在。",
      "error.error.create.analysis.failed": "SQL 优化失败。",
      "error.error.login.invalidCredentials": "PawSQL 权限验证错误，请确认你的用户名密码信息输入无误。",
      "error.error.backendUrl.invalid": "请检查您的 PawSQL 服务器地址是否输入正确，或服务器是否在运行。",
      "error.error.frontendUrl.invalid": "请检查 frontendUrl 是否配置正确，或服务器是否在运行。",
      "error.error.config.validate.failed": "验证失败。",
      "error.error.load.data.failed": "加载数据失败。",
      "error.nonexistent": "邮箱不存在。",
      "error.unactive": "很抱歉，您的邮箱尚未激活，请登录您的注册邮箱完成激活。",
      "error.workload.only.support.single.query": "只支持单条 SQL 语句优化。"
    },
    "en": {
      "brand": "PawSQL SQL Optimizer",

      "config.title": "Configuration",
      "config.expand": "Expand configuration",
      "config.collapse": "Collapse configuration",
      "config.serverUrl": "Server URL",
      "config.serverUrlPlaceholder": "https://www.pawsql.com",
      "config.authGroup": "Authentication",
      "config.anonymous": "Anonymous",
      "config.anonymousHint": "Use the shared key, no sign-in required. Sign in to select a workspace for schema-aware analysis.",
      "config.email": "Email",
      "config.password": "Password",
      "config.signin": "Sign in",
      "config.signout": "Sign out",
      "config.signedInAs": "Signed in as {email}",
      "config.testConnection": "Test Connection",
      "config.testing": "Testing connection…",
      "config.testSuccess": "Connection successful.",
      "config.testFailed": "Connection failed: {message}",
      "config.testRequiredAnon": "Please fill in the server URL.",
      "config.testRequired": "Please fill in email and password.",
      "config.optGroup": "Optimization options",
      "config.validate": "Validate performance (What-If)",
      "config.analyze": "Explain analyze",
      "config.rewrite": "Suggest query rewrites",
      "config.dedup": "Deduplicate recommended indexes",
      "config.indexOnly": "Index-advice only",
      "config.updateStats": "Update statistics before validation",
      "config.save": "Save configuration",
      "config.saved": "Configuration saved.",
      "config.saveFailed": "Save failed: {message}",
      "config.signinFailed": "Sign-in failed: {message}",

      "input.title": "Optimize SQL",
      "input.sqlLabel": "SQL text",
      "input.sqlPlaceholder": "Paste the SQL statement to optimize…",
      "input.dbType": "Database type",
      "input.optimize": "Optimize",
      "input.optimizing": "Optimizing…",
      "input.sqlEmpty": "Paste the SQL to optimize first.",
      "input.detectedDbType": "Database type detected from the connection: {dbType}",

      "workspace.label": "Workspace",
      "workspace.placeholder": "Select a workspace",
      "workspace.refresh": "Refresh",
      "workspace.required": "Please select a workspace first.",
      "workspace.loading": "Loading workspaces…",
      "workspace.emptyHint": "No workspaces for this account; create one in the PawSQL console, then refresh.",
      "workspace.loadFailed": "Failed to load workspaces: {message}",

      "progress.creating": "Submitting the analysis…",
      "progress.running": "Waiting for the analysis to finish…",
      "progress.details": "Fetching the optimization report…",

      "report.empty": "Paste SQL and hit Optimize to see the PawSQL report here.",
      "report.title": "Optimization Result",
      "report.overview.violations": "Violations",
      "report.overview.rewrites": "Rewrite optimizations",
      "report.overview.indexes": "Recommended indexes",
      "report.overview.improve": "Est. improve",
      "report.overview.estimate": "Estimated",
      "report.overview.improveValidated": "Validated improvement",
      "report.verified": "Verified",
      "report.violations.title": "Review violations",
      "report.violations.none": "No violations found",
      "report.violations.colLevel": "Severity",
      "report.violations.colName": "Rule",
      "report.violations.colFragment": "Violated SQL",
      "report.indexes.title": "Recommended indexes",
      "report.rewrites.title": "Query rewrites",
      "report.original.title": "Optimized SQL",
      "report.validation.title": "Performance validation",
      "report.validation.before": "Before",
      "report.validation.after": "After",
      "report.copy": "Copy",
      "report.moreDetails": "More optimization details in browser",
      "level.critical": "Critical",
      "level.warning": "Warning",
      "level.info": "Info",

      "markdown.failedTitle": "PawSQL Optimization Failed",
      "toast.copied": "Copied",
      "toast.copyFailed": "Copy failed — select the text manually",
      "err.backendHint": "Optimization runs through the native sidecar; open this workbench inside DBX or build the backend first.",
      "err.invoke": "Call failed: {message}",

      /* PawSQL server error codes (the envelope `message` field) -> human text. */
      "error.unknown.error": "An unknown error occurred, please try again later.",
      "error.api.key.not.configured": "API key is not configured.",
      "error.license.code.not.exist": "API key does not exist. Please contact the PawSQL administrator.",
      "error.license.code.not.valid": "Invalid license code. Please contact the PawSQL administrator.",
      "error.beta.analysis.over.limit": "Beta optimization limit exceeded.",
      "error.analysis.over.limit": "Optimization limit exceeded.",
      "error.plan.user.exceeded.maximum.opt.times": "Optimization limit exceeded for your plan. Please upgrade to continue.",
      "error.workspace.not.exist": "Workspace does not exist.",
      "error.workspace.has.no.analysis": "No optimizations in this workspace.",
      "error.analysis.not.exist": "Optimization does not exist.",
      "error.error.create.analysis.failed": "SQL optimization failed.",
      "error.error.login.invalidCredentials": "PawSQL authentication failed. Please check your username and password.",
      "error.error.backendUrl.invalid": "Please check your PawSQL server URL or whether the server is running.",
      "error.error.frontendUrl.invalid": "Please check the frontend URL configuration or whether the server is running.",
      "error.error.config.validate.failed": "Validation failed.",
      "error.error.load.data.failed": "Failed to load data.",
      "error.nonexistent": "Email does not exist.",
      "error.unactive": "Sorry, your email is not activated. Please log in to your mailbox to activate it.",
      "error.workload.only.support.single.query": "Only a single SQL statement can be optimized."
    }
  };

  let locale = "en";

  function normalize(value) {
    if (!value) return "en";
    const lower = String(value).toLowerCase();
    if (lower.startsWith("zh")) return "zh-CN";
    return "en";
  }

  function setLocale(value) {
    locale = normalize(value);
    return locale;
  }

  function interpolate(text, params) {
    if (!params) return text;
    return text.replace(/\{(\w+)\}/g, (match, key) =>
      Object.prototype.hasOwnProperty.call(params, key) ? String(params[key]) : match);
  }

  function t(key, params) {
    const table = dictionaries[locale] || dictionaries["en"];
    const text = table[key] !== undefined ? table[key] : dictionaries["en"][key];
    return text === undefined ? key : interpolate(text, params);
  }

  function isChinese() {
    return locale === "zh-CN";
  }

  /* Localise a PawSQL server error. The sidecar surfaces the envelope `message`
     field verbatim, which is an error *code* (e.g. "analysis.over.limit"), not
     human text; map it through `error.<code>`. Non-code messages pass through. */
  function serverError(message) {
    if (!message) return "";
    const code = String(message).trim();
    if (!code) return "";
    const key = "error." + code;
    const text = t(key);
    return text === key ? message : text;
  }

  window.HC = window.HC || {};
  window.HC.i18n = { t, setLocale, isChinese, serverError, get locale() { return locale; }, dictionaries };
})();
