// fashion.js —— 组4「外观与坐骑」：时装一键 / 时装身份装发放 / 风物志卡牌 /
// 风物志头发 / 坐骑 / 待机站姿。全部走 M1 目录选择器（按需拉取 + 分页 +
// 最近选用），替代旧版全量 <select size=10>。

import { h, panelSection, numField, cmdBtn, catalogPicker, hint, toast } from "../ui.js";
import { catalog, command } from "../api.js";
import { refreshNow } from "../state.js";

async function applyCommand(payload, okText) {
  const result = await command(payload);
  toast(result.ok ? "✔ " + okText : "✘ " + result.message, result.ok ? "ok" : "err");
  if (result.ok) refreshNow();
  return result;
}

function renderFashionAll(host) {
  const { section, body } = panelSection({
    id: "sec-fashion-all",
    title: "时装",
    desc: "一键激活当前角色的全部服装（共 525 套），激活后属性面板「时装」页签显示为已激活，并写入新版服装数据永久存档。身份装：按当前角色门派激活门派身份套装（登录时也会自动补齐）。",
  });
  const typeSelect = h("ax-select", {
    content: JSON.stringify([
      { label: "0 江湖装", value: "0" },
      { label: "1 身份装", value: "1" },
      { label: "2 时装", value: "2" },
    ]),
    style: "width:140px;",
  });
  const setType = h("ax-btn", { label: "切换显示类型", type: "plain" });
  setType.addEventListener("click", async () => {
    await applyCommand({ action: "set_showequip_type", type: Number(typeSelect.value || 0) }, "显示类型已切换");
  });
  body.appendChild(
    h("div", { class: "actions-row" }, [
      cmdBtn({ label: "一键激活全部服装", theme: "prim", action: "grant_all_fashion", section: "时装" }),
      cmdBtn({ label: "激活本门派身份装", action: "grant_identity_suits", section: "时装" }),
      h("span", { class: "field-row" }, [typeSelect, setType]),
    ])
  );
  host.appendChild(section);
}

function renderFashionGrant(host) {
  const { section, body } = panelSection({
    id: "sec-fashion-grant",
    title: "时装 / 身份装发放",
    desc: "目录按 EquipType 分类：时装帽子/衣服/鞋子为分件，时装套装（FashionCoat）与身份装套装（Suit）为整套。「立即穿戴」直接穿进身份装/时装栏位并同步外观；「加入装备背包」由玩家自己穿。",
  });
  let selected = null;
  let categories = [];
  const catSelect = h("ax-select", { style: "min-width:170px;", placeholder: "选择分类" });
  const note = hint();
  const picker = catalogPicker({
    recentKey: "fashion",
    placeholder: "按中文名或 ConfigID 过滤，可先选分类缩小范围",
    emptyText: "没有匹配的时装/身份装",
    fetchRows: async (query) => {
      if (!categories.length) {
        categories = await catalog("/api/fashion-catalog");
        catSelect.set({
          key: "content",
          value: JSON.stringify([{ label: "全部分类", value: "" }, ...categories.map((c) => ({ label: c.label, value: c.key }))]),
        });
      }
      const activeKey = catSelect.value ? categoryKeyByLabel(catSelect.value) : "";
      const needle = query.toLowerCase();
      const out = [];
      for (const category of categories) {
        if (activeKey && category.key !== activeKey) continue;
        for (const item of category.items) {
          if (needle && !item.name.toLowerCase().includes(needle) && !item.configId.toLowerCase().includes(needle)) continue;
          out.push({ label: item.name, sub: category.label + " · " + item.configId, value: item.configId });
        }
      }
      return out;
    },
    onPick(row) {
      selected = row;
      note.set("已选中：" + row.label + "（" + row.value + "）", true);
    },
  });
  function categoryKeyByLabel(label) {
    const hit = categories.find((c) => c.label === label);
    return hit ? hit.key : "";
  }
  catSelect.addEventListener("change", () => picker.refresh()); // ax-select 只派发原生 change（「changed」走 .on() 通道）
  body.appendChild(h("div", { class: "field-row" }, [h("label", {}, ["分类"]), catSelect]));
  body.appendChild(picker.node);
  const wear = h("ax-btn", { label: "立即穿戴", theme: "prim" });
  wear.addEventListener("click", () => {
    if (!selected) return toast("请先选择一件时装/身份装", "warn");
    applyCommand({ action: "grant_item", itemConfig: selected.value, amount: 1, wearNow: true }, "已穿戴：" + selected.label);
  });
  const toBag = h("ax-btn", { label: "加入装备背包" });
  toBag.addEventListener("click", () => {
    if (!selected) return toast("请先选择一件时装/身份装", "warn");
    applyCommand({ action: "grant_item", itemConfig: selected.value, amount: 1, wearNow: false }, "已加入背包：" + selected.label);
  });
  body.appendChild(h("div", { class: "actions-row" }, [wear, toBag]));
  body.appendChild(note.node);
  host.appendChild(section);
}

function cardPicker({ recentKey, hint, hairOnly }) {
  let selected = null;
  const rowsCache = { loaded: false, rows: [] };
  const picker = catalogPicker({
    recentKey,
    placeholder: "搜索中文名或卡牌 ID，例如：屠龙刀鞘、1（拉取一次后本地过滤）",
    emptyText: "没有匹配的卡牌",
    fetchRows: async (query) => {
      if (!rowsCache.loaded) {
        rowsCache.rows = await catalog(hairOnly ? "/api/hair-cards" : "/api/cards");
        rowsCache.loaded = true;
      }
      const needle = query.toLowerCase();
      return rowsCache.rows
        .filter((row) => !needle || row.name.toLowerCase().includes(needle) || String(row.id) === needle)
        .map((row) => ({ label: row.name, sub: "ID " + row.id + " · " + row.category + " · " + row.config_id, value: row.id }));
    },
    onPick(row) {
      selected = row;
    },
  });
  return { picker, getSelected: () => selected };
}

function renderCards(host) {
  const { section, body } = panelSection({
    id: "sec-cards",
    title: "风物志卡牌",
    desc: "基于 CardRec（收集记录）与 UseCardRec（已装备卡牌）。点击卡牌名选中后「装备」即可上身（优先显示卡牌模型）。",
  });
  const card = cardPicker({ recentKey: "cards" });
  const equip = h("ax-btn", { label: "装备选中卡牌", theme: "prim" });
  equip.addEventListener("click", () => {
    const selected = card.getSelected();
    if (!selected) return toast("请先选择一张卡牌", "warn");
    applyCommand({ action: "equip_card", cardId: selected.value }, "已装备：" + selected.label);
  });
  body.appendChild(card.picker.node);
  body.appendChild(
    h("div", { class: "actions-row" }, [
      equip,
      cmdBtn({ label: "一键获取全部风物志", action: "grant_all_cards", section: "风物志卡牌" }),
      cmdBtn({ label: "卸下全部风物志", action: "unequip_card", section: "风物志卡牌" }),
    ])
  );
  host.appendChild(section);
}

function renderHair(host) {
  const { section, body } = panelSection({
    id: "sec-hair",
    title: "风物志头发",
    desc: "仅列出只含 Hat 部件的纯发型卡牌。装备发型会叠加在已有风物志之上，已装备的衣服/武器外形不会被替换。",
  });
  const hair = cardPicker({ recentKey: "hair", hairOnly: true });
  const equip = h("ax-btn", { label: "装备选中发型", theme: "prim" });
  equip.addEventListener("click", () => {
    const selected = hair.getSelected();
    if (!selected) return toast("请先选择一个发型卡牌", "warn");
    applyCommand({ action: "equip_hair", cardId: selected.value }, "已装备发型：" + selected.label);
  });
  body.appendChild(hair.picker.node);
  body.appendChild(
    h("div", { class: "actions-row" }, [
      equip,
      cmdBtn({ label: "一键获取全部风物志", action: "grant_all_cards", section: "风物志头发" }),
      cmdBtn({ label: "卸下全部风物志", action: "unequip_card", section: "风物志头发" }),
    ])
  );
  host.appendChild(section);
}

function renderMounts(host) {
  const { section, body } = panelSection({
    id: "sec-mounts",
    title: "坐骑管理",
    desc: "选择坐骑后点击「骑乘」即可直接激活并骑乘。已激活的坐骑再次使用不会消耗道具。",
  });
  let selected = null;
  const picker = catalogPicker({
    recentKey: "mounts",
    placeholder: "搜索坐骑名或 ConfigID（拉取一次后本地过滤）",
    emptyText: "没有匹配的坐骑",
    fetchRows: async (query) => {
      const rows = await catalog("/api/mounts");
      const needle = query.toLowerCase();
      return rows
        .filter((row) => !needle || row.name.toLowerCase().includes(needle) || row.configID.toLowerCase().includes(needle))
        .map((row) => ({ label: row.name, sub: row.configID + " · ArtPack=" + row.artPack, value: row.configID }));
    },
    onPick(row) {
      selected = row;
    },
  });
  const ride = h("ax-btn", { label: "骑乘", theme: "prim" });
  picker.refresh(); // 进板块即拉一次（本地过滤型目录）
  ride.addEventListener("click", () => {
    if (!selected) return toast("请先选择一个坐骑", "warn");
    applyCommand({ action: "mount_ride", mountConfig: selected.value }, "已骑乘：" + selected.label);
  });
  body.appendChild(picker.node);
  body.appendChild(h("div", { class: "actions-row" }, [ride]));
  host.appendChild(section);
}

function renderStances(host) {
  const { section, body } = panelSection({
    id: "sec-stances",
    title: "待机站姿（持久化）",
    desc: "切换玩家待机站姿并按角色持久化：重登、过图、受击/挖矿/技能结束后的回站都会保持所选站姿。动作名来自客户端 index.ini 节名。",
  });
  let selected = null;
  const note = hint();
  const presets = [
    ["stand", "默认站姿 [stand]"],
    ["khbd_stand_01", "葵花站姿 [khbd_stand_01]"],
    ["TJstand", "无根站姿 [TJstand]"],
    ["", "恢复当前站姿"],
  ];
  const picker = catalogPicker({
    recentKey: "stances",
    placeholder: "搜索站姿（skeleton / action，拉取一次后本地过滤）",
    emptyText: "没有匹配的站姿",
    fetchRows: async (query) => {
      const rows = await catalog("/api/idle-stances");
      const needle = query.toLowerCase();
      return rows
        .filter((row) => !needle || row.action.toLowerCase().includes(needle) || (row.skeleton || "").toLowerCase().includes(needle))
        .map((row) => ({
          label: row.skeleton ? row.skeleton : row.action,
          sub: "[" + row.action + "]",
          value: row.action,
        }));
    },
    onPick(row) {
      selected = row;
      note.set("已选中：" + row.label, true);
    },
  });
  body.appendChild(picker.node);
  const apply = h("ax-btn", { label: "切换到选中站姿", theme: "prim" });
  picker.refresh(); // 进板块即拉一次（本地过滤型目录）
  apply.addEventListener("click", () => {
    if (!selected) return toast("请先选择一个站姿", "warn");
    applyCommand({ action: "idle_stance_set", stanceAction: selected.value }, "站姿已切换");
  });
  const quickRow = h("div", { class: "actions-row" }, [apply]);
  for (const [action, label] of presets) {
    const btn = h("ax-btn", { label, type: "plain" });
    btn.addEventListener("click", () => applyCommand({ action: "idle_stance_set", stanceAction: action }, label));
    quickRow.appendChild(btn);
  }
  body.appendChild(quickRow);
  body.appendChild(note.node);
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = {
    "fashion-all": renderFashionAll,
    "fashion-grant": renderFashionGrant,
    cards: renderCards,
    hair: renderHair,
    mounts: renderMounts,
    stances: renderStances,
  };
  (renderers[sectionId] || renderFashionAll)(host);
}
