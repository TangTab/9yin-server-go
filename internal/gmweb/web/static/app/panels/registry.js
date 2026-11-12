// registry.js —— 板块注册表（单一事实来源）。导航、路由、「其他」兜底都从
// 这里生成；新增板块只改本文件 + 对应面板模块。迁移状态见设计文档附录 A。

export const GROUPS = [
  {
    id: "role",
    name: "角色状态",
    load: () => import("./role.js"),
    sections: ["online", "five", "power", "vitals", "alignment", "titles"],
  },
  {
    id: "item",
    name: "货币与物品",
    load: () => import("./item.js"),
    sections: ["currency", "gifts", "chest"],
  },
  {
    id: "equip",
    name: "装备与铸造",
    load: () => import("./equip.js"),
    sections: ["items", "recast", "template", "smelt", "binglu"],
  },
  {
    id: "fashion",
    name: "外观与坐骑",
    load: () => import("./fashion.js"),
    sections: ["fashion-all", "fashion-grant", "cards", "hair", "mounts", "stances"],
  },
  {
    id: "skill",
    name: "武学与修炼",
    load: () => import("./skill.js"),
    sections: ["faction", "taolu", "faculty", "jingmai", "jingmai-tier"],
  },
  {
    id: "scene",
    name: "传送与场景",
    load: () => import("./scene.js"),
    sections: ["presets", "tiguan", "teleport", "weather", "clone", "worldboss"],
  },
  {
    id: "battle",
    name: "战斗与演练",
    load: () => import("./battle.js"),
    sections: ["dummy", "combat-dummy", "bot", "yufeng"],
  },
  {
    id: "params",
    name: "游戏参数",
    load: () => import("./params.js"),
    sections: ["respawn", "autosave", "movement"],
  },
  {
    id: "quest",
    name: "任务与身份",
    load: () => import("./quest.js"),
    sections: ["scene-task", "twilight", "ninjiuzhi", "origin", "repute"],
  },
  {
    id: "uipref",
    name: "界面与操作",
    load: () => import("./uipref.js"),
    sections: ["shortcuts", "keybind"],
  },
  {
    id: "debug",
    name: "调试工具",
    load: () => import("./debug.js"),
    sections: ["probe", "chat", "storytest"],
  },
  {
    id: "misc",
    name: "其他",
    load: () => import("./misc.js"),
    sections: ["mail"],
  },
];

export const SECTION_TITLES = {
  online: "在线角色",
  five: "五维属性",
  power: "实力等级",
  vitals: "体力 / 饱食 / 杀戮 / 酒量",
  alignment: "性格善恶",
  titles: "称号管理",
  currency: "碎银 / 官银 / 黄金 / 怒气",
  gifts: "礼包与物品",
  chest: "世界宝箱",
  items: "所有道具与换装",
  recast: "洗练（身上装备 · 指定部位）",
  template: "模板生成（带词条装备）",
  smelt: "冶炼",
  binglu: "兵录设置",
  "fashion-all": "时装",
  "fashion-grant": "时装 / 身份装发放",
  cards: "风物志卡牌",
  hair: "风物志头发",
  mounts: "坐骑管理",
  stances: "待机站姿（持久化）",
  faction: "门派",
  taolu: "武学套路",
  faculty: "修为",
  jingmai: "经脉",
  "jingmai-tier": "经脉周天设置",
  presets: "常用传送",
  tiguan: "踢馆传送",
  teleport: "自定义切图与场景内瞬移",
  weather: "场景天气",
  clone: "副本",
  worldboss: "世界 BOSS",
  dummy: "训练木桩",
  "combat-dummy": "添加假人（战斗）",
  bot: "伙伴系统",
  yufeng: "御风神水（大轻功）",
  respawn: "怪物刷新",
  autosave: "数据存盘",
  movement: "移动参数",
  "scene-task": "场景任务验证",
  twilight: "暮色剧情验证",
  ninjiuzhi: "九阴志全局门控",
  origin: "身份系统（一键激活全部）",
  repute: "门派声望（自定义）",
  shortcuts: "快捷栏位",
  keybind: "快捷键方案",
  probe: "S2C 自定义消息探针",
  chat: "聊天与气泡调试",
  storytest: "动作与剧情测试",
  mail: "GM 邮件",
};

export function findSection(id) {
  for (const group of GROUPS) {
    const index = group.sections.indexOf(id);
    if (index >= 0) return { group, index };
  }
  return null;
}
