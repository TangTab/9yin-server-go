// db.js —— 角色查询（nx_roles，按账号）+ 告警流水（nx_monitor）。

import { h, confirmDanger } from "../ui.js";
import { runOp } from "../ops.js";
import * as api from "../api.js";

function sectionRoles(view) {
  const input = h("input", { type: "text", placeholder: "账号名（如 test001）", style: "width:220px" });
  const tbl = h("table", { class: "tbl" }, [
    h("tr", {}, ["账号", "角色", "UID", "区服", "在线", "状态", "操作"].map((t) => h("th", {}, [t]))),
  ]);
  const hint = h("div", { class: "panel-desc" }, ["查 nx_roles（含软删/跨区服）；数据来自 MySQL 直连。"]);

  async function query() {
    const account = input.value.trim();
    if (!account) return;
    try {
      const data = await api.roles(account);
      while (tbl.rows.length > 1) tbl.deleteRow(1);
      const list = data.roles || [];
      if (!list.length) {
        tbl.appendChild(h("tr", {}, [h("td", { class: "tbl-empty", colspan: "7" }, ["（无角色记录）"])]));
        return;
      }
      for (const r of list) {
        const detail = h("ax-btn", { label: "详情", type: "plain" });
        detail.addEventListener("click", async () => {
          try {
            const d = await api.roleDetail(r.account, r.name, r.serverId);
            const m = d.meta || {};
            hint.textContent = r.name + "：区服 " + m.serverId + " · 累计 " + (m.totalSec || 0) + "s · 场景 "
              + (m.sceneId || 0) + " · 位置 " + (m.location || "-") + " · 存档状态 "
              + (m.roleStateJson ? "有（role_state_json " + m.roleStateJson.length + "B）" : "无（旧档）");
          } catch (err) {
            hint.textContent = "详情读取失败：" + (err.message || err);
          }
        });
        tbl.appendChild(h("tr", {}, [
          h("td", {}, [r.account]),
          h("td", {}, [r.name]),
          h("td", {}, [r.uid || "-"]),
          h("td", {}, [String(r.serverId)]),
          h("td", {}, [r.online ? "●" : "-"]),
          h("td", {}, [r.deleted ? "已删" : "正常"]),
          h("td", {}, [detail]),
        ]));
      }
    } catch (err) {
      hint.textContent = "查询失败：" + (err.message || err);
    }
  }

  const btn = h("ax-btn", { label: "查询角色", theme: "info" });
  btn.addEventListener("click", query);
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter") query();
  });

  view.appendChild(h("section", { class: "panel" }, [
    h("h2", {}, ["角色查询"]),
    h("div", { class: "toolbar" }, [input, btn]),
    hint,
    tbl,
  ]));
}

function sectionAlerts(view) {
  const limitSel = h("select", {}, [50, 100, 500].map((n) => h("option", { value: String(n) }, ["最近 " + n + " 条"])));
  const tbl = h("table", { class: "tbl" }, [
    h("tr", {}, ["时间", "类型", "对象", "原因"].map((t) => h("th", {}, [t]))),
  ]);
  const hint = h("div", { class: "panel-desc" }, ["nx_monitor 告警流水（监控服 onReportWarning 落库）。"]);

  async function query() {
    try {
      const data = await api.alerts(parseInt(limitSel.value, 10) || 50);
      while (tbl.rows.length > 1) tbl.deleteRow(1);
      const list = data.alerts || [];
      if (!list.length) {
        tbl.appendChild(h("tr", {}, [h("td", { class: "tbl-empty", colspan: "4" }, ["（暂无告警）"])]));
        return;
      }
      for (const a of list) {
        tbl.appendChild(h("tr", {}, [
          h("td", {}, [a.time || "-"]),
          h("td", {}, [String(a.type)]),
          h("td", {}, [a.name || "-"]),
          h("td", { class: "wrap" }, [a.reason || "-"]),
        ]));
      }
    } catch (err) {
      hint.textContent = "告警读取失败：" + (err.message || err);
    }
  }

  limitSel.addEventListener("change", query);
  const btn = h("ax-btn", { label: "刷新", type: "plain" });
  btn.addEventListener("click", query);
  view.appendChild(h("section", { class: "panel" }, [
    h("h2", {}, ["告警流水"]),
    h("div", { class: "toolbar" }, [limitSel, btn]),
    hint,
    tbl,
  ]));
  query();
}

export function mount(view) {
  sectionRoles(view);
  sectionAlerts(view);
  return { refresh: () => {} };
}
