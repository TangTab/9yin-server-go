// players.js —— 在线玩家（world echo plist）：名单 + 计数 + 踢下线。

import { h, fmtTime, confirmDanger, stateTag } from "../ui.js";
import { runOp } from "../ops.js";
import * as api from "../api.js";

function row(p) {
  const kick = h("ax-btn", { label: "踢下线", type: "plain", theme: "error" });
  kick.addEventListener("click", async () => {
    const who = p.name || p.account;
    if (!(await confirmDanger("确定将 " + who + " 踢下线？"))) return;
    runOp(api.gmKick(who), "踢下线 " + who + "：");
  });
  return h("tr", {}, [
    h("td", {}, [p.account || "-"]),
    h("td", {}, [p.name || "-"]),
    h("td", {}, [stateTag(p.state === "Online" ? "online" : p.state)]),
    h("td", {}, [p.scene ? String(p.scene) : "-"]),
    h("td", {}, [fmtTime(p.loginAt)]),
    h("td", {}, [fmtTime(p.activeAt)]),
    h("td", {}, [kick]),
  ]);
}

export function mount(view) {
  const count = h("div", { class: "panel-desc" }, ["读取中……"]);
  const tbl = h("table", { class: "tbl" }, [
    h("tr", {}, ["账号", "角色", "状态", "场景", "登录时间", "最近活跃", "操作"].map((t) => h("th", {}, [t]))),
  ]);
  view.appendChild(h("section", { class: "panel" }, [
    h("h2", {}, ["在线玩家"]),
    count,
    tbl,
  ]));

  async function refreshPlayers() {
    try {
      const data = await api.players();
      const list = data.players || [];
      count.textContent = "共 " + list.length + " 条在册（含登录中/排队态）；Online = 已进场。";
      while (tbl.rows.length > 1) tbl.deleteRow(1);
      if (!list.length) {
        tbl.appendChild(h("tr", {}, [h("td", { class: "tbl-empty", colspan: "7" }, ["（空——world 无在册玩家）"])]));
        return;
      }
      for (const p of list) tbl.appendChild(row(p));
    } catch (err) {
      count.textContent = "world 不可达：" + (err.message || err);
    }
  }

  refreshPlayers();
  // 玩家名单不占全局 tick（登录链低频），自轮询 5s。
  setInterval(refreshPlayers, 5000);

  return { refresh: () => {} };
}
