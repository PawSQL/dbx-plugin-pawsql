/* DOM helpers, formatting, toasts and SQL highlighting for the PawSQL workbench. */

(function () {
  const t = (key, params) => HC.i18n.t(key, params);

  function $(selector, root) { return (root || document).querySelector(selector); }
  function $$(selector, root) { return Array.prototype.slice.call((root || document).querySelectorAll(selector)); }

  function el(tag, props, children) {
    const node = document.createElement(tag);
    if (props) {
      Object.keys(props).forEach((key) => {
        const value = props[key];
        if (value === undefined || value === null) return;
        if (key === "class") node.className = value;
        else if (key === "text") node.textContent = value;
        else if (key === "html") node.innerHTML = value;
        else if (key === "dataset") Object.assign(node.dataset, value);
        else if (key === "style") Object.assign(node.style, value);
        else if (key === "on") Object.keys(value).forEach((event) => node.addEventListener(event, value[event]));
        else if (key === "checked" || key === "disabled" || key === "selected" || key === "hidden") node[key] = !!value;
        else if (key === "value") node.value = value;
        else node.setAttribute(key, value);
      });
    }
    if (children) {
      const flat = [];
      (function walk(items) {
        (Array.isArray(items) ? items : [items]).forEach((child) => {
          if (child === null || child === undefined || child === false) return;
          if (Array.isArray(child)) { walk(child); return; }
          flat.push(child);
        });
      })(children);
      flat.forEach((child) => {
        node.append(child instanceof Node ? child : document.createTextNode(String(child)));
      });
    }
    return node;
  }

  function clear(node) { while (node.firstChild) node.removeChild(node.firstChild); return node; }

  function escapeHtml(text) {
    return String(text == null ? "" : text)
      .replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  function escapeAttr(value) { return escapeHtml(value); }

  /* ------------------------------------------------------------ clipboard -- */

  async function copyText(text) {
    try {
      if (navigator.clipboard && navigator.clipboard.writeText) {
        await navigator.clipboard.writeText(text);
        return true;
      }
    } catch (error) { /* fall through to the legacy path */ }
    try {
      const area = el("textarea", { value: text, style: { position: "fixed", opacity: "0" } });
      document.body.append(area);
      area.select();
      const ok = document.execCommand("copy");
      area.remove();
      return ok;
    } catch (error) {
      return false;
    }
  }

  async function copy(text) {
    if (window.HC && HC.bridge && HC.bridge.copy) {
      const ok = await HC.bridge.copy(text);
      if (ok) return true;
    }
    return copyText(text);
  }

  /* --------------------------------------------------------------- toasts -- */

  function toast(message, kind) {
    const host = $("#toast-host");
    if (!host) return;
    const node = el("div", { class: `hc-toast${kind ? ` is-${kind}` : ""}`, text: message });
    host.append(node);
    setTimeout(() => {
      node.style.transition = "opacity .2s";
      node.style.opacity = "0";
      setTimeout(() => node.remove(), 220);
    }, kind === "error" ? 6000 : 2800);
  }

  function formatPercent(value) {
    if (value === null || value === undefined || Number.isNaN(value)) return "-";
    return `${Number(value).toFixed(2)}%`;
  }

  /* --------------------------------------------------------- highlighting -- */

  const SQL_KEYWORDS = /\b(SELECT|FROM|WHERE|AND|OR|NOT|IN|EXISTS|BETWEEN|LIKE|IS|NULL|AS|ON|JOIN|INNER|LEFT|RIGHT|FULL|OUTER|CROSS|UNION|ALL|GROUP|BY|ORDER|HAVING|LIMIT|OFFSET|INSERT|INTO|VALUES|UPDATE|SET|DELETE|CREATE|ALTER|DROP|TABLE|INDEX|VIEW|KEY|PRIMARY|FOREIGN|UNIQUE|DEFAULT|CONSTRAINT|IF|CASE|WHEN|THEN|ELSE|END|ASC|DESC|DISTINCT|COUNT|SUM|AVG|MIN|MAX|EXPLAIN|ANALYZE)\b/gi;

  /* Highlights SQL in every `pre code` under `root`. Mirrors the DBeaver
     renderer's pawHighlight() so keyword/string/comment colouring matches. */
  function highlightSql(root) {
    const nodes = $$("pre code", root || document);
    nodes.forEach((code) => {
      const text = code.textContent || "";
      const highlighted = escapeHtml(text)
        .replace(/(--[^\n]*)/g, '<span class="sql-comment">$1</span>')
        .replace(/('(?:[^']|'')*')/g, '<span class="sql-string">$1</span>')
        .replace(SQL_KEYWORDS, '<span class="sql-keyword">$&</span>');
      if (highlighted !== escapeHtml(text)) code.innerHTML = highlighted;
    });
  }

  window.HC = window.HC || {};
  window.HC.util = {
    $, $$, el, clear, escapeHtml, escapeAttr, copyText, copy, toast, formatPercent, highlightSql
  };
})();
