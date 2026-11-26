// scene_subpage.mjs —— GM 测试台「传送与场景」子页逻辑回归（由 Go 测试
// jstest_test.go 在 Node 可用时调起；场景数据取自真起的面板测试服务）。
//
// 最小假 DOM（fakedom.mjs）+ fetch 桩，真跑 panels/test.js 的 mount(view, "scene")：加载
// 场景表 → 填充下拉（分组/重名/回退键）→ 选驿站填坐标 → 生成 gm scene
// 命令行 → 接口失败回退静态预设。环境变量：
//
//	GM_JS_APP=<目录>      含 ui.js / api.js / panels/test.js（从嵌入资源拷出）
//	GM_JS_SCENES=<文件>   /api/gm/scenes 的 JSON
//
// 断言不过即未捕获异常退出非零；通过打印 "all assertions passed"。
import { readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";
import path from "node:path";
import assert from "node:assert/strict";
import { El, installDOM } from "./fakedom.mjs";

const appDir = process.env.GM_JS_APP;
const scenesFile = process.env.GM_JS_SCENES;
if (!appDir || !scenesFile) {
  throw new Error("GM_JS_APP / GM_JS_SCENES 未设置");
}
const scenes = JSON.parse(readFileSync(scenesFile, "utf8")).scenes;
const STATIC_PRESETS = 23; // panels/test.js 的 SCENE_PRESETS 条数（回退表）

installDOM();

const reply = (body) => ({ ok: true, status: 200, json: async () => body });
const fail = () => ({ ok: false, status: 404, json: async () => ({}) });
const posted = [];
const stateQueries = []; // 全部 gm state 命令行（坐标读数走这条，静默）
let scenesAvailable = true;
let playersAvailable = false; // 第 7 节起 /api/players 给一名在线玩家
// world echo 的 gm state 方言桩（gmcmd.go state 分支五行）：stateError 非空
// = 只回这一行（目标不存在等，无坐标行）。
let stateError = null;
let stateScene = "ini\\scene\\born_single_01";
let statePos = "200.136,5.685,-116.064,3.140";
const stateLines = () =>
  stateError ? [stateError] : [
    "account=test001 name=岳绮丝 state=Online",
    "str=3 dex=3 ing=3 spi=3 sta=3",
    "hp=70 mp=40 sp=80 parry=100 money=9999",
    "scene=" + stateScene + " pos=" + statePos,
    "skills=27 neigong=6 cur=ng_wd_001 facultySeed=0",
  ];
// 挂起下一条 gm state 应答（迟到应答丢弃用）：holdNext 置真后，应答存入
// held，由测试择时放行。rawError 非空 = /api/gm/raw 整体回 503（world 不可达）。
let holdNext = false;
const held = [];
let rawError = null;
globalThis.fetch = async (url, init = {}) => {
  if (url === "/api/gm/scenes") return scenesAvailable ? reply({ scenes }) : fail();
  if (url === "/api/players") {
    return playersAvailable
      ? reply({ players: [{ account: "test001", name: "岳绮丝", state: "Online" }] })
      : fail();
  }
  if (url === "/api/gm/raw") {
    const line = JSON.parse(init.body || "{}").line;
    posted.push(line);
    if (rawError) return { ok: false, status: 503, json: async () => ({ error: rawError }) };
    if (!line.startsWith("gm state ")) return reply({ lines: ["ok"] });
    stateQueries.push(line);
    const lines = stateLines(); // 应答内容在请求时刻定格（迟到应答仍是旧值）
    if (!holdNext) return reply({ lines });
    holdNext = false;
    return new Promise((resolve) => held.push(() => resolve(reply({ lines }))));
  }
  return fail();
};

const testURL = pathToFileURL(path.join(appDir, "panels", "test.js")).href;
const mod = await import(testURL);
const settle = () => new Promise((r) => setTimeout(r, 20));
// 最近一次 gm scene 命令行（点击后还会跟一条坐标读数的 gm state）。
const lastSceneCmd = () => posted.filter((l) => l.startsWith("gm scene ")).pop();

async function mountScene(page) {
  const view = new El("div");
  page.mount(view, "scene");
  await settle(); // 等场景表加载
  const [target, presetSel, wpSel] = view.collect("SELECT");
  if (!target.options().some((o) => o.attrs.value)) {
    // 玩家名单接口不给数据时放一个可选目标（真实界面从在线列表选）
    const who = new El("option");
    who.setAttribute("value", "test001");
    target.appendChild(who);
    target.value = "test001";
  }
  // 瞬移工具栏的四个输入：x/y/z/orient（前四个 input）
  const [tpX, tpY, tpZ] = view.collect("INPUT").slice(0, 3);
  // 场景工具栏的四个输入：覆盖x/y/z、朝向（最后四个 input）
  const [sx, sy, sz, so] = view.collect("INPUT").slice(-4);
  const button = (label) =>
    view.find((n) => (n.tagName === "AX-BTN" && n.attrs.label === label) || (n.tagName === "BUTTON" && n.textContent === label));
  const btn = button("gm scene（跨场景切换）");
  assert.ok(presetSel && wpSel && btn, "scene toolbar built");
  // 当前坐标读数（一行）：场景键 + 四项数值（标签 → 值）+ 说明
  const line = view.find((n) => n._class === "toolbar pos-line");
  const val = Object.fromEntries(
    line.collect("SPAN").filter((n) => n._class === "pos-item").map((item) => [item.children[0].textContent, item.children[1]])
  );
  const part = (cls) => line.children.find((c) => c._class === cls);
  return {
    view, target, presetSel, wpSel, sx, sy, sz, so, btn, tpX, tpY, tpZ, button, val,
    scene: part("pos-scene"), items: part("pos-items"), note: part("pos-note"),
    log: view.find((n) => n._class === "log-box"),
  };
}

// 坐标读数四项的当前文本（x / y / z / 朝向）
const posText = (ui) => ["x", "y", "z", "朝向"].map((k) => ui.val[k].textContent);

const labels = (group) => group.children.filter((c) => c.tagName === "OPTION").map((o) => o.textContent);

// ---- 1. 接口数据到达：按类分组、全部场景 + 表外补充键、保留原选中项 ----
const ui = await mountScene(mod);
const groups = ui.presetSel.children.filter((c) => c.nodeType === 1 && c.tagName === "OPTGROUP");
assert.deepEqual(groups.map((g) => g.attrs.label), ["主城", "出生镇", "门派", "江湖", "其它"]);
assert.equal(ui.presetSel.options().length, scenes.length + 1, "options = 场景表 + 新手引导");
assert.equal(ui.presetSel.value, "born01", "previous selection kept");
const [, , school, jianghu, other] = groups.map(labels);
assert.ok(jianghu.includes("凌霄城 scene06") && jianghu.includes("凌霄城 scene25"), "同组重名追加 key");
// 此前无中文名、回退显示资源名的两个场景：场景名取自文案表的场景配置目录
// 键；星垂城与 scene35 同名，同样追加 key。（无中文名回退键见第 6 节。）
assert.ok(school.includes("神机营") && jianghu.includes("星垂城 scene35") && jianghu.includes("星垂城 scene36"), "目录键场景名 + 同组重名");
assert.ok(!school.includes("school21") && !jianghu.includes("scene36"), "有中文名后不再回退资源名");
assert.deepEqual(other, ["新手引导（born_single_01 教学渔村）"], "表外补充键归其它组");
// 表里的每个场景都在下拉里（value = key）——下拉列的 world 都能解析。
const values = new Set(ui.presetSel.options().map((o) => o.attrs.value));
for (const sc of scenes) assert.ok(values.has(sc.key), "scene " + sc.key + " listed");

// ---- 2. 选移花宫 → 驿站下拉；选驿站 → 覆盖坐标 → 命令行带坐标朝向 ----
const yihua = scenes.find((sc) => sc.key === "school09");
assert.ok(yihua && yihua.waypoints.length >= 2, "school09 has waypoints");
ui.presetSel.value = "school09";
ui.presetSel.fire("change");
assert.equal(ui.wpSel.options()[0].textContent, "默认落点");
assert.deepEqual(ui.wpSel.options().slice(1).map((o) => o.textContent), yihua.waypoints.map((w) => w.title || w.name));
const target = yihua.waypoints[yihua.waypoints.length - 1];
ui.wpSel.value = target.name;
ui.wpSel.fire("change");
assert.deepEqual([ui.sx.value, ui.sy.value, ui.sz.value, ui.so.value], [String(target.x), String(target.y), String(target.z), String(target.orient)]);
await ui.btn.fire("click");
await settle();
assert.equal(lastSceneCmd(), ["gm scene", "test001", "school09", target.x, target.y, target.z, target.orient].join(" "));

// ---- 3. 默认落点：清空覆盖、不带坐标 ----
ui.wpSel.value = "";
ui.wpSel.fire("change");
assert.deepEqual([ui.sx.value, ui.sy.value, ui.sz.value, ui.so.value], ["", "", "", ""]);
await ui.btn.fire("click");
await settle();
assert.equal(lastSceneCmd(), "gm scene test001 school09");

// ---- 4. 换场景清空覆盖 ----
ui.presetSel.value = "scene12";
ui.presetSel.fire("change");
assert.deepEqual([ui.sx.value, ui.sy.value, ui.sz.value, ui.so.value], ["", "", "", ""]);
assert.deepEqual(ui.wpSel.options().map((o) => o.textContent), ["默认落点", "考古点"]);
await ui.btn.fire("click");
await settle();
assert.equal(lastSceneCmd(), "gm scene test001 scene12");

// ---- 5. 接口取不到：回退静态预设（模块级缓存按实例计——用新实例） ----
scenesAvailable = false;
const fresh = await import(testURL + "?fallback=1");
const fb = await mountScene(fresh);
assert.equal(fb.presetSel.options().length, STATIC_PRESETS, "static fallback options");
assert.equal(fb.presetSel.value, "born01");
assert.deepEqual(fb.wpSel.options().map((o) => o.textContent), ["默认落点"]);

// ---- 6. groupScenes 纯函数边界 ----
assert.deepEqual(mod.groupScenes([]), []);
assert.deepEqual(
  mod.groupScenes([{ key: "x1", name: "", category: "scene", categoryName: "江湖" }], [["born_single_01", "新手引导"]]),
  [
    { category: "scene", label: "江湖", options: [{ value: "x1", label: "x1" }] },
    { category: "other", label: "其它", options: [{ value: "born_single_01", label: "新手引导" }] },
  ],
);
// 跨组同名不加后缀（念萝坝：门派 school15 / 江湖 scene05）。
const cross = mod.groupScenes([
  { key: "school15", name: "念萝坝", category: "school", categoryName: "门派" },
  { key: "scene05", name: "念萝坝", category: "scene", categoryName: "江湖" },
]);
assert.deepEqual(cross.map((g) => g.options[0].label), ["念萝坝", "念萝坝"]);

// ---- 7. 当前坐标读数：选角色即读 gm state（静默，不进应答输出） ----
// 7a. parsePos 纯函数：world gmcmd.go 的 state 行方言
const live = mod.parsePos([
  "account=test001 name=岳绮丝 state=Online",
  "str=3 dex=3 ing=3 spi=3 sta=3",
  "scene=ini\\scene\\born_single_01 pos=200.136,5.685,-116.064,3.140",
  "skills=27 neigong=6 cur=ng_wd_001 facultySeed=0",
]);
assert.deepEqual(live, {
  scene: "ini\\scene\\born_single_01", x: "200.136", y: "5.685", z: "-116.064", orient: "3.140", degrees: "179.9",
});
assert.equal(mod.parsePos(['gm state: player "ghost" not found']), null, "无坐标行 → null");
assert.equal(mod.parsePos([]), null);
assert.equal(mod.parsePos(undefined), null, "接口失败无应答行 → null");
assert.equal(mod.parsePos(["scene= pos=0.000,0.000,0.000,0.000"]).scene, "", "未进场 scene 空串");
// 朝向弧度折 [0,360)：-π/2 → 270°；驿站表上界 6.21 → 355.8°
assert.equal(mod.parsePos(["scene=a pos=0,0,0,-1.571"]).degrees, "270.0");
assert.equal(mod.parsePos(["scene=a pos=0,0,0,6.21"]).degrees, "355.8");

// 7b. 在线名单到位即读；坐标读取不进应答输出
playersAvailable = true;
const pv = await mountScene(mod);
assert.equal(pv.target.value, "岳绮丝", "目标 = 在线名单的名字");
assert.equal(stateQueries.at(-1), "gm state 岳绮丝", "名单到位即读坐标");
assert.equal(pv.scene.textContent, "ini\\scene\\born_single_01", "场景键随行");
assert.ok(!pv.items.style.display, "读到坐标即展开四项");
assert.equal(pv.note.textContent, "");
assert.deepEqual(posText(pv), ["200.136", "5.685", "-116.064", "3.140 ≈ 179.9°"]);
assert.ok(!pv.log.textContent.includes("gm state"), "坐标读取不进应答输出");
// 玩家名单取不到（第 5 节的回退实例）：数值收起，说明行提示，不发查询
assert.equal(fb.items.style.display, "none");
assert.equal(fb.note.textContent, "world 不可达或未选择目标");

// 7c. 刷新坐标
stateScene = "ini\\scene\\born03";
statePos = "-8.250,0.500,77.000,-1.571";
await pv.button("刷新坐标").fire("click");
await settle();
assert.equal(pv.scene.textContent, "ini\\scene\\born03");
assert.deepEqual(posText(pv), ["-8.250", "0.500", "77.000", "-1.571 ≈ 270.0°"]);

// 7d. gm tp 后重读坐标
statePos = "1.000,2.000,3.000,1.000";
pv.tpX.value = "1";
pv.tpY.value = "2";
pv.tpZ.value = "3";
await pv.button("gm tp（0x1F 瞬移）").fire("click");
await settle();
assert.equal(posted.at(-2), "gm tp 岳绮丝 1 2 3 1");
assert.equal(stateQueries.at(-1), "gm state 岳绮丝", "瞬移后重读坐标");
assert.deepEqual(posText(pv), ["1.000", "2.000", "3.000", "1.000 ≈ 57.3°"]);

// 7e. 换目标即读；先发的旧目标应答迟到不覆盖（请求序丢弃）
const second = new El("option");
second.setAttribute("value", "test002");
pv.target.appendChild(second);
holdNext = true;
statePos = "9.000,9.000,9.000,0.000";
pv.target.value = "test002";
pv.target.fire("change"); // test002 的查询发出，应答挂起
assert.equal(stateQueries.at(-1), "gm state test002", "换目标即读");
assert.equal(held.length, 1);
statePos = "7.000,7.000,7.000,0.000";
pv.target.value = "岳绮丝";
pv.target.fire("change"); // 换回：新查询立即应答
await settle();
assert.deepEqual(posText(pv).slice(0, 3), ["7.000", "7.000", "7.000"]);
held.shift()(); // 旧目标的应答此刻才到
await settle();
assert.deepEqual(posText(pv).slice(0, 3), ["7.000", "7.000", "7.000"], "迟到的旧目标应答丢弃");

// 7f. 目标不存在 / 未进场 / world 不可达：数值收起，说明行给原因
stateError = 'gm state: player "岳绮丝" not found';
await pv.button("刷新坐标").fire("click");
await settle();
assert.equal(pv.items.style.display, "none", "目标不存在：数值收起");
assert.equal(pv.scene.textContent, "");
assert.equal(pv.note.textContent, 'gm state: player "岳绮丝" not found');
stateError = null;
stateScene = "";
statePos = "0.000,0.000,0.000,0.000";
await pv.button("刷新坐标").fire("click");
await settle();
assert.equal(pv.items.style.display, "none", "未进场不显示零坐标");
assert.equal(pv.note.textContent, "未进场");
rawError = "echo 127.0.0.1:15001 不可达: dial tcp 127.0.0.1:15001: connectex: No connection could be made";
await pv.button("刷新坐标").fire("click");
await settle();
assert.equal(pv.items.style.display, "none", "world 不可达：数值收起");
assert.equal(pv.note.textContent, rawError, "world 不可达：错误行作说明");
rawError = null;

// ---- 8. 应答输出显示 world 的逐行应答（gmRaw 曾把应答体的 lines 丢在
// postJSON 里，应答区对任何命令恒显「（无应答）」） ----
assert.ok(pv.log.textContent.includes("$ gm tp 岳绮丝 1 2 3 1\nok\n"), "gm tp 应答逐行进应答区");
assert.ok(pv.log.textContent.includes("$ gm scene test001 scene12\nok\n"), "gm scene 应答逐行进应答区");
assert.ok(!pv.log.textContent.includes("（无应答）"), "无「（无应答）」");

console.log("all assertions passed");
process.exit(0); // mount 的玩家名单轮询 setInterval 会吊住事件循环
