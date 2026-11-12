// skill.js —— 组5「武学与修炼」：门派 / 武学套路 / 修为 / 经脉 / 经脉周天。
// 套路与内功目录按需拉取（一次全量 + 本地过滤 + 最近选用）。

import { h, panelSection, numField, cmdBtn, actionBtn, catalogPicker, hint, toast } from "../ui.js";
import { catalog } from "../api.js";
import { refreshNow } from "../state.js";

/* ---------------- 门派 ---------------- */

const FACTION_GROUPS = [
  ["八大门派", ["shaolin:少林", "wudang:武当", "emei:峨眉", "gaibang:丐帮", "tangmen:唐门", "junzitang:君子堂", "jinyiwei:锦衣卫", "jilegu:极乐谷"]],
  ["后续门派", ["mingjiao:明教", "tianshan:天山派", "kunlun:昆仑派", "xingtianbao:刑天城", "wuxu:无虚门"]],
  ["江湖势力", ["yihuagong:移花宫", "taohuadao:桃花岛", "wugenmen:无根门", "xujia:徐家庄", "wanshou:万兽山庄", "jinzhen:金针沈家", "tianlun:天轮寺", "qingyi:青衣阁"]],
  ["宗门", ["xuedao:血刀门", "huashan:华山派", "gumu:古墓派", "damo:达摩派", "shenshui:神水宫", "changfeng:长风镖局", "nianluo:念萝坝", "wuxian:无仙教", "shenji:神机营", "xingmiao:星渺阁", "tianya:天涯海阁", "wanghui:王回洲", "shenjihui:神机汇"]],
];

function renderFaction(host) {
  const { section, body } = panelSection({
    id: "sec-faction",
    title: "门派",
    desc: "修改角色原生门派属性（School/Force/NewSchool）并持久化落库；不自动赠送对应套路、身份装或任务线（身份装可用「外观与坐骑」另行激活）。",
  });
  const content = [];
  for (const [group, items] of FACTION_GROUPS) {
    for (const item of items) {
      const [key, name] = item.split(":");
      content.push({ label: name + "（" + group + "）", value: key });
    }
  }
  const factionSelect = h("ax-select", { content: JSON.stringify(content), search: true, full: true, "max-height": "300" });
  const apply = actionBtn({
    label: "应用门派",
    build: () => (factionSelect.value ? { action: "set_faction", faction: factionSelect.value } : (toast("请先选择门派", "warn"), null)),
    okText: "门派已应用",
    section: "门派",
  });
  body.appendChild(h("div", { class: "field-row" }, [factionSelect]));
  body.appendChild(h("div", { class: "actions-row" }, [apply, cmdBtn({ label: "一键学会全部内功", action: "learn_all_neigong", section: "门派" })]));
  host.appendChild(section);
}

/* ---------------- 武学套路 ---------------- */

function renderTaolu(host) {
  const { section, body } = panelSection({
    id: "sec-taolu",
    title: "武学套路",
    desc: "整套学习/移除：给指定套路/内功一次性发放并直达指定层数。层数留空或 0=各自满级。移除为学习的逆操作：非登录基线项彻底删除，基线项重置回初始档。等级重登保持。",
  });
  let taoluSelected = null;
  let neigongSelected = null;
  const taoluLevel = numField({ label: "层数（0=满级）", value: "", min: 0, max: 999, width: "150px" });
  const neigongLevel = numField({ label: "层数（0=满级）", value: "", min: 0, max: 999, width: "150px" });
  const taoluPicker = catalogPicker({
    recentKey: "taolus",
    placeholder: "搜索套路（拉取一次后本地过滤）",
    emptyText: "没有匹配的套路",
    fetchRows: async (query) => {
      const rows = await catalog("/api/taolus");
      const needle = query.toLowerCase();
      return rows
        .filter((row) => !needle || row.label.toLowerCase().includes(needle) || row.key.toLowerCase().includes(needle))
        .map((row) => ({ label: row.label, sub: row.key + " · " + row.skills.length + " 招", value: row.key }));
    },
    onPick(row) {
      taoluSelected = row;
    },
  });
  const neigongPicker = catalogPicker({
    recentKey: "neigongs",
    placeholder: "搜索内功（拉取一次后本地过滤）",
    emptyText: "没有匹配的内功",
    fetchRows: async (query) => {
      const rows = await catalog("/api/neigongs");
      const needle = query.toLowerCase();
      return rows
        .filter((row) => !needle || (row.name || "").toLowerCase().includes(needle) || row.configId.toLowerCase().includes(needle))
        .map((row) => ({ label: row.name || row.configId, sub: row.configId + " · 满 " + row.maxLevel + " 层", value: row.configId }));
    },
    onPick(row) {
      neigongSelected = row;
    },
  });
  body.appendChild(h("h3", { style: "margin:0 0 6px;font-size:14px;" }, ["套路"]));
  body.appendChild(taoluPicker.node);
  body.appendChild(
    h("div", { class: "actions-row", style: "margin:8px 0 14px;" }, [
      taoluLevel.node,
      actionBtn({
        label: "学习所选套路",
        build: () => (taoluSelected ? { action: "learn_taolu", skillKey: taoluSelected.value, skillLevel: taoluLevel.get() ?? 0 } : (toast("请先选择套路", "warn"), null)),
        okText: () => "已学习：" + taoluSelected.label,
        section: "武学套路",
      }),
      actionBtn({
        label: "移除所选套路",
        danger: true,
        confirm: "确定移除所选套路？（非登录基线招式彻底删除，基线招式重置回 1 层）",
        build: () => (taoluSelected ? { action: "forget_taolu", skillKey: taoluSelected.value } : (toast("请先选择套路", "warn"), null)),
        okText: "已移除",
        section: "武学套路",
      }),
    ])
  );
  body.appendChild(h("h3", { style: "margin:0 0 6px;font-size:14px;" }, ["内功"]));
  body.appendChild(neigongPicker.node);
  body.appendChild(
    h("div", { class: "actions-row", style: "margin-top:8px;" }, [
      neigongLevel.node,
      actionBtn({
        label: "学习所选内功",
        build: () => (neigongSelected ? { action: "learn_neigong", skillKey: neigongSelected.value, skillLevel: neigongLevel.get() ?? 0 } : (toast("请先选择内功", "warn"), null)),
        okText: () => "已学习：" + neigongSelected.label,
        section: "武学套路",
      }),
      actionBtn({
        label: "移除所选内功",
        danger: true,
        confirm: "确定移除所选内功？（非登录基线项彻底删除，基线项重置回初始档）",
        build: () => (neigongSelected ? { action: "forget_neigong", skillKey: neigongSelected.value } : (toast("请先选择内功", "warn"), null)),
        okText: "已移除",
        section: "武学套路",
      }),
    ])
  );
  body.appendChild(
    h("div", { class: "actions-row", style: "margin-top:14px;border-top:1px solid var(--gm-line);padding-top:12px;" }, [
      cmdBtn({
        label: "一键学满全部武学套路",
        danger: true,
        confirm: "确定把全部武学套路升到满级？",
        action: "learn_all_taolu",
        section: "武学套路",
      }),
      cmdBtn({
        label: "删除全部武学（重置）",
        danger: true,
        confirm: "确定删除全部已学武学套路（重置为初始状态）？",
        action: "forget_all_taolu",
        section: "武学套路",
      }),
    ])
  );
  host.appendChild(section);
}

/* ---------------- 修为 ---------------- */

function renderFaculty(host) {
  const { section, body } = panelSection({
    id: "sec-faculty",
    title: "修为",
    desc: "修为=修炼点余额（主供给为历练转化）。修炼速度拆为基础值 × 倍率；填满当前内功修炼度立即升一层。角色级持久化，重登沿用。",
  });
  const value = numField({ label: "修为", value: 10000, min: 0, max: 2000000000, width: "120px" });
  const base = numField({ label: "基础值/min", value: 200, min: 1, max: 1000000, width: "150px" });
  const multiplier = numField({ label: "倍率", value: 1, min: 0.1, max: 100, step: 0.1, width: "80px" });
  const dayLimit = numField({ label: "每日碎银上限(两)", value: 0, min: 0, max: 999999, width: "150px" });
  const livepoint = numField({ label: "历练点数", value: 99, min: 1, max: 999, width: "150px" });
  const hold = numField({ label: "潜能修为", value: 0, min: 0, max: 2000000000, width: "150px" });
  const livePower = numField({ label: "转化加成%", value: 0, min: 0, max: 1000, width: "150px" });
  body.appendChild(
    h("div", { class: "field-row" }, [
      value.node,
      actionBtn({
        label: "设置修为",
        build: () => ({ action: "set_faculty", value: value.get() ?? 0 }),
        okText: "修为已设置",
        section: "修为",
      }),
      cmdBtn({
        label: "填满当前内功修炼度（升一层）",
        danger: true,
        confirm: "确定把当前修炼内功的本层修炼度填满（立即升一层）？",
        action: "fill_current_neigong",
        section: "修为",
      }),
    ])
  );
  body.appendChild(
    h("div", { class: "field-row" }, [
      base.node,
      multiplier.node,
      actionBtn({
        label: "应用修炼速度",
        build: () => ({ action: "set_fill_speed", base: base.get() ?? 200, multiplier: multiplier.get() ?? 1 }),
        okText: "修炼速度已应用",
        section: "修为",
      }),
    ])
  );
  body.appendChild(
    h("div", { class: "field-row" }, [
      dayLimit.node,
      actionBtn({
        label: "应用演武碎银上限",
        build: () => ({ action: "set_act_day_limit", value: dayLimit.get() ?? 0 }),
        okText: "演武碎银上限已应用（0=不限制）",
        section: "修为",
      }),
      cmdBtn({ label: "清除团练疲劳", action: "clear_team_fatigue", section: "修为" }),
    ])
  );
  body.appendChild(
    h("div", { class: "field-row" }, [
      livepoint.node,
      actionBtn({
        label: "发放历练",
        build: () => ({ action: "add_livepoint", value: livepoint.get() ?? 99 }),
        okText: "历练已发放",
        section: "修为",
      }),
      hold.node,
      actionBtn({
        label: "设置潜能修为",
        build: () => ({ action: "set_hold_faculty", value: hold.get() ?? 0 }),
        okText: "潜能修为已设置",
        section: "修为",
      }),
      livePower.node,
      actionBtn({
        label: "应用转化加成",
        build: () => ({ action: "set_live_speed_power", value: livePower.get() ?? 0 }),
        okText: "转化加成已应用",
        section: "修为",
      }),
    ])
  );
  host.appendChild(section);
}

/* ---------------- 经脉（每日真气硬闸） ---------------- */

function renderJingmai(host) {
  const { section, body } = panelSection({
    id: "sec-jingmai",
    title: "经脉",
    desc: "每日真气硬闸：开启后当日已获真气达到阈值即停止入账，次日 UTC 日切自动恢复；关闭则不限量。角色级持久化，修改立即下发属性帧。",
  });
  const enabled = h("input", { type: "checkbox" });
  enabled.style.cssText = "width:16px;height:16px;accent-color:var(--gm-gold);";
  const value = numField({ label: "每日阈值", value: 500, min: 1, max: 10000000, width: "150px" });
  const apply = actionBtn({
    label: "应用设置",
    build: () => {
      const threshold = value.get();
      if (!threshold || threshold < 1) return (toast("每日阈值必须 ≥ 1", "warn"), null);
      return { action: "set_zhenqi_cap", enabled: enabled.checked ? 1 : 0, value: threshold };
    },
    okText: "真气硬闸已应用",
    section: "经脉",
  });
  body.appendChild(h("div", { class: "field-row" }, [h("label", {}, ["是否开启每日硬闸"]), enabled, value.node, apply]));
  host.appendChild(section);
}

/* ---------------- 经脉周天 ---------------- */

function renderJingmaiTier(host) {
  const { section, body } = panelSection({
    id: "sec-jingmai-tier",
    title: "经脉周天设置",
    desc: "所有正经默认已开通为 216 周天（客户端可直接点激活/关闭）。此处用于把选中经脉的周天档位改为 36/72/…/216（同时改最大层数和当前层数），改完重推经脉视图并刷新五维。",
  });
  let selected = null;
  const tierSelect = h("ax-select", {
    content: JSON.stringify([
      { label: "36 周天", value: "36" }, { label: "72 周天", value: "72" }, { label: "108 周天", value: "108" },
      { label: "144 周天", value: "144" }, { label: "180 周天", value: "180" }, { label: "216 周天", value: "216" },
    ]),
    style: "width:120px;",
    value: "216",
  });
  const picker = catalogPicker({
    recentKey: "meridians",
    placeholder: "搜索经脉（拉取一次后本地过滤）",
    emptyText: "没有匹配的经脉",
    fetchRows: async (query) => {
      const rows = await catalog("/api/meridians");
      const needle = query.toLowerCase();
      return rows
        .filter((row) => !needle || (row.name || "").toLowerCase().includes(needle) || row.id.toLowerCase().includes(needle))
        .map((row) => ({ label: row.name || row.id, sub: row.id + " · 默认满 " + row.defaultMax + " 层", value: row.id }));
    },
    onPick(row) {
      selected = row;
    },
  });
  const apply = actionBtn({
    label: "设置周天",
    build: () => (selected ? { action: "set_jingmai_tier", jingMaiId: selected.value, jingMaiTier: Number(tierSelect.value || 216) } : (toast("请先选择一条经脉", "warn"), null)),
    okText: () => "周天已设置：" + selected.label,
    section: "经脉周天",
  });
  body.appendChild(picker.node);
  body.appendChild(h("div", { class: "actions-row", style: "margin-top:8px;" }, [h("label", {}, ["目标档位"]), tierSelect, apply]));
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = { faction: renderFaction, taolu: renderTaolu, faculty: renderFaculty, jingmai: renderJingmai, "jingmai-tier": renderJingmaiTier };
  (renderers[sectionId] || renderFaction)(host);
}
