// pending.js —— 尚未迁移到 AXUI 版的分组占位：说明去向 + 跳旧版对应板块。

import { h } from "../ui.js";
import { GROUPS } from "./registry.js";

export function render(view, { sectionId }) {
  const found = GROUPS.find((group) => group.sections.includes(sectionId));
  const title = found ? found.name : "该分组";
  view.appendChild(
    h("section", { class: "panel" }, [
      h("h2", {}, ["迁移中"]),
      h("p", { class: "desc" }, [
        "「" + title + "」分组尚未迁移到新版页面。迁移期间旧版控制台保持全量可用，可点击下方按钮继续使用旧版。",
      ]),
      h("div", { class: "actions-row" }, [
        h("a", { href: "/console.html", style: "color:var(--gm-blue);font-size:14px;" }, ["→ 打开旧版控制台"]),
      ]),
    ])
  );
}
