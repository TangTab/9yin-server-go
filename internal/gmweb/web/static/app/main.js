// main.js —— AXUI 版 GM 控制台启动入口：组件注册、导航构建、hash 路由、
// 顶栏角色卡与底栏连接状态。面板模块按分组懒加载（首次切入才 import）。
//
// AXUI v3.1.33 的 ESM 构建【不自动注册 Web Component】（ax-api.ts 头部
// 注释的说法与运行时不符，regComp 已导出但从不被调用）——必须在启动时
// 遍历命名空间把 *Elem 类逐个 define，否则 <ax-btn>/<ax-select> 等全部
// 不渲染（2026-11-12 浏览器实测：未注册时 193 个按钮 0 尺寸）。

import * as axui from "/static/ax.esm.js";
import { regComp } from "/static/ax.esm.js";
import { h, mountHistoryDrawer } from "./ui.js";
import { subscribe } from "./state.js";
import { GROUPS, SECTION_TITLES, findSection } from "./panels/registry.js";
import { openPalette, mountPaletteHotkey } from "./palette.js";

function registerComponents() {
  let count = 0;
  for (const value of Object.values(axui)) {
    if (typeof value === "function" && value.name.endsWith("Elem")) {
      regComp(value);
      count++;
    }
  }
  return count;
}

const LS_TAB_KEY = "gm-ax-tab";
const LS_GROUP_KEY = "gm-ax-group-collapsed";

/* ---- 导航 ---- */

function collapsedSet() {
  try {
    const raw = localStorage.getItem(LS_GROUP_KEY);
    if (raw) return new Set(JSON.parse(raw));
  } catch {
    /* 损坏则回退默认 */
  }
  return new Set(); // 新版默认全展开（组少且可折叠）
}

function saveCollapsed(set) {
  localStorage.setItem(LS_GROUP_KEY, JSON.stringify(Array.from(set)));
}

function buildNav() {
  const nav = document.querySelector("#gm-nav");
  nav.innerHTML = "";
  const collapsed = collapsedSet();
  for (const group of GROUPS) {
    const items = h("div", { class: "grp-items" });
    for (const sectionId of group.sections) {
      const link = h("a", { "data-target": sectionId }, [SECTION_TITLES[sectionId] || sectionId]);
      link.addEventListener("click", () => {
        location.hash = "#/" + sectionId;
        closeNavOnNarrow();
      });
      items.appendChild(link);
    }
    const box = h("div", { class: "grp" + (collapsed.has(group.id) ? "" : " open"), "data-group": group.id }, [
      h("div", { class: "grp-hdr", title: group.sections.length + " 个板块" }, [group.name]),
      items,
    ]);
    box.querySelector(".grp-hdr").addEventListener("click", () => {
      box.classList.toggle("open");
      const set = collapsedSet();
      if (box.classList.contains("open")) set.delete(group.id);
      else set.add(group.id);
      saveCollapsed(set);
    });
    nav.appendChild(box);
  }
}

function markActiveNav(sectionId) {
  document.querySelectorAll("#gm-nav a").forEach((link) => {
    link.classList.toggle("active", link.dataset.target === sectionId);
  });
  const active = document.querySelector('#gm-nav a.active');
  if (active) {
    active.closest(".grp")?.classList.add("open");
  }
}

/* ---- 路由 ---- */

const loadedGroups = new Map(); // groupId -> {sections: Map<id, HTMLElement>}

async function showSection(sectionId) {
  const found = findSection(sectionId);
  if (!found) {
    location.hash = "#/" + GROUPS[0].sections[0];
    return;
  }
  const { group } = found;
  localStorage.setItem(LS_TAB_KEY, sectionId);
  markActiveNav(sectionId);

  const view = document.querySelector("#view");
  if (!loadedGroups.has(group.id)) {
    view.innerHTML = "";
    view.appendChild(
      h("section", { class: "panel" }, [h("ax-skeleton", { type: "card" })])
    );
    try {
      const module = await group.load();
      const holder = new Map();
      for (const id of group.sections) {
        const host = h("div", { "data-section": id, style: "display:none;" });
        view.innerHTML = "";
        view.appendChild(host);
        module.render(host, { sectionId: id });
        holder.set(id, host);
      }
      loadedGroups.set(group.id, holder);
    } catch (err) {
      console.error("panel load failed", err);
      view.innerHTML = "";
      view.appendChild(
        h("section", { class: "panel" }, [
          h("h2", {}, ["加载失败"]),
          h("p", { class: "desc" }, [String((err && err.message) || err)]),
        ])
      );
      return;
    }
  }
  const holder = loadedGroups.get(group.id);
  const view2 = document.querySelector("#view");
  view2.innerHTML = "";
  for (const id of group.sections) {
    const host = holder.get(id);
    host.style.display = id === sectionId ? "" : "none";
    view2.appendChild(host);
  }
}

function routeFromHash() {
  const id = (location.hash || "").replace(/^#\//, "");
  if (id && findSection(id)) showSection(id);
  else {
    const saved = localStorage.getItem(LS_TAB_KEY);
    if (saved && findSection(saved)) location.hash = "#/" + saved;
    else location.hash = "#/" + GROUPS[0].sections[0];
  }
}

/* ---- 顶栏角色卡 / 底栏状态 ---- */

function chip(label, value, title) {
  return h("span", { class: "chip", title: title || "" }, [h("b", {}, [label]), String(value)]);
}

function renderPlayerCard(state) {
  const box = document.querySelector("#player-card");
  if (!box) return;
  box.innerHTML = "";
  if (!state.reachable) {
    box.appendChild(chip("状态", "游戏服不可达——等待自动恢复"));
    return;
  }
  if (!state.sessions.length) {
    box.appendChild(chip("状态", "没有在线角色——请先登录游戏"));
    return;
  }
  const player = state.sessions[0];
  const position = player.position && typeof player.position.x === "number"
    ? player.position.x.toFixed(1) + ", " + player.position.y.toFixed(1) + ", " + player.position.z.toFixed(1)
    : "-";
  const scene = (player.scene && (player.scene.name || player.scene.config)) || "-";
  box.appendChild(chip("角色", player.name || "-"));
  box.appendChild(chip("门派", player.factionName || player.faction || "-", player.faction || ""));
  box.appendChild(chip("场景", scene, (player.scene && player.scene.config) || ""));
  box.appendChild(chip("坐标", position));
  box.appendChild(chip("碎银", player.silver != null ? player.silver : "-"));
  box.appendChild(chip("官银", player.guanYin != null ? player.guanYin : "-"));
  box.appendChild(chip("黄金", player.gold != null ? player.gold : "-"));
  box.appendChild(chip("怒气", player.sp != null ? player.sp : "-"));
  box.appendChild(chip("修为", player.faculty != null ? player.faculty : "-"));
  box.appendChild(chip("最近操作", player.lastAction || "-"));
}

function renderConnStatus(state) {
  const conn = document.querySelector("#conn-status");
  if (conn) {
    conn.classList.toggle("online", state.reachable);
    conn.classList.toggle("offline", !state.reachable);
    conn.innerHTML = "";
    conn.appendChild(h("span", { class: "dot" }));
    conn.appendChild(document.createTextNode(state.reachable ? "已连接（在线 " + state.sessions.length + "）" : "游戏服不可达·自动恢复中"));
  }
  const last = document.querySelector("#last-refresh");
  if (last && state.updatedAt) {
    last.textContent = "最后刷新 " + state.updatedAt.toTimeString().slice(0, 8);
  }
}

/* ---- 窄屏导航开关 ---- */

function closeNavOnNarrow() {
  if (window.innerWidth <= 960) document.querySelector("#gm-nav").classList.remove("open");
}

function wireNavToggle() {
  const toggle = document.querySelector("#nav-toggle");
  toggle.addEventListener("click", () => {
    document.querySelector("#gm-nav").classList.toggle("open");
  });
}

/* ---- 主题切换（dark 默认，localStorage 记忆） ---- */

const SCHEME_KEY = "gm-ax-scheme";

function applyScheme(scheme) {
  if (scheme === "light") document.body.removeAttribute("scheme");
  else document.body.setAttribute("scheme", "dark");
}

function wireThemeToggle() {
  const btn = document.querySelector("#theme-btn");
  if (!btn) return;
  applyScheme(localStorage.getItem(SCHEME_KEY) || "dark");
  btn.addEventListener("click", () => {
    const next = document.body.hasAttribute("scheme") ? "light" : "dark";
    localStorage.setItem(SCHEME_KEY, next);
    applyScheme(next);
  });
}

/* ---- 启动 ---- */

function main() {
  const registered = registerComponents();
  console.info("[gm-ax] AXUI 组件已注册 " + registered + " 个" + (customElements.get("ax-btn") ? "（ax-btn ✓）" : "（ax-btn 缺失！）"));
  buildNav();
  wireNavToggle();
  wireThemeToggle();
  mountHistoryDrawer();
  mountPaletteHotkey();
  const paletteBtn = document.querySelector("#palette-btn");
  if (paletteBtn) paletteBtn.addEventListener("click", () => openPalette());
  const badge = document.querySelector("#gm-server-badge");
  badge.innerHTML = "<b>服务器</b>";
  badge.appendChild(document.createTextNode(location.host || "本机"));
  subscribe((state) => {
    renderPlayerCard(state);
    renderConnStatus(state);
  });
  window.addEventListener("hashchange", routeFromHash);
  routeFromHash();
}

main();
