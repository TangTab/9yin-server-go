// params.js —— 组8「游戏参数」：怪物刷新 / 数据存盘 / 移动参数。
// 全是 M3/M4 形态的数值与开关；界值与旧版一致（服务端契约层再权威校验）。

import { h, panelSection, numField, actionBtn, cmdBtn } from "../ui.js";

/* ---------------- 怪物刷新 ---------------- */

function renderRespawn(host) {
  const { section, body } = panelSection({
    id: "sec-respawn",
    title: "怪物刷新",
    desc: "批量刷新：每到间隔统一重生所有已死亡怪物（活怪不动），0=关闭批量回到每怪独立计时。尸体清除：死亡超过秒数后尸体消失，0=保留到重生。宝箱存留：挂尸掉落/宝箱实体超时未拾自动消失（0=保留到拾空）。自动追击：进程级总开关，关闭后仇恨照常持有但怪停在出生点。",
  });
  const interval = numField({ label: "批量间隔/秒", value: 30, min: 0, max: 86400, width: "150px" });
  const corpse = numField({ label: "尸体清除/秒", value: 10, min: 0, max: 3600, width: "150px" });
  const lootbox = numField({ label: "宝箱存留/秒", value: 300, min: 0, max: 86400, width: "150px" });
  const chase = h("input", { type: "checkbox", checked: true });
  chase.style.cssText = "width:16px;height:16px;accent-color:var(--gm-gold);";
  body.appendChild(
    h("div", { class: "field-row" }, [
      interval.node,
      actionBtn({
        label: "应用批量间隔",
        build: () => ({ action: "set_respawn_interval", value: interval.get() ?? 30 }),
        okText: "批量刷新间隔已应用",
        section: "怪物刷新",
      }),
      corpse.node,
      actionBtn({
        label: "应用尸体清除",
        build: () => ({ action: "set_corpse_ttl", value: corpse.get() ?? 10 }),
        okText: "尸体清除已应用",
        section: "怪物刷新",
      }),
    ])
  );
  body.appendChild(
    h("div", { class: "field-row" }, [
      lootbox.node,
      actionBtn({
        label: "应用宝箱存留",
        build: () => ({ action: "set_lootbox_ttl", value: lootbox.get() ?? 300 }),
        okText: "宝箱存留已应用",
        section: "怪物刷新",
      }),
      h("label", {}, ["自动追击"]), chase,
      actionBtn({
        label: "应用自动追击",
        build: () => ({ action: "set_npc_chase", enabled: chase.checked }),
        okText: chase.checked ? "自动追击已开启" : "自动追击已关闭",
        section: "怪物刷新",
      }),
    ])
  );
  host.appendChild(section);
}

/* ---------------- 数据存盘 ---------------- */

function renderAutosave(host) {
  const { section, body } = panelSection({
    id: "sec-autosave",
    title: "数据存盘",
    desc: "自动存盘：每到间隔把所有在线玩家的内功修为/经脉快照落库（与正常下线同一组存盘，强杀最多丢一个周期）；0=关闭，仅断线与 Ctrl+C 优雅关服时存。进程级设置，改动即刻生效，重启回落启动参数。",
  });
  const interval = numField({ label: "间隔/秒", value: 300, min: 0, max: 86400, width: "150px" });
  body.appendChild(
    h("div", { class: "field-row" }, [
      interval.node,
      actionBtn({
        label: "应用",
        build: () => ({ action: "set_autosave_interval", value: interval.get() ?? 300 }),
        okText: "自动存盘间隔已应用",
        section: "数据存盘",
      }),
    ])
  );
  host.appendChild(section);
}

/* ---------------- 移动参数 ---------------- */

function renderMovement(host) {
  const { section, body } = panelSection({
    id: "sec-movement",
    title: "移动参数",
    desc: "移动速度同时更新 MoveSpeed 与 RunSpeed；下落重力数值越大下落越快，下一次起跳生效（范围 0.1–100）。",
  });
  const speed = numField({ label: "移动速度", value: 5, min: 0.1, max: 100, step: 0.5, width: "150px" });
  const gravity = numField({ label: "下落重力", value: 9.8, min: 0.1, max: 100, step: 0.1, width: "150px" });
  body.appendChild(
    h("div", { class: "field-row" }, [
      speed.node,
      actionBtn({
        label: "应用速度",
        build: () => ({ action: "set_move_speed", value: speed.get() ?? 5 }),
        okText: "移动速度已应用",
        section: "移动参数",
      }),
      gravity.node,
      actionBtn({
        label: "应用重力",
        build: () => ({ action: "set_gravity", value: gravity.get() ?? 9.8 }),
        okText: "下落重力已应用",
        section: "移动参数",
      }),
      cmdBtn({ label: "测试伤害飘字 24680", action: "fightword_probe", section: "移动参数" }),
    ])
  );
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = { respawn: renderRespawn, autosave: renderAutosave, movement: renderMovement };
  (renderers[sectionId] || renderRespawn)(host);
}
