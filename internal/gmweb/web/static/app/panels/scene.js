// scene.js —— 组6「传送与场景」：常用传送 / 踢馆传送 / 自定义切图与瞬移 /
// 场景天气 / 副本 / 世界 BOSS。
// PRESETS / TIGUAN 数据逐条提取自旧版 console.html（脚本提取，无手抄误差）。

import { h, panelSection, cmdBtn, catalogPicker, numField, hint, toast } from "../ui.js";
import { catalog, command } from "../api.js";
import { refreshNow, snapshot } from "../state.js";

async function applyCommand(payload, okText) {
  const result = await command(payload);
  toast(result.ok ? "✔ " + okText : "✘ " + result.message, result.ok ? "ok" : "err");
  if (result.ok) refreshNow();
  return result;
}

export const PRESETS = [
  { label: "主城 / 新手村", items: [{ key: "yanjing", name: "燕京" }, { key: "suzhou", name: "苏州" }, { key: "jinling", name: "金陵" }, { key: "luoyang", name: "洛阳" }, { key: "chengdu", name: "成都" }, { key: "jimingyi", name: "鸡鸣驿" }, { key: "yanyuzhuang", name: "烟雨庄" }, { key: "qiandengzhen", name: "千灯镇" }, { key: "erengu", name: "恶人谷" }] },
  { label: "八大门派", items: [{ key: "jinyiwei", name: "锦衣卫" }, { key: "gaibang", name: "丐帮" }, { key: "junzitang", name: "君子堂" }, { key: "jilegu", name: "极乐谷" }, { key: "tangmen", name: "唐门" }, { key: "emei", name: "峨眉" }, { key: "wudang", name: "武当" }, { key: "shaolin", name: "少林" }] },
  { label: "江湖势力", items: [{ key: "yihuagong", name: "移花宫" }, { key: "taohuadao", name: "桃花岛" }, { key: "wugenmen", name: "无根门" }, { key: "xujia", name: "徐家庄" }, { key: "jinzhen", name: "金针沈家" }, { key: "wanshou", name: "万兽山庄" }, { key: "tianlun", name: "天轮寺" }, { key: "qingyi", name: "青衣楼" }] },
  { label: "隐世宗门", items: [{ key: "xuedao", name: "血刀门" }, { key: "huashan", name: "华山派" }, { key: "gumu", name: "古墓派" }, { key: "damo", name: "达摩宗" }, { key: "shenshui", name: "神水宫" }, { key: "changfeng", name: "长风镖局" }, { key: "nianluo", name: "念萝崖" }, { key: "wuxian", name: "无仙教" }, { key: "shenji", name: "神机营" }, { key: "xingmiao", name: "星渺阁" }, { key: "tianya", name: "天涯海阁" }, { key: "wanghui", name: "王回庄" }, { key: "wuxu", name: "无虚" }, { key: "shenjihui", name: "神机营" }, { key: "kunlunpai", name: "昆仑派" }, { key: "xingtianbao", name: "刑天堡" }, { key: "tianshanpai", name: "天山派" }, { key: "mingjiao", name: "明教" }] },
  { label: "野外世界场景", items: [{ key: "scene01", name: "游云十六洲" }, { key: "scene02", name: "蓬莱仙境" }, { key: "scene03", name: "泛黄平原" }, { key: "scene04", name: "襄阳坡" }, { key: "scene04b", name: "向阳坡北" }, { key: "scene05", name: "念萝坝" }, { key: "scene06", name: "凌霄城" }, { key: "scene07", name: "梅花岭" }, { key: "scene08", name: "塞外大漠" }, { key: "scene09", name: "塞北草原" }, { key: "scene10", name: "华山论剑" }, { key: "scene11", name: "快活岛" }, { key: "scene16", name: "蓬莱岛" }, { key: "scene17", name: "侠客岛" }, { key: "scene19", name: "无寂道" }, { key: "scene32", name: "永夜城" }, { key: "scene33", name: "鬼市" }, { key: "scene14", name: "场景14" }, { key: "scene15", name: "场景15" }] },
  { label: "江湖禁地 / 副本入口", items: [{ key: "jindi_klsg", name: "孔雀山庄" }, { key: "jindi_lmkc", name: "龙门客栈" }, { key: "jindi_qingy", name: "青云堡" }, { key: "jindi_yamg", name: "雁门关" }, { key: "jindi_sjd", name: "神机岛" }, { key: "jindi_xyg", name: "血影宫" }, { key: "jindi_ysg", name: "阎王阁" }, { key: "jindi_zyg", name: "紫云宫" }, { key: "jindi_shg", name: "森罗殿" }, { key: "jindi_ysl", name: "药王谷" }, { key: "jindi_xyu", name: "星月湖" }, { key: "jindi_hlg", name: "火麟洞" }, { key: "jindi_qlg", name: "七里谷" }, { key: "jindi_dmgc", name: "大漠鬼城" }, { key: "jindi_qcxi", name: "千苍峡" }, { key: "jindi_hygu", name: "横云谷" }, { key: "jindi_tsha", name: "天山海岛" }, { key: "jindi_ywf", name: "隐王峰" }, { key: "jindi_wxtg", name: "五行天宫" }, { key: "jindi_gshe", name: "鬼市蛇王" }, { key: "jindi_gwng", name: "鬼市鬼王" }, { key: "jindi_sdmz", name: "十大试炼" }, { key: "jindi_nzyhb", name: "逆转阴阳堡" }, { key: "jindi_czx", name: "潮之巷" }, { key: "jindi_csyt", name: "春山隐庭" }, { key: "jindi_mx", name: "溟墟" }, { key: "jindi_qyb", name: "竞速·青云堡" }, { key: "jindi_kssg", name: "竞速·孔雀山庄" }, { key: "jindi_lmkc2", name: "竞速·龙门客栈" }, { key: "jindi_dmgc2", name: "竞速·大漠鬼城" }] },
  { label: "跨服 / 活动场景", items: [{ key: "worldwar01", name: "世界战场(一)" }, { key: "worldwar02", name: "世界战场(二)" }, { key: "battle_myz", name: "牧云泽" }, { key: "battle_gmp", name: "古墓坡" }, { key: "battle_myz2", name: "牧云泽(二)" }, { key: "tianlundb", name: "天伦夺宝" }, { key: "schoolwar5", name: "门派战·锦衣卫" }, { key: "schoolwar6", name: "门派战·丐帮" }, { key: "schoolwar7", name: "门派战·君子堂" }, { key: "schoolwar8", name: "门派战·极乐谷" }, { key: "schoolwar9", name: "门派战·唐门" }, { key: "schoolwar10", name: "门派战·峨眉" }, { key: "schoolwar11", name: "门派战·武当" }, { key: "schoolwar12", name: "门派战·少林" }, { key: "schoolwar20", name: "门派战·明教" }, { key: "schoolwar22", name: "门派战·天山" }] },
  { label: "冒险场景 / 秘境", items: [{ key: "adv024", name: "万象门" }, { key: "adv023", name: "千机阁" }, { key: "adv032", name: "玄冰洞" }, { key: "adv011", name: "玄冥洞" }, { key: "adv033", name: "烈焰洞" }, { key: "adv100", name: "藏经阁" }, { key: "adv101", name: "天机阁" }, { key: "adv041", name: "碧波潭" }, { key: "adv045", name: "桃花谷" }, { key: "adv052", name: "寒冰洞" }, { key: "adv053", name: "火焰洞" }, { key: "adv054", name: "白雾谷" }, { key: "adv055", name: "青竹谷" }, { key: "adv057", name: "迷踪林" }, { key: "adv102", name: "天武阁" }, { key: "adv106", name: "望月台" }, { key: "adv120", name: "凌云台" }, { key: "adv121", name: "千雪峰" }, { key: "adv122", name: "望月崖" }, { key: "adv123", name: "观星台" }, { key: "adv148", name: "明月(冥岳)" }, { key: "adv152", name: "五行天宫" }, { key: "adv153", name: "净天宫" }, { key: "adv199", name: "宝藏岛" }] },
  { label: "江湖险境", items: [{ key: "jianghu_hole01", name: "险境一" }, { key: "jianghu_hole02", name: "险境二" }, { key: "jianghu_hole03", name: "险境三" }, { key: "jianghu_hole04", name: "险境四" }, { key: "jianghu_hole05", name: "险境五" }, { key: "jianghu_hole06", name: "险境六" }, { key: "jianghu_newtype01", name: "新境一" }, { key: "jianghu_newtype02", name: "新境二" }] },
  { label: "内景 / 其它", items: [{ key: "groupscene045", name: "天隐谷" }, { key: "prison", name: "监狱" }, { key: "fight05", name: "比武场五" }, { key: "fight06", name: "比武场六" }, { key: "fight08", name: "比武场八" }, { key: "loading01", name: "世界王场景" }, { key: "scene12", name: "场景12" }, { key: "scene13", name: "场景13" }] }
];

export const TIGUAN = [
  { label: "成都场景", items: [{ key: "tiguan_1", name: "浣花剑派" }, { key: "tiguan_3", name: "沈家堡" }, { key: "tiguan_4", name: "徐家庄" }, { key: "tiguan_5", name: "南宫世家" }] },
  { label: "烟雨庄", items: [{ key: "tiguan_7", name: "盐帮" }, { key: "tiguan_13", name: "神风帮" }, { key: "tiguan_25", name: "天外天" }] },
  { label: "燕京", items: [{ key: "tiguan_15", name: "古墓派" }, { key: "tiguan_18", name: "东方世家" }, { key: "tiguan_23", name: "杀人庄" }] },
  { label: "洛阳", items: [{ key: "tiguan_20", name: "燕门世家" }, { key: "tiguan_28", name: "抱犊寨" }, { key: "tiguan_26", name: "秦王府" }] },
  { label: "唐门", items: [{ key: "tiguan_16", name: "沐云庄" }, { key: "tiguan_17", name: "十二连环坞" }, { key: "tiguan_29", name: "天宝寨" }] },
  { label: "恶人谷", items: [{ key: "tiguan_6", name: "星宿派" }, { key: "tiguan_21", name: "素云宫" }] },
  { label: "塞外大漠", items: [{ key: "tiguan_24", name: "大漠鬼城" }, { key: "tiguan_32", name: "天罗教" }] },
  { label: "其他场景", items: [{ key: "tiguan_2", name: "天香茶林(千灯镇)" }, { key: "tiguan_8", name: "仙霞派(蓬莱仙境)" }, { key: "tiguan_9", name: "霓裳宫(念萝坝)" }, { key: "tiguan_10", name: "霹雳堂(梅花岭)" }, { key: "tiguan_11", name: "万兽山庄(丐帮)" }, { key: "tiguan_12", name: "嵩山派(少林)" }, { key: "tiguan_14", name: "幽冥宫(泛黄平原)" }, { key: "tiguan_19", name: "慕容世家(苏州)" }, { key: "tiguan_22", name: "凌霄城" }, { key: "tiguan_27", name: "恶虎山寨(漠河镇)" }, { key: "tiguan_30", name: "虎啸山庄(武当)" }, { key: "tiguan_31", name: "幽灵山庄(锦衣卫)" }, { key: "tiguan_33", name: "双龙堡(峨眉)" }, { key: "tiguan_34", name: "冥岳" }, { key: "tiguan_35", name: "净天宫" }] }
];

/* ---- 常用传送 / 踢馆传送：分组按钮矩阵 + 顶部过滤 ---- */

function renderMatrix(host, { id, title, desc, groups }) {
  const { section: panel, body } = panelSection({ id, title, desc });
  const filterInput = h("input", { type: "text", placeholder: "按名称过滤全部分组，例如：少林" });
  filterInput.style.cssText =
    "width:100%;max-width:420px;background:#11141b;border:1px solid var(--gm-line);border-radius:6px;color:var(--gm-text);padding:6px 10px;margin-bottom:10px;";
  const matrix = h("div", { class: "btn-matrix" });
  const groupNodes = [];
  for (const group of groups) {
    const items = h("div", { class: "matrix-items" });
    for (const item of group.items) {
      const btn = h("ax-btn", { label: item.name, type: "plain", "data-name": item.name });
      btn.addEventListener("click", () =>
        applyCommand({ action: "switch_scene_preset", preset: item.key }, "传送：" + item.name)
      );
      items.appendChild(btn);
    }
    groupNodes.push({ items, buttons: Array.from(items.children) });
    matrix.appendChild(
      h("div", { class: "matrix-group" }, [h("div", { class: "matrix-label" }, [group.label]), items])
    );
  }
  filterInput.addEventListener("input", () => {
    const needle = filterInput.value.trim().toLowerCase();
    for (const group of groupNodes) {
      let visible = 0;
      for (const btn of group.buttons) {
        const hit = !needle || (btn.getAttribute("data-name") || "").toLowerCase().includes(needle);
        btn.style.display = hit ? "" : "none";
        if (hit) visible++;
      }
      group.items.parentElement.style.display = visible ? "" : "none";
    }
  });
  body.appendChild(filterInput);
  body.appendChild(matrix);
  host.appendChild(panel);
}

function renderPresets(host) {
  renderMatrix(host, {
    id: "sec-presets",
    title: "常用传送",
    desc: "主城、门派、野外、禁地、活动与秘境全量传送点。顶部过滤框跨全部分组即时过滤；点击即传送。",
    groups: PRESETS,
  });
}

function renderTiguan(host) {
  renderMatrix(host, {
    id: "sec-tiguan",
    title: "踢馆传送",
    desc: "点击对应踢馆名称，传送到入口 NPC 面前。共 35 个踢馆。",
    groups: TIGUAN,
  });
}

/* ---- 自定义切图与场景内瞬移 ---- */

function renderTeleport(host) {
  const { section, body } = panelSection({
    id: "sec-teleport",
    title: "自定义切图与场景内瞬移",
    desc: "切图格式：场景资源,场景码,X,Y,Z,朝向。场景目录按需拉取，选中后自动填入目的地（坐标默认 0，可用「填入当前坐标」补全）；瞬移只改当前场景内坐标。",
  });
  const destination = h("input", {
    type: "text",
    value: "ini\\scene\\city04_LuoYang,city04,243.387,43.277,403.821,5.024",
  });
  destination.style.cssText =
    "width:100%;max-width:560px;background:#11141b;border:1px solid var(--gm-line);border-radius:6px;color:var(--gm-text);padding:6px 10px;";
  const scenePicker = catalogPicker({
    recentKey: "scenes",
    placeholder: "按场景名/资源名过滤（拉取一次后本地过滤），选中后自动填入目的地",
    emptyText: "没有匹配的场景",
    fetchRows: async (query) => {
      const rows = await catalog("/api/scenes");
      const needle = query.toLowerCase();
      return rows
        .filter(
          (row) =>
            !needle ||
            (row.name || "").toLowerCase().includes(needle) ||
            row.config.toLowerCase().includes(needle) ||
            row.resource.toLowerCase().includes(needle)
        )
        .map((row) => ({
          label: (row.name || row.resource) + "（" + row.config + "）",
          sub: row.resource,
          value: row.config,
        }));
    },
    onPick(row) {
      destination.value = "ini\\scene\\" + row.sub + "," + row.value + ",0,0,0,0";
    },
  });
  const switchBtn = h("ax-btn", { label: "切换场景", theme: "prim" });
  switchBtn.addEventListener("click", () => {
    const value = destination.value.trim();
    if (!value) return toast("请先填写目的地", "warn");
    applyCommand({ action: "switch_scene", destination: value }, "切图指令已发送");
  });
  body.appendChild(scenePicker.node);
  body.appendChild(h("div", { class: "field-row", style: "margin-top:10px;" }, [destination]));
  const coords = h("input", { type: "text", placeholder: "X,Y,Z,朝向（场景内瞬移）" });
  coords.style.cssText =
    "width:260px;background:#11141b;border:1px solid var(--gm-line);border-radius:6px;color:var(--gm-text);padding:6px 10px;";
  const fillCurrent = h("ax-btn", { label: "填入当前坐标", type: "plain" });
  fillCurrent.addEventListener("click", () => {
    const state = snapshot();
    const player = state.sessions[0];
    if (!player || !player.position || typeof player.position.x !== "number") {
      return toast("没有在线角色", "warn");
    }
    const p = player.position;
    coords.value = [p.x, p.y, p.z, p.orient].join(",");
  });
  const teleport = h("ax-btn", { label: "场景内瞬移", theme: "prim" });
  teleport.addEventListener("click", () => {
    const value = coords.value.trim();
    if (!value) return toast("请先填写坐标", "warn");
    applyCommand({ action: "teleport", coords: value }, "瞬移指令已发送");
  });
  body.appendChild(h("div", { class: "actions-row" }, [switchBtn]));
  body.appendChild(h("div", { class: "field-row", style: "margin-top:10px;" }, [coords, fillCurrent, teleport]));
  host.appendChild(section);
}

/* ---- 场景天气 ---- */

function renderWeather(host) {
  const { section, body } = panelSection({
    id: "sec-weather",
    title: "场景天气",
    desc: "实时切换客户端天气渲染包（目录名带 _001 为昼、_501 为夜）。雨/雪粒子需客户端开启天气并使用高或极高画质。切图重进即恢复场景默认天气。",
  });
  let rows = null;
  let selected = null;
  const picker = catalogPicker({
    recentKey: "weather",
    placeholder: "搜索天气（拉取一次后本地过滤）",
    emptyText: "没有匹配的天气",
    fetchRows: async (query) => {
      rows = rows || (await catalog("/api/weather"));
      const needle = query.toLowerCase();
      return rows
        .filter(
          (row) =>
            !needle ||
            (row.Label || "").toLowerCase().includes(needle) ||
            (row.Key || "").toLowerCase().includes(needle)
        )
        .map((row) => ({ label: row.Label, sub: row.Key, value: row.Key }));
    },
    onPick(row) {
      selected = row;
    },
  });
  const fade = numField({ label: "过渡秒", value: 10, min: 0, max: 600, width: "70px" });
  picker.refresh(); // 进板块即拉一次（本地过滤型目录）
  const apply = h("ax-btn", { label: "切换天气", theme: "prim" });
  apply.addEventListener("click", () => {
    if (!selected) return toast("请先选择一种天气", "warn");
    applyCommand(
      { action: "set_weather", weather: selected.value, fade: fade.get() ?? 10 },
      "天气已切换：" + selected.label
    );
  });
  body.appendChild(picker.node);
  body.appendChild(h("div", { class: "actions-row", style: "margin-top:10px;" }, [fade.node, apply]));
  host.appendChild(section);
}

/* ---- 副本 ---- */

function renderClone(host) {
  const { section, body } = panelSection({
    id: "sec-clone",
    title: "副本",
    desc: "列表来自服务端已安装的 cloneinfo.ini。进入即切场景并按波次逐级揭示；副本内击杀不复活，全灭自动推进；进入不限次，销毁回进本前位置。",
  });
  const configSelect = h("ax-select", { search: true, full: true, "max-height": "260", placeholder: "选择副本（可输入过滤）" });
  const funSelect = h("ax-select", { style: "width:110px;" });
  const note = hint();
  const byConfig = new Map();
  catalog("/api/clones")
    .then((rows) => {
      for (const row of rows) byConfig.set(row.config, row);
      configSelect.set({
        key: "content",
        value: JSON.stringify(
          rows.map((row) => ({
            label: row.config + (row.description ? " — " + row.description : ""),
            value: row.config,
          }))
        ),
      });
      syncFun();
    })
    .catch((err) => note.set(err.message || String(err), false));
  function syncFun() {
    const config = configSelect.value;
    const row = config ? byConfig.get(config) : null;
    const allowed = row ? row.difficulties : [1];
    const names = { 1: "普通", 2: "困难", 3: "地狱", 4: "炼狱" };
    funSelect.set({
      key: "content",
      value: JSON.stringify([1, 2, 3, 4].filter((n) => allowed.includes(n)).map((n) => ({ label: names[n], value: String(n) }))),
    });
  }
  configSelect.addEventListener("changed", syncFun);
  const enter = h("ax-btn", { label: "创建并进入副本", theme: "prim" });
  enter.addEventListener("click", () => {
    const config = configSelect.value;
    if (!config) return toast("请先选择副本", "warn");
    applyCommand(
      { action: "enter_clone", config, fun: Number(funSelect.value || 1) },
      "已请求进入副本：" + config
    );
  });
  const reenter = h("ax-btn", { label: "进入已打开副本", type: "plain" });
  reenter.addEventListener("click", () => applyCommand({ action: "reenter_clone" }, "已重新进入副本"));
  body.appendChild(h("div", { class: "field-row" }, [configSelect]));
  body.appendChild(h("div", { class: "field-row" }, [h("label", {}, ["难度"]), funSelect]));
  body.appendChild(
    h("div", { class: "actions-row" }, [
      enter,
      reenter,
      cmdBtn({
        label: "销毁副本",
        danger: true,
        confirm: "确定销毁当前已打开副本及其未结算进度？",
        action: "destroy_clone",
        section: "副本",
      }),
    ])
  );
  body.appendChild(note.node);
  host.appendChild(section);
}

/* ---- 世界 BOSS ---- */

function renderBoss(host) {
  const { section, body } = panelSection({
    id: "sec-worldboss",
    title: "世界 BOSS",
    desc: "强制生成按 normalboss/jhsceneboss 刷新表驱动：门派固定点每点 1 只，江湖 BOSS 在随机场景池内随机点降世。击杀后消失不重生；会话级摆放，切图清空。",
  });
  body.appendChild(
    h("div", { class: "actions-row" }, [
      cmdBtn({ label: "强制生成全部", theme: "prim", action: "worldboss_spawn", section: "世界 BOSS" }),
      cmdBtn({ label: "清除在场 BOSS", danger: true, action: "worldboss_clear", section: "世界 BOSS" }),
      cmdBtn({ label: "查看在场情况", type: "plain", action: "worldboss_list", section: "世界 BOSS" }),
    ])
  );
  host.appendChild(section);
}

export function render(host, { sectionId }) {
  const renderers = {
    presets: renderPresets,
    tiguan: renderTiguan,
    teleport: renderTeleport,
    weather: renderWeather,
    clone: renderClone,
    worldboss: renderBoss,
  };
  (renderers[sectionId] || renderPresets)(host);
}
