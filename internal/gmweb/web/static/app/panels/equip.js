// equip.js —— 组3「装备与铸造」：所有道具与换装 / 洗练 / 模板生成 / 冶炼 / 兵录。
// 命令参数与旧 console-features.js 逐字对齐；RECAST_SLOTS 槽位表原样搬自旧页。

import { h, panelSection, numField, cmdBtn, actionBtn, catalogPicker, hint, toast } from "../ui.js";
import { catalog, search, command } from "../api.js";
import { refreshNow } from "../state.js";

/* ---------------- 所有道具与换装 ---------------- */

function renderItems(host) {
  const { section, body } = panelSection({
    id: "sec-items",
    title: "所有道具与换装",
    desc: "目录合并 equipment.ini 与 tool_item.ini，中文名以 stringname.idres 为准（与游戏内显示同源）。ViewID 把道具分别放入普通、装备、材料、任务背包；「立即穿戴」数量固定 1。宝物洗炼入口在选中宝物后可用。",
  });
  let selected = null;
  const amount = numField({ label: "数量", value: 1, min: 1, max: 9999, width: "150px" });
  const viewSelect = h("ax-select", {
    content: JSON.stringify([
      { label: "全部分类", value: "0" },
      { label: "普通道具", value: "1" },
      { label: "装备", value: "2" },
      { label: "材料", value: "3" },
      { label: "任务物品", value: "4" },
    ]),
    style: "width:130px;",
  });
  const picker = catalogPicker({
    recentKey: "items-grant",
    placeholder: "输入中文名或 ConfigID，例如：久仙刀、药、礼包（服务端搜索分页，可翻完整目录）",
    emptyText: "没有找到匹配项",
    fetchRows: async (query, signal, page) => {
      const view = viewSelect.value || "0";
      // 带 offset/limit 走服务端分页信封 {rows,total}：目录 10 万+ 行，
      // 旧「一次截 300 条」把总量与翻页都做成了假的。
      const data = await search(
        "/api/items?q=" + encodeURIComponent(query) + "&view=" + view + "&offset=" + (page.offset || 0) + "&limit=" + (page.limit || 50),
        signal
      );
      return {
        rows: (data.rows || []).map((row) => ({
          label: row.name || row.configId,
          sub: row.configId + " · " + ["普通", "装备", "材料", "任务"][row.viewId - 1] + (row.wearable ? " · 可穿戴" : ""),
          value: row.configId,
        })),
        total: data.total || 0,
      };
    },
    onPick(row) {
      selected = row;
    },
  });
  // ax-select 只派发原生 change（CustomEvent）与 input；「changed」仅存在于
  // AXUI 的 .on()/on-changed 订阅通道，addEventListener("changed") 永远不触发。
  viewSelect.addEventListener("change", () => picker.refresh());
  const grant = actionBtn({
    label: "加入对应背包",
    build: () => (selected ? { action: "grant_item", itemConfig: selected.value, amount: amount.get() ?? 1 } : (toast("请先搜索并选择道具", "warn"), null)),
    okText: () => "已加入背包：" + selected.label,
    section: "所有道具与换装",
  });
  const wear = actionBtn({
    label: "立即穿戴",
    build: () => (selected ? { action: "grant_item", itemConfig: selected.value, amount: 1, wearNow: true } : (toast("请先搜索并选择道具", "warn"), null)),
    okText: () => "已穿戴：" + selected.label,
    section: "所有道具与换装",
  });
  body.appendChild(picker.node);
  body.appendChild(h("div", { class: "actions-row", style: "margin-top:10px;" }, [viewSelect, amount.node, grant, wear]));
  body.appendChild(
    h("p", { class: "hint" }, ["宝物洗炼（发放宝物后开窗）：火浣台宝物走老界面、TersureType 0..2 走新界面。"])
  );
  const washOld = actionBtn({
    label: "打开洗炼·老界面（属性/武学）",
    theme: "",
    build: () => (selected ? { action: "tersure_wash_open", itemConfig: "old" } : (toast("请先选择一件宝物", "warn"), null)),
    section: "所有道具与换装",
  });
  const washNew = actionBtn({
    label: "打开洗炼·新界面（主武学）",
    theme: "",
    build: () => (selected ? { action: "tersure_wash_open", itemConfig: "new" } : (toast("请先选择一件宝物", "warn"), null)),
    section: "所有道具与换装",
  });
  body.appendChild(h("div", { class: "actions-row" }, [washOld, washNew]));
  host.appendChild(section);
}

/* ---------------- 洗练 ---------------- */

// 槽位表原样搬自旧 console-features.js RECAST_SLOTS（身上装备槽 22..48）。
const RECAST_SLOTS = [
  ["武器", 22, "武器"], ["防具", 1, "头发"], ["防具", 3, "衣服"], ["防具", 6, "护腕"],
  ["防具", 4, "裤子"], ["防具", 7, "绑腿"], ["防具", 8, "鞋子"], ["首饰", 11, "耳环1"],
  ["首饰", 12, "耳环2"], ["首饰", 13, "项链"], ["首饰", 14, "戒指1"], ["首饰", 15, "戒指2"],
  ["宝物", 25, "宝物1"], ["宝物", 26, "宝物2"], ["宝物", 27, "宝物3"], ["宝物", 28, "宝物4"],
  ["宝物", 29, "宝物5"], ["娃娃", 44, "娃娃1"], ["娃娃", 45, "娃娃2"], ["娃娃", 46, "娃娃3"],
  ["娃娃", 47, "娃娃4"], ["娃娃", 48, "娃娃5"],
];

function renderRecast(host) {
  const { section, body } = panelSection({
    id: "sec-recast",
    title: "洗练（身上装备 · 指定部位）",
    desc: "先选部位再点功能：随机=词条属性+数值全随机重掷；定向=保持词条属性、同属性族内重掷数值；抛光=把每条数值拉到属性族最高档；招式开光=重掷已学招式词条。定向/抛光需要该部位已有词条。",
  });
  const slotSelect = h("ax-select", {
    content: JSON.stringify(RECAST_SLOTS.map((s) => ({ label: "[" + s[0] + "] " + s[2] + " (槽" + s[1] + ")", value: String(s[1]) }))),
    search: true,
    style: "min-width:190px;",
    value: "22",
  });
  const note = hint();
  const currentSlot = () => Number(slotSelect.value || 0);
  const recastBtn = (label, action, theme) =>
    actionBtn({
      label,
      theme: theme || "",
      build: () => (currentSlot() ? { action, slot: currentSlot() } : (toast("请先选择目标部位", "warn"), null)),
      okText: label + " 已提交",
      section: "洗练",
    });
  body.appendChild(h("div", { class: "field-row" }, [h("label", {}, ["目标部位"]), slotSelect]));
  body.appendChild(
    h("div", { class: "actions-row" }, [
      recastBtn("随机洗练", "recast_slot_prop", "prim"),
      recastBtn("定向洗练", "recast_slot_dire", ""),
      recastBtn("抛光", "recast_slot_polish", ""),
      recastBtn("招式开光", "recast_slot_skill", ""),
    ])
  );
  body.appendChild(note.node);

  // —— 自定义词条（折叠区：拉目录 + 复选 + 过滤 + 批量加入/清空） ——
  const propBox = h("div", { class: "affix-list" });
  const skillBox = h("div", { class: "affix-list" });
  const filterInput = h("input", { type: "text", placeholder: "输入 名称/ID 筛选…" });
  filterInput.style.cssText = "width:220px;background:#11141b;border:1px solid var(--gm-line);border-radius:6px;color:var(--gm-text);padding:5px 10px;";
  const affixHint = hint();
  let loaded = false;
  const listWrap = h("details", {}, [
    h("summary", { style: "cursor:pointer;color:var(--gm-gold);font-size:13px;margin:12px 0 6px;" }, ["全部词条列表（勾选后批量加入当前部位）"]),
  ]);
  const listBody = h("div", { style: "display:flex;gap:24px;flex-wrap:wrap;" }, [
    h("div", { style: "flex:1;min-width:300px;" }, [h("div", { class: "matrix-label" }, ["普通词条（PropBuilt）"]), propBox]),
    h("div", { style: "flex:1;min-width:300px;" }, [h("div", { class: "matrix-label" }, ["招式词条（SkillPack）"]), skillBox]),
  ]);
  listWrap.appendChild(filterInput);
  listWrap.appendChild(listBody);
  listWrap.appendChild(affixHint.node);
  listWrap.addEventListener("toggle", () => {
    if (listWrap.open && !loaded) {
      loaded = true;
      catalog("/api/recast-affixes")
        .then((data) => {
          if (data.loaded === false) {
            affixHint.set("洗练资源未加载（缺少 senior_weapon_*.ini），词条列表为空", false);
            return;
          }
          for (const entry of data.prop || []) {
            propBox.appendChild(affixRow(entry, "prop"));
          }
          for (const entry of data.skill || []) {
            skillBox.appendChild(affixRow(entry, "skill"));
          }
          affixHint.set("词条目录已载入（普通 " + (data.prop || []).length + " / 招式 " + (data.skill || []).length + "）", true);
          applyFilter();
        })
        .catch((err) => affixHint.set(err.message || String(err), false));
    }
  });
  function affixRow(entry, kind) {
    const box = h("input", { type: "checkbox", value: String(entry.id), "data-text": (entry.name + " " + (entry.source || "")).toLowerCase() });
    return h("label", { class: "picker-row", style: "gap:8px;" }, [box, h("span", {}, [entry.name]), entry.source ? h("span", { class: "sub" }, [entry.source]) : null]);
  }
  function applyFilter() {
    const needle = filterInput.value.trim().toLowerCase();
    for (const input of listBody.querySelectorAll("input[type=checkbox]")) {
      const row = input.parentElement;
      row.style.display = !needle || input.dataset.text.includes(needle) ? "" : "none";
    }
  }
  filterInput.addEventListener("input", applyFilter);
  const addChecked = (kind) =>
    actionBtn({
      label: kind === "prop" ? "加入词条（普通）" : "加入词条（招式）",
      theme: "prim",
      build: () => {
        const ids = Array.from((kind === "prop" ? propBox : skillBox).querySelectorAll("input:checked")).map((c) => Number(c.value));
        if (!ids.length) return (toast("请先勾选要加入的词条", "warn"), null);
        if (!currentSlot()) return (toast("请先选择目标部位", "warn"), null);
        return { action: "recast_add_affix", slot: currentSlot(), packIDs: ids, affixKind: kind };
      },
      okText: (payload) => "已加入 " + payload.packIDs.length + " 条词条",
      section: "洗练",
    });
  const clearAffix = actionBtn({
    label: "清空装备词条",
    danger: true,
    confirm: "确定清空该装备的全部词条（普通+招式）？此操作不可恢复。",
    build: () => (currentSlot() ? { action: "recast_clear_affix", slot: currentSlot() } : (toast("请先选择目标部位", "warn"), null)),
    okText: "词条已清空",
    section: "洗练",
  });
  body.appendChild(listWrap);
  body.appendChild(h("div", { class: "actions-row" }, [addChecked("prop"), addChecked("skill"), clearAffix]));
  host.appendChild(section);
}

/* ---------------- 模板生成 ---------------- */

function renderTemplate(host) {
  const { section, body } = panelSection({
    id: "sec-template",
    title: "模板生成（带词条装备）",
    desc: "按官方生活技能模板生成并立即穿戴（模板优先取 51 级金色 60501 家族基座，词条随装备持久化）：厨师=6 条属性词条；铁匠=3 属性 + 3 招式；护腕=3 阳刚臂力 + 3 招式（按已穿武器类型）；防具 8 部位=固定 5 条红词条。武器可指定词条计划（下方三个计数任一 >0 生效，覆盖职业随机）：伤害=外功伤害递增档、破击=破防能力递增档、技能=招式威力（skillpack）；档位默认每类取最高 N 档，可切池内随机。技能词条可按套路定向：勾选具体招式=每招精确 1 条（列表标注实招/虚招/架招/怒招），只选套路=同套路威力招式随机。",
  });
  const weaponSelect = h("ax-select", {
    content: JSON.stringify([
      { label: "单刀", value: "101" }, { label: "单剑", value: "102" }, { label: "匕首", value: "103" },
      { label: "双刀", value: "104" }, { label: "双剑", value: "105" }, { label: "双刺", value: "106" },
      { label: "长棍", value: "107" }, { label: "短棍", value: "108" }, { label: "重剑", value: "122" },
    ]),
    style: "width:150px;",
    value: "102",
  });
  const crafterSelect = h("ax-select", {
    content: JSON.stringify([
      { label: "厨师（6 属性词条）", value: "cook" },
      { label: "铁匠（3 属性 + 3 招式）", value: "smith" },
    ]),
    style: "width:190px;",
  });
  const armorSelect = h("ax-select", {
    content: JSON.stringify([
      { label: "帽子", value: "helmet" }, { label: "项链", value: "amulet" }, { label: "衣服", value: "cloth" },
      { label: "裤子", value: "pants" }, { label: "绑腿", value: "leg" }, { label: "鞋子", value: "shoes" },
      { label: "耳环", value: "ear" }, { label: "戒指", value: "ring" },
    ]),
    style: "width:150px;",
  });
  // 指定词条计划：三类计数（0=不指定）+ 档位开关；全 0 时按职业随机。
  const damageCount = numField({ label: "伤害词条", value: 0, min: 0, max: 6, width: "110px" });
  const pierceCount = numField({ label: "破击词条", value: 0, min: 0, max: 6, width: "110px" });
  const skillCount = numField({ label: "技能词条", value: 0, min: 0, max: 6, width: "110px" });
  const tierSelect = h("ax-select", {
    content: JSON.stringify([
      { label: "最高档", value: "max" },
      { label: "随机档", value: "rand" },
    ]),
    style: "width:110px;",
    value: "max",
  });

  // —— 技能词条按套路定向（折叠区：套路下拉 + 招式勾选，类型标注） ——
  // 勾选招式 = 每招精确 1 条威力词条；只选套路不勾 = 该套路威力招式中
  // 随机「技能词条」计数条（不放回，不同招式）。
  const taoluSkillBox = h("div", { class: "affix-list" });
  const taoluHint = hint();
  const taoluSelect = h("ax-select", { style: "min-width:190px;", search: true });
  let taoluData = [];
  let taoluLoaded = false;
  function fillTaoluSelect() {
    const options = [{ label: "不按套路（全池随机）", value: "" }];
    for (const t of taoluData) options.push({ label: t.label + "（" + t.skills.length + " 招）", value: t.key });
    taoluSelect.setAttribute("content", JSON.stringify(options));
    taoluSelect.setAttribute("value", "");
  }
  function renderTaoluSkills(key) {
    taoluSkillBox.textContent = "";
    const taolu = taoluData.find((t) => t.key === key);
    if (!taolu) return;
    for (const skill of taolu.skills) {
      taoluSkillBox.appendChild(
        h("label", { class: "picker-row", style: "gap:8px;" }, [
          h("input", { type: "checkbox", value: skill.key }),
          h("span", {}, [skill.name]),
          h("span", { class: "sub" }, ["[" + skill.type + "]"]),
        ])
      );
    }
  }
  taoluSelect.addEventListener("change", () => renderTaoluSkills(taoluSelect.value || ""));
  const taoluWrap = h("details", {}, [
    h("summary", { style: "cursor:pointer;color:var(--gm-gold);font-size:13px;margin:12px 0 6px;" }, ["技能词条按套路选择（勾选=精确指定招式；只选套路=同套路威力招式随机）"]),
  ]);
  taoluWrap.appendChild(h("div", { class: "field-row" }, [h("label", {}, ["套路"]), taoluSelect]));
  taoluWrap.appendChild(taoluSkillBox);
  taoluWrap.appendChild(taoluHint.node);
  taoluWrap.addEventListener("toggle", () => {
    if (taoluWrap.open && !taoluLoaded) {
      taoluLoaded = true;
      catalog("/api/skill-affixes")
        .then((data) => {
          if (data.loaded === false) {
            taoluHint.set("招式威力词条目录未加载（缺少 skillpack/技能静态表）", false);
            return;
          }
          taoluData = data.taolus || [];
          fillTaoluSelect();
          taoluHint.set("套路目录已载入（" + taoluData.length + " 个套路有威力词条招式）", true);
        })
        .catch((err) => taoluHint.set(err.message || String(err), false));
    }
  });

  const genWeapon = actionBtn({
    label: "生成武器",
    build: () => {
      const body = {
        action: "gm_generate_weapon",
        weaponType: Number(weaponSelect.value || 102),
        crafter: crafterSelect.value || "cook",
        damageAffixes: damageCount.get() ?? 0,
        pierceAffixes: pierceCount.get() ?? 0,
        skillAffixes: skillCount.get() ?? 0,
        maxTier: (tierSelect.value || "max") === "max",
      };
      const taolu = taoluSelect.value || "";
      if (taolu) {
        body.skillTaolu = taolu;
        const checked = Array.from(taoluSkillBox.querySelectorAll("input:checked")).map((c) => c.value);
        if (checked.length) body.skillKeys = checked;
      }
      return body;
    },
    okText: "武器生成已提交",
    section: "模板生成",
  });
  const genArmor = actionBtn({
    label: "生成防具",
    build: () => ({ action: "gm_generate_armor", armorSlot: armorSelect.value || "helmet" }),
    okText: "防具生成已提交",
    section: "模板生成",
  });
  const genBracer = actionBtn({
    label: "生成带技能词条的护腕",
    build: () => ({ action: "gm_generate_bracer" }),
    okText: "护腕生成已提交",
    section: "模板生成",
  });
  body.appendChild(h("div", { class: "field-row" }, [h("label", {}, ["武器类型"]), weaponSelect, h("label", {}, ["职业"]), crafterSelect, genWeapon]));
  body.appendChild(h("div", { class: "field-row" }, [damageCount.node, pierceCount.node, skillCount.node, tierSelect]));
  body.appendChild(taoluWrap);
  body.appendChild(h("div", { class: "field-row" }, [h("label", {}, ["防具部位"]), armorSelect, genArmor, genBracer]));
  host.appendChild(section);
}

/* ---------------- 冶炼（最小登录模式） ---------------- */

function renderSmelt(host) {
  const { section, body } = panelSection({
    id: "sec-smelt",
    title: "冶炼（已对齐 D:\\ 参考端）",
    desc: "打开冶炼只发 466 open_id=2 + 570 util_show_form，全链路零 413/416/gamestep/825 应答。仍报「当前活动未开启」时：(1) 先完整重启 FxGame.exe 清原生 SwitchManager 缓存；(2) 切下方「最小登录模式」再重登一次，若冶炼恢复即说明 H 某条登录推送污染了客户端原生 SwitchManager 表。",
  });
  const state = h("span", { class: "hint" }, ["状态读取中……"]);
  async function loadState() {
    try {
      const response = await fetch("/api/minimal-login");
      const data = await response.json();
      state.textContent = data.enabled
        ? "当前：开启（下次重登录生效，H-only 推送全部跳过，与 D:\\ 字节级对齐）"
        : "当前：关闭（默认，H 全量登录推送）";
    } catch (err) {
      state.textContent = "读取失败：" + (err.message || err);
    }
  }
  // minimal-login 是面板自身端点（POST /api/minimal-login），不走命令通道。
  const toggleBtn = h("ax-btn", { label: "切换最小登录模式", theme: "prim" });
  toggleBtn.addEventListener("click", async () => {
    toggleBtn.setAttribute("disabled", "");
    try {
      await fetch("/api/minimal-login", { method: "POST" });
      toast("✔ 最小登录模式已切换", "ok");
    } catch (err) {
      toast("✘ 切换失败：" + (err.message || err), "err");
    }
    toggleBtn.removeAttribute("disabled");
    loadState();
  });
  body.appendChild(h("div", { class: "actions-row" }, [toggleBtn, state]));
  loadState();
  host.appendChild(section);
}

/* ---------------- 兵录 ---------------- */

function renderBinglu(host) {
  const { section, body } = panelSection({
    id: "sec-binglu",
    title: "兵录设置",
    desc: "直接修改角色兵录（百晓生兵录集）：等级=伤害提高、进度=x/y、上限。修改即时落盘并实时推送到客户端面板。",
  });
  const rowSelect = h("ax-select", {
    content: JSON.stringify([
      { label: "徒手", value: "0" }, { label: "单刀", value: "1" }, { label: "单剑", value: "2" },
      { label: "匕首", value: "3" }, { label: "双刀", value: "4" }, { label: "双剑", value: "5" },
      { label: "双刺", value: "6" }, { label: "长棍", value: "7" }, { label: "短棍", value: "8" },
      { label: "暗器", value: "10" }, { label: "奇门", value: "11" },
    ]),
    style: "width:150px;",
    value: "1",
  });
  const cur = numField({ label: "等级", value: 9, min: 0, width: "150px" });
  const fill = numField({ label: "进度", value: 0, min: 0, width: "150px" });
  const max = numField({ label: "上限（空=不改）", value: "", min: 0, width: "150px" });
  const set = actionBtn({
    label: "设置",
    build: () => {
      const parts = ["set", rowSelect.value || "1", cur.get() ?? 0, fill.get() ?? 0];
      if (max.get() !== null) parts.push(max.get());
      return { action: "binglu", bingluArgs: parts.join(" ") };
    },
    okText: "兵录已设置",
    section: "兵录",
  });
  const maxRow = actionBtn({
    label: "单行升满",
    build: () => ({ action: "binglu", bingluArgs: "max " + (rowSelect.value || "1") }),
    okText: "单行已升满",
    section: "兵录",
  });
  const maxAll = actionBtn({
    label: "全部升满",
    build: () => ({ action: "binglu", bingluArgs: "max" }),
    okText: "兵录已全部升满",
    section: "兵录",
  });
  body.appendChild(h("div", { class: "field-row" }, [h("label", {}, ["兵器"]), rowSelect, cur.node, fill.node, max.node, set, maxRow, maxAll]));
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = { items: renderItems, recast: renderRecast, template: renderTemplate, smelt: renderSmelt, binglu: renderBinglu };
  (renderers[sectionId] || renderItems)(host);
}
