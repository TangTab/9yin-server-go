// logs.js —— 日志查看：服务选择 + tail + 暂停/自动滚底（3s 自轮询，
// 不占全局 tick——翻历史时暂停不受打扰）。

import { h } from "../ui.js";
import * as api from "../api.js";

export function mount(view) {
  let current = "world";
  let paused = false;
  let stick = true;

  const select = h("select", {});
  const tailSel = h("select", {}, [200, 400, 1000, 2000].map((n) => h("option", { value: String(n) }, [String(n)])));
  tailSel.value = "400";
  const box = h("div", { class: "log-box" });
  const fileLabel = h("span", { class: "meta" }, [""]);

  const pauseBtn = h("ax-btn", { label: "⏸ 暂停", type: "plain" });
  pauseBtn.addEventListener("click", () => {
    paused = !paused;
    pauseBtn.setAttribute("label", paused ? "▶ 继续" : "⏸ 暂停");
    if (!paused) refreshLog();
  });
  const topBtn = h("ax-btn", { label: "⤒ 回到底部", type: "plain" });
  topBtn.addEventListener("click", () => {
    stick = true;
    box.scrollTop = box.scrollHeight;
  });
  const clearBtn = h("ax-btn", { label: "清屏", type: "plain" });
  clearBtn.addEventListener("click", () => {
    box.textContent = "";
  });

  select.addEventListener("change", () => {
    current = select.value;
    stick = true;
    refreshLog();
  });
  box.addEventListener("scroll", () => {
    stick = box.scrollTop + box.clientHeight >= box.scrollHeight - 30;
  });

  view.appendChild(h("section", { class: "panel" }, [
    h("h2", {}, ["日志"]),
    h("div", { class: "toolbar" }, [
      select, h("span", { class: "meta" }, ["tail 行数"]), tailSel,
      pauseBtn, topBtn, clearBtn, h("span", { class: "grow" }), fileLabel,
    ]),
    box,
  ]));

  function fillServices(services) {
    const names = (services || []).map((s) => s.name);
    const known = Array.from(select.options).map((o) => o.value);
    if (names.join() === known.join()) return;
    select.innerHTML = "";
    for (const name of names) {
      select.appendChild(h("option", { value: name }, [name]));
    }
    if (names.includes(current)) {
      select.value = current;
    } else if (names.length) {
      current = names.includes("world") ? "world" : names[0];
      select.value = current;
    }
  }

  async function refreshLog() {
    try {
      const data = await api.logs(current, parseInt(tailSel.value, 10) || 400);
      fileLabel.textContent = data.file || "（暂无日志——服务启动后面板托管模式才落盘）";
      box.textContent = (data.lines || []).join("\n");
      if (stick) box.scrollTop = box.scrollHeight;
    } catch (err) {
      box.textContent = "日志读取失败：" + (err.message || err);
    }
  }

  function refresh(data) {
    fillServices(data && data.services);
  }

  refreshLog();
  setInterval(() => {
    if (!paused) refreshLog();
  }, 3000);

  return { refresh };
}
