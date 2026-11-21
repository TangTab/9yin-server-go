// overview.js —— 总览：统计卡 + 服务网格 + 全集群启停。

import { h, toast, fmtBytes, confirmDanger } from "../ui.js";
import { runOp } from "../ops.js";
import * as api from "../api.js";

function statCard(k, v, small) {
  return h("div", { class: "stat-card" }, [
    h("div", { class: "k" }, [k]),
    h("div", { class: "v" }, [v, small ? h("small", {}, [" " + small]) : null]),
  ]);
}

function svcCard(svc) {
  const dot = h("span", { class: "dot " + (svc.state === "running" ? "on" : svc.state === "external" ? "ext" : "off") });
  const card = h("div", { class: "svc-card" }, [
    h("div", { class: "hd" }, [dot, h("span", { class: "nm" }, [svc.name])]),
    h("div", { class: "sub" }, [
      svc.title + (svc.pid ? " · PID " + svc.pid : "") + " · 探活 :" + svc.probePort,
    ]),
    h("div", { class: "ops" }),
  ]);
  const ops = card.querySelector(".ops");
  if (svc.state === "stopped") {
    const btn = h("ax-btn", { label: "启动", type: "plain" });
    btn.addEventListener("click", () => runOp(api.svcStart(svc.name), "启动 " + svc.name + "："));
    ops.appendChild(btn);
  } else {
    const stop = h("ax-btn", { label: "停止", type: "plain", theme: "error" });
    stop.addEventListener("click", async () => {
      if (!(await confirmDanger("确定停止 " + svc.name + "？在线玩家会中断。"))) return;
      runOp(api.svcStop(svc.name), "停止 " + svc.name + "：");
    });
    ops.appendChild(stop);
  }
  return card;
}

export function mount(view) {
  const stats = h("div", { class: "stat-row" });
  const grid = h("div", { class: "grid" });
  const allOps = h("div", { class: "toolbar" }, [
    (() => {
      const b = h("ax-btn", { label: "▶ 启动全部服务", theme: "info" });
      b.addEventListener("click", async () => {
        if (!(await confirmDanger("按依赖序拉起全部服务？"))) return;
        toast("集群启动中……（日志服→世界→存储→公共→列表→网关→场景→拍卖→监控，约 10s）", "info", 6000);
        runOp(api.svcStartAll(), "启动全部：");
      });
      return b;
    })(),
    (() => {
      const b = h("ax-btn", { label: "■ 停止全部服务", theme: "error" });
      b.addEventListener("click", async () => {
        if (!(await confirmDanger("逆序停止全部服务？在线玩家会全部中断。"))) return;
        runOp(api.svcStopAll(), "停止全部：");
      });
      return b;
    })(),
  ]);
  view.appendChild(stats);
  view.appendChild(h("section", { class: "panel" }, [h("h2", {}, ["服务"]), grid]));
  view.appendChild(allOps);

  let lastServices = [];

  function refresh(data) {
    if (!data) return;
    lastServices = data.services || [];
    stats.innerHTML = "";
    const running = lastServices.filter((s) => s.state === "running").length;
    const external = lastServices.filter((s) => s.state === "external").length;
    stats.appendChild(statCard("世界状态", data.world && data.world.alive ? (data.world.state || "在线") : "离线"));
    stats.appendChild(statCard("在线玩家", data.online >= 0 ? String(data.online) : "-"));
    stats.appendChild(statCard("服务", running + external, "/" + lastServices.length + "（托管 " + running + " 外部 " + external + "）"));
    if (data.sys) {
      stats.appendChild(statCard("内存负载", Math.round((data.sys.memLoad || 0) * 100) + "%",
        "Go 运行时视角 " + fmtBytes(data.sys.totalPhy)));
    }
    grid.innerHTML = "";
    for (const svc of lastServices) grid.appendChild(svcCard(svc));
  }

  return { refresh };
}
