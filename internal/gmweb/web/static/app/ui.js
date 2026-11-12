// ui.js —— §4.3 五个全局交互模式的工厂（M1 CatalogPicker / M2 CmdBtn /
// M3 StatForm / M4 ToggleRow / M5 FeedbackFlow）。所有面板从这里拼装，
// 不再各写各的 fetch/confirm/提示。

import { command, search, debounce } from "./api.js";
import { refreshNow } from "./state.js";
import { confirm as axConfirm } from "/static/ax.esm.js";

/* ---------------- DOM 小工具 ---------------- */

/** 创建元素；attrs 里的 on* 不支持（用 addEventListener），children 接受节点/字符串/数组/null。 */
export function h(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs || {})) {
    if (value === null || value === undefined || value === false) continue;
    if (key === "class") node.className = value;
    else if (key === "text") node.textContent = value;
    else if (key === "html") node.innerHTML = value; // 仅限静态字面量，禁止动态数据
    else node.setAttribute(key, value === true ? "" : value);
  }
  for (const child of [].concat(children)) {
    if (child === null || child === undefined || child === false) continue;
    node.appendChild(typeof child === "string" ? document.createTextNode(child) : child);
  }
  return node;
}

/* ---------------- M5 FeedbackFlow ---------------- */

export function toast(message, theme = "info", ms = 3200) {
  const stack = document.querySelector("#toast-stack");
  if (!stack) return;
  const node = h("div", { class: "toast " + (theme === "info" ? "" : theme) }, [message]);
  stack.appendChild(node);
  requestAnimationFrame(() => node.classList.add("show"));
  setTimeout(() => {
    node.classList.remove("show");
    setTimeout(() => node.remove(), 250);
  }, ms);
}

/** 操作历史（localStorage，上限 100 条）。 */
const HISTORY_KEY = "gm-ax-history";
const HISTORY_LIMIT = 100;

export function pushHistory(entry) {
  try {
    const list = JSON.parse(localStorage.getItem(HISTORY_KEY) || "[]");
    list.unshift(Object.assign({ at: new Date().toISOString() }, entry));
    localStorage.setItem(HISTORY_KEY, JSON.stringify(list.slice(0, HISTORY_LIMIT)));
  } catch {
    /* 隐私模式等场景放弃持久化 */
  }
}

export function readHistory() {
  try {
    return JSON.parse(localStorage.getItem(HISTORY_KEY) || "[]");
  } catch {
    return [];
  }
}

/** 危险操作二次确认。优先 AXUI confirm；异常时回退 window.confirm。 */
export async function confirmDanger(text) {
  try {
    const result = await axConfirm({ content: text, theme: "error" });
    return result === true || result === "true" || result?.button === "ok" || result === undefined ? true : false;
  } catch {
    return window.confirm(text);
  }
}

/* ---------------- M2 CmdBtn ---------------- */

/**
 * 命令按钮：点击 → 互斥发送 → Toast + 历史 → 单次会话刷新。
 * opts: {label, theme, danger, confirm, action, params, onDone, section}
 */
export function cmdBtn(opts) {
  const node = h("ax-btn", { label: opts.label, theme: opts.danger ? "error" : opts.theme || "" });
  node.setAttribute("type", "plain");
  node.addEventListener("click", async () => {
    if (opts.danger && !(await confirmDanger(opts.confirm || ("确定执行「" + opts.label + "」？该操作不可撤回。")))) {
      return;
    }
    node.setAttribute("disabled", "");
    const status = document.querySelector("#command-status");
    if (status) status.textContent = "发送中……";
    const payload = Object.assign({ action: opts.action }, opts.params || {});
    const result = await command(payload);
    node.removeAttribute("disabled");
    if (status) status.textContent = "";
    toast((result.ok ? "✔ " : "✘ ") + opts.label + "：" + result.message, result.ok ? "ok" : "err");
    pushHistory({ section: opts.section || "", label: opts.label, payload, ok: result.ok, message: result.message });
    if (opts.onDone) opts.onDone(result);
    refreshNow();
  });
  return node;
}

/**
 * 参数命令按钮（M2 变体）：build() 在点击时构造 payload 再发送。
 * opts: {label, theme, danger, confirm, build, okText, section, onDone}
 * 返回按钮节点；发送结果走统一 Toast + 历史 + 会话刷新。
 */
export function actionBtn(opts) {
  const node = h("ax-btn", { label: opts.label, theme: opts.danger ? "error" : opts.theme || "prim" });
  node.setAttribute("type", opts.theme || opts.danger ? "solid" : "plain");
  node.addEventListener("click", async () => {
    if (opts.danger && !(await confirmDanger(opts.confirm || ("确定执行「" + opts.label + "」？该操作不可撤回。")))) {
      return;
    }
    node.setAttribute("disabled", "");
    const payload = opts.build();
    if (!payload) {
      node.removeAttribute("disabled");
      return;
    }
    const result = await command(payload);
    node.removeAttribute("disabled");
    const okText = typeof opts.okText === "function" ? opts.okText(payload) : opts.okText || opts.label;
    toast((result.ok ? "✔ " : "✘ ") + okText + "：" + result.message, result.ok ? "ok" : "err");
    pushHistory({ section: opts.section || "", label: opts.label, payload, ok: result.ok, message: result.message });
    if (opts.onDone) opts.onDone(result);
    refreshNow();
  });
  return node;
}

/* ---------------- M3 StatForm 基件 ---------------- */

/** 数值输入（ax-number）；返回 {node, get}。 */
export function numField({ label = "", value = "", min, max, step, width } = {}) {
  const input = h("ax-number", { placeholder: String(value) });
  if (min !== undefined) input.setAttribute("min", String(min));
  if (max !== undefined) input.setAttribute("max", String(max));
  if (step !== undefined) input.setAttribute("step", String(step));
  if (width) input.style.width = width;
  const wrap = label ? h("label", { class: "field-row" }, [label + " ", input]) : input;
  return {
    node: wrap,
    get() {
      const raw = input.value;
      if (raw === "" || raw === null || raw === undefined) return null;
      const parsed = Number(raw);
      return Number.isFinite(parsed) ? parsed : null;
    },
    set(v) {
      input.setAttribute("value", String(v));
    },
  };
}

/** 文本输入（ax-input）；返回 {node, get, clear}。 */
export function textField({ label = "", placeholder = "", width } = {}) {
  const input = h("ax-input", { placeholder });
  if (width) input.style.width = width;
  const wrap = label ? h("label", { class: "field-row" }, [label + " ", input]) : input;
  return {
    node: wrap,
    get: () => (input.value || "").trim(),
    clear() {
      input.setAttribute("value", "");
      input.value = "";
    },
  };
}

/** 板块容器：标题 + 说明 + 内容。 */
export function panelSection({ id, title, desc }) {
  const body = h("div", { class: "panel-body" });
  const section = h("section", { class: "panel", id });
  section.appendChild(h("h2", {}, [title]));
  if (desc) section.appendChild(h("p", { class: "desc" }, [desc]));
  section.appendChild(body);
  return { section, body };
}

/** 提示行（成功/失败着色）。返回 {node, set(text, ok)}。 */
export function hint() {
  const node = h("div", { class: "hint" });
  return {
    node,
    set(text, ok) {
      node.textContent = text || "";
      node.classList.toggle("ok", ok === true);
      node.classList.toggle("err", ok === false);
    },
  };
}

/** 小目录下拉（<200 行）：ax-select content + search，一次拉全量前端过滤。 */
export function smallSelect({ placeholder = "输入过滤…", items, value }) {
  const content = items.map((item) => ({
    label: item.label,
    value: String(item.value),
  }));
  const node = h("ax-select", {
    search: true,
    full: true,
    placeholder,
    content: JSON.stringify(content),
    "max-height": "260",
  });
  if (value !== undefined) node.setAttribute("value", String(value));
  return node;
}

/* ---------------- M1 CatalogPicker ---------------- */

/**
 * 目录选择器：服务端搜索 + 分页 + 最近选用。
 * cfg: {
 *   placeholder, pageSize=50,
 *   fetchRows: async (query, signal, page) => 行数组（本地分页）或
 *              {rows, total}（服务端分页：page={offset,limit}，翻页按
 *              total 计页数并逐页向 fetchRows 取数）,
 *   onPick: (row) => void,
 *   emptyText
 * }
 * 返回 {node, refresh(), setSelected(row)}。
 */
export function catalogPicker(cfg) {
  const pageSize = cfg.pageSize || 50;
  let rows = [];
  let page = 0;
  let serverTotal = null; // null=本地分页（fetchRows 返回数组）；数字=服务端分页总量
  let selected = null;
  let lastQuery = "";
  let lastToken = ""; // query@offset 过期响应丢弃（分页翻页与改词并发）
  let aborter = null;

  const input = h("input", { class: "picker-input", type: "text", placeholder: cfg.placeholder || "输入中文名或 ID 搜索…" });
  input.style.cssText = "flex:1;min-width:200px;background:#11141b;border:1px solid var(--gm-line);border-radius:6px;color:var(--gm-text);padding:6px 10px;";
  const listEl = h("div", { class: "picker-list" });
  const emptyEl = h("div", { class: "picker-empty" }, ["加载中……"]);
  const infoEl = h("span", {}, [""]);
  const prevBtn = h("button", { class: "icon-btn" }, ["‹ 上一页"]);
  const nextBtn = h("button", { class: "icon-btn" }, ["下一页 ›"]);
  const pager = h("div", { class: "picker-pager" }, [prevBtn, infoEl, nextBtn]);
  const recentWrap = h("div", { class: "picker-pager", style: "border-top:1px solid #1e222c;" });
  const box = h("div", { class: "picker" }, [
    h("div", { class: "picker-toolbar" }, [input]),
    listEl,
    emptyEl,
    pager,
    recentWrap,
  ]);

  function renderRecent() {
    recentWrap.innerHTML = "";
    const key = "gm-ax-recent:" + (cfg.recentKey || location.hash);
    let list = [];
    try {
      list = JSON.parse(localStorage.getItem(key) || "[]").slice(0, 8);
    } catch {
      /* 忽略损坏数据 */
    }
    if (!list.length) {
      recentWrap.style.display = "none";
      return;
    }
    recentWrap.style.display = "";
    recentWrap.appendChild(h("span", {}, ["最近选用："]));
    for (const item of list) {
      const chip = h("button", { class: "icon-btn", title: item.sub || "" }, [item.label]);
      chip.addEventListener("click", () => pick(item, false));
      recentWrap.appendChild(chip);
    }
  }

  function remember(row) {
    const key = "gm-ax-recent:" + (cfg.recentKey || location.hash);
    try {
      const list = JSON.parse(localStorage.getItem(key) || "[]").filter((x) => x.value !== row.value);
      list.unshift({ label: row.label, sub: row.sub || "", value: row.value });
      localStorage.setItem(key, JSON.stringify(list.slice(0, 8)));
    } catch {
      /* 放弃 */
    }
    renderRecent();
  }

  function pick(row, save = true) {
    selected = row;
    renderList();
    if (save) remember(row);
    if (cfg.onPick) cfg.onPick(row);
  }

  function renderList() {
    listEl.innerHTML = "";
    const count = serverTotal !== null ? serverTotal : rows.length;
    if (!rows.length) {
      emptyEl.style.display = "";
      pager.style.display = "none";
      return;
    }
    emptyEl.style.display = "none";
    pager.style.display = "";
    const start = page * pageSize;
    // 服务端分页：rows 恰为本页数据，不切片；本地分页：在整包里切当前页。
    const slice = serverTotal !== null ? rows : rows.slice(start, start + pageSize);
    for (const row of slice) {
      const item = h("div", { class: "picker-row" + (selected && selected.value === row.value ? " selected" : "") }, [
        h("span", {}, [row.label]),
        row.sub ? h("span", { class: "sub" }, [row.sub]) : null,
      ]);
      item.addEventListener("click", () => pick(row));
      listEl.appendChild(item);
    }
    const totalPages = Math.max(1, Math.ceil(count / pageSize));
    infoEl.textContent = (start + 1) + "-" + Math.min(start + pageSize, count) + " / 共 " + count + " 条，第 " + (page + 1) + "/" + totalPages + " 页";
    prevBtn.disabled = page <= 0;
    nextBtn.disabled = page >= totalPages - 1;
  }

  async function runQuery(query, offset = 0) {
    lastQuery = query;
    lastToken = query + "@" + offset;
    if (aborter) aborter.abort();
    aborter = new AbortController();
    emptyEl.style.display = "";
    emptyEl.textContent = "搜索中……";
    listEl.innerHTML = "";
    pager.style.display = "none";
    try {
      const result = await cfg.fetchRows(query, aborter.signal, { offset, limit: pageSize });
      if (lastToken !== query + "@" + offset) return; // 过期响应丢弃
      if (Array.isArray(result)) {
        rows = result;
        serverTotal = null;
      } else {
        rows = (result && result.rows) || [];
        serverTotal = result && result.total ? result.total : 0;
      }
      page = Math.floor(offset / pageSize);
      emptyEl.textContent = rows.length ? "" : (cfg.emptyText || "没有匹配项");
      renderList();
    } catch (err) {
      if (err && err.name === "AbortError") return;
      emptyEl.textContent = err.message || "搜索失败";
      toast(String(err.message || err), "err");
    }
  }

  input.addEventListener("input", debounce(() => runQuery(input.value.trim()), 300));
  prevBtn.addEventListener("click", () => {
    if (page <= 0) return;
    page--;
    if (serverTotal !== null) runQuery(lastQuery, page * pageSize);
    else renderList();
  });
  nextBtn.addEventListener("click", () => {
    const count = serverTotal !== null ? serverTotal : rows.length;
    if ((page + 1) * pageSize >= count) return;
    page++;
    if (serverTotal !== null) runQuery(lastQuery, page * pageSize);
    else renderList();
  });

  renderRecent();
  // 进入板块即拉第一页（空查询=全量目录首屏），不要求先输入关键词搜索。
  runQuery("");

  return {
    node: box,
    refresh() {
      runQuery(input.value.trim());
    },
    setSelected(row) {
      pick(row, false);
    },
    getSelected() {
      return selected;
    },
  };
}

/* ---------------- M5 操作历史抽屉 ---------------- */

/** 操作历史抽屉：顶栏 🕰 呼出，最近 100 条；复制 JSON / 清空。 */
export function mountHistoryDrawer() {
  const btn = document.querySelector("#history-btn");
  if (!btn) return;
  let overlay = null;

  function close() {
    if (overlay) {
      overlay.remove();
      overlay = null;
    }
  }

  function renderRow(entry) {
    const time = entry.at ? entry.at.replace("T", " ").slice(5, 19) : "";
    const status = entry.ok ? "✔" : "✘";
    return h(
      "div",
      { class: "picker-row", style: "flex-direction:column;align-items:flex-start;gap:2px;" },
      [
        h("div", { style: "display:flex;gap:8px;width:100%;" }, [
          h("span", { style: entry.ok ? "color:#6fd692;" : "color:#ff8080;" }, [status]),
          h("span", {}, [entry.label || (entry.payload && entry.payload.action) || "-"]),
          h("span", { class: "sub", style: "margin-left:auto;" }, [time]),
        ]),
        h("span", { class: "sub", style: "white-space:normal;word-break:break-all;" }, [
          (entry.section ? entry.section + " · " : "") + (entry.message || ""),
        ]),
      ]
    );
  }

  function open() {
    if (overlay) return close();
    const list = readHistory();
    const listBox = h("div", { style: "max-height:60vh;overflow-y:auto;" }, []);
    if (!list.length) {
      listBox.appendChild(h("div", { class: "picker-empty" }, ["暂无操作记录"]));
    } else {
      for (const entry of list) listBox.appendChild(renderRow(entry));
    }
    const copyBtn = h("ax-btn", { label: "复制 JSON", type: "plain" });
    copyBtn.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(JSON.stringify(list, null, 2));
        toast("已复制 " + list.length + " 条历史到剪贴板", "ok");
      } catch {
        toast("剪贴板不可用（需授权）", "warn");
      }
    });
    const clearBtn = h("ax-btn", { label: "清空", type: "plain" });
    clearBtn.addEventListener("click", () => {
      localStorage.removeItem(HISTORY_KEY);
      toast("操作历史已清空", "ok");
      close();
      open();
    });
    const panel = h("div", { class: "palette-panel", style: "width:400px;" }, [
      h("div", { style: "display:flex;align-items:center;gap:8px;margin-bottom:10px;" }, [
        h("b", { style: "color:var(--gm-gold);" }, ["操作历史（" + list.length + "）"]),
        h("span", { style: "margin-left:auto;display:flex;gap:6px;" }, [copyBtn, clearBtn]),
      ]),
      listBox,
    ]);
    overlay = h("div", { class: "palette-overlay" }, [panel]);
    overlay.addEventListener("click", (event) => {
      if (event.target === overlay) close();
    });
    document.body.appendChild(overlay);
  }

  btn.addEventListener("click", open);
}
