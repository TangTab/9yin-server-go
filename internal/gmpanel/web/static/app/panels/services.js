// services.js —— 服务管理：全量表格 + 单服启停/重启 + 全集群启停。

import { h, confirmDanger, stateTag } from "../ui.js";
import { runOp } from "../ops.js";
import * as api from "../api.js";

function opBtn(label, theme, danger, confirmText, fn) {
  const btn = h("ax-btn", { label, type: "plain", theme: theme || "" });
  btn.addEventListener("click", async () => {
    if (danger && !(await confirmDanger(confirmText))) return;
    runOp(fn(), label + "：");
  });
  return btn;
}

function row(svc) {
  const ops = h("div", { style: "display:flex;gap:6px;" });
  if (svc.state === "stopped") {
    ops.appendChild(opBtn("启动", "success", false, "", () => api.svcStart(svc.name)));
  } else {
    ops.appendChild(opBtn("停止", "error", true,
      "确定停止 " + svc.name + "？（" + (svc.canGrace ? "echo quit 优雅停" : "无优雅停面，将强杀") + "）",
      () => api.svcStop(svc.name)));
    ops.appendChild(opBtn("重启", "warning", true, "确定重启 " + svc.name + "？", () => api.svcRestart(svc.name)));
  }
  return h("tr", {}, [
    h("td", {}, [svc.name]),
    h("td", { class: "wrap" }, [svc.title]),
    h("td", {}, [svc.exe + (svc.args ? " " + svc.args : "")]),
    h("td", {}, [":" + svc.probePort]),
    h("td", {}, [stateTag(svc.state)]),
    h("td", {}, [svc.pid ? String(svc.pid) : "-"]),
    h("td", {}, [svc.canGrace ? "quit" : "强杀"]),
    h("td", {}, [ops]),
  ]);
}

export function mount(view) {
  const tbl = h("table", { class: "tbl" }, [
    h("tr", {}, ["服务", "说明", "可执行", "探活口", "状态", "PID", "停机方式", "操作"]
      .map((t) => h("th", {}, [t]))),
  ]);
  view.appendChild(h("section", { class: "panel" }, [
    h("h2", {}, ["服务管理"]),
    h("div", { class: "panel-desc" }, [
      "托管 = 面板拉起的子进程（日志落盘 " + "logs/&lt;name&gt;-日期.log）；外部 = bat/手工启动（观察模式，停止走 taskkill）。",
    ]),
    h("div", { class: "toolbar" }, [
      opBtn("▶ 启动全部（依赖序）", "info", true, "按依赖序拉起全部服务？", api.svcStartAll),
      opBtn("■ 停止全部（逆序）", "error", true, "逆序停止全部服务？在线玩家全部中断。", api.svcStopAll),
    ]),
    tbl,
  ]));

  function refresh(data) {
    if (!data || !data.services) return;
    while (tbl.rows.length > 1) tbl.deleteRow(1);
    for (const svc of data.services) tbl.appendChild(row(svc));
  }

  return { refresh };
}
