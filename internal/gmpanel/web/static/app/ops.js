// ops.js —— 操作反馈封装（POST → toast → 触发全局刷新钩子）。
// 独立于 main.js，避免 panels ↔ main 循环依赖。

import { toast } from "./ui.js";

let refreshHook = () => {};

/** main.js 启动时注册全局刷新（tick）。 */
export function setRefreshHook(fn) {
  refreshHook = fn || (() => {});
}

/** 统一执行写操作：按钮态 + toast + 完成后立即刷新一次。 */
export async function runOp(promise, label) {
  const status = document.querySelector("#op-status");
  if (status) status.textContent = (label || "操作") + "执行中……";
  let result;
  try {
    result = await promise;
  } catch (err) {
    result = { ok: false, message: err && err.message ? err.message : String(err) };
  }
  if (status) status.textContent = "";
  toast((result.ok ? "✔ " : "✘ ") + (result.ok ? result.message : (label ? label : "") + result.message),
    result.ok ? "ok" : "err", result.ok ? 3600 : 6000);
  refreshHook();
  return result;
}
