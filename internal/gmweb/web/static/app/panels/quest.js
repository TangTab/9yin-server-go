// quest.js —— 组9「任务与身份」：场景任务 / 暮色剧情 / 九阴志 / 身份系统 / 门派声望。

import { h, panelSection, numField, actionBtn, cmdBtn, hint } from "../ui.js";
import { catalog, command } from "../api.js";
import { refreshNow } from "../state.js";

/* ---------------- 场景任务验证 ---------------- */

function renderSceneTask(host) {
  const { section, body } = panelSection({
    id: "sec-scene-task",
    title: "场景任务验证",
    desc: "上半：成都厨师钱安康场景任务（真实 SelectMenu 创建任务记录）。下半：全场景任务全流程验证（任务 ID 留空时作用于最早的活跃任务）。",
  });
  body.appendChild(
    h("div", { class: "actions-row" }, [
      cmdBtn({ label: "开始成都场景任务测试 791", theme: "prim", action: "scene_task_start", section: "场景任务" }),
      cmdBtn({ label: "直接完成采集并交付", action: "scene_task_submit_ready", section: "场景任务" }),
      cmdBtn({ label: "清理场景任务测试", danger: true, action: "scene_task_remove", section: "场景任务" }),
    ])
  );
  const taskId = numField({ label: "任务ID", value: 791, min: 0, width: "90px" });
  const authored = (label, action) =>
    actionBtn({
      label,
      theme: "",
      build: () => ({ action, taskId: taskId.get() ?? 0 }),
      okText: label,
      section: "场景任务",
    });
  body.appendChild(
    h("div", { class: "field-row", style: "margin-top:12px;" }, [
      taskId.node,
      authored("接取任务", "authored_task_start"),
      authored("推进全部步骤（可交付）", "authored_task_submit_ready"),
      authored("清理任务", "authored_task_remove"),
      authored("补发任务奖励", "authored_task_reward"),
    ])
  );
  host.appendChild(section);
}

/* ---------------- 暮色剧情验证 ---------------- */

function renderTwilight(host) {
  const { section, body } = panelSection({
    id: "sec-twilight",
    title: "暮色剧情验证",
    desc: "机关任务：庙内固定点生成钥匙箱，真实确认后获得旧钥匙。NPC 护送：生成躲藏村民，确认后按服务端移动帧走向终点。剧情归位：按 creator 表固定坐标全量下发 clone003 摆位（628 个），波次推到最大。难度档：影响 BOSS 出生 buff 与死亡掉宝箱矩阵。",
  });
  const group = (title, buttons) => {
    const row = h("div", { class: "actions-row" }, buttons);
    return h("div", { class: "matrix-group", style: "margin-bottom:10px;" }, [h("div", { class: "matrix-label" }, [title]), row]);
  };
  body.appendChild(
    group("机关任务", [
      cmdBtn({ label: "开始灰尘盒子机关任务", theme: "prim", action: "twilight_mechanism_start", section: "暮色剧情" }),
      cmdBtn({ label: "清理机关任务", action: "twilight_mechanism_remove", section: "暮色剧情" }),
    ])
  );
  body.appendChild(
    group("NPC 定点移动", [
      cmdBtn({ label: "开始点击 NPC 定点移动", theme: "prim", action: "twilight_npc_move_start", section: "暮色剧情" }),
      cmdBtn({ label: "清理移动 NPC", action: "twilight_npc_move_remove", section: "暮色剧情" }),
    ])
  );
  body.appendChild(
    group("剧情坐标归位", [
      cmdBtn({ label: "开始暮色剧情归位验证", theme: "prim", action: "twilight_stages_start", section: "暮色剧情" }),
      cmdBtn({ label: "重置/清理验证", danger: true, action: "twilight_stages_remove", section: "暮色剧情" }),
    ])
  );
  const difficulty = [1, 2, 3, 4].map((level) =>
    actionBtn({
      label: "难度：" + ["", "普通", "困难", "地狱", "修罗"][level],
      theme: "",
      build: () => ({ action: "twilight_difficulty", level }),
      okText: "难度已切换",
      section: "暮色剧情",
    })
  );
  body.appendChild(group("副本难度档", difficulty));
  host.appendChild(section);
}

/* ---------------- 九阴志全局门控 ---------------- */

function renderNinjiuzhi(host) {
  const { section, body } = panelSection({
    id: "sec-ninjiuzhi",
    title: "九阴志全局门控",
    desc: "九阴志阶段是全服唯一的单向刻度：1=1内 … 9=6内。阶段只增不减；需要回退用「重置阶段」。每次生效向全服在线连接广播阶段帧。",
  });
  const stage = h("ax-select", {
    content: JSON.stringify([
      { label: "1 — 1内", value: "1" }, { label: "2 — 2内", value: "2" }, { label: "3 — 3内", value: "3" },
      { label: "4 — 4内", value: "4" }, { label: "5 — 5内", value: "5" }, { label: "6 — 5内后期", value: "6" },
      { label: "7 — 6内前期", value: "7" }, { label: "8 — 6内中期", value: "8" }, { label: "9 — 6内", value: "9" },
    ]),
    style: "width:140px;",
  });
  const note = hint();
  const setResult = (result, text) => note.set((result.ok ? "✔ " : "✘ ") + text + "：" + result.message, result.ok);
  body.appendChild(
    h("div", { class: "field-row" }, [
      h("label", {}, ["目标阶段"]), stage,
      actionBtn({
        label: "设置阶段（单向递增）",
        build: () => ({ action: "ninjiuzhi_set", stage: Number(stage.value || 1) }),
        okText: "阶段已设置",
        section: "九阴志",
      }),
      cmdBtn({ label: "重置阶段", danger: true, action: "ninjiuzhi_reset", section: "九阴志" }),
    ])
  );
  body.appendChild(
    h("div", { class: "field-row" }, [
      h("span", { class: "hint", style: "margin:0;" }, ["事件点注入（击杀/任务累计到阈值自动推进阶段）："]),
      cmdBtn({ label: "击杀事件 +1", action: "ninjiuzhi_event", params: { eventKind: "kill" }, section: "九阴志" }),
      cmdBtn({ label: "任务事件 +1", action: "ninjiuzhi_event", params: { eventKind: "task" }, section: "九阴志" }),
      cmdBtn({
        label: "查询状态",
        theme: "",
        action: "ninjiuzhi_status",
        section: "九阴志",
        onDone: (result) => setResult(result, "状态查询"),
      }),
    ])
  );
  body.appendChild(note.node);
  host.appendChild(section);
}

/* ---------------- 身份系统 ---------------- */

function renderOrigin(host) {
  const { section, body } = panelSection({
    id: "sec-origin",
    title: "身份系统（一键激活全部）",
    desc: "把全部身份一次性加入【已激活】集合（重登/切图保持），【未领取】状态不变；玩家在身份界面领奖时才真发奖（物品/称号/声望）。数据源 origin.ini（1197 条），C2S 协议 700/702/706 已接。",
  });
  const state = h("span", { class: "hint" }, ["状态读取中……"]);
  async function loadState() {
    try {
      const data = await catalog("/api/origin/status");
      state.textContent = data.loaded
        ? "已加载 " + data.total + " 个身份（C2S 协议 700/701/702/705/706 已挂载）"
        : "身份数据未加载：缺少 data/origin/origins.json";
    } catch (err) {
      state.textContent = "读取失败：" + (err.message || err);
    }
  }
  body.appendChild(
    h("div", { class: "actions-row" }, [
      cmdBtn({
        label: "一键激活全部身份",
        theme: "prim",
        action: "origin_activate_all",
        section: "身份系统",
        onDone: () => loadState(),
      }),
      state,
    ])
  );
  loadState();
  host.appendChild(section);
}

/* ---------------- 门派声望 ---------------- */

function renderRepute(host) {
  const { section, body } = panelSection({
    id: "sec-repute",
    title: "门派声望（自定义）",
    desc: "覆盖全部八大门派、江湖势力与隐世宗门（含隐士、江湖声望）：直接写入角色的声望进度并即时刷新客户端声望表。",
  });
  const nameSelect = h("ax-select", { search: true, full: true, "max-height": "260", placeholder: "选择声望目标（加载中…）" });
  const value = numField({ label: "点数", value: 5000, min: 0, max: 9999999, width: "110px" });
  const note = hint();
  async function load() {
    try {
      const data = await catalog("/api/repute/status");
      const options = [{ label: "全部（含隐士、江湖）", value: "all" }];
      for (const row of data.reputes || []) {
        options.push({ label: row.display + " [" + row.name + "] 上限" + row.maxValue, value: row.name });
      }
      nameSelect.set({ key: "content", value: JSON.stringify(options) });
      note.set(data.loaded ? "已加载 " + (data.reputes || []).length + " 项门派/势力声望" : "声望表未加载", data.loaded);
    } catch (err) {
      note.set("读取失败：" + (err.message || err), false);
    }
  }
  const apply = (mode, label) =>
    actionBtn({
      label,
      build: () => ({ action: "repute_apply", reputeName: nameSelect.value || "all", reputeMode: mode, reputeValue: value.get() ?? 0 }),
      okText: label,
      section: "门派声望",
    });
  body.appendChild(h("div", { class: "field-row" }, [nameSelect]));
  body.appendChild(
    h("div", { class: "actions-row", style: "margin-top:8px;" }, [
      value.node,
      apply("set", "设定为该值"),
      apply("add", "累加该值"),
      actionBtn({ label: "刷新声望列表", theme: "", build: () => (load(), null) }),
    ])
  );
  body.appendChild(note.node);
  load();
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = { "scene-task": renderSceneTask, twilight: renderTwilight, ninjiuzhi: renderNinjiuzhi, origin: renderOrigin, repute: renderRepute };
  (renderers[sectionId] || renderSceneTask)(host);
}
