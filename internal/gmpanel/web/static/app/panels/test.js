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
  { id: "mail", title: "邮件", desc: "GM 系统邮件（文本+物品附件，RecvLetterRec 319 行）" },
  { id: "skill", title: "武学与修炼", desc: "重发武学 View40 / 内功 View43 / 轻功 View46" },
  { id: "dummy", title: "训练木桩", desc: "面前生成不还手的木桩（新手村官方练习木桩，血量自定义）" },
  { id: "scene", title: "传送与场景", desc: "场景内瞬移（0x1F）+ 跨场景切换（0x0C→0x0B→0x0D→0x09）" },
  { id: "debug", title: "调试工具", desc: "mXXX 自定义帧探针 + ReadyFrames 重放" },
  { id: "archive", title: "存档管理", desc: "0x04 存档（role_state_json 落 nx_roles，重登 RST1 回灌）" },
];

// 场景切换预设（gm scene key 直传）。场景下拉的正式数据来自 /api/gm/scenes
//（scenedata 内嵌表：客户端有地形的全部世界场景，world 侧同一份）；本表是
// 接口取不到时的回退（旧面板 exe / 请求失败），以及表外的补充键
//（born_single_01 新手引导）。
const SCENE_PRESETS = [
  ["born01", "鸡鸣驿（漠河镇）"], ["born02", "恶人谷"], ["born03", "烟雨庄"], ["born04", "千灯镇"],
  ["born_single_01", "新手引导（born_single_01 教学渔村）"],
  ["city01", "燕京"], ["city02", "苏州"], ["city03", "金陵"], ["city04", "洛阳"], ["city05", "成都"],
  ["school01", "锦衣卫"], ["school02", "丐帮"], ["school03", "君子堂"], ["school04", "极乐谷"],
  ["school05", "唐门"], ["school06", "峨眉"], ["school07", "武当"], ["school08", "少林"], ["school09", "移花宫"],
  ["scene01", "幽云十六州"], ["scene05", "捻罗坝"], ["scene06", "凌霄城"], ["scene08", "塞外大漠"],
];

// 场景表模块级缓存（切子页不重取）；null = 取不到，回退静态预设。
let sceneTable = null;
let sceneTableLoading = null;

function loadScenes() {
  if (!sceneTableLoading) {
    sceneTableLoading = api.gmScenes()
      .then((res) => (sceneTable = res.scenes || null))
      .catch(() => null);
  }
  return sceneTableLoading;
}

/** 场景表 → 下拉分组 [{category, label, options:[{value, label}]}]。分组与
 *  场景序 = 接口序（类序 + 编号）；无中文名回退 key；同组重名追加 key 区分
 *  （江湖·凌霄城 scene06 / scene25）；extra = 表外的补充预设，归「其它」组。
 *  纯函数（不依赖 DOM）。 */
export function groupScenes(scenes, extra = []) {
  const entries = scenes.map((sc) => ({
    value: sc.key, label: sc.name || sc.key, category: sc.category, categoryName: sc.categoryName,
  }));
  for (const [key, label] of extra) entries.push({ value: key, label, category: "other", categoryName: "其它" });
  const seen = new Map();
  for (const e of entries) {
    const id = e.category + "/" + e.label;
    seen.set(id, (seen.get(id) || 0) + 1);
  }
  const groups = [];
  const indexOf = new Map();
  for (const e of entries) {
    if (!indexOf.has(e.category)) {
      indexOf.set(e.category, groups.length);
      groups.push({ category: e.category, label: e.categoryName || e.category, options: [] });
    }
    const dup = seen.get(e.category + "/" + e.label) > 1;
    groups[indexOf.get(e.category)].options.push({ value: e.value, label: dup ? e.label + " " + e.value : e.label });
  }
  return groups;
}

const PROPS = [
  ["str", "臂力 Str"], ["dex", "身法 Dex"], ["ing", "内息 Ing"],
  ["spi", "气血 Spi"], ["sta", "体魄 Sta"],
  ["hp", "HP(含Max)"], ["mp", "MP(含Max)"], ["sp", "SP(含Max)"],
  ["parry", "招架 Parry"],
];

// 当前坐标读数（传送与场景子页）的四项：gm state 的 pos 字段 ↔ 标签。
const POS_KEYS = [["x", "x"], ["y", "y"], ["z", "z"], ["orient", "朝向"]];

/** gm state 应答行 → {scene, x, y, z, orient, degrees}（world gmcmd.go 的
 *  `scene=<场景键> pos=x,y,z,orient` 行；orient 弧度，atan2(dx,dz) 口径，
 *  degrees = 折到 [0,360) 的角度串）。无坐标行（目标不存在/world 报错）
 *  回 null；未进场 = scene 空串。纯函数。 */
export function parsePos(lines) {
  for (const line of lines || []) {
    const m = /(?:^|\s)scene=(\S*)\s+pos=(-?[\d.]+),(-?[\d.]+),(-?[\d.]+),(-?[\d.]+)(?:\s|$)/.exec(line);
    if (!m) continue;
    const degrees = ((((Number(m[5]) * 180) / Math.PI) % 360) + 360) % 360;
    return { scene: m[1], x: m[2], y: m[3], z: m[4], orient: m[5], degrees: degrees.toFixed(1) };
  }
  return null;
}

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

// 行内小按钮（坐标读数行用）：样式见 app.css .pos-line .inline-btn——
// ax-btn 35px / 框架 button 34px 高，都会把那一行撑起来。
function inlineBtn(label, fn) {
  const btn = h("button", { type: "button", class: "inline-btn" }, [label]);
  btn.addEventListener("click", fn);
  return btn;
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

  /* ---- 当前坐标（传送与场景子页）：gm state 静默读取，不进应答历史。
   * 选角色即读；瞬移/切图后与玩家名单轮询（5s）同步刷新。 ---- */

  // 读数节点（subScene 装配；其它子页为 null）
  let posUI = null;
  let posSeq = 0; // 最新一次请求序：换目标后迟到的旧应答丢弃

  // 有坐标 = 场景键 + 四项数值；否则收起数值，说明行给原因（未进场 /
  // 目标不存在 / world 不可达）。
  function renderPos(pos, lines) {
    const ok = !!(pos && pos.scene);
    posUI.items.style.display = ok ? "" : "none";
    posUI.scene.textContent = ok ? pos.scene : "";
    posUI.note.textContent = ok ? "" : pos ? "未进场" : (lines && lines[0]) || "（无应答）";
    if (!ok) return;
    for (const [key] of POS_KEYS) posUI.val[key].textContent = pos[key];
    posUI.val.orient.textContent = pos.orient + " ≈ " + pos.degrees + "°"; // 弧度原值 + 角度换算
  }

  async function refreshPos() {
    if (!posUI) return;
    if (!who()) {
      posSeq += 1;
      renderPos(null, ["world 不可达或未选择目标"]);
      return;
    }
    const seq = ++posSeq;
    let result;
    try {
      result = await api.gmRaw("gm state " + who());
    } catch (err) {
      result = { lines: [String((err && err.message) || err)] };
    }
    if (seq !== posSeq) return; // 已换目标：旧应答丢弃
    const lines = result.lines || [result.message]; // 接口失败：错误信息作说明行
    renderPos(parsePos(lines), lines);
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

  function subMail(box) {
    // 离线投递：目标须为角色名（store name-only 查档往返）；在线目标可
    // 用下拉名。标题/正文下划线转空格。
    const target = field("目标角色名（离线可投）", 170);
    target.value = who() || "";
    const title = field("标题（_转空格 ≤64）", 150);
    const content = field("正文（_转空格 ≤1024）", 240);
    const item = field("物品 config（可空）", 150);
    const amount = field("数量", 60);
    box.appendChild(h("div", { class: "toolbar" }, [
      target, title, content, item, amount,
      opBtn("gm mail（发信）", "info", async () => {
        if (!target.value || !title.value) return;
        const parts = ["gm mail", target.value, title.value, content.value || "-"];
        if (item.value) parts.push(item.value + ":" + (amount.value || "1"));
        await send(parts.join(" "));
      }),
    ]));
    box.appendChild(h("div", { class: "panel-desc" }, [
      "在线：单行 0x11 直推+存档；离线：store 往返（LoadRole name-only → 补信 → SaveRole，重启不丢）。",
      "玩家侧读信=353/领附件=359（附件入包 0x19+0x18；邮箱容量 50）。纯文本信物品留空即可。",
    ]));
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

  function subDummy(box) {
    const hp = field("血量（≥1）", 120);
    hp.value = "100000";
    const count = field("数量", 60);
    count.value = "1";
    box.appendChild(h("div", { class: "toolbar" }, [
      hp, count,
      opBtn("生成木桩", "info", async () => {
        if (!who() || !hp.value) return;
        await send("gm dummy " + who() + " " + hp.value + " " + (count.value || "1"));
      }),
      opBtn("100 万血木桩", "", () => who() && send("gm dummy " + who() + " 1000000")),
      opBtn("清除全部木桩", "warning", () => who() && send("gm dummy " + who() + " clear")),
    ]));
    box.appendChild(h("div", { class: "panel-desc" }, [
      "新手村官方练习木桩（funnpcborn003043）：不还手、不移动、脱战不回血，血量打空即死（尸体 5s 后消失，不重生）。",
      "单只 = 面朝方向前 1.8 米（普攻射程内），多只环绕 3 米；每人场上上限 20 只，切场景 / 下线自动清除。",
      "木桩只对生成它的玩家可见；命中、伤害、击杀帧与普通战斗 NPC 同一条链",
      "（NINEYIN_COMBAT=off 时只物化、不掉血）。",
    ]));
  }

  function subScene(box) {
    // 当前坐标读数：选中角色的 x/y/z/朝向（场景键随行），一行排开，先看位置再传送。
    const posVal = {};
    const posItems = POS_KEYS.map(([key, label]) => {
      posVal[key] = h("span", { class: "pos-v" }, ["-"]);
      return h("span", { class: "pos-item" }, [h("span", { class: "pos-k" }, [label]), posVal[key]]);
    });
    posUI = {
      scene: h("span", { class: "pos-scene" }),
      items: h("span", { class: "pos-items", style: "display:none" }, posItems), // 读到坐标才展开
      note: h("span", { class: "pos-note" }),
      val: posVal,
    };
    box.appendChild(h("div", { class: "toolbar pos-line" }, [
      h("span", { class: "pos-k" }, ["当前坐标"]), posUI.scene, posUI.items, posUI.note,
      inlineBtn("刷新坐标", refreshPos),
    ]));

    const tx = field("x"), ty = field("y"), tz = field("z"), to = field("orient（默认1）", 120);
    box.appendChild(h("div", { class: "toolbar" }, [
      tx, ty, tz, to,
      opBtn("gm tp（0x1F 瞬移）", "warning", async () => {
        if (!who() || !tx.value || !ty.value || !tz.value) return;
        await send(["gm tp", who(), tx.value, ty.value, tz.value, to.value || "1"].join(" "));
        refreshPos();
      }),
    ]));
    // 跨场景切换：场景下拉（接口数据，按类分组）+ 可选驿站（选中即以其坐标
    // 覆盖落点），完整重入序列。
    const presetSel = h("select", {}, SCENE_PRESETS.map(([v, label]) => h("option", { value: v }, [label])));
    const wpSel = h("select", { title: "驿站：选中后以其坐标覆盖落点" }, [h("option", { value: "" }, ["默认落点"])]);
    const sx = field("覆盖x"), sy = field("覆盖y"), sz = field("覆盖z"), so = field("朝向(可空)", 90);
    const byKey = new Map(); // key → 场景（接口数据到达后填充）
    const fillWaypoints = (scene) => {
      wpSel.textContent = "";
      wpSel.appendChild(h("option", { value: "" }, ["默认落点"]));
      for (const wp of (scene && scene.waypoints) || []) {
        wpSel.appendChild(h("option", { value: wp.name }, [wp.title || wp.name]));
      }
    };
    presetSel.addEventListener("change", () => {
      fillWaypoints(byKey.get(presetSel.value));
      sx.value = sy.value = sz.value = so.value = "";
    });
    wpSel.addEventListener("change", () => {
      const wp = ((byKey.get(presetSel.value) || {}).waypoints || []).find((w) => w.name === wpSel.value);
      sx.value = wp ? wp.x : "";
      sy.value = wp ? wp.y : "";
      sz.value = wp ? wp.z : "";
      so.value = wp ? wp.orient : "";
    });
    loadScenes().then((scenes) => {
      if (!scenes) return;
      const prev = presetSel.value;
      for (const sc of scenes) byKey.set(sc.key, sc);
      const extra = SCENE_PRESETS.filter(([key]) => !byKey.has(key));
      presetSel.textContent = "";
      for (const group of groupScenes(scenes, extra)) {
        presetSel.appendChild(h("optgroup", { label: group.label },
          group.options.map((o) => h("option", { value: o.value }, [o.label]))));
      }
      presetSel.value = byKey.has(prev) ? prev : (scenes[0] && scenes[0].key) || "";
      fillWaypoints(byKey.get(presetSel.value));
    });
    box.appendChild(h("div", { class: "toolbar", style: "margin-top:8px" }, [
      presetSel, wpSel, sx, sy, sz, so,
      opBtn("gm scene（跨场景切换）", "error", async () => {
        if (!who() || !presetSel.value) return;
        const parts = ["gm scene", who(), presetSel.value];
        if (sx.value && sy.value && sz.value) {
          parts.push(sx.value, sy.value, sz.value);
          if (so.value !== "") parts.push(so.value);
        }
        await send(parts.join(" "));
        refreshPos();
      }),
    ]));
    box.appendChild(h("div", { class: "panel-desc" }, [
      "gm scene：0x0C 退出 → 750ms → 0x0B/0x0D 进新场景 → 客户端 0x09 → 0x1F+vitals+开图+NPC 首拍+",
      "Ready 重推+邮箱重放（2433 scene_transition 真机时序）；落点默认该场景出生点（BornCJ）或",
      "默认驿站（场景表：客户端有地形的全部世界场景）；下拉选驿站即按点传送。",
      "走近传送门（door_data 提取 113 条路线）自动触发，2s 冷却。",
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

  ({ role: subRole, item: subItem, mail: subMail, skill: subSkill, dummy: subDummy, scene: subScene, debug: subDebug, archive: subArchive }[sub])(body);
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
    if (sub === "scene") refreshPos();
  }

  target.addEventListener("change", () => {
    if (sub === "scene") refreshPos();
  });

  refreshPlayers();
  // 应答历史重建后滚底；玩家名单自轮询 5s
  setInterval(() => {
    if (!document.contains(target)) return; // 已切页：停更
    refreshPlayers();
  }, 5000);

  return { refresh: () => {} };
}
