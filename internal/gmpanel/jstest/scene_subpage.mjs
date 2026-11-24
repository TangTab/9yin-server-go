// scene_subpage.mjs —— GM 测试台「传送与场景」子页逻辑回归（由 Go 测试
// jstest_test.go 在 Node 可用时调起；场景数据取自真起的面板测试服务）。
//
// 最小假 DOM + fetch 桩，真跑 panels/test.js 的 mount(view, "scene")：加载
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

const appDir = process.env.GM_JS_APP;
const scenesFile = process.env.GM_JS_SCENES;
if (!appDir || !scenesFile) {
  throw new Error("GM_JS_APP / GM_JS_SCENES 未设置");
}
const scenes = JSON.parse(readFileSync(scenesFile, "utf8")).scenes;
const STATIC_PRESETS = 23; // panels/test.js 的 SCENE_PRESETS 条数（回退表）

class Text {
  constructor(text) {
    this.nodeType = 3;
    this.text = String(text);
  }
}

class El {
  constructor(tag) {
    this.nodeType = 1;
    this.tagName = tag.toUpperCase();
    this.children = [];
    this.attrs = {};
    this.listeners = {};
    this.style = {};
    this._text = "";
    this._class = "";
    this._props = {};
    this._forcedValue = undefined;
  }
  get className() {
    return this._class;
  }
  set className(v) {
    this._class = v;
  }
  setAttribute(k, v) {
    this.attrs[k] = String(v);
  }
  getAttribute(k) {
    return this.attrs[k];
  }
  appendChild(child) {
    this.children.push(child);
    child.parent = this;
    return child;
  }
  get textContent() {
    return this._text + this.children.map((c) => (c.nodeType === 3 ? c.text : c.textContent)).join("");
  }
  set textContent(v) {
    this._text = String(v);
    this.children = [];
    this._forcedValue = undefined;
  }
  set innerHTML(v) {
    this.textContent = v;
  }
  addEventListener(type, fn) {
    (this.listeners[type] ||= []).push(fn);
  }
  fire(type) {
    for (const fn of this.listeners[type] || []) fn({ target: this });
  }
  // <select>：显式设值则选中匹配 option（无匹配清空），否则跟随首个 option。
  get value() {
    if (this.tagName !== "SELECT") return this._props.value ?? this.attrs.value ?? "";
    if (this._forcedValue !== undefined) return this._forcedValue;
    const first = this.options()[0];
    return first ? first.attrs.value : "";
  }
  set value(v) {
    if (this.tagName === "SELECT") {
      const has = this.options().some((o) => o.attrs.value === String(v));
      this._forcedValue = has ? String(v) : "";
      return;
    }
    this._props.value = String(v);
  }
  options() {
    const out = [];
    const walk = (n) => {
      for (const c of n.children) {
        if (c.nodeType === 3) continue;
        if (c.tagName === "OPTION") out.push(c);
        walk(c);
      }
    };
    walk(this);
    return out;
  }
  find(pred) {
    let hit = null;
    const walk = (n) => {
      for (const c of n.children) {
        if (c.nodeType === 3 || hit) continue;
        if (pred(c)) hit = c;
        walk(c);
      }
    };
    walk(this);
    return hit;
  }
  collect(tag, out = []) {
    const walk = (n) => {
      for (const c of n.children) {
        if (c.nodeType === 3) continue;
        if (c.tagName === tag) out.push(c);
        walk(c);
      }
    };
    walk(this);
    return out;
  }
}

globalThis.document = {
  createElement: (tag) => new El(tag),
  createTextNode: (text) => new Text(text),
  querySelector: () => null,
  contains: () => true,
};

const posted = [];
let scenesAvailable = true;
globalThis.fetch = async (url, init = {}) => {
  if (url === "/api/gm/scenes") {
    return scenesAvailable
      ? { ok: true, status: 200, json: async () => ({ scenes }) }
      : { ok: false, status: 404, json: async () => ({}) };
  }
  if (url === "/api/gm/raw") {
    posted.push(JSON.parse(init.body || "{}").line);
    return { ok: true, status: 200, json: async () => ({ lines: [">ok"] }) };
  }
  return { ok: false, status: 404, json: async () => ({}) };
};

const testURL = pathToFileURL(path.join(appDir, "panels", "test.js")).href;
const mod = await import(testURL);
const settle = () => new Promise((r) => setTimeout(r, 20));
// 最近一次 gm scene 命令行（点击后还会跟一条 refreshState 的 gm state）。
const lastSceneCmd = () => posted.filter((l) => l.startsWith("gm scene ")).pop();

async function mountScene(page) {
  const view = new El("div");
  page.mount(view, "scene");
  await settle(); // 等场景表加载
  const [target, presetSel, wpSel] = view.collect("SELECT");
  const who = new El("option"); // 真实界面从在线列表选；这里放一个可选目标
  who.setAttribute("value", "test001");
  target.appendChild(who);
  target.value = "test001";
  // 场景工具栏的四个输入：覆盖x/y/z、朝向（最后四个 input）
  const [sx, sy, sz, so] = view.collect("INPUT").slice(-4);
  const btn = view.find((n) => n.tagName === "AX-BTN" && n.attrs.label === "gm scene（跨场景切换）");
  assert.ok(presetSel && wpSel && btn, "scene toolbar built");
  return { presetSel, wpSel, sx, sy, sz, so, btn };
}

const labels = (group) => group.children.filter((c) => c.tagName === "OPTION").map((o) => o.textContent);

// ---- 1. 接口数据到达：按类分组、全部场景 + 表外补充键、保留原选中项 ----
const ui = await mountScene(mod);
const groups = ui.presetSel.children.filter((c) => c.nodeType === 1 && c.tagName === "OPTGROUP");
assert.deepEqual(groups.map((g) => g.attrs.label), ["主城", "出生镇", "门派", "江湖", "其它"]);
assert.equal(ui.presetSel.options().length, scenes.length + 1, "options = 场景表 + 新手引导");
assert.equal(ui.presetSel.value, "born01", "previous selection kept");
const [, , school, jianghu, other] = groups.map(labels);
assert.ok(jianghu.includes("凌霄城 scene06") && jianghu.includes("凌霄城 scene25"), "同组重名追加 key");
assert.ok(school.includes("school21") && jianghu.includes("scene36"), "无中文名回退 key");
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

console.log("all assertions passed");
process.exit(0); // mount 的玩家名单轮询 setInterval 会吊住事件循环
