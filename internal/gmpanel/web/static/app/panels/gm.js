// gm.js —— GM 命令入口：踢人 / 封号·解封 / 开关服。

import { h, confirmDanger } from "../ui.js";
import { runOp } from "../ops.js";
import * as api from "../api.js";

function form(title, desc, fields, submitLabel, theme, danger, confirmText, onSubmit) {
  const inputs = {};
  const box = h("div", { class: "svc-card" }, [
    h("div", { class: "hd" }, [h("span", { class: "nm" }, [title])]),
    h("div", { class: "sub" }, [desc]),
    h("div", { class: "toolbar" }),
  ]);
  const bar = box.querySelector(".toolbar");
  for (const [key, placeholder] of Object.entries(fields)) {
    inputs[key] = h("input", { type: "text", placeholder });
    bar.appendChild(inputs[key]);
  }
  const btn = h("ax-btn", { label: submitLabel, theme: theme || "" });
  btn.addEventListener("click", async () => {
    const values = {};
    for (const [key] of Object.entries(inputs)) values[key] = inputs[key].value.trim();
    if (Object.values(values).some((v) => !v)) return;
    if (danger && !(await confirmDanger(confirmText(values)))) return;
    runOp(onSubmit(values), submitLabel + "：");
  });
  bar.appendChild(btn);
  return box;
}

export function mount(view) {
  const grid = h("div", { class: "grid" });
  grid.appendChild(form(
    "踢下线", "按角色名或账号（封号联动同款通道：world → entry 断开帧，存档走正常下线链）",
    { name: "角色名 / 账号" }, "踢下线", "error", true,
    (v) => "确定将 " + v.name + " 踢下线？",
    (v) => api.gmKick(v.name),
  ));
  grid.appendChild(form(
    "封号", "dc_accounts.status=0 + 清 logged 会话 + 联动踢下线（在线时）",
    { account: "账号" }, "封号", "error", true,
    (v) => "确定封禁账号 " + v.account + "？该账号将无法登录。",
    (v) => api.gmBan(v.account),
  ));
  grid.appendChild(form(
    "解封", "dc_accounts.status=1（恢复登录门）",
    { account: "账号" }, "解封", "success", false, "",
    (v) => api.gmUnban(v.account),
  ));
  const worldOps = h("div", { class: "svc-card" }, [
    h("div", { class: "hd" }, [h("span", { class: "nm" }, ["世界开关服"])]),
    h("div", { class: "sub" }, ["world echo open/close：开服放行登录；关服拒新登录（存量不踢）。"]),
    h("div", { class: "ops" }, [
      (() => {
        const b = h("ax-btn", { label: "开服", theme: "info" });
        b.addEventListener("click", () => runOp(api.worldOpen(), "开服："));
        return b;
      })(),
      (() => {
        const b = h("ax-btn", { label: "关服（拒新登录）", type: "plain", theme: "warning" });
        b.addEventListener("click", () => runOp(api.worldClose(), "关服："));
        return b;
      })(),
    ]),
  ]);
  grid.appendChild(worldOps);

  view.appendChild(h("section", { class: "panel" }, [
    h("h2", {}, ["GM 命令"]),
    h("div", { class: "panel-desc" }, [
      "踢人走 world→entry 断开帧（官方拒登同路径，存档/登出走正常链）；封号直写 MySQL 并清会话门。",
      "发物品/改档属重 GM 动作，V1 未开放（服务端 SaveRole 接口已备，按需扩展）。",
    ]),
    grid,
  ]));

  return { refresh: () => {} };
}
