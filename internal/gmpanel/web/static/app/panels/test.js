// test.js —— GM 测试台（二级目录壳）：左侧导航的子页分组对齐 2433
// gm-panel 的 registry GROUPS 语义（角色状态/货币与物品/武学与修炼/
// 传送与场景/调试工具/存档管理）。公共层 = 目标玩家选择 + 应答输出区
//（模块级持久，切子页不丢历史）；子页经 mount(view, sub) 分派。
// 命令透传 POST /api/gm/raw（world echo `gm` 族，方言 gmcmd.go）。

import { h } from "../ui.js";
import * as api from "../api.js";

const SUBS = [
  { id: "role", title: "角色状态", desc: "gm state 快照 + 五维/资源热更（0x10）" },
  { id: "item", title: "货币与物品", desc: "碎银 CapitalType1（2433 线同名对拍）" },
  { id: "skill", title: "武学与修炼", desc: "重发武学 View40 / 内功 View43 / 轻功 View46" },
  { id: "scene", title: "传送与场景", desc: "场景内瞬移（0x1F 位置帧）" },
  { id: "debug", title: "调试工具", desc: "mXXX 自定义帧探针 + ReadyFrames 重放" },
  { id: "archive", title: "存档管理", desc: "0x04 存档（role_state_json 落 nx_roles，重登 RST1 回灌）" },
];

const PROPS = [
  ["str", "臂力 Str"], ["dex", "身法 Dex"], ["ing", "内息 Ing"],
  ["spi", "气血 Spi"], ["sta", "体魄 Sta"],
  ["hp", "HP(含Max)"], ["mp", "MP(含Max)"], ["sp", "SP(含Max)"],
  ["parry", "招架 Parry"],
];

// 应答历史（模块级：子页切换不丢，上限 200 条滚动清）
const HISTORY_MAX = 200;
let history = [];
let seq = 0;

function opBtn(label, theme, fn) {
  const btn = h("ax-btn", { label, theme: theme || "" });
  btn.addEventListener("click", fn);
  return btn;
}

function field(placeholder, width) {
  return h("input", { type: "text", placeholder, style: "width:" + (width || 120) + "px" });
}

export function mount(view, sub) {
  sub = SUBS.some((s) => s.id === sub) ? sub : "role";
  const meta = SUBS.find((s) => s.id === sub);
  const target = h("select", {});
  const out = h("div", { class: "log-box", style: "height:300px" });
  const stateBox = h("div", { class: "grid" });

  function renderOut() {
    out.textContent = "";
    for (const entry of history) {
      out.appendChild(h("span", { class: "meta" }, ["[" + entry.stamp + "] " + entry.cmd + "\n"]));
      out.appendChild(document.createTextNode((entry.lines || []).join("\n") + "\n"));
    }
    out.scrollTop = out.scrollHeight;
  }

  async function send(line) {
    seq += 1;
    const result = await api.gmRaw(line);
    history.push({ stamp: new Date().toLocaleTimeString("zh-CN", { hour12: false }), cmd: "$ " + line, lines: result.lines || [result.message || "（无应答）"] });
    if (history.length > HISTORY_MAX) history = history.slice(-HISTORY_MAX);
    renderOut();
    return result.lines || [];
  }

  const who = () => target.value;

  async function refreshState() {
    if (!who()) return;
    stateBox.innerHTML = "";
    const lines = await send("gm state " + who());
    const kv = [];
    for (const line of lines) {
      for (const seg of line.split(" ")) {
        const pair = seg.split("=");
        if (pair.length === 2) kv.push([pair[0], pair[1]]);
      }
    }
    for (const [k, v] of kv) {
      stateBox.appendChild(h("div", { class: "stat-card" }, [
        h("div", { class: "k" }, [k]), h("div", { class: "v" }, [v]),
      ]));
    }
    if (!kv.length) stateBox.appendChild(h("div", { class: "panel-desc" }, ["（无状态数据——world 不可达或玩家未进场）"]));
  }

  /* ---- 子页渲染 ---- */

  function subRole(box) {
    const propSel = h("select", {}, PROPS.map(([v, label]) => h("option", { value: v }, [label])));
    const val = field("数值（≥0）", 110);
    box.appendChild(h("div", { class: "toolbar" }, [
      propSel, val,
      opBtn("gm set（0x10 热发）", "info", async () => {
        if (!who() || !val.value) return;
        await send("gm set " + who() + " " + propSel.value + " " + val.value);
        refreshState();
      }),
      opBtn("读状态", "", refreshState),
    ]));
    box.appendChild(h("section", { class: "panel" }, [h("h2", {}, ["状态快照"]), stateBox]));
  }

  function subItem(box) {
    const val = field("碎银数额（≥0）", 140);
    box.appendChild(h("div", { class: "toolbar" }, [
      val,
      opBtn("gm money（CapitalType1 热发）", "info", async () => {
        if (!who() || !val.value) return;
        await send("gm money " + who() + " " + val.value);
        refreshState();
      }),
      opBtn("读余额", "", refreshState),
    ]));
    box.appendChild(h("div", { class: "panel-desc" }, [
      "官银/黄金（CapitalType0/2）留多币种扩展；建号初值 99999（2433 线 starter 口径）。",
    ]));
    box.appendChild(h("section", { class: "panel" }, [h("h2", {}, ["状态快照"]), stateBox]));
  }

  function subSkill(box) {
    box.appendChild(h("div", { class: "toolbar" }, [
      opBtn("重发武学 View40", "info", () => who() && send("gm view " + who() + " 40")),
      opBtn("重发内功 View43", "info", () => who() && send("gm view " + who() + " 43")),
      opBtn("重发轻功 View46", "info", () => who() && send("gm view " + who() + " 46")),
    ]));
    box.appendChild(h("div", { class: "panel-desc" }, [
      "技能/内功数据源 = roleState（nx_roles.role_state_json）——学/忘技能的写口属装备背包域扩展（主树暂无 inventory 地基）。",
    ]));
  }

  function subScene(box) {
    const tx = field("x"), ty = field("y"), tz = field("z"), to = field("orient（默认1）", 120);
    box.appendChild(h("div", { class: "toolbar" }, [
      tx, ty, tz, to,
      opBtn("gm tp（0x1F 瞬移）", "warning", async () => {
        if (!who() || !tx.value || !ty.value || !tz.value) return;
        await send(["gm tp", who(), tx.value, ty.value, tz.value, to.value || "1"].join(" "));
        refreshState();
      }),
    ]));
    box.appendChild(h("div", { class: "panel-desc" }, [
      "跨场景切换（switch_scene/副本）2433 有、主树留场景域扩展。",
    ]));
  }

  function subDebug(box) {
    const msgId = field("mXXX（≥256）", 120);
    const msgArgs = field("int 参数（空格分隔，可空）", 220);
    box.appendChild(h("div", { class: "toolbar" }, [
      msgId, msgArgs,
      opBtn("gm msg（自定义帧探针）", "info", async () => {
        if (!who() || !msgId.value) return;
        await send(("gm msg " + who() + " " + msgId.value + " " + msgArgs.value).trim());
      }),
    ]));
    box.appendChild(h("div", { class: "toolbar" }, [
      opBtn("gm flush（重放 ReadyFrames）", "warning", () => who() && send("gm flush " + who())),
    ]));
    box.appendChild(h("div", { class: "panel-desc" }, [
      "msg 对应 2433 的 custom_probe（数值寻址 mXXX）；flush 用于帧对拍与客户端卡死恢复。",
    ]));
  }

  function subArchive(box) {
    box.appendChild(h("div", { class: "toolbar" }, [
      opBtn("立即存档（0x04）", "error", () => who() && send("gm save " + who())),
      opBtn("读状态", "", refreshState),
    ]));
    box.appendChild(h("div", { class: "panel-desc" }, [
      "存档链：gm set → 内存 + 0x10 热发 → gm save / 下线自动存 → 0x04 merge 落 nx_roles.role_state_json →",
      " 重登 0x02 RST1 回灌 → 进场帧新值。store 端 merge 保留 world 不可知列（save_para1/role_index）。",
      "周期存档 ticker 与官方 0xD0B save_data blob 留后续。",
    ]));
    box.appendChild(h("section", { class: "panel" }, [h("h2", {}, ["状态快照"]), stateBox]));
  }

  /* ---- 壳装配 ---- */

  const body = h("div", {});
  view.appendChild(h("section", { class: "panel" }, [
    h("h2", {}, ["GM 测试台 · " + meta.title]),
    h("div", { class: "panel-desc" }, [meta.desc + "；目标 = 名字或账号，命令经 world echo 15001 透传。"]),
    h("div", { class: "toolbar" }, [h("span", {}, ["目标"]), target, opBtn("刷新玩家", "", refreshPlayers)]),
  ]));
  view.appendChild(body);
  view.appendChild(h("section", { class: "panel" }, [h("h2", {}, ["应答输出"]), out]));

  ({ role: subRole, item: subItem, skill: subSkill, scene: subScene, debug: subDebug, archive: subArchive }[sub])(body);
  if (sub === "role" || sub === "item" || sub === "archive") refreshState();
  renderOut();

  async function refreshPlayers() {
    try {
      const data = await api.players();
      const cur = target.value;
      target.innerHTML = "";
      for (const p of data.players || []) {
        const name = p.name || p.account;
        if (name) target.appendChild(h("option", { value: name }, [name + "（" + p.state + "）"]));
      }
      if (cur && Array.from(target.options).some((o) => o.value === cur)) target.value = cur;
    } catch {
      target.innerHTML = "";
      target.appendChild(h("option", {}, ["（world 不可达）"]));
    }
  }

  refreshPlayers();
  // 应答历史重建后滚底；玩家名单自轮询 5s
  setInterval(() => {
    if (!document.contains(target)) return; // 已切页：停更
    refreshPlayers();
  }, 5000);

  return { refresh: () => {} };
}
