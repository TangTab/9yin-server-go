// item.js —— 组2「货币与物品」：货币四项 / 礼包与物品 / 世界宝箱。
// 货币区保持旧版快捷片（怒气 +10/+50/回满）与界值；礼包/物品走 M1 服务端搜索。

import { h, panelSection, numField, cmdBtn, catalogPicker, hint, toast } from "../ui.js";
import { catalog, command, search } from "../api.js";
import { refreshNow } from "../state.js";

function renderCurrency(host) {
  const { section, body } = panelSection({
    id: "sec-currency",
    title: "碎银 / 官银 / 黄金 / 怒气",
    desc: "四项均为本会话数值（重登重置；官银/黄金的 vip.json 级持久化待接）：碎银=CapitalType1、官银=CapitalType2、黄金=CapitalType0、怒气=SP（0–100，设置后立即下发属性帧）。",
  });
  const silver = numField({ label: "碎银", value: 99999, min: 0, width: "150px" });
  const guanYin = numField({ label: "官银", value: 99999, min: 0, width: "150px" });
  const gold = numField({ label: "黄金", value: 9999, min: 0, width: "150px" });
  const sp = numField({ label: "怒气", value: 100, min: 0, max: 100, width: "150px" });
  const rage = numField({ label: "自动回复/秒", value: 0, min: 0, max: 1000, width: "150px" });
  body.appendChild(h("div", { class: "field-row" }, [silver.node, guanYin.node, gold.node, sp.node]));

  const setSilver = h("ax-btn", { label: "设置碎银" });
  setSilver.addEventListener("click", () => applyCommand({ action: "set_silver", silver: silver.get() ?? 0 }, "碎银已设置"));
  const setGuanYin = h("ax-btn", { label: "设置官银" });
  setGuanYin.addEventListener("click", () => applyCommand({ action: "set_guanyin", guanYin: guanYin.get() ?? 0 }, "官银已设置"));
  const setGold = h("ax-btn", { label: "设置黄金" });
  setGold.addEventListener("click", () => applyCommand({ action: "set_gold", gold: gold.get() ?? 0 }, "黄金已设置"));
  const setSP = h("ax-btn", { label: "设置怒气" });
  setSP.addEventListener("click", () => applyCommand({ action: "set_sp", sp: sp.get() ?? 0 }, "怒气已设置"));
  body.appendChild(h("div", { class: "actions-row" }, [setSilver, setGuanYin, setGold, setSP]));

  const applyRage = h("ax-btn", { label: "应用自动回复", type: "plain" });
  applyRage.addEventListener("click", () => applyCommand({ action: "set_rage_recovery", value: rage.get() ?? 0 }, "怒气自动回复已应用"));
  body.appendChild(h("div", { class: "field-row" }, [rage.node, applyRage]));
  body.appendChild(
    h("p", { class: "hint" }, ["自动回复 0=关闭；官方满档 5；会话级，重登归零。"])
  );
  const sp10 = h("ax-btn", { label: "怒气 +10", type: "plain" });
  sp10.addEventListener("click", () => applyCommand({ action: "add_sp", sp: 10 }, "怒气 +10"));
  const sp50 = h("ax-btn", { label: "怒气 +50", type: "plain" });
  sp50.addEventListener("click", () => applyCommand({ action: "add_sp", sp: 50 }, "怒气 +50"));
  const spFull = h("ax-btn", { label: "回满怒气", type: "plain" });
  spFull.addEventListener("click", () => applyCommand({ action: "fill_sp" }, "怒气已回满"));
  body.appendChild(h("div", { class: "actions-row" }, [sp10, sp50, spFull]));
  host.appendChild(section);
}

async function applyCommand(payload, okText) {
  const result = await command(payload);
  toast(result.ok ? "✔ " + okText : "✘ " + result.message, result.ok ? "ok" : "err");
  if (result.ok) refreshNow();
  return result;
}

function renderGifts(host) {
  const { section, body } = panelSection({
    id: "sec-gifts",
    title: "礼包与物品",
    desc: "礼包列表来自 tool_item.ini（ItemType 2102/2103 开箱类）。标「有内容」的打开会出真物品，其余提示内容未配置。下方搜索直连服务端道具目录（上限 300），绝不拉全量。",
  });

  // —— 礼包（小目录，一次拉全量前端过滤） ——
  let giftSelected = null;
  const giftAmount = numField({ label: "数量", value: 1, min: 1, max: 99, width: "150px" });
  const giftPicker = catalogPicker({
    recentKey: "giftboxes",
    placeholder: "搜索礼包名字或 ID（拉取一次后本地过滤）",
    emptyText: "没有匹配的礼包",
    fetchRows: async (query) => {
      const rows = await catalog("/api/giftboxes");
      const needle = query.toLowerCase();
      return rows
        .filter((row) => !needle || row.name.toLowerCase().includes(needle) || row.config.toLowerCase().includes(needle))
        .map((row) => ({
          label: row.name,
          sub: row.config + (row.hasContent ? " · 有内容" : " · 未配置内容"),
          value: row.config,
          hasContent: row.hasContent,
        }));
    },
    onPick(row) {
      giftSelected = row;
    },
  });
  const addGift = h("ax-btn", { label: "礼包加入背包", theme: "prim" });
  addGift.addEventListener("click", () => {
    if (!giftSelected) return toast("请先选择一个礼包", "warn");
    applyCommand(
      { action: "add_item", itemConfig: giftSelected.value, amount: giftAmount.get() ?? 1 },
      "礼包已加入背包 ×" + (giftAmount.get() ?? 1)
    );
  });
  body.appendChild(h("h3", { style: "margin:0 0 8px;font-size:14px;color:var(--gm-text);" }, ["礼包"]));
  body.appendChild(giftPicker.node);
  body.appendChild(h("div", { class: "actions-row" }, [giftAmount.node, addGift]));

  body.appendChild(h("div", { style: "height:14px;" }));
  body.appendChild(h("h3", { style: "margin:0 0 8px;font-size:14px;color:var(--gm-text);" }, ["任意物品（服务端搜索）"]));

  // —— 任意物品（服务端搜索直连） ——
  let itemSelected = null;
  const itemAmount = numField({ label: "数量", value: 1, min: 1, max: 9999, width: "150px" });
  const itemPicker = catalogPicker({
    recentKey: "items",
    placeholder: "输入中文名或 ConfigID，例如：久仙刀、药、礼包（服务端搜索）",
    emptyText: "没有找到匹配项",
    fetchRows: async (query, signal) => {
      const rows = await search("/api/items?q=" + encodeURIComponent(query), signal);
      return rows.map((row) => ({
        label: row.name || row.configId,
        sub: row.configId + " · " + ["普通", "装备", "材料", "任务"][row.viewId - 1] + (row.wearable ? " · 可穿戴" : ""),
        value: row.configId,
      }));
    },
    onPick(row) {
      itemSelected = row;
    },
  });
  const addItem = h("ax-btn", { label: "物品加入背包", theme: "prim" });
  addItem.addEventListener("click", () => {
    if (!itemSelected) return toast("请先搜索并选择物品", "warn");
    applyCommand(
      { action: "add_item", itemConfig: itemSelected.value, amount: itemAmount.get() ?? 1 },
      "物品已加入背包 ×" + (itemAmount.get() ?? 1)
    );
  });
  const sendMail = h("ax-btn", { label: "作为系统邮件附件发送", type: "plain" });
  sendMail.addEventListener("click", () => {
    if (!itemSelected) return toast("请先搜索并选择物品", "warn");
    applyCommand(
      { action: "send_test_mail", itemConfig: itemSelected.value, amount: itemAmount.get() ?? 1 },
      "测试邮件已发送"
    );
  });
  body.appendChild(itemPicker.node);
  body.appendChild(h("div", { class: "actions-row" }, [itemAmount.node, addItem, sendMail]));
  host.appendChild(section);
}

function renderChest(host) {
  const { section, body } = panelSection({
    id: "sec-chest",
    title: "世界宝箱",
    desc: "在角色当前位置生成一个「镀银宝箱」（BoxNpc10006，洛阳同款）：点击开箱 → 蹲下拾取动作 + 开盖动画 + 拾取框（随机装备/礼包/带词条宝物），拾取完宝箱自动消失。可反复生成。",
  });
  body.appendChild(
    h("div", { class: "actions-row" }, [
      cmdBtn({ label: "生成镀银宝箱", theme: "prim", action: "spawn_chest", section: "世界宝箱" }),
      cmdBtn({ label: "把礼包测试商人拉到身边", action: "giftbox_merchant_here", section: "世界宝箱" }),
    ])
  );
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = { currency: renderCurrency, gifts: renderGifts, chest: renderChest };
  (renderers[sectionId] || renderCurrency)(host);
}
