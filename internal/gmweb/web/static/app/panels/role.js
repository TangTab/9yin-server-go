// role.js —— 组1「角色状态」：在线角色 / 五维 / 实力等级 / 活力 / 性格 / 称号。
// 命令参数与旧版 console-features.js 逐字对齐（附录 A 映射表）。

import { h, panelSection, numField, cmdBtn, catalogPicker, hint, toast } from "../ui.js";
import { catalog, command } from "../api.js";
import { subscribe, refreshNow } from "../state.js";

function renderOnline(host) {
  const { section, body } = panelSection({
    id: "sec-online",
    title: "在线角色",
    desc: "仅监听 127.0.0.1。命令直接交给在线游戏会话执行；顶部实时显示当前角色状态。",
  });
  const card = h("div", { class: "player-chips", style: "margin-bottom:10px;" }, ["读取中……"]);
  const details = h("details", {}, [h("summary", {}, ["原始会话 JSON（排查用）"])]);
  const pre = h("pre", { style: "max-height:260px;overflow:auto;font-size:12px;" }, [""]);
  details.appendChild(pre);
  subscribe((state) => {
    if (!state.reachable) {
      card.innerHTML = "";
      card.appendChild(h("span", { class: "chip" }, ["游戏服不可达——等待自动恢复"]));
      pre.textContent = "";
      return;
    }
    card.innerHTML = "";
    if (!state.sessions.length) {
      card.appendChild(h("span", { class: "chip" }, ["没有在线角色——请先登录游戏"]));
    } else {
      const player = state.sessions[0];
      for (const [label, value] of [
        ["角色", player.name],
        ["门派", player.factionName || player.faction],
        ["场景", (player.scene && (player.scene.name || player.scene.config)) || "-"],
        ["碎银", player.silver],
        ["官银", player.guanYin],
        ["黄金", player.gold],
        ["怒气", player.sp],
        ["修为", player.faculty],
      ]) {
        card.appendChild(h("span", { class: "chip" }, [h("b", {}, [label]), String(value ?? "-")]));
      }
    }
    pre.textContent = JSON.stringify(state.sessions, null, 2);
  });
  body.appendChild(card);
  const refreshBtn = h("ax-btn", { label: "刷新", type: "plain" });
  refreshBtn.addEventListener("click", async () => {
    await refreshNow();
    toast("会话状态已刷新", "ok");
  });
  body.appendChild(
    h("div", { class: "actions-row" }, [
      cmdBtn({ label: "弹出游戏 GM 面板", theme: "prim", action: "popup_gm_panel", section: "在线角色" }),
      refreshBtn,
    ])
  );
  body.appendChild(details);
  host.appendChild(section);
}

function renderFive(host) {
  const { section, body } = panelSection({
    id: "sec-five",
    title: "五维属性",
    desc: "直接修改角色五维（臂力/体魄/身法/内息/罡气），改动后立即重算派生战斗属性并实时刷新客户端面板。留空或填 0 表示该维不修改。属性为覆盖式写入（非累加）。",
  });
  const defs = [
    ["attrStr", "臂力", 0],
    ["attrSta", "体魄", 0],
    ["attrDex", "身法", 0],
    ["attrIng", "内息", 0],
    ["attrSpi", "罡气", 0],
  ];
  const fields = {};
  const row = h("div", { class: "field-row" });
  for (const [key, label, def] of defs) {
    fields[key] = numField({ label, value: def, min: 0, width: "110px" });
    row.appendChild(fields[key].node);
  }
  const note = hint();
  body.appendChild(row);
  const apply = h("ax-btn", { label: "应用五维并刷新", theme: "prim" });
  apply.addEventListener("click", async () => {
    const payload = { action: "set_five_attributes" };
    for (const [key] of defs) payload[key] = fields[key].get() ?? 0;
    const result = await command(payload);
    note.set(result.ok ? "五维已应用" : result.message, result.ok);
    if (result.ok) refreshNow();
  });
  body.appendChild(h("div", { class: "actions-row" }, [apply]));
  body.appendChild(note.node);
  host.appendChild(section);
}

function renderPower(host) {
  const { section, body } = panelSection({
    id: "sec-power",
    title: "实力等级",
    desc: "设置玩家 PowerLevel（0–160），服务端自动按功力等级映射称号 LevelTitle 并刷新界面。",
  });
  const value = numField({ label: "实力等级", value: 0, min: 0, max: 160, width: "110px" });
  body.appendChild(h("div", { class: "field-row" }, [value.node]));
  body.appendChild(
    h("div", { class: "actions-row" }, [
      (() => {
        const btn = h("ax-btn", { label: "设置实力等级", theme: "prim" });
        btn.addEventListener("click", async () => {
                const result = await command({ action: "set_power_level", value: value.get() ?? 0 });
          toast(result.ok ? "实力等级已设置" : result.message, result.ok ? "ok" : "err");
          if (result.ok) refreshNow();
        });
        return btn;
      })(),
    ])
  );
  host.appendChild(section);
}

function renderVitals(host) {
  const { section, body } = panelSection({
    id: "sec-vitals",
    title: "体力 / 饱食 / 杀戮 / 酒量",
    desc: "一次性设置 9 项活力属性；当前体力/饱食默认填满至上限，上限加成默认 0。服务端刷新界面并持久化 save=true 项（体力/饱食/杀戮值/酒量等级）。",
  });
  const defs = [
    ["ene", "当前体力", 123],
    ["maxEne", "体力上限", 123],
    ["maxEneAdd", "体力上限加成", 0],
    ["resumeEne", "体力恢复/秒", 0],
    ["sat", "当前饱食", 123],
    ["maxSat", "饱食上限", 123],
    ["maxSatAdd", "饱食上限加成", 0],
    ["pkValue", "杀戮值", 0],
    ["drunkLevel", "酒量等级", 1],
  ];
  const fields = {};
  const row = h("div", { class: "field-row" });
  for (const [key, label, def] of defs) {
    fields[key] = numField({ label, value: def, width: "110px" });
    row.appendChild(fields[key].node);
  }
  body.appendChild(row);
  const apply = h("ax-btn", { label: "设置活力属性", theme: "prim" });
  apply.addEventListener("click", async () => {
    const payload = { action: "set_vitals" };
    for (const [key] of defs) payload[key] = fields[key].get() ?? 0;
    const result = await command(payload);
    toast(result.ok ? "活力属性已设置" : result.message, result.ok ? "ok" : "err");
    if (result.ok) refreshNow();
  });
  body.appendChild(h("div", { class: "actions-row" }, [apply]));
  host.appendChild(section);
}

function renderAlignment(host) {
  const { section, body } = panelSection({
    id: "sec-alignment",
    title: "性格善恶",
    desc: "设置玩家 善值/恶值/基础狂邪，服务端自动计算性格标签（0正常 1侠 2恶 3狂 4邪）与性格数值并刷新界面。",
  });
  const justice = numField({ label: "善值", value: 0, width: "110px" });
  const evil = numField({ label: "恶值", value: 0, width: "110px" });
  const freeBase = numField({ label: "基础狂邪", value: 0, width: "110px" });
  body.appendChild(h("div", { class: "field-row" }, [justice.node, evil.node, freeBase.node]));
  const apply = h("ax-btn", { label: "设置性格善恶", theme: "prim" });
  apply.addEventListener("click", async () => {
    const result = await command({
      action: "set_alignment",
      justice: justice.get() ?? 0,
      evil: evil.get() ?? 0,
      freeBase: freeBase.get() ?? 0,
    });
    toast(result.ok ? "性格善恶已设置" : result.message, result.ok ? "ok" : "err");
    if (result.ok) refreshNow();
  });
  body.appendChild(h("div", { class: "actions-row" }, [apply]));
  host.appendChild(section);
}

function renderTitles(host) {
  const { section, body } = panelSection({
    id: "sec-titles",
    title: "称号管理",
    desc: "玩家初始无任何称号；发放/佩戴/取消均经服务端校验。称号 ID 与名称取自称号总表。佩戴即写入 TitleID 属性并刷新界面。",
  });
  let selected = null;
  const note = hint();
  const picker = catalogPicker({
    recentKey: "titles",
    placeholder: "搜索称号中文名或 ID，例如：隐士高人、9193",
    emptyText: "没有匹配的称号",
    fetchRows: async (query) => {
      const rows = await catalog("/api/titles");
      const needle = query.toLowerCase();
      if (!needle) return rows.map((row) => ({ label: row.name, sub: "ID " + row.id, value: row.id }));
      return rows
        .filter((row) => row.name.toLowerCase().includes(needle) || String(row.id) === needle)
        .map((row) => ({ label: row.name, sub: "ID " + row.id, value: row.id }));
    },
    onPick(row) {
      selected = row;
      note.set("已选中：" + row.label + "（ID " + row.value + "）", true);
    },
  });
  body.appendChild(picker.node);
  body.appendChild(
    h("div", { class: "actions-row" }, [
      (() => {
        const btn = h("ax-btn", { label: "发放选中称号", theme: "prim" });
        btn.addEventListener("click", () => {
          if (!selected) return toast("请先选择一个称号", "warn");
          command({ action: "grant_title", titleId: selected.value }).then((r) => {
            note.set(r.ok ? "已发放：" + selected.label : r.message, r.ok);
            if (r.ok) refreshNow();
          });
        });
        return btn;
      })(),
      (() => {
        const btn = h("ax-btn", { label: "佩戴选中称号" });
        btn.addEventListener("click", () => {
          if (!selected) return toast("请先选择一个称号", "warn");
          command({ action: "wear_title", titleId: selected.value }).then((r) => {
            note.set(r.ok ? "已佩戴：" + selected.label : r.message, r.ok);
            if (r.ok) refreshNow();
          });
        });
        return btn;
      })(),
      cmdBtn({ label: "取消佩戴", action: "cancel_title", section: "称号管理" }),
      cmdBtn({
        label: "清空全部称号",
        danger: true,
        confirm: "确定清空该角色全部称号并取消佩戴？",
        action: "clear_titles",
        section: "称号管理",
      }),
    ])
  );
  body.appendChild(note.node);
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = {
    online: renderOnline,
    five: renderFive,
    power: renderPower,
    vitals: renderVitals,
    alignment: renderAlignment,
    titles: renderTitles,
  };
  (renderers[sectionId] || renderOnline)(host);
}
