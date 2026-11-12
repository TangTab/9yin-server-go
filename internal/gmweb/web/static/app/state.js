// state.js —— 会话快照的唯一持有者（顶栏角色卡与各面板都从这里订阅）。
// 轮询 5s（旧版 2.5s 降频；ETag 304 待服务端 P0）；页面不可见时暂停。

import { sessions } from "./api.js";

const POLL_MS = 5000;
const listeners = new Set();

let current = { sessions: [], reachable: false, updatedAt: null };

function notify() {
  for (const fn of listeners) {
    try {
      fn(current);
    } catch (err) {
      console.error("state listener error", err);
    }
  }
}

async function poll() {
  const list = await sessions();
  if (list === null) {
    if (current.reachable || current.updatedAt === null) {
      current = { sessions: [], reachable: false, updatedAt: new Date() };
      notify();
    }
    return;
  }
  current = { sessions: list, reachable: true, updatedAt: new Date() };
  notify();
}

/** 订阅快照变化；返回取消函数。回调立即以当前值触发一次。 */
export function subscribe(fn) {
  listeners.add(fn);
  fn(current);
  return () => listeners.delete(fn);
}

/** 手动触发一次即时刷新（命令回执后调用）。 */
export function refreshNow() {
  return poll();
}

/** 当前快照（只读引用）。 */
export function snapshot() {
  return current;
}

let timer = null;

function start() {
  if (timer) return;
  poll();
  timer = setInterval(() => {
    if (!document.hidden) poll();
  }, POLL_MS);
  document.addEventListener("visibilitychange", () => {
    if (!document.hidden) poll();
  });
}

start();
