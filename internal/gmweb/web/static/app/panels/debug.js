// debug.js —— 组11「调试工具」：S2C 消息探针 / 聊天气泡 / 动作剧情测试。

import { h, panelSection, numField, textField, actionBtn, cmdBtn, hint } from "../ui.js";

/* ---------------- S2C 自定义消息探针 ---------------- */

const PROBE_PRESETS = [
  [517, null, "517 精炼"], [517, 899000001, "517 重制"], [517, 899000008, "517 洗炼"],
  [526, null, "526 兑换"], [514, null, "514"], [515, null, "515"], [516, null, "516"],
  [620, null, "620"], [621, null, "621"], [736, null, "736"], [737, null, "737"],
  [764, null, "764"], [688, null, "688 打造台"], [419, null, "419 九阴志步数"],
  [413, null, "413 开关开启"], [417, null, "417 九阴志高亮"], [418, null, "418 地图图标"],
  [416, 2, "416 空事件"],
];

function renderProbe(host) {
  const { section, body } = panelSection({
    id: "sec-probe",
    title: "S2C 自定义消息探针",
    desc: "向当前角色发送一条数字型 S2C custom 消息（客户端槽位 = 2×ID+15）。subType 默认 899000000（精炼窗）。发送后观察客户端是否弹出对应窗口或报错。",
  });
  const msgId = numField({ label: "msgId", value: 517, min: 1, max: 764, width: "80px" });
  const subType = numField({ label: "subType", value: 899000000, width: "130px" });
  const note = hint();
  const customSend = h("ax-btn", { label: "发送探针", theme: "prim" });
  customSend.addEventListener("click", () => {
    const payload = { action: "custom_probe", msgId: msgId.get() ?? 517, subType: subType.get() ?? 899000000 };
    note.set("已发送 msgId=" + payload.msgId + " subType=" + payload.subType + "，观察客户端是否弹出对应窗口或报错", true);
    import("../api.js").then(({ command }) => command(payload));
  });
  body.appendChild(h("div", { class: "field-row" }, [msgId.node, subType.node, customSend]));
  body.appendChild(note.node);
  const matrix = h("div", { class: "btn-matrix", style: "margin-top:8px;" });
  const items = h("div", { class: "matrix-items" }, []);
  for (const [id, sub, label] of PROBE_PRESETS) {
    const btn = h("ax-btn", { label, type: "plain" });
    btn.addEventListener("click", () => {
      const payload = { action: "custom_probe", msgId: id, subType: sub ?? 899000000 };
      note.set("已发送 msgId=" + payload.msgId + " subType=" + payload.subType, true);
      import("../api.js").then(({ command }) => command(payload));
    });
    items.appendChild(btn);
  }
  matrix.appendChild(h("div", { class: "matrix-group" }, [h("div", { class: "matrix-label" }, ["常用 ID"]), items]));
  body.appendChild(matrix);
  host.appendChild(section);
}

/* ---------------- 聊天与气泡调试 ---------------- */

function renderChat(host) {
  const { section, body } = panelSection({
    id: "sec-chat",
    title: "聊天与气泡调试",
    desc: "聊天 C2S(1 = CLIENT_CUSTOMMSG_CHAT) 已接入：游戏内聊天框发送会回显为头顶气泡 + 面板文字。下方按钮直接走 0x1A 广播链路，验证气泡/面板显示。",
  });
  const text = h("input", { type: "text", placeholder: "输入要广播的聊天内容，例如：大家好" });
  text.style.cssText = "width:320px;background:#11141b;border:1px solid var(--gm-line);border-radius:6px;color:var(--gm-text);padding:6px 10px;";
  const broadcast = h("ax-btn", { label: "聊天广播(真实链路)", theme: "prim" });
  broadcast.addEventListener("click", () => {
    const value = text.value.trim();
    if (!value) return;
    import("../api.js").then(({ command }) => command({ action: "chat_broadcast", speechText: value }));
  });
  const lenBtn = h("ax-btn", { label: "气泡(长度前缀)", type: "plain" });
  lenBtn.addEventListener("click", () => {
    const value = text.value.trim();
    if (value) import("../api.js").then(({ command }) => command({ action: "speech_len", speechText: value }));
  });
  const nulBtn = h("ax-btn", { label: "气泡(空终止)", type: "plain" });
  nulBtn.addEventListener("click", () => {
    const value = text.value.trim();
    if (value) import("../api.js").then(({ command }) => command({ action: "speech_nul", speechText: value }));
  });
  body.appendChild(h("div", { class: "field-row" }, [text, broadcast, lenBtn, nulBtn]));
  host.appendChild(section);
}

/* ---------------- 动作与剧情测试 ---------------- */

function renderStorytest(host) {
  const { section, body } = panelSection({
    id: "sec-storytest",
    title: "动作与剧情测试",
    desc: "红霸体 20 秒 / 神鬼莫敌等动作状态与剧情提示帧；「恢复待机」结束测试态。",
  });
  const stories = [
    ["red_super_armor", "红霸体 20 秒"],
    ["action_state_sgmd", "神鬼莫敌动作"],
    ["action_state_hsqs05", "第五招动作"],
    ["action_state_stop", "恢复待机"],
    ["npc_bubble", "NPC 气泡"],
    ["drama_prompt", "章节提示"],
  ];
  const items = h("div", { class: "matrix-items" }, []);
  for (const [action, label] of stories) {
    items.appendChild(cmdBtn({ label, theme: "", action, section: "动作剧情" }));
  }
  body.appendChild(h("div", { class: "btn-matrix" }, [h("div", { class: "matrix-group" }, [items])]));
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = { probe: renderProbe, chat: renderChat, storytest: renderStorytest };
  (renderers[sectionId] || renderProbe)(host);
}
