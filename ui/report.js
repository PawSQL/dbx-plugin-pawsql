/* Renders a structured PawSQL optimization report from the normalized
   OptimizeResult returned by the sidecar. Mirrors the DBeaver plugin's
   PawResultRenderer (overview cards, violation table, index DDLs, rewrites,
   rewritten SQL and What-If validation). */

(function () {
  const util = HC.util;
  const t = (key, params) => HC.i18n.t(key, params);

  /* Severity: 0=Critical, 1=Warning, 2=Info (smaller = more severe). */
  function levelText(level) {
    switch (level) {
      case 0: return t("level.critical");
      case 1: return t("level.warning");
      case 2: return t("level.info");
      default: return `Level ${level}`;
    }
  }

  function levelClass(level) {
    const lv = Math.max(0, Math.min(2, level));
    return lv === 0 ? "lv0" : (lv === 1 ? "lv1" : "lv2");
  }

  /* Rule display name follows the UI locale, falling back to the code. */
  function displayName(rule) {
    if (!rule) return "";
    if (HC.i18n.isChinese()) return rule.nameZh || rule.nameEn || rule.ruleCode || "";
    return rule.nameEn || rule.nameZh || rule.ruleCode || "";
  }

  function codeBlock(sql) {
    const btn = util.el("button", {
      class: "copybtn", type: "button", text: t("report.copy"),
      on: { click: async () => {
        const ok = await util.copy(sql);
        if (ok) {
          btn.textContent = "✓";
          setTimeout(() => { btn.textContent = t("report.copy"); }, 1200);
        } else {
          util.toast(t("toast.copyFailed"), "error");
        }
      } }
    });
    const pre = util.el("pre", {}, [util.el("code", { text: sql })]);
    return util.el("div", { class: "block" }, [btn, pre]);
  }

  function card(cssClass, num, label, color) {
    return util.el("div", { class: `card ${cssClass}` }, [
      util.el("div", { class: "num", style: { color: color || "inherit" }, text: num }),
      util.el("div", { class: "lbl", text: label })
    ]);
  }

  function renderOverview(result, estimated) {
    const detail = result.detail || {};
    const violations = (detail.violations || []).length;
    const rewrites = (detail.rewrites || []).length;
    const indexes = (detail.indexRecommended || []).length;
    const improve = util.formatPercent(result.improvePercent);

    const cards = util.el("div", { class: "cards" }, [
      card("violations", String(violations), t("report.overview.violations"), violations > 0 ? "#c0392b" : null),
      card("rewrites", String(rewrites), t("report.overview.rewrites"), null),
      card("indexes", String(indexes), t("report.overview.indexes"), null)
    ]);

    if (!estimated && result.improvePercent != null) {
      cards.append(util.el("div", { class: "card improve improve-validated" }, [
        util.el("span", { class: "verified-badge", text: t("report.verified") }),
        util.el("div", { class: "num", style: { color: "#1e8449" }, text: improve }),
        util.el("div", { class: "lbl", text: t("report.overview.improveValidated") })
      ]));
    } else {
      cards.append(util.el("div", { class: "card improve improve-estimated" }, [
        util.el("span", { class: "estimate-badge", text: t("report.overview.estimate") }),
        util.el("div", { class: "num", text: improve }),
        util.el("div", { class: "lbl", text: t("report.overview.improve") })
      ]));
    }
    return cards;
  }

  function renderViolations(detail) {
    const section = util.el("section", { class: "report-section" }, [
      util.el("h2", { text: t("report.violations.title") })
    ]);
    const violations = (detail.violations || []).slice();
    if (!violations.length) {
      section.append(util.el("span", { class: "badge ok", text: t("report.violations.none") }));
      return section;
    }
    violations.sort((a, b) => (a.level || 0) - (b.level || 0));

    const rows = violations.map((v) => {
      const lv = v.level || 0;
      const badge = util.el("span", { class: `badge ${levelClass(lv)}`, text: levelText(lv) });
      const fragments = (v.fragments || []);
      let fragCell;
      if (!fragments.length) {
        fragCell = util.el("td", { text: "—" });
      } else {
        const code = util.el("code", { text: fragments.join("\n") });
        fragCell = util.el("td", {}, [util.el("pre", { style: { margin: "0" } }, [code])]);
      }
      return util.el("tr", {}, [
        util.el("td", {}, [badge]),
        util.el("td", { text: displayName(v) }),
        fragCell
      ]);
    });

    const table = util.el("table", { class: "vt" }, [
      util.el("thead", {}, [util.el("tr", {}, [
        util.el("th", { style: { width: "16%" }, text: t("report.violations.colLevel") }),
        util.el("th", { style: { width: "30%" }, text: t("report.violations.colName") }),
        util.el("th", { text: t("report.violations.colFragment") })
      ])]),
      util.el("tbody", {}, rows)
    ]);
    section.append(table);
    return section;
  }

  function renderIndexes(detail) {
    const indexes = detail.indexRecommended || [];
    if (!indexes.length) return null;
    return util.el("section", { class: "report-section" }, [
      util.el("h2", { text: t("report.indexes.title") }),
      codeBlock(indexes.join("\n\n"))
    ]);
  }

  function renderRewrites(detail) {
    const rewrites = detail.rewrites || [];
    if (!rewrites.length) return null;
    const blocks = rewrites.map((r) => {
      const block = util.el("div", { class: "block" }, [
        util.el("strong", { text: displayName(r) })
      ]);
      (r.violated || []).forEach((sql) => {
        block.append(util.el("div", { class: "sqlfrag" }, [
          util.el("pre", {}, [util.el("code", { text: sql })])
        ]));
      });
      return block;
    });
    return util.el("section", { class: "report-section" }, [
      util.el("h2", { text: t("report.rewrites.title") }),
      blocks
    ]);
  }

  function renderRewrittenSql(detail) {
    for (const r of (detail.rewrites || [])) {
      if (r.rewritten && r.rewritten.length && r.rewritten[0]) {
        return util.el("section", { class: "report-section" }, [
          util.el("h2", { text: t("report.original.title") }),
          codeBlock(r.rewritten[0])
        ]);
      }
    }
    return null;
  }

  function renderValidation(detail) {
    const v = detail.validation;
    if (!v || (!v.beforePlan && !v.afterPlan)) return null;
    const boxes = [];
    if (v.beforePlan) {
      boxes.push(util.el("div", { class: "sqlbox" }, [
        util.el("strong", { text: t("report.validation.before") }),
        codeBlock(v.beforePlan)
      ]));
    }
    if (v.afterPlan) {
      boxes.push(util.el("div", { class: "sqlbox" }, [
        util.el("strong", { text: t("report.validation.after") }),
        codeBlock(v.afterPlan)
      ]));
    }
    return util.el("section", { class: "report-section" }, [
      util.el("h2", { text: t("report.validation.title") }),
      util.el("div", { class: "compare" }, boxes)
    ]);
  }

  function render(result, opts) {
    const estimated = !!(opts && opts.anonymous) || !result.validated;
    const root = util.el("div", { class: "report" });
    root.append(util.el("h2", { class: "report-title", text: t("report.title") }));
    root.append(renderOverview(result, estimated));
    const detail = result.detail || {};
    const parts = [
      renderViolations(detail),
      renderIndexes(detail),
      renderRewrites(detail),
      renderRewrittenSql(detail),
      renderValidation(detail)
    ].filter(Boolean);
    parts.forEach((part) => root.append(part));

    if (result.detailUrl) {
      root.append(util.el("p", { class: "report-more" }, [
        util.el("a", {
          href: result.detailUrl,
          rel: "noopener",
          text: t("report.moreDetails"),
          on: { click: (event) => {
            event.preventDefault();
            HC.bridge.openExternal(result.detailUrl).catch(() => {
              window.open(result.detailUrl, "_blank", "noopener");
            });
          } }
        })
      ]));
    }
    return root;
  }

  function renderError(message) {
    return util.el("div", { class: "report report-error" }, [
      util.el("h2", { text: t("markdown.failedTitle") }),
      util.el("p", { style: { color: "#c0392b" }, text: message })
    ]);
  }

  window.HC.report = { render, renderError, levelText, displayName };
})();
