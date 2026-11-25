// main.js —— 运维面板启动入口：AXUI 组件注册、导航、hash 路由、全局轮询。
//
// AXUI 的 ESM 构建【不自动注册 Web Component】（regComp 已导出但从不被
// 调用）——必须启动时遍历命名空间把 *Elem 类逐个 define，否则
// <ax-btn>/<ax-select> 等全部不渲染（wt1653 线 2026-11-12 浏览器实证：
// 未注册时按钮 0 尺寸）。

import * as axui from "/static/ax.esm.js";
import { regComp } from "/static/ax.esm.js";
import { h } from "./ui.js";
import * as api from "./api.js";
import { setRefreshHook } from "./ops.js";

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

/* ---- 面板注册表（懒加载；children = 二级目录，分组语义对齐 2433
   gm-panel 的 panels/registry.js GROUPS）。children 与子页自己那份清单
   （panels/test.js 的 SUBS）是两份手写表，须同项同序——nav_test.go 守卫 ---- */

const SECTIONS = [
  { id: "overview", title: "总览", load: () => import("./panels/overview.js") },
  { id: "services", title: "服务管理", load: () => import("./panels/services.js") },
  { id: "logs", title: "日志", load: () => import("./panels/logs.js") },
  { id: "players", title: "在线玩家", load: () => import("./panels/players.js") },
  { id: "db", title: "角色 / 告警", load: () => import("./panels/db.js") },
  { id: "gm", title: "GM 命令", load: () => import("./panels/gm.js") },
  {
    id: "test", title: "GM 测试台", load: () => import("./panels/test.js"),
    children: [
      { id: "role", title: "角色状态" },
      { id: "item", title: "货币与物品" },
      { id: "mail", title: "邮件" },
      { id: "skill", title: "武学与修炼" },
      { id: "dummy", title: "训练木桩" },
      { id: "scene", title: "传送与场景" },
      { id: "debug", title: "调试工具" },
      { id: "archive", title: "存档管理" },
    ],
  },
];

let activePanel = null; // {refresh(): void} —— 全局 tick 调当前页刷新

/* ---- 导航 / 路由 ---- */

function buildNav() {
  const nav = document.querySelector("#side-nav");
  nav.innerHTML = "";
  for (const sec of SECTIONS) {
    const link = h("a", { "data-target": sec.id }, [sec.title]);
    link.addEventListener("click", () => {
      location.hash = "#/" + sec.id + (sec.children ? "/" + sec.children[0].id : "");
    });
    nav.appendChild(link);
    for (const child of sec.children || []) {
      const sub = h("a", { class: "sub", "data-target": sec.id + "/" + child.id }, [child.title]);
      sub.addEventListener("click", () => {
        location.hash = "#/" + sec.id + "/" + child.id;
      });
      nav.appendChild(sub);
    }
  }
}

function markActive(target) {
  // 子项精确命中；无子项命中时父级保持高亮（组内其他视图）
  document.querySelectorAll("#side-nav a").forEach((link) => {
    link.classList.toggle("active", link.dataset.target === target);
  });
  if (!document.querySelector("#side-nav a.active")) {
    const head = target.split("/")[0];
    document.querySelectorAll('#side-nav a:not(.sub)').forEach((link) => {
      link.classList.toggle("active", link.dataset.target === head);
    });
  }
}

// parseHash 拆 #/<section>[/<sub>]（二级目录路由）。
function parseHash() {
  const path = location.hash.replace(/^#\//, "") || "overview";
  const seg = path.split("/").filter(Boolean);
  return { id: seg[0] || "overview", sub: seg[1] || "" };
}

async function showSection() {
  const { id, sub } = parseHash();
  const sec = SECTIONS.find((s) => s.id === id) || SECTIONS[0];
  markActive(sec.id + (sub ? "/" + sub : ""));
  const view = document.querySelector("#view");
  view.innerHTML = "";
  view.appendChild(h("section", { class: "panel" }, [h("ax-skeleton", { type: "card" })]));
  activePanel = null;
  try {
    const mod = await sec.load();
    view.innerHTML = "";
    activePanel = mod.mount(view, sub);
  } catch (err) {
    console.error("panel load failed", err);
    view.innerHTML = "";
    view.appendChild(h("section", { class: "panel" }, ["面板加载失败：" + (err && err.message ? err.message : err)]));
  }
}

/* ---- 顶栏全局状态（跟随轮询） ---- */

function renderTopbar(data) {
  const set = (id, text) => {
    const node = document.querySelector(id);
    if (node) node.textContent = text;
  };
  set("#mode-badge", data.mode === "single" ? "合一形态" : "集群形态");
  set("#world-badge", "world " + (data.world && data.world.alive ? data.world.state || "alive" : "离线"));
  set("#online-badge", "在线 " + (data.online >= 0 ? data.online : "-"));
  const db = document.querySelector("#db-badge");
  if (db) db.textContent = "DB " + (data.db ? "正常" : "不可用");
  if (db) db.style.color = data.db ? "" : "#ed4014";
  set("#clock", data.time || "");
  const conn = document.querySelector("#conn-status");
  if (conn) {
    conn.querySelector(".dot").className = "dot on";
    conn.lastChild.textContent = "已连接";
  }
  document.querySelector("#last-refresh").textContent = "刷新于 " + new Date().toLocaleTimeString("zh-CN", { hour12: false });
}

/* ---- 全局轮询（2.5s：顶栏 + 当前页） ---- */

let lastOverview = null;

async function tick() {
  try {
    const data = await api.overview();
    lastOverview = data;
    renderTopbar(data);
    if (activePanel && typeof activePanel.refresh === "function") {
      activePanel.refresh(data);
    }
  } catch (err) {
    const conn = document.querySelector("#conn-status");
    if (conn) {
      conn.querySelector(".dot").className = "dot off";
      conn.lastChild.textContent = "面板不可达：" + (err.message || err);
    }
  }
}

/* ---- 启动 ---- */

registerComponents();
buildNav();
setRefreshHook(tick);
document.querySelector("#nav-toggle").addEventListener("click", () => document.body.classList.toggle("nav-open"));
window.addEventListener("hashchange", () => showSection());
showSection();
tick();
setInterval(tick, 2500);
