// debug_subpage.mjs —— GM 测试台「调试工具」子页逻辑回归（由 Go 测试
// jstest_test.go 在 Node 可用时调起）。
//
// 最小假 DOM（fakedom.mjs）+ fetch 桩，真跑 panels/test.js 的
// mount(view, "debug")：怒气 / 移动参数两组控件生成的 gm rage / gm move
// 命令行，须与 world echo 方言（gmcmd.go gmdebug.go）逐字对得上——命令行
// 两端各写一份，面板拼错一个词，服务端只会回用法行。环境变量：
//
//	GM_JS_APP=<目录>      含 ui.js / api.js / panels/test.js（从嵌入资源拷出）
//
// 断言不过即未捕获异常退出非零；通过打印 "all assertions passed"。
import { pathToFileURL } from "node:url";
import path from "node:path";
import assert from "node:assert/strict";
import { El, installDOM } from "./fakedom.mjs";

const appDir = process.env.GM_JS_APP;
if (!appDir) {
  throw new Error("GM_JS_APP 未设置");
}

installDOM();

const reply = (body) => ({ ok: true, status: 200, json: async () => body });
const fail = () => ({ ok: false, status: 404, json: async () => ({}) });
const posted = []; // 全部 gm 命令行
let playersAvailable = true; // 第 6 节起 /api/players 不给数据（无目标）
globalThis.fetch = async (url, init = {}) => {
  if (url === "/api/players") {
    return playersAvailable
      ? reply({ players: [{ account: "test001", name: "岳绮丝", state: "Online" }] })
      : fail();
  }
  if (url === "/api/gm/raw") {
    const line = JSON.parse(init.body || "{}").line;
    posted.push(line);
    return reply({ lines: ["gm ok: " + line] });
  }
  return fail();
};

const testURL = pathToFileURL(path.join(appDir, "panels", "test.js")).href;
const mod = await import(testURL);
const settle = () => new Promise((r) => setTimeout(r, 20));

async function mountDebug() {
  const view = new El("div");
  mod.mount(view, "debug");
  await settle(); // 等玩家名单
  const [target] = view.collect("SELECT");
  const button = (label) =>
    view.find((n) => (n.tagName === "AX-BTN" && n.attrs.label === label) || (n.tagName === "BUTTON" && n.textContent === label));
  const inputs = view.collect("INPUT");
  const byPlaceholder = (prefix) => inputs.find((n) => (n.attrs.placeholder || "").startsWith(prefix));
  return {
    view, target, button,
    msgId: byPlaceholder("mXXX"), msgArgs: byPlaceholder("int 参数"),
    rage: byPlaceholder("怒气"), regen: byPlaceholder("自动回复"),
    speed: byPlaceholder("移动速度"), gravity: byPlaceholder("下落重力"),
    log: view.find((n) => n._class === "log-box"),
  };
}

// ---- 1. 装配：六个输入框齐全，怒气 / 移动参数带缺省值 ----
const ui = await mountDebug();
assert.equal(ui.target.value, "岳绮丝", "目标 = 在线名单的名字");
for (const [name, node] of Object.entries({
  msgId: ui.msgId, msgArgs: ui.msgArgs, rage: ui.rage, regen: ui.regen, speed: ui.speed, gravity: ui.gravity,
})) {
  assert.ok(node, "输入框 " + name);
}
assert.deepEqual([ui.rage.value, ui.regen.value, ui.speed.value, ui.gravity.value], ["100", "0", "3.9", "9.8"], "缺省值");
assert.deepEqual(
  ui.view.collect("SPAN").filter((n) => n._class === "group-label").map((n) => n.textContent),
  ["怒气", "移动参数"],
  "分组标签",
);

// ---- 2. 原有探针不受影响 ----
ui.msgId.value = "366";
ui.msgArgs.value = "7";
await ui.button("gm msg（自定义帧探针）").fire("click");
await settle();
assert.equal(posted.pop(), "gm msg 岳绮丝 366 7");
ui.msgId.value = "";
posted.length = 0;
await ui.button("gm msg（自定义帧探针）").fire("click");
assert.equal(posted.length, 0, "msgId 空不发");
await ui.button("gm flush（重放 ReadyFrames）").fire("click");
await settle();
assert.equal(posted.pop(), "gm flush 岳绮丝");

// ---- 3. 怒气：设置怒气 = 怒气值 + 自动回复/秒（留空 = 不动当前速率） ----
await ui.button("设置怒气").fire("click");
await settle();
assert.equal(posted.pop(), "gm rage 岳绮丝 100 0", "缺省：设 100、回复 0");
ui.rage.value = "50";
ui.regen.value = "5";
await ui.button("设置怒气").fire("click");
await settle();
assert.equal(posted.pop(), "gm rage 岳绮丝 50 5");
ui.regen.value = "";
await ui.button("设置怒气").fire("click");
await settle();
assert.equal(posted.pop(), "gm rage 岳绮丝 50", "回复留空 = 不带速率参数");
ui.regen.value = "12";
await ui.button("设置怒气").fire("click");
await settle();
assert.equal(posted.pop(), "gm rage 岳绮丝 50 12", "回复速率不设上限，原样下发");
ui.regen.value = "3";
await ui.button("回满").fire("click");
await settle();
assert.equal(posted.pop(), "gm rage 岳绮丝 max 3", "回满走服务端上限");
ui.regen.value = "";
await ui.button("回满").fire("click");
await settle();
assert.equal(posted.pop(), "gm rage 岳绮丝 max");
await ui.button("读取").fire("click");
await settle();
assert.equal(posted.pop(), "gm rage 岳绮丝", "裸 gm rage = 读当前值");
ui.rage.value = "";
posted.length = 0;
await ui.button("设置怒气").fire("click");
assert.equal(posted.length, 0, "怒气留空不发");

// ---- 4. 移动参数：速度 / 重力各自应用，留空不发，恢复默认 ----
ui.speed.value = "25";
await ui.button("应用移动速度").fire("click");
await settle();
assert.equal(posted.pop(), "gm move 岳绮丝 speed 25");
ui.gravity.value = "30";
await ui.button("应用下落重力").fire("click");
await settle();
assert.equal(posted.pop(), "gm move 岳绮丝 gravity 30");
ui.speed.value = "";
ui.gravity.value = "";
posted.length = 0;
await ui.button("应用移动速度").fire("click");
await ui.button("应用下落重力").fire("click");
assert.equal(posted.length, 0, "数值留空不发");
await ui.button("恢复默认").fire("click");
await settle();
assert.equal(posted.pop(), "gm move 岳绮丝 reset");

// ---- 5. 应答输出：命令与 world 的逐行应答进应答区 ----
assert.ok(ui.log.textContent.includes("$ gm rage 岳绮丝 50 5\ngm ok: gm rage 岳绮丝 50 5\n"), "gm rage 应答进应答区");
assert.ok(ui.log.textContent.includes("$ gm move 岳绮丝 speed 25\ngm ok: gm move 岳绮丝 speed 25\n"), "gm move 应答进应答区");

// ---- 6. 无目标（名单取不到）：新加的按钮一律不发 ----
playersAvailable = false;
const bare = await mountDebug();
assert.ok(!bare.target.value, "名单取不到：无目标");
posted.length = 0;
for (const label of ["设置怒气", "回满", "读取", "应用移动速度", "应用下落重力", "恢复默认"]) {
  assert.ok(bare.button(label), "按钮 " + label);
  await bare.button(label).fire("click");
}
await settle();
assert.equal(posted.length, 0, "无目标不发命令");

console.log("all assertions passed");
process.exit(0); // mount 的玩家名单轮询 setInterval 会吊住事件循环
