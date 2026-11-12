// palette.js —— 全局 Ctrl+K 命令面板：跨分组搜板块直达 + 全量传送点直达。
// 键盘：↑↓ 选择、Enter 执行、Esc 关闭；传送项直接发送 switch_scene_preset。

import { h, toast, pushHistory } from "./ui.js";
import { command } from "./api.js";
import { refreshNow } from "./state.js";
import { GROUPS, SECTION_TITLES } from "./panels/registry.js";
import { PRESETS, TIGUAN } from "./panels/scene.js";

const TELEPORTS = [];
for (const group of PRESETS) {
  for (const item of group.items) {
    TELEPORTS.push({ kind: "tp", label: item.name, sub: "传送 · " + group.label, key: item.key });
  }
}
for (const group of TIGUAN) {
  for (const item of group.items) {
    TELEPORTS.push({ kind: "tp", label: item.name, sub: "踢馆 · " + group.label, key: item.key });
  }
}

const SECTIONS = [];
for (const group of GROUPS) {
  for (const sectionId of group.sections) {
    SECTIONS.push({ kind: "nav", label: SECTION_TITLES[sectionId] || sectionId, sub: group.name, id: sectionId });
  }
}

let overlay = null;
let activeIndex = 0;
let visibleItems = [];

function buildResults(query) {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return SECTIONS.slice(0, 10);
  }
  const scored = [];
  for (const item of SECTIONS) {
    const text = item.label.toLowerCase();
    if (text.includes(needle)) scored.push({ item, weight: text.startsWith(needle) ? 0 : 1 });
  }
  for (const item of TELEPORTS) {
    const text = item.label.toLowerCase();
    if (text.includes(needle)) scored.push({ item, weight: (text.startsWith(needle) ? 10 : 11) + (item.sub.includes("踢馆") ? 1 : 0) });
  }
  scored.sort((a, b) => a.weight - b.weight);
  return scored.slice(0, 12).map((entry) => entry.item);
}

function runItem(item) {
  close();
  if (item.kind === "nav") {
    location.hash = "#/" + item.id;
    return;
  }
  // 传送：直接发送（GM 移动自身角色，低风险）；结果走统一 Toast + 历史
  command({ action: "switch_scene_preset", preset: item.key }).then((result) => {
    toast((result.ok ? "✔ 传送：" : "✘ 传送：") + item.label + "：" + result.message, result.ok ? "ok" : "err");
    pushHistory({ section: "全局搜索", label: "传送·" + item.label, payload: { action: "switch_scene_preset", preset: item.key }, ok: result.ok, message: result.message });
    if (result.ok) refreshNow();
  });
}

function renderList(box) {
  box.innerHTML = "";
  visibleItems.forEach((item, index) => {
    const row = h(
      "div",
      { class: "picker-row" + (index === activeIndex ? " selected" : ""), "data-index": String(index) },
      [h("span", {}, [item.label]), h("span", { class: "sub" }, [item.sub])]
    );
    row.addEventListener("click", () => runItem(item));
    box.appendChild(row);
  });
}

function close() {
  if (overlay) {
    overlay.remove();
    overlay = null;
  }
}

export function openPalette() {
  if (overlay) {
    close();
    return;
  }
  const input = h("input", { type: "text", placeholder: "搜索板块或传送点（如：少林、坐骑、燕京）…" });
  input.style.cssText =
    "width:100%;background:#11141b;border:1px solid var(--gm-line);border-radius:8px;color:var(--gm-text);padding:10px 14px;font-size:14px;box-sizing:border-box;";
  const list = h("div", { class: "picker-list", style: "max-height:340px;" }, []);
  const footer = h("div", { class: "picker-pager" }, ["↑↓ 选择 · Enter 执行 · Esc 关闭"]);
  const panel = h("div", { class: "palette-panel" }, [input, list, footer]);
  overlay = h("div", { class: "palette-overlay" }, [panel]);
  overlay.addEventListener("click", (event) => {
    if (event.target === overlay) close();
  });
  document.body.appendChild(overlay);

  activeIndex = 0;
  visibleItems = buildResults("");
  renderList(list);
  setTimeout(() => input.focus(), 0);

  input.addEventListener("input", () => {
    activeIndex = 0;
    visibleItems = buildResults(input.value);
    renderList(list);
  });
  input.addEventListener("keydown", (event) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      activeIndex = Math.min(activeIndex + 1, visibleItems.length - 1);
      renderList(list);
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      activeIndex = Math.max(activeIndex - 1, 0);
      renderList(list);
    } else if (event.key === "Enter") {
      event.preventDefault();
      if (visibleItems[activeIndex]) runItem(visibleItems[activeIndex]);
    } else if (event.key === "Escape") {
      close();
    }
  });
}

export function mountPaletteHotkey() {
  document.addEventListener("keydown", (event) => {
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
      event.preventDefault();
      openPalette();
    } else if (event.key === "Escape" && overlay) {
      close();
    }
  });
}
