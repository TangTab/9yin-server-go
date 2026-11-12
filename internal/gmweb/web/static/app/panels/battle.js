// battle.js —— 组7「战斗与演练」：训练木桩 / 战斗假人 / 伙伴系统 / 御风神水。

import { h, panelSection, numField, textField, cmdBtn, actionBtn, hint, toast } from "../ui.js";
import { catalog, command } from "../api.js";
import { subscribe, refreshNow } from "../state.js";

/* ---------------- 训练木桩 ---------------- */

function renderDummy(host) {
  const { section, body } = panelSection({
    id: "sec-dummy",
    title: "训练木桩",
    desc: "在角色当前位置生成一只纯沙包怪物：移动速度 0、不会还手、血量自定义，打空即死（可反复生成）。测输出/测技能专用。",
  });
  const hp = numField({ label: "血量", value: 100000, min: 1, width: "120px" });
  body.appendChild(
    h("div", { class: "field-row" }, [
      hp.node,
      actionBtn({
        label: "生成木桩",
        build: () => ({ action: "spawn_dummy", dummyHp: hp.get() ?? 0 }),
        okText: "木桩已生成",
        section: "训练木桩",
      }),
      actionBtn({
        label: "生成 100 万血木桩",
        build: () => ({ action: "spawn_dummy", dummyHp: 1000000 }),
        okText: "百万血木桩已生成",
        section: "训练木桩",
      }),
    ])
  );
  host.appendChild(section);
}

/* ---------------- 战斗假人 ---------------- */

function renderCombatDummy(host) {
  const { section, body } = panelSection({
    id: "sec-combat-dummy",
    title: "添加假人（战斗）",
    desc: "在角色面前生成一只可战斗假人（血量默认 5 万、名字可自定义）：选中出招命中后假人周期还手（近身挥击、恒 1 点实伤），打空血 15 秒后满血重生。招式用顿号/逗号分隔。",
  });
  const name = textField({ label: "名字", value: "战斗假人", width: "140px" });
  const hp = numField({ label: "血量", value: 50000, min: 1, width: "150px" });
  const damage = numField({ label: "单段伤害(0=按技能表)", value: 0, min: 0, width: "150px" });
  const preset = h("ax-select", { style: "min-width:170px;", placeholder: "选择套路预设" });
  const skills = h("textarea", { rows: 3, placeholder: "招式（顿号/逗号分隔，可自行增删）" });
  skills.style.cssText = "width:100%;max-width:420px;background:#11141b;border:1px solid var(--gm-line);border-radius:6px;color:var(--gm-text);padding:8px 10px;font-size:13px;";
  let taolus = [];
  catalog("/api/combat-dummy/taolus")
    .then((rows) => {
      taolus = rows || [];
      preset.set({ key: "content", value: JSON.stringify([{ label: "随机套路", value: "random" }, ...taolus.map((t) => ({ label: t.label + "（" + t.skills.length + "招）", value: t.key }))]) });
    })
    .catch(() => {});
  preset.addEventListener("change", () => { // ax-select 只派发原生 change（「changed」走 .on() 通道）
    const key = preset.value;
    let taolu = null;
    if (key === "随机套路") {
      if (!taolus.length) return;
      taolu = taolus[Math.floor(Math.random() * taolus.length)];
    } else {
      taolu = taolus.find((t) => t.label === key || t.key === key);
    }
    if (taolu) skills.value = taolu.skills.join("、");
  });
  const spawn = actionBtn({
    label: "生成战斗假人",
    build: () => ({
      action: "spawn_combat_dummy",
      dummyName: name.get() || "战斗假人",
      dummyHp: hp.get() ?? 0,
      dummyDamage: damage.get() ?? 0,
      dummySkills: skills.value.trim(),
    }),
    okText: "战斗假人已生成",
    section: "战斗假人",
  });
  body.appendChild(h("div", { class: "field-row" }, [name.node, hp.node, damage.node, h("label", {}, ["预设套路"]), preset]));
  body.appendChild(h("div", { class: "field-row" }, [skills]));
  body.appendChild(
    h("div", { class: "actions-row" }, [
      spawn,
      cmdBtn({ label: "清除全部战斗假人", danger: true, action: "clear_combat_dummies", section: "战斗假人" }),
    ])
  );
  host.appendChild(section);
}

/* ---------------- 伙伴系统 ---------------- */

function renderBot(host) {
  const { section, body } = panelSection({
    id: "sec-bot",
    title: "伙伴系统",
    desc: "伙伴操作直接作用于当前在线角色；拉伙伴不要求重新邀请，且会使用当前场景。伙伴默认跟随（走远自动追上，切图后在新场景身边重生）；辅助按周期给玩家回血。",
  });
  const enabled = h("ax-select", {
    content: JSON.stringify([
      { label: "开", value: "1" },
      { label: "关", value: "0" },
    ]),
    style: "width:80px;",
    value: "0",
  });
  const interval = numField({ label: "间隔(秒)", value: 5, min: 1, max: 60, width: "150px" });
  const heal = numField({ label: "治疗量", value: 500, min: 0, max: 100000, width: "150px" });
  const botCommand = (label, action, params) =>
    actionBtn({
      label,
      theme: "",
      build: () => Object.assign({ action }, params || {}),
      okText: label,
      section: "伙伴系统",
    });
  body.appendChild(
    h("div", { class: "actions-row" }, [
      botCommand("创建 / 校验青禾", "bot_ensure"),
      botCommand("拉伙伴到我身边", "bot_pull_here"),
      botCommand("开始跟随", "bot_follow_start"),
      botCommand("停止跟随", "bot_follow_stop"),
      botCommand("复活回满血", "bot_revive"),
    ])
  );
  body.appendChild(
    h("div", { class: "field-row", style: "margin-top:10px;" }, [
      h("label", {}, ["辅助"]), enabled, interval.node, heal.node,
      actionBtn({
        label: "配置辅助技能",
        build: () => ({
          action: "bot_support_configure",
          enabled: Number(enabled.value || 0),
          intervalSec: interval.get() ?? 5,
          heal: heal.get() ?? 0,
        }),
        okText: "辅助技能已配置",
        section: "伙伴系统",
      }),
    ])
  );
  const botState = h("pre", { style: "max-height:200px;overflow:auto;font-size:12px;margin-top:10px;" }, ["读取伙伴状态中……"]);
  subscribe((state) => {
    const bot = state.sessions[0] && state.sessions[0].bot;
    botState.textContent = bot ? JSON.stringify(bot, null, 2) : "无伙伴（点击「创建 / 校验青禾」生成）";
  });
  body.appendChild(botState);
  host.appendChild(section);
}

/* ---------------- 御风神水 ---------------- */

function renderYufeng(host) {
  const { section, body } = panelSection({
    id: "sec-yufeng",
    title: "御风神水（大轻功）",
    desc: "大轻功（百鸟凌云系列）冲刺要求 BuffStore 存在御风状态：先点「添加御风状态」（持续 10 分钟，可重复刷新），再上马/起跳后按轻功冲刺键。真实道具路线：发放神水 → 右键使用 → 蓄力条走完才加状态。",
  });
  const rideItems = [
    ["ride_windrunner_002", "发放御风道具（3天）"],
    ["ride_windrunner_004", "发放御风神水（7天）"],
    ["ride_windrunner_003", "发放御风神水（10天）"],
    ["ride_windrunner_005", "发放御风神水（14天）"],
  ];
  body.appendChild(
    h("div", { class: "actions-row" }, [
      cmdBtn({ label: "添加御风状态", theme: "prim", action: "yufeng_add", section: "御风神水" }),
      cmdBtn({ label: "取消御风状态", action: "yufeng_remove", section: "御风神水" }),
    ])
  );
  const rideRow = h("div", { class: "actions-row", style: "margin-top:8px;" }, []);
  for (const [configId, label] of rideItems) {
    rideRow.appendChild(
      actionBtn({
        label,
        theme: "",
        build: () => ({ action: "add_item", itemConfig: configId, amount: 1 }),
        okText: label,
        section: "御风神水",
      })
    );
  }
  body.appendChild(rideRow);
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = { dummy: renderDummy, "combat-dummy": renderCombatDummy, bot: renderBot, yufeng: renderYufeng };
  (renderers[sectionId] || renderDummy)(host);
}
