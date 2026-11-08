// 九阴本地 GM 控制台 · 核心层 —— 会话刷新、导航分组框架与命令通道。
// 本面板直接服务当前游戏服（单服、仅监听 127.0.0.1），无网关转发：
// 所有 /api 请求都打到面板自身，功能板块保持与网关版一致，
// 服务端尚未实现的命令会在点击时给出明确提示。
// 加载顺序：本文件必须先于 console-features.js（console.html 的两个 defer <script> 按序执行）。
    // —— 单服直连：GM_SERVER 仅用于顶栏徽标展示，不再参与请求路由 ——
    const GM_SERVER = location.host;
    const GM_SERVER_NAME = "当前服";
    let sessions = [],
      cloneOptions = [];
    async function refresh() {
      // 独立面板部署时本接口经面板代理游戏服 internal API；
      // 游戏服重启窗口内会拿到 503 {"error":...}——按“暂不可达”处理，
      // 轮询继续，游戏服回来后自动恢复。
      let r = await fetch("/api/sessions");
      if (!r.ok) {
        sessions = [];
        let st = document.querySelector("#state");
        if (st) st.textContent = "游戏服不可达（" + r.status + "）——等待自动恢复";
        renderPlayerCard();
        refreshBots();
        return;
      }
      sessions = await r.json();
      let st = document.querySelector("#state");
      if (st) st.textContent = JSON.stringify(sessions, null, 2);
      renderPlayerCard();
      refreshBots();
    }
    function renderPlayerCard() {
      let box = document.querySelector("#player-card");
      if (!box) return;
      if (!sessions.length) {
        box.innerHTML =
          '<span class="chip"><b>状态</b>没有在线角色——请先登录游戏</span>';
        return;
      }
      let p = sessions[0];
      let pos =
        p.position && typeof p.position.x === "number"
          ? p.position.x.toFixed(1) +
          ", " +
          p.position.y.toFixed(1) +
          ", " +
          p.position.z.toFixed(1)
          : "-";
      let scene = (p.scene && (p.scene.name || p.scene.config)) || "-";
      let sceneCode = (p.scene && p.scene.config) || "";
      let faction = p.factionName || p.faction || "-";
      box.innerHTML =
        '<span class="chip"><b>角色</b>' +
        (p.name || "-") +
        "</span>" +
        '<span class="chip" title="' +
        (p.faction || "") +
        '"><b>门派</b>' +
        faction +
        "</span>" +
        '<span class="chip" title="' +
        sceneCode +
        '"><b>场景</b>' +
        scene +
        "</span>" +
        '<span class="chip"><b>坐标</b>' +
        pos +
        "</span>" +
        '<span class="chip"><b>碎银</b>' +
        (p.silver != null ? p.silver : "-") +
        "</span>" +
        '<span class="chip"><b>官银</b>' +
        (p.guanYin != null ? p.guanYin : "-") +
        "</span>" +
        '<span class="chip"><b>黄金</b>' +
        (p.gold != null ? p.gold : "-") +
        "</span>" +
        '<span class="chip"><b>怒气</b>' +
        (p.sp != null ? p.sp : "-") +
        "</span>" +
        '<span class="chip"><b>修为</b>' +
        (p.faculty != null ? p.faculty : "-") +
        "</span>" +
        '<span class="chip"><b>最近操作</b>' +
        (p.lastAction || "-") +
        "</span>";
    }
    /** GM 面板功能分组（2026-11-02 定稿）：一级=分组、二级=板块；按 h2 标题精确
     *  匹配，改名/新增的板块自动落入末尾「其他」组兜底，不会失联。 */
    const GM_GROUPS = [
      { name: "角色状态", items: ["在线角色", "五维属性", "实力等级", "体力 / 饱食 / 杀戮 / 酒量", "性格善恶", "称号管理"] },
      { name: "货币与物品", items: ["碎银 / 官银 / 黄金 / 怒气", "礼包与物品", "世界宝箱"] },
      { name: "装备与铸造", items: ["所有道具与换装", "洗练（身上装备 · 指定部位）", "模板生成（带词条装备）", "冶炼（已对齐 D:\\ 参考端）", "兵录设置"] },
      { name: "外观与坐骑", items: ["时装", "时装 / 身份装发放", "风物志卡牌", "风物志头发", "坐骑管理", "待机站姿（持久化）"] },
      { name: "武学与修炼", items: ["门派", "武学套路", "修为", "经脉", "经脉周天设置"] },
      { name: "传送与场景", items: ["常用传送", "踢馆传送", "自定义切图与场景内瞬移", "场景天气", "副本", "世界 BOSS"] },
      { name: "战斗与演练", items: ["训练木桩", "添加假人（战斗）", "伙伴系统", "御风神水（大轻功）"] },
      { name: "游戏参数", items: ["怪物刷新", "数据存盘", "移动参数"] },
      { name: "任务与身份", items: ["场景任务验证", "暮色剧情验证", "九阴志全局门控", "身份系统（一键激活全部）", "门派声望（自定义）"] },
      { name: "界面与操作", items: ["快捷栏位", "快捷键方案"] },
      { name: "调试工具", items: ["S2C 自定义消息探针", "聊天与气泡调试", "动作与剧情测试"] },
    ];

    let sectionGroup = {};

    function normTitle(t) {
      return t.replace(/\s+/g, " ").trim();
    }

    function buildNav() {
      let nav = document.querySelector("#gm-nav");
      if (!nav) return;
      let sections = Array.from(document.querySelectorAll("main.grid > section"));
      let byTitle = new Map();
      sections.forEach(function (sec, i) {
        if (!sec.id) sec.id = "sec-" + i;
        let h = sec.querySelector("h2");
        if (h) byTitle.set(normTitle(h.textContent), sec);
      });
      let listed = new Set();
      let appendGroup = function (name, secs) {
        let box = document.createElement("div");
        box.className = "grp";
        box.dataset.group = name;
        let hdr = document.createElement("div");
        hdr.className = "grp-hdr";
        hdr.textContent = name;
        hdr.title = secs.length + " 个板块";
        hdr.onclick = function () {
          toggleGroup(name);
        };
        box.appendChild(hdr);
        secs.forEach(function (sec) {
          sectionGroup[sec.id] = name;
          listed.add(sec.id);
          let a = document.createElement("a");
          a.textContent = normTitle(sec.querySelector("h2").textContent);
          a.dataset.target = sec.id;
          a.onclick = function () {
            showSection(sec.id);
          };
          box.appendChild(a);
        });
        nav.appendChild(box);
      };
      GM_GROUPS.forEach(function (g) {
        let secs = g.items
          .map(function (title) {
            return byTitle.get(normTitle(title));
          })
          .filter(Boolean);
        appendGroup(g.name, secs);
      });
      // 未列入分组的板块 → 「其他」，防止改名/新增后导航失联
      let stray = sections.filter(function (sec) {
        return sec.querySelector("h2") && !listed.has(sec.id);
      });
      if (stray.length) appendGroup("其他", stray);
      let saved = localStorage.getItem("gm-tab");
      let target =
        saved && sections.some(function (sec) { return sec.id === saved; })
          ? saved
          : sections[0] && sections[0].id;
      if (target) showSection(target);
    }
    function collapsedSet() {
      let raw = localStorage.getItem("gm-group-collapsed");
      if (raw) {
        try {
          return new Set(JSON.parse(raw));
        } catch (e) { /* 损坏则回退默认 */ }
      }
      // 初次访问：默认全收起，仅当前板块所在组展开
      let all = new Set();
      document
        .querySelectorAll("#gm-nav .grp")
        .forEach(function (b) {
          all.add(b.dataset.group);
        });
      return all;
    }
    function saveCollapsed(set) {
      localStorage.setItem(
        "gm-group-collapsed",
        JSON.stringify(Array.from(set))
      );
    }
    function toggleGroup(name) {
      let set = collapsedSet();
      if (set.has(name)) set.delete(name);
      else set.add(name);
      saveCollapsed(set);
      applyGroupStates();
    }
    function applyGroupStates() {
      let active = document.querySelector("main.grid > section.active");
      let activeGroup = active ? sectionGroup[active.id] : null;
      let collapsed = collapsedSet();
      document
        .querySelectorAll("#gm-nav .grp")
        .forEach(function (box) {
          let name = box.dataset.group;
          box.classList.toggle("open", !collapsed.has(name) || name === activeGroup);
        });
    }
    function showSection(id) {
      document
        .querySelectorAll("main.grid > section")
        .forEach(function (sec) {
          sec.classList.toggle("active", sec.id === id);
        });
      document.querySelectorAll("#gm-nav a").forEach(function (a) {
        a.classList.toggle("active", a.dataset.target === id);
      });
      localStorage.setItem("gm-tab", id);
      applyGroupStates();
    }
    async function loadClones() {
      cloneOptions = await (await fetch("/api/clones")).json();
      let s = document.querySelector("#clone-config");
      s.innerHTML = "";
      for (let c of cloneOptions) {
        let o = document.createElement("option");
        o.value = c.config;
        o.textContent =
          c.config + (c.description ? " — " + c.description : "");
        s.appendChild(o);
      }
      syncCloneFun();
    }
    function syncCloneFun() {
      let c = cloneOptions.find(
        (x) => x.config === document.querySelector("#clone-config").value,
      );
      let allowed = c ? c.difficulties : [1];
      for (let o of document.querySelector("#clone-fun").options)
        o.disabled = !allowed.includes(+o.value);
      let current = document.querySelector("#clone-fun");
      if (current.selectedOptions[0].disabled)
        current.value = String(allowed[0]);
    }
    async function command(payload) {
      await refresh();
      if (!sessions.length) return alert("没有在线角色");
      let status = document.querySelector("#command-status");
      status.textContent = "发送中……";
      let r = await fetch(
        "/api/sessions/" + sessions[0].sessionId + "/command",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        },
      );
      let j = await r.json();
      if (!r.ok) {
        // 服务端白名单之外的动作（面板功能全量保留，服务端逐个实现中）。
        let message =
          j.error === "unsupported action"
            ? "当前服暂未实现该命令（" + payload.action + "），等待服务端支持"
            : j.error || "命令失败";
        status.textContent = "失败：" + message;
        return alert(message);
      }
      await refresh();
      status.textContent =
        (sessions[0] && sessions[0].lastAction) || "已发送";
    }
    function popupGMPanel() {
      command({ action: "popup_gm_panel" });
    }
    // 伙伴：botCommand 走与 command() 相同的会话通道（bot_* 动作可带参数）；
    // refreshBots 渲染 /api/sessions 的实时伙伴状态（sessions[0].bot，无伙伴显示占位）。
    function refreshBots() {
      let el = document.querySelector("#bot-state");
      if (!el) return;
      let bot = sessions[0] && sessions[0].bot;
      el.textContent = bot
        ? JSON.stringify(bot, null, 2)
        : "无伙伴（点击「创建 / 校验青禾」生成）";
    }
    function botCommand(action, params) {
      command(Object.assign({ action: action }, params || {})).then(refresh);
    }
