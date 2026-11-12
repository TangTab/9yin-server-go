// misc.js —— 「其他」兜底组：GM 邮件（迁移完成后本组可并入货币与物品）。

import { h, panelSection, numField, textField, actionBtn, toast } from "../ui.js";

function renderMail(host) {
  const { section, body } = panelSection({
    id: "sec-mail",
    title: "GM 邮件",
    desc: "以系统邮件发自定义信息或物品附件；正文与物品附件至少填一项。收件角色留空 = 发给当前选中会话的角色；填角色名可发给其他玩家——在线实时推送，离线写入信箱存档。",
  });
  const role = textField({ label: "收件角色名（留空=当前会话角色）", placeholder: "留空发给当前角色", width: "200px" });
  const title = textField({ label: "标题", placeholder: "默认「系统邮件」", width: "200px" });
  const content = h("textarea", { rows: 3, placeholder: "正文（与物品附件至少填一项）" });
  content.style.cssText = "width:100%;max-width:460px;background:#11141b;border:1px solid var(--gm-line);border-radius:6px;color:var(--gm-text);padding:8px 10px;";
  const item = textField({ label: "物品 ConfigID（可选）", placeholder: "如 yaopin_00100", width: "180px" });
  const amount = numField({ label: "数量", value: 1, min: 1, max: 9999, width: "80px" });
  const send = actionBtn({
    label: "发送邮件",
    danger: true,
    confirm: "确定发送系统邮件？发送后不可撤回。",
    build: () => {
      const text = content.value.trim();
      const itemConfig = item.get();
      if (!text && !itemConfig) return (toast("正文与物品附件至少填一项", "warn"), null);
      const payload = {
        action: "send_mail",
        content: text,
        itemConfig: itemConfig || "",
        amount: amount.get() ?? 1,
      };
      if (role.get()) payload.roleName = role.get();
      if (title.get()) payload.title = title.get();
      return payload;
    },
    okText: "邮件已发送",
    section: "GM 邮件",
    onDone: (result) => {
      if (result.ok) {
        content.value = "";
        item.clear();
      }
    },
  });
  body.appendChild(h("div", { class: "field-row" }, [role.node, title.node]));
  body.appendChild(h("div", { class: "field-row" }, [content]));
  body.appendChild(h("div", { class: "field-row" }, [item.node, amount.node, send]));
  host.appendChild(section);
}

export function render(host) {
  renderMail(host);
}
