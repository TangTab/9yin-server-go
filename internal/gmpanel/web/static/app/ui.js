// ui.js —— DOM 小工具与全局反馈（范式照 wt1653 gmweb/ui.js 精简）。

/** 创建元素；attrs 支持 class/text，children 接受节点/字符串/数组/null。
 *  ax-* 组件未显式给 size 时统一压到 sm —— 框架默认 md 比工具栏原生输入框
 *  (13px/5px 内边距) 大一号，混排突兀（角色查询条按钮高出输入框一圈）。 */
export function h(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  if (tag.startsWith("ax-") && attrs && !("size" in attrs)) attrs.size = "sm";
  for (const [key, value] of Object.entries(attrs || {})) {
    if (value === null || value === undefined || value === false) continue;
    if (key === "class") node.className = value;
    else if (key === "text") node.textContent = value;
    else node.setAttribute(key, value === true ? "" : value);
  }
  for (const child of [].concat(children)) {
    if (child === null || child === undefined || child === false) continue;
    if (typeof child === "string" || typeof child === "number") {
      node.appendChild(document.createTextNode(String(child)));
    } else {
      node.appendChild(child);
    }
  }
  return node;
}

/** 右下角 toast（theme: info/ok/err）。 */
export function toast(message, theme = "info", ms = 3600) {
  const stack = document.querySelector("#toast-stack");
  if (!stack) return;
  const node = h("div", { class: "toast " + (theme === "info" ? "" : theme) }, [message]);
  stack.appendChild(node);
  requestAnimationFrame(() => node.classList.add("show"));
  setTimeout(() => {
    node.classList.remove("show");
    setTimeout(() => node.remove(), 250);
  }, ms);
}

/** 危险操作二次确认（ax confirm 优先，异常回退 window.confirm）。 */
export async function confirmDanger(text) {
  try {
    const mod = await import("/static/ax.esm.js");
    const result = await mod.confirm({ content: text, theme: "error" });
    return result === true || result === "true" || result?.button === "ok" || result === undefined;
  } catch {
    return window.confirm(text);
  }
}

/** unix 秒 → 本地时间串。 */
export function fmtTime(sec) {
  if (!sec) return "-";
  return new Date(sec * 1000).toLocaleString("zh-CN", { hour12: false });
}

/** 字节 → 可读（GiB/MiB）。 */
export function fmtBytes(n) {
  if (!n && n !== 0) return "-";
  if (n >= 1024 ** 3) return (n / 1024 ** 3).toFixed(1) + " GiB";
  if (n >= 1024 ** 2) return (n / 1024 ** 2).toFixed(0) + " MiB";
  return (n / 1024).toFixed(0) + " KiB";
}

/** 状态徽标节点。 */
export function stateTag(state) {
  const label = { running: "运行中", external: "外部", stopped: "已停止", online: "在线" }[state] || state;
  return h("span", { class: "state-tag " + state }, [label]);
}
