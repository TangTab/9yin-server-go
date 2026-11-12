// uipref.js —— 组10「界面与操作」：快捷栏位 / 快捷键方案。

import { h, panelSection, numField, textField, cmdBtn, actionBtn } from "../ui.js";

/* ---------------- 快捷栏位 ---------------- */

function renderShortcuts(host) {
  const { section, body } = panelSection({
    id: "sec-shortcuts",
    title: "快捷栏位",
    desc: "存档玩家的快捷栏布局（数字键 1-0、Ctrl+1-0、F1-F5 等）。每次登录自动读取存档；游戏内拖放技能时自动保存。",
  });
  body.appendChild(
    h("div", { class: "actions-row" }, [
      cmdBtn({ label: "保存当前快捷栏到文件", theme: "prim", action: "save_shortcuts", section: "快捷栏位" }),
    ])
  );
  host.appendChild(section);
}

/* ---------------- 快捷键方案 ---------------- */

function renderKeybind(host) {
  const { section, body } = panelSection({
    id: "sec-keybind",
    title: "快捷键方案",
    desc: "游戏内按键窗口点【确定/应用】或切换 3D/2.5D 时客户端自动上传整套配置并落盘，下次登录自动恢复。此处手动调试：绑定需按键 ID（7=跳跃、1=前进、2=后退…）。",
  });
  const keyId = numField({ label: "键位ID", value: 7, width: "70px" });
  const key = textField({ label: "按键", value: "Space", width: "90px" });
  const ctrl = h("input", { type: "checkbox" });
  const shift = h("input", { type: "checkbox" });
  const alt = h("input", { type: "checkbox" });
  for (const box of [ctrl, shift, alt]) box.style.cssText = "width:15px;height:15px;accent-color:var(--gm-gold);";
  const keybind = (args, label) =>
    actionBtn({
      label,
      theme: "",
      build: () => ({ action: "keybind", keyArgs: args }),
      okText: label,
      section: "快捷键方案",
    });
  body.appendChild(
    h("div", { class: "field-row" }, [
      keyId.node, key.node,
      h("label", {}, ["Ctrl", ctrl]), h("label", {}, ["Shift", shift]), h("label", {}, ["Alt", alt]),
      actionBtn({
        label: "绑定",
        build: () => {
          const flags = [];
          if (ctrl.checked) flags.push("ctrl");
          if (shift.checked) flags.push("shift");
          if (alt.checked) flags.push("alt");
          const args = ("bind " + (keyId.get() ?? 7) + " " + (key.get() || "Space") + " " + flags.join(" ")).trim();
          return { action: "keybind", keyArgs: args };
        },
        okText: "键位已绑定",
        section: "快捷键方案",
      }),
      actionBtn({
        label: "强制覆盖",
        build: () => {
          const flags = ["force"];
          if (ctrl.checked) flags.push("ctrl");
          if (shift.checked) flags.push("shift");
          if (alt.checked) flags.push("alt");
          return { action: "keybind", keyArgs: ("bind " + (keyId.get() ?? 7) + " " + (key.get() || "Space") + " " + flags.join(" ")).trim() };
        },
        okText: "键位已强制覆盖",
        section: "快捷键方案",
      }),
      actionBtn({
        label: "解绑",
        build: () => ({ action: "keybind", keyArgs: "unbind " + (keyId.get() ?? 7) }),
        okText: "键位已解绑",
        section: "快捷键方案",
      }),
    ])
  );
  body.appendChild(
    h("div", { class: "actions-row", style: "margin-top:10px;" }, [
      keybind("mode 0", "3D 模式"),
      keybind("mode 1", "2.5D 模式"),
      keybind("default", "恢复默认"),
      keybind("undo", "撤销"),
      keybind("redo", "重做"),
    ])
  );
  body.appendChild(
    h("div", { class: "actions-row" }, [
      keybind("save 1", "保存到方案1"),
      keybind("save 2", "保存到方案2"),
      keybind("load 1", "加载方案1"),
      keybind("load 2", "加载方案2"),
    ])
  );
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = { shortcuts: renderShortcuts, keybind: renderKeybind };
  (renderers[sectionId] || renderShortcuts)(host);
}
