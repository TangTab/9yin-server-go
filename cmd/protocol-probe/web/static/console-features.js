// 九阴本地 GM 控制台 · 功能层 —— 各板块功能函数（含原洗练/模板生成板块内嵌脚本）
// 与文件末尾的初始化序列；依赖 console-core.js 的 command()/sessions/refresh 等全局。
          const RECAST_SLOTS = [
            ["武器", 22, "武器"],
            ["防具", 1, "头发"],
            ["防具", 3, "衣服"],
            ["防具", 6, "护腕"],
            ["防具", 4, "裤子"],
            ["防具", 7, "绑腿"],
            ["防具", 8, "鞋子"],
            ["首饰", 11, "耳环1"],
            ["首饰", 12, "耳环2"],
            ["首饰", 13, "项链"],
            ["首饰", 14, "戒指1"],
            ["首饰", 15, "戒指2"],
            ["宝物", 25, "宝物1"],
            ["宝物", 26, "宝物2"],
            ["宝物", 27, "宝物3"],
            ["宝物", 28, "宝物4"],
            ["宝物", 29, "宝物5"],
            ["娃娃", 44, "娃娃1"],
            ["娃娃", 45, "娃娃2"],
            ["娃娃", 46, "娃娃3"],
            ["娃娃", 47, "娃娃4"],
            ["娃娃", 48, "娃娃5"],
          ];
          (function () {
            let sel = document.querySelector("#recast-target-slot");
            if (!sel) return;
            for (let s of RECAST_SLOTS) {
              let o = document.createElement("option");
              o.value = s[1];
              o.textContent = "[" + s[0] + "] " + s[2] + " (槽" + s[1] + ")";
              sel.appendChild(o);
            }
          })();
          function gmRecast(action) {
            let sel = document.querySelector("#recast-target-slot");
            let slot = sel ? +sel.value : 0;
            if (!slot) return alert("请先选择目标部位");
            let hint = document.querySelector("#recast-result");
            if (hint) hint.textContent = "洗练中…";
            command({ action: action, slot: slot });
          }
          async function loadRecastAffixes() {
            try {
              let j = await (await fetch("/api/recast-affixes")).json();
              let ph = document.querySelector("#recast-affix-hint");
              if (ph && j.loaded === false)
                ph.textContent =
                  "洗练资源未加载（缺少 senior_weapon_*.ini），词条列表为空";
              function fill(id, arr, label, kind) {
                let box = document.querySelector(id);
                if (!box) return;
                box.innerHTML = "";
                if (!arr || !arr.length) {
                  box.textContent = "（无" + label + "）";
                  return;
                }
                for (let e of arr) {
                  let lab = document.createElement("label");
                  lab.className = "affix-item";
                  lab.setAttribute("data-weapon", e.weapon || "");
                  let title, text;
                  if (kind === "prop") {
                    title =
                      "ID " +
                      e.id +
                      (e.source ? " [" + e.source + "]" : "") +
                      (e.name ? " " + e.name : "");
                    text = e.name || "ID " + e.id;
                    if (e.source)
                      text +=
                        ' <span class="muted">[' + e.source + "]</span>";
                  } else {
                    title =
                      "ID " +
                      e.id +
                      (e.source ? " " + e.source : "") +
                      (e.detail ? " " + e.detail : "");
                    text = e.detail || e.name || e.source || "ID " + e.id;
                    if (e.taolu)
                      text += '<span class="muted"> [' + e.taolu + "]</span>";
                  }
                  lab.title = title;
                  lab.innerHTML =
                    '<input type="checkbox" value="' + e.id + '"> ' + text;
                  box.appendChild(lab);
                }
              }
              fill("#recast-prop-list", j.prop, "普通词条", "prop");
              fill("#recast-skill-list", j.skill, "招式词条", "skill");
              buildWeaponFilter(j.skill || []);
            } catch (e) {
              let ph = document.querySelector("#recast-affix-hint");
              if (ph) ph.textContent = "加载词条列表失败：" + e.message;
            }
          }
          function buildWeaponFilter(skillArr) {
            let sel = document.querySelector("#affix-weapon-filter");
            if (!sel) return;
            let seen = {};
            for (let e of skillArr) {
              if (e.weapon) seen[e.weapon] = 1;
            }
            let groups = ["剑", "刀", "刺", "棍", "棒", "枪", "徒手"].filter(
              function (g) {
                return seen[g];
              },
            );
            let others = [];
            for (let k in seen) {
              if (["剑", "刀", "刺", "棍", "棒", "枪", "徒手"].indexOf(k) < 0)
                others.push(k);
            }
            let opts = [["all", "全部"]].concat(
              groups.map(function (g) {
                return [g, g];
              }),
            );
            if (others.length) opts.push(["其他", "其他"]);
            sel.innerHTML = "";
            for (let o of opts) {
              let el = document.createElement("option");
              el.value = o[0];
              el.textContent = o[1];
              sel.appendChild(el);
            }
          }
          function toggleAffixList() {
            let box = document.querySelector("#affix-list-box"),
              flt = document.querySelector("#affix-filter"),
              wlb = document.querySelector("#affix-weapon-label");
            if (!box) return;
            let show = box.style.display === "none";
            box.style.display = show ? "block" : "none";
            if (flt) flt.style.display = show ? "inline-block" : "none";
            if (wlb) wlb.style.display = show ? "inline-block" : "none";
            if (show) loadRecastAffixes();
          }
          function filterAffixList() {
            let kw =
              (document.querySelector("#affix-filter") || {}).value || "";
            kw = kw.toLowerCase();
            let wp =
              (document.querySelector("#affix-weapon-filter") || {}).value ||
              "all";
            document
              .querySelectorAll("#recast-prop-list .affix-item")
              .forEach(function (lab) {
                let okK =
                  !kw || lab.textContent.toLowerCase().indexOf(kw) >= 0;
                lab.style.display = okK ? "inline-flex" : "none";
              });
            document
              .querySelectorAll("#recast-skill-list .affix-item")
              .forEach(function (lab) {
                let okK =
                  !kw || lab.textContent.toLowerCase().indexOf(kw) >= 0;
                let okW =
                  wp === "all" || lab.getAttribute("data-weapon") === wp;
                lab.style.display = okK && okW ? "inline-flex" : "none";
              });
          }
          function recastChecked(kind) {
            let sel = document.querySelector("#recast-target-slot");
            let slot = sel ? +sel.value : 0;
            if (!slot) return alert("请先在上面选择目标部位");
            let ids = [];
            document
              .querySelectorAll("#recast-" + kind + "-list input:checked")
              .forEach(function (c) {
                ids.push(+c.value);
              });
            if (!ids.length)
              return alert(
                "请先在列表里勾选要加入的" +
                (kind === "prop" ? "普通" : "招式") +
                "词条",
              );
            let hint = document.querySelector("#recast-result");
            if (hint) hint.textContent = "加入词条中…";
            command({
              action: "recast_add_affix",
              slot: slot,
              packIDs: ids,
              affixKind: kind,
            });
          }
          function recastClearAffix() {
            let sel = document.querySelector("#recast-target-slot");
            let slot = sel ? +sel.value : 0;
            if (!slot) return alert("请先在上面选择目标部位");
            if (
              !confirm(
                "确定清空该装备的全部词条（普通+招式）？此操作不可恢复。",
              )
            )
              return;
            let hint = document.querySelector("#recast-result");
            if (hint) hint.textContent = "清空词条中…";
            command({ action: "recast_clear_affix", slot: slot });
          }

          function gmTemplateWeapon() {
            command({
              action: "gm_generate_weapon",
              weaponType: +document.querySelector("#tpl-weapon-type").value,
              crafter: document.querySelector("#tpl-crafter").value,
            });
          }
          function gmTemplateArmor() {
            command({
              action: "gm_generate_armor",
              armorSlot: document.querySelector("#tpl-armor-slot").value,
            });
          }
          function gmTemplateBracer() {
            command({ action: "gm_generate_bracer" });
          }

    function silver() {
      command({
        action: "set_silver",
        silver: +document.querySelector("#silver").value,
      });
    }
    function guanyinSet() {
      command({
        action: "set_guanyin",
        guanYin: +document.querySelector("#guanyin").value,
      });
    }
    function goldSet() {
      command({
        action: "set_gold",
        gold: +document.querySelector("#gold").value,
      });
    }
    // 经脉：每日真气硬闸（是否开启 + 每日阈值，角色级持久化）。
    function zhenqiCapSet() {
      let enabled = document.querySelector("#zhenqi-cap-enabled").checked ? 1 : 0;
      let value = +document.querySelector("#zhenqi-cap-value").value;
      if (!value || value < 1) {
        return alert("每日阈值必须 ≥ 1");
      }
      command({ action: "set_zhenqi_cap", enabled: enabled, value: value });
    }
    function authoredTask(action) {
      command({
        action,
        taskId: +document.querySelector("#authored-task-id").value || 0,
      });
    }
    function setSP() {
      command({ action: "set_sp", sp: +document.querySelector("#sp").value });
    }
    function addSP(sp) {
      command({ action: "add_sp", sp });
    }
    function fillSP() {
      command({ action: "fill_sp" });
    }
    function setMoveSpeed() {
      command({
        action: "set_move_speed",
        value: +document.querySelector("#move-speed").value,
      });
    }
    function setGravity() {
      command({
        action: "set_gravity",
        value: +document.querySelector("#gravity").value,
      });
    }
    function setFaculty() {
      command({
        action: "set_faculty",
        value: +document.querySelector("#faculty-value").value,
      });
    }
    function setFillSpeed() {
      command({
        action: "set_fill_speed",
        base: +document.querySelector("#fill-speed-base").value,
        multiplier: +document.querySelector("#fill-speed-multiplier").value,
      });
    }
    function setActDayLimit() {
      command({
        action: "set_act_day_limit",
        value: +document.querySelector("#act-day-limit").value,
      });
    }
    function spawnDummy(hp) {
      command({
        action: "spawn_dummy",
        dummyHp: hp || +document.querySelector("#dummy-hp").value || 0,
      });
    }
    function spawnChest() {
      command({ action: "spawn_chest" });
    }
    let CD_TAOLUS = [];
    async function loadCombatDummyTaolus() {
      try {
        CD_TAOLUS = await (await fetch("/api/combat-dummy/taolus")).json();
        let s = document.querySelector("#cd-preset");
        if (!s) return;
        s.innerHTML = '<option value="random">随机套路</option>';
        for (let t of CD_TAOLUS) {
          let o = document.createElement("option");
          o.value = t.key;
          o.textContent = t.label + "（" + t.skills.length + "招）";
          s.appendChild(o);
        }
      } catch (e) { }
    }
    function cdPreset(key) {
      if (key === "random") {
        if (!CD_TAOLUS.length) return;
        let t = CD_TAOLUS[Math.floor(Math.random() * CD_TAOLUS.length)];
        document.querySelector("#cd-skills").value = t.skills.join("、");
        return;
      }
      let t = CD_TAOLUS.find(function (x) {
        return x.key === key;
      });
      document.querySelector("#cd-skills").value = t
        ? t.skills.join("、")
        : "";
    }
    function spawnCombatDummy() {
      command({
        action: "spawn_combat_dummy",
        dummyName: document.querySelector("#cd-name").value.trim(),
        dummyHp: +document.querySelector("#cd-hp").value || 0,
        dummyDamage: +document.querySelector("#cd-dmg").value || 0,
        dummySkills: document.querySelector("#cd-skills").value.trim(),
      });
    }
    function clearCombatDummies() {
      if (confirm("确定清除当前场景的全部战斗假人？"))
        command({ action: "clear_combat_dummies" });
    }
    // 伙伴辅助配置：开关 + 间隔秒 + 单次治疗量（bot_support_configure）。
    function botSupportConfigure() {
      botCommand("bot_support_configure", {
        enabled: +document.querySelector("#bot-support-enabled").value,
        intervalSec: +document.querySelector("#bot-support-interval").value || 5,
        heal: +document.querySelector("#bot-support-heal").value || 0,
      });
    }
    function setPKMode(mode) {
      command({ action: "set_pk_mode", pkMode: mode });
    }
    function setWeather() {
      command({
        action: "set_weather",
        weather: document.querySelector("#weather").value,
        fadeSec: +document.querySelector("#weather-fade").value || 10,
      });
    }
    function scene(destination) {
      command({ action: "switch_scene", destination });
    }
    function preset(key) {
      command({ action: "switch_scene_preset", preset: key });
    }
    function setFaction() {
      command({
        action: "set_faction",
        faction: document.querySelector("#faction").value,
      });
    }
    function setPowerLevel() {
      command({
        action: "set_power_level",
        value: +document.querySelector("#power-level").value,
      });
    }
    function setAlignment() {
      command({
        action: "set_alignment",
        justice: +document.querySelector("#justice").value,
        evil: +document.querySelector("#evil").value,
        freeBase: +document.querySelector("#free-base").value,
      });
    }
    function setVitals() {
      command({
        action: "set_vitals",
        ene: +document.querySelector("#vit-ene").value,
        maxEne: +document.querySelector("#vit-maxene").value,
        maxEneAdd: +document.querySelector("#vit-maxeneadd").value,
        resumeEne: +document.querySelector("#vit-resumeene").value,
        sat: +document.querySelector("#vit-sat").value,
        maxSat: +document.querySelector("#vit-maxsat").value,
        maxSatAdd: +document.querySelector("#vit-maxsatadd").value,
        pkValue: +document.querySelector("#vit-pk").value,
        drunkLevel: +document.querySelector("#vit-drunk").value,
      });
    }
    function useCurrentPosition() {
      if (!sessions.length) return alert("没有在线角色");
      let p = sessions[0].position;
      document.querySelector("#teleport").value = [
        p.x,
        p.y,
        p.z,
        p.orient,
      ].join(",");
    }
    function teleportPlayer() {
      command({
        action: "teleport",
        position: document.querySelector("#teleport").value,
      });
    }
    let SCENES = [];
    async function loadScenes() {
      SCENES = await (await fetch("/api/scenes")).json();
      let s = document.querySelector("#scene-select");
      s.innerHTML = "";
      for (let sc of SCENES) {
        let o = document.createElement("option");
        o.value = sc.config;
        o.dataset.resource = sc.resource;
        o.textContent = sc.name + " — " + sc.resource + " (" + sc.id + ")";
        s.appendChild(o);
      }
    }
    function filterScenes() {
      let kw = document.querySelector("#scene-filter").value.trim().toLowerCase();
      let s = document.querySelector("#scene-select");
      let i = 0;
      for (let sc of SCENES) {
        let text = (sc.name + " " + sc.resource + " " + sc.config).toLowerCase();
        s.options[i++].hidden = kw && !text.includes(kw);
      }
    }
    function fillSceneDestination() {
      let o = document.querySelector("#scene-select");
      if (!o.value) return;
      let tail = document.querySelector("#destination").value.split(",").slice(-4);
      if (tail.length !== 4 || tail.some((v) => isNaN(parseFloat(v)))) {
        tail = ["0", "0", "0", "0"];
      }
      document.querySelector("#destination").value = [
        o.value,
        o.selectedOptions[0].dataset.resource,
      ]
        .concat(tail)
        .join(",");
    }
    function enterClone() {
      command({
        action: "enter_clone",
        cloneConfig: document.querySelector("#clone-config").value,
        value: +document.querySelector("#clone-fun").value,
      });
    }
    function reenterClone() {
      command({ action: "reenter_clone" });
    }
    function destroyClone() {
      if (confirm("确定销毁当前已打开副本及其未结算进度？"))
        command({ action: "destroy_clone" });
    }
    function story(action, params) {
      command(params ? { action, ...params } : { action });
    }
    function ninJiuZhiSet() {
      let stage = +document.querySelector("#ninjiuzhi-stage").value;
      command({ action: "ninjiuzhi_set", stage });
    }
    function ninJiuZhiEvent(kind) {
      command({ action: "ninjiuzhi_event", eventKind: kind });
    }
    function ninJiuZhiReset() {
      if (
        confirm(
          "重置九阴志全局阶段？全服九阴志章节会退回起点，事件点数清零。",
        )
      )
        command({ action: "ninjiuzhi_reset" });
    }
    function ninJiuZhiStatus() {
      command({ action: "ninjiuzhi_status" });
    }
    let giftBoxes = [];
    async function loadGiftBoxes() {
      giftBoxes = await (await fetch("/api/giftboxes")).json();
      let s = document.querySelector("#gift-box");
      s.innerHTML = "";
      for (let b of giftBoxes) {
        let o = document.createElement("option");
        o.value = b.config;
        o.textContent =
          (b.hasContent ? "[有内容] " : "[未配置] ") +
          b.config +
          (b.name ? " — " + b.name : "");
        s.appendChild(o);
      }
    }
    async function loadWeather() {
      let options = await (await fetch("/api/weather")).json();
      let s = document.querySelector("#weather");
      s.innerHTML = "";
      for (let w of options) {
        let o = document.createElement("option");
        o.value = w.Key;
        o.textContent = w.Label + (w.TXID ? " — " + w.TXID : "");
        s.appendChild(o);
      }
    }
    function filterGifts() {
      let kw = document.querySelector("#gift-filter").value.toLowerCase();
      let s = document.querySelector("#gift-box");
      let i = 0;
      for (let b of giftBoxes) {
        let text = (b.config + " " + (b.name || "")).toLowerCase();
        s.options[i++].hidden = kw && !text.includes(kw);
      }
    }
    function addGiftBox() {
      command({
        action: "add_item",
        itemConfig: document.querySelector("#gift-box").value,
        amount: +document.querySelector("#gift-amount").value,
      });
    }
    function addItem() {
      command({
        action: "add_item",
        itemConfig: document.querySelector("#item-config").value.trim(),
        amount: +document.querySelector("#item-amount").value,
      });
    }
    function addYufengItem(id) {
      command({ action: "add_item", itemConfig: id, amount: 1 });
    }
    function sendTestMail() {
      command({
        action: "send_test_mail",
        itemConfig: document.querySelector("#item-config").value.trim(),
        amount: +document.querySelector("#item-amount").value,
      });
    }
    function sendMail() {
      let content = document.querySelector("#mail-content").value.trim();
      let item = document.querySelector("#mail-item").value.trim();
      if (!content && !item) return alert("正文与物品附件至少填一项");
      let payload = {
        action: "send_mail",
        content,
        itemConfig: item,
        amount: +document.querySelector("#mail-amount").value,
      };
      let role = document.querySelector("#mail-role").value.trim();
      if (role) payload.roleName = role;
      let title = document.querySelector("#mail-title").value.trim();
      if (title) payload.title = title;
      command(payload);
    }
    function chatBroadcast() {
      let t = document.querySelector("#chat-text").value.trim();
      if (!t) return alert("请输入聊天内容");
      command({ action: "chat_broadcast", speechText: t });
    }
    function speechLen() {
      let t = document.querySelector("#chat-text").value.trim();
      command({ action: "speech_len", speechText: t });
    }
    function speechNul() {
      let t = document.querySelector("#chat-text").value.trim();
      command({ action: "speech_nul", speechText: t });
    }
    function equipCard() {
      command({
        action: "equip_card",
        cardId: +document.querySelector("#card-id").value,
      });
    }
    async function loadCardCategories() {
      try {
        let rows = await (await fetch("/api/cards")).json();
        let cats = new Set();
        for (let c of rows) cats.add(c.category);
        let s = document.querySelector("#card-category");
        for (let cat of Array.from(cats).sort()) {
          let o = document.createElement("option");
          o.value = cat;
          o.textContent = cat;
          s.appendChild(o);
        }
      } catch (e) { }
    }
    async function renderCards(rows) {
      let s = document.querySelector("#card-result");
      s.innerHTML = "";
      for (let c of rows) {
        let o = document.createElement("option");
        o.value = c.id;
        o.textContent = c.name + " — ID " + c.id + " [" + c.category + "]";
        s.appendChild(o);
      }
      document.querySelector("#card-hint").textContent = rows.length
        ? "共 " + rows.length + " 张卡牌"
        : "没有匹配的卡牌";
    }
    async function loadAllCards() {
      try {
        let rows = await (await fetch("/api/cards")).json();
        renderCards(rows);
      } catch (e) {
        document.querySelector("#card-hint").textContent =
          "加载失败：" + e.message;
      }
    }
    async function searchCards() {
      let q = document
        .querySelector("#card-query")
        .value.trim()
        .toLowerCase();
      let cat = document.querySelector("#card-category").value;
      try {
        let rows = await (await fetch("/api/cards")).json();
        let filtered = rows.filter(
          (c) =>
            (!cat || c.category === cat) &&
            (!q ||
              c.name.toLowerCase().includes(q) ||
              String(c.id) === q ||
              String(c.config_id) === q),
        );
        renderCards(filtered);
      } catch (e) {
        document.querySelector("#card-hint").textContent =
          "搜索失败：" + e.message;
      }
    }
    function equipSelectedCard() {
      let sel = document.querySelector("#card-result");
      if (!sel.value) return alert("请先选择一张卡牌");
      command({ action: "equip_card", cardId: +sel.value });
    }
    async function loadHairCards() {
      try {
        let rows = await (await fetch("/api/hair-cards")).json();
        let s = document.querySelector("#hair-result");
        s.innerHTML = "";
        for (let c of rows) {
          let o = document.createElement("option");
          o.value = c.id;
          o.textContent = c.name + " — ID " + c.id + " [" + c.category + "]";
          s.appendChild(o);
        }
        document.querySelector("#hair-hint").textContent = rows.length
          ? "共 " + rows.length + " 个发型卡牌"
          : "没有匹配的发型卡牌";
      } catch (e) {
        document.querySelector("#hair-hint").textContent =
          "加载失败：" + e.message;
      }
    }
    function equipSelectedHair() {
      let sel = document.querySelector("#hair-result");
      if (!sel.value) return alert("请先选择一个发型卡牌");
      command({ action: "equip_hair", cardId: +sel.value });
    }
    let fashionCatalog = [];
    let fashionItems = [];
    async function loadFashionCatalog() {
      try {
        fashionCatalog = await (await fetch("/api/fashion-catalog")).json();
        let s = document.querySelector("#fashion-cat");
        s.innerHTML = "";
        for (let c of fashionCatalog) {
          let o = document.createElement("option");
          o.value = c.key;
          o.textContent = c.label + " (" + c.items.length + ")";
          s.appendChild(o);
        }
        s.onchange = renderFashionItems;
        renderFashionItems();
      } catch (e) {
        document.querySelector("#fashion-hint").textContent =
          "目录加载失败：" + e.message;
      }
    }
    function renderFashionItems() {
      let key = document.querySelector("#fashion-cat").value;
      let cat = fashionCatalog.find(function (c) {
        return c.key === key;
      }) || { label: "", items: [] };
      let q = document
        .querySelector("#fashion-query")
        .value.trim()
        .toLowerCase();
      fashionItems = cat.items.filter(function (it) {
        return (
          !q ||
          it.name.toLowerCase().includes(q) ||
          it.configId.toLowerCase().includes(q)
        );
      });
      let s = document.querySelector("#fashion-result");
      s.innerHTML = "";
      for (let it of fashionItems) {
        let o = document.createElement("option");
        o.value = it.configId;
        o.textContent = it.name + " — " + it.configId;
        s.appendChild(o);
      }
      document.querySelector("#fashion-hint").textContent =
        cat.label +
        "：共 " +
        cat.items.length +
        " 件" +
        (q ? "，当前匹配 " + fashionItems.length + " 件" : "");
    }
    function grantFashion(wearNow) {
      let sel = document.querySelector("#fashion-result");
      if (!sel.value) return alert("请先选择一件时装/身份装");
      command({
        action: "grant_item",
        itemConfig: sel.value,
        amount: 1,
        wearNow,
      });
    }
    function setFiveAttributes() {
      command({
        action: "set_five_attributes",
        attrStr: +document.querySelector("#attr-str").value || 0,
        attrSta: +document.querySelector("#attr-sta").value || 0,
        attrDex: +document.querySelector("#attr-dex").value || 0,
        attrIng: +document.querySelector("#attr-ing").value || 0,
        attrSpi: +document.querySelector("#attr-spi").value || 0,
      });
    }
    async function searchItems() {
      let q = document.querySelector("#item-query").value.trim();
      if (!q) return alert("请输入中文名或 ConfigID");
      let view = +document.querySelector("#item-view").value;
      let s = document.querySelector("#item-result");
      s.innerHTML = "";
      try {
        let rows = await (
          await fetch(
            "/api/items?q=" + encodeURIComponent(q) + "&view=" + view,
          )
        ).json();
        let labels = ["", "普通", "装备", "材料", "任务"];
        for (let x of rows) {
          let o = document.createElement("option");
          o.value = x.configId;
          let sex = x.needSex === 0 ? "男" : x.needSex === 1 ? "女" : "通用";
          let detail = x.wearable
            ? x.equipType + "/" + sex
            : x.script || "道具";
          o.textContent =
            x.name +
            " — " +
            x.configId +
            " [" +
            labels[x.viewId] +
            "/" +
            detail +
            "/堆叠" +
            x.maxAmount +
            "/品质" +
            x.colorLevel +
            "]";
          s.appendChild(o);
        }
        document.querySelector("#item-hint").textContent = rows.length
          ? "找到 " + rows.length + " 件；同名道具请按 ConfigID 区分"
          : "没有在 equipment.ini / tool_item.ini 找到匹配项";
      } catch (e) {
        document.querySelector("#item-hint").textContent =
          "搜索失败：" + e.message;
      }
    }
    function grantItem(wearNow) {
      let selected = document.querySelector("#item-result").value;
      let typed = document.querySelector("#item-query").value.trim();
      let amount = +document.querySelector("#item-amount").value || 1;
      if (wearNow) amount = 1;
      if (!selected && !typed)
        return alert("请先搜索并选择一件道具，或直接输入 ConfigID");
      command({
        action: "grant_item",
        itemConfig: selected || typed,
        amount,
        wearNow,
      });
    }
    async function loadAllTitles() {
      try {
        let rows = await (await fetch("/api/titles")).json();
        renderTitles(rows);
      } catch (e) {
        document.querySelector("#title-hint").textContent =
          "加载失败：" + e.message;
      }
    }
    async function searchTitles() {
      let q = document
        .querySelector("#title-query")
        .value.trim()
        .toLowerCase();
      try {
        let rows = await (await fetch("/api/titles")).json();
        let filtered = rows.filter(
          (t) => !q || t.name.toLowerCase().includes(q) || String(t.id) === q,
        );
        renderTitles(filtered);
      } catch (e) {
        document.querySelector("#title-hint").textContent =
          "搜索失败：" + e.message;
      }
    }
    async function renderTitles(rows) {
      let s = document.querySelector("#title-result");
      s.innerHTML = "";
      for (let t of rows) {
        let o = document.createElement("option");
        o.value = t.id;
        o.textContent =
          t.id + " — " + t.name + (t.type ? " [类型" + t.type + "]" : "");
        s.appendChild(o);
      }
      document.querySelector("#title-hint").textContent = rows.length
        ? "共 " + rows.length + " 个称号"
        : "没有匹配的称号";
    }
    function grantSelectedTitle() {
      let sel = document.querySelector("#title-result");
      if (!sel.value) return alert("请先选择一个称号");
      command({ action: "grant_title", titleId: +sel.value });
    }
    function wearSelectedTitle() {
      let sel = document.querySelector("#title-result");
      if (!sel.value) return alert("请先选择一个称号");
      command({ action: "wear_title", titleId: +sel.value });
    }
    async function loadMeridians() {
      try {
        let rows = await (await fetch("/api/meridians")).json();
        let s = document.querySelector("#jingmai-id");
        s.innerHTML = "";
        for (let m of rows) {
          let o = document.createElement("option");
          o.value = m.id;
          o.textContent =
            m.name + " — " + m.id + " [默认" + m.defaultMax + "周天]";
          s.appendChild(o);
        }
        document.querySelector("#jingmai-hint").textContent = rows.length
          ? "共 " + rows.length + " 条可设置经脉"
          : "经脉数据未加载";
      } catch (e) {
        document.querySelector("#jingmai-hint").textContent =
          "加载失败：" + e.message;
      }
    }
    function setJingMaiTier() {
      let id = document.querySelector("#jingmai-id").value;
      let tier = +document.querySelector("#jingmai-tier").value;
      if (!id) return alert("请先选择一条经脉");
      command({
        action: "set_jingmai_tier",
        jingMaiId: id,
        jingMaiTier: tier,
      });
    }
    function customProbe(msgId, subType) {
      let id = msgId || +document.querySelector("#probe-msgid").value;
      let sub =
        subType != null
          ? subType
          : +document.querySelector("#probe-subtype").value;
      document.querySelector("#probe-hint").textContent =
        "发送 msgId=" +
        id +
        " subType=" +
        sub +
        "，观察客户端是否弹出对应窗口或报错";
      command({ action: "custom_probe", msgId: id, subType: sub });
    }
    function keybind(args) {
      command({ action: "keybind", keyArgs: args });
    }
    function keybindBind(force) {
      let flags = [];
      if (document.querySelector("#kb-ctrl").checked) flags.push("ctrl");
      if (document.querySelector("#kb-shift").checked) flags.push("shift");
      if (document.querySelector("#kb-alt").checked) flags.push("alt");
      if (force) flags.push("force");
      keybind(
        (
          "bind " +
          document.querySelector("#kb-id").value +
          " " +
          document.querySelector("#kb-key").value.trim() +
          " " +
          flags.join(" ")
        ).trim(),
      );
    }
    function binglu(args) {
      command({ action: "binglu", bingluArgs: args });
    }
    function bingluSet() {
      let row = document.querySelector("#binglu-row").value;
      let cur = +document.querySelector("#binglu-cur").value;
      let fill = +document.querySelector("#binglu-fill").value;
      let max = document.querySelector("#binglu-max").value.trim();
      binglu(
        "set " + row + " " + cur + " " + fill + (max !== "" ? " " + max : ""),
      );
    }
    function bingluMax() {
      binglu("max " + document.querySelector("#binglu-row").value);
    }
    async function loadMounts() {
      try {
        let rows = await (await fetch("/api/mounts")).json();
        let s = document.querySelector("#mount-result");
        s.innerHTML = "";
        for (let m of rows) {
          let o = document.createElement("option");
          o.value = m.configID;
          o.textContent =
            m.name + " — " + m.configID + " [ArtPack=" + m.artPack + "]";
          s.appendChild(o);
        }
        document.querySelector("#mount-hint").textContent = rows.length
          ? "共 " + rows.length + " 个坐骑"
          : "没有加载到坐骑";
      } catch (e) {
        document.querySelector("#mount-hint").textContent =
          "加载失败：" + e.message;
      }
    }
    async function originActivateAll() {
      await command({ action: "origin_activate_all" });
      loadOriginStatus();
    }
    async function loadReputeStatus() {
      try {
        let j = await (await fetch("/api/repute/status")).json();
        let s = document.querySelector("#repute-name");
        let el = document.querySelector("#repute-state");
        if (s) {
          let keep = s.value;
          s.innerHTML = '<option value="all">全部（含隐士、江湖）</option>';
          for (let r of j.reputes || []) {
            let o = document.createElement("option");
            o.value = r.name;
            o.textContent = r.display + " [" + r.name + "] 上限" + r.maxValue;
            s.appendChild(o);
          }
          if (keep) s.value = keep;
        }
        if (el)
          el.textContent = j.loaded
            ? "已加载 " + j.reputes.length + " 项门派/势力声望"
            : "声望表未加载";
      } catch (e) {
        let el = document.querySelector("#repute-state");
        if (el) el.textContent = "读取失败：" + e.message;
      }
    }
    async function reputeApply(mode) {
      let sel = document.querySelector("#repute-name");
      let name = sel ? sel.value : "all";
      let value = +document.querySelector("#repute-value").value || 0;
      await command({
        action: "repute_apply",
        reputeName: name,
        reputeMode: mode,
        reputeValue: value,
      });
      loadReputeStatus();
    }
    async function loadOriginStatus() {
      try {
        let j = await (await fetch("/api/origin/status")).json();
        let el = document.querySelector("#origin-state");
        if (!el) return;
        el.textContent = j.loaded
          ? "已加载 " +
          j.total +
          " 个身份（C2S 协议 700/701/702/705/706 已挂载）"
          : "身份数据未加载：缺少 data/origin/origins.json";
      } catch (e) {
        let el = document.querySelector("#origin-state");
        if (el) el.textContent = "读取失败：" + e.message;
      }
    }
    async function loadMinimalLogin() {
      try {
        let j = await (await fetch("/api/minimal-login")).json();
        let el = document.querySelector("#minimal-login-state");
        if (!el) return;
        el.textContent = j.enabled
          ? "当前：开启（下次重登录生效，H-only 推送全部跳过，与 D:\\ 字节级对齐）"
          : "当前：关闭（默认，H 全量登录推送）";
      } catch (e) {
        let el = document.querySelector("#minimal-login-state");
        if (el) el.textContent = "读取失败：" + e.message;
      }
    }
    async function toggleMinimalLogin() {
      try {
        await fetch("/api/minimal-login", { method: "POST" });
      } catch (e) {
        alert("切换失败：" + e.message);
      }
      loadMinimalLogin();
    }
    async function loadIdleStances() {
      try {
        let rows = await (await fetch("/api/idle-stances")).json();
        let s = document.querySelector("#idle-stance-list");
        s.innerHTML = "";
        for (let m of rows) {
          let o = document.createElement("option");
          o.value = m.action;
          o.textContent = m.skeleton
            ? m.skeleton + " [" + m.action + "]"
            : m.action;
          s.appendChild(o);
        }
        document.querySelector("#idle-stance-hint").textContent =
          "共 " +
          rows.length +
          " 个站姿候选（来源：b_action_index.ini 全部含 stand 节）";
      } catch (e) {
        document.querySelector("#idle-stance-hint").textContent =
          "加载失败：" + e.message;
      }
    }
    function setIdleStance() {
      let s = document.querySelector("#idle-stance-list");
      if (!s.value) return alert("请先选择一个站姿");
      command({ action: "idle_stance_set", stanceAction: s.value });
    }
    // 快捷站姿按钮：空串=恢复当前站姿（重推存档，不改档）。
    function setIdleStancePreset(action) {
      command({ action: "idle_stance_set", stanceAction: action });
    }
    function mountSelected() {
      let sel = document.querySelector("#mount-result");
      if (!sel.value) return alert("请先选择一个坐骑");
      command({ action: "mount_ride", mountConfig: sel.value });
    }

    document
      .querySelector("#clone-config")
      .addEventListener("change", syncCloneFun);
    let gmBadge = document.querySelector("#gm-server-badge");
    if (gmBadge && GM_SERVER) {
      gmBadge.innerHTML = "<b>服务器</b>";
      gmBadge.appendChild(document.createTextNode(GM_SERVER_NAME));
      gmBadge.title = "GM 面板 " + GM_SERVER;
    }
    buildNav();
    refresh();
    loadReputeStatus();
    loadClones();
    loadGiftBoxes();
    loadWeather();
    loadScenes();
    loadCardCategories();
    loadAllCards();
    loadAllTitles();
    loadMeridians();
    loadHairCards();
    loadFashionCatalog();
    loadMounts();
    loadIdleStances();
    loadMinimalLogin();
    loadOriginStatus();
    loadCombatDummyTaolus();
    setInterval(refresh, 2500);
