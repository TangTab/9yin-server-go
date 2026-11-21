/* ============================================================================
 * AXUI v3 · 全量 TypeScript API 声明（AI 友好版）
 * ----------------------------------------------------------------------------
 * 生成自 : axui.cn v3 离线文档 + ax.esm.js 源码静态分析
 * 配套   : ./ax.esm.js（本文件声明的运行时）+ ./ax.css（必需样式）
 *
 * 【引入方式】
 *   <link rel="stylesheet" href="ax.css">
 *   <script type="module" src="ax.esm.js"></script>
 *   （ESM 导入即自动注册全部 Web Component；本文件仅是类型声明，无运行时代码）
 *
 * 【本框架三类 API —— 使用前必读】
 *   1. 组件 Component：HTML 自定义元素，标签名 = 类名去掉 Elem 前缀 ax- 后小写。
 *      例：FlagElem → <ax-flag>、PaginationElem → <ax-pagination>。
 *      JS 侧统一接口：elem.set() / elem.reset() / elem.clear() / elem.on()。
 *      共 45 个，见【组件】章节；另 ax-br / ax-bc 为纯 CSS 间隙标签（无 JS 类）。
 *   2. 模块 Module：JS 类，形如 new Dialog(elem, options)，实例方法见 AxModBase
 *      + 各模块 Instance 接口。共 40 个，见【模块】章节。
 *   3. 工具函数：window.ax 命名空间（ajax/alert/confirm/prompt/notice/storage 等）。
 *
 * 【命名规律】
 *   HTML 属性(kebab-case)  ⇄  options/propsProxy 属性(camelCase)
 *   例：cont-type 属性 ⇄ contType 选项；回调 on-show ⇄ onShow。
 *   组件的 Attributes 接口给出 kebab 写法，模块的 Options 接口给出 camel 写法。
 *
 * 【快速上手】
 *   <ax-flag type="circle" theme="error" label="99+"></ax-flag>
 *   const flag = document.querySelector('ax-flag');  // 类型: AxFlagElem
 *   flag.set({ theme: 'warn' });                     // 或 {key,value} 数组
 *   new Dialog('#btn', { heading: '标题', content: '内容' });  // 模块
 *   import { Toast } from './ax.esm.js';              // ESM 按需导入
 *   ax.confirm({ content: '确认?' }).then(r => ...);  // 全局工具
 * ========================================================================== */

/* ============================================================================
 * 0. 公共枚举类型（多个组件/模块共用的取值域）
 * ========================================================================== */

/** 颜色主题（绝大多数组件/模块通用） */
export type AxTheme =
  | 'prim'      // 主色（默认）
  | 'error'     // 错误/危险
  | 'succ'      // 成功
  | 'info'      // 信息
  | 'warn'      // 警告
  | 'issue'     // 议题
  | 'text'      // 正文色
  | 'caption';  // 说明文字色

/** 尺寸（多数组件通用） */
export type AxSize = 'xxs' | 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'xxl';

/** 定位（popup/dialog/flag 等通用，支持组合值如 'left-top'） */
export type AxPlacement =
  | 'top' | 'bottom' | 'left' | 'right' | 'center'
  | 'left-top' | 'left-bottom' | 'right-top' | 'right-bottom'
  | 'center-top' | 'center-bottom' | 'center-middle' | (string & {});

/** 内容类型（content 系列参数通用） */
export type AxContType =
  | 'text' | 'html' | 'image' | 'video' | 'audio' | 'iframe'
  | 'async' | 'ajax' | 'js' | 'css' | (string & {});

/** 形状（ax-btn 等） */
export type AxShape = 'circle' | 'square' | 'round' | 'plain' | (string & {});

/** 回调函数通用形态 */
export type AxFn = (...args: any[]) => any;

/* ============================================================================
 * 1. 组件公共基类（所有 <ax-*> 元素共享的变量与方法）
 *    文档：start/109.html「组件」
 * ========================================================================== */

/** set() 支持的四种参数形态（键值对 / 键值对数组 / 普通对象 / JSON 字符串） */
export type AxCompSetData =
  | { key: string; value: any }
  | { key: string; value: any }[]
  | Record<string, any>
  | string;

export interface AxCompBase extends HTMLElement {
  /* ---- 公共变量 ---- */
  /** attributes 转 properties 的初始值（reset 回退基准，始终不变） */
  readonly propsRaw: Record<string, any>;
  /** 随时可能变化的 properties 值 */
  readonly properties: Record<string, any>;
  /** properties 值的代理对象（读值用；写值请走 set()/setAttribute） */
  readonly propsProxy: Record<string, any>;
  /** 主节点（shadow 模式下为影子根内主容器） */
  readonly wrapEl: HTMLElement;
  /** 影子节点（模块式组件） */
  readonly shadowEl?: HTMLElement;
  /** 插槽节点（模块式组件） */
  readonly slotEl?: HTMLElement;
  /** 主模块实例（模块式组件，如 ax-pagination 的 Pagination 实例） */
  readonly ins?: Record<string, any>;
  /** 监听数据（发布/订阅） */
  readonly plans: Record<string, any>[];
  /** 组件载入页面的时间戳 */
  readonly timestamp: number;
  /** 组件载入页面的次数 */
  readonly connCount: number;
  /** 是否允许监听 attrs 属性变化 */
  readonly canListen: boolean;
  /** 是否已经是载入状态 */
  readonly connected: boolean;

  /* ---- 公共方法 ---- */
  /** 设置属性；触发 'set' 事件；支持键值对/数组/对象/JSON 字符串 */
  set(data: AxCompSetData): this;
  /** 重置组件为 propsRaw 初始属性并重渲染；触发 'reset' 事件 */
  reset(): this;
  /** 清除某 HTML 属性及值；触发 'cleared' 事件 */
  clear(attr?: string): this;

  /* ---- 事件三件套 ---- */
  /** 事件监听。公共事件：connected / disconnected / adopted / set / cleared / reset */
  on(type: string, handler: (...args: any[]) => void): this;
  /** 注销事件监听 */
  off(type: string, handler?: (...args: any[]) => void): this;
  /** 主动触发事件 */
  emit(type: string, ...params: any[]): this;
}

/* 【本文件目录】
 *   0. 公共枚举类型 / 组件公共基类 AxCompBase
 *   一、组件（Web Components，45 个，按字母排序）
 *       ax-accordion ax-alarm ax-anchors ax-avatar ax-badge ax-btn ax-buoy ax-callout ax-category ax-checkbox ax-checkboxes ax-color ax-datetime ax-deform ax-divider ax-editor ax-fields ax-file ax-flag ax-format ax-good ax-icon ax-input ax-line ax-menu ax-more ax-number ax-pagination ax-pill ax-progress ax-radio ax-radios ax-range ax-rate ax-result ax-search ax-select ax-skeleton ax-stats ax-status ax-step ax-textarea ax-tree ax-twilight ax-upload
 *   二、模块（JS Classes，40 个）
 *       Accordion Autocomplete Color Datetime Dialog Dodge Drag Drawer Dropdown Editor Flip Gesture Hover Infinite Lazy Masonry Menu Message More Pagination Panel Popup Progress Range Rate Retrieval Router Scroll Select Spy Swipe Tab Tags Toast Tooltip Tree Upload Valid Viewer Virtualize
 *   3. AxModBase 模块公共基类（位于组件段之后、模块段之前）
 *   4. 模块详细 Options/Instance —— 见「二、模块」内各小节
 *   5. 全局对象 window.ax 与工具函数
 *   6. 全局类型增强（querySelector 自动类型 / CSS 载体标签）
 *   7. CSS 工具类速查（_ 前缀工具类）
 */

/* ############################### 一、组件（Web Components） ############################### */

/* ========================================================================== */
/* ax-accordion · 手风琴折叠菜单组件                                                   */
/* 文档: components/347.html                                                    */
/* ========================================================================== */
// 模块式组件：HTML 属性与 Accordion 模块选项一一对应（kebab-case），
// 完整说明/默认值见 AxAccordionOptions（模块章节）。
/** <ax-accordion> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxAccordionAttributes {
  /** 隐藏字段的name属性 */
  name?: string;
  /** 数据源 */
  content?: any;
  /** 内容类型 默认 `'text'` */
  'cont-type'?: string;
  /** 请求content时发送的数据 默认 `{}` */
  'cont-data'?: Record<string, any>;
  /** 异步请求参数 默认 `{}` */
  ajax?: Record<string, any>;
  /** 是否完全禁用 默认 `false` */
  passive?: boolean;
  /** 初始禁用项 */
  disable?: string | number | HTMLElement | Record<string, any> | any[];
  /** 初始只读项 */
  readonly?: string | number | HTMLElement | Record<string, any> | any[];
  /** 默认 `{"value": "", "all": false, "only": true, "linkage": false}` */
  expand?: Record<string, any>;
  /** 配置指示箭头 默认 `{"enable": true, "show": "icon-down", "hide": "icon-down", "anim": "rotate180", "type": "icon", "trigger": "click"}` */
  arrow?: boolean | Record<string, any>;
  /** 配置复选/单选 默认 `{"enable": false, "type": "checkbox", "value": ""}` */
  check?: boolean | Record<string, any>;
  /** 配置编辑工具 默认 `{"enable": false, "trigger": "hover", "children": ["add", "edit", "remove"]}` */
  tools?: boolean | Record<string, any>;
  /** 配置输出 默认 `{"enable": true, "target": "", "prop": "label"}` */
  output?: boolean | Record<string, any>;
  /** 点击分支行行为 默认 `'none'` */
  'click-line'?: 'checked' | 'expanded' | 'none';
  /** 是否延迟渲染子节点 默认 `false` */
  deferred?: boolean;
  /** 事件防抖延迟(ms) 默认 `100` */
  delay?: number;
  /** 配置拖拽 默认 `{"enable": false, "exclude": ""}` */
  drag?: boolean | Record<string, any>;
  /** 默认 `{"attr": "dropnode", "global": false}` */
  drop?: Record<string, any>;
  /** 配置指示灯 默认 `{"enable": false, "placement": "bottom"}` */
  lamp?: boolean | Record<string, any>;
  /** 动画持续时间(ms) 默认 `0` */
  duration?: number;
  /** 附加CSS类名 */
  classes?: string;
  /** 头部节点布局 默认 `'check(icon|disk|cube|label|tips)custom|tools|badge|arrow'` */
  layout?: string;
  /** 默认 `{"rounded": true, "bordered": true, "divided": true, "gapped": false}` */
  appear?: Record<string, any>;
  /** 展开前执行 默认 `null` */
  'b4-expand'?: string;
  /** 折叠前执行 默认 `null` */
  'b4-collapse'?: string;
  /** 新增前执行 默认 `null` */
  'b4-add'?: string;
  /** 编辑前执行 默认 `null` */
  'b4-edit'?: string;
  /** 删除前执行 默认 `null` */
  'b4-remove'?: string;
  /** 拖拽前执行 默认 `null` */
  'b4-graft'?: string;
  /** 节点树创建完成后触发 默认 `null` */
  'on-rendered'?: string;
  /** 节点触发后触发 默认 `null` */
  'on-trigger'?: string;
  /** 新增后触发 默认 `null` */
  'on-added'?: string;
  /** 编辑后触发 默认 `null` */
  'on-edited'?: string;
  /** 删除后触发 默认 `null` */
  'on-removed'?: string;
  /** 禁用后触发 默认 `null` */
  'on-disabled'?: string;
  /** 启用后触发 默认 `null` */
  'on-enabled'?: string;
  /** 全部禁用后触发 默认 `null` */
  'on-disabledall'?: string;
  /** 全部启用后触发 默认 `null` */
  'on-enabledall'?: string;
  /** 设置只读后触发 默认 `null` */
  'on-readonly'?: string;
  /** 全部只读后触发 默认 `null` */
  'on-readonlyall'?: string;
  /** 展开时触发 默认 `null` */
  'on-expand'?: string;
  /** 展开后触发 默认 `null` */
  'on-expanded'?: string;
  /** 折叠时触发 默认 `null` */
  'on-collapse'?: string;
  /** 折叠后触发 默认 `null` */
  'on-collapsed'?: string;
  /** 全部展开时触发 默认 `null` */
  'on-expandall'?: string;
  /** 全部展开后触发 默认 `null` */
  'on-expandedAll'?: string;
  /** 全部折叠后触发 默认 `null` */
  'on-collapsedall'?: string;
  /** 选中后触发 默认 `null` */
  'on-selected'?: string;
  /** 取消选中后触发 默认 `null` */
  'on-deselected'?: string;
  /** 全部选中后触发 默认 `null` */
  'on-selectedall'?: string;
  /** 勾选后触发 默认 `null` */
  'on-checked'?: string;
  /** 取消勾选后触发 默认 `null` */
  'on-unchecked'?: string;
  /** 全部勾选后触发 默认 `null` */
  'on-checkedall'?: string;
  /** 填充brief后触发 默认 `null` */
  'on-filled'?: string;
  /** 翻页后触发 默认 `null` */
  'on-turned'?: string;
  /** 取值后触发 默认 `null` */
  'on-got'?: string;
  /** 设置值后触发 默认 `null` */
  'on-set'?: string;
  /** 清除值后触发 默认 `null` */
  'on-cleared'?: string;
  /** 拖拽完成后触发 默认 `null` */
  'on-grafted'?: string;
  /** 内容更新后触发 默认 `null` */
  'on-updatedcont'?: string;
  /** 异步请求完成后触发 默认 `null` */
  'on-request'?: string;
  /** 选中数量过少时触发 默认 `null` */
  'on-toofew'?: string;
  /** 选中数量过多时触发 默认 `null` */
  'on-toomany'?: string;
  /** 默认 `null` */
  'on-locked'?: string;
  /** 默认 `null` */
  'on-unlocked'?: string;
  'ins-name'?: string;
  'stor-name'?: string;
  /** 默认 `[]` */
  'stor-keys'?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 语言包（JSON） 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  'b4-init'?: string;
  /** 默认 `null` */
  'on-constructed'?: string;
  /** 默认 `null` */
  'on-initiate'?: string;
  /** 默认 `null` */
  'on-initiated'?: string;
  /** 默认 `null` */
  'on-destroyed'?: string;
  /** 默认 `null` */
  'on-error'?: string;
  /** 默认 `null` */
  'on-reset'?: string;
  /** 默认 `null` */
  'on-updated'?: string;
  /** 默认 `null` */
  'on-saved'?: string;
  /** 默认 `null` */
  'on-clearedcache'?: string;
  /** 默认 `null` */
  'on-updatedcache'?: string;
}

/** <ax-accordion> 元素实例（querySelector('ax-accordion') 的类型） */
export interface AxAccordionElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxAccordionAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxAccordionAttributes>;
}

/* ========================================================================== */
/* ax-alarm · 警报闪烁组件                                                          */
/* 文档: components/115.html                                                    */
/* ========================================================================== */
/** <ax-alarm> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxAlarmAttributes {
  /** 图标类名 默认 `undefined` */
  icon?: string;
  /** 图片地址 默认 `undefined` */
  image?: string;
  /** 主题风格 默认 `'error'` */
  theme?: 'prim' | 'error' | 'info' | 'warn' | 'succ' | 'issue';
  /** 尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 文本 默认 `undefined` */
  label?: string;
}

/** <ax-alarm> 元素实例（querySelector('ax-alarm') 的类型） */
export interface AxAlarmElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxAlarmAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxAlarmAttributes>;
}

/* ========================================================================== */
/* ax-anchors · 标题导航组件                                                        */
/* 文档: components/384.html                                                    */
/* ========================================================================== */
/** <ax-anchors> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxAnchorsAttributes {
  /** 是否平滑滚动 */
  smooth?: string;
  /** 标题选择器 默认 `'h2'` */
  headings?: string;
  /** 标题距离顶部的距离 */
  offset?: string;
  /** 被激活的标题或索引 */
  active?: string;
  root?: string;
}

/** <ax-anchors> 元素实例（querySelector('ax-anchors') 的类型） */
export interface AxAnchorsElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxAnchorsAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxAnchorsAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  getHeadings(...args: any[]): any;
  getData(...args: any[]): any;
  get2FillTreeNodes(...args: any[]): any;
  activate(...args: any[]): any;
  addSmooth(...args: any[]): any;
  removeSmooth(...args: any[]): any;
  smoothToActive(...args: any[]): any;
}

/* ========================================================================== */
/* ax-avatar · 头像组件                                                           */
/* 文档: components/118.html                                                    */
/* ========================================================================== */
/** <ax-avatar> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxAvatarAttributes {
  /** 头像类型，默认image图片类型，可选择text纯文本类型 默认 `'image'` */
  type?: 'text' | 'image';
  /** 主题颜色，对type=text类型有效，使用lt后缀表示浅色背景 默认 `undefined` */
  theme?: 'prim[-lt]' | 'error[-lt]' | 'succ[-lt]' | 'info[-lt]' | 'warn[-lt]' | 'issue[-lt]' | 'text[-lt]';
  /** 使用遮罩和提示文本，为空则不会显示 默认 `undefined` */
  tips?: string;
  /** 外型轮廓，默认小圆角，可选择square方或round圆形 默认 `'radius'` */
  shape?: 'radius' | 'square' | 'round';
  /** 是否使用小点徽章，默认不启用 默认 `undefined` */
  badge?: string;
  /** 是否禁用（不能点击） 默认 `undefined` */
  disabled?: 'undefined';
  /** 是否使用指示箭头，ouside表示以定位的方式脱离头像，inside是以内容的形式插入头像后方 默认 `'outside'` */
  arrow?: 'undefined';
  /** 头像图片或文本 */
  src?: 'undefined';
  /** 头像的超链接 默认 `undefined` */
  href?: string;
  /** 超链接的打开方式 默认 `undefined` */
  target?: '_blank' | '_self' | 'undefined';
  /** 超链接的rel属性 默认 `undefined` */
  rel?: 'nofollow' | 'canonical‌' | 'ugc';
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
}

/** <ax-avatar> 元素实例（querySelector('ax-avatar') 的类型） */
export interface AxAvatarElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxAvatarAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxAvatarAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  getMainEl(...args: any[]): any;
}

/* ========================================================================== */
/* ax-badge · 徽章组件                                                            */
/* 文档: components/119.html                                                    */
/* ========================================================================== */
/** <ax-badge> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxBadgeAttributes {
  /** 徽章类型，默认纯色填充 默认 `'solid'` */
  type?: 'seal' | 'solid';
  /** 主题颜色，默认红色 默认 `'error'` */
  theme?: 'prim' | 'error' | 'succ' | 'info' | 'warn' | 'issue' | 'text' | 'brief' | 'caption';
  /** 外型轮廓，默认大圆角 默认 `'round'` */
  shape?: 'radius' | 'square' | 'round' | 'drop';
  /** 是否半透明 默认 `undefined` */
  glassy?: 'undefined';
}

/** <ax-badge> 元素实例（querySelector('ax-badge') 的类型） */
export interface AxBadgeElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxBadgeAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxBadgeAttributes>;
}

/* ========================================================================== */
/* ax-btn · 按钮组件                                                              */
/* 文档: components/130.html                                                    */
/* ========================================================================== */
/** <ax-btn> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxBtnAttributes {
  /** 按钮类型，默认普通实心按钮 默认 `'solid'` */
  type?: 'seal' | 'plain' | 'solid';
  /** 按钮的颜色主体，默认没有颜色 */
  theme?: 'prim' | 'error' | 'succ' | 'info' | 'warn' | 'issue' | 'text' | 'breif' | 'caption';
  /** 按钮的文本 */
  label?: string;
  /** 跳转链接，如果启用，相当于A链接 */
  href?: string;
  /** 跳转方式，有href属性才有效 默认 `'_blank'` */
  target?: string;
  /** 按钮关联内容，有href属性才有效 */
  rel?: string;
  /** 按钮下载文件的名称，有href属性才有效 */
  download?: string;
  /** tabindex值（有:focus状态），如果有href值（是a链接）便不需要设置tab， */
  tab?: '0' | '1';
  /** 按钮的整体尺寸，xs=1.8rem，sm=2.4rem，md=4rem，lg=5.6rem ,默认md尺寸 默认 `'md'` */
  size?: 'xs' | 'sm' | 'md' | 'lg';
  /** 按钮的宽度，3表示width=height*3，1和2没有padding，neat表示与input宽度相同，full表示100%宽度 */
  width?: string;
  /** 按钮的外围轮廓形状，默认为小圆角边框 默认 `'radius'` */
  shape?: 'radius' | 'square' | 'round';
  /** 按钮加载或检测状态，ing表示正在进行，ed表示完成 */
  check?: 'ing' | 'ed';
  /** 按钮的前置图标class样式类 */
  icon?: string;
  /** 按钮的后置图标class样式类 */
  tail?: string;
  /** 按钮的意指，将在按钮尾部自动添加意指图标 */
  mean?: 'update' | 'remove' | 'expand' | 'more';
  /** 圆形图片地址 */
  disk?: string;
  /** 方形图片地址 */
  cube?: string;
  /** 普通图片地址 */
  image?: string;
  /** 信息数量提示 */
  badge?: string;
  /** 按钮提示（tooltip实例） */
  tips?: string;
  /** 是否禁用按钮 */
  disabled?: string;
  /** 是否使用按钮阴影 */
  shaded?: string;
  /** 是否使用渐变按钮 */
  gradient?: string;
  /** 是否反色（父节点为深色） */
  inverted?: string;
  /** 布尔属性，书写即生效 */
  grad?: boolean;
}

/** <ax-btn> 元素实例（querySelector('ax-btn') 的类型） */
export interface AxBtnElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxBtnAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxBtnAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  initLabel(...args: any[]): any;
  getWrap(...args: any[]): any;
}

/* ========================================================================== */
/* ax-buoy · 浮标组件                                                             */
/* 文档: components/129.html                                                    */
/* ========================================================================== */
/** <ax-buoy> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxBuoyAttributes {
  /** 浮标的文本 默认 `undefined` */
  label?: 'undefined';
  /** 浮标的值 默认 `undefined` */
  value?: 'undefined';
  /** 浮标的颜色主题 */
  theme?: 'prim' | 'succ' | 'error' | 'info' | 'warn' | 'issue' | 'text' | 'brief';
  /** 浮标的图标样式类 默认 `undefined` */
  icon?: string;
  /** 浮标的尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 浮标是否使用arrow指示箭头， 默认 `undefined` */
  arrow?: 'undefined';
  /** 是否将label和value的位置对调 默认 `undefined` */
  inverted?: 'undefined';
}

/** <ax-buoy> 元素实例（querySelector('ax-buoy') 的类型） */
export interface AxBuoyElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxBuoyAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxBuoyAttributes>;
}

/* ========================================================================== */
/* ax-callout · 告示组件                                                          */
/* 文档: components/335.html                                                    */
/* ========================================================================== */
/** <ax-callout> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxCalloutAttributes {
  /** 显示标题 */
  label?: string;
  /** 主题颜色 默认 `'info'` */
  theme?: 'prim' | 'error' | 'succ' | 'info' | 'warn' | 'issue' | 'text';
  /** 图标样式类 */
  icon?: string;
  /** 圆形图片 */
  disk?: string;
  /** 方形图片 */
  cube?: string;
  /** 普通图片 */
  image?: string;
  /** 超链接 */
  href?: string;
  /** 超链接的打开方式 */
  target?: string;
  /** 超链接的rel属性 */
  rel?: 'nofollow' | 'canonical‌' | 'ugc';
  /** 组合风格 */
  feature?: string;
  /** 是否倒计时自动关闭 */
  autoclose?: string;
  /** 是否可点击隐藏 */
  closable?: string;
  /** 悬浮时，是否不透明 */
  opaque?: string;
  /** 是否隐藏 */
  hidden?: string;
  /** 是否显示动态图标 */
  result?: string;
  /** 是否使用直角边框 */
  square?: string;
  /** 尺寸大小 */
  size?: string;
  /** 是否突出显示 */
  notable?: string;
}

/** <ax-callout> 元素实例（querySelector('ax-callout') 的类型） */
export interface AxCalloutElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxCalloutAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxCalloutAttributes>;
}

/* ========================================================================== */
/* ax-category · 板块标题组件                                                       */
/* 文档: components/394.html                                                    */
/* ========================================================================== */
/** <ax-category> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxCategoryAttributes {
  /** 图标类名 */
  icon?: string;
  /** 圆形图片地址 */
  disk?: string;
  /** 方形图片地址 */
  cube?: string;
  /** 普通图片地址 */
  image?: string;
  /** 是否启用指示箭头 默认 `false` */
  arrow?: boolean;
  /** 主标题文本 */
  label?: string;
  /** 辅助提示文本 */
  tips?: string;
  /** 主题风格 */
  theme?: 'prim' | 'error' | 'succ' | 'info' | 'issue' | 'warn' | 'text';
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
}

/** <ax-category> 元素实例（querySelector('ax-category') 的类型） */
export interface AxCategoryElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxCategoryAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxCategoryAttributes>;
}

/* ========================================================================== */
/* ax-checkbox · 复选组件                                                         */
/* 文档: components/331.html                                                    */
/* ========================================================================== */
/** <ax-checkbox> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxCheckboxAttributes {
  /** 表单字段名称 */
  name?: string;
  /** 表单字段值 */
  value?: string;
  /** 选中状态 */
  check?: string;
  /** 标签文本 */
  label?: string;
  /** 尺寸规格 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 禁用状态 */
  disabled?: boolean;
  /** 显示变体 */
  type?: string;
  /** 是否包含开关标签 */
  contained?: boolean;
  /** 透传给内部 input 的属性（JSON） */
  attrs?: string;
}

/** <ax-checkbox> 元素实例（querySelector('ax-checkbox') 的类型） */
export interface AxCheckboxElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxCheckboxAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxCheckboxAttributes>;
}

/* ========================================================================== */
/* ax-checkboxes · 复选组组件                                                      */
/* 文档: components/332.html                                                    */
/* ========================================================================== */
/** <ax-checkboxes> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxCheckboxesAttributes {
  /** 表单字段名称 */
  name?: string;
  /** 数据源 */
  content?: string;
  /** 预设选中项 */
  checked?: string;
  /** 尺寸规格 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 禁用项配置 */
  disable?: string;
  /** 显示变体类型 */
  type?: 'chera' | 'switch' | 'btn' | 'text';
  /** 布局方式 默认 `'block'` */
  layout?: 'grid' | 'flex' | 'block';
  /** 网格布局列数 默认 `4` */
  cols?: number;
  /** 外层容器样式类 */
  'wrap-classes'?: string;
  /** 单项元素样式类 */
  'item-classes'?: string;
  /** 内部表单字段样式类 */
  'input-classes'?: string;
  switch?: string;
  'on-checkedall'?: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 布尔属性，书写即生效 */
  inverted?: boolean;
}

/** <ax-checkboxes> 元素实例（querySelector('ax-checkboxes') 的类型） */
export interface AxCheckboxesElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxCheckboxesAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxCheckboxesAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  updateInputs(...args: any[]): any;
  referSwitch(...args: any[]): any;
}

/* ========================================================================== */
/* ax-color · 颜色拾取器 (Component)                                               */
/* 文档: components/434.html                                                    */
/* ========================================================================== */
/** <ax-color> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxColorAttributes {
  /** 是否禁用 */
  disabled?: string;
  /** 是否只读 */
  readonly?: string;
  /** 是否手动填值 默认 `true` */
  manual?: boolean;
  /** 是否铺满父容器 默认 `false` */
  full?: boolean;
  /** 是否自动校正面板颜色 默认 `false` */
  detectable?: boolean;
  /** 尺寸类型 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 标题文本 */
  label?: string;
  /** 占位文本 */
  placeholder?: string;
  /** 面板布局类型 默认 `'picker'` */
  layout?: 'picker' | 'palette' | 'full';
  /** 初始值 */
  value?: string;
  /** 填值格式 默认 `'hsla'` */
  format?: 'hsl' | 'hsla' | 'rgb' | 'rgba' | 'hex' | 'hexa' | 'hwb' | 'hwba';
  /** Color实例的参数 默认 `{}` */
  color?: Record<string, any>;
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 值为 JSON 字符串 */
  lang?: Record<string, any>;
}

/** <ax-color> 元素实例（querySelector('ax-color') 的类型） */
export interface AxColorElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxColorAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxColorAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  setLamp(...args: any[]): any;
}

/* ========================================================================== */
/* ax-datetime · 日期选择组件                                                       */
/* 文档: components/330.html                                                    */
/* ========================================================================== */
/** <ax-datetime> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxDatetimeAttributes {
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 字段值 */
  value?: string;
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
  format?: string;
  /** 附加 CSS 类名 */
  classes?: string;
  mode?: string;
  feature?: string;
  display?: string;
  /** 占位提示文本 */
  placeholder?: string;
  /** 标签文本 */
  label?: string;
  /** 尾部工具按钮配置（JSON 或类名） */
  tools?: string;
  'max-selection'?: string;
  'min-date'?: string;
  'max-date'?: string;
  datespan?: string;
  timespan?: string;
  rows?: string;
  cols?: string;
  'week-start'?: string;
  separator?: string;
  'btn-sel'?: string;
  'pos-sel'?: string;
  'input-sel'?: string;
  'child-sel'?: string;
  lunar?: string;
  events?: string;
  menu?: string;
  bubble?: string;
  footer?: string;
  /** 语言包（JSON） */
  lang?: string;
  /** 透传给内部 input 的属性（JSON） */
  attrs?: string;
  /** 布尔属性，书写即生效 */
  async?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 布尔属性，书写即生效 */
  'to-drawer'?: boolean;
  /** 是否撑满父容器宽度 */
  full?: boolean;
  /** 布尔属性，书写即生效 */
  multiline?: boolean;
  /** 布尔属性，书写即生效 */
  'now-hide'?: boolean;
  /** 布尔属性，书写即生效 */
  'now-show'?: boolean;
  /** 布尔属性，书写即生效 */
  'clear-show'?: boolean;
  /** 布尔属性，书写即生效 */
  'close-show'?: boolean;
  /** 布尔属性，书写即生效 */
  'cancel-show'?: boolean;
  /** 布尔属性，书写即生效 */
  'confirm-hide'?: boolean;
  /** 布尔属性，书写即生效 */
  'auto-fill'?: boolean;
  /** 布尔属性，书写即生效 */
  manual?: boolean;
  /** 布尔属性，书写即生效 */
  'auto-correct'?: boolean;
  /** 布尔属性，书写即生效 */
  required?: boolean;
  /** 布尔属性，书写即生效 */
  'fill-now'?: boolean;
  /** 布尔属性，书写即生效 */
  rtl?: boolean;
}

/** <ax-datetime> 元素实例（querySelector('ax-datetime') 的类型） */
export interface AxDatetimeElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxDatetimeAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxDatetimeAttributes>;
}

/* ========================================================================== */
/* ax-deform · 形变按钮组件                                                         */
/* 文档: components/326.html                                                    */
/* ========================================================================== */
/** <ax-deform> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxDeformAttributes {
  /** 图标类型 */
  type?: 'info' | 'menu' | 'add' | 'close' | 'left' | 'right' | 'up' | 'down' | 'menuStart' | 'menuEnd';
  /** 主体风格 */
  theme?: 'prim' | 'succ' | 'error' | 'warn' | 'info' | 'issue' | 'text' | 'brief' | 'caption';
  /** 尺寸大小 */
  size?: string;
  /** 是否实心填充 */
  filled?: string;
  /** 是否使用边框 */
  bordered?: string;
  /** 外轮廓造型 默认 `'radius'` */
  shape?: 'square' | 'radius' | 'round';
  /** 是否禁止操作 */
  disabled?: string;
  /** 布尔属性，书写即生效 */
  dark?: boolean;
}

/** <ax-deform> 元素实例（querySelector('ax-deform') 的类型） */
export interface AxDeformElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxDeformAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxDeformAttributes>;
}

/* ========================================================================== */
/* ax-divider · 分隔线组件                                                         */
/* 文档: components/324.html                                                    */
/* ========================================================================== */
/** <ax-divider> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxDividerAttributes {
  /** 文本 */
  label?: string;
  /** 上下的间距 默认 `'sm'` */
  break?: 'xxs' | 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'xxl';
  /** 文本的字号 */
  fs?: 'sm' | 'md' | 'lg';
  /** label的字号 默认 `'sm'` */
  size?: 'sm' | 'md' | 'lg';
  /** 是否启用反色模式 默认 `false` */
  inverted?: 'false' | 'true';
}

/** <ax-divider> 元素实例（querySelector('ax-divider') 的类型） */
export interface AxDividerElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxDividerAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxDividerAttributes>;
}

/* ========================================================================== */
/* ax-editor · 编辑器组件                                                          */
/* 文档: components/320.html                                                    */
/* ========================================================================== */
/** <ax-editor> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxEditorAttributes {
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 字段值 */
  value?: string;
  /** 事件防抖延迟(ms) */
  delay?: string;
  /** 附加 CSS 类名 */
  classes?: string;
  content?: string;
  'cont-type'?: string;
  'cont-data'?: string;
  /** 异步请求参数（JSON） */
  ajax?: string;
  appear?: string;
  header?: string;
  mode?: string;
  'min-height'?: string;
  'max-height'?: string;
  feature?: string;
  /** 语言包（JSON） */
  lang?: string;
  /** 布尔属性，书写即生效 */
  async?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否只读 */
  readonly?: boolean;
  /** 布尔属性，书写即生效 */
  deferred?: boolean;
}

/** <ax-editor> 元素实例（querySelector('ax-editor') 的类型） */
export interface AxEditorElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxEditorAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxEditorAttributes>;
}

/* ========================================================================== */
/* ax-fields ·                                                                */
/* （无独立文档页，参数由源码提取）                                                           */
/* ========================================================================== */
/** <ax-fields> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxFieldsAttributes {
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
  /** 标签文本 */
  label?: string;
  /** 单位文本（显示在输入框尾部） */
  unit?: string;
  connector?: string;
  align?: string;
  shape?: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否撑满父容器宽度 */
  full?: boolean;
  /** 布尔属性，书写即生效 */
  dividable?: boolean;
}

/** <ax-fields> 元素实例（querySelector('ax-fields') 的类型） */
export interface AxFieldsElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxFieldsAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxFieldsAttributes>;
}

/* ========================================================================== */
/* ax-file · 单行文件上传组件                                                         */
/* 文档: components/125.html                                                    */
/* ========================================================================== */
/** <ax-file> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxFileAttributes {
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 字段值 */
  value?: string;
  accept?: string;
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
  /** 标签文本 */
  label?: string;
  show?: string;
  /** 尾部工具按钮配置（JSON 或类名） */
  tools?: string;
  /** 透传给内部 input 的属性（JSON） */
  attrs?: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否只读 */
  readonly?: boolean;
  /** 布尔属性，书写即生效 */
  multiple?: boolean;
  /** 是否撑满父容器宽度 */
  full?: boolean;
}

/** <ax-file> 元素实例（querySelector('ax-file') 的类型） */
export interface AxFileElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxFileAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxFileAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  read2Append(...args: any[]): any;
}

/* ========================================================================== */
/* ax-flag · 角标组件                                                             */
/* 文档: components/131.html                                                    */
/* ========================================================================== */
/** <ax-flag> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxFlagAttributes {
  /** 角标类型，默认triangle（三角），circle为圆形角标，horn为羊角书签，ribbon为绸带 默认 `'triangle'` */
  type?: 'triangle' | 'circle' | 'horn' | 'ribbon';
  /** 角标的颜色主题，默认为主色（prim） 默认 `'prim'` */
  theme?: 'prim' | 'error' | 'succ' | 'info' | 'warn' | 'issue' | 'text' | 'caption';
  /** 角标的文本 */
  label?: string;
  /** 角标所在位置，默认在左侧（left） 默认 `'left'` */
  placement?: 'left' | 'right';
}

/** <ax-flag> 元素实例（querySelector('ax-flag') 的类型） */
export interface AxFlagElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxFlagAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxFlagAttributes>;
}

/* ========================================================================== */
/* ax-format · 特殊格式文本组件                                                       */
/* 文档: components/132.html                                                    */
/* ========================================================================== */
/** <ax-format> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxFormatAttributes {
  /** 特色格式文本类型 */
  type?: 'ruby' | 'break' | 'latex' | 'suffix' | 'prefix' | 'localenum' | 'plainnum' | 'padstart' | 'padend';
  /** 原文本 */
  label?: string;
  /** 使用ruby类型的帽子文本 */
  rts?: string;
  /** 使用break类型的行数 默认 `'2'` */
  lines?: string;
  /** 使用break类型的宽度 */
  width?: string;
  /** 使用break类型的字号，例如size=18px */
  size?: string;
  /** 后缀内容 默认 `{content:''}` */
  suffix?: Record<string, any>;
  /** 前缀内容 默认 `{content:''}` */
  prefix?: Record<string, any>;
  /** 本地语言 默认 `'en-US'` */
  locale?: string;
  /** 补零位数 默认 `'2'` */
  places?: string;
  /** 必须包含label和el参数 默认 `'katex.render(label, el)'` */
  engin?: string;
  info?: string;
  /** 提示气泡文本 */
  tips?: string;
  /** 值为 JSON 字符串 */
  lang?: Record<string, any>;
}

/** <ax-format> 元素实例（querySelector('ax-format') 的类型） */
export interface AxFormatElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxFormatAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxFormatAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  getRuby(...args: any[]): any;
  getBrText(...args: any[]): any;
  getLocaleNum(...args: any[]): any;
  getPlainNum(...args: any[]): any;
  getSeparator(...args: any[]): any;
  getPadstr(...args: any[]): any;
}

/* ========================================================================== */
/* ax-good · 点赞组件                                                             */
/* 文档: components/133.html                                                    */
/* ========================================================================== */
/** <ax-good> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxGoodAttributes {
  /** 头文本 */
  label?: string;
  /** 点赞值 */
  value?: string;
  /** 点赞图标 */
  icon?: string;
  /** 尾文本 */
  tips?: string;
  /** 是否禁用 */
  disabled?: string;
  /** 是否表示完成 */
  increased?: string;
}

/** <ax-good> 元素实例（querySelector('ax-good') 的类型） */
export interface AxGoodElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxGoodAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxGoodAttributes>;
}

/* ========================================================================== */
/* ax-icon · 图标组件                                                             */
/* 文档: components/310.html                                                    */
/* ========================================================================== */
/** <ax-icon> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxIconAttributes {
  /** 是否禁用 */
  disabled?: string;
  /** 提示文本 默认 `'h2'` */
  tips?: string;
  /** 标题文本 */
  label?: string;
  /** 图标类型 默认 `'icon'` */
  type?: 'icon' | 'cube' | 'disk' | 'image';
  /** 徽章文本 */
  badge?: string;
  /** 布局方向 默认 `'h'` */
  dir?: 'h' | 'v';
  /** 背景色 */
  bg?: string;
  /** 超链接 */
  href?: string;
  /** 超链接的target属性 默认 `'_blank'` */
  target?: string;
  /** 超链接的rel属性 */
  rel?: string;
  /** 超链接的download属性 */
  download?: string;
}

/** <ax-icon> 元素实例（querySelector('ax-icon') 的类型） */
export interface AxIconElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxIconAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxIconAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  setLegendBg(...args: any[]): any;
  toggleLegend(...args: any[]): any;
}

/* ========================================================================== */
/* ax-input · 单行文本输入组件                                                        */
/* 文档: components/126.html                                                    */
/* ========================================================================== */
/** <ax-input> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxInputAttributes {
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 占位提示文本 */
  placeholder?: string;
  /** 字段/组件类型 */
  type?: string;
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
  /** 字符数量限制 */
  limit?: string;
  /** 尾部工具按钮配置（JSON 或类名） */
  tools?: string;
  /** 前置图标类名（如 ax-icon 图标类） */
  icon?: string;
  /** 方形图片 url */
  cube?: string;
  /** 前置磁盘图片 url */
  disk?: string;
  /** 磁盘图片 url（替代默认图标） */
  image?: string;
  /** 尾部按钮，值为按钮文本或 HTML */
  btn?: string;
  /** 尾部按钮触发的动作（表单提交 url 等） */
  action?: string;
  /** 标签文本 */
  label?: string;
  /** 单位文本（显示在输入框尾部） */
  unit?: string;
  /** 自定义尾部内容（HTML 选择器或节点） */
  custom?: string;
  /** 输入意指说明（语义提示） */
  mean?: string;
  /** 输入任务说明（用途提示） */
  task?: string;
  /** 透传给内部 input 的属性（JSON） */
  attrs?: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否只读 */
  readonly?: boolean;
  /** 是否锁定（不可交互但可传值） */
  blocked?: boolean;
  /** 是否撑满父容器宽度 */
  full?: boolean;
}

/** <ax-input> 元素实例（querySelector('ax-input') 的类型） */
export interface AxInputElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxInputAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxInputAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  updateLimit(...args: any[]): any;
}

/* ========================================================================== */
/* ax-line · 分割线组件                                                            */
/* 文档: components/140.html                                                    */
/* ========================================================================== */
/** <ax-line> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxLineAttributes {
  /** 分割线类型，默认普通实实线（solid），可选择dashed（虚线）和slash（斜线） 默认 `'solid'` */
  type?: 'solid' | 'dashed' | 'slash';
  /** 分割线的颜色主题，默认为灰色 */
  theme?: 'prim' | 'error' | 'succ' | 'info' | 'warn' | 'issue' | 'text';
  /** 分割线的文本 */
  label?: string;
  /** 分割线文本所在位置，默认是center居中，可选择start头部或end尾部 默认 `'center'` */
  align?: 'center' | 'start' | 'end';
  /** 分割线的高度，sm=2px，md=4px，lg=8px ,默认1px */
  size?: 'sm' | 'md' | 'lg';
  /** 分割线的上下文间隙：xxs->4px;xs->8px;sm->14px;md->24px;lg->32px;xl->40px;xxl->48px */
  break?: 'xxs' | 'xs' | 'sm' | 'md' | 'lg' | 'xl' | 'xxl';
  /** 水平还是垂直分割线 默认 `'h'` */
  dir?: 'h' | 'v';
  /** 是否启用反色模式 默认 `false` */
  inverted?: 'false' | 'true';
}

/** <ax-line> 元素实例（querySelector('ax-line') 的类型） */
export interface AxLineElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxLineAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxLineAttributes>;
}

/* ========================================================================== */
/* ax-menu · 菜单组件                                                             */
/* 文档: components/402.html                                                    */
/* ========================================================================== */
// 模块式组件：HTML 属性与 Menu 模块选项一一对应（kebab-case），
// 完整说明/默认值见 AxMenuOptions（模块章节）。
/** <ax-menu> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxMenuAttributes {
  /** 颜色主题：prim/error/succ/info/warn/issue/text/caption 默认 `'light'` */
  theme?: string;
  content?: string;
  /** 默认 `'text'` */
  'cont-type'?: string;
  /** 默认 `{}` */
  'cont-data'?: Record<string, any>;
  /** 异步请求参数（JSON） 默认 `{}` */
  ajax?: Record<string, any>;
  /** 默认 `false` */
  passive?: boolean;
  /** 默认 `false` */
  expose?: boolean;
  active?: string;
  disable?: string;
  /** 默认 `false` */
  multiple?: boolean;
  /** 默认 `false` */
  'expand-all'?: boolean;
  /** 默认 `0` */
  'z-index'?: number;
  /** 默认 `'click'` */
  trigger?: string;
  /** 默认 `null` */
  'root-start'?: string;
  /** 默认 `null` */
  'id-start'?: string;
  /** 默认 `null` */
  'floor-start'?: string;
  /** 默认 `null` */
  'path-hyphen'?: string;
  /** 动画时长(ms) 默认 `0` */
  duration?: number;
  /** 默认 `false` */
  'page-close'?: boolean;
  /** 默认 `{"enable": true, "icon": ""}` */
  arrow?: Record<string, any>;
  /** 附加 CSS 类名 */
  classes?: string;
  /** 默认 `true` */
  linkage?: boolean;
  /** 默认 `false` */
  unpadded?: boolean;
  lamp?: string;
  /** 是否撑满父容器宽度 默认 `false` */
  full?: boolean;
  /** 默认 `false` */
  spill?: boolean;
  /** 默认 `'indent|legend|icon|disk|cube|image|label|custom|tips||badge|arrow'` */
  layout?: string;
  /** 默认 `{}` */
  tree?: Record<string, any>;
  /** 默认 `{"host": "", "placement": "left", "footer": false}` */
  drawer?: Record<string, any>;
  /** 默认 `{"enable": false, "headWidth": "", "bodyWidth": "", "marginLeft": "", "marginRight": "", "align": "left"}` */
  nav?: Record<string, any>;
  /** 默认 `null` */
  'on-activated'?: string;
  /** 默认 `null` */
  'on-updatedcont'?: string;
  'ins-name'?: string;
  'stor-name'?: string;
  /** 默认 `[]` */
  'stor-keys'?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 语言包（JSON） 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  'b4-init'?: string;
  /** 默认 `null` */
  'on-constructed'?: string;
  /** 默认 `null` */
  'on-initiate'?: string;
  /** 默认 `null` */
  'on-initiated'?: string;
  /** 默认 `null` */
  'on-destroyed'?: string;
  /** 默认 `null` */
  'on-error'?: string;
  /** 默认 `null` */
  'on-reset'?: string;
  /** 默认 `null` */
  'on-updated'?: string;
  /** 默认 `null` */
  'on-saved'?: string;
  /** 默认 `null` */
  'on-clearedcache'?: string;
  /** 默认 `null` */
  'on-updatedcache'?: string;
}

/** <ax-menu> 元素实例（querySelector('ax-menu') 的类型） */
export interface AxMenuElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxMenuAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxMenuAttributes>;
}

/* ========================================================================== */
/* ax-more ·                                                                  */
/* （无独立文档页，参数由源码提取）                                                           */
/* ========================================================================== */
// 模块式组件：HTML 属性与 More 模块选项一一对应（kebab-case），
// 完整说明/默认值见 AxMoreOptions（模块章节）。
/** <ax-more> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxMoreAttributes {
  /** 默认 `["content", "length", "folded"]` */
  'stor-keys'?: any[];
  content?: string;
  /** 默认 `60` */
  length?: number;
  /** 默认 `true` */
  folded?: boolean;
  /** 默认 `null` */
  'text-fold'?: string;
  /** 默认 `null` */
  'text-unfold'?: string;
  /** 默认 `'c-prim'` */
  'class-fold'?: string;
  /** 默认 `'c-prim'` */
  'class-unfold'?: string;
  /** 默认 `null` */
  'on-fold'?: string;
  /** 默认 `null` */
  'on-unfold'?: string;
  /** 默认 `null` */
  'on-updatedcont'?: string;
  'ins-name'?: string;
  'stor-name'?: string;
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 语言包（JSON） 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  'b4-init'?: string;
  /** 默认 `null` */
  'on-constructed'?: string;
  /** 默认 `null` */
  'on-initiate'?: string;
  /** 默认 `null` */
  'on-initiated'?: string;
  /** 默认 `null` */
  'on-destroyed'?: string;
  /** 默认 `null` */
  'on-error'?: string;
  /** 默认 `null` */
  'on-reset'?: string;
  /** 默认 `null` */
  'on-updated'?: string;
  /** 默认 `null` */
  'on-saved'?: string;
  /** 默认 `null` */
  'on-clearedcache'?: string;
  /** 默认 `null` */
  'on-updatedcache'?: string;
}

/** <ax-more> 元素实例（querySelector('ax-more') 的类型） */
export interface AxMoreElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxMoreAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxMoreAttributes>;
}

/* ========================================================================== */
/* ax-number · 数值输入组件                                                         */
/* 文档: components/194.html                                                    */
/* ========================================================================== */
/** <ax-number> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxNumberAttributes {
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 占位提示文本 */
  placeholder?: string;
  layout?: string;
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
  max?: string;
  min?: string;
  step?: string;
  /** 标签文本 */
  label?: string;
  /** 透传给内部 input 的属性（JSON） */
  attrs?: string;
  'on-exceeded'?: string;
  /** 提示气泡文本 */
  tips?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否只读 */
  readonly?: boolean;
  /** 是否撑满父容器宽度 */
  full?: boolean;
}

/** <ax-number> 元素实例（querySelector('ax-number') 的类型） */
export interface AxNumberElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxNumberAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxNumberAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  updateTips(...args: any[]): any;
  setBtnDisabled(...args: any[]): any;
}

/* ========================================================================== */
/* ax-pagination · 分页组件                                                       */
/* 文档: components/382.html                                                    */
/* ========================================================================== */
// 模块式组件：HTML 属性与 Pagination 模块选项一一对应（kebab-case），
// 完整说明/默认值见 AxPaginationOptions（模块章节）。
/** <ax-pagination> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxPaginationAttributes {
  /** 列表容器选择器 */
  'list-sel'?: string;
  /** 每页显示的条目数 默认 `10` */
  count?: number;
  /** 可选的每页条目数，如：'10,20,30' */
  counts?: string;
  /** 默认 `{}` */
  dropdown?: Record<string, any>;
  /** 当前页码 默认 `1` */
  current?: number;
  /** 分页器尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 是否展开显示所有页码 默认 `false` */
  expanded?: boolean;
  /** 可见页码配置 默认 `{"edge": 2, "center": 2}` */
  visible?: Record<string, any>;
  /** 分页器对齐方式 */
  align?: string;
  /** 分页器样式类型 */
  type?: string;
  /** 是否启用弹性布局 默认 `false` */
  flexible?: boolean;
  /** 分页器结构布局 默认 `'first|prev|pages|next|last'` */
  layout?: string;
  /** 自定义 CSS 类名 */
  classes?: string;
  /** 跳转防抖延迟(ms) 默认 `100` */
  delay?: number;
  /** 分页器各部分节点名称配置 默认 `{"main": "div", "total": "span", "count": "span", "locate": "span", "pages": "span", "tips": "span", "ellipsis": "span", "prev": "a", "next": "a", "first": "a", "last": "a", "item": "a", "section": "section"}` */
  names?: Record<string, any>;
  /** 数据源，默认自带1000条数据 默认 `1000` */
  content?: number | string | any[];
  /** 内容类型 */
  'cont-type'?: string;
  /** 请求参数配置 默认 `{}` */
  'cont-data'?: Record<string, any>;
  /** AJAX 请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /** 模板字符串 */
  'tpl-str'?: string;
  /** 模板引擎配置 默认 `null` */
  'tpl-eng'?: (...args: any[]) => any | null;
  /** 页码跳转前等待 默认 `null` */
  'b4-locate'?: string;
  /** 列表渲染前等待 默认 `null` */
  'b4-renderlist'?: string;
  /** 页码跳转完成回调 默认 `null` */
  'on-located'?: string;
  /** 内容获取完成回调 默认 `null` */
  'on-gotcont'?: string;
  /** 跳转首页回调 默认 `null` */
  'on-tofirst'?: string;
  /** 跳转尾页回调 默认 `null` */
  'on-tolast'?: string;
  /** 页码超出范围回调 默认 `null` */
  'on-exceeded'?: string;
  /** 列表渲染完成回调 默认 `null` */
  'on-renderedlist'?: string;
  /** 页码渲染完成回调 默认 `null` */
  'on-renderedpages'?: string;
  'ins-name'?: string;
  'stor-name'?: string;
  /** 默认 `[]` */
  'stor-keys'?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 语言包（JSON） 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  'b4-init'?: string;
  /** 默认 `null` */
  'on-constructed'?: string;
  /** 默认 `null` */
  'on-initiate'?: string;
  /** 默认 `null` */
  'on-initiated'?: string;
  /** 默认 `null` */
  'on-destroyed'?: string;
  /** 默认 `null` */
  'on-error'?: string;
  /** 默认 `null` */
  'on-reset'?: string;
  /** 默认 `null` */
  'on-updated'?: string;
  /** 默认 `null` */
  'on-saved'?: string;
  /** 默认 `null` */
  'on-clearedcache'?: string;
  /** 默认 `null` */
  'on-updatedcache'?: string;
}

/** <ax-pagination> 元素实例（querySelector('ax-pagination') 的类型） */
export interface AxPaginationElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxPaginationAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxPaginationAttributes>;
}

/* ========================================================================== */
/* ax-pill · 胶囊tab组件                                                          */
/* 文档: components/424.html                                                    */
/* ========================================================================== */
/** <ax-pill> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxPillAttributes {
  active?: string;
  disable?: string;
  refer?: string;
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  'on-changed'?: string;
  /** 是否禁用 */
  disabled?: boolean;
}

/** <ax-pill> 元素实例（querySelector('ax-pill') 的类型） */
export interface AxPillElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxPillAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxPillAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  getLegend(...args: any[]): any;
  getItemsData(...args: any[]): any;
  getItems(...args: any[]): any;
  active(...args: any[]): any;
  disable(...args: any[]): any;
  setTranslate(...args: any[]): any;
}

/* ========================================================================== */
/* ax-progress · 进度条组件                                                        */
/* 文档: components/300.html                                                    */
/* ========================================================================== */
// 模块式组件：HTML 属性与 Progress 模块选项一一对应（kebab-case），
// 完整说明/默认值见 AxProgressOptions（模块章节）。
/** <ax-progress> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxProgressAttributes {
  /** 进度条类型 默认 `'line'` */
  type?: 'line' | 'circle' | 'semicircle' | 'gapcircle';
  /** 颜色风格 */
  theme?: string;
  /** 是否使用渐变色 默认 `false` */
  gradient?: boolean;
  /** 线条端点形状 默认 `'round'` */
  linecap?: 'round' | 'square';
  /** 进度条宽度 */
  size?: string;
  /** 进度条厚度 */
  thk?: string;
  /** 追加样式类 */
  classes?: string;
  /** 取值范围 默认 `[0, 100]` */
  range?: any[];
  /** 初始进度值 默认 `0` */
  value?: number;
  /** 动画曲线 默认 `'easeOut'` */
  curve?: 'easeOut' | 'linear' | 'easeIn' | 'easeInOut' | 'easeOutIn';
  /** 动画时长(ms) 默认 `0` */
  duration?: number;
  /** 是否旋转 默认 `false` */
  rotate?: boolean;
  /** 是否隐藏轨道 默认 `false` */
  simple?: boolean;
  /** 是否显示控制按钮 默认 `false` */
  control?: boolean;
  /** 进度阶段设置 默认 `[]` */
  steps?: any[];
  /** 标签文本 默认 `{"enable": true, "template": "", "placement": "", "unit": "%", "tips": false}` */
  label?: Record<string, any>;
  /** 默认 `{"type": "percent", "trunc": true, "emphatic": false}` */
  number?: Record<string, any>;
  /** 设置值前等待函数 默认 `null` */
  'b4-set'?: string;
  /** 进程开始回调 默认 `null` */
  'on-start'?: string;
  /** 进程进行中回调 默认 `null` */
  'on-move'?: string;
  /** 进程结束回调 默认 `null` */
  'on-end'?: string;
  /** 进度为0%回调 默认 `null` */
  'on-zero'?: string;
  /** 进度完成回调 默认 `null` */
  'on-completed'?: string;
  /** 设置值后回调 默认 `null` */
  'on-set'?: string;
  /** 暂停后回调 默认 `null` */
  'on-pause'?: string;
  /** 继续进行回调 默认 `null` */
  'on-continue'?: string;
  /** 恢复初始值回调 默认 `null` */
  'on-restored'?: string;
  'ins-name'?: string;
  'stor-name'?: string;
  /** 默认 `[]` */
  'stor-keys'?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 语言包（JSON） 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  'b4-init'?: string;
  /** 默认 `null` */
  'on-constructed'?: string;
  /** 默认 `null` */
  'on-initiate'?: string;
  /** 默认 `null` */
  'on-initiated'?: string;
  /** 默认 `null` */
  'on-destroyed'?: string;
  /** 默认 `null` */
  'on-error'?: string;
  /** 默认 `null` */
  'on-reset'?: string;
  /** 默认 `null` */
  'on-updated'?: string;
  /** 默认 `null` */
  'on-saved'?: string;
  /** 默认 `null` */
  'on-clearedcache'?: string;
  /** 默认 `null` */
  'on-updatedcache'?: string;
}

/** <ax-progress> 元素实例（querySelector('ax-progress') 的类型） */
export interface AxProgressElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxProgressAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxProgressAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  getVal(...args: any[]): any;
}

/* ========================================================================== */
/* ax-radio · 单选表单域组件                                                         */
/* 文档: components/297.html                                                    */
/* ========================================================================== */
/** <ax-radio> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxRadioAttributes {
  /** 表单字段名称 */
  name?: string;
  /** 表单字段值 */
  value?: string;
  /** 选中状态 */
  check?: string;
  /** 标签文本 */
  label?: string;
  /** 尺寸规格 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 禁用状态 */
  disabled?: boolean;
  /** 显示变体类型 */
  type?: string;
  /** 包含开关标签 */
  contained?: boolean;
  /** 透传给内部 input 的属性（JSON） */
  attrs?: string;
}

/** <ax-radio> 元素实例（querySelector('ax-radio') 的类型） */
export interface AxRadioElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxRadioAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxRadioAttributes>;
}

/* ========================================================================== */
/* ax-radios · 单选组组件                                                          */
/* 文档: components/298.html                                                    */
/* ========================================================================== */
/** <ax-radios> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxRadiosAttributes {
  /** 表单字段名称 */
  name?: string;
  /** 数据源 */
  content?: string;
  /** 预设选中项 */
  checked?: string;
  /** 尺寸规格 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 禁用项配置 */
  disable?: string;
  /** 显示变体类型 */
  type?: 'chera' | 'switch' | 'btn' | 'text';
  /** 布局方式 默认 `'block'` */
  layout?: 'grid' | 'flex' | 'block';
  /** 网格布局列数 默认 `4` */
  cols?: number;
  /** 外层容器样式类 */
  'wrap-classes'?: string;
  /** 单项元素样式类 */
  'item-classes'?: string;
  /** 内部表单字段样式类 */
  'input-classes'?: string;
  'on-checked'?: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 布尔属性，书写即生效 */
  inverted?: boolean;
}

/** <ax-radios> 元素实例（querySelector('ax-radios') 的类型） */
export interface AxRadiosElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxRadiosAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxRadiosAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  updateInputs(...args: any[]): any;
}

/* ========================================================================== */
/* ax-range · 范围输入组件                                                          */
/* 文档: components/296.html                                                    */
/* ========================================================================== */
/** <ax-range> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxRangeAttributes {
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 字段值 */
  value?: string;
  step?: string;
  max?: string;
  min?: string;
  flow?: string;
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
  /** 附加 CSS 类名 */
  classes?: string;
  separator?: string;
  hyphen?: string;
  fence?: string;
  button?: string;
  ruler?: string;
  result?: string;
  /** 语言包（JSON） */
  lang?: string;
  /** 布尔属性，书写即生效 */
  async?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否撑满父容器宽度 */
  full?: boolean;
  /** 布尔属性，书写即生效 */
  'limit-show'?: boolean;
  /** 布尔属性，书写即生效 */
  'tip-show'?: boolean;
  /** 布尔属性，书写即生效 */
  multiple?: boolean;
  /** 布尔属性，书写即生效 */
  locked?: boolean;
  /** 布尔属性，书写即生效 */
  rtl?: boolean;
}

/** <ax-range> 元素实例（querySelector('ax-range') 的类型） */
export interface AxRangeElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxRangeAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxRangeAttributes>;
}

/* ========================================================================== */
/* ax-rate · 星级评分组件                                                           */
/* 文档: components/292.html                                                    */
/* ========================================================================== */
// 模块式组件：HTML 属性与 Rate 模块选项一一对应（kebab-case），
// 完整说明/默认值见 AxRateOptions（模块章节）。
/** <ax-rate> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxRateAttributes {
  /** 隐藏input的name属性 */
  name?: string;
  /** 初始评分值 默认 `0` */
  value?: number;
  /** 填充值类型 默认 `'value'` */
  fill?: 'value' | 'stars';
  /** 左侧主标题 */
  heading?: string;
  /** 是否支持半星 默认 `false` */
  half?: boolean;
  /** 评星总数 默认 `5` */
  count?: number;
  /** 单星代表分值 默认 `1` */
  increment?: number;
  /** 默认 `{"enable": false}` */
  tooltip?: Record<string, any>;
  /** 默认 `{"enable": false}` */
  result?: Record<string, any>;
  /** 标签文本 默认 `{"enable": false, "format": ""}` */
  label?: Record<string, any>;
  /** 是否可清除 默认 `false` */
  clearable?: boolean;
  /** 星星尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 星星图标类名 默认 `'icon-star-f'` */
  star?: string;
  /** 星星图标类型 默认 `'icon'` */
  type?: 'icon' | 'image';
  /** 星星对照地图 默认 `[]` */
  map?: any[];
  /** 是否只点亮单个 默认 `false` */
  highlight?: boolean;
  /** 是否忽略灰色星 默认 `false` */
  omitted?: boolean;
  /** 追加样式类 */
  classes?: string;
  /** 是否禁用 默认 `false` */
  disabled?: boolean;
  /** 是否只读 默认 `false` */
  readonly?: boolean;
  /** 设置新值后回调 默认 `null` */
  'on-set'?: string;
  /** 值变化后回调 默认 `null` */
  'on-changed'?: string;
  /** 恢复初始值回调 默认 `null` */
  'on-restored'?: string;
  /** 清除归零后回调 默认 `null` */
  'on-cleared'?: string;
  'ins-name'?: string;
  'stor-name'?: string;
  /** 默认 `[]` */
  'stor-keys'?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 语言包（JSON） 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  'b4-init'?: string;
  /** 默认 `null` */
  'on-constructed'?: string;
  /** 默认 `null` */
  'on-initiate'?: string;
  /** 默认 `null` */
  'on-initiated'?: string;
  /** 默认 `null` */
  'on-destroyed'?: string;
  /** 默认 `null` */
  'on-error'?: string;
  /** 默认 `null` */
  'on-reset'?: string;
  /** 默认 `null` */
  'on-updated'?: string;
  /** 默认 `null` */
  'on-saved'?: string;
  /** 默认 `null` */
  'on-clearedcache'?: string;
  /** 默认 `null` */
  'on-updatedcache'?: string;
}

/** <ax-rate> 元素实例（querySelector('ax-rate') 的类型） */
export interface AxRateElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxRateAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxRateAttributes>;
}

/* ========================================================================== */
/* ax-result · 结果和状态组件                                                        */
/* 文档: components/143.html                                                    */
/* ========================================================================== */
/** <ax-result> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxResultAttributes {
  /** 图标类型，默认svg 默认 `'svg'` */
  type?: 'svg' | 'font';
  /** 结果状态，默认info */
  to?: 'error' | 'succ' | 'info' | 'warn';
  /** 图标字体造型，type=font下有效 默认 `'thick'` */
  shape?: 'thin' | 'thick' | 'fill';
  /** 图标下方文本 */
  label?: string;
}

/** <ax-result> 元素实例（querySelector('ax-result') 的类型） */
export interface AxResultElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxResultAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxResultAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  getIcon(...args: any[]): any;
}

/* ========================================================================== */
/* ax-search · 搜索表单域组件                                                        */
/* 文档: components/287.html                                                    */
/* ========================================================================== */
/** <ax-search> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxSearchAttributes {
  /** 是否禁用 */
  disabled?: string;
  /** 是否全宽显示 */
  full?: string;
  /** 是否突出显示 */
  notable?: string;
  /** 尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 标题 */
  label?: string;
  /** 边框形状 默认 `'radius'` */
  shape?: 'radius' | 'round' | 'square';
  feature?: string;
}

/** <ax-search> 元素实例（querySelector('ax-search') 的类型） */
export interface AxSearchElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxSearchAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxSearchAttributes>;
}

/* ========================================================================== */
/* ax-select · 下拉菜单组件                                                         */
/* 文档: components/279.html                                                    */
/* ========================================================================== */
/** <ax-select> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxSelectAttributes {
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 字段值 */
  value?: string;
  field?: string;
  /** 字段/组件类型 */
  type?: string;
  exclude?: string;
  min?: string;
  max?: string;
  span?: string;
  content?: string;
  'cont-type'?: string;
  'cont-Data'?: string;
  /** 异步请求参数（JSON） */
  ajax?: string;
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
  'max-height'?: string;
  search?: string;
  /** 尾部工具按钮配置（JSON 或类名） */
  tools?: string;
  popup?: string;
  /** 语言包（JSON） */
  lang?: string;
  /** 透传给内部 input 的属性（JSON） */
  attrs?: string;
  /** 布尔属性，书写即生效 */
  async?: boolean;
  /** 布尔属性，书写即生效 */
  manual?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否只读 */
  readonly?: boolean;
  /** 是否撑满父容器宽度 */
  full?: boolean;
  /** 布尔属性，书写即生效 */
  multiple?: boolean;
  /** 布尔属性，书写即生效 */
  sliced?: boolean;
  /** 布尔属性，书写即生效 */
  removable?: boolean;
  /** 布尔属性，书写即生效 */
  unique?: boolean;
  /** 布尔属性，书写即生效 */
  collapse?: boolean;
  /** 状态标记 */
  status?: boolean;
  /** 布尔属性，书写即生效 */
  'auto-width'?: boolean;
}

/** <ax-select> 元素实例（querySelector('ax-select') 的类型） */
export interface AxSelectElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxSelectAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxSelectAttributes>;
}

/* ========================================================================== */
/* ax-skeleton · 骨架屏组件                                                        */
/* 文档: components/398.html                                                    */
/* ========================================================================== */
/** <ax-skeleton> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxSkeletonAttributes {
  /** 骨架类型 */
  type?: string;
  /** 主题颜色 默认 `'text'` */
  theme?: string;
  /** 列数 默认 `null` */
  cols?: string;
  /** 行数 默认 `1` */
  rows?: number;
  /** 动画效果 */
  anim?: string;
  /** 轮廓外型 默认 `'radius'` */
  shape?: 'round' | 'square' | 'radius';
}

/** <ax-skeleton> 元素实例（querySelector('ax-skeleton') 的类型） */
export interface AxSkeletonElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxSkeletonAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxSkeletonAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  getRows(...args: any[]): any;
  getCols(...args: any[]): any;
  getSkelStr(...args: any[]): any;
}

/* ========================================================================== */
/* ax-stats · 数据统计组件                                                          */
/* 文档: components/276.html                                                    */
/* ========================================================================== */
/** <ax-stats> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxStatsAttributes {
  /** 是否禁用 */
  disabled?: string;
  /** 是否反向布局 */
  flipped?: string;
  /** 布局方向 默认 `'h'` */
  flow?: 'h' | 'v';
  /** 徽章文本 */
  badge?: string;
  /** 主标题 */
  label?: string;
  /** 单位文本 */
  unit?: string;
  /** 提示文本 */
  tips?: string;
  /** 方形图片地址 */
  cube?: string;
  /** 圆形图片地址 */
  disk?: string;
  /** 普通图片地址 */
  image?: string;
  /** 图标类名 */
  icon?: string;
  /** 前缀文本 */
  prefix?: string;
  /** 后缀文本 */
  suffix?: string;
}

/** <ax-stats> 元素实例（querySelector('ax-stats') 的类型） */
export interface AxStatsElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxStatsAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxStatsAttributes>;
}

/* ========================================================================== */
/* ax-status · 状态组件                                                           */
/* 文档: components/391.html                                                    */
/* ========================================================================== */
/** <ax-status> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxStatusAttributes {
  /** 指示灯类型 默认 `'dot'` */
  type?: 'dot' | 'circle';
  /** 主题风格 默认 `'prim'` */
  current?: 'error' | 'succ' | 'info' | 'warn' | 'issue' | 'confirm' | 'cancel' | 'forbid';
  /** 标题文本 */
  label?: string;
  /** 指示灯颜色 */
  lamp?: string;
}

/** <ax-status> 元素实例（querySelector('ax-status') 的类型） */
export interface AxStatusElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxStatusAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxStatusAttributes>;
}

/* ========================================================================== */
/* ax-step · 步骤组件                                                             */
/* 文档: components/390.html                                                    */
/* ========================================================================== */
/** <ax-step> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxStepAttributes {
  /** 节点类型 默认 `'dot'` */
  type?: 'dot' | 'dot-fill' | 'circle' | 'circle-fill' | 'plain';
  /** 主题风格 默认 `'prim'` */
  theme?: 'prim' | 'error' | 'succ' | 'info' | 'warn' | 'issue' | 'text';
  /** 是否显示头部 默认 `false` */
  'head-show'?: boolean;
  /** 是否显示主体文本 默认 `false` */
  'body-show'?: boolean;
  /** 真实步骤 */
  active?: string;
  /** 报错的步骤 */
  error?: string;
  /** 当前active步骤是否表示未完成 默认 `false` */
  pedding?: boolean;
  /** 具体数据 */
  content?: string;
  /** 布局方向，h指水平，v只垂直 默认 `'h'` */
  dir?: 'h' | 'v';
  /** 是否两头对齐（dir=h时有效） 默认 `false` */
  justify?: boolean;
  /** 是否翻转顺序（dir=v时有效） 默认 `false` */
  inverted?: boolean;
}

/** <ax-step> 元素实例（querySelector('ax-step') 的类型） */
export interface AxStepElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxStepAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxStepAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  getArrContent(...args: any[]): any;
  stdContent(...args: any[]): any;
  getLegend(...args: any[]): any;
  setLegendType(...args: any[]): any;
}

/* ========================================================================== */
/* ax-textarea · 多行文本输入组件                                                     */
/* 文档: components/127.html                                                    */
/* ========================================================================== */
/** <ax-textarea> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxTextareaAttributes {
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 占位提示文本 */
  placeholder?: string;
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
  /** 尾部工具按钮配置（JSON 或类名） */
  tools?: string;
  /** 字符数量限制 */
  limit?: string;
  /** 标签文本 */
  label?: string;
  /** 输入意指说明（语义提示） */
  mean?: string;
  /** 输入任务说明（用途提示） */
  task?: string;
  /** 透传给内部 input 的属性（JSON） */
  attrs?: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否只读 */
  readonly?: boolean;
  /** 布尔属性，书写即生效 */
  single?: boolean;
  /** 是否撑满父容器宽度 */
  full?: boolean;
}

/** <ax-textarea> 元素实例（querySelector('ax-textarea') 的类型） */
export interface AxTextareaElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxTextareaAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxTextareaAttributes>;
  /* ---- 组件特有方法（源码提取，用法详见文档） ---- */
  updateLimit(...args: any[]): any;
}

/* ========================================================================== */
/* ax-tree · 树型菜单组件                                                           */
/* 文档: components/270.html                                                    */
/* ========================================================================== */
// 模块式组件：HTML 属性与 Tree 模块选项一一对应（kebab-case），
// 完整说明/默认值见 AxTreeOptions（模块章节）。
/** <ax-tree> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxTreeAttributes {
  /** 隐藏input的name属性 */
  name?: string;
  /** 初始值 */
  value?: string;
  /** 数据源内容 */
  content?: string;
  /** 内容类型 默认 `'text'` */
  'cont-type'?: string;
  /** 请求数据参数 默认 `{}` */
  'cont-data'?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /** 是否完全禁止使用 默认 `false` */
  passive?: boolean;
  /** 初始化禁用项 */
  disable?: string | number | HTMLElement | Record<string, any> | any[];
  /** 初始化只读项 */
  readonly?: string | number | HTMLElement | Record<string, any> | any[];
  /** 限制分支最少数量 默认 `0` */
  min?: number;
  /** 限制分支最多数量 默认 `0` */
  max?: number;
  /** 是否裁剪掉超出的部分 默认 `true` */
  sliced?: boolean;
  /** 默认 `{"value": "", "all": false, "only": true, "linkage": false}` */
  expand?: Record<string, any>;
  /** 默认 `{"enable": true, "only": true, "value": "", "linkage": false, "span": "tree", "addSelected": false}` */
  select?: Record<string, any>;
  /** 默认 `{"enable": true, "show": "icon-right", "hide": "icon-right", "anim": "rotate90", "type": "icon", "trigger": "click"}` */
  arrow?: Record<string, any>;
  /** 默认 `{"enable": false, "parent": ["icon-folder", "icon-folder-open"], "child": "icon-file-text", "type": "icon"}` */
  legend?: Record<string, any>;
  /** 默认 `{"enable": false, "type": "checkbox", "value": "", "min": 0, "max": 0, "sliced": true, "linkage": true, "only": false, "span": "tree"}` */
  check?: Record<string, any>;
  /** 尾部工具按钮配置（JSON 或类名） 默认 `{"enable": false, "trigger": "hover", "children": ["folder", "file", "edit", "remove"]}` */
  tools?: Record<string, any>;
  /** 默认 `{"enable": true, "target": "", "type": "", "from": "selected", "field": "label", "instant": true, "autosort": false}` */
  output?: Record<string, any>;
  /** 默认 `{"deep": 0, "start": 0, "count": 0, "order": "id asc", "where": ""}` */
  'sql-data'?: Record<string, any>;
  /** 默认 `{"enable": false, "count": 10, "override": false, "exception": false}` */
  paginated?: Record<string, any>;
  /** 触发事件的防抖延迟时间(ms) 默认 `100` */
  delay?: number;
  /** 是否暂时不渲染子节点 默认 `false` */
  deferred?: boolean;
  /** 默认 `{"enable": false, "span": "blank", "mean": "selected"}` */
  shortcut?: Record<string, any>;
  /** 默认 `{"target": "", "trigger": "input", "delay": 500, "value": "", "limit": 0, "fuzzy": true, "ignore": true}` */
  search?: Record<string, any>;
  /** 默认 `{"enable": false, "exclude": ""}` */
  drag?: Record<string, any>;
  /** 默认 `{"attr": "dropnode", "global": false}` */
  drop?: Record<string, any>;
  /** 是否显示连线 默认 `false` */
  chain?: boolean;
  /** 是否自动填充content 默认 `true` */
  'auto-fill'?: boolean;
  /** 顶层分支的编号 默认 `null` */
  'root-start'?: number;
  /** 分支id开始编号 默认 `null` */
  'id-start'?: number;
  /** 默认 `null` */
  'floor-start'?: string;
  /** 默认 `null` */
  'path-hyphen'?: string;
  /** 动画时长(ms) 默认 `0` */
  duration?: number;
  /** 默认 `{"enable": false, "type": "popup"}` */
  bubble?: Record<string, any>;
  /** 附加 CSS 类名 */
  classes?: string;
  /** 默认 `'inline'` */
  display?: string;
  /** head部分的节点布局 默认 `'indent|arrow|check|legend|icon|disk|cube|image|label|badge|tips|custom|tools'` */
  layout?: string;
  /** 默认 `{"type": "", "expand": {"all": true, "only": false}, "check": {"linkage": true, "only": false, "span": "tree"}, "layout": {"dropdown": "indent|arrow|icon|disk|cube|image|label|holder|tips|badge", "select": "indent|arrow|icon|disk|cube|image|label|holder|tips|badge", "check": "indent|arrow|check|icon|disk|cube|image|label|holder|tips|badge", "button": "icon|arrow|disk|cube|image|label|tips|badge", "list": "indent|arrow|icon|disk|cube|image(label|badge|tips|custom)"}}` */
  feature?: Record<string, any>;
  /** 展开前等待函数 默认 `null` */
  'b4-expand'?: string;
  /** 折叠前等待函数 默认 `null` */
  'b4-collapse'?: string;
  /** 添加前等待函数 默认 `null` */
  'b4-add'?: string;
  /** 编辑前等待函数 默认 `null` */
  'b4-edit'?: string;
  /** 删除前等待函数 默认 `null` */
  'b4-remove'?: string;
  /** 拖拽前等待函数 默认 `null` */
  'b4-graft'?: string;
  /** 渲染完成回调 默认 `null` */
  'on-rendered'?: string;
  /** 默认 `null` */
  'on-insertitem'?: string;
  /** 默认 `null` */
  'on-insertitems'?: string;
  /** 节点触发回调 默认 `null` */
  'on-trigger'?: string;
  /** 添加后回调 默认 `null` */
  'on-added'?: string;
  /** 编辑后回调 默认 `null` */
  'on-edited'?: string;
  /** 删除后回调 默认 `null` */
  'on-removed'?: string;
  /** 禁用回调 默认 `null` */
  'on-disabled'?: string;
  /** 启用回调 默认 `null` */
  'on-enabled'?: string;
  /** 全部禁用回调 默认 `null` */
  'on-disabledall'?: string;
  /** 全部启用回调 默认 `null` */
  'on-enabledall'?: string;
  /** 只读回调 默认 `null` */
  'on-readonly'?: string;
  /** 全部只读回调 默认 `null` */
  'on-readonlyall'?: string;
  /** 展开前回调 默认 `null` */
  'on-expand'?: string;
  /** 展开后回调 默认 `null` */
  'on-expanded'?: string;
  /** 折叠前回调 默认 `null` */
  'on-collapse'?: string;
  /** 折叠后回调 默认 `null` */
  'on-collapsed'?: string;
  /** 全部展开前回调 默认 `null` */
  'on-expandall'?: string;
  /** 全部展开后回调 默认 `null` */
  'on-expandedAll'?: string;
  /** 全部折叠前回调 默认 `null` */
  'on-collapsedall'?: string;
  /** 选中回调 默认 `null` */
  'on-selected'?: string;
  /** 取消选中回调 默认 `null` */
  'on-deselected'?: string;
  /** 全选回调 默认 `null` */
  'on-selectedall'?: string;
  /** 勾选回调 默认 `null` */
  'on-checked'?: string;
  /** 取消勾选回调 默认 `null` */
  'on-unchecked'?: string;
  /** 全部勾选回调 默认 `null` */
  'on-checkedall'?: string;
  /** 填充完成回调 默认 `null` */
  'on-filled'?: string;
  /** 翻页回调 默认 `null` */
  'on-turned'?: string;
  /** 默认 `null` */
  'on-got'?: string;
  /** 设置值回调 默认 `null` */
  'on-set'?: string;
  /** 清除值回调 默认 `null` */
  'on-cleared'?: string;
  /** 转移节点回调 默认 `null` */
  'on-grafted'?: string;
  /** 内容更新回调 默认 `null` */
  'on-updatedcont'?: string;
  /** 请求完成回调 默认 `null` */
  'on-request'?: string;
  /** 选择过少回调 默认 `null` */
  'on-toofew'?: string;
  /** 选择过多回调 默认 `null` */
  'on-toomany'?: string;
  /** 默认 `null` */
  'on-locked'?: string;
  /** 默认 `null` */
  'on-unlocked'?: string;
  /** 输出值时监听 默认 `null` */
  'on-output'?: string;
  'ins-name'?: string;
  'stor-name'?: string;
  /** 默认 `[]` */
  'stor-keys'?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 语言包（JSON） 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  'b4-init'?: string;
  /** 默认 `null` */
  'on-constructed'?: string;
  /** 默认 `null` */
  'on-initiate'?: string;
  /** 默认 `null` */
  'on-initiated'?: string;
  /** 默认 `null` */
  'on-destroyed'?: string;
  /** 默认 `null` */
  'on-error'?: string;
  /** 默认 `null` */
  'on-reset'?: string;
  /** 默认 `null` */
  'on-updated'?: string;
  /** 默认 `null` */
  'on-saved'?: string;
  /** 默认 `null` */
  'on-clearedcache'?: string;
  /** 默认 `null` */
  'on-updatedcache'?: string;
}

/** <ax-tree> 元素实例（querySelector('ax-tree') 的类型） */
export interface AxTreeElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxTreeAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxTreeAttributes>;
}

/* ========================================================================== */
/* ax-twilight · 白天黑夜切换组件                                                     */
/* 文档: components/261.html                                                    */
/* ========================================================================== */
/** <ax-twilight> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxTwilightAttributes {
  /** 指定目标节点 */
  target?: string;
  /** 组合风格 */
  feature?: string;
  /** 尺寸大小 */
  size?: string;
  /** 标签文本，用英文逗号隔开 */
  labels?: string;
  /** 布尔属性，书写即生效 */
  ''?: boolean;
}

/** <ax-twilight> 元素实例（querySelector('ax-twilight') 的类型） */
export interface AxTwilightElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxTwilightAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxTwilightAttributes>;
}

/* ========================================================================== */
/* ax-upload · 文件上传组件                                                         */
/* 文档: components/260.html                                                    */
/* ========================================================================== */
/** <ax-upload> 的 HTML 属性（也用于 set() / propsProxy） */
export interface AxUploadAttributes {
  /** 表单字段 name 属性（提交时的键名） */
  name?: string;
  /** 字段值 */
  value?: string;
  url?: string;
  content?: string;
  'cont-type'?: string;
  'cont-data'?: string;
  /** 异步请求参数（JSON） */
  ajax?: string;
  /** 字符数量限制 */
  limit?: string;
  accept?: string;
  /** 字段/组件类型 */
  type?: string;
  feature?: string;
  /** 尺寸：xxs/xs/sm/md/lg/xl/xxl */
  size?: string;
  table?: string;
  /** 附加 CSS 类名 */
  classes?: string;
  /** 状态标记 */
  status?: string;
  'choose-btn'?: string;
  'upload-btn'?: string;
  'clear-btn'?: string;
  cloud?: string;
  /** 透传给内部 input 的属性（JSON） */
  attrs?: string;
  /** 布尔属性，书写即生效 */
  async?: boolean;
  /** 是否禁用 */
  disabled?: boolean;
  /** 是否只读 */
  readonly?: boolean;
  /** 布尔属性，书写即生效 */
  multiple?: boolean;
  /** 布尔属性，书写即生效 */
  manual?: boolean;
  /** 布尔属性，书写即生效 */
  pastable?: boolean;
}

/** <ax-upload> 元素实例（querySelector('ax-upload') 的类型） */
export interface AxUploadElem extends AxCompBase {
  /** 响应式属性代理 */
  readonly propsProxy: AxUploadAttributes;
  /** 初始属性快照 */
  readonly propsRaw: Readonly<AxUploadAttributes>;
}


/* ============================================================================
 * 2. 模块公共基类（所有 new XxxModule(...) 实例共享的变量与方法）
 *    文档：start/104.html「模块」
 * ========================================================================== */

export interface AxModBase {
  /* ---- 公共变量 ---- */
  /** 用户参数（构造时传入的 options，已与默认参数合并） */
  options: Record<string, any>;
  /** 初始化次数 */
  readonly initCount: number;
  /** 目标数据（宿主为 data 类型时有值） */
  readonly targetData: any[];
  /** 目标节点（宿主为 el 类型时有值） */
  readonly targetEl: HTMLElement;
  /** 缓存键值对 */
  readonly storTmp: Record<string, any>;
  /** storTmp 的 proxy 对象 */
  readonly storObs: Record<string, any>;
  /** 目标节点的原始 HTML 文本 */
  readonly rawHtml: string;
  /** 目标节点的原始子节点 */
  readonly rawEl: HTMLElement;
  /** 从参数 map 中取得的原始参数 */
  readonly rawOpts: Record<string, any>;
  /** 基于 rawOpts 合并用户参数的默认参数 */
  readonly dftOpts: Record<string, any>;
  /** 监听事件集合 */
  readonly plans: Record<string, any>;
  /** 销毁状态 */
  readonly destroyed: boolean;
  /** 创建实例时的时间戳 */
  readonly timestamp: number;
  /** 原始数据 */
  readonly respSource: any;
  /** 是否已经初始化 */
  readonly initialized: boolean;
  /** 主节点渲染次数 */
  readonly renderCount: number;
  /** 模块名称（小写） */
  readonly moduleName: string;
  /** 模块的模板字符串 */
  readonly tplStr: string;
  /** 模块的模板引擎 */
  readonly tplEng: AxFn;

  /* ---- 公共方法 ---- */
  /** 初始化实例，返回 Promise<this> */
  init(cb?: () => void): Promise<this>;
  /** 销毁实例 */
  destroy(cb?: () => void): this;
  /** 实例重置 */
  reset(cb?: () => void): this;
  /** 更新参数（可传部分参数，将与当前参数合并） */
  update(settings: Record<string, any>, cb?: () => void): Promise<this>;
  /** 清除缓存 */
  clearCache(value?: any): this;
  /** 更新缓存（一般不使用） */
  updateCache(value: Record<string, any>): this;
  /** 保存缓存（一般不使用） */
  save(cb?: () => void): this;
  /** 禁用模块 */
  lock(cb?: () => void): this;
  /** 解禁模块 */
  unlock(cb?: () => void): this;
  /** 事件监听（事件名见各模块 EventName 类型） */
  on(type: string, handler: (...args: any[]) => void): this;
  off(type: string, handler?: (...args: any[]) => void): this;
  emit(type: string, ...params: any[]): this;
}


/* ############################### 二、模块（JS Classes） ############################### */

/* ========================================================================== */
/* Accordion · 折叠面板模块                                                         */
/* 文档: modules/345.html                                                       */
/* ========================================================================== */
/** Accordion 构造参数 —— new Accordion(elem, options) 的 options */
export interface AxAccordionOptions {
  /* ---- 基础配置 ---- */
  /** 隐藏字段的name属性 */
  name?: string;
  /** 数据源 */
  content?: any;
  /** 内容类型 默认 `'text'` */
  contType?: string;
  /** 请求content时发送的数据 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求参数 默认 `{}` */
  ajax?: Record<string, any>;
  /** 是否完全禁用 默认 `false` */
  passive?: boolean;
  /** 初始禁用项 */
  disable?: string | number | HTMLElement | Record<string, any> | any[];
  /** 初始只读项 */
  readonly?: string | number | HTMLElement | Record<string, any> | any[];
  /** 是否延迟渲染子节点 默认 `false` */
  deferred?: boolean;
  /** 事件防抖延迟(ms) 默认 `100` */
  delay?: number;
  /* ---- 展开/折叠配置 ---- */
  /** 点击分支行行为 默认 `'none'` */
  clickLine?: 'checked' | 'expanded' | 'none';
  /* ---- 交互配置 ---- */
  /** 配置指示箭头 默认 `{"enable": true, "show": "icon-down", "hide": "icon-down", "anim": "rotate180", "type": "icon", "trigger": "click"}` */
  arrow?: boolean | AxAccordionArrowOptions;
  /** 配置复选/单选 默认 `{"enable": false, "type": "checkbox", "value": ""}` */
  check?: boolean | AxAccordionCheckOptions;
  /* ---- 工具配置 ---- */
  /** 配置编辑工具 默认 `{"enable": false, "trigger": "hover", "children": ["add", "edit", "remove"]}` */
  tools?: boolean | AxAccordionToolsOptions;
  /* ---- 外观配置 ---- */
  /** 配置指示灯 默认 `{"enable": false, "placement": "bottom"}` */
  lamp?: boolean | AxAccordionLampOptions;
  /** 动画持续时间(ms) 默认 `0` */
  duration?: number;
  /** 附加CSS类名 */
  classes?: string;
  /** 头部节点布局 默认 `'check(icon|disk|cube|label|tips)custom|tools|badge|arrow'` */
  layout?: string;
  /* ---- 数据输出配置 ---- */
  /** 配置输出 默认 `{"enable": true, "target": "", "prop": "label"}` */
  output?: boolean | AxAccordionOutputOptions;
  /* ---- 拖拽配置 ---- */
  /** 配置拖拽 默认 `{"enable": false, "exclude": ""}` */
  drag?: boolean | AxAccordionDragOptions;
  /* ---- 前置函数 ---- */
  /** 展开前执行 默认 `null` */
  b4Expand?: (...args: any[]) => any;
  /** 折叠前执行 默认 `null` */
  b4Collapse?: (...args: any[]) => any;
  /** 新增前执行 默认 `null` */
  b4Add?: (...args: any[]) => any;
  /** 编辑前执行 默认 `null` */
  b4Edit?: (...args: any[]) => any;
  /** 删除前执行 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /** 拖拽前执行 默认 `null` */
  b4Graft?: (...args: any[]) => any;
  /* ---- 事件监听 ---- */
  /** 节点树创建完成后触发 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 节点触发后触发 默认 `null` */
  onTrigger?: (...args: any[]) => any;
  /** 新增后触发 默认 `null` */
  onAdded?: (...args: any[]) => any;
  /** 编辑后触发 默认 `null` */
  onEdited?: (...args: any[]) => any;
  /** 删除后触发 默认 `null` */
  onRemoved?: (...args: any[]) => any;
  /** 禁用后触发 默认 `null` */
  onDisabled?: (...args: any[]) => any;
  /** 启用后触发 默认 `null` */
  onEnabled?: (...args: any[]) => any;
  /** 全部禁用后触发 默认 `null` */
  onDisabledAll?: (...args: any[]) => any;
  /** 全部启用后触发 默认 `null` */
  onEnabledAll?: (...args: any[]) => any;
  /** 设置只读后触发 默认 `null` */
  onReadonly?: (...args: any[]) => any;
  /** 全部只读后触发 默认 `null` */
  onReadonlyAll?: (...args: any[]) => any;
  /** 展开时触发 默认 `null` */
  onExpand?: (...args: any[]) => any;
  /** 展开后触发 默认 `null` */
  onExpanded?: (...args: any[]) => any;
  /** 折叠时触发 默认 `null` */
  onCollapse?: (...args: any[]) => any;
  /** 折叠后触发 默认 `null` */
  onCollapsed?: (...args: any[]) => any;
  /** 全部展开时触发 默认 `null` */
  onExpandAll?: (...args: any[]) => any;
  /** 全部展开后触发 默认 `null` */
  onExpandedAll?: (...args: any[]) => any;
  /** 全部折叠后触发 默认 `null` */
  onCollapsedAll?: (...args: any[]) => any;
  /** 选中后触发 默认 `null` */
  onSelected?: (...args: any[]) => any;
  /** 取消选中后触发 默认 `null` */
  onDeselected?: (...args: any[]) => any;
  /** 全部选中后触发 默认 `null` */
  onSelectedAll?: (...args: any[]) => any;
  /** 勾选后触发 默认 `null` */
  onChecked?: (...args: any[]) => any;
  /** 取消勾选后触发 默认 `null` */
  onUnchecked?: (...args: any[]) => any;
  /** 全部勾选后触发 默认 `null` */
  onCheckedAll?: (...args: any[]) => any;
  /** 填充brief后触发 默认 `null` */
  onFilled?: (...args: any[]) => any;
  /** 翻页后触发 默认 `null` */
  onTurned?: (...args: any[]) => any;
  /** 取值后触发 默认 `null` */
  onGot?: (...args: any[]) => any;
  /** 设置值后触发 默认 `null` */
  onSet?: (...args: any[]) => any;
  /** 清除值后触发 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 拖拽完成后触发 默认 `null` */
  onGrafted?: (...args: any[]) => any;
  /** 内容更新后触发 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /** 异步请求完成后触发 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /** 选中数量过少时触发 默认 `null` */
  onTooFew?: (...args: any[]) => any;
  /** 选中数量过多时触发 默认 `null` */
  onTooMany?: (...args: any[]) => any;
  /** 全局禁用后触发 默认 `null` */
  onPassivated?: (...args: any[]) => any;
  /** 取消全局禁用后触发 默认 `null` */
  onActivated?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"value": "", "all": false, "only": true, "linkage": false}` */
  expand?: AxAccordionExpandOptions;
  /** 默认 `{"attr": "dropnode", "global": false}` */
  drop?: AxAccordionDropOptions;
  /** 默认 `{"rounded": true, "bordered": true, "divided": true, "gapped": false}` */
  appear?: AxAccordionAppearOptions;
  /** 默认 `null` */
  onLocked?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUnlocked?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Accordion 的 arrow 配置组 */
export interface AxAccordionArrowOptions {
  /* ---- 交互配置 ---- */
  /** 是否启用指示箭头 默认 `true` */
  enable?: boolean;
  /** 展开状态的类名 默认 `'_icon-down'` */
  show?: string;
  /** 折叠状态的类名 默认 `'_icon-down'` */
  hide?: string;
  /** 动画类名 默认 `'_rotate180'` */
  anim?: string;
  /** 箭头类型 默认 `'icon'` */
  type?: 'icon' | 'image';
  /** 触发展开的事件 默认 `'click'` */
  trigger?: 'click' | 'hover';
}

/** Accordion 的 check 配置组 */
export interface AxAccordionCheckOptions {
  /* ---- 交互配置 ---- */
  /** 是否启用复选/单选 默认 `false` */
  enable?: boolean;
  /** 选择模式 默认 `'checkbox'` */
  type?: 'checkbox' | 'radio' | 'none';
  /** 初始选中项 */
  value?: string | number | HTMLElement | Record<string, any> | any[];
}

/** Accordion 的 tools 配置组 */
export interface AxAccordionToolsOptions {
  /* ---- 工具配置 ---- */
  /** 是否显示编辑工具 默认 `false` */
  enable?: boolean;
  /** 工具显示触发方式 默认 `'hover'` */
  trigger?: 'click' | 'hover' | 'none';
  /** 工具按钮类型 默认 `['add', 'edit', 'remove']` */
  children?: any[];
}

/** Accordion 的 lamp 配置组 */
export interface AxAccordionLampOptions {
  /* ---- 外观配置 ---- */
  /** 是否启用标签指示灯 默认 `false` */
  enable?: boolean;
  /** 指示灯位置 默认 `'bottom'` */
  placement?: 'left' | 'right' | 'top' | 'bottom';
}

/** Accordion 的 output 配置组 */
export interface AxAccordionOutputOptions {
  /* ---- 数据输出配置 ---- */
  /** 是否启用数据输出 默认 `true` */
  enable?: boolean;
  /** 数据输出目标节点 */
  target?: string;
  /** 多值分隔符 默认 `','` */
  separator?: string;
  /** 取值属性 默认 `'label'` */
  prop?: string;
}

/** Accordion 的 drag 配置组 */
export interface AxAccordionDragOptions {
  /* ---- 拖拽配置 ---- */
  /** 是否启用拖拽 默认 `false` */
  enable?: boolean;
  /** 不可拖拽节点选择器 */
  exclude?: string;
}

/** Accordion 的 expand 配置组 */
export interface AxAccordionExpandOptions {
  /* ---- 展开/折叠配置 ---- */
  /** 初始展开项 */
  value?: string | number | HTMLElement | Record<string, any> | any[];
  /** 是否展开全部 默认 `false` */
  all?: boolean;
  /** 是否仅展开一项 默认 `true` */
  only?: boolean;
  /** 是否联动父层展开 默认 `false` */
  linkage?: boolean;
}

/** Accordion 的 drop 配置组 */
export interface AxAccordionDropOptions {
  /* ---- 拖拽配置 ---- */
  /** 可释放节点属性 默认 `'dropnode'` */
  attr?: string;
  /** 是否全局释放 默认 `false` */
  global?: boolean;
}

/** Accordion 的 appear 配置组 */
export interface AxAccordionAppearOptions {
  /* ---- 外观配置 ---- */
  /** 是否显示圆角 默认 `true` */
  rounded?: boolean;
  /** 是否显示边框 默认 `true` */
  bordered?: boolean;
  /** 是否显示分隔线 默认 `true` */
  divided?: boolean;
  /** 是否显示项目间隔 默认 `false` */
  gapped?: boolean;
}

/** Accordion 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxAccordionEventName = 'added' | 'checkedAll' | 'collapse' | 'collapsed' | 'constructed' | 'edited' | 'expand' | 'expanded' | 'filled' | 'got' | 'grafted' | 'initiate' | 'initiated' | 'removed' | 'set' | 'trigger';

/** Accordion 模块实例（new Accordion() 的返回值） */
export interface AxAccordionInstance extends AxModBase {
  options: AxAccordionOptions;
  /** 事件监听（事件名见 AxAccordionEventName；回调参数视事件而定） */
  on(type: AxAccordionEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  getDataToRender(...args: any[]): any;
  getTreeFlat(...args: any[]): any;
  getObserver(...args: any[]): any;
  initExpandeds(...args: any[]): any;
  renderData(...args: any[]): any;
  updateArrowEl(...args: any[]): any;
  createHeadEl(...args: any[]): any;
  createBodyEl(...args: any[]): any;
  setAttrs(...args: any[]): any;
  updateExpandEvt(...args: any[]): any;
  updateEvt(...args: any[]): any;
  renderFinish(...args: any[]): any;
  getTriggerEl(...args: any[]): any;
  addTrigger(...args: any[]): any;
  removeTrigger(...args: any[]): any;
  toggleExpanded(...args: any[]): any;
  setContExtra(...args: any[]): any;
  eachCollapse(...args: any[]): any;
  eachExpand(...args: any[]): any;
  uncheckSibings(...args: any[]): any;
  check(...args: any[]): any;
  checkAll(...args: any[]): any;
  add(...args: any[]): any;
  remove(...args: any[]): any;
  inputLabelCont(...args: any[]): any;
  edit(...args: any[]): any;
  graft(...args: any[]): any;
  updateVals(...args: any[]): any;
  setVals(...args: any[]): any;
  clearVals(...args: any[]): any;
  getVals(...args: any[]): any;
  setContent(...args: any[]): any;
  updateCont(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Autocomplete · 自动补全模块                                                      */
/* 文档: modules/336.html                                                       */
/* ========================================================================== */
/** Autocomplete 构造参数 —— new Autocomplete(elem, options) 的 options */
export interface AxAutocompleteOptions {
  /* ---- 基础参数 ---- */
  /** 填充值到编辑器的属性，通常为value或label 默认 `'value'` */
  field?: string;
  /* ---- 数据源参数 ---- */
  /** 数据源 */
  content?: any;
  /** 数据源类型 默认 `'text'` */
  contType?: string;
  /** 发起请求时的数据 默认 `{}` */
  contData?: Record<string, any>;
  /** ajax请求参数 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 模板参数 ---- */
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎 默认 `null` */
  tplEng?: (...args: any[]) => any;
  /* ---- 弹窗参数 ---- */
  /** 弹窗模块参数配置 默认 `{}` */
  popup?: Record<string, any>;
  /** 检索模块参数配置 默认 `{}` */
  retrieval?: Record<string, any>;
  /* ---- 前置函数 ---- */
  /** 填充值之前的回调函数，支持一个预填项参数 默认 `null` */
  b4Fill?: (...args: any[]) => any;
  /* ---- 监听事件 ---- */
  /** 窗口显示后的回调函数 默认 `null` */
  onShown?: (...args: any[]) => any;
  /** 窗口关闭后的回调函数 默认 `null` */
  onHidden?: (...args: any[]) => any;
  /** 清除列表后的回调函数 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 重置并强制打开后的回调函数 默认 `null` */
  onReset2Show?: (...args: any[]) => any;
  /** 内容更新后的回调函数，支持一个content参数 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Autocomplete 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxAutocompleteEventName = 'constructed' | 'hidden' | 'initiate' | 'initiated' | 'shown';

/** Autocomplete 模块实例（new Autocomplete() 的返回值） */
export interface AxAutocompleteInstance extends AxModBase {
  options: AxAutocompleteOptions;
  /** 事件监听（事件名见 AxAutocompleteEventName；回调参数视事件而定） */
  on(type: AxAutocompleteEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setVals(...args: any[]): any;
  setEmpty(...args: any[]): any;
  setAttrs(...args: any[]): any;
  createPopup(...args: any[]): any;
  createRetrieval(...args: any[]): any;
  reset2Show(...args: any[]): any;
  clear(...args: any[]): any;
  show(...args: any[]): any;
  hide(...args: any[]): any;
  updateCont(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Color · 颜色拾取器 (Library)                                                    */
/* 文档: modules/435.html                                                       */
/* ========================================================================== */
/** Color 构造参数 —— new Color(elem, options) 的 options */
export interface AxColorOptions {
  /* ---- 核心参数 ---- */
  /** 设置面板初始色值，可识别多种颜色格式，并支持透明度通道，例如#ff0000、rgba(0,0,0,0.5), hsl(0,0%,0%)、hwb(0,0%,0%,0.5)等 */
  value?: string;
  /** 布局结构，可选填full（两栏结构，包含picker和palette），picker（仅显示拾色器），palette（仅显示预设值） 默认 `'picker'` */
  layout?: string;
  /** 给目拾色器容器添加class类，一个多个均可，写法同ax.classes.add */
  classes?: string;
  /** 窗口气泡示例的参数，同模块Popup的参数 默认 `{}` */
  popup?: Record<string, any>;
  /* ---- 画布相参数 ---- */
  /** 画布配置，包括尺寸、色彩空间模式和初始色相值 默认 `{"showLabel": true, "showTools": true, "width": 370, "height": 180, "mode": "hsv", "hue": 0}` */
  canvas?: AxColorCanvasOptions;
  /* ---- 输出相关属性 ---- */
  /** 输出配置，包括透明度通道、显示模式和格式 默认 `{"alpha": true, "mode": "hsl", "format": "hsla", "child": "", "manual": true}` */
  output?: AxColorOutputOptions;
  /* ---- 等待函数 ---- */
  /** 等待函数，返回Promise，设置值前执行，支持一个参数即被设置的值 默认 `null` */
  b4Assign?: (...args: any[]) => any;
  /** 等待函数，返回Promise，清除值前执行，无参数 默认 `null` */
  b4Clear?: (...args: any[]) => any;
  /** 等待函数，返回Promise，填充值前执行，支持一个参数被填充的值 默认 `null` */
  b4Fill?: (...args: any[]) => any;
  /** 等待函数，返回Promise，恢复初始值前执行，无参数 默认 `null` */
  b4Restore?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 监听函数，设置值后执行，支持一个参数即设置值 默认 `null` */
  onAssigned?: (...args: any[]) => any;
  /** 监听函数，清空值后执行，无参数 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 监听函数，恢复初始值后执行，无参数 默认 `null` */
  onRestored?: (...args: any[]) => any;
  /** 监听函数，填充值后执行，支持一个参数即被填充的文本值 默认 `null` */
  onFilled?: (...args: any[]) => any;
  /** 监听函数，生成面板色值后执行，支持一个参数即生成的色值 默认 `null` */
  onOutput?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Color 的 canvas 配置组 */
export interface AxColorCanvasOptions {
  /* ---- 画布相参数 ---- */
  /** 是否显示色彩空间的横纵坐标的label名称 默认 `true` */
  showLabel?: boolean;
  /** 是否显示画布的几个工具 默认 `true` */
  showTools?: boolean;
  /** 画布的宽度，支持3种格式，可填400,400px或40rem，由于手机端宽度有限，该尺寸在手机端竖屏将转为300px显示 默认 `370` */
  width?: number | string;
  /** 画布的高度，格式同width，由于手机端宽度有限，该尺寸在手机端竖屏将转为150px显示 默认 `180` */
  height?: number | string;
  /** 画布色彩空间模式，默认hsv，可选择hsl 默认 `"hsv"` */
  mode?: string;
  /** 色彩空间初始色相值，范围0-360，360与0的效果一样，默认为0即纯红 默认 `0` */
  hue?: number;
}

/** Color 的 output 配置组 */
export interface AxColorOutputOptions {
  /* ---- 输出相关属性 ---- */
  /** 是否显示透明度通道 默认 `true` */
  alpha?: boolean;
  /** 默认面板显示哪个类型的值，可选择rgb、hwb、hsl或hex 默认 `"hsl"` */
  mode?: string;
  /** 默认输出值的格式，可选择rgb、rgba、hwb、hwba、hsl、hsla、hex、hexa 默认 `"hsla"` */
  format?: string;
  /** 填值时的宿主子节点选择器，为空则向宿主填值 默认 `""` */
  child?: string;
  /** 是否手动填充值，false将不显示"确定"按钮并自动填值，true则通过"确定"按钮填充值 默认 `true` */
  manual?: boolean;
}

/** Color 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxColorEventName = 'assigned' | 'constructed' | 'filled' | 'initiate' | 'initiated' | 'output' | 'restored';

/** Color 模块实例（new Color() 的返回值） */
export interface AxColorInstance extends AxModBase {
  options: AxColorOptions;
  /** 事件监听（事件名见 AxColorEventName；回调参数视事件而定） */
  on(type: AxColorEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  updateReverse(...args: any[]): any;
  setAttrs(...args: any[]): any;
  toggleOutputAct(...args: any[]): any;
  updateHueSlider(...args: any[]): any;
  updateAlphaSlider(...args: any[]): any;
  drawCanvas(...args: any[]): any;
  getColorFromCanvas(...args: any[]): any;
  canvasInteraction(...args: any[]): any;
  updatePreview(...args: any[]): any;
  updatePreviewOutput(...args: any[]): any;
  setOutputStr(...args: any[]): any;
  setOutputVals(...args: any[]): any;
  setPopup(...args: any[]): any;
  createPanel(...args: any[]): any;
  getThemePresets(...args: any[]): any;
  getHistoryPresets(...args: any[]): any;
  addHistory(...args: any[]): any;
  storColor(...args: any[]): any;
  getRegularPresets(...args: any[]): any;
  createColorGrid(...args: any[]): any;
  setAxisLabel(...args: any[]): any;
  getIndCoord(...args: any[]): any;
  updateIndCoord(...args: any[]): any;
  getCanvasPos(...args: any[]): any;
  setValFromCanvas(...args: any[]): any;
  setGesture(...args: any[]): any;
  parseVals(...args: any[]): any;
  getCoordFromVal(...args: any[]): any;
  assignVals(...args: any[]): any;
  initVals(...args: any[]): any;
  getVals(...args: any[]): any;
  fillVals(...args: any[]): any;
  clearVals(...args: any[]): any;
  restoreVals(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Datetime · 日期选择模块                                                          */
/* 文档: modules/328.html                                                       */
/* ========================================================================== */
/** Datetime 构造参数 —— new Datetime(elem, options) 的 options */
export interface AxDatetimeOptions {
  /* ---- 基础配置参数 ---- */
  /** 目标节点的name属性 */
  name?: string;
  /** 日期值，支持多种格式如：2022-2-3、2022/2/3或Date对象 */
  value?: string | Date;
  /** 是否禁用 默认 `false` */
  disabled?: boolean;
  /** 目标节点尺寸 */
  size?: string;
  /** 日期格式，如：YYYY-MM-DD、YYYY/MM/DD hh:mm:ss等 */
  format?: string;
  /** 日期窗口的自定义样式类 */
  classes?: string;
  /* ---- 显示模式参数 ---- */
  /** 选择模式：单选/多选/区间 默认 `'single'` */
  mode?: 'single' | 'multiple' | 'range';
  /** 特殊功能组合类型 默认 `'single'` */
  feature?: 'single' | 'static';
  /** 显示方式 默认 `'popup'` */
  display?: 'popup' | 'inline' | 'dialog' | 'drawer';
  /** 手机端是否转为抽屉式显示 默认 `true` */
  toDrawer?: boolean;
  /** 是否100%铺满容器 默认 `false` */
  full?: boolean;
  /* ---- 输入框配置 ---- */
  /** 是否启用多行显示 默认 `false` */
  multiline?: boolean;
  /** 输入框占位文本 默认 `null` */
  placeholder?: string;
  /** 输入框标签文本 */
  label?: string;
  /** 工具箱配置 默认 `{"enable": false, "children": ["close"]}` */
  tools?: Record<string, any>;
  /* ---- 日期限制参数 ---- */
  /** 最大选择数量(多选模式) 默认 `100` */
  maxSelection?: number;
  /** 年月日有效时间跨度，可多段，包含min和max属性 默认 `[]` */
  datespan?: any[];
  /** 时间有效跨度，可多段，包含min和max属性 默认 `[]` */
  timespan?: any[];
  /* ---- 布局参数 ---- */
  /** 面板网格行数 默认 `1` */
  rows?: number;
  /** 面板网格列数 默认 `1` */
  cols?: number;
  /** 周起始日 默认 `'monday'` */
  weekStart?: 'monday' | 'sunday';
  /** 多个日期的分隔符 默认 `','` */
  separator?: string;
  /* ---- 按钮控制参数 ---- */
  /** 点击"现在"按钮是否关闭窗口 默认 `false` */
  nowHide?: boolean;
  /** 是否显示"现在"按钮 默认 `false` */
  nowShow?: boolean;
  /** 是否显示"清除"按钮 默认 `true` */
  clearShow?: boolean;
  /** 是否显示"关闭"按钮 默认 `true` */
  closeShow?: boolean;
  /** 是否显示"取消"按钮 默认 `false` */
  cancelShow?: boolean;
  /** 点击确认按钮是否关闭窗口 默认 `true` */
  confirmHide?: boolean;
  /* ---- 自动化行为参数 ---- */
  /** 是否自动填充选中值 默认 `false` */
  autoFill?: boolean;
  /** 是否手动填值 默认 `false` */
  manual?: boolean;
  /** 是否自动校正当前值 默认 `true` */
  autoCorrect?: boolean;
  /** 是否强制选择至少一个日期 默认 `false` */
  required?: boolean;
  /** 无值时是否自动填入今天 默认 `false` */
  fillNow?: boolean;
  /* ---- 选择器参数 ---- */
  /** 触发按钮选择器 */
  btnSel?: string;
  /** 定位节点选择器 */
  posSel?: string;
  /** 输入输出节点选择器 */
  inputSel?: string;
  /** 子节点选择器 */
  childSel?: string;
  /* ---- 扩展功能参数 ---- */
  /** 农历日期配置 默认 `{"enable": false, "handler": "", "map": {}, "tplStr": "<div>农历：{{this.animal}}年 {{this.month}}{{this.date}}</div>\n                    <div>干支：{{this.gzYear}}年 {{this.gzMonth}}月 {{this.gzDate}}日</div>\n                    <div>节日/节气：{{this.term}} {{this.lunarFestival}} {{this.solarFestival}}</div>\n                    ", "tplEng": null, "arrange": ["solarFestival", "lunarFestival", "term", "date"], "tags": []}` */
  lunar?: Record<string, any>;
  /** 事件指示功能配置 默认 `{"enable": false, "display": "popup", "tplStr": "<div =\"label\">{{this.label}}</div><div =\"content\">{{this.content}}</div>", "tplEng": null, "heading": "YYYY年MM月DD日 WW", "list": []}` */
  events?: Record<string, any>;
  /** 左侧工具面板配置 默认 `[]` */
  menu?: any[];
  /** 弹窗参数配置 默认 `{}` */
  bubble?: Record<string, any>;
  /** 底部按钮配置 默认 `{"enable": true, "layout": "plain", "children": ["clear", "confirm"]}` */
  footer?: Record<string, any>;
  /* ---- 前置函数 ---- */
  /** 填充值前回调 默认 `null` */
  b4fill?: (...args: any[]) => any;
  /** 移除选中项前回调 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /** 设置选中项前回调 默认 `null` */
  b4Set?: (...args: any[]) => any;
  /** 清空选中项前回调 默认 `null` */
  b4Clear?: (...args: any[]) => any;
  /** 取消前回调 默认 `null` */
  b4Cancel?: (...args: any[]) => any;
  /** 确认前回调 默认 `null` */
  b4Confirm?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 填充值后回调 默认 `null` */
  onFilled?: (...args: any[]) => any;
  /** 移除选中项后回调 默认 `null` */
  onRemoved?: (...args: any[]) => any;
  /** 设置选中项后回调 默认 `null` */
  onSet?: (...args: any[]) => any;
  /** 设置日期区间后回调 默认 `null` */
  onSetRange?: (...args: any[]) => any;
  /** 创建新面板后回调 默认 `null` */
  onCreatedPanels?: (...args: any[]) => any;
  /** 取消后回调 默认 `null` */
  onCanceled?: (...args: any[]) => any;
  /** 确认后回调 默认 `null` */
  onConfirmed?: (...args: any[]) => any;
  /** 清除后回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 显示后回调 默认 `null` */
  onOpened?: (...args: any[]) => any;
  /** 关闭后回调 默认 `null` */
  onClosed?: (...args: any[]) => any;
  /** 转移节点后回调 默认 `null` */
  onTransferred?: (...args: any[]) => any;
  /** 更新时间后回调 默认 `null` */
  onUpdatedTime?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  minDate?: string;
  maxDate?: string;
  /** 默认 `false` */
  resultShow?: boolean;
  next?: string;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Datetime 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxDatetimeEventName = 'constructed' | 'createPanels' | 'filled' | 'hidden' | 'initiate' | 'initiated' | 'output' | 'ranged' | 'removed' | 'set' | 'shown' | 'updatedTime';

/** Datetime 模块实例（new Datetime() 的返回值） */
export interface AxDatetimeInstance extends AxModBase {
  options: AxDatetimeOptions;
  /** 事件监听（事件名见 AxDatetimeEventName；回调参数视事件而定） */
  on(type: AxDatetimeEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setFeature(...args: any[]): any;
  getTargets(...args: any[]): any;
  setAttrs(...args: any[]): any;
  createPanel(...args: any[]): any;
  createPanels(...args: any[]): any;
  inSelectedDate(...args: any[]): any;
  showEvent(...args: any[]): any;
  inCurPanels(...args: any[]): any;
  inputVal2Dates(...args: any[]): any;
  str2Dates(...args: any[]): any;
  getValFromTgt(...args: any[]): any;
  setValFromTgt(...args: any[]): any;
  vals2Arr(...args: any[]): any;
  handleDayCont(...args: any[]): any;
  stdTimeStr(...args: any[]): any;
  handleTimeCont(...args: any[]): any;
  dates2Objs(...args: any[]): any;
  checkB4Show(...args: any[]): any;
  createSkeleton(...args: any[]): any;
  createMenu(...args: any[]): any;
  createMonthPanel(...args: any[]): any;
  createYearPanel(...args: any[]): any;
  createYearList(...args: any[]): any;
  goto(...args: any[]): any;
  offset(...args: any[]): any;
  selectedAttrToggle(...args: any[]): any;
  select(...args: any[]): any;
  deselect(...args: any[]): any;
  setRangeAttr(...args: any[]): any;
  removeRangeAttr(...args: any[]): any;
  getDatespan(...args: any[]): any;
  getTimespan(...args: any[]): any;
  getDateTimeSpan(...args: any[]): any;
  setWrapDisabled(...args: any[]): any;
  compareVal(...args: any[]): any;
  setDatespanDisabled(...args: any[]): any;
  setTimespanDisabled(...args: any[]): any;
  getDateVal(...args: any[]): any;
  getCurGrids(...args: any[]): any;
  findFromPanels(...args: any[]): any;
  afterCreatePanels(...args: any[]): any;
  renderFinish(...args: any[]): any;
  setResultWidth(...args: any[]): any;
  openTime2Scroll(...args: any[]): any;
  searchYear2Update(...args: any[]): any;
  createPanelDayList(...args: any[]): any;
  createPanelMonthList(...args: any[]): any;
  createPanelYearList(...args: any[]): any;
  createLoopStr(...args: any[]): any;
  createTimeList(...args: any[]): any;
  toggleTimeList(...args: any[]): any;
  getTimeListArr(...args: any[]): any;
  showForemost(...args: any[]): any;
  createTimeSelectPanel(...args: any[]): any;
  timeToUpdate(...args: any[]): any;
  setTimeBtnHTML(...args: any[]): any;
  setDaytime(...args: any[]): any;
  assignDaytime(...args: any[]): any;
  cancel(...args: any[]): any;
  hide(...args: any[]): any;
  show(...args: any[]): any;
  confirm(...args: any[]): any;
  fillVals(...args: any[]): any;
  handleVal(...args: any[]): any;
  setNow(...args: any[]): any;
  removeIns(...args: any[]): any;
  update(...args: any[]): any;
  setVals2Objs(...args: any[]): any;
  getVals(...args: any[]): any;
  setVals(...args: any[]): any;
  setRange(...args: any[]): any;
  removeVals(...args: any[]): any;
  clear(...args: any[]): any;
  transfer(...args: any[]): any;
  disable(...args: any[]): any;
  enable(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Dialog · 对话框模块                                                             */
/* 文档: modules/325.html                                                       */
/* ========================================================================== */
/** Dialog 构造参数 —— new Dialog(elem, options) 的 options */
export interface AxDialogOptions {
  /* ---- 基础配置参数 ---- */
  /** 窗口标题 */
  heading?: string;
  /** 是否显示标题分割线 默认 `false` */
  divider?: boolean;
  /** 动画时长(ms) 默认 `0` */
  duration?: number;
  /** 是否自动计算动画时间 默认 `false` */
  autoDur?: boolean;
  /** 是否禁用show方法 默认 `false` */
  asleep?: boolean;
  /** 打开后是否禁止关闭 默认 `false` */
  deadShow?: boolean;
  /** 是否只允许显示不允许关闭 默认 `false` */
  keepShow?: boolean;
  /** 窗口样式类 */
  classes?: string;
  /** 窗口层级 默认 `0` */
  zIndex?: number;
  /** 是否阻止与body联动滚动 默认 `false` */
  dedicated?: boolean;
  /* ---- Dialog专属配置 ---- */
  /** 窗口显示位置 默认 `'center'` */
  placement?: 'left-top' | 'left-bottom' | 'center-top' | 'center-bottom' | 'right-top' | 'right-bottom' | 'max';
  /** 窗口触发方式 默认 `'click'` */
  trigger?: 'click' | 'sticky' | 'none';
  /** 窗口尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'xxl';
  /** 窗口内容高度 */
  height?: string;
  /** 特殊对话框类型 */
  feature?: string;
  /** 打开动画类型 默认 `'fadeIn'` */
  in?: 'fadeIn' | 'fadeInDown' | 'fadeInUp' | 'slideDown' | 'fadeInStart' | 'fadeInEnd' | 'springIn' | 'scaleIn' | 'easeIn';
  /** 关闭动画类型 默认 `'fadeOut'` */
  out?: 'fadeOut' | 'fadeOutDown' | 'fadeOutUp' | 'slideUp' | 'fadeOutStart' | 'fadeOutEnd' | 'springOut' | 'scaleOut' | 'easeOut';
  /** 是否可拖拽 默认 `true` */
  draggable?: boolean;
  /** 手机端是否自动居中 默认 `true` */
  adaptive?: boolean;
  /** 拖拽实例参数 默认 `{}` */
  gesture?: Record<string, any>;
  /** 父容器选择器 */
  parent?: string;
  /* ---- 内容配置 ---- */
  /** 数据源 */
  content?: any;
  /** 内容类型 默认 `'text'` */
  contType?: 'text' | 'html' | 'image' | 'video' | 'audio' | 'iframe' | 'async' | 'form' | 'input' | 'textarea' | 'auto';
  /** 请求数据或媒体属性 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求参数 默认 `{}` */
  ajax?: Record<string, any>;
  /** 打开时目标节点添加的类名 默认 `null` */
  actClass?: string;
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎 默认 `null` */
  tplEng?: (...args: any[]) => any;
  /** 点击触发判断函数 默认 `null` */
  canClick?: (...args: any[]) => any;
  /* ---- 等待函数 ---- */
  /** 显示前等待函数 默认 `null` */
  b4Show?: (...args: any[]) => any;
  /** 隐藏前等待函数 默认 `null` */
  b4Hide?: (...args: any[]) => any;
  /** 填充前等待函数 默认 `null` */
  b4Fill?: (...args: any[]) => any;
  /** 确认前等待函数 默认 `null` */
  b4Confirm?: (...args: any[]) => any;
  /** 取消前等待函数 默认 `null` */
  b4Cancel?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 显示前回调 默认 `null` */
  onShow?: (...args: any[]) => any;
  /** 隐藏前回调 默认 `null` */
  onHide?: (...args: any[]) => any;
  /** 显示后回调 默认 `null` */
  onShown?: (...args: any[]) => any;
  /** 隐藏后回调 默认 `null` */
  onHidden?: (...args: any[]) => any;
  /** 表单验证失败回调 默认 `null` */
  onInvalidated?: (...args: any[]) => any;
  /** 表单验证成功回调 默认 `null` */
  onValidated?: (...args: any[]) => any;
  /** 表单发送成功回调 默认 `null` */
  onSended?: (...args: any[]) => any;
  /** 渲染开始回调 默认 `null` */
  onRender?: (...args: any[]) => any;
  /** 渲染完成回调 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 内容更新回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /** 宿主节点更新回调 默认 `null` */
  onTransferred?: (...args: any[]) => any;
  /** 确认按钮点击回调 默认 `null` */
  onConfirmed?: (...args: any[]) => any;
  /** 取消按钮点击回调 默认 `null` */
  onCanceled?: (...args: any[]) => any;
  /** 关闭按钮点击回调 默认 `null` */
  onClosed?: (...args: any[]) => any;
  /** 清除按钮点击回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 异步请求完成回调 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /** 列表checked回调 默认 `null` */
  onBulletsChecked?: (...args: any[]) => any;
  /** 列表selected回调 默认 `null` */
  onBulletsSelected?: (...args: any[]) => any;
  /** 目标节点填充回调 默认 `null` */
  onTargetSet?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `true` */
  multiple?: boolean;
  /** 默认 `{"selector": "", "actClass": ""}` */
  wing?: AxDialogWingOptions;
  /** 默认 `{"enable": true, "closable": true}` */
  mask?: AxDialogMaskOptions;
  /** 默认 `{"enable": true, "body": true, "foot": true}` */
  padding?: AxDialogPaddingOptions;
  /** 默认 `{"title": "", "brief": ""}` */
  media?: AxDialogMediaOptions;
  /** 默认 `{"enable": false, "inputSel": "", "childSel": "", "detectable": false, "attr": ""}` */
  autoFill?: AxDialogAutofillOptions;
  /** 默认 `{"enable": false, "type": "custom", "parentNode": "ul", "childNode": "li", "unpadded": false, "hoverable": false, "divisible": false, "multiline": false, "cols": 0, "lines": "", "action": null}` */
  bullet?: AxDialogBulletOptions;
  /** 默认 `{"enable": true, "placement": "inside", "children": ["close"]}` */
  tools?: AxDialogToolsOptions;
  /** 默认 `{"enable": true, "layout": "plain", "note": false, "divider": false, "padding": false, "children": ["cancel", "confirm"]}` */
  footer?: AxDialogFooterOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Dialog 的 wing 配置组 */
export interface AxDialogWingOptions {
  /* ---- Dialog专属配置 ---- */
  /** 触发元素选择器 */
  selector?: string | any[];
  /** 触发按钮激活样式类 */
  actClass?: string;
}

/** Dialog 的 mask 配置组 */
export interface AxDialogMaskOptions {
  /* ---- Dialog专属配置 ---- */
  /** 是否显示遮罩层 默认 `true` */
  enable?: boolean;
  /** 点击遮罩是否关闭窗口 默认 `true` */
  closable?: boolean;
}

/** Dialog 的 padding 配置组 */
export interface AxDialogPaddingOptions {
  /* ---- 内边距配置 ---- */
  /** 是否显示内边距 默认 `true` */
  enable?: boolean;
  /** 内容容器的内边距类名，true即添加_p 默认 `true` */
  body?: string | boolean;
  /** 底部容器的内边距类名，true即添加_p 默认 `true` */
  foot?: string | boolean;
}

/** Dialog 的 media 配置组 */
export interface AxDialogMediaOptions {
  /* ---- 媒体配置 ---- */
  /** 媒体标题 */
  caption?: string;
  /** 媒体简介 */
  brief?: string;
}

/** Dialog 的 autoFill 配置组 */
export interface AxDialogAutofillOptions {
  /* ---- 自动填充配置 ---- */
  /** 是否自动填充 默认 `false` */
  enable?: boolean;
  /** 填充节点选择器 */
  inputSel?: string;
  /** 子节点选择器 */
  childSel?: string;
  /** 是否允许反向取值 默认 `false` */
  detectable?: boolean;
  /** 填充节点属性 */
  attr?: string;
}

/** Dialog 的 bullet 配置组 */
export interface AxDialogBulletOptions {
  /* ---- 列表配置 ---- */
  /** 是否启用列表模式 默认 `false` */
  enable?: boolean;
  /** 列表类型 默认 `'custom'` */
  type?: 'checkboxes' | 'radios' | 'select-single' | 'select-multi' | 'custom';
  /** 列表父节点名 默认 `'ul'` */
  parentNode?: string;
  /** 列表子节点名 默认 `'li'` */
  childNode?: string;
  /** 是否无内间距 默认 `false` */
  unpadded?: boolean;
  /** 是否支持悬停效果 默认 `false` */
  hoverable?: boolean;
  /** 是否显示分割线 默认 `false` */
  divisible?: boolean;
  /** 是否多行显示 默认 `false` */
  multiline?: boolean;
  /** 网格列数 默认 `0` */
  cols?: number;
  /** 项目间隔类型 */
  lines?: any;
  /** 列表项处理函数 默认 `null` */
  action?: (...args: any[]) => any;
}

/** Dialog 的 tools 配置组 */
export interface AxDialogToolsOptions {
  /* ---- 工具栏配置 ---- */
  /** 是否显示工具栏 默认 `true` */
  enable?: boolean;
  /** 工具栏位置 默认 `'inside'` */
  placement?: 'inside' | 'outside';
  /** 工具按钮列表 默认 `['close']` */
  children?: any[];
}

/** Dialog 的 footer 配置组 */
export interface AxDialogFooterOptions {
  /* ---- 底部配置 ---- */
  /** 是否显示底部 默认 `true` */
  enable?: boolean;
  /** 按钮布局方式 默认 `'plain'` */
  layout?: 'center' | 'plain' | 'right' | 'left';
  /** 是否显示底部提示 默认 `false` */
  note?: boolean;
  /** 是否显示底部分割线 默认 `false` */
  divider?: boolean;
  /** 是否显示内边距 默认 `false` */
  padding?: boolean;
  /** 底部按钮列表 默认 `['cancel', 'confirm']` */
  children?: any[];
}

/** Dialog 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxDialogEventName = 'constructed' | 'hidden' | 'hide' | 'initiate' | 'initiated' | 'show' | 'shown';

/** Dialog 模块实例（new Dialog() 的返回值） */
export interface AxDialogInstance extends AxModBase {
  options: AxDialogOptions;
  /** 事件监听（事件名见 AxDialogEventName；回调参数视事件而定） */
  on(type: AxDialogEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setFeature(...args: any[]): any;
  addPulseEvt(...args: any[]): any;
  removePulseEvt(...args: any[]): any;
  fixAni(...args: any[]): any;
  handleTools(...args: any[]): any;
  setAttrs(...args: any[]): any;
  setDraggable(...args: any[]): any;
  bindTrigger(...args: any[]): any;
  unbindTrigger(...args: any[]): any;
  toggleHide(...args: any[]): any;
  show(...args: any[]): any;
  hide(...args: any[]): any;
}

/* ========================================================================== */
/* Dodge                                                                      */
/* （无独立文档页，参数由源码默认值提取）                                                        */
/* ========================================================================== */
/** Dodge 构造参数 —— new Dodge(elem, options) 的 options */
export interface AxDodgeOptions {
  /* ---- 源码默认参数（文档未单列） ---- */
  placeholder?: string;
  /** 默认 `null` */
  onSetted?: ((...args: any[]) => any);
  /** 默认 `null` */
  onCanceled?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Dodge 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxDodgeEventName = 'constructed' | 'initiate' | 'initiated' | 'set';

/** Dodge 模块实例（new Dodge() 的返回值） */
export interface AxDodgeInstance extends AxModBase {
  options: AxDodgeOptions;
  /** 事件监听（事件名见 AxDodgeEventName；回调参数视事件而定） */
  on(type: AxDodgeEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  getPlaceholder(...args: any[]): any;
  setAttrs(...args: any[]): any;
  set(...args: any[]): any;
  cancel(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Drag · 拖拽模块                                                                */
/* 文档: modules/323.html                                                       */
/* ========================================================================== */
/** Drag 构造参数 —— new Drag(elem, options) 的 options */
export interface AxDragOptions {
  /* ---- 基础配置 ---- */
  /** 是否使用原生拖拽事件（PC端使用drag/drop，移动端使用touch） 默认 `true` */
  original?: boolean;
  /** 定义可互相拖拽的实例组标识 */
  group?: string;
  /** 拖拽过程中的节流时间（ms） 默认 `100` */
  throttle?: number;
  /** 拖拽释放后的动画时长（ms） 默认 `200` */
  duration?: number;
  /* ---- 节点选择器 ---- */
  /** 可拖拽元素选择器，默认[dragnode] */
  drags?: string | Element[];
  /** 可放置目标选择器，默认[dropnode] */
  drops?: string | Element[];
  /** drags/drops的共同父节点选择器 */
  parent?: string | HTMLElement;
  /** 可接收空拖放的容器选择器 */
  wraps?: string | Element[];
  /** 拖拽触发手柄选择器 */
  handle?: string;
  /** 禁止拖拽的元素选择器 */
  ignore?: string | Element[];
  /* ---- 拖拽行为 ---- */
  /** 节点转移时机：'hover'(悬停时)/'drop'(释放时) 默认 `'hover'` */
  method?: string;
  /** 拖拽目的：'sort'(排序)/'embed'(嵌入)/'swap'(交换)/'auto'(自动) 默认 `'auto'` */
  purpose?: string;
  /** 是否显示空列表占位符 默认 `true` */
  showEmpty?: boolean;
  /** 是否允许从其他区域拖入 默认 `true` */
  pullable?: boolean;
  /* ---- 高级配置 ---- */
  /** 节点嵌入配置 默认 `{position:'after',selector:''}` */
  embed?: Record<string, any>;
  /** 节点推出配置 默认 `{enable:true,intent:'cut'}` */
  pushable?: boolean | Record<string, any>;
  /** 插入位置判定点 默认 `{before:['t/3','t/2'],after:['b/3','b/2']}` */
  point?: Record<string, any>;
  /** 占位符配置 默认 `{enable:false,style:[]}` */
  holder?: boolean | Record<string, any>;
  /** 指示箭头配置 默认 `{enable:false,icon:'_icon-right',placement:'left',selector:''}` */
  arrow?: boolean | Record<string, any>;
  /** 手势模块配置 默认 `{}` */
  gesture?: Record<string, any>;
  /** 翻转动画配置 默认 `{enable:true,fluct:-100}` */
  flip?: boolean | Record<string, any>;
  /* ---- 等待函数 ---- */
  /** 释放前等待函数 默认 `null` */
  b4Drop?: (...args: any[]) => any;
  /** 触发前等待函数 默认 `null` */
  b4Trigger?: (...args: any[]) => any;
  /* ---- 事件回调 ---- */
  /** 开始拖拽事件 默认 `null` */
  onDragStart?: (...args: any[]) => any;
  /** 进入有效区域事件 默认 `null` */
  onDragEnter?: (...args: any[]) => any;
  /** 离开有效区域事件 默认 `null` */
  onDragLeave?: (...args: any[]) => any;
  /** 拖拽移动事件 默认 `null` */
  onDragMove?: (...args: any[]) => any;
  /** 释放事件 默认 `null` */
  onDropped?: (...args: any[]) => any;
  /** 拖拽结束事件 默认 `null` */
  onDragEnd?: (...args: any[]) => any;
  /** 拖拽取消事件 默认 `null` */
  onDragCancel?: (...args: any[]) => any;
  /** 节点转移完成事件 默认 `null` */
  onTransferred?: (...args: any[]) => any;
  /** 节点交换完成事件 默认 `null` */
  onSwapped?: (...args: any[]) => any;
  /** 节点嵌入完成事件 默认 `null` */
  onEmbeded?: (...args: any[]) => any;
  /** 节点排序完成事件 默认 `null` */
  onSorted?: (...args: any[]) => any;
  /** 节点克隆完成事件 默认 `null` */
  onCloned?: (...args: any[]) => any;
  /** 拖拽触发事件 默认 `null` */
  onTrigger?: (...args: any[]) => any;
  /** 拖拽完全结束事件 默认 `null` */
  onFinished?: (...args: any[]) => any;
  /** 拖至无效区域事件 默认 `null` */
  onDragOut?: (...args: any[]) => any;
  /** 无效区域释放事件 默认 `null` */
  onDropOut?: (...args: any[]) => any;
}

/** Drag 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxDragEventName = 'cloned' | 'constructed' | 'dragCancel' | 'dragEnd' | 'dragEnter' | 'dragLeave' | 'dragMove' | 'dragOut' | 'dragStart' | 'dropOut' | 'dropped' | 'embeded' | 'finished' | 'initiate' | 'initiated' | 'sorted' | 'swapped' | 'trigger' | 'updatedList';

/** Drag 模块实例（new Drag() 的返回值） */
export interface AxDragInstance extends AxModBase {
  options: AxDragOptions;
  /** 事件监听（事件名见 AxDragEventName；回调参数视事件而定） */
  on(type: AxDragEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setCommDrag(...args: any[]): any;
  canClone(...args: any[]): any;
  cloneStart(...args: any[]): any;
  cloneEnd(...args: any[]): any;
  getTransferType(...args: any[]): any;
  getFlipChldren(...args: any[]): any;
  getCopyNode(...args: any[]): any;
  getGhostNode(...args: any[]): any;
  setGhostPos(...args: any[]): any;
  hideGhost(...args: any[]): any;
  updateHandleEls(...args: any[]): any;
  updateIgnoreEls(...args: any[]): any;
  updateWrapEls(...args: any[]): any;
  setMutationObs(...args: any[]): any;
  setWrapsObs(...args: any[]): any;
  hideHolder(...args: any[]): any;
  replaceNodes(...args: any[]): any;
  swapNodes(...args: any[]): any;
  getEmbedEl(...args: any[]): any;
  insertable(...args: any[]): any;
  getEmbedPoint(...args: any[]): any;
  insertEl(...args: any[]): any;
  updatePoints(...args: any[]): any;
  insertUpdateArrow(...args: any[]): any;
  removeArrow(...args: any[]): any;
  setHolderAttrs(...args: any[]): any;
  toggleDropped(...args: any[]): any;
  removeDroppeds(...args: any[]): any;
  toggleDropping(...args: any[]): any;
  removeDroppings(...args: any[]): any;
  removeDraggables(...args: any[]): any;
  setFlip(...args: any[]): any;
  setDestopDrag(...args: any[]): any;
  addMouseEvents(...args: any[]): any;
  removeMouseEvents(...args: any[]): any;
  addDestopEvents(...args: any[]): any;
  removeDestopEvents(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getGoupDrags(...args: any[]): any;
  getSelfDropEls(...args: any[]): any;
  getSelfDragEls(...args: any[]): any;
  updateDragsDrops(...args: any[]): any;
  handleEmpty(...args: any[]): any;
  updateList(...args: any[]): any;
  revert(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Drawer · 抽屉模块                                                              */
/* 文档: modules/322.html                                                       */
/* ========================================================================== */
/** Drawer 构造参数 —— new Drawer(elem, options) 的 options */
export interface AxDrawerOptions {
  /* ---- 基础配置参数 ---- */
  /** 窗口标题 */
  heading?: string;
  /** 是否显示标题分割线 默认 `false` */
  divider?: boolean;
  /** 动画时长(ms) 默认 `0` */
  duration?: number;
  /** 是否自动计算动画时间 默认 `false` */
  autoDur?: boolean;
  /** 是否禁用show方法 默认 `false` */
  asleep?: boolean;
  /** 打开后是否禁止关闭 默认 `false` */
  deadShow?: boolean;
  /** 是否只允许显示不允许关闭 默认 `false` */
  keepShow?: boolean;
  /** 窗口样式类 */
  classes?: string;
  /** 窗口层级 默认 `0` */
  zIndex?: number;
  /** 是否阻止与body联动滚动 默认 `false` */
  dedicated?: boolean;
  /* ---- Drawer专属配置 ---- */
  /** 抽屉显示位置 默认 `'right'` */
  placement?: 'left' | 'right' | 'top' | 'bottom' | 'max';
  /** 抽屉触发方式 默认 `'click'` */
  trigger?: 'click' | 'sticky' | 'none';
  /** 抽屉尺寸 默认 `'sm'` */
  size?: 'sm' | 'md' | 'lg';
  /** 子层抽屉偏移距离 默认 `'100px'` */
  offset?: string;
  /** 父容器选择器 */
  parent?: string;
  /* ---- 内容配置 ---- */
  /** 数据源 */
  content?: any;
  /** 内容类型 默认 `'text'` */
  contType?: 'text' | 'html' | 'image' | 'video' | 'audio' | 'iframe' | 'async' | 'form' | 'input' | 'textarea' | 'auto';
  /** 请求数据或媒体属性 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求参数 默认 `{}` */
  ajax?: Record<string, any>;
  /** 打开时目标节点添加的类名 默认 `null` */
  actClass?: string;
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎 默认 `null` */
  tplEng?: (...args: any[]) => any;
  /** 点击触发判断函数 默认 `null` */
  canClick?: (...args: any[]) => any;
  /* ---- 等待函数 ---- */
  /** 显示前等待函数 默认 `null` */
  b4Show?: (...args: any[]) => any;
  /** 隐藏前等待函数 默认 `null` */
  b4Hide?: (...args: any[]) => any;
  /** 填充前等待函数 默认 `null` */
  b4Fill?: (...args: any[]) => any;
  /** 确认前等待函数 默认 `null` */
  b4Confirm?: (...args: any[]) => any;
  /** 取消前等待函数 默认 `null` */
  b4Cancel?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 显示前回调 默认 `null` */
  onShow?: (...args: any[]) => any;
  /** 隐藏前回调 默认 `null` */
  onHide?: (...args: any[]) => any;
  /** 显示后回调 默认 `null` */
  onShown?: (...args: any[]) => any;
  /** 隐藏后回调 默认 `null` */
  onHidden?: (...args: any[]) => any;
  /** 表单验证失败回调 默认 `null` */
  onInvalidated?: (...args: any[]) => any;
  /** 表单验证成功回调 默认 `null` */
  onValidated?: (...args: any[]) => any;
  /** 表单发送成功回调 默认 `null` */
  onSended?: (...args: any[]) => any;
  /** 渲染开始回调 默认 `null` */
  onRender?: (...args: any[]) => any;
  /** 渲染完成回调 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 内容更新回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /** 宿主节点更新回调 默认 `null` */
  onTransferred?: (...args: any[]) => any;
  /** 确认按钮点击回调 默认 `null` */
  onConfirmed?: (...args: any[]) => any;
  /** 取消按钮点击回调 默认 `null` */
  onCanceled?: (...args: any[]) => any;
  /** 关闭按钮点击回调 默认 `null` */
  onClosed?: (...args: any[]) => any;
  /** 清除按钮点击回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 异步请求完成回调 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /** 列表checked回调 默认 `null` */
  onBulletsChecked?: (...args: any[]) => any;
  /** 列表selected回调 默认 `null` */
  onBulletsSelected?: (...args: any[]) => any;
  /** 目标节点填充回调 默认 `null` */
  onTargetSet?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `true` */
  multiple?: boolean;
  /** 默认 `{"selector": "", "actClass": ""}` */
  wing?: AxDrawerWingOptions;
  /** 默认 `{"enable": true, "closable": true}` */
  mask?: AxDrawerMaskOptions;
  /** 默认 `{"enable": true, "body": true, "foot": true}` */
  padding?: AxDrawerPaddingOptions;
  /** 默认 `{"title": "", "brief": ""}` */
  media?: AxDrawerMediaOptions;
  /** 默认 `{"enable": false, "inputSel": "", "childSel": "", "detectable": false, "attr": ""}` */
  autoFill?: AxDrawerAutofillOptions;
  /** 默认 `{"enable": false, "type": "custom", "parentNode": "ul", "childNode": "li", "unpadded": false, "hoverable": false, "divisible": false, "multiline": false, "cols": 0, "lines": "", "action": null}` */
  bullet?: AxDrawerBulletOptions;
  /** 默认 `{"enable": true, "placement": "inside", "children": ["close"]}` */
  tools?: AxDrawerToolsOptions;
  /** 默认 `{"enable": true, "layout": "plain", "note": false, "divider": false, "padding": false, "children": ["cancel", "confirm"]}` */
  footer?: AxDrawerFooterOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Drawer 的 wing 配置组 */
export interface AxDrawerWingOptions {
  /* ---- Drawer专属配置 ---- */
  /** 触发元素选择器 */
  selector?: string | any[];
  /** 触发按钮激活样式类 */
  actClass?: string;
}

/** Drawer 的 mask 配置组 */
export interface AxDrawerMaskOptions {
  /* ---- Drawer专属配置 ---- */
  /** 是否显示遮罩层 默认 `true` */
  enable?: boolean;
  /** 点击遮罩是否关闭窗口 默认 `true` */
  closable?: boolean;
}

/** Drawer 的 padding 配置组 */
export interface AxDrawerPaddingOptions {
  /* ---- 内边距配置 ---- */
  /** 是否显示内边距 默认 `true` */
  enable?: boolean;
  /** 内容容器的内边距类名，true即添加_p 默认 `true` */
  body?: string | boolean;
  /** 底部容器的内边距类名，true即添加_p 默认 `true` */
  foot?: string | boolean;
}

/** Drawer 的 media 配置组 */
export interface AxDrawerMediaOptions {
  /* ---- 媒体配置 ---- */
  /** 媒体标题 */
  caption?: string;
  /** 媒体简介 */
  brief?: string;
}

/** Drawer 的 autoFill 配置组 */
export interface AxDrawerAutofillOptions {
  /* ---- 自动填充配置 ---- */
  /** 是否自动填充 默认 `false` */
  enable?: boolean;
  /** 填充节点选择器 */
  inputSel?: string;
  /** 子节点选择器 */
  childSel?: string;
  /** 是否允许反向取值 默认 `false` */
  detectable?: boolean;
  /** 填充节点属性 */
  attr?: string;
}

/** Drawer 的 bullet 配置组 */
export interface AxDrawerBulletOptions {
  /* ---- 列表配置 ---- */
  /** 是否启用列表模式 默认 `false` */
  enable?: boolean;
  /** 列表类型 默认 `'custom'` */
  type?: 'checkboxes' | 'radios' | 'select-single' | 'select-multi' | 'custom';
  /** 列表父节点名 默认 `'ul'` */
  parentNode?: string;
  /** 列表子节点名 默认 `'li'` */
  childNode?: string;
  /** 是否无内间距 默认 `false` */
  unpadded?: boolean;
  /** 是否支持悬停效果 默认 `false` */
  hoverable?: boolean;
  /** 是否显示分割线 默认 `false` */
  divisible?: boolean;
  /** 是否多行显示 默认 `false` */
  multiline?: boolean;
  /** 网格列数 默认 `0` */
  cols?: number;
  /** 项目间隔类型 */
  lines?: any;
  /** 列表项处理函数 默认 `null` */
  action?: (...args: any[]) => any;
}

/** Drawer 的 tools 配置组 */
export interface AxDrawerToolsOptions {
  /* ---- 工具栏配置 ---- */
  /** 是否显示工具栏 默认 `true` */
  enable?: boolean;
  /** 工具栏位置 默认 `'inside'` */
  placement?: 'inside' | 'outside';
  /** 工具按钮列表 默认 `['close']` */
  children?: any[];
}

/** Drawer 的 footer 配置组 */
export interface AxDrawerFooterOptions {
  /* ---- 底部配置 ---- */
  /** 是否显示底部 默认 `true` */
  enable?: boolean;
  /** 按钮布局方式 默认 `'plain'` */
  layout?: 'center' | 'plain' | 'right' | 'left';
  /** 是否显示底部提示 默认 `false` */
  note?: boolean;
  /** 是否显示底部分割线 默认 `false` */
  divider?: boolean;
  /** 是否显示内边距 默认 `false` */
  padding?: boolean;
  /** 底部按钮列表 默认 `['cancel', 'confirm']` */
  children?: any[];
}

/** Drawer 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxDrawerEventName = 'constructed' | 'hidden' | 'hide' | 'initiate' | 'initiated' | 'show' | 'shown';

/** Drawer 模块实例（new Drawer() 的返回值） */
export interface AxDrawerInstance extends AxModBase {
  options: AxDrawerOptions;
  /** 事件监听（事件名见 AxDrawerEventName；回调参数视事件而定） */
  on(type: AxDrawerEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  handleTools(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getBeforeItems(...args: any[]): any;
  getToggleItems(...args: any[]): any;
  getIngItems(...args: any[]): any;
  bindTrigger(...args: any[]): any;
  unbindTrigger(...args: any[]): any;
  show(...args: any[]): any;
  hide(...args: any[]): any;
}

/* ========================================================================== */
/* Dropdown · 下拉菜单模块                                                          */
/* 文档: modules/321.html                                                       */
/* ========================================================================== */
/** Dropdown 构造参数 —— new Dropdown(elem, options) 的 options */
export interface AxDropdownOptions {
  /* ---- 数据源配置 ---- */
  /** 数据源 */
  content?: any;
  /** 数据类型 默认 `'text'` */
  contType?: string;
  /** 请求数据 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求参数 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 显示配置 ---- */
  /** 相对位置 默认 `'bottom-start'` */
  placement?: string;
  /** 特殊组合类型 */
  feature?: string;
  /** 弹窗模块参数 默认 `{}` */
  popup?: Record<string, any>;
  /* ---- 列表配置 ---- */
  /** 列表类型 */
  listType?: string;
  /** 是否无内边距 默认 `false` */
  paddingless?: boolean;
  /** 是否支持悬停效果 默认 `false` */
  hoverable?: boolean;
  /** 是否显示分割线 默认 `false` */
  divisible?: boolean;
  /** 是否多行显示 默认 `false` */
  multiline?: boolean;
  /** 网格布局列数 默认 `0` */
  cols?: number;
  /** 项目间隔类型 */
  lines?: string;
  /* ---- 节点配置 ---- */
  /** 子节点选择器 默认 `'[="label"]'` */
  childSel?: string;
  /** 取值属性名 */
  valKey?: string;
  /** 是否校验值一致性 默认 `false` */
  detectable?: boolean;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `false` */
  unpadded?: boolean;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Dropdown 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxDropdownEventName = 'constructed' | 'initiate' | 'initiated';

/** Dropdown 模块实例（new Dropdown() 的返回值） */
export interface AxDropdownInstance extends AxModBase {
  options: AxDropdownOptions;
  /** 事件监听（事件名见 AxDropdownEventName；回调参数视事件而定） */
  on(type: AxDropdownEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setAttrs(...args: any[]): any;
  setFeature(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Editor · 编辑器模块                                                             */
/* 文档: modules/317.html                                                       */
/* ========================================================================== */
/** Editor 构造参数 —— new Editor(elem, options) 的 options */
export interface AxEditorOptions {
  /* ---- 基础配置 ---- */
  /** 隐藏字段的name属性 */
  name?: string;
  /** 初始纯文本内容 */
  value?: string;
  /** 是否禁用编辑器 默认 `false` */
  disabled?: boolean;
  /** 是否只读模式 默认 `false` */
  readonly?: boolean;
  /** 是否延迟渲染子节点 默认 `false` */
  deferred?: boolean;
  /** 编辑事件防抖时间(ms) 默认 `200` */
  delay?: number;
  /** 编辑器额外样式类 */
  classes?: string;
  /* ---- 数据源配置 ---- */
  /** 编辑器数据源 */
  content?: string | any[];
  /** 内容类型 默认 `'text'` */
  contType?: 'text' | 'async' | 'html' | 'auto';
  /** 请求数据参数 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 外观配置 ---- */
  /** 显示模式 默认 `'editor'` */
  mode?: 'source' | 'editor';
  /** 最小高度 */
  minHeight?: string;
  /** 最大高度 */
  maxHeight?: string;
  /** 功能组合类型 默认 `'simple'` */
  feature?: 'simple' | 'comm' | 'flat' | 'group' | 'full';
  /* ---- 等待函数 ---- */
  /** 粘贴前等待函数 默认 `null` */
  b4Paste?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 编辑后回调 默认 `null` */
  onEdited?: (...args: any[]) => any;
  /** 禁用后回调 默认 `null` */
  onDisabled?: (...args: any[]) => any;
  /** 启用后回调 默认 `null` */
  onEnabled?: (...args: any[]) => any;
  /** 设置只读后回调 默认 `null` */
  onReadonly?: (...args: any[]) => any;
  /** 填充内容后回调 默认 `null` */
  onFilled?: (...args: any[]) => any;
  /** 设置值后回调 默认 `null` */
  onSet?: (...args: any[]) => any;
  /** 清除值后回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 更新内容后回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /** 异步请求完成回调 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /** 输出值后回调 默认 `null` */
  onOutput?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"rounded": true, "bordered": true}` */
  appear?: AxEditorAppearOptions;
  /** 默认 `{"enable": true, "children": []}` */
  header?: AxEditorHeaderOptions;
  /** 默认 `{"enable": true}` */
  footer?: AxEditorFooterOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Editor 的 appear 配置组 */
export interface AxEditorAppearOptions {
  /* ---- 外观配置 ---- */
  /** 是否显示圆角 默认 `true` */
  rounded?: boolean;
  /** 是否显示边框 默认 `true` */
  bordered?: boolean;
}

/** Editor 的 header 配置组 */
export interface AxEditorHeaderOptions {
  /* ---- 工具栏配置 ---- */
  /** 是否启用头部按钮组 默认 `true` */
  enable?: boolean;
  /** 按钮布局配置 默认 `[]` */
  children?: any[];
}

/** Editor 的 footer 配置组 */
export interface AxEditorFooterOptions {
  /* ---- 工具栏配置 ---- */
  /** 是否启用底部信息栏 默认 `true` */
  enable?: boolean;
}

/** Editor 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxEditorEventName = 'constructed' | 'edited' | 'filled' | 'initiate' | 'initiated' | 'output' | 'pasted' | 'set';

/** Editor 模块实例（new Editor() 的返回值） */
export interface AxEditorInstance extends AxModBase {
  options: AxEditorOptions;
  /** 事件监听（事件名见 AxEditorEventName；回调参数视事件而定） */
  on(type: AxEditorEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  insertNewPara(...args: any[]): any;
  getTools(...args: any[]): any;
  renderMain(...args: any[]): any;
  updateInfo(...args: any[]): any;
  setPathInfo(...args: any[]): any;
  setStateInfo(...args: any[]): any;
  setAttrs(...args: any[]): any;
  readonly(...args: any[]): any;
  disable(...args: any[]): any;
  enable(...args: any[]): any;
  renderFinish(...args: any[]): any;
  fullTextNode(...args: any[]): any;
  setChildRange(...args: any[]): any;
  setStyles(...args: any[]): any;
  getParentNode(...args: any[]): any;
  removeStyles(...args: any[]): any;
  toggleStyles(...args: any[]): any;
  addOneWrap(...args: any[]): any;
  addWraps(...args: any[]): any;
  replaceLineTag(...args: any[]): any;
  setLineStyles(...args: any[]): any;
  createNestWrap(...args: any[]): any;
  insertNestNodes(...args: any[]): any;
  insertTag(...args: any[]): any;
  clearRanges(...args: any[]): any;
  isValidRange(...args: any[]): any;
  createFontsizeItems(...args: any[]): any;
  createToggleItems(...args: any[]): any;
  createAlignItems(...args: any[]): any;
  createLineTags(...args: any[]): any;
  createSpanTags(...args: any[]): any;
  createCloseTags(...args: any[]): any;
  createNestTags(...args: any[]): any;
  fillCont(...args: any[]): any;
  updateInputVal(...args: any[]): any;
  setVals(...args: any[]): any;
  getVals(...args: any[]): any;
  clearVals(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Flip · 动画模块                                                                */
/* 文档: modules/392.html                                                       */
/* ========================================================================== */
/** Flip 构造参数 —— new Flip(elem, options) 的 options */
export interface AxFlipOptions {
  /* ---- 基础配置 ---- */
  /** 父容器选择器 */
  parent?: string;
  /** 子节点选择器 */
  children?: string | Element[];
  /** 是否禁止play 默认 `false` */
  prevent?: boolean;
  /** flip动画时长波动值，单位ms，默认为0即自动计算 默认 `0` */
  fluct?: number;
  /** 动画曲线，同transition-timing-function值 默认 `'ease-out'` */
  easing?: string;
  /* ---- 等待函数 ---- */
  /** 添加一个或多个节点Flip动画之前执行 默认 `null` */
  b4Add?: (...args: any[]) => any;
  /** 删除一个或多个节点Flip动画之前执行 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /** 单个节点play之前执行 默认 `null` */
  b4Play?: (...args: any[]) => any;
  /** 所有节点play之前执行 默认 `null` */
  b4PlayAll?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 单个节点play完成时回调 默认 `null` */
  onPlayed?: (...args: any[]) => any;
  /** 所有节点play完成时回调 默认 `null` */
  onPlayedAll?: (...args: any[]) => any;
  /** 添加新节点Flip动画回调 默认 `null` */
  onAdded?: (...args: any[]) => any;
  /** 删除节点Flip动画回调 默认 `null` */
  onRemoved?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  other?: string;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Flip 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxFlipEventName = 'added' | 'constructed' | 'initiate' | 'initiated' | 'played' | 'playedAll' | 'removed';

/** Flip 模块实例（new Flip() 的返回值） */
export interface AxFlipInstance extends AxModBase {
  options: AxFlipOptions;
  /** 事件监听（事件名见 AxFlipEventName；回调参数视事件而定） */
  on(type: AxFlipEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  play(...args: any[]): any;
  getDuration(...args: any[]): any;
  getTranslate(...args: any[]): any;
  getRect(...args: any[]): any;
  setFirstRect(...args: any[]): any;
  setNowRects(...args: any[]): any;
  updateFlipEls(...args: any[]): any;
  add(...args: any[]): any;
  remove(...args: any[]): any;
  playAll(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Gesture · 多终端统一手势库模块                                                       */
/* 文档: modules/311.html                                                       */
/* ========================================================================== */
/** Gesture 构造参数 —— new Gesture(elem, options) 的 options */
export interface AxGestureOptions {
  /* ---- 基础配置 ---- */
  /** 父容器选择器 */
  parent?: string;
  /** 不绑定事件的元素选择器 默认 `'[unbound]'` */
  unbound?: string;
  /** 元素变换原点 默认 `null` */
  origin?: Record<string, any> | null;
  /** 点击抖动阈值 默认 `10` */
  jitterClick?: number;
  /** 移动防抖值 默认 `1` */
  jitterTrans?: number;
  /* ---- 回调函数 ---- */
  /** 开始触摸回调 默认 `null` */
  onStart?: (...args: any[]) => any;
  /** 移动过程回调 默认 `null` */
  onMove?: (...args: any[]) => any;
  /** 结束触摸回调 默认 `null` */
  onEnd?: (...args: any[]) => any;
  /** 点击回调 默认 `null` */
  onClick?: (...args: any[]) => any;
  /** 双击回调 默认 `null` */
  onDblclick?: (...args: any[]) => any;
  /** 长按回调 默认 `null` */
  onHold?: (...args: any[]) => any;
  /** 开始缩放回调 默认 `null` */
  onScale?: (...args: any[]) => any;
  /** 缩放过程回调 默认 `null` */
  onScaling?: (...args: any[]) => any;
  /** 缩放结束回调 默认 `null` */
  onScaled?: (...args: any[]) => any;
  /** 开始旋转回调 默认 `null` */
  onRotate?: (...args: any[]) => any;
  /** 旋转过程回调 默认 `null` */
  onRotating?: (...args: any[]) => any;
  /** 旋转结束回调 默认 `null` */
  onRotated?: (...args: any[]) => any;
  /** 开始位移回调 默认 `null` */
  onTranslate?: (...args: any[]) => any;
  /** 位移过程回调 默认 `null` */
  onTranslating?: (...args: any[]) => any;
  /** 位移结束回调 默认 `null` */
  onTranslated?: (...args: any[]) => any;
  /** 开始漂移回调 默认 `null` */
  onDrift?: (...args: any[]) => any;
  /** 漂移过程回调 默认 `null` */
  onDrifting?: (...args: any[]) => any;
  /** 漂移结束回调 默认 `null` */
  onDrifted?: (...args: any[]) => any;
  /** 失去目标回调 默认 `null` */
  onCanceled?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"enable": false, "selector": "", "dmpRatio": 0.8, "bouncy": true, "duration": 0}` */
  viewport?: AxGestureViewportOptions;
  /** 默认 `{"enable": false}` */
  spy?: Record<string, any>;
  /** 默认 `{"timeThr": 200, "delay": 200, "dblclickable": true, "holdDelay": 1000, "hold2Menu": false}` */
  click?: AxGestureClickOptions;
  /** 默认 `{"mode": "translate", "axis": "y", "value": 0, "duration": 500, "intvl": 200, "linkage": true, "reverse": false, "curve": "easeOut"}` */
  step?: AxGestureStepOptions;
  /** 默认 `false` */
  wheel?: boolean;
  /** 默认 `{"enable": false, "prev": "ArrowUp", "next": "ArrowDown", "intvl": 200, "target": null}` */
  keyboard?: AxGestureKeyboardOptions;
  /** 默认 `{"enable": true, "max": 10000000, "min": -10000000, "instead": false, "target": ""}` */
  translate?: AxGestureTranslateOptions;
  /** 默认 `{"enable": false, "timeThr": 300, "distThr": 20, "coef": 0.5, "duration": 500, "curve": "easeOut", "auto": true}` */
  drift?: AxGestureDriftOptions;
  /** 默认 `{"enable": false, "timeThr": 100, "distThr": 20, "coef": 0.5, "duration": 500, "curve": "easeOut", "auto": true}` */
  swipe?: AxGestureSwipeOptions;
  /** 默认 `{"enable": false, "centered": true, "max": 100, "min": 0}` */
  scale?: AxGestureScaleOptions;
  /** 默认 `{"enable": false, "max": 3600, "min": -3600}` */
  rotate?: AxGestureRotateOptions;
  /** 默认 `null` */
  initialVal?: any;
  /** 默认 `null` */
  b4Trigger?: ((...args: any[]) => any);
  /** 默认 `null` */
  onEnter?: ((...args: any[]) => any);
  /** 默认 `null` */
  onLeave?: ((...args: any[]) => any);
  /** 默认 `null` */
  onTrigger?: ((...args: any[]) => any);
  /** 默认 `null` */
  onFinished?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Gesture 的 viewport 配置组 */
export interface AxGestureViewportOptions {
  /* ---- 视窗配置 ---- */
  /** 是否启用反弹视窗 默认 `false` */
  enable?: boolean;
  /** 视窗节点选择器 */
  selector?: string;
  /** 拖拽阻尼系数 默认 `0.8` */
  dmpRatio?: number;
  /** 是否允许边缘反弹 默认 `true` */
  bouncy?: boolean;
  /** 反弹动画时长(ms) 默认 `0` */
  duration?: number;
}

/** Gesture 的 click 配置组 */
export interface AxGestureClickOptions {
  /* ---- 点击配置 ---- */
  /** 点击事件时间阈值(ms) 默认 `200` */
  timeThr?: number;
  /** 双击等待时间(ms) 默认 `200` */
  delay?: number;
  /** 是否允许双击 默认 `true` */
  dblclickable?: boolean;
  /** 长按延时(ms) 默认 `1000` */
  holdDelay?: number;
  /** 长按是否等效右键 默认 `false` */
  hold2Menu?: boolean;
}

/** Gesture 的 step 配置组 */
export interface AxGestureStepOptions {
  /* ---- 步进配置 ---- */
  /** 步进模式 默认 `'translate'` */
  mode?: 'translate' | 'scale' | 'rotate';
  /** 移动方向 默认 `'y'` */
  axis?: 'x' | 'y';
  /** 步进值 默认 `0` */
  value?: number;
  /** 动画时长(ms) 默认 `500` */
  duration?: number;
  /** 节流间隔(ms) 默认 `200` */
  intvl?: number;
  /** 是否触发相应事件 默认 `true` */
  linkage?: boolean;
  /** 是否逆向操作 默认 `false` */
  reverse?: boolean;
  /** 动画曲线 默认 `'easeOut'` */
  curve?: 'linear' | 'ease' | 'easeIn' | 'easeOut' | 'easeInOut' | 'easeOutIn';
}

/** Gesture 的 keyboard 配置组 */
export interface AxGestureKeyboardOptions {
  /* ---- 键盘配置 ---- */
  /** 是否启用键盘 默认 `false` */
  enable?: boolean;
  /** 前进按键 默认 `'ArrowUp'` */
  prev?: string;
  /** 后退按键 默认 `'ArrowDown'` */
  next?: string;
  /** 节流间隔(ms) 默认 `200` */
  intvl?: number;
  /** 触发键盘事件的节点 默认 `null` */
  target?: string | null;
}

/** Gesture 的 translate 配置组 */
export interface AxGestureTranslateOptions {
  /* ---- 位移配置 ---- */
  /** 是否启用位移 默认 `true` */
  enable?: boolean;
  /** 最大位移值(px) 默认 `10000000` */
  max?: number;
  /** 最小位移值(px) 默认 `-10000000` */
  min?: number;
  /** 是否使用position 默认 `false` */
  instead?: boolean;
  /** 位移手柄选择器 */
  target?: string;
}

/** Gesture 的 drift 配置组 */
export interface AxGestureDriftOptions {
  /* ---- 漂移配置 ---- */
  /** 是否启用漂移 默认 `false` */
  enable?: boolean;
  /** 时间阈值(ms) 默认 `300` */
  timeThr?: number;
  /** 距离阈值(px) 默认 `20` */
  distThr?: number;
  /** 漂移系数 默认 `0.5` */
  coef?: number;
  /** 动画时长(ms) 默认 `500` */
  duration?: number;
  /** 运动曲线 默认 `'easeOut'` */
  curve?: 'linear' | 'ease' | 'easeIn' | 'easeOut' | 'easeInOut' | 'easeOutIn';
  /** 是否自动完成 默认 `true` */
  auto?: boolean;
}

/** Gesture 的 swipe 配置组 */
export interface AxGestureSwipeOptions {
  /* ---- 滑动配置 ---- */
  /** 是否启用滑动 默认 `false` */
  enable?: boolean;
  /** 时间阈值(ms) 默认 `100` */
  timeThr?: number;
  /** 距离阈值(px) 默认 `20` */
  distThr?: number;
  /** 滑动系数 默认 `0.5` */
  coef?: number;
  /** 动画时长(ms) 默认 `500` */
  duration?: number;
  /** 运动曲线 默认 `'easeOut'` */
  curve?: 'linear' | 'ease' | 'easeIn' | 'easeOut' | 'easeInOut' | 'easeOutIn';
  /** 是否自动完成 默认 `true` */
  auto?: boolean;
}

/** Gesture 的 scale 配置组 */
export interface AxGestureScaleOptions {
  /* ---- 缩放配置 ---- */
  /** 是否启用缩放 默认 `false` */
  enable?: boolean;
  /** 是否居中缩放 默认 `true` */
  centered?: boolean;
  /** 最大缩放倍数 默认 `100` */
  max?: number;
  /** 最小缩放倍数 默认 `0` */
  min?: number;
}

/** Gesture 的 rotate 配置组 */
export interface AxGestureRotateOptions {
  /* ---- 旋转配置 ---- */
  /** 是否启用旋转 默认 `false` */
  enable?: boolean;
  /** 最大旋转角度 默认 `3600` */
  max?: number;
  /** 最小旋转角度 默认 `-3600` */
  min?: number;
}

/** Gesture 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxGestureEventName = 'click' | 'constructed' | 'dblclick' | 'end' | 'enter' | 'finished' | 'hidden' | 'hold' | 'initiate' | 'initiated' | 'leave' | 'move' | 'rotate' | 'rotated' | 'rotating' | 'scale' | 'scaled' | 'scaling' | 'showing' | 'shown' | 'start' | 'step' | 'stepped' | 'translate' | 'translated' | 'translating' | 'trigger';

/** Gesture 模块实例（new Gesture() 的返回值） */
export interface AxGestureInstance extends AxModBase {
  options: AxGestureOptions;
  /** 事件监听（事件名见 AxGestureEventName；回调参数视事件而定） */
  on(type: AxGestureEventName, handler: (...args: any[]) => void): this;
  getStartVals(...args: any[]): any;
  init(...args: any[]): any;
  setEmpty(...args: any[]): any;
  prevEvtDft(...args: any[]): any;
  setSpy(...args: any[]): any;
  isUnbound(...args: any[]): any;
  updateEaseParamsListen(...args: any[]): any;
  updateNowTrans(...args: any[]): any;
  getMoveVals(...args: any[]): any;
  addPrimEvents(...args: any[]): any;
  removePrimEvents(...args: any[]): any;
  removeSecondEvents(...args: any[]): any;
  addSecondEvents(...args: any[]): any;
  setCompleted(...args: any[]): any;
  getOrign(...args: any[]): any;
  getScaleTrans(...args: any[]): any;
  getTouchCoords(...args: any[]): any;
  getCenterCoord(...args: any[]): any;
  getTriangleVals(...args: any[]): any;
  correctRangeTrans(...args: any[]): any;
  correctRangeRotate(...args: any[]): any;
  correctDiffRotate(...args: any[]): any;
  correctRangeScale(...args: any[]): any;
  clampVals(...args: any[]): any;
  correctWheelDirection(...args: any[]): any;
  getStepVal(...args: any[]): any;
  getViewportSizeDiff(...args: any[]): any;
  getViewRange(...args: any[]): any;
  getDmpVals(...args: any[]): any;
  getBounceVals(...args: any[]): any;
  getViewLimitVals(...args: any[]): any;
  getAutoDur(...args: any[]): any;
  rebound(...args: any[]): any;
  reset(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Hover                                                                      */
/* （无独立文档页，参数由源码默认值提取）                                                        */
/* ========================================================================== */
/** Hover 构造参数 —— new Hover(elem, options) 的 options */
export interface AxHoverOptions {
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `50` */
  timeout?: number;
  /** 默认 `50` */
  intvl?: number;
  /** 默认 `200` */
  throttle?: number;
  /** 默认 `7` */
  threshold?: number;
  hold?: string;
  /** 默认 `null` */
  onMove?: ((...args: any[]) => any);
  /** 默认 `null` */
  onEnter?: ((...args: any[]) => any);
  /** 默认 `null` */
  onLeave?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Hover 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxHoverEventName = 'constructed' | 'enter' | 'initiate' | 'initiated' | 'leave' | 'move';

/** Hover 模块实例（new Hover() 的返回值） */
export interface AxHoverInstance extends AxModBase {
  options: AxHoverOptions;
  /** 事件监听（事件名见 AxHoverEventName；回调参数视事件而定） */
  on(type: AxHoverEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Infinite · 无限加载模块                                                          */
/* 文档: modules/309.html                                                       */
/* ========================================================================== */
/** Infinite 构造参数 —— new Infinite(elem, options) 的 options */
export interface AxInfiniteOptions {
  /* ---- 基础配置 ---- */
  /** 无限滚动触发方式 默认 `'scroll'` */
  trigger?: 'scroll' | 'click' | 'clicks';
  /** 窗口样式类 */
  classes?: string;
  /** 加载器HTML 默认 `'<ax-spin></ax-spin>'` */
  spin?: string;
  /** 最大页数限制 默认 `20` */
  max?: number;
  /* ---- 数据源配置 ---- */
  /** 数据源内容 */
  content?: string;
  /** 内容格式类型 默认 `'text'` */
  contType?: 'text' | 'html' | 'node' | 'image' | 'async';
  /** 请求数据参数 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 模板配置 ---- */
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎 默认 `null` */
  tplEng?: (...args: any[]) => any;
  /* ---- 等待函数 ---- */
  /** 添加节点前等待函数 默认 `null` */
  b4Append?: (...args: any[]) => any;
  /** 移除节点前等待函数 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /** 清空节点前等待函数 默认 `null` */
  b4Clear?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 添加节点后回调 默认 `null` */
  onAppend?: (...args: any[]) => any;
  /** 加载完成回调 默认 `null` */
  onFinish?: (...args: any[]) => any;
  /** 请求错误回调 默认 `null` */
  onError?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"enable": true}` */
  spy?: AxInfiniteSpyOptions;
  /** 默认 `{"title": "", "brief": ""}` */
  media?: AxInfiniteMediaOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Infinite 的 spy 配置组 */
export interface AxInfiniteSpyOptions {
  /* ---- 监听配置 ---- */
  /** 是否启用spy侦听 默认 `true` */
  enable?: boolean;
}

/** Infinite 的 media 配置组 */
export interface AxInfiniteMediaOptions {
  /* ---- 媒体配置 ---- */
  /** 媒体文件名称 */
  caption?: string;
  /** 媒体文件简介 */
  brief?: string;
}

/** Infinite 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxInfiniteEventName = 'append' | 'constructed' | 'error' | 'finish' | 'initiate' | 'initiated';

/** Infinite 模块实例（new Infinite() 的返回值） */
export interface AxInfiniteInstance extends AxModBase {
  options: AxInfiniteOptions;
  /** 事件监听（事件名见 AxInfiniteEventName；回调参数视事件而定） */
  on(type: AxInfiniteEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setAttrs(...args: any[]): any;
  finish(...args: any[]): any;
  getListSpyIns(...args: any[]): any;
  getStatusSpyIns(...args: any[]): any;
  renderList(...args: any[]): any;
  getStatusEl(...args: any[]): any;
  getInitContent(...args: any[]): any;
  getPageData(...args: any[]): any;
  arr2Nodes(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Lazy · 懒加载模块                                                               */
/* 文档: modules/308.html                                                       */
/* ========================================================================== */
/** Lazy 构造参数 —— new Lazy(elem, options) 的 options */
export interface AxLazyOptions {
  /* ---- 基础配置 ---- */
  /** 参照父容器选择器 默认 `null` */
  root?: string | null;
  /** 懒加载类型 默认 `'src'` */
  type?: 'src' | 'async';
  /** 数据连接符 默认 `null` */
  hyphen?: string;
  /* ---- 数据源配置 ---- */
  /** 数据源内容 */
  content?: string;
  /** 内容类型 默认 `'text'` */
  contType?: 'text' | 'image' | 'video' | 'audio' | 'iframe' | 'async' | 'html' | 'form' | 'auto';
  /** 请求数据参数 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 模板配置 ---- */
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎 默认 `null` */
  tplEng?: (...args: any[]) => any;
  /* ---- 监听配置 ---- */
  /** spy模块参数 默认 `{}` */
  spy?: Record<string, any>;
  /* ---- 回调函数 ---- */
  /** 资源显示后回调 默认 `null` */
  onShown?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `'ing'` */
  trigger?: string;
  /** 默认 `null` */
  onShowing?: ((...args: any[]) => any);
  /** 默认 `null` */
  onTrigger?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Lazy 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxLazyEventName = 'constructed' | 'initiate' | 'initiated' | 'showing' | 'shown' | 'trigger';

/** Lazy 模块实例（new Lazy() 的返回值） */
export interface AxLazyInstance extends AxModBase {
  options: AxLazyOptions;
  /** 事件监听（事件名见 AxLazyEventName；回调参数视事件而定） */
  on(type: AxLazyEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Masonry · 瀑布流布局                                                            */
/* 文档: modules/233.html                                                       */
/* ========================================================================== */
/** Masonry 构造参数 —— new Masonry(elem, options) 的 options */
export interface AxMasonryOptions {
  /* ---- 布局配置 ---- */
  /** 网格间隙尺寸 默认 `0` */
  gap?: string | number;
  /** 每行列数，为0则由css控制 默认 `0` */
  cols?: number;
  /** 目标节点样式类 */
  classes?: string;
  /* ---- 数据源配置 ---- */
  /** 数据源内容 */
  content?: string;
  /** 内容类型 默认 `'text'` */
  contType?: 'text' | 'html' | 'node' | 'image' | 'async';
  /** 请求数据参数 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 模板配置 ---- */
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎 默认 `null` */
  tplEng?: (...args: any[]) => any;
  /* ---- 等待函数 ---- */
  /** 添加项目前等待函数 默认 `null` */
  b4Add?: (...args: any[]) => any;
  /** 删除项目前等待函数 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /** 清空节点前等待函数 默认 `null` */
  b4Clear?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 添加项目后回调 默认 `null` */
  onAdded?: (...args: any[]) => any;
  /** 删除项目后回调 默认 `null` */
  onRemoved?: (...args: any[]) => any;
  /** 清空项目后回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 更新内容后回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"title": "", "brief": ""}` */
  media?: AxMasonryMediaOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Masonry 的 media 配置组 */
export interface AxMasonryMediaOptions {
  /* ---- 媒体配置 ---- */
  /** 媒体文件名称 */
  caption?: string;
  /** 媒体文件简介 */
  brief?: string;
}

/** Masonry 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxMasonryEventName = 'add' | 'constructed' | 'initiate' | 'initiated' | 'remove';

/** Masonry 模块实例（new Masonry() 的返回值） */
export interface AxMasonryInstance extends AxModBase {
  options: AxMasonryOptions;
  /** 事件监听（事件名见 AxMasonryEventName；回调参数视事件而定） */
  on(type: AxMasonryEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  getGap(...args: any[]): any;
  setAttrs(...args: any[]): any;
  updateItemStyle(...args: any[]): any;
  setItem(...args: any[]): any;
  add(...args: any[]): any;
  remove(...args: any[]): any;
  clear(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Menu · 菜单                                                                  */
/* 文档: modules/141.html                                                       */
/* ========================================================================== */
/** Menu 构造参数 —— new Menu(elem, options) 的 options */
export interface AxMenuOptions {
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `'light'` */
  theme?: string;
  content?: string;
  /** 默认 `'text'` */
  contType?: string;
  /** 默认 `{}` */
  contData?: Record<string, any>;
  /** 默认 `{}` */
  ajax?: Record<string, any>;
  /** 默认 `false` */
  passive?: boolean;
  /** 默认 `false` */
  expose?: boolean;
  active?: string;
  disable?: string;
  /** 默认 `false` */
  multiple?: boolean;
  /** 默认 `false` */
  expandAll?: boolean;
  /** 默认 `0` */
  zIndex?: number;
  /** 默认 `'click'` */
  trigger?: string;
  /** 默认 `null` */
  rootStart?: any;
  /** 默认 `null` */
  idStart?: any;
  /** 默认 `null` */
  floorStart?: any;
  /** 默认 `null` */
  pathHyphen?: any;
  /** 默认 `0` */
  duration?: number;
  /** 默认 `false` */
  pageClose?: boolean;
  /** 默认 `{"enable": true, "icon": ""}` */
  arrow?: Record<string, any>;
  classes?: string;
  /** 默认 `true` */
  linkage?: boolean;
  /** 默认 `false` */
  unpadded?: boolean;
  lamp?: string;
  /** 默认 `false` */
  full?: boolean;
  /** 默认 `false` */
  spill?: boolean;
  /** 默认 `'indent|legend|icon|disk|cube|image|label|custom|tips||badge|arrow'` */
  layout?: string;
  /** 默认 `{}` */
  tree?: Record<string, any>;
  /** 默认 `{"host": "", "placement": "left", "footer": false}` */
  drawer?: Record<string, any>;
  /** 默认 `{"enable": false, "headWidth": "", "bodyWidth": "", "marginLeft": "", "marginRight": "", "align": "left"}` */
  nav?: Record<string, any>;
  /** 默认 `null` */
  onActivated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCont?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Menu 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxMenuEventName = 'activated' | 'constructed' | 'initiate' | 'initiated' | 'locked' | 'unlocked' | 'updatedCont';

/** Menu 模块实例（new Menu() 的返回值） */
export interface AxMenuInstance extends AxModBase {
  options: AxMenuOptions;
  /** 事件监听（事件名见 AxMenuEventName；回调参数视事件而定） */
  on(type: AxMenuEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  getTree(...args: any[]): any;
  setHover(...args: any[]): any;
  addNavAttr(...args: any[]): any;
  setAttrs(...args: any[]): any;
  /** 激活节点，items可以为单值也可为数组（返回 this） */
  activate(...args: any[]): any;
  updateCont(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Message · 即时消息模块                                                           */
/* 文档: modules/307.html                                                       */
/* ========================================================================== */
/** Message 构造参数 —— new Message(elem, options) 的 options */
export interface AxMessageOptions {
  /* ---- 基础显示配置 ---- */
  /** 标题文本 */
  heading?: string | boolean;
  /** 消息内容 */
  content?: string | boolean;
  /** 自定义样式类 */
  classes?: string;
  /** 弹出位置 默认 `'center-top'` */
  placement?: 'left-top' | 'center-top' | 'right-top' | 'left-center' | 'center' | 'right-center' | 'left-bottom' | 'center-bottom' | 'right-bottom';
  /** 消息层级 默认 `0` */
  zIndex?: number;
  /* ---- 状态配置 ---- */
  /** 消息状态类型 默认 `'info'` */
  status?: 'succ' | 'error' | 'issue' | 'info' | 'warn';
  /** 是否显示状态图标 默认 `false` */
  iconShow?: boolean;
  /** 是否使用突出风格 默认 `false` */
  notable?: boolean;
  /* ---- 行为配置 ---- */
  /** 自动关闭延时(ms) 默认 `3000` */
  delay?: number;
  /** 是否显示倒计时进度条 默认 `true` */
  progress?: boolean;
  /** 是否显示关闭按钮 默认 `true` */
  closable?: boolean;
  /** 是否手动关闭 默认 `false` */
  manual?: boolean;
  /** 是否立即展示 默认 `false` */
  eager?: boolean;
  /* ---- 回调函数 ---- */
  /** 消息显示后回调 默认 `null` */
  onShown?: (...args: any[]) => any;
  /** 消息关闭后回调 默认 `null` */
  onHidden?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Message 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxMessageEventName = 'constructed' | 'hidden' | 'initiate' | 'initiated' | 'shown';

/** Message 模块实例（new Message() 的返回值） */
export interface AxMessageInstance extends AxModBase {
  options: AxMessageOptions;
  /** 事件监听（事件名见 AxMessageEventName；回调参数视事件而定） */
  on(type: AxMessageEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  createSection(...args: any[]): any;
  getCloseBtns(...args: any[]): any;
  show(...args: any[]): any;
  hide(...args: any[]): any;
  updateCont(...args: any[]): any;
  update(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* More                                                                       */
/* （无独立文档页，参数由源码默认值提取）                                                        */
/* ========================================================================== */
/** More 构造参数 —— new More(elem, options) 的 options */
export interface AxMoreOptions {
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `[]` */
  storKeys?: any[];
  content?: string;
  /** 默认 `60` */
  length?: number;
  /** 默认 `true` */
  folded?: boolean;
  /** 默认 `null` */
  textFold?: any;
  /** 默认 `null` */
  textUnfold?: any;
  /** 默认 `'c-prim'` */
  classFold?: string;
  /** 默认 `'c-prim'` */
  classUnfold?: string;
  /** 默认 `null` */
  onFold?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUnfold?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCont?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** More 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxMoreEventName = 'constructed' | 'fold' | 'initiate' | 'initiated' | 'unfold';

/** More 模块实例（new More() 的返回值） */
export interface AxMoreInstance extends AxModBase {
  options: AxMoreOptions;
  /** 事件监听（事件名见 AxMoreEventName；回调参数视事件而定） */
  on(type: AxMoreEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setAttrs(...args: any[]): any;
  toggle(...args: any[]): any;
  getPureContent(...args: any[]): any;
  destroy(...args: any[]): any;
  updateCont(...args: any[]): any;
}

/* ========================================================================== */
/* Pagination · 分页模块                                                          */
/* 文档: modules/378.html                                                       */
/* ========================================================================== */
/** Pagination 构造参数 —— new Pagination(elem, options) 的 options */
export interface AxPaginationOptions {
  /* ---- 基础配置 ---- */
  /** 列表容器选择器 */
  listSel?: string;
  /** 每页显示的条目数 默认 `10` */
  count?: number;
  /** 可选的每页条目数，如：'10,20,30' */
  counts?: string;
  /** 当前页码 默认 `1` */
  current?: number;
  /** 分页器尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /* ---- 显示控制 ---- */
  /** 是否展开显示所有页码 默认 `false` */
  expanded?: boolean;
  /** 可见页码配置 默认 `{"edge": 2, "center": 2}` */
  visible?: Record<string, any>;
  /** 分页器对齐方式 */
  align?: string;
  /** 分页器样式类型 */
  type?: string;
  /** 是否启用弹性布局 默认 `false` */
  flexible?: boolean;
  /** 分页器结构布局 默认 `'first|prev|pages|next|last'` */
  layout?: string;
  /** 自定义 CSS 类名 */
  classes?: string;
  /** 跳转防抖延迟(ms) 默认 `100` */
  delay?: number;
  /* ---- 内容配置 ---- */
  /** 数据源，默认自带1000条数据 默认 `1000` */
  content?: number | string | any[];
  /** 内容类型 */
  contType?: string;
  /** 请求参数配置 默认 `{}` */
  contData?: Record<string, any>;
  /** AJAX 请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎配置 默认 `null` */
  tplEng?: (...args: any[]) => any | null;
  /* ---- DOM 结构 ---- */
  /** 分页器各部分节点名称配置 默认 `{"main": "div", "total": "span", "count": "span", "locate": "span", "pages": "span", "tips": "span", "ellipsis": "span", "prev": "a", "next": "a", "first": "a", "last": "a", "item": "a", "section": "section"}` */
  names?: Record<string, any>;
  /* ---- 等待函数 ---- */
  /** 页码跳转前等待 默认 `null` */
  b4Locate?: (...args: any[]) => any | null;
  /** 列表渲染前等待 默认 `null` */
  b4RenderList?: (...args: any[]) => any | null;
  /* ---- 事件回调 ---- */
  /** 页码跳转完成回调 默认 `null` */
  onLocated?: (...args: any[]) => any | null;
  /** 内容获取完成回调 默认 `null` */
  onGotCont?: (...args: any[]) => any | null;
  /** 跳转首页回调 默认 `null` */
  onToFirst?: (...args: any[]) => any | null;
  /** 跳转尾页回调 默认 `null` */
  onToLast?: (...args: any[]) => any | null;
  /** 页码超出范围回调 默认 `null` */
  onExceeded?: (...args: any[]) => any | null;
  /** 列表渲染完成回调 默认 `null` */
  onRenderedList?: (...args: any[]) => any | null;
  /** 页码渲染完成回调 默认 `null` */
  onRenderedPages?: (...args: any[]) => any | null;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{}` */
  dropdown?: Record<string, any>;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Pagination 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxPaginationEventName = 'constructed' | 'error' | 'exceeded' | 'gotCont' | 'initiate' | 'initiated' | 'located' | 'renderedList' | 'renderedPages' | 'toFirst' | 'toLast';

/** Pagination 模块实例（new Pagination() 的返回值） */
export interface AxPaginationInstance extends AxModBase {
  options: AxPaginationOptions;
  /** 事件监听（事件名见 AxPaginationEventName；回调参数视事件而定） */
  on(type: AxPaginationEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setOutputObs(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getNodesTree(...args: any[]): any;
  getInitData(...args: any[]): any;
  getData(...args: any[]): any;
  isSqlReq(...args: any[]): any;
  sliceData(...args: any[]): any;
  getCurCont(...args: any[]): any;
  getItemNode(...args: any[]): any;
  renderTotalEl(...args: any[]): any;
  renderTipsEl(...args: any[]): any;
  renderLocateEl(...args: any[]): any;
  renderCountEl(...args: any[]): any;
  renderPagesEl(...args: any[]): any;
  createItem(...args: any[]): any;
  hasEll(...args: any[]): any;
  correctCur(...args: any[]): any;
  updateCur(...args: any[]): any;
  rerenderListEl(...args: any[]): any;
  toggleSelected(...args: any[]): any;
  locate(...args: any[]): any;
  prev(...args: any[]): any;
  next(...args: any[]): any;
  first(...args: any[]): any;
  last(...args: any[]): any;
  setDisabled(...args: any[]): any;
  setEnabled(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Panel · 面板模块                                                               */
/* 文档: modules/354.html                                                       */
/* ========================================================================== */
/** Panel 构造参数 —— new Panel(elem, options) 的 options */
export interface AxPanelOptions {
  /* ---- 内容参数 ---- */
  /** 数据源，支持数组或链接字符串 */
  content?: string | any[];
  /** 内容类型 默认 `'text'` */
  contType?: string;
  /** 请求数据或媒体节点属性配置 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求的ajax配置参数 默认 `{}` */
  ajax?: Record<string, any>;
  /** 模板字符串 */
  tplStr?: string;
  /** 第三方模板引擎 默认 `null` */
  tplEng?: any;
  /* ---- 显示参数 ---- */
  /** 面板标题 */
  label?: string;
  /** 面板提示文本 */
  tips?: string;
  /** 方形指示图片URL */
  cube?: string;
  /** 圆形指示图片URL */
  disk?: string;
  /** 普通指示图片URL */
  image?: string;
  /** 指示图标 */
  icon?: string;
  /** 徽章内容 默认 `null` */
  badge?: string | null;
  /** 标题辅助说明 */
  annot?: string;
  /** 指示灯位置 */
  lamp?: 'left' | 'right' | 'top' | 'bottom';
  /** 用户自定义内容 */
  custom?: string | HTMLElement;
  /** 面板简介 */
  brief?: string;
  /** 目标节点的class类名 */
  classes?: string;
  /** 主题风格 */
  theme?: string;
  /* ---- 功能配置 ---- */
  /** 指示箭头配置 默认 `{"enable": false, "show": "icon-up", "hide": "icon-up", "anim": "rotate180", "type": "icon", "placement": "end"}` */
  arrow?: Record<string, any>;
  /** Tab模块配置 默认 `{"enable": true}` */
  tab?: Record<string, any>;
  /** 工具箱配置 默认 `{"enable": false, "children": ["toggle"]}` */
  tools?: Record<string, any>;
  /** 是否延迟渲染内容 默认 `false` */
  deferred?: boolean;
  /** 固定结构和风格 */
  feature?: string;
  /** 是否使用分割线 默认 `false` */
  dividable?: boolean;
  /** 是否初始折叠 默认 `false` */
  collapsed?: boolean;
  /** 是否可点击头部切换 默认 `false` */
  headClosable?: boolean;
  /** 是否使用内边距 默认 `true` */
  padded?: boolean;
  /* ---- 等待函数 ---- */
  /** 填充前等待函数 默认 `null` */
  b4Fill?: (...args: any[]) => any;
  /** 展开前等待函数 默认 `null` */
  b4Expand?: (...args: any[]) => any;
  /** 折叠前等待函数 默认 `null` */
  b4Collapse?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 填充后回调函数 默认 `null` */
  onFilled?: (...args: any[]) => any;
  /** 渲染后回调函数 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 展开前回调函数 默认 `null` */
  onExpand?: (...args: any[]) => any;
  /** 展开后回调函数 默认 `null` */
  onExpanded?: (...args: any[]) => any;
  /** 折叠前回调函数 默认 `null` */
  onCollapse?: (...args: any[]) => any;
  /** 折叠后回调函数 默认 `null` */
  onCollapsed?: (...args: any[]) => any;
  /** 内容更新后回调函数 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /** 异步请求完成回调函数 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Panel 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxPanelEventName = 'collapse' | 'collapsed' | 'constructed' | 'expand' | 'expanded' | 'filled' | 'initiate' | 'initiated';

/** Panel 模块实例（new Panel() 的返回值） */
export interface AxPanelInstance extends AxModBase {
  options: AxPanelOptions;
  /** 事件监听（事件名见 AxPanelEventName；回调参数视事件而定） */
  on(type: AxPanelEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  correctOpts(...args: any[]): any;
  fillBody(...args: any[]): any;
  getDataToRender(...args: any[]): any;
  toggleArrow(...args: any[]): any;
  setArrow(...args: any[]): any;
  createHeadEl(...args: any[]): any;
  createBodyEl(...args: any[]): any;
  renderMain(...args: any[]): any;
  setAttrs(...args: any[]): any;
  renderFinish(...args: any[]): any;
  expand(...args: any[]): any;
  collapse(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Popup · 气泡模块                                                               */
/* 文档: modules/303.html                                                       */
/* ========================================================================== */
/** Popup 构造参数 —— new Popup(elem, options) 的 options */
export interface AxPopupOptions {
  /* ---- 基础配置参数 ---- */
  /** 窗口标题 */
  heading?: string;
  /** 是否显示标题分割线 默认 `false` */
  divider?: boolean;
  /** 动画时长(ms) 默认 `0` */
  duration?: number;
  /** 是否自动计算动画时间 默认 `false` */
  autoDur?: boolean;
  /** 是否禁用show方法 默认 `false` */
  asleep?: boolean;
  /** 打开后是否禁止关闭 默认 `false` */
  deadShow?: boolean;
  /** 是否只允许显示不允许关闭 默认 `false` */
  keepShow?: boolean;
  /** 窗口样式类 */
  classes?: string;
  /** 窗口层级 默认 `0` */
  zIndex?: number;
  /** 是否阻止与body联动滚动 默认 `false` */
  dedicated?: boolean;
  /* ---- Popup专属配置 ---- */
  /** 主题颜色 */
  theme?: string;
  /** 窗口显示位置 默认 `'top'` */
  placement?: 'top-start' | 'top-center' | 'top-end' | 'bottom-start' | 'bottom-center' | 'bottom-end' | 'left-start' | 'left-center' | 'left-end' | 'right-start' | 'right-center' | 'right-end' | 'top-max' | 'bottom-max' | 'left-max' | 'right-max' | 'center-max' | 'center' | 'left' | 'right' | 'top' | 'bottom';
  /** 窗口尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg' | 'xl' | 'xxl';
  /** 窗口内容高度 */
  height?: string;
  /** 显示触发方式 默认 `'hover'` */
  trigger?: 'hover' | 'click' | 'sticky' | 'input' | 'none';
  /** 点击外部是否关闭 默认 `true` */
  pageClose?: boolean;
  /** 是否允许多窗口 默认 `false` */
  multiple?: boolean;
  /** 是否自适应位置 默认 `true` */
  adaptive?: boolean;
  /** 定位引擎参数 默认 `{}` */
  position?: Record<string, any>;
  /** 打开动画类型 默认 `'fadeIn'` */
  in?: 'fadeIn' | 'fadeInDown' | 'fadeInUp' | 'slideDown' | 'fadeInStart' | 'fadeInEnd' | 'springIn' | 'scaleIn' | 'easeIn';
  /** 关闭动画类型 默认 `'fadeOut'` */
  out?: 'fadeOut' | 'fadeOutDown' | 'fadeOutUp' | 'slideUp' | 'fadeOutStart' | 'fadeOutEnd' | 'springOut' | 'scaleOut' | 'easeOut';
  /** 窗口位置更新回调 默认 `null` */
  onReposition?: (...args: any[]) => any;
  /* ---- 内容配置 ---- */
  /** 数据源 */
  content?: any;
  /** 内容类型 默认 `'text'` */
  contType?: 'text' | 'html' | 'image' | 'video' | 'audio' | 'iframe' | 'async' | 'form' | 'input' | 'textarea' | 'auto';
  /** 请求数据或媒体属性 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求参数 默认 `{}` */
  ajax?: Record<string, any>;
  /** 打开时目标节点添加的类名 默认 `null` */
  actClass?: string;
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎 默认 `null` */
  tplEng?: (...args: any[]) => any;
  /** 点击触发判断函数 默认 `null` */
  canClick?: (...args: any[]) => any;
  /* ---- 等待函数 ---- */
  /** 显示前等待函数 默认 `null` */
  b4Show?: (...args: any[]) => any;
  /** 隐藏前等待函数 默认 `null` */
  b4Hide?: (...args: any[]) => any;
  /** 填充前等待函数 默认 `null` */
  b4Fill?: (...args: any[]) => any;
  /** 确认前等待函数 默认 `null` */
  b4Confirm?: (...args: any[]) => any;
  /** 取消前等待函数 默认 `null` */
  b4Cancel?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 显示前回调 默认 `null` */
  onShow?: (...args: any[]) => any;
  /** 隐藏前回调 默认 `null` */
  onHide?: (...args: any[]) => any;
  /** 显示后回调 默认 `null` */
  onShown?: (...args: any[]) => any;
  /** 隐藏后回调 默认 `null` */
  onHidden?: (...args: any[]) => any;
  /** 表单验证失败回调 默认 `null` */
  onInvalidated?: (...args: any[]) => any;
  /** 表单验证成功回调 默认 `null` */
  onValidated?: (...args: any[]) => any;
  /** 表单发送成功回调 默认 `null` */
  onSended?: (...args: any[]) => any;
  /** 渲染开始回调 默认 `null` */
  onRender?: (...args: any[]) => any;
  /** 渲染完成回调 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 内容更新回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /** 宿主节点更新回调 默认 `null` */
  onTransferred?: (...args: any[]) => any;
  /** 确认按钮点击回调 默认 `null` */
  onConfirmed?: (...args: any[]) => any;
  /** 取消按钮点击回调 默认 `null` */
  onCanceled?: (...args: any[]) => any;
  /** 关闭按钮点击回调 默认 `null` */
  onClosed?: (...args: any[]) => any;
  /** 清除按钮点击回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 异步请求完成回调 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /** 列表checked回调 默认 `null` */
  onBulletsChecked?: (...args: any[]) => any;
  /** 列表selected回调 默认 `null` */
  onBulletsSelected?: (...args: any[]) => any;
  /** 目标节点填充回调 默认 `null` */
  onTargetSet?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `null` */
  canClose?: any;
  /** 默认 `{"enable": true, "size": "0.5rem", "offset": "2.8rem", "gap": ".8rem"}` */
  arrow?: AxPopupArrowOptions;
  /** 默认 `{"selector": "", "trigger": "click", "actClass": ""}` */
  wing?: AxPopupWingOptions;
  /** 默认 `{"enable": true, "body": true, "foot": true}` */
  padding?: AxPopupPaddingOptions;
  /** 默认 `{"title": "", "brief": ""}` */
  media?: AxPopupMediaOptions;
  /** 默认 `{"enable": false, "inputSel": "", "childSel": "", "detectable": false, "attr": ""}` */
  autoFill?: AxPopupAutofillOptions;
  /** 默认 `{"enable": false, "type": "custom", "parentNode": "ul", "childNode": "li", "unpadded": false, "hoverable": false, "divisible": false, "multiline": false, "cols": 0, "lines": "", "action": null}` */
  bullet?: AxPopupBulletOptions;
  /** 默认 `{"enable": true, "placement": "inside", "children": ["close"]}` */
  tools?: AxPopupToolsOptions;
  /** 默认 `{"enable": true, "layout": "plain", "note": false, "divider": false, "padding": false, "children": ["cancel", "confirm"]}` */
  footer?: AxPopupFooterOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Popup 的 arrow 配置组 */
export interface AxPopupArrowOptions {
  /* ---- Popup专属配置 ---- */
  /** 是否显示箭头 默认 `true` */
  enable?: boolean;
  /** 箭头尺寸 默认 `'0.5rem'` */
  size?: string;
  /** 箭头边距 默认 `'2.8rem'` */
  offset?: string;
  /** 窗口与按钮距离 默认 `'.8rem'` */
  gap?: string;
}

/** Popup 的 wing 配置组 */
export interface AxPopupWingOptions {
  /* ---- Popup专属配置 ---- */
  /** 触发元素选择器 */
  selector?: string | any[];
  /** 关联按钮触发方式 默认 `'click'` */
  trigger?: 'click' | 'hover' | 'sticky' | 'input' | 'none';
  /** 触发按钮激活样式类 */
  actClass?: string;
}

/** Popup 的 padding 配置组 */
export interface AxPopupPaddingOptions {
  /* ---- 内边距配置 ---- */
  /** 是否显示内边距 默认 `true` */
  enable?: boolean;
  /** 内容容器的内边距类名，true即添加_p 默认 `true` */
  body?: string | boolean;
  /** 底部容器的内边距类名，true即添加_p 默认 `true` */
  foot?: string | boolean;
}

/** Popup 的 media 配置组 */
export interface AxPopupMediaOptions {
  /* ---- 媒体配置 ---- */
  /** 媒体标题 */
  caption?: string;
  /** 媒体简介 */
  brief?: string;
}

/** Popup 的 autoFill 配置组 */
export interface AxPopupAutofillOptions {
  /* ---- 自动填充配置 ---- */
  /** 是否自动填充 默认 `false` */
  enable?: boolean;
  /** 填充节点选择器 */
  inputSel?: string;
  /** 子节点选择器 */
  childSel?: string;
  /** 是否允许反向取值 默认 `false` */
  detectable?: boolean;
  /** 填充节点属性 */
  attr?: string;
}

/** Popup 的 bullet 配置组 */
export interface AxPopupBulletOptions {
  /* ---- 列表配置 ---- */
  /** 是否启用列表模式 默认 `false` */
  enable?: boolean;
  /** 列表类型 默认 `'custom'` */
  type?: 'checkboxes' | 'radios' | 'select-single' | 'select-multi' | 'custom';
  /** 列表父节点名 默认 `'ul'` */
  parentNode?: string;
  /** 列表子节点名 默认 `'li'` */
  childNode?: string;
  /** 是否无内间距 默认 `false` */
  unpadded?: boolean;
  /** 是否支持悬停效果 默认 `false` */
  hoverable?: boolean;
  /** 是否显示分割线 默认 `false` */
  divisible?: boolean;
  /** 是否多行显示 默认 `false` */
  multiline?: boolean;
  /** 网格列数 默认 `0` */
  cols?: number;
  /** 项目间隔类型 */
  lines?: any;
  /** 列表项处理函数 默认 `null` */
  action?: (...args: any[]) => any;
}

/** Popup 的 tools 配置组 */
export interface AxPopupToolsOptions {
  /* ---- 工具栏配置 ---- */
  /** 是否显示工具栏 默认 `true` */
  enable?: boolean;
  /** 工具栏位置 默认 `'inside'` */
  placement?: 'inside' | 'outside';
  /** 工具按钮列表 默认 `['close']` */
  children?: any[];
}

/** Popup 的 footer 配置组 */
export interface AxPopupFooterOptions {
  /* ---- 底部配置 ---- */
  /** 是否显示底部 默认 `true` */
  enable?: boolean;
  /** 按钮布局方式 默认 `'plain'` */
  layout?: 'center' | 'plain' | 'right' | 'left';
  /** 是否显示底部提示 默认 `false` */
  note?: boolean;
  /** 是否显示底部分割线 默认 `false` */
  divider?: boolean;
  /** 是否显示内边距 默认 `false` */
  padding?: boolean;
  /** 底部按钮列表 默认 `['cancel', 'confirm']` */
  children?: any[];
}

/** Popup 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxPopupEventName = 'constructed' | 'hidden' | 'hide' | 'initiate' | 'initiated' | 'reposition' | 'show' | 'shown';

/** Popup 模块实例（new Popup() 的返回值） */
export interface AxPopupInstance extends AxModBase {
  options: AxPopupOptions;
  /** 事件监听（事件名见 AxPopupEventName；回调参数视事件而定） */
  on(type: AxPopupEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  fixAni(...args: any[]): any;
  handleTools(...args: any[]): any;
  setAttrs(...args: any[]): any;
  createPosition(...args: any[]): any;
  bindTrigger(...args: any[]): any;
  unbindTrigger(...args: any[]): any;
  fixEaseAni(...args: any[]): any;
  reposition(...args: any[]): any;
  show(...args: any[]): any;
  hide(...args: any[]): any;
}

/* ========================================================================== */
/* Progress · 进度条模块                                                           */
/* 文档: modules/301.html                                                       */
/* ========================================================================== */
/** Progress 构造参数 —— new Progress(elem, options) 的 options */
export interface AxProgressOptions {
  /* ---- 基础外观配置 ---- */
  /** 进度条类型 默认 `'line'` */
  type?: 'line' | 'circle' | 'semicircle' | 'gapcircle';
  /** 颜色风格 */
  theme?: string;
  /** 是否使用渐变色 默认 `false` */
  gradient?: boolean;
  /** 线条端点形状 默认 `'round'` */
  linecap?: 'round' | 'square';
  /** 进度条宽度 */
  size?: string;
  /** 进度条厚度 */
  thk?: string;
  /** 追加样式类 */
  classes?: string;
  /* ---- 取值配置 ---- */
  /** 取值范围 默认 `[0, 100]` */
  range?: any[];
  /** 初始进度值 默认 `0` */
  value?: number;
  /** 进度阶段设置 默认 `[]` */
  steps?: any[];
  /* ---- 动画配置 ---- */
  /** 动画曲线 默认 `'easeOut'` */
  curve?: 'easeOut' | 'linear' | 'easeIn' | 'easeInOut' | 'easeOutIn';
  /** 动画时长(ms) 默认 `0` */
  duration?: number;
  /** 是否旋转 默认 `false` */
  rotate?: boolean;
  /* ---- 显示控制 ---- */
  /** 是否隐藏轨道 默认 `false` */
  simple?: boolean;
  /** 是否显示控制按钮 默认 `false` */
  control?: boolean;
  /* ---- 等待函数 ---- */
  /** 设置值前等待函数 默认 `null` */
  b4Set?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 进程开始回调 默认 `null` */
  onStart?: (...args: any[]) => any;
  /** 进程进行中回调 默认 `null` */
  onMove?: (...args: any[]) => any;
  /** 进程结束回调 默认 `null` */
  onEnd?: (...args: any[]) => any;
  /** 进度为0%回调 默认 `null` */
  onZero?: (...args: any[]) => any;
  /** 进度完成回调 默认 `null` */
  onCompleted?: (...args: any[]) => any;
  /** 设置值后回调 默认 `null` */
  onSet?: (...args: any[]) => any;
  /** 暂停后回调 默认 `null` */
  onPause?: (...args: any[]) => any;
  /** 继续进行回调 默认 `null` */
  onContinue?: (...args: any[]) => any;
  /** 恢复初始值回调 默认 `null` */
  onRestored?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"enable": true, "template": "", "placement": "", "unit": "%", "tips": false}` */
  label?: AxProgressLabelOptions;
  /** 默认 `{"type": "percent", "trunc": true, "emphatic": false}` */
  number?: AxProgressNumberOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Progress 的 label 配置组 */
export interface AxProgressLabelOptions {
  /* ---- 标签配置 ---- */
  /** 是否显示文本 默认 `true` */
  enable?: boolean;
  /** 文本HTML模板 */
  template?: string;
  /** 文本位置 */
  placement?: string;
  /** 单位 默认 `'%'` */
  unit?: string;
  /** 是否显示提示 默认 `false` */
  tips?: boolean;
}

/** Progress 的 number 配置组 */
export interface AxProgressNumberOptions {
  /* ---- 数字显示配置 ---- */
  /** 显示类型 默认 `'percent'` */
  type?: 'percent' | 'value';
  /** 是否取整 默认 `true` */
  trunc?: boolean;
  /** 是否强调显示 默认 `false` */
  emphatic?: boolean;
}

/** Progress 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxProgressEventName = 'completed' | 'constructed' | 'continue' | 'end' | 'initiate' | 'initiated' | 'move' | 'pause' | 'restored' | 'set' | 'start' | 'zero';

/** Progress 模块实例（new Progress() 的返回值） */
export interface AxProgressInstance extends AxModBase {
  options: AxProgressOptions;
  /** 事件监听（事件名见 AxProgressEventName；回调参数视事件而定） */
  on(type: AxProgressEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setObs(...args: any[]): any;
  setControl(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getInitVals(...args: any[]): any;
  getVal(...args: any[]): any;
  getBarVal(...args: any[]): any;
  getRatio(...args: any[]): any;
  getNumber(...args: any[]): any;
  getVals(...args: any[]): any;
  setAnim(...args: any[]): any;
  watchEnds(...args: any[]): any;
  watchSteps(...args: any[]): any;
  getStepVals(...args: any[]): any;
  animateTo(...args: any[]): any;
  locateTo(...args: any[]): any;
  restore(...args: any[]): any;
  setCompleted(...args: any[]): any;
  setZero(...args: any[]): any;
  continue(...args: any[]): any;
  pause(...args: any[]): any;
  get(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Range · 范围输入模块                                                             */
/* 文档: modules/295.html                                                       */
/* ========================================================================== */
/** Range 构造参数 —— new Range(elem, options) 的 options */
export interface AxRangeOptions {
  /* ---- 基础配置 ---- */
  /** 隐藏input的name属性 */
  name?: string;
  /** 滑块步进值 默认 `1` */
  step?: number;
  /** 最大值 默认 `100` */
  max?: number;
  /** 最小值 默认 `0` */
  min?: number;
  /** 初始值 默认 `0` */
  value?: number | any[];
  /** 滑动方向 默认 `'h'` */
  flow?: 'h' | 'v';
  /** 是否禁用 默认 `false` */
  disabled?: boolean;
  /* ---- 显示配置 ---- */
  /** 滑块尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 是否横向铺满 默认 `false` */
  full?: boolean;
  /** 追加样式类 */
  classes?: string;
  /** 是否显示极限值气泡 默认 `true` */
  limitShow?: boolean;
  /** 是否显示滑块气泡 默认 `true` */
  tipShow?: boolean;
  /* ---- 多滑块配置 ---- */
  /** 是否启用双滑块 默认 `false` */
  multiple?: boolean;
  /** 是否锁定滑块间距 默认 `false` */
  locked?: boolean;
  /** input值分隔符 默认 `null` */
  separator?: string;
  /** 气泡值连接符 默认 `null` */
  hyphen?: string;
  /* ---- 按钮配置 ---- */
  /** 是否支持键盘操作 默认 `false` */
  keyboard?: boolean;
  /* ---- 回调函数 ---- */
  /** 设置新值后回调 默认 `null` */
  onSet?: (...args: any[]) => any;
  /** 到达最小值回调 默认 `null` */
  onToStart?: (...args: any[]) => any;
  /** 到达最大值回调 默认 `null` */
  onToEnd?: (...args: any[]) => any;
  /** 启用后回调 默认 `null` */
  onEnabled?: (...args: any[]) => any;
  /** 禁用后回调 默认 `null` */
  onDisabled?: (...args: any[]) => any;
  /** 恢复初始值回调 默认 `null` */
  onRestored?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{"enable": false, "min": 0, "max": 100}` */
  fence?: AxRangeFenceOptions;
  /** 默认 `{"enable": false, "decrease": "<i class=\"icon-minus-o-f\"></i>", "increase": "<i class=\"icon-plus-o-f\"></i>"}` */
  button?: AxRangeButtonOptions;
  /** 默认 `{"enable": false, "majorEqual": 4, "minorEqual": "auto", "labels": []}` */
  ruler?: AxRangeRulerOptions;
  /** 默认 `{"enable": false, "target": null}` */
  result?: AxRangeResultOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Range 的 fence 配置组 */
export interface AxRangeFenceOptions {
  /* ---- 范围配置 ---- */
  /** 是否启用有效范围 默认 `false` */
  enable?: boolean;
  /** 有效范围最小值 默认 `0` */
  min?: number;
  /** 有效范围最大值 默认 `100` */
  max?: number;
}

/** Range 的 button 配置组 */
export interface AxRangeButtonOptions {
  /* ---- 按钮配置 ---- */
  /** 是否启用增减按钮 默认 `false` */
  enable?: boolean;
  /** 减少按钮HTML */
  decrease?: string;
  /** 增加按钮HTML */
  increase?: string;
}

/** Range 的 ruler 配置组 */
export interface AxRangeRulerOptions {
  /* ---- 标尺配置 ---- */
  /** 是否显示标尺 默认 `false` */
  enable?: boolean;
  /** 大刻度等分数 默认 `4` */
  majorEqual?: number;
  /** 小刻度等分数 默认 `'auto'` */
  minorEqual?: number;
  /** 大刻度标签 默认 `[]` */
  labels?: any[];
}

/** Range 的 result 配置组 */
export interface AxRangeResultOptions {
  /* ---- 结果显示配置 ---- */
  /** 是否显示结果 默认 `false` */
  enable?: boolean;
  /** 结果显示容器选择器 默认 `null` */
  target?: string | null;
}

/** Range 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxRangeEventName = 'constructed' | 'initiate' | 'initiated' | 'output' | 'restored' | 'set' | 'toEnd' | 'toStart';

/** Range 模块实例（new Range() 的返回值） */
export interface AxRangeInstance extends AxModBase {
  options: AxRangeOptions;
  /** 事件监听（事件名见 AxRangeEventName；回调参数视事件而定） */
  on(type: AxRangeEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  correctOpts(...args: any[]): any;
  updateSizes(...args: any[]): any;
  setHandleIns(...args: any[]): any;
  setHandleActive(...args: any[]): any;
  getRatioFromVal(...args: any[]): any;
  getValFromRatio(...args: any[]): any;
  decrease(...args: any[]): any;
  increase(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getInitVal(...args: any[]): any;
  getPlaces(...args: any[]): any;
  correctVal(...args: any[]): any;
  initRender(...args: any[]): any;
  setVals(...args: any[]): any;
  endsListen(...args: any[]): any;
  getVals(...args: any[]): any;
  toEnd(...args: any[]): any;
  toStart(...args: any[]): any;
  slideSet(...args: any[]): any;
  mergeTips(...args: any[]): any;
  hideEnds(...args: any[]): any;
  setTicks(...args: any[]): any;
  setFence(...args: any[]): any;
  setMinorTicks(...args: any[]): any;
  disable(...args: any[]): any;
  enable(...args: any[]): any;
  clear(...args: any[]): any;
  restore(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Rate · 星级评分模块                                                              */
/* 文档: modules/291.html                                                       */
/* ========================================================================== */
/** Rate 构造参数 —— new Rate(elem, options) 的 options */
export interface AxRateOptions {
  /* ---- 基础配置 ---- */
  /** 隐藏input的name属性 */
  name?: string;
  /** 初始评分值 默认 `0` */
  value?: number;
  /** 填充值类型 默认 `'value'` */
  fill?: 'value' | 'stars';
  /** 左侧主标题 */
  heading?: string;
  /** 是否禁用 默认 `false` */
  disabled?: boolean;
  /** 是否只读 默认 `false` */
  readonly?: boolean;
  /** 追加样式类 */
  classes?: string;
  /* ---- 评分配置 ---- */
  /** 是否支持半星 默认 `false` */
  half?: boolean;
  /** 评星总数 默认 `5` */
  count?: number;
  /** 单星代表分值 默认 `1` */
  increment?: number;
  /** 是否可清除 默认 `false` */
  clearable?: boolean;
  /** 是否只点亮单个 默认 `false` */
  highlight?: boolean;
  /** 是否忽略灰色星 默认 `false` */
  omitted?: boolean;
  /* ---- 外观配置 ---- */
  /** 星星尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 星星图标类名 默认 `'icon-star-f'` */
  star?: string;
  /** 星星图标类型 默认 `'icon'` */
  type?: 'icon' | 'image';
  /** 星星对照地图 默认 `[]` */
  map?: any[];
  /* ---- 回调函数 ---- */
  /** 设置新值后回调 默认 `null` */
  onSet?: (...args: any[]) => any;
  /** 值变化后回调 默认 `null` */
  onChanged?: (...args: any[]) => any;
  /** 恢复初始值回调 默认 `null` */
  onRestored?: (...args: any[]) => any;
  /** 清除归零后回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"enable": false}` */
  tooltip?: AxRateTooltipOptions;
  /** 默认 `{"enable": false}` */
  result?: AxRateResultOptions;
  /** 默认 `{"enable": false, "format": ""}` */
  label?: AxRateLabelOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Rate 的 tooltip 配置组 */
export interface AxRateTooltipOptions {
  /* ---- 提示配置 ---- */
  /** 是否显示气泡 默认 `false` */
  enable?: boolean;
  /** 气泡内容格式 默认 `'{{this.stars}}星,评分:{{this.value}}'` */
  format?: string;
}

/** Rate 的 result 配置组 */
export interface AxRateResultOptions {
  /* ---- 提示配置 ---- */
  /** 是否显示得分 默认 `false` */
  enable?: boolean;
  /** 得分文本格式 默认 `'{{this.stars}}星'` */
  format?: string;
}

/** Rate 的 label 配置组 */
export interface AxRateLabelOptions {
  /* ---- 标签配置 ---- */
  /** 是否显示标签 默认 `false` */
  enable?: boolean;
  /** 标签内容格式 */
  format?: string;
}

/** Rate 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxRateEventName = 'changed' | 'constructed' | 'initiate' | 'initiated' | 'restored' | 'set';

/** Rate 模块实例（new Rate() 的返回值） */
export interface AxRateInstance extends AxModBase {
  options: AxRateOptions;
  /** 事件监听（事件名见 AxRateEventName；回调参数视事件而定） */
  on(type: AxRateEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  createOutputIns(...args: any[]): any;
  renderFinish(...args: any[]): any;
  renderFormat(...args: any[]): any;
  findLegend(...args: any[]): any;
  getStarPoint(...args: any[]): any;
  createTooltip(...args: any[]): any;
  getStarsHtml(...args: any[]): any;
  renderMain(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getVals(...args: any[]): any;
  correctVal(...args: any[]): any;
  getFloatStar(...args: any[]): any;
  toggleSelected(...args: any[]): any;
  updateStarsState(...args: any[]): any;
  setVals(...args: any[]): any;
  restore(...args: any[]): any;
  clear(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Retrieval · 检索模块                                                           */
/* 文档: modules/288.html                                                       */
/* ========================================================================== */
/** Retrieval 构造参数 —— new Retrieval(elem, options) 的 options */
export interface AxRetrievalOptions {
  /* ---- 检索基础配置 ---- */
  /** 检索关键字 */
  keys?: string;
  /** 检索属性 默认 `'label'` */
  props?: string;
  /** 是否模糊检索 默认 `true` */
  fuzzy?: boolean;
  /** 结果排序规则 */
  order?: string;
  /** 结果数量限制 默认 `0` */
  limit?: number;
  /** 是否忽略大小写 默认 `true` */
  ignore?: boolean;
  /** 分隔符 默认 `null` */
  hyphen?: string;
  /** 空值处理方式 默认 `'reset'` */
  nullHandle?: 'reset' | 'clear';
  /* ---- 数据源配置 ---- */
  /** 数据源内容 */
  content?: string;
  /** 内容类型 默认 `'text'` */
  contType?: 'text' | 'async' | 'html' | 'auto';
  /** 请求数据参数 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 编辑器配置 ---- */
  /** 编辑器选择器 */
  editor?: string;
  /** 触发事件类型 默认 `'input'` */
  trigger?: 'input' | 'change' | 'blur';
  /** 输入节流时间(ms) 默认 `200` */
  delay?: number;
  /** 列表容器选择器 */
  target?: string;
  /* ---- 列表配置 ---- */
  /** 列表样式类 */
  classes?: string;
  /* ---- 模板配置 ---- */
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎 默认 `null` */
  tplEng?: (...args: any[]) => any;
  /* ---- 等待函数 ---- */
  /** 填充前等待函数 默认 `null` */
  b4Fill?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 渲染完成回调 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 搜索完成回调 默认 `null` */
  onSearched?: (...args: any[]) => any;
  /** 数据处理回调 默认 `null` */
  onProcessed?: (...args: any[]) => any;
  /** 关键词更新回调 默认 `null` */
  onUpdatedKeys?: (...args: any[]) => any;
  /** 请求完成回调 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /** 内容更新回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"enable": false, "props": "label", "node": "i", "classes": "c-issue"}` */
  highlight?: AxRetrievalHighlightOptions;
  /** 默认 `{"enable": false, "placement": "list-top"}` */
  status?: AxRetrievalStatusOptions;
  /** 默认 `{"type": "custom", "childNode": "div", "unpadded": false, "hoverable": false, "divisible": false, "multiline": false}` */
  bullet?: AxRetrievalBulletOptions;
  /** 默认 `null` */
  'onProcessed"'?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Retrieval 的 highlight 配置组 */
export interface AxRetrievalHighlightOptions {
  /* ---- 高亮配置 ---- */
  /** 是否启用高亮 默认 `false` */
  enable?: boolean;
  /** 高亮属性 默认 `'label'` */
  props?: string;
  /** 高亮节点名 默认 `'i'` */
  node?: string;
  /** 高亮样式类 默认 `'ax-c-issue'` */
  classes?: string;
}

/** Retrieval 的 status 配置组 */
export interface AxRetrievalStatusOptions {
  /* ---- 状态配置 ---- */
  /** 是否显示状态 默认 `false` */
  enable?: boolean;
  /** 状态栏位置 默认 `'list-top'` */
  placement?: 'editor-top' | 'editor-btm' | 'list-top' | 'list-btm';
}

/** Retrieval 的 bullet 配置组 */
export interface AxRetrievalBulletOptions {
  /* ---- 列表配置 ---- */
  /** 列表类型 默认 `'custom'` */
  type?: 'checkboxes' | 'radios' | 'select-single' | 'select-multi' | 'custom';
  /** 子节点名称 默认 `'div'` */
  childNode?: string;
  /** 是否无内边距 默认 `false` */
  paddingless?: boolean;
  /** 是否支持悬停 默认 `false` */
  hoverable?: boolean;
  /** 是否有分割线 默认 `false` */
  divisible?: boolean;
  /** 是否多行显示 默认 `false` */
  multiline?: boolean;
}

/** Retrieval 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxRetrievalEventName = 'constructed' | 'initiate' | 'initiated' | 'processed' | 'searched' | 'updatedKeys';

/** Retrieval 模块实例（new Retrieval() 的返回值） */
export interface AxRetrievalInstance extends AxModBase {
  options: AxRetrievalOptions;
  /** 事件监听（事件名见 AxRetrievalEventName；回调参数视事件而定） */
  on(type: AxRetrievalEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setEmpty(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getStdData(...args: any[]): any;
  searchData(...args: any[]): any;
  useTpl(...args: any[]): any;
  createItemEl(...args: any[]): any;
  getHighlightData(...args: any[]): any;
  updateKeys(...args: any[]): any;
  input2Render(...args: any[]): any;
  getInputVals(...args: any[]): any;
  resetList(...args: any[]): any;
  clearList(...args: any[]): any;
  updateStatus(...args: any[]): any;
  updateCont(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Router                                                                     */
/* （无独立文档页，参数由源码默认值提取）                                                        */
/* ========================================================================== */
/** Router 构造参数 —— new Router(elem, options) 的 options */
export interface AxRouterOptions {
}

/** Router 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxRouterEventName = 'constructed' | 'initiate' | 'initiated' | 'located';

/** Router 模块实例（new Router() 的返回值） */
export interface AxRouterInstance extends AxModBase {
  options: AxRouterOptions;
  /** 事件监听（事件名见 AxRouterEventName；回调参数视事件而定） */
  on(type: AxRouterEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  getUrlPath(...args: any[]): any;
  getRoute(...args: any[]): any;
  add(...args: any[]): any;
  remove(...args: any[]): any;
  push(...args: any[]): any;
  replace(...args: any[]): any;
  back(...args: any[]): any;
  forward(...args: any[]): any;
  go(...args: any[]): any;
  clear(...args: any[]): any;
  routeChange(...args: any[]): any;
  beforeCheck(...args: any[]): any;
  afterCheck(...args: any[]): any;
  restore(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Scroll · 模拟滚动模块                                                            */
/* 文档: modules/349.html                                                       */
/* ========================================================================== */
/** Scroll 构造参数 —— new Scroll(elem, options) 的 options */
export interface AxScrollOptions {
  /* ---- 基础配置 ---- */
  /** 滚动容器选择器 */
  wrapper?: string;
  /** 滚动方向 默认 `'v'` */
  flow?: 'h' | 'v';
  /** 是否允许边缘回弹 默认 `true` */
  bounce?: boolean;
  /** 尺寸变化防抖延时 默认 `200` */
  delay?: number;
  /** 是否使用grid布局 默认 `false` */
  gridded?: boolean;
  /** 拖拽阻尼系数 默认 `0.5` */
  dmpRatio?: number;
  /** 动画中是否可插入事件 默认 `true` */
  insertable?: boolean;
  /* ---- 初始化配置 ---- */
  /** 初始化滚动模式 默认 `'scroll'` */
  toInitMode?: 'scroll' | 'locate';
  /** 动画基础时间(ms) 默认 `300` */
  duration?: number;
  /** 重置次数阈值 默认 `1` */
  resizeThr?: number;
  /* ---- 滚轮配置 ---- */
  /** 是否启用键盘控制 默认 `false` */
  keyboard?: boolean;
  /* ---- 边界配置 ---- */
  /** 完成阈值(px) 默认 `50` */
  completedThr?: number;
  /** 头部边界距离 默认 `0` */
  paddingStart?: number | string;
  /** 尾部边界距离 默认 `0` */
  paddingEnd?: number | string;
  /* ---- 等待函数 ---- */
  /** 头部回弹前等待 默认 `null` */
  b4ToStart?: (...args: any[]) => any;
  /** 尾部回弹前等待 默认 `null` */
  b4ToEnd?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 拖出完成回调 默认 `null` */
  onCompleted?: (...args: any[]) => any;
  /** 超出边界回调 默认 `null` */
  onExceed?: (...args: any[]) => any;
  /** 开始触摸回调 默认 `null` */
  onStart?: (...args: any[]) => any;
  /** 移动过程回调 默认 `null` */
  onMove?: (...args: any[]) => any;
  /** 结束触摸回调 默认 `null` */
  onEnd?: (...args: any[]) => any;
  /** 获取吸附项回调 默认 `null` */
  onGetSnappeds?: (...args: any[]) => any;
  /** 节点变化回调 默认 `null` */
  onRefactored?: (...args: any[]) => any;
  /** 开始吸附回调 默认 `null` */
  onSnap?: (...args: any[]) => any;
  /** 完成吸附回调 默认 `null` */
  onSnapped?: (...args: any[]) => any;
  /** 完成重置回调 默认 `null` */
  onResized?: (...args: any[]) => any;
  /** 开始滚动回调 默认 `null` */
  onScroll?: (...args: any[]) => any;
  /** 完成滚动回调 默认 `null` */
  onScrolled?: (...args: any[]) => any;
  /** 开始跳转回调 默认 `null` */
  onLocate?: (...args: any[]) => any;
  /** 完成跳转回调 默认 `null` */
  onLocated?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"enable": false, "mode": "auto", "force": false, "offset": 0, "selector": ""}` */
  snap?: AxScrollSnapOptions;
  /** 默认 `{"enable": false, "mode": "auto", "value": 0, "coef": 2}` */
  drift?: AxScrollDriftOptions;
  /** 默认 `{"enable": false, "mode": "auto", "value": 0, "coef": 2}` */
  swipe?: AxScrollSwipeOptions;
  /** 默认 `{"enable": false, "sticky": false, "triggerable": true}` */
  bar?: AxScrollBarOptions;
  /** 默认 `{"value": null, "mode": "scroll"}` */
  initial?: AxScrollInitialOptions;
  /** 默认 `{"size": null, "gap": 0, "settable": false}` */
  child?: AxScrollChildOptions;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{"mode": "const", "value": 0, "reverse": false}` */
  wheel?: AxScrollWheelOptions;
  /** 默认 `{}` */
  gesture?: Record<string, any>;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Scroll 的 snap 配置组 */
export interface AxScrollSnapOptions {
  /* ---- 吸附配置 ---- */
  /** 是否启用吸附 默认 `false` */
  enable?: boolean;
  /** 吸附模式 默认 `'auto'` */
  mode?: 'auto' | 'manual';
  /** 是否强制再次吸附 默认 `false` */
  force?: boolean;
  /** 吸附偏差值 默认 `0` */
  offset?: number;
  /** 吸附节点选择器 */
  selector?: string;
}

/** Scroll 的 drift 配置组 */
export interface AxScrollDriftOptions {
  /* ---- 漂移配置 ---- */
  /** 是否启用漂移 默认 `false` */
  enable?: boolean;
  /** 漂移模式 默认 `'auto'` */
  mode?: 'auto' | 'snap' | 'slide' | 'ratio' | 'const';
  /** 漂移步进值 默认 `0` */
  value?: number;
  /** 漂移系数 默认 `2` */
  coef?: number;
}

/** Scroll 的 swipe 配置组 */
export interface AxScrollSwipeOptions {
  /* ---- 滑动配置 ---- */
  /** 是否启用滑动 默认 `false` */
  enable?: boolean;
  /** 滑动模式 默认 `'auto'` */
  mode?: 'auto' | 'snap' | 'slide' | 'ratio' | 'const';
  /** 滑动步进值 默认 `0` */
  value?: number;
  /** 滑动系数 默认 `2` */
  coef?: number;
}

/** Scroll 的 bar 配置组 */
export interface AxScrollBarOptions {
  /* ---- 滚动条配置 ---- */
  /** 是否显示滚动条 默认 `false` */
  enable?: boolean;
  /** 是否默认显示 默认 `false` */
  sticky?: boolean;
  /** 是否可触发拖动 默认 `true` */
  triggerable?: boolean;
}

/** Scroll 的 initial 配置组 */
export interface AxScrollInitialOptions {
  /* ---- 初始化配置 ---- */
  /** 初始滚动距离 默认 `0` */
  value?: number | HTMLElement | string;
  /** 跳转模式 默认 `'scroll'` */
  mode?: 'scroll' | 'locate';
}

/** Scroll 的 child 配置组 */
export interface AxScrollChildOptions {
  /* ---- 子元素配置 ---- */
  /** 子元素尺寸 默认 `null` */
  size?: number | string | null;
  /** 子元素间距 默认 `0` */
  gap?: number | string;
  /** 是否设置尺寸 默认 `false` */
  settable?: boolean;
}

/** Scroll 的 wheel 配置组 */
export interface AxScrollWheelOptions {
  /* ---- 滚轮配置 ---- */
  /** 滚轮模式 默认 `'const'` */
  mode?: 'snap' | 'slide' | 'const' | 'ratio';
  /** 滚动距离 默认 `0` */
  value?: number;
  /** 是否反向滚动 默认 `false` */
  reverse?: boolean;
}

/** Scroll 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxScrollEventName = 'completed' | 'constructed' | 'end' | 'exceeded' | 'getSnappeds' | 'initiate' | 'initiated' | 'locate' | 'located' | 'move' | 'prepared' | 'refactored' | 'resized' | 'scroll' | 'scrolled' | 'snap' | 'snapped' | 'start' | 'toEnd' | 'toStart';

/** Scroll 模块实例（new Scroll() 的返回值） */
export interface AxScrollInstance extends AxModBase {
  options: AxScrollOptions;
  /** 事件监听（事件名见 AxScrollEventName；回调参数视事件而定） */
  on(type: AxScrollEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  initSnapArgs(...args: any[]): any;
  setEmpty(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getGestureIns(...args: any[]): any;
  getRtlRange(...args: any[]): any;
  setTransInit(...args: any[]): any;
  createBar(...args: any[]): any;
  setObserver(...args: any[]): any;
  response(...args: any[]): any;
  getDmpVals(...args: any[]): any;
  getTruePadding(...args: any[]): any;
  setChildSize(...args: any[]): any;
  updateDiff(...args: any[]): any;
  getNowTargetDist(...args: any[]): any;
  setEaseTo(...args: any[]): any;
  getEaseDist(...args: any[]): any;
  getEaseNextTrans(...args: any[]): any;
  getStepVal(...args: any[]): any;
  getWheelNextTrans(...args: any[]): any;
  getEdgeValue(...args: any[]): any;
  updateProg(...args: any[]): any;
  getBarSize(...args: any[]): any;
  updateBarProg(...args: any[]): any;
  exceedListen(...args: any[]): any;
  handleExceedLimit(...args: any[]): any;
  crossLinkage(...args: any[]): any;
  toStart(...args: any[]): any;
  toEnd(...args: any[]): any;
  toPrev(...args: any[]): any;
  toNext(...args: any[]): any;
  getSnappeds(...args: any[]): any;
  getChildParentOffset(...args: any[]): any;
  sortEls(...args: any[]): any;
  getNearestSnap(...args: any[]): any;
  getNearestSlide(...args: any[]): any;
  setSnapped(...args: any[]): any;
  toggleSnappedAttr(...args: any[]): any;
  scrollTo(...args: any[]): any;
  locateTo(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Select · 下拉菜单模块                                                            */
/* 文档: modules/285.html                                                       */
/* ========================================================================== */
/** Select 构造参数 —— new Select(elem, options) 的 options */
export interface AxSelectOptions {
  /* ---- 基础配置 ---- */
  /** 隐藏input的name属性 */
  name?: string;
  /** 初始值 */
  value?: string;
  /** 输出值字段 默认 `'label'` */
  field?: 'label' | 'value';
  /** 列表类型 默认 `'dropdown'` */
  type?: 'dropdown' | 'check' | 'button' | 'list';
  /** 是否手动编辑模式 默认 `false` */
  manual?: boolean;
  /** 是否禁用 默认 `false` */
  disabled?: boolean;
  /** 是否只读 默认 `false` */
  readonly?: boolean;
  /** 是否横向铺满 默认 `false` */
  full?: boolean;
  /** 是否多选 默认 `false` */
  multiple?: boolean;
  /** 初始禁用项 */
  exclude?: string | number | HTMLElement | Record<string, any> | any[];
  /** 最少选择数量 默认 `0` */
  min?: number;
  /** 最多选择数量 默认 `0` */
  max?: number;
  /** 是否裁剪超出部分 默认 `true` */
  sliced?: boolean;
  /** 是否显示清除按钮 默认 `true` */
  removable?: boolean;
  /** 值是否唯一 默认 `true` */
  unique?: boolean;
  /** 选择范围 默认 `'leaf'` */
  span?: 'tree' | 'branch' | 'leaf';
  /** 是否折叠所有分支 默认 `false` */
  collapse?: boolean;
  /* ---- 数据源配置 ---- */
  /** 数据源内容 */
  content?: string;
  /** 内容类型 默认 `'text'` */
  contType?: 'text' | 'async' | 'html' | 'auto';
  /** 请求数据参数 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 外观配置 ---- */
  /** 选择器尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 最大高度 */
  maxHeight?: string;
  /** 是否保持下拉菜单宽度 默认 `true` */
  autoWidth?: boolean;
  /** 是否使用状态栏 默认 `false` */
  status?: boolean;
  /* ---- 工具模块配置 ---- */
  /** 标签组件参数 默认 `{}` */
  tags?: Record<string, any>;
  /** 树形组件参数 默认 `{}` */
  tree?: Record<string, any>;
  /* ---- 等待函数 ---- */
  /** 展开前等待函数 默认 `null` */
  b4Expand?: (...args: any[]) => any;
  /** 折叠前等待函数 默认 `null` */
  b4Collapse?: (...args: any[]) => any;
  /** 添加前等待函数 默认 `null` */
  b4Add?: (...args: any[]) => any;
  /** 编辑前等待函数 默认 `null` */
  b4Edit?: (...args: any[]) => any;
  /** 删除前等待函数 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /** 拖拽前等待函数 默认 `null` */
  b4Graft?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 渲染完成回调 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 节点触发回调 默认 `null` */
  onTrigger?: (...args: any[]) => any;
  /** 添加后回调 默认 `null` */
  onAdded?: (...args: any[]) => any;
  /** 编辑后回调 默认 `null` */
  onEdited?: (...args: any[]) => any;
  /** 删除后回调 默认 `null` */
  onRemoved?: (...args: any[]) => any;
  /** 输出值回调 默认 `null` */
  onOutput?: (...args: any[]) => any;
  /** 禁用回调 默认 `null` */
  onDisabled?: (...args: any[]) => any;
  /** 启用回调 默认 `null` */
  onEnabled?: (...args: any[]) => any;
  /** 全部禁用回调 默认 `null` */
  onDisabledAll?: (...args: any[]) => any;
  /** 全部启用回调 默认 `null` */
  onEnabledAll?: (...args: any[]) => any;
  /** 只读回调 默认 `null` */
  onReadonly?: (...args: any[]) => any;
  /** 全部只读回调 默认 `null` */
  onReadonlyAll?: (...args: any[]) => any;
  /** 全局禁用回调 默认 `null` */
  onPassivated?: (...args: any[]) => any;
  /** 取消全局禁用回调 默认 `null` */
  onActivated?: (...args: any[]) => any;
  /** 展开前回调 默认 `null` */
  onExpand?: (...args: any[]) => any;
  /** 展开后回调 默认 `null` */
  onExpanded?: (...args: any[]) => any;
  /** 折叠前回调 默认 `null` */
  onCollapse?: (...args: any[]) => any;
  /** 折叠后回调 默认 `null` */
  onCollapsed?: (...args: any[]) => any;
  /** 全部展开前回调 默认 `null` */
  onExpandAll?: (...args: any[]) => any;
  /** 全部展开后回调 默认 `null` */
  onExpandedAll?: (...args: any[]) => any;
  /** 全部折叠前回调 默认 `null` */
  onCollapsedAll?: (...args: any[]) => any;
  /** 选中回调 默认 `null` */
  onSelected?: (...args: any[]) => any;
  /** 取消选中回调 默认 `null` */
  onDeselected?: (...args: any[]) => any;
  /** 全选回调 默认 `null` */
  onSelectedAll?: (...args: any[]) => any;
  /** 勾选回调 默认 `null` */
  onChecked?: (...args: any[]) => any;
  /** 取消勾选回调 默认 `null` */
  onUnchecked?: (...args: any[]) => any;
  /** 全部勾选回调 默认 `null` */
  onCheckedAll?: (...args: any[]) => any;
  /** 选择过少回调 默认 `null` */
  onTooFew?: (...args: any[]) => any;
  /** 选择过多回调 默认 `null` */
  onTooMany?: (...args: any[]) => any;
  /** 填充完成回调 默认 `null` */
  onFilled?: (...args: any[]) => any;
  /** 翻页回调 默认 `null` */
  onTurned?: (...args: any[]) => any;
  /** 取值回调 默认 `null` */
  onGetted?: (...args: any[]) => any;
  /** 设置值回调 默认 `null` */
  onSetted?: (...args: any[]) => any;
  /** 清除值回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 节点转移回调 默认 `null` */
  onGrafted?: (...args: any[]) => any;
  /** 内容更新回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /** 请求完成回调 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"enable": false, "value": "", "fuzzy": true, "ignore": true, "result": true}` */
  search?: AxSelectSearchOptions;
  /** 默认 `{"enable": false, "children": ["close"]}` */
  tools?: AxSelectToolsOptions;
  /** 默认 `{"enable": true}` */
  popup?: AxSelectPopupOptions;
  /** 默认 `null` */
  onGot?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSet?: ((...args: any[]) => any);
  /** 默认 `null` */
  onLocked?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUnlocked?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Select 的 search 配置组 */
export interface AxSelectSearchOptions {
  /* ---- 搜索配置 ---- */
  /** 是否启用搜索 默认 `false` */
  enable?: boolean;
  /** 搜索初始值 */
  value?: string;
  /** 是否模糊搜索 默认 `true` */
  fuzzy?: boolean;
  /** 是否忽略大小写 默认 `true` */
  ignore?: boolean;
  /** 是否显示搜索结果 默认 `true` */
  result?: boolean;
}

/** Select 的 tools 配置组 */
export interface AxSelectToolsOptions {
  /* ---- 工具模块配置 ---- */
  /** 是否启用工具箱 默认 `false` */
  enable?: boolean;
  /** 工具类型列表 默认 `['close']` */
  children?: any[];
}

/** Select 的 popup 配置组 */
export interface AxSelectPopupOptions {
  /* ---- 工具模块配置 ---- */
  /** 是否启用弹出层 默认 `true` */
  enable?: boolean;
}

/** Select 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxSelectEventName = 'constructed' | 'deselected' | 'got' | 'initiate' | 'initiated' | 'output' | 'selected' | 'selectedAll';

/** Select 模块实例（new Select() 的返回值） */
export interface AxSelectInstance extends AxModBase {
  options: AxSelectOptions;
  /** 事件监听（事件名见 AxSelectEventName；回调参数视事件而定） */
  on(type: AxSelectEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  renderMain(...args: any[]): any;
  updateResult(...args: any[]): any;
  updateStatus(...args: any[]): any;
  setPopup(...args: any[]): any;
  setTags(...args: any[]): any;
  setTree(...args: any[]): any;
  correctOpts(...args: any[]): any;
  setAttrs(...args: any[]): any;
  renderFinish(...args: any[]): any;
  getVals(...args: any[]): any;
  select(...args: any[]): any;
  deselect(...args: any[]): any;
  clear(...args: any[]): any;
  selectAll(...args: any[]): any;
  disable(...args: any[]): any;
  enable(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Spy · 滚动侦听模块                                                               */
/* 文档: modules/277.html                                                       */
/* ========================================================================== */
/** Spy 构造参数 —— new Spy(elem, options) 的 options */
export interface AxSpyOptions {
  /* ---- 基础配置 ---- */
  /** 触发偏移距离 默认 `'0px'` */
  margin?: string;
  /** 可见区域占比 默认 `[0, 1]` */
  threshold?: any[];
  /** 参照父节点选择器 默认 `null` */
  root?: string | null;
  /** 防抖间隔时间(ms) 默认 `500` */
  intvl?: number;
  /** 是否重复执行动画 默认 `true` */
  repeat?: boolean;
  /** 是否手动添加观察对象 默认 `false` */
  manual?: boolean;
  /** 追加样式类 */
  classes?: string;
  /* ---- 选择器配置 ---- */
  /** 监听子元素选择器 */
  spySel?: string;
  /** 菜单节点选择器 */
  navSel?: string;
  /** 菜单子元素选择器 */
  subSel?: string;
  /* ---- 动画配置 ---- */
  /** 激活属性或类名 */
  active?: string;
  /** 进入视窗动画类名 */
  in?: string;
  /** 离开视窗动画类名 */
  out?: string;
  /** 是否使用透明度 默认 `true` */
  visible?: boolean;
  /* ---- 等待函数 ---- */
  /** 添加交互前等待 默认 `null` */
  b4Add?: (...args: any[]) => any;
  /** 删除交互前等待 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 开始进入视窗回调 默认 `null` */
  onShow?: (...args: any[]) => any;
  /** 正在进入视窗回调 默认 `null` */
  onShowing?: (...args: any[]) => any;
  /** 完全显示在视窗回调 默认 `null` */
  onShown?: (...args: any[]) => any;
  /** 进入视窗回调 默认 `null` */
  onIn?: (...args: any[]) => any;
  /** 离开视窗回调 默认 `null` */
  onOut?: (...args: any[]) => any;
  /** 新增监听节点回调 默认 `null` */
  onAdded?: (...args: any[]) => any;
  /** 取消监听节点回调 默认 `null` */
  onRemoved?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"selector": "", "offset": 0, "duration": 0}` */
  wing?: AxSpyWingOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Spy 的 wing 配置组 */
export interface AxSpyWingOptions {
  /* ---- 关联按钮配置 ---- */
  /** 按钮节点选择器 */
  selector?: string;
  /** 顶部偏移距离 默认 `0` */
  offset?: number;
  /** 滑动动画时长 默认 `0` */
  duration?: number;
}

/** Spy 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxSpyEventName = 'add' | 'constructed' | 'in' | 'initiate' | 'initiated' | 'out' | 'remove' | 'showing' | 'shown';

/** Spy 模块实例（new Spy() 的返回值） */
export interface AxSpyInstance extends AxModBase {
  options: AxSpyOptions;
  /** 事件监听（事件名见 AxSpyEventName；回调参数视事件而定） */
  on(type: AxSpyEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  getActive(...args: any[]): any;
  setActiveOnly(...args: any[]): any;
  setSpyAttr(...args: any[]): any;
  getWings(...args: any[]): any;
  getEls(...args: any[]): any;
  add(...args: any[]): any;
  remove(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Swipe · 轮播模块                                                               */
/* 文档: modules/275.html                                                       */
/* ========================================================================== */
/** Swipe 构造参数 —— new Swipe(elem, options) 的 options */
export interface AxSwipeOptions {
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `'h'` */
  flow?: string;
  /** 默认 `false` */
  keyboard?: boolean;
  /** 默认 `0` */
  active?: number;
  /** 默认 `500` */
  duration?: number;
  /** 默认 `0` */
  throttle?: number;
  /** 默认 `1` */
  slides?: number;
  /** 默认 `1` */
  divide?: number;
  /** 默认 `0` */
  group?: number;
  /** 默认 `false` */
  loop?: boolean;
  /** 默认 `false` */
  autoHeight?: boolean;
  /** 默认 `{}` */
  countdown?: Record<string, any>;
  /** 默认 `0` */
  gap?: number;
  /** 默认 `'[unalignable]'` */
  unalign?: string;
  /** 默认 `{"enable": false, "fill": false}` */
  centered?: Record<string, any>;
  /** 默认 `true` */
  resizeReset?: boolean;
  /** 默认 `1` */
  drift?: number;
  /** 默认 `1` */
  wheel?: number;
  /** 默认 `{}` */
  scroll?: Record<string, any>;
  /** 默认 `0` */
  height?: number;
  /** 默认 `0` */
  width?: number;
  /** 默认 `{"enable": false, "placement": "right-top", "children": []}` */
  tools?: Record<string, any>;
  /** 默认 `{"enable": false, "placement": "right-top", "format": "<i></i><s>/</s><u></u>", "selector": ""}` */
  total?: Record<string, any>;
  /** 默认 `{"enable": false, "classes": "", "type": "lamp", "selector": "", "data": [], "before": null, "after": null}` */
  pgn?: Record<string, any>;
  /** 默认 `{"enable": true, "classes": "", "fill": false, "intvl": 500, "prev": {"selector": "", "classes": "", "text": ""}, "next": {"selector": "", "classes": "", "text": ""}}` */
  nav?: Record<string, any>;
  /** 默认 `{"content": null, "type": "swipe", "attribute": "selected", "mode": "scroll"}` */
  thumb?: Record<string, any>;
  /** 默认 `{"enable": false, "step": 0.5, "min": 0, "max": 10, "value": 2, "selector": ""}` */
  zoom?: Record<string, any>;
  /** 默认 `{"enable": false, "delay": 4000, "countdown": false}` */
  autoplay?: Record<string, any>;
  /** 默认 `false` */
  integrated?: boolean;
  content?: string;
  /** 默认 `'text'` */
  contType?: string;
  /** 默认 `{}` */
  contData?: Record<string, any>;
  /** 默认 `{}` */
  ajax?: Record<string, any>;
  /** 默认 `null` */
  hyphen?: any;
  tplStr?: string;
  /** 默认 `null` */
  tplEng?: any;
  /** 默认 `false` */
  lazy?: boolean;
  /** 默认 `{"title": "", "brief": ""}` */
  media?: Record<string, any>;
  classes?: string;
  /** 默认 `null` */
  onPrepared?: ((...args: any[]) => any);
  /** 默认 `null` */
  onChanged?: ((...args: any[]) => any);
  /** 默认 `null` */
  onPaused?: ((...args: any[]) => any);
  /** 默认 `null` */
  onPlayed?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Swipe 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxSwipeEventName = 'add' | 'changed' | 'constructed' | 'initiate' | 'initiated' | 'paused' | 'played' | 'prepared' | 'remove' | 'resized';

/** Swipe 模块实例（new Swipe() 的返回值） */
export interface AxSwipeInstance extends AxModBase {
  options: AxSwipeOptions;
  /** 事件监听（事件名见 AxSwipeEventName；回调参数视事件而定） */
  on(type: AxSwipeEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getDriftParams(...args: any[]): any;
  getWheelVal(...args: any[]): any;
  getZoomEl(...args: any[]): any;
  setZoomVal(...args: any[]): any;
  enableZoom(...args: any[]): any;
  enable2Init(...args: any[]): any;
  toggleZoom(...args: any[]): any;
  setZoomIns(...args: any[]): any;
  setScroll(...args: any[]): any;
  getSliders(...args: any[]): any;
  findSlider(...args: any[]): any;
  appendSliders(...args: any[]): any;
  createWrap(...args: any[]): any;
  fillWedgets(...args: any[]): any;
  updateWedgets(...args: any[]): any;
  setSlidesObs(...args: any[]): any;
  setBulletsObs(...args: any[]): any;
  getThumbSlides(...args: any[]): any;
  findPrevNextBullet(...args: any[]): any;
  toggleNavDisabled(...args: any[]): any;
  setSlideSize(...args: any[]): any;
  updateGap(...args: any[]): any;
  getIdxFromActive(...args: any[]): any;
  getActiveIdex(...args: any[]): any;
  getInitSlider(...args: any[]): any;
  getAccuracy(...args: any[]): any;
  getSingleSize(...args: any[]): any;
  setSingleSize(...args: any[]): any;
  setSingleGap(...args: any[]): any;
  getBullets(...args: any[]): any;
  updateUnsnappeds(...args: any[]): any;
  getSplits(...args: any[]): any;
  getSanpOffsetArr(...args: any[]): any;
  updateTotal(...args: any[]): any;
  fillTotal(...args: any[]): any;
  getTotal(...args: any[]): any;
  fillNav(...args: any[]): any;
  fillPgn(...args: any[]): any;
  setBulletAction(...args: any[]): any;
  updatePgn(...args: any[]): any;
  findNavSnap(...args: any[]): any;
  findSnap(...args: any[]): any;
  findBullet(...args: any[]): any;
  toBullet(...args: any[]): any;
  toPrevBullet(...args: any[]): any;
  toNextBullet(...args: any[]): any;
  toSnap(...args: any[]): any;
  toLocate(...args: any[]): any;
  toPrevSnap(...args: any[]): any;
  toNextSnap(...args: any[]): any;
  toStart(...args: any[]): any;
  toEnd(...args: any[]): any;
  setLoopPlay(...args: any[]): any;
  startCountdown(...args: any[]): any;
  setAutoPlay(...args: any[]): any;
  continue(...args: any[]): any;
  play(...args: any[]): any;
  pause(...args: any[]): any;
  autoPlay(...args: any[]): any;
  autoPause(...args: any[]): any;
  add(...args: any[]): any;
  remove(...args: any[]): any;
  destroy(...args: any[]): any;
  updateCont(...args: any[]): any;
}

/* ========================================================================== */
/* Tab · 页签模块                                                                 */
/* 文档: modules/274.html                                                       */
/* ========================================================================== */
/** Tab 构造参数 —— new Tab(elem, options) 的 options */
export interface AxTabOptions {
  /* ---- 源码默认参数（文档未单列） ---- */
  content?: string;
  /** 默认 `'text'` */
  contType?: string;
  /** 默认 `{}` */
  contData?: Record<string, any>;
  /** 默认 `{}` */
  ajax?: Record<string, any>;
  /** 默认 `false` */
  passive?: boolean;
  active?: string;
  disable?: string;
  /** 默认 `'click'` */
  trigger?: string;
  /** 默认 `0` */
  duration?: number;
  /** 默认 `0` */
  max?: number;
  /** 默认 `false` */
  embed?: boolean;
  /** 默认 `{"enable": false, "arrowShow": false, "options": {"slides": "auto", "scroll": {"drift": {"mode": "free", "coef": 1}}}}` */
  slider?: Record<string, any>;
  /** 默认 `{"enable": false, "options": {"columns": 4, "size": "lg"}}` */
  menu?: Record<string, any>;
  /** 默认 `{"enable": false, "position": "left"}` */
  col?: Record<string, any>;
  /** 默认 `{"target": {}, "head": {}, "body": {}}` */
  attrs?: Record<string, any>;
  /** 默认 `{"target": "", "head": "", "body": ""}` */
  classes?: Record<string, any>;
  /** 默认 `{"enable": false, "size": ""}` */
  divider?: Record<string, any>;
  /** 默认 `true` */
  addActive?: boolean;
  /** 默认 `{"enable": false, "children": ["close"]}` */
  tools?: Record<string, any>;
  /** 默认 `'lamp'` */
  layout?: string;
  /** 默认 `{"head": "", "body": ""}` */
  selector?: Record<string, any>;
  /** 默认 `0` */
  sticky?: number;
  /** 默认 `null` */
  b4Activate?: ((...args: any[]) => any);
  /** 默认 `null` */
  b4Add?: ((...args: any[]) => any);
  /** 默认 `null` */
  b4Edit?: ((...args: any[]) => any);
  /** 默认 `null` */
  b4Remove?: ((...args: any[]) => any);
  /** 默认 `null` */
  b4Graft?: ((...args: any[]) => any);
  /** 默认 `null` */
  onRendered?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInsertItem?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInsertItems?: ((...args: any[]) => any);
  /** 默认 `null` */
  onTrigger?: ((...args: any[]) => any);
  /** 默认 `null` */
  onAdded?: ((...args: any[]) => any);
  /** 默认 `null` */
  onRemoved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onEdited?: ((...args: any[]) => any);
  /** 默认 `null` */
  onGrafted?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDisabled?: ((...args: any[]) => any);
  /** 默认 `null` */
  onEnabled?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDisabledAll?: ((...args: any[]) => any);
  /** 默认 `null` */
  onEnabledAll?: ((...args: any[]) => any);
  /** 默认 `null` */
  onActivated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedItemCont?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCont?: ((...args: any[]) => any);
  /** 默认 `null` */
  onRequest?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Tab 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxTabEventName = 'activated' | 'added' | 'constructed' | 'edited' | 'grafted' | 'initiate' | 'initiated' | 'insertItems' | 'removed' | 'updatedItemCont';

/** Tab 模块实例（new Tab() 的返回值） */
export interface AxTabInstance extends AxModBase {
  options: AxTabOptions;
  /** 事件监听（事件名见 AxTabEventName；回调参数视事件而定） */
  on(type: AxTabEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  initActive(...args: any[]): any;
  getDataToRender(...args: any[]): any;
  getTreeFlat(...args: any[]): any;
  getObserver(...args: any[]): any;
  htmlToArr(...args: any[]): any;
  renderData(...args: any[]): any;
  createItemEl(...args: any[]): any;
  getTools(...args: any[]): any;
  fillContent(...args: any[]): any;
  updateHeadBodyAttrs(...args: any[]): any;
  setAttrs(...args: any[]): any;
  updateEvt(...args: any[]): any;
  renderFinish(...args: any[]): any;
  getSelected(...args: any[]): any;
  addTrigger(...args: any[]): any;
  removeTrigger(...args: any[]): any;
  toggleDisabled(...args: any[]): any;
  disable(...args: any[]): any;
  disableAll(...args: any[]): any;
  enable(...args: any[]): any;
  enableAll(...args: any[]): any;
  activate(...args: any[]): any;
  add(...args: any[]): any;
  remove(...args: any[]): any;
  edit(...args: any[]): any;
  graft(...args: any[]): any;
  updateItemCont(...args: any[]): any;
  updateCont(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Tags · 标记模块                                                                */
/* 文档: modules/350.html                                                       */
/* ========================================================================== */
/** Tags 构造参数 —— new Tags(elem, options) 的 options */
export interface AxTagsOptions {
  /* ---- 数据源配置 ---- */
  /** 数据源内容 */
  content?: string;
  /** 内容类型 默认 `'text'` */
  contType?: string;
  /** 请求数据参数 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /** 显示字段名 默认 `'label'` */
  field?: string;
  /** 数据分隔符 默认 `null` */
  separator?: string;
  /* ---- 外观配置 ---- */
  /** 主题风格 */
  theme?: 'prim' | 'succ' | 'error' | 'warn' | 'issue' | 'info' | 'text' | 'brief' | 'caption';
  /** 标签尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 标签类型 */
  type?: string;
  /** 是否紧凑边距 默认 `false` */
  compact?: boolean;
  /** 边框圆角 */
  shape?: string;
  /** 追加样式类 */
  classes?: string;
  /* ---- 数量限制配置 ---- */
  /** 最大标签数量 默认 `0` */
  max?: number;
  /** 最小标签数量 默认 `0` */
  min?: number;
  /** 是否截断超出部分 默认 `true` */
  sliced?: boolean;
  /* ---- 编辑器配置 ---- */
  /** 是否显示关闭按钮 默认 `false` */
  removable?: boolean;
  /* ---- 等待函数 ---- */
  /** 填充前等待函数 默认 `null` */
  b4Fill?: (...args: any[]) => any;
  /** 添加前等待函数 默认 `null` */
  b4Add?: (...args: any[]) => any;
  /** 删除前等待函数 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /** 编辑前等待函数 默认 `null` */
  b4Edit?: (...args: any[]) => any;
  /** 清除前等待函数 默认 `null` */
  b4Clear?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 添加后回调 默认 `null` */
  onAdded?: (...args: any[]) => any;
  /** 删除后回调 默认 `null` */
  onRemoved?: (...args: any[]) => any;
  /** 编辑后回调 默认 `null` */
  onEdited?: (...args: any[]) => any;
  /** 清空后回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 重复添加回调 默认 `null` */
  onDuplicated?: (...args: any[]) => any;
  /** 请求完成回调 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /** 输出结果回调 默认 `null` */
  onOutput?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"enable": false, "refer": "id", "template": "{{this.field}}<i class=\"c-brief\">({{this.key}}:{{this.value}})</i>"}` */
  unique?: AxTagsUniqueOptions;
  /** 默认 `{"enable": true}` */
  empty?: AxTagsEmptyOptions;
  /** 默认 `{"enable": false, "addable": true, "deletable": true, "selector": ""}` */
  editor?: AxTagsEditorOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Tags 的 unique 配置组 */
export interface AxTagsUniqueOptions {
  /* ---- 唯一性配置 ---- */
  /** 是否要求唯一值 默认 `false` */
  enable?: boolean;
  /** 重复标识字段 默认 `'id'` */
  refer?: string;
  /** 内容模板 默认 `'{{this.field}}({{this.key}}:{{this.value}})'` */
  template?: string;
}

/** Tags 的 empty 配置组 */
export interface AxTagsEmptyOptions {
  /* ---- 空状态配置 ---- */
  /** 是否显示空状态 默认 `true` */
  enable?: boolean;
  /** 空状态内容 默认 `'[空状态文本]'` */
  content?: string;
}

/** Tags 的 editor 配置组 */
export interface AxTagsEditorOptions {
  /* ---- 编辑器配置 ---- */
  /** 是否显示输入框 默认 `false` */
  enable?: boolean;
  /** 是否可回车添加 默认 `true` */
  addable?: boolean;
  /** 是否可删除键删除 默认 `true` */
  deletable?: boolean;
  /** 外部编辑器选择器 */
  selector?: string;
  /** 输入框占位符 默认 `'[提示文本]'` */
  placeholder?: string;
}

/** Tags 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxTagsEventName = 'added' | 'constructed' | 'duplicated' | 'edited' | 'got' | 'initiate' | 'initiated' | 'output' | 'removed';

/** Tags 模块实例（new Tags() 的返回值） */
export interface AxTagsInstance extends AxModBase {
  options: AxTagsOptions;
  /** 事件监听（事件名见 AxTagsEventName；回调参数视事件而定） */
  on(type: AxTagsEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  initContent(...args: any[]): any;
  setDataObs(...args: any[]): any;
  setOutput(...args: any[]): any;
  renderContent(...args: any[]): any;
  getStrVals(...args: any[]): any;
  getSource(...args: any[]): any;
  setAttrs(...args: any[]): any;
  setEmpty(...args: any[]): any;
  useLegend(...args: any[]): any;
  createTag(...args: any[]): any;
  createEditor(...args: any[]): any;
  toggleEmpty(...args: any[]): any;
  renderFinish(...args: any[]): any;
  getVals(...args: any[]): any;
  updateCont(...args: any[]): any;
  saveRaw(...args: any[]): any;
  add(...args: any[]): any;
  remove(...args: any[]): any;
  edit(...args: any[]): any;
  clear(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Toast · 短消息                                                                */
/* 文档: modules/426.html                                                       */
/* ========================================================================== */
/** Toast 构造参数 —— new Toast(elem, options) 的 options */
export interface AxToastOptions {
  /* ---- 基础显示配置 ---- */
  /** Toast 显示内容 */
  content?: string | boolean;
  /** 自定义 CSS 类名 */
  classes?: string;
  /** 显示位置 默认 `'bottom'` */
  placement?: 'top' | 'center' | 'bottom';
  /** 显示特性 */
  feature?: 'plain';
  /** 层级 z-index 值 默认 `0` */
  zIndex?: number;
  /* ---- 状态配置 ---- */
  /** Toast 状态类型 默认 `'info'` */
  status?: 'succ' | 'error' | 'issue' | 'info' | 'warn';
  /** 是否显示图标 默认 `false` */
  iconized?: boolean;
  /* ---- 行为配置 ---- */
  /** 自动关闭延迟(毫秒) 默认 `3000` */
  delay?: number;
  /** 是否立即显示 默认 `false` */
  eager?: boolean;
  /* ---- 回调函数 ---- */
  /** 显示后回调 默认 `null` */
  onShown?: (...args: any[]) => any;
  /** 隐藏后回调 默认 `null` */
  onHidden?: (...args: any[]) => any;
  /** 更新后回调 默认 `null` */
  onUpdated?: (...args: any[]) => any;
  /** 内容更新后回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Toast 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxToastEventName = 'constructed' | 'hidden' | 'initiate' | 'initiated' | 'shown';

/** Toast 模块实例（new Toast() 的返回值） */
export interface AxToastInstance extends AxModBase {
  options: AxToastOptions;
  /** 事件监听（事件名见 AxToastEventName；回调参数视事件而定） */
  on(type: AxToastEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setAttrs(...args: any[]): any;
  createSection(...args: any[]): any;
  hideOthers(...args: any[]): any;
  show(...args: any[]): any;
  hide(...args: any[]): any;
  updateCont(...args: any[]): any;
  update(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Tooltip · 气泡提示模块                                                           */
/* 文档: modules/271.html                                                       */
/* ========================================================================== */
/** Tooltip 构造参数 —— new Tooltip(elem, options) 的 options */
export interface AxTooltipOptions {
  /* ---- 源码默认参数（文档未单列） ---- */
  theme?: string;
  /** 默认 `'top'` */
  placement?: string;
  /** 默认 `'hover'` */
  trigger?: string;
  /** 默认 `true` */
  pageClose?: boolean;
  /** 默认 `null` */
  canClose?: any;
  /** 默认 `'md'` */
  size?: string;
  /** 默认 `'fadeIn'` */
  in?: string;
  /** 默认 `'fadeOut'` */
  out?: string;
  /** 默认 `false` */
  multiple?: boolean;
  height?: string;
  /** 默认 `true` */
  adaptive?: boolean;
  /** 默认 `{}` */
  position?: Record<string, any>;
  /** 默认 `{"enable": true, "size": "0.5rem", "offset": "2.8rem", "gap": ".8rem"}` */
  arrow?: Record<string, any>;
  /** 默认 `{"selector": "", "trigger": "click", "actClass": ""}` */
  wing?: Record<string, any>;
  /** 默认 `null` */
  onReposition?: ((...args: any[]) => any);
  heading?: string;
  /** 默认 `false` */
  divider?: boolean;
  /** 默认 `0` */
  duration?: number;
  /** 默认 `false` */
  autoDur?: boolean;
  /** 默认 `false` */
  asleep?: boolean;
  /** 默认 `false` */
  deadShow?: boolean;
  /** 默认 `false` */
  keepShow?: boolean;
  classes?: string;
  /** 默认 `{"enable": true, "body": true, "foot": true}` */
  padding?: Record<string, any>;
  content?: string;
  /** 默认 `'text'` */
  contType?: string;
  /** 默认 `{}` */
  contData?: Record<string, any>;
  /** 默认 `{}` */
  ajax?: Record<string, any>;
  /** 默认 `null` */
  actClass?: any;
  tplStr?: string;
  /** 默认 `null` */
  tplEng?: any;
  /** 默认 `0` */
  zIndex?: number;
  /** 默认 `false` */
  dedicated?: boolean;
  /** 默认 `null` */
  canClick?: any;
  /** 默认 `{"title": "", "brief": ""}` */
  media?: Record<string, any>;
  /** 默认 `{"enable": false, "inputSel": "", "childSel": "", "detectable": false, "attr": ""}` */
  autoFill?: Record<string, any>;
  /** 默认 `{"enable": false, "type": "custom", "parentNode": "ul", "childNode": "li", "unpadded": false, "hoverable": false, "divisible": false, "multiline": false, "cols": 0, "lines": "", "action": null}` */
  bullet?: Record<string, any>;
  /** 默认 `{"enable": true, "placement": "inside", "children": ["close"]}` */
  tools?: Record<string, any>;
  /** 默认 `{"enable": true, "layout": "plain", "note": false, "divider": false, "padding": false, "children": ["cancel", "confirm"]}` */
  footer?: Record<string, any>;
  /** 默认 `null` */
  b4Show?: ((...args: any[]) => any);
  /** 默认 `null` */
  b4Hide?: ((...args: any[]) => any);
  /** 默认 `null` */
  b4Fill?: ((...args: any[]) => any);
  /** 默认 `null` */
  b4Confirm?: ((...args: any[]) => any);
  /** 默认 `null` */
  b4Cancel?: ((...args: any[]) => any);
  /** 默认 `null` */
  onShow?: ((...args: any[]) => any);
  /** 默认 `null` */
  onHide?: ((...args: any[]) => any);
  /** 默认 `null` */
  onShown?: ((...args: any[]) => any);
  /** 默认 `null` */
  onHidden?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInvalidated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onValidated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSended?: ((...args: any[]) => any);
  /** 默认 `null` */
  onRender?: ((...args: any[]) => any);
  /** 默认 `null` */
  onRendered?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCont?: ((...args: any[]) => any);
  /** 默认 `null` */
  onTransferred?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConfirmed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onCanceled?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClosed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onCleared?: ((...args: any[]) => any);
  /** 默认 `null` */
  onRequest?: ((...args: any[]) => any);
  /** 默认 `null` */
  onBulletsChecked?: ((...args: any[]) => any);
  /** 默认 `null` */
  onBulletsSelected?: ((...args: any[]) => any);
  /** 默认 `null` */
  onTargetSet?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Tooltip 模块实例（new Tooltip() 的返回值） */
export interface AxTooltipInstance extends AxModBase {
  options: AxTooltipOptions;
}

/* ========================================================================== */
/* Tree · 树型菜单模块                                                              */
/* 文档: modules/269.html                                                       */
/* ========================================================================== */
/** Tree 构造参数 —— new Tree(elem, options) 的 options */
export interface AxTreeOptions {
  /* ---- 基础配置 ---- */
  /** 隐藏input的name属性 */
  name?: string;
  /** 初始值 */
  value?: string;
  /** 数据源内容 */
  content?: string;
  /** 内容类型 默认 `'text'` */
  contType?: string;
  /** 请求数据参数 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /** 是否完全禁止使用 默认 `false` */
  passive?: boolean;
  /** 初始化禁用项 */
  disable?: string | number | HTMLElement | Record<string, any> | any[];
  /** 初始化只读项 */
  readonly?: string | number | HTMLElement | Record<string, any> | any[];
  /** 限制分支最少数量 默认 `0` */
  min?: number;
  /** 限制分支最多数量 默认 `0` */
  max?: number;
  /** 是否裁剪掉超出的部分 默认 `true` */
  sliced?: boolean;
  /** 触发事件的防抖延迟时间(ms) 默认 `100` */
  delay?: number;
  /** 是否暂时不渲染子节点 默认 `false` */
  deferred?: boolean;
  /** 是否显示连线 默认 `false` */
  chain?: boolean;
  /** 是否自动填充content 默认 `true` */
  autoFill?: boolean;
  /** 顶层分支的编号 默认 `null` */
  rootStart?: number;
  /** 分支id开始编号 默认 `null` */
  idStart?: number;
  /* ---- 工具配置 ---- */
  /** 标签组件参数 默认 `{}` */
  tags?: Record<string, any>;
  /** 树形组件参数 默认 `{}` */
  tree?: Record<string, any>;
  /* ---- 布局和特性配置 ---- */
  /** head部分的节点布局 默认 `'indent|arrow|check|legend|icon|disk|cube|image|label|badge|tips|custom|tools'` */
  layout?: string;
  /* ---- 等待函数 ---- */
  /** 展开前等待函数 默认 `null` */
  b4Expand?: (...args: any[]) => any;
  /** 折叠前等待函数 默认 `null` */
  b4Collapse?: (...args: any[]) => any;
  /** 添加前等待函数 默认 `null` */
  b4Add?: (...args: any[]) => any;
  /** 编辑前等待函数 默认 `null` */
  b4Edit?: (...args: any[]) => any;
  /** 删除前等待函数 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /** 拖拽前等待函数 默认 `null` */
  b4Graft?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 渲染完成回调 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 节点触发回调 默认 `null` */
  onTrigger?: (...args: any[]) => any;
  /** 添加后回调 默认 `null` */
  onAdded?: (...args: any[]) => any;
  /** 编辑后回调 默认 `null` */
  onEdited?: (...args: any[]) => any;
  /** 删除后回调 默认 `null` */
  onRemoved?: (...args: any[]) => any;
  /** 输出值时监听 默认 `null` */
  onOutput?: (...args: any[]) => any;
  /** 禁用回调 默认 `null` */
  onDisabled?: (...args: any[]) => any;
  /** 启用回调 默认 `null` */
  onEnabled?: (...args: any[]) => any;
  /** 全部禁用回调 默认 `null` */
  onDisabledAll?: (...args: any[]) => any;
  /** 全部启用回调 默认 `null` */
  onEnabledAll?: (...args: any[]) => any;
  /** 只读回调 默认 `null` */
  onReadonly?: (...args: any[]) => any;
  /** 全部只读回调 默认 `null` */
  onReadonlyAll?: (...args: any[]) => any;
  /** 全局禁用后执行 默认 `null` */
  onPassivated?: (...args: any[]) => any;
  /** 取消全局禁用后执行 默认 `null` */
  onActivated?: (...args: any[]) => any;
  /** 展开前回调 默认 `null` */
  onExpand?: (...args: any[]) => any;
  /** 展开后回调 默认 `null` */
  onExpanded?: (...args: any[]) => any;
  /** 折叠前回调 默认 `null` */
  onCollapse?: (...args: any[]) => any;
  /** 折叠后回调 默认 `null` */
  onCollapsed?: (...args: any[]) => any;
  /** 全部展开前回调 默认 `null` */
  onExpandAll?: (...args: any[]) => any;
  /** 全部展开后回调 默认 `null` */
  onExpandedAll?: (...args: any[]) => any;
  /** 全部折叠前回调 默认 `null` */
  onCollapsedAll?: (...args: any[]) => any;
  /** 选中回调 默认 `null` */
  onSelected?: (...args: any[]) => any;
  /** 取消选中回调 默认 `null` */
  onDeselected?: (...args: any[]) => any;
  /** 全选回调 默认 `null` */
  onSelectedAll?: (...args: any[]) => any;
  /** 勾选回调 默认 `null` */
  onChecked?: (...args: any[]) => any;
  /** 取消勾选回调 默认 `null` */
  onUnchecked?: (...args: any[]) => any;
  /** 全部勾选回调 默认 `null` */
  onCheckedAll?: (...args: any[]) => any;
  /** 选择过少回调 默认 `null` */
  onTooFew?: (...args: any[]) => any;
  /** 选择过多回调 默认 `null` */
  onTooMany?: (...args: any[]) => any;
  /** 填充完成回调 默认 `null` */
  onFilled?: (...args: any[]) => any;
  /** 翻页回调 默认 `null` */
  onTurned?: (...args: any[]) => any;
  /** 取值回调 默认 `null` */
  onGetted?: (...args: any[]) => any;
  /** 设置值回调 默认 `null` */
  onSet?: (...args: any[]) => any;
  /** 清除值回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 转移节点回调 默认 `null` */
  onGrafted?: (...args: any[]) => any;
  /** 内容更新回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /** 请求完成回调 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /** checked数量太少时监听 默认 `null` */
  onToofew?: (...args: any[]) => any;
  /** checked数量太多时监听 默认 `null` */
  onToomany?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"value": "", "all": false, "only": true, "linkage": false}` */
  expand?: AxTreeExpandOptions;
  /** 默认 `{"enable": true, "only": true, "value": "", "linkage": false, "span": "tree", "addSelected": false}` */
  select?: AxTreeSelectOptions;
  /** 默认 `{"enable": true, "show": "icon-right", "hide": "icon-right", "anim": "rotate90", "type": "icon", "trigger": "click"}` */
  arrow?: AxTreeArrowOptions;
  /** 默认 `{"enable": false, "parent": ["icon-folder", "icon-folder-open"], "child": "icon-file-text", "type": "icon"}` */
  legend?: AxTreeLegendOptions;
  /** 默认 `{"enable": false, "type": "checkbox", "value": "", "min": 0, "max": 0, "sliced": true, "linkage": true, "only": false, "span": "tree"}` */
  check?: AxTreeCheckOptions;
  /** 默认 `{"enable": false, "trigger": "hover", "children": ["folder", "file", "edit", "remove"]}` */
  tools?: AxTreeToolsOptions;
  /** 默认 `{"enable": true, "target": "", "type": "", "from": "selected", "field": "label", "instant": true, "autosort": false}` */
  output?: AxTreeOutputOptions;
  /** 默认 `{"deep": 0, "start": 0, "count": 0, "order": "id asc", "where": ""}` */
  sqlData?: AxTreeSqldataOptions;
  /** 默认 `{"enable": false, "count": 10, "override": false, "exception": false}` */
  paginated?: AxTreePaginatedOptions;
  /** 默认 `{"enable": false, "span": "blank", "mean": "selected"}` */
  shortcut?: AxTreeShortcutOptions;
  /** 默认 `{"target": "", "trigger": "input", "delay": 500, "value": "", "limit": 0, "fuzzy": true, "ignore": true}` */
  search?: AxTreeSearchOptions;
  /** 默认 `{"enable": false, "exclude": ""}` */
  drag?: AxTreeDragOptions;
  /** 默认 `{"attr": "dropnode", "global": false}` */
  drop?: AxTreeDropOptions;
  /** 默认 `null` */
  floorStart?: any;
  /** 默认 `null` */
  pathHyphen?: any;
  /** 默认 `0` */
  duration?: number;
  /** 默认 `{"enable": false, "type": "popup"}` */
  bubble?: Record<string, any>;
  classes?: string;
  /** 默认 `'inline'` */
  display?: string;
  /** 默认 `{"type": "", "expand": {"all": true, "only": false}, "check": {"linkage": true, "only": false, "span": "tree"}, "layout": {"dropdown": "indent|arrow|icon|disk|cube|image|label|holder|tips|badge", "select": "indent|arrow|icon|disk|cube|image|label|holder|tips|badge", "check": "indent|arrow|check|icon|disk|cube|image|label|holder|tips|badge", "button": "icon|arrow|disk|cube|image|label|tips|badge", "list": "indent|arrow|icon|disk|cube|image(label|badge|tips|custom)"}}` */
  feature?: AxTreeFeatureOptions;
  /** 默认 `null` */
  onInsertItem?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInsertItems?: ((...args: any[]) => any);
  /** 默认 `null` */
  onGot?: ((...args: any[]) => any);
  /** 默认 `null` */
  onLocked?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUnlocked?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
  /** popup 配置组（详见 AxTreePopupOptions） */
  popup?: AxTreePopupOptions;
}

/** Tree 的 expand 配置组 */
export interface AxTreeExpandOptions {
  /* ---- 展开配置 ---- */
  /** 初始化展开项 */
  value?: string | number | HTMLElement | Record<string, any> | any[];
  /** 初始化时是否展开全部 默认 `false` */
  all?: boolean;
  /** 是否仅展开一项 默认 `true` */
  only?: boolean;
  /** 是否允许联动父层展开 默认 `false` */
  linkage?: boolean;
}

/** Tree 的 select 配置组 */
export interface AxTreeSelectOptions {
  /* ---- 选择配置 ---- */
  /** 是否可选择分支作为焦点 默认 `true` */
  enable?: boolean;
  /** 是否只能让一个分支成为选择焦点 默认 `true` */
  only?: boolean;
  /** 初始化selected项 */
  value?: string | number | HTMLElement | Record<string, any> | any[];
  /** 是否父子联动selected 默认 `false` */
  linkage?: boolean;
  /** 设置可select的范围 默认 `'tree'` */
  span?: 'tree' | 'leaf' | 'branch';
  /** 是否添加新分支后立即selected高亮 默认 `false` */
  addSelected?: boolean;
}

/** Tree 的 arrow 配置组 */
export interface AxTreeArrowOptions {
  /* ---- 箭头配置 ---- */
  /** 是否启用指示箭头 默认 `true` */
  enable?: boolean;
  /** 展开状态的箭头样式类名 默认 `'_icon-right'` */
  show?: string;
  /** 折叠状态的箭头样式类名 默认 `'_icon-right'` */
  hide?: string;
  /** 箭头的动画样式类 默认 `'_rotate90'` */
  anim?: string;
  /** 图标类型 默认 `'icon'` */
  type?: 'icon' | 'image';
  /** 打开菜单的事件类型 默认 `'click'` */
  trigger?: 'click' | 'hover' | 'none';
}

/** Tree 的 legend 配置组 */
export interface AxTreeLegendOptions {
  /* ---- 图例配置 ---- */
  /** 是否启用节点指示图标 默认 `false` */
  enable?: boolean;
  /** 父节点的图标，分别是未打开和打开状态 默认 `['_icon-folder','_icon-folder-open']` */
  parent?: any[];
  /** 子节点图标 默认 `'icon-file-text'` */
  child?: string;
  /** 图标类型 默认 `'icon'` */
  type?: 'icon' | 'image';
}

/** Tree 的 check 配置组 */
export interface AxTreeCheckOptions {
  /* ---- 检查配置 ---- */
  /** 是否启用checkbox和radio 默认 `false` */
  enable?: boolean;
  /** 节点的check模式 默认 `'checkbox'` */
  type?: 'checkbox' | 'radio' | 'none';
  /** 初始化checked项 */
  value?: string | number | HTMLElement | Record<string, any> | any[];
  /** 可check的最少数量 默认 `0` */
  min?: number;
  /** 可check的最多数量 默认 `0` */
  max?: number;
  /** 超出checked数量是否裁剪 默认 `true` */
  sliced?: boolean;
  /** 父层子层是否checked联动 默认 `true` */
  linkage?: boolean;
  /** 是否只check一个 默认 `false` */
  only?: boolean;
  /** 可check的范围 默认 `'tree'` */
  span?: 'tree' | 'leaf' | 'branch';
}

/** Tree 的 tools 配置组 */
export interface AxTreeToolsOptions {
  /* ---- 工具配置 ---- */
  /** 是否显示编辑工具 默认 `false` */
  enable?: boolean;
  /** 显示工具的触发方式 默认 `'hover'` */
  trigger?: 'hover' | 'click' | 'none';
  /** 工具类型列表 默认 `['close']` */
  children?: any[];
}

/** Tree 的 output 配置组 */
export interface AxTreeOutputOptions {
  /* ---- 输出配置 ---- */
  /** 是否可输出值 默认 `true` */
  enable?: boolean;
  /** 接收值的节点选择器 */
  target?: string;
  /** 输出路径值链接符 默认 `'/'` */
  connector?: string;
  /** 输出多值的分隔符 默认 `','` */
  separator?: string;
  /** 输出值类型 */
  type?: any;
  /** 从哪类原生获得数据源 默认 `'selected'` */
  from?: 'selected' | 'checked';
  /** 取值属性 默认 `'label'` */
  field?: string;
  /** 是否通过select或check分支立即赋值 默认 `true` */
  instant?: boolean;
  /** 是否按tree中节点先后顺序输出 默认 `false` */
  autosort?: boolean;
}

/** Tree 的 sqlData 配置组 */
export interface AxTreeSqldataOptions {
  /* ---- SQL数据配置 ---- */
  /** 是否展开父节点后请求所有子/孙数据 默认 `0` */
  deep?: number;
  /** 请求到的满足条件的数据从第几条开始截取 默认 `0` */
  start?: number;
  /** 一共截取几条 默认 `0` */
  count?: number;
  /** 检索结果按什么字段排序 默认 `'id asc'` */
  order?: string;
  /** 检索条件，写法同sql */
  where?: string;
}

/** Tree 的 paginated 配置组 */
export interface AxTreePaginatedOptions {
  /* ---- 分页配置 ---- */
  /** 是否是可分页的 默认 `false` */
  enable?: boolean;
  /** 每一页最多数量 默认 `10` */
  count?: number;
  /** 是否新页覆盖页码 默认 `false` */
  override?: boolean;
  /** 是否忽略主干分页 默认 `false` */
  exception?: boolean;
}

/** Tree 的 shortcut 配置组 */
export interface AxTreeShortcutOptions {
  /* ---- 快捷操作配置 ---- */
  /** 是否启用快捷操作 默认 `false` */
  enable?: boolean;
  /** 可点击区域 默认 `'blank'` */
  span?: 'blank' | 'whole';
  /** 点击分支行等同于什么 默认 `'selected'` */
  mean?: 'selected' | 'checked';
}

/** Tree 的 search 配置组 */
export interface AxTreeSearchOptions {
  /* ---- 搜索配置 ---- */
  /** 检索input的节点选择器 */
  target?: string;
  /** target触发检索的事件 默认 `'input'` */
  trigger?: 'input' | 'change';
  /** 检索时的防抖延迟 默认 `500` */
  delay?: number;
  /** 初始值 */
  value?: string;
  /** 是否限制检索结果数量 默认 `0` */
  limit?: number;
  /** 是否为模糊检索 默认 `true` */
  fuzzy?: boolean;
  /** 是否忽略大小写 默认 `true` */
  ignore?: boolean;
  /** 是否显示检索结果 默认 `true` */
  result?: boolean;
}

/** Tree 的 drag 配置组 */
export interface AxTreeDragOptions {
  /* ---- 拖拽配置 ---- */
  /** 是否可拖拽 默认 `false` */
  enable?: boolean;
  /** 排除不可拖拽的节点 */
  exclude?: string;
}

/** Tree 的 drop 配置组 */
export interface AxTreeDropOptions {
  /* ---- 拖拽配置 ---- */
  /** 增加可释放节点的属性 默认 `'dropnode'` */
  attr?: string;
  /** 是否全局释放 默认 `false` */
  global?: boolean;
}

/** Tree 的 feature 配置组 */
export interface AxTreeFeatureOptions {
  /* ---- 布局和特性配置 ---- */
  /** 分支的风格特色，可选值：select，check，button，list，'' */
  type?: string;
  /** expand 配置组（详见 AxTreeFeatureExpandOptions） */
  expand?: AxTreeFeatureExpandOptions;
  /** check 配置组（详见 AxTreeFeatureCheckOptions） */
  check?: AxTreeFeatureCheckOptions;
  /** layout 配置组（详见 AxTreeFeatureLayoutOptions） */
  layout?: AxTreeFeatureLayoutOptions;
}

/** Tree 的 popup 配置组 */
export interface AxTreePopupOptions {
  /* ---- 工具配置 ---- */
  /** 是否启用弹出层 默认 `true` */
  enable?: boolean;
}

/** Tree 的 expand 配置组 */
export interface AxTreeFeatureExpandOptions {
  /* ---- 布局和特性配置 ---- */
  /** 是否展开全部 默认 `true` */
  all?: boolean;
  /** 是否只展开一个 默认 `false` */
  only?: boolean;
}

/** Tree 的 check 配置组 */
export interface AxTreeFeatureCheckOptions {
  /* ---- 布局和特性配置 ---- */
  /** 父子是否联动checked 默认 `true` */
  linkage?: boolean;
  /** 是否只check一个 默认 `false` */
  only?: boolean;
  /** 可check的范围，linkage=false下有效 默认 `'tree'` */
  span?: 'tree' | 'leaf' | 'branch';
}

/** Tree 的 layout 配置组 */
export interface AxTreeFeatureLayoutOptions {
  /* ---- 布局和特性配置 ---- */
  /** type=dropdown的分支布局 默认 `'indent | |icon|disk|cube|image|label|holder|tips|badge'` */
  dropdown?: string;
  /** type=select的分支布局 默认 `'indent | arrow |icon|disk|cube|image|label|holder|tips|badge'` */
  select?: string;
  /** type=check的分支布局 默认 `'indent | arrow |check|icon|disk|cube|image|label|holder|tips|badge'` */
  check?: string;
  /** type=button的分支布局 默认 `'icon | arrow |disk|cube|image|label|tips|badge'` */
  button?: string;
  /** type=list的分支布局 默认 `'indent | arrow |icon|disk|cube|image(label|badge|tips|custom)'` */
  list?: string;
}

/** Tree 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxTreeEventName = 'added' | 'checkedAll' | 'collapse' | 'collapsed' | 'constructed' | 'deselectedAll' | 'edited' | 'expand' | 'expanded' | 'filled' | 'got' | 'grafted' | 'initiate' | 'initiated' | 'insertItem' | 'insertItems' | 'output' | 'rechecked' | 'removed' | 'searched' | 'selectedAll' | 'set' | 'stopChecked' | 'stopUnchecked' | 'trigger' | 'turned';

/** Tree 模块实例（new Tree() 的返回值） */
export interface AxTreeInstance extends AxModBase {
  options: AxTreeOptions;
  /** 事件监听（事件名见 AxTreeEventName；回调参数视事件而定） */
  on(type: AxTreeEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  correctOpts(...args: any[]): any;
  setFeature(...args: any[]): any;
  sqlToAdd(...args: any[]): any;
  getDataToRender(...args: any[]): any;
  getTreeFlat(...args: any[]): any;
  getObserver(...args: any[]): any;
  updateSeqItems(...args: any[]): any;
  initExpandeds(...args: any[]): any;
  initSelecteds(...args: any[]): any;
  getSelecteds(...args: any[]): any;
  getParentsGroupBy(...args: any[]): any;
  getIndentHtml(...args: any[]): any;
  updateChildrenFloor(...args: any[]): any;
  renderData(...args: any[]): any;
  updateArrowEl(...args: any[]): any;
  setBrief(...args: any[]): any;
  setLegendVal(...args: any[]): any;
  toggleParentLegend(...args: any[]): any;
  updateLegendEl(...args: any[]): any;
  getLegendEl(...args: any[]): any;
  createHeadEl(...args: any[]): any;
  createBodyEl(...args: any[]): any;
  createFootEl(...args: any[]): any;
  togglePgnDisabled(...args: any[]): any;
  turnNextPage(...args: any[]): any;
  turnFirstPage(...args: any[]): any;
  updateInfoEl(...args: any[]): any;
  createChildrenEl(...args: any[]): any;
  child2Parent(...args: any[]): any;
  parent2Child(...args: any[]): any;
  setAttrs(...args: any[]): any;
  setDragDrop(...args: any[]): any;
  updateExpandEvt(...args: any[]): any;
  updateEvt(...args: any[]): any;
  renderFinish(...args: any[]): any;
  getDrops(...args: any[]): any;
  getReadonly(...args: any[]): any;
  getTriggerEl(...args: any[]): any;
  addTrigger(...args: any[]): any;
  removeTrigger(...args: any[]): any;
  toggleExpanded(...args: any[]): any;
  toggleSelected(...args: any[]): any;
  eachCollapse(...args: any[]): any;
  eachExpand(...args: any[]): any;
  floatDown(...args: any[]): any;
  floatUp(...args: any[]): any;
  eachCheck(...args: any[]): any;
  getCheckType(...args: any[]): any;
  uncheckSibings(...args: any[]): any;
  expandParents(...args: any[]): any;
  select(...args: any[]): any;
  selectAll(...args: any[]): any;
  deselectAll(...args: any[]): any;
  deselect(...args: any[]): any;
  checkOnlyone(...args: any[]): any;
  check(...args: any[]): any;
  checkAll(...args: any[]): any;
  add(...args: any[]): any;
  remove(...args: any[]): any;
  search(...args: any[]): any;
  inputLabel(...args: any[]): any;
  edit(...args: any[]): any;
  graft(...args: any[]): any;
  setSearch(...args: any[]): any;
  setResult(...args: any[]): any;
  updateVals(...args: any[]): any;
  saveRaw(...args: any[]): any;
  updateResult(...args: any[]): any;
  clearResult(...args: any[]): any;
  setVals(...args: any[]): any;
  clearVals(...args: any[]): any;
  getVals(...args: any[]): any;
  getValItems(...args: any[]): any;
  updateCont(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Upload · 文件上传                                                              */
/* 文档: modules/257.html                                                       */
/* ========================================================================== */
/** Upload 构造参数 —— new Upload(elem, options) 的 options */
export interface AxUploadOptions {
  /* ---- 基础配置 ---- */
  /** 表单隐藏字段name */
  name?: string;
  /** 初始文件值 */
  value?: string;
  /** 文件上传地址 */
  url?: string;
  /** 是否禁用 默认 `false` */
  disabled?: boolean;
  /** 是否只读 默认 `false` */
  readonly?: boolean;
  /** 是否多选文件 默认 `true` */
  multiple?: boolean;
  /** 是否手动上传 默认 `false` */
  manual?: boolean;
  /** 是否允许粘贴上传 默认 `true` */
  pastable?: boolean;
  /** card类型一行几列 默认 `4` */
  cols?: number;
  /* ---- 数据源配置 ---- */
  /** 数据源内容 */
  content?: string;
  /** 内容类型 默认 `'text'` */
  contType?: string;
  /** 请求参数 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 限制配置 ---- */
  /** 允许的MIME类型 */
  accept?: string;
  /* ---- 显示配置 ---- */
  /** 列表类型 默认 `'text'` */
  type?: 'text' | 'bullet' | 'cube' | 'card' | 'table';
  /** 固定格式 */
  feature?: string;
  /** 组件尺寸 默认 `'md'` */
  size?: 'sm' | 'md' | 'lg';
  /** 追加样式类 */
  classes?: string;
  /** 状态栏内容 */
  status?: string;
  /* ---- 等待函数 ---- */
  /** 清除前等待 默认 `null` */
  b4Clear?: (...args: any[]) => any;
  /** 删除前等待 默认 `null` */
  b4Remove?: (...args: any[]) => any;
  /** 上传前等待 默认 `null` */
  b4Upload?: (...args: any[]) => any;
  /** 批量上传前等待 默认 `null` */
  b4UploadAll?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 渲染完成回调 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 添加完成回调 默认 `null` */
  onAdded?: (...args: any[]) => any;
  /** 删除完成回调 默认 `null` */
  onRemoved?: (...args: any[]) => any;
  /** 清除完成回调 默认 `null` */
  onCleared?: (...args: any[]) => any;
  /** 粘贴上传回调 默认 `null` */
  onPasted?: (...args: any[]) => any;
  /** 拖拽上传回调 默认 `null` */
  onDropped?: (...args: any[]) => any;
  /** 上传中回调 默认 `null` */
  onUploading?: (...args: any[]) => any;
  /** 上传完成回调 默认 `null` */
  onUploaded?: (...args: any[]) => any;
  /** 服务端接收回调 默认 `null` */
  onReceived?: (...args: any[]) => any;
  /** 批量接收回调 默认 `null` */
  onReceivedAll?: (...args: any[]) => any;
  /** 开始批量上传回调 默认 `null` */
  onUploadAll?: (...args: any[]) => any;
  /** 完成批量上传回调 默认 `null` */
  onUploadedAll?: (...args: any[]) => any;
  /** 上传失败回调 默认 `null` */
  onFailed?: (...args: any[]) => any;
  /** 上传错误回调 默认 `null` */
  onError?: (...args: any[]) => any;
  /** 输出值回调 默认 `null` */
  onOutput?: (...args: any[]) => any;
  /** 内容更新回调 默认 `null` */
  onUpdatedCont?: (...args: any[]) => any;
  /** 请求完成回调 默认 `null` */
  onRequest?: (...args: any[]) => any;
  /** 全局禁用回调 默认 `null` */
  onPassivated?: (...args: any[]) => any;
  /** 取消禁用回调 默认 `null` */
  onActivated?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{}` */
  icons?: Record<string, any>;
  /** 默认 `null` */
  typeGetter?: any;
  /** 默认 `{"min": 0, "max": 0, "size": 0, "suffix": ""}` */
  limit?: AxUploadLimitOptions;
  /** 默认 `{"header": true}` */
  table?: AxUploadTableOptions;
  /** 默认 `{"target": "", "icon": "icon-plus", "attrs": {}}` */
  chooseBtn?: AxUploadChoosebtnOptions;
  /** 默认 `{"enable": false, "target": "", "icon": "icon-upload", "attrs": {}}` */
  uploadBtn?: AxUploadUploadbtnOptions;
  /** 默认 `{"enable": false, "target": "", "icon": "icon-trash", "attrs": {}}` */
  clearBtn?: AxUploadClearbtnOptions;
  /** 默认 `{"icon": "icon-plus", "attrs": {}}` */
  pictureBtn?: Record<string, any>;
  /** 默认 `{"icon": "icon-addpic", "attrs": {}}` */
  galleryBtn?: Record<string, any>;
  /** 默认 `null` */
  separator?: any;
  /** 默认 `null` */
  success?: any;
  /** 默认 `{"enable": false, "field": "", "server": "", "domain": "", "map": {"name": "", "url": "", "size": "", "time": ""}}` */
  cloud?: AxUploadCloudOptions;
  /** 默认 `null` */
  onLocked?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUnlocked?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Upload 的 limit 配置组 */
export interface AxUploadLimitOptions {
  /* ---- 限制配置 ---- */
  /** 最少文件数 默认 `0` */
  min?: number;
  /** 最多文件数 默认 `0` */
  max?: number;
  /** 文件大小限制(MB) 默认 `0` */
  size?: number;
  /** 允许的文件后缀 */
  suffix?: string;
}

/** Upload 的 table 配置组 */
export interface AxUploadTableOptions {
  /* ---- 显示配置 ---- */
  /** 是否显示表头 默认 `true` */
  header?: boolean;
}

/** Upload 的 chooseBtn 配置组 */
export interface AxUploadChoosebtnOptions {
  /* ---- 按钮配置 ---- */
  /** 选择按钮选择器 */
  target?: string;
  /** 选择按钮图标 默认 `'_icon-plus'` */
  icon?: string;
  /** 选择按钮属性 默认 `{}` */
  attrs?: Record<string, any>;
}

/** Upload 的 uploadBtn 配置组 */
export interface AxUploadUploadbtnOptions {
  /* ---- 按钮配置 ---- */
  /** 是否启用上传按钮 默认 `false` */
  enable?: boolean;
  /** 上传按钮选择器 */
  target?: string;
  /** 上传按钮图标 默认 `'_icon-upload'` */
  icon?: string;
  /** 上传按钮属性 默认 `{}` */
  attrs?: Record<string, any>;
}

/** Upload 的 clearBtn 配置组 */
export interface AxUploadClearbtnOptions {
  /* ---- 按钮配置 ---- */
  /** 是否启用清除按钮 默认 `false` */
  enable?: boolean;
  /** 清除按钮选择器 */
  target?: string;
  /** 清除按钮图标 默认 `'_icon-trash'` */
  icon?: string;
  /** 清除按钮属性 默认 `{}` */
  attrs?: Record<string, any>;
}

/** Upload 的 cloud 配置组 */
export interface AxUploadCloudOptions {
  /* ---- 云存储配置 ---- */
  /** 是否启用云存储 默认 `false` */
  enable?: boolean;
  /** 字段名称 */
  field?: string;
  /** 服务器地址 */
  server?: string;
  /** 域名地址 */
  domain?: string;
  /** map 配置组（详见 AxUploadCloudMapOptions） */
  map?: AxUploadCloudMapOptions;
}

/** Upload 的 map 配置组 */
export interface AxUploadCloudMapOptions {
  /* ---- 云存储配置 ---- */
  /** 名称字段映射 */
  name?: string;
  /** 地址字段映射 */
  url?: string;
  /** 大小字段映射 */
  size?: string;
  /** 时间字段映射 */
  time?: string;
}

/** Upload 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxUploadEventName = 'added' | 'changed' | 'constructed' | 'dropped' | 'error' | 'failed' | 'initiate' | 'initiated' | 'output' | 'pasted' | 'received' | 'receivedAll' | 'removed' | 'uploadAll' | 'uploaded' | 'uploadedAll' | 'uploading';

/** Upload 模块实例（new Upload() 的返回值） */
export interface AxUploadInstance extends AxModBase {
  options: AxUploadOptions;
  /** 事件监听（事件名见 AxUploadEventName；回调参数视事件而定） */
  on(type: AxUploadEventName, handler: (...args: any[]) => void): this;
  createSetList(...args: any[]): any;
  init(...args: any[]): any;
  initShow(...args: any[]): any;
  setFeature(...args: any[]): any;
  getFileTypes(...args: any[]): any;
  getTips(...args: any[]): any;
  createStart(...args: any[]): any;
  createBox(...args: any[]): any;
  createBtn(...args: any[]): any;
  createInput(...args: any[]): any;
  createSkeleton(...args: any[]): any;
  addEvts(...args: any[]): any;
  removeEvts(...args: any[]): any;
  renderFinish(...args: any[]): any;
  setAttrs(...args: any[]): any;
  createCtrlv(...args: any[]): any;
  getSummary(...args: any[]): any;
  createItem(...args: any[]): any;
  updateInsProg(...args: any[]): any;
  updateItemProg(...args: any[]): any;
  updateItemValid(...args: any[]): any;
  renderItem(...args: any[]): any;
  getEntries(...args: any[]): any;
  hasSame(...args: any[]): any;
  renderList(...args: any[]): any;
  uploadAll(...args: any[]): any;
  getUploadingItems(...args: any[]): any;
  actOpt(...args: any[]): any;
  canUpload(...args: any[]): any;
  upload(...args: any[]): any;
  listenReceivedAll(...args: any[]): any;
  items2Vals(...args: any[]): any;
  saveRaw(...args: any[]): any;
  setInputVal(...args: any[]): any;
  getFileType(...args: any[]): any;
  fillHeader(...args: any[]): any;
  fileValid(...args: any[]): any;
  globalValid(...args: any[]): any;
  findItem(...args: any[]): any;
  remove(...args: any[]): any;
  clear(...args: any[]): any;
  getVals(...args: any[]): any;
  add(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Valid · 表单校验                                                               */
/* 文档: modules/256.html                                                       */
/* ========================================================================== */
/** Valid 构造参数 —— new Valid(elem, options) 的 options */
export interface AxValidOptions {
  /* ---- 基础配置 ---- */
  /** 表单元素替代名称 */
  label?: string;
  /** 校验文字展示方式 */
  placement?: HTMLElement;
  /** 校验类型，多个用逗号分隔 默认 `'required'` */
  type?: string;
  /** 父form节点选择器 */
  parent?: string;
  /** 多值分隔符 默认 `','` */
  separator?: string;
  /** 消息容器节点名 默认 `'div'` */
  nodeName?: string;
  /** 自定义错误信息格式 */
  fail?: string;
  /** 自定义成功信息格式 */
  succ?: string;
  /** Message模块参数 默认 `{}` */
  message?: Record<string, any>;
  /** Popup模块参数 默认 `{}` */
  popup?: Record<string, any>;
  /* ---- 显示配置 ---- */
  /** 是否显示提示图标 默认 `true` */
  iconShow?: boolean;
  /** 是否对宿主表单域使用样式 默认 `false` */
  styleHost?: boolean;
  /** 校验通过是否显示提示 默认 `true` */
  succShow?: boolean;
  /** 追加样式类 */
  classes?: string;
  /* ---- 校验规则配置 ---- */
  /** 扩展校验类型 默认 `{}` */
  extend?: Record<string, any>;
  /** 本地文字字符正则 */
  regLocal?: string;
  /** 特殊字符正则 */
  regChars?: string;
  /** 强度校验默认字符数量 默认 `6` */
  lengthStr?: number;
  /** 触发校验方式 默认 `'blur'` */
  trigger?: 'blur' | 'change' | 'input' | 'load';
  /** 异步校验参数 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 回调函数 ---- */
  /** 事件触发后回调 默认 `null` */
  onTrigger?: (...args: any[]) => any;
  /** 状态变化后回调 默认 `null` */
  onChanged?: (...args: any[]) => any;
  /** 显示后回调 默认 `null` */
  onShown?: (...args: any[]) => any;
  /** 隐藏后回调 默认 `null` */
  onHidden?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Valid 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxValidEventName = 'changed' | 'constructed' | 'hidden' | 'initiate' | 'initiated' | 'shown' | 'trigger';

/** Valid 模块实例（new Valid() 的返回值） */
export interface AxValidInstance extends AxModBase {
  options: AxValidOptions;
  /** 事件监听（事件名见 AxValidEventName；回调参数视事件而定） */
  on(type: AxValidEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setEmpty(...args: any[]): any;
  loopTestRules(...args: any[]): any;
  loopIncludeRules(...args: any[]): any;
  loopSameRules(...args: any[]): any;
  loopDateRules(...args: any[]): any;
  loopThanRules(...args: any[]): any;
  loopLengthRules(...args: any[]): any;
  loopValuesRules(...args: any[]): any;
  testRegex(...args: any[]): any;
  extendRules(...args: any[]): any;
  getUserRules(...args: any[]): any;
  getVals(...args: any[]): any;
  getStrength(...args: any[]): any;
  toggleStyle(...args: any[]): any;
  createMsgEl(...args: any[]): any;
  createObs(...args: any[]): any;
  pushValids(...args: any[]): any;
  hide(...args: any[]): any;
  show(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Viewer · 媒体查看器                                                             */
/* 文档: modules/427.html                                                       */
/* ========================================================================== */
/** Viewer 构造参数 —— new Viewer(elem, options) 的 options */
export interface AxViewerOptions {
  /* ---- 源码默认参数（文档未单列） ---- */
  target?: string;
  /** 默认 `false` */
  autoIdx?: boolean;
  content?: string;
  /** 默认 `true` */
  gallery?: boolean;
  /** 默认 `false` */
  autoplay?: boolean;
  /** 默认 `{"enable": false, "show": true, "placement": "bottom", "size": 70, "gap": 8, "divide": 1}` */
  thumb?: Record<string, any>;
  /** 默认 `{"enable": false, "expanded": true, "placement": "end", "title": "", "closable": true, "delay": 0, "getter": null, "template": ""}` */
  aside?: Record<string, any>;
  /** 默认 `'fixed'` */
  mode?: string;
  /** 默认 `false` */
  inverted?: boolean;
  /** 默认 `0` */
  width?: number;
  /** 默认 `0` */
  height?: number;
  /** 默认 `{}` */
  swipe?: Record<string, any>;
  /** 默认 `0` */
  active?: number;
  /** 默认 `0` */
  zIndex?: number;
  /** 默认 `{"enable": true, "children": ["close"]}` */
  tools?: Record<string, any>;
  /** 默认 `{"enable": true, "format": ""}` */
  total?: Record<string, any>;
  /** 默认 `null` */
  actClass?: any;
  /** 默认 `'text'` */
  contType?: string;
  /** 默认 `{}` */
  contData?: Record<string, any>;
  /** 默认 `{}` */
  ajax?: Record<string, any>;
  /** 默认 `null` */
  onChanged?: ((...args: any[]) => any);
  /** 默认 `null` */
  onFilledAside?: ((...args: any[]) => any);
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onError?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Viewer 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxViewerEventName = 'changed' | 'constructed' | 'expanded' | 'filledAside' | 'hiddden' | 'hide' | 'initiate' | 'initiated' | 'show' | 'shown';

/** Viewer 模块实例（new Viewer() 的返回值） */
export interface AxViewerInstance extends AxModBase {
  options: AxViewerOptions;
  /** 事件监听（事件名见 AxViewerEventName；回调参数视事件而定） */
  on(type: AxViewerEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setTools(...args: any[]): any;
  toggleMoreAttr(...args: any[]): any;
  toggleScreenAttr(...args: any[]): any;
  togglePlayAttr(...args: any[]): any;
  getCurData(...args: any[]): any;
  fillAside(...args: any[]): any;
  setAsideCont(...args: any[]): any;
  setStageIns(...args: any[]): any;
  setThumbIns(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getStageSlideStr(...args: any[]): any;
  fillStage(...args: any[]): any;
  fillThumb(...args: any[]): any;
  getStructure(...args: any[]): any;
  getDataFromFigure(...args: any[]): any;
  getDataFromStr(...args: any[]): any;
  getStdData(...args: any[]): any;
  show(...args: any[]): any;
  hide(...args: any[]): any;
  expand(...args: any[]): any;
  collapse(...args: any[]): any;
  destroy(...args: any[]): any;
}

/* ========================================================================== */
/* Virtualize · 虚拟列表                                                          */
/* 文档: modules/236.html                                                       */
/* ========================================================================== */
/** Virtualize 构造参数 —— new Virtualize(elem, options) 的 options */
export interface AxVirtualizeOptions {
  /* ---- 基础配置 ---- */
  /** 滚动方向 默认 `'y'` */
  axis?: 'x' | 'y';
  /** 项目初始尺寸(px/rem) 默认 `0` */
  size?: number;
  /** 外溢缓冲项目数量 默认 `2` */
  spill?: number;
  /** 起始渲染位置 默认 `0` */
  index?: number;
  /** 项目尺寸是否可变 默认 `true` */
  dynamic?: boolean;
  /** 追加样式类 */
  classes?: string;
  /* ---- 数据源配置 ---- */
  /** 数据源内容 */
  content?: string;
  /** 内容格式类型 默认 `'text'` */
  contType?: 'text' | 'html' | 'node' | 'image' | 'async';
  /** 请求参数或媒体节点数据 默认 `{}` */
  contData?: Record<string, any>;
  /** 异步请求配置 默认 `{}` */
  ajax?: Record<string, any>;
  /* ---- 模板配置 ---- */
  /** 模板字符串 */
  tplStr?: string;
  /** 模板引擎类型 默认 `null` */
  tplEng?: string | null;
  /* ---- 等待函数 ---- */
  /** 添加节点前等待 默认 `null` */
  b4Append?: (...args: any[]) => any;
  /** 更新项目前等待 默认 `null` */
  b4UpdateItem?: (...args: any[]) => any;
  /** 清空节点前等待 默认 `null` */
  b4Clear?: (...args: any[]) => any;
  /* ---- 回调函数 ---- */
  /** 添加节点后回调 默认 `null` */
  onAppend?: (...args: any[]) => any;
  /** 获取数据时回调 默认 `null` */
  onGetCont?: (...args: any[]) => any;
  /** 所有数据渲染完回调 默认 `null` */
  onExhausted?: (...args: any[]) => any;
  /** 滚动到头部回调 默认 `null` */
  onToStart?: (...args: any[]) => any;
  /** 滚动到尾部回调 默认 `null` */
  onToEnd?: (...args: any[]) => any;
  /** 更新项目时回调 默认 `null` */
  onUpdatedItem?: (...args: any[]) => any;
  /** 渲染完成回调 默认 `null` */
  onRendered?: (...args: any[]) => any;
  /** 请求错误回调 默认 `null` */
  onError?: (...args: any[]) => any;
  /* ---- 源码默认参数（文档未单列） ---- */
  /** 默认 `{"wrap": "div", "list": "div", "cont": "div", "item": "section"}` */
  names?: AxVirtualizeNamesOptions;
  insName?: string;
  storName?: string;
  /** 默认 `[]` */
  storKeys?: any[];
  /** 默认 `{}` */
  breakpoints?: Record<string, any>;
  /** 默认 `false` */
  rtl?: boolean;
  /** 默认 `{}` */
  lang?: Record<string, any>;
  /** 默认 `null` */
  b4Init?: ((...args: any[]) => any);
  /** 默认 `null` */
  onConstructed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiate?: ((...args: any[]) => any);
  /** 默认 `null` */
  onInitiated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onDestroyed?: ((...args: any[]) => any);
  /** 默认 `null` */
  onReset?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdated?: ((...args: any[]) => any);
  /** 默认 `null` */
  onSaved?: ((...args: any[]) => any);
  /** 默认 `null` */
  onClearedCache?: ((...args: any[]) => any);
  /** 默认 `null` */
  onUpdatedCache?: ((...args: any[]) => any);
}

/** Virtualize 的 names 配置组 */
export interface AxVirtualizeNamesOptions {
  /* ---- 节点名称配置 ---- */
  /** 外容器节点名 默认 `'div'` */
  wrap?: string;
  /** 列表节点名 默认 `'div'` */
  list?: string;
  /** 内部容器节点名 默认 `'div'` */
  cont?: string;
  /** 单项节点名 默认 `'section'` */
  item?: string;
}

/** Virtualize 事件名（实例.on(name, cb) / options 中 onXxx 回调） */
export type AxVirtualizeEventName = 'append' | 'constructed' | 'error' | 'exhausted' | 'getCont' | 'initiate' | 'initiated' | 'toEnd' | 'toStart' | 'updateItem';

/** Virtualize 模块实例（new Virtualize() 的返回值） */
export interface AxVirtualizeInstance extends AxModBase {
  options: AxVirtualizeOptions;
  /** 事件监听（事件名见 AxVirtualizeEventName；回调参数视事件而定） */
  on(type: AxVirtualizeEventName, handler: (...args: any[]) => void): this;
  init(...args: any[]): any;
  setAttrs(...args: any[]): any;
  getAvgArr(...args: any[]): any;
  setSizesObs(...args: any[]): any;
  setSpy(...args: any[]): any;
  setNodeMutation(...args: any[]): any;
  getNodesTree(...args: any[]): any;
  getWrapSize(...args: any[]): any;
  get2SetWrapSize(...args: any[]): any;
  getInitData(...args: any[]): any;
  getData(...args: any[]): any;
  getItemNode(...args: any[]): any;
  getDirScrollVal(...args: any[]): any;
  setDirScrollVal(...args: any[]): any;
  renderFromStart(...args: any[]): any;
  fillBlank(...args: any[]): any;
  appendUpdateSize(...args: any[]): any;
  updateSizeOnly(...args: any[]): any;
  renderFromEnd(...args: any[]): any;
  getPadStartFromIdx(...args: any[]): any;
  setScrollVal(...args: any[]): any;
  updateScrollVal(...args: any[]): any;
  compensate(...args: any[]): any;
  get2SetContOffset(...args: any[]): any;
  getContOffset(...args: any[]): any;
  setContOffset(...args: any[]): any;
  scrollTo(...args: any[]): any;
  toStart(...args: any[]): any;
  toEnd(...args: any[]): any;
  append(...args: any[]): any;
  updateItem(...args: any[]): any;
  clear(...args: any[]): any;
  destroy(...args: any[]): any;
}


/* ============================================================================
 * 5. 全局对象 window.ax 与工具函数
 * ========================================================================== */

/** ax.ajax 请求参数（基于 fetch 封装） */
export interface AxAjaxOptions {
  url: string;
  type?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'HEAD' | 'OPTIONS' | string;
  data?: Record<string, any> | string | null;
  headers?: Record<string, string>;
  contentType?: string;
  responseType?: 'json' | 'text' | 'blob' | 'arrayBuffer' | 'formData';
  timeout?: number;
  before?: AxFn;
  after?: AxFn;
  [key: string]: any;
}

/** ax.alert / ax.confirm / ax.notice / ax.prompt 的公共参数 */
export interface AxDialogUtilOptions {
  /** 内容 */
  content: string;
  /** 内容类型，默认 'text' */
  contType?: AxContType;
  /** 请求数据或媒体属性 */
  contData?: Record<string, any>;
  /** 窗口标题 */
  heading?: string;
  /** 自定义模板 */
  tplStr?: string;
  tplEng?: AxFn;
  /** 是否立即显示 */
  init?: boolean;
  /** 确认回调（alert/notice/prompt） */
  yes?: AxFn;
  /** 取消回调（confirm/prompt） */
  no?: AxFn;
  /** 透传给 Dialog 模块的参数（见 AxDialogOptions） */
  dialog?: Partial<AxDialogOptions>;
}

/** ax.prompt 输入参数（在 AxDialogUtilOptions 基础上增加输入框配置） */
export interface AxPromptOptions extends AxDialogUtilOptions {
  label?: string;
  placeholder?: string;
  field?: Partial<AxInputAttributes>;
}

/** localStorage/localStorage 模拟封装（window.ax.storage） */
export interface AxStorage {
  /** 写入；opts: { unit: 'd'|'h'|'m'|'s', expires: number, over: boolean } */
  set(key: string, value: any, opts?: { unit?: string; expires?: number; over?: boolean }): void;
  /** 读取 */
  get(key: string, returnAll?: boolean): any;
  /** 删除 */
  remove(key: string | string[]): void;
  /** 清空 */
  clear(): void;
}

/** 模板引擎：将 `{{prop}}` 占位符替换为 data 值 */
export type AxRenderTpl = (html: string, data: Record<string, any>) => string;

/** window.ax 全局对象 */
export interface AxGlobal {
  /** 样式前缀，默认 'ax' */
  readonly prefix: string;
  /** 组件别名属性名 */
  readonly alias: string;
  /** 屏幕宽度断点代号 */
  readonly screen: string;
  readonly fsRoot: string;
  readonly fullGap: string;
  /** 内置图片 data-url（加载器/占位图等） */
  readonly images: { spin: string; spinDk: string; blank: string;
                     avatar: string; empty: string; none: string };
  /** 是否窄屏（移动端） */
  readonly isNarrowScr: boolean;
  /** 是否触摸屏 */
  readonly isTouchScr: boolean;
  /** 表单即时消息注册表 */
  readonly messages: Record<string, any>;
  /** 校验器注册表 */
  readonly valids: Record<string, any>;
  /** 异步请求封装，返回 Promise */
  ajax(options: AxAjaxOptions): Promise<any>;
  /** 警告窗（基于 Dialog），返回 Promise<boolean> */
  alert(options: AxDialogUtilOptions): Promise<boolean>;
  /** 确认窗（基于 Dialog），返回 Promise<boolean>（yes=true / no=false） */
  confirm(options: AxDialogUtilOptions): Promise<boolean>;
  /** 输入窗（基于 Dialog），返回 Promise<{ type, label, value } | null> */
  prompt(options: AxPromptOptions): Promise<{ type: string; label: string; value: any } | null>;
  /** 通知窗（基于 Dialog） */
  notice(options: AxDialogUtilOptions): Promise<boolean>;
  /** 本地存储封装 */
  storage: AxStorage;
  /** 模板渲染 */
  renderTpl: AxRenderTpl;
  [key: string]: any;
}


/* ============================================================================
 * 6. 全局类型增强（querySelector 自动获得正确类型）
 * ========================================================================== */

declare global {
  interface Window { ax: AxGlobal; }

  interface HTMLElementTagNameMap {
    'ax-accordion': AxAccordionElem;
    'ax-alarm': AxAlarmElem;
    'ax-anchors': AxAnchorsElem;
    'ax-avatar': AxAvatarElem;
    'ax-badge': AxBadgeElem;
    'ax-btn': AxBtnElem;
    'ax-buoy': AxBuoyElem;
    'ax-callout': AxCalloutElem;
    'ax-category': AxCategoryElem;
    'ax-checkbox': AxCheckboxElem;
    'ax-checkboxes': AxCheckboxesElem;
    'ax-color': AxColorElem;
    'ax-datetime': AxDatetimeElem;
    'ax-deform': AxDeformElem;
    'ax-divider': AxDividerElem;
    'ax-editor': AxEditorElem;
    'ax-fields': AxFieldsElem;
    'ax-file': AxFileElem;
    'ax-flag': AxFlagElem;
    'ax-format': AxFormatElem;
    'ax-good': AxGoodElem;
    'ax-icon': AxIconElem;
    'ax-input': AxInputElem;
    'ax-line': AxLineElem;
    'ax-menu': AxMenuElem;
    'ax-more': AxMoreElem;
    'ax-number': AxNumberElem;
    'ax-pagination': AxPaginationElem;
    'ax-pill': AxPillElem;
    'ax-progress': AxProgressElem;
    'ax-radio': AxRadioElem;
    'ax-radios': AxRadiosElem;
    'ax-range': AxRangeElem;
    'ax-rate': AxRateElem;
    'ax-result': AxResultElem;
    'ax-search': AxSearchElem;
    'ax-select': AxSelectElem;
    'ax-skeleton': AxSkeletonElem;
    'ax-stats': AxStatsElem;
    'ax-status': AxStatusElem;
    'ax-step': AxStepElem;
    'ax-textarea': AxTextareaElem;
    'ax-tree': AxTreeElem;
    'ax-twilight': AxTwilightElem;
    'ax-upload': AxUploadElem;
  }

  /* ---- 纯 CSS 载体标签（未注册 WebComponent，仅承载样式） ---- */
  /* <ax-br size="xxs|xs|sm|md|lg|xl|xxl">  上下行文间隙（4/8/14/24/32/40/48px） */
  /* <ax-bc size="xxs|xs|sm|md|lg|xl|xxl">  左右行文间隙                              */
}

/* ============================================================================
 * 7. CSS 工具类速查（ax.css；类名以 _ 开头，详见 native 文档）
 *    说明：以下为常用工具类；ax.css 还包含全部组件样式。
 * ========================================================================== */
/* ---- Background 背景（native/108.html） ---- */
/* [关联类名] */
/*   _bg                          舞台色背景（同stage） */
/*   _bg-prim[-lt]                prim主题背景 */
/*   _bg-error[-lt]               error主题背景 */
/*   _bg-succ[-lt]                succ主题背景 */
/*   _bg-info[-lt]                info主题背景 */
/*   _bg-warn[-lt]                warn主题背景 */
/*   _bg-issue[-lt]               issue主题背景 */
/*   _bg-text[-lt]                text主题背景 */
/*   _bg-brief[-lt]               brief主题背景 */
/*   _bg-caption[-lt]             ignore主题背景 */
/*   _bg-stage                    body背景色（白色） */
/*   _bg-mask                     遮罩色（黑色） */
/*   _bg-white                    白色背景 */
/*   _bg-black                    黑色背景 */

/* ---- Border 边框（native/120.html） ---- */
/* [关联类名] */
/*   _bd                          普通1px实心边框 */
/*   _bd-0                        无边框，border:0 */
/*   _bd-1                        1px边框，border-width:1px，效果同_bd */
/*   _bd-2                        2px边框，border-width:2px */
/*   _bd-3                        3px边框，border-width:3px */
/*   _bd-[4~10]                   4px~10px的边框 */
/*   _bd-s                        左侧实边 */
/*   _bd-e                        右侧实边 */
/*   _bd-t                        上部实边 */
/*   _bd-b                        下部实边 */
/*   _bd-se                       左右实边 */
/*   _bd-tb                       上下实边 */
/*   _bd-solid                    实心边框，效果同_bd */
/*   _bd-dashed                   短横虚线边框 */
/*   _bd-dotted                   小点虚线边框 */
/*   _bd-prim                     prim主题色边框，追加-lt为同色浅色边框 */
/*   _bd-error                    error主题色边框，追加-lt为同色浅色边框 */
/*   _bd-succ                     succ主题色边框，追加-lt为同色浅色边框 */
/*   _bd-info                     info主题色边框，追加-lt为同色浅色边框 */
/*   _bd-warn                     warn主题色边框，追加-lt为同色浅色边框 */
/*   _bd-issue                    issue主题色边框，追加-lt为同色浅色边框 */
/*   _bd-text                     text主题色边框，追加-lt为同色浅色边框 */
/*   _bd-brief                    brief主题色边框，追加-lt为同色浅色边框 */
/*   _bd-caption                  caption主题色边框，追加-lt为同色浅色边框 */

/* ---- Ellipsis 单行和多行省略（native/110.html） ---- */
/* [字号和行高] */
/*   _ell-h6                      通常用于内容简介省略 */
/*   _ell-h5                      通常用于列表省略 */
/*   _ell-h4                      通常用小标题省略 */
/*   _ell-h3                      通常用于板块标题省略 */
/*   _ell-h2                      通常用于板块标题省略 */
/*   _ell-h1                      通常用最大标题省略 */
/* [字号和行高] */
/*   _ell-caption                 通常用于小字省略 */
/*   _ell-body                    通常用于正文省略 */
/*   _ell-lead                    通常用于小标题省略 */
/*   _ell-subtitle                通常用于中标题省略 */
/*   _ell-title                   通常用于大标题省略 */
/*   _ell-hero                    通常用于超大标题省略 */

/* ---- Flex 弹性布局（native/102.html） ---- */
/* [常用类名] */
/*   _row                         水平弹性布局父容器类名。子元素的display样式都将失效。 */
/*   _col                         垂直弹性布局父容器类名。子元素的display样式都将失效。 */
/*   _flex                        弹性布局子容器类名，剩余空间均分。 */
/*   _flex-none                   将自己转为普通的非弹性元素 */
/*   _flex-*                      弹性布局子容器类名，*值范围为0~24，表示占用剩余空间份额。 */
/*   _own-*                       弹性布局子容器类名，*值范围为0~24，表示占用总空间的份额。 */
/*   _avg-*                       弹性布局父容器类名，*值范围为1~24，表示均分父容器为几份。 */
/* [间隙类名] */
/*   _g                           弹性子元素之间有1.4rem间隙(同_g-sm) */
/*   _g-0                         弹性子元素之间没有间隙 */
/*   _g-1                         弹性子元素之间有1px的间隙 */
/*   _g-2                         弹性子元素之间有2px的间隙 */
/*   _g-3                         弹性子元素之间有3px的间隙 */
/*   _g-xxs                       弹性子元素之间有0.4rem的间隙 */
/*   _g-xs                        弹性子元素之间有0.8rem的间隙 */
/*   _g-sm                        弹性子元素之间有1.4rem的间隙 */
/*   _g-md                        弹性子元素之间有2.4rem的间隙 */
/*   _g-lg                        弹性子元素之间有3.2rem的间隙 */
/*   _g-xl                        弹性子元素之间有4rem的间隙 */
/*   _g-xxl                       弹性子元素之间有4.8rem的间隙 */
/* [自适应类名] */
/*   _xxs-*                       手机端中一行（列）显示几个，*的值范围为1~24 */
/*   _xs-*                        普通平板端中一行（列）显示几个，*的值范围为1~24 */
/*   _sm-*                        平板端横屏中一行（列）显示几个，*的值范围为1~24 */
/*   _md-*                        高清平板端中一行（列）显示几个，*的值范围为1~24 */
/*   _lg-*                        桌面电脑端中一行（列）显示几个，*的值范围为1~24 */
/*   _xl-*                        2k桌面电脑端中一行（列）显示几个，*的值范围为1~24 */
/*   _xxl-*                       4k桌面电脑端中一行（列）显示几个，*的值范围为1~24 */
/*   _hh-*                        手持移动设备中一行（列）显示几个，*的值范围为1~24 */
/*   _tb-*                        平板设备中一行（列）显示几个，*的值范围为1~24 */
/*   _dt-*                        桌面设备中一行（列）显示几个，*的值范围为1~24 */
/* [其他类名] */
/*   _clip                        在终端自适应中（使用了_xxs-*~_lg-*）只显示第一行 */

/* ---- Font 字体和字号（native/351.html） ---- */
/* [字体] */
/*   _ff-body                     主正文字体，适用于文章内容、段落、导航、说明文本等 */
/*   _ff-icon                     图标字体专用，例如 UI 图标库、图形字体 */
/*   _ff-alt                      显目数字、大写字母、统计信息、替代用字体等 */
/*   _ff-code                     等宽字体，用于显示代码片段、脚本、终端输出、注释等 */
/* [字重] */
/*   _fw-normal                   正常字重，默认值，适用于正文、说明文本等 */
/*   _fw-bold                     中等粗体，适用于长标题或需要强调的段落文字 */
/*   _fw-bolder                   非常粗，适用于短标题、强调词、Hero 标题等 */
/*   _fw-lighter                  极细字重，适合小标题、展示风格、科技感页面等 */
/* [字重] */
/*   _fw-100                      lighter 范围 */
/*   _fw-200                      lighter 范围 */
/*   _fw-300                      lighter 范围 */
/*   _fw-400                      normal 范围 */
/*   _fw-500                      normal 范围 */
/*   _fw-600                      bold 范围 */
/*   _fw-700                      bold 范围 */
/*   _fw-800                      bolder 范围 */
/*   _fw-900                      bolder 范围 */
/* [字号] */
/*   _fs-caption                  小型字号，用于说明/标签等 */
/*   _fs-body                     正文字号，基础阅读字号 */
/*   _fs-lead                     标题字号或引导段 */
/*   _fs-subtitle                 更大的小标题 */
/*   _fs-title                    新闻/文章主标题（_fs-body ×2） */
/*   _fs-display                  大标题（_fs-lead ×2） */
/*   _fs-hero                     超大标题，用于首页 Hero 区 */
/* [字号] */
/*   _lh-caption                  小型字号的固定行高 */
/*   _lh-body                     正文字号的固定行高 */
/*   _lh-lead                     引导段或小标题的固定行高 */
/*   _lh-subtitle                 更大的小标题固定行高 */
/*   _lh-title                    主标题固定行高 */
/*   _lh-display                  大标题的固定行高 */
/*   _lh-hero                     超大标题的固定行高（Hero 区） */
/* [字号] */
/*   _fs-caption */
/*   _fs-body */
/*   _fs-lead */
/*   _fs-subtitle */
/*   _fs-title */
/*   _fs-display */
/*   _fs-hero */
/* [字号] */
/*   _fs-h1                       1级标题，通常用于页面主标题 （_fs-lead*2） */
/*   _fs-h2                       2级标题，适合文章标题或分区标题 （_fs-body*2） */
/*   _fs-h3                       3级标题，适合小区块或小节标题 */
/*   _fs-h4                       4级标题，适合注释性副标题 */
/*   _fs-h5                       5级标题，接近正文大小，可用于引导 */
/*   _fs-h6                       6级标题，通常用于最小的分组标题 */
/* [字号] */
/*   _lh-h1                       用于 1级标题，紧凑视觉效果，常见于首页主标题 */
/*   _lh-h2                       用于 2级标题，略放松，增强可读性 */
/*   _lh-h3                       用于 3级标题，适用于较长的子标题段落 */
/*   _lh-h4                       用于 4级标题，适中行高，兼顾紧凑与清晰 */
/*   _lh-h5                       用于 5级标题，接近正文风格，清晰舒适 */
/*   _lh-h6                       用于 6级标题，适用于细节注释、小节名等 */
/* [通用行高] */
/*   _lh-sm                       适用于较小、不突出的文本；也适合用于多行大标题，制造紧凑视觉感 */
/*   _lh-md                       适合短小段落、小标题，常用于显眼的两行以内标题文本 */
/*   _lh-lg                       适合大段正文内容，行距较宽松，提升可读性，尤其适合长文阅读体验 */

/* ---- Gradient 渐变背景（native/135.html） ---- */
/* [关联类名] */
/*   _gd                          普通主色渐变 */
/*   _gd-[0~360]                  渐变角度 */
/*   _gd-prim                     prim主题渐变 */
/*   _gd-error                    error主题渐变 */
/*   _gd-succ                     succ主题渐变 */
/*   _gd-info                     info主题渐变 */
/*   _gd-warn                     warn主题渐变 */
/*   _gd-issue                    issue主题渐变 */
/*   _gd-text                     text主题渐变 */

/* ---- Grid 栅格（native/100.html） ---- */
/* [常用类名] */
/*   _grid                        主类名，默认一行显示一列。 */
/*   _avg-*                       参数*可填1~24，表示一行显示多少列。 */
/*   _fence                       模拟表格，最后一行无缺口。 */
/*   _fluid                       模拟表格，最后一行可能有缺口。 */
/* [比例类名] */
/*   _ratio                       宽高比：1:1 */
/*   _ratio-1x1                   宽高比：1:1 */
/*   _ratio-16x9                  宽高比：16:9 */
/*   _ratio-16x10                 宽高比：16:10 */
/*   _ratio-4x3                   宽高比：4:3 */
/*   _ratio-3x2                   宽高比：3:2 */
/*   _ratio-2x1                   宽高比：2:1 */
/*   _ratio-5x4                   宽高比：5:4 */
/* [间隙类名] */
/*   _g                           栅格子元素之间有1.4rem间隙(同_g-sm) */
/*   _g-0                         栅格子元素之间没有间隙 */
/*   _g-1                         栅格子元素之间有1px的间隙 */
/*   _g-2                         栅格子元素之间有2px的间隙 */
/*   _g-3                         栅格子元素之间有3px的间隙 */
/*   _g-xxs                       栅格子元素之间有0.4rem的间隙 */
/*   _g-xs                        栅格子元素之间有0.8rem的间隙 */
/*   _g-sm                        栅格子元素之间有1.4rem的间隙 */
/*   _g-md                        栅格子元素之间有2.4rem的间隙 */
/*   _g-lg                        栅格子元素之间有3.2rem的间隙 */
/*   _g-xl                        栅格子元素之间有4rem的间隙 */
/*   _g-xxl                       栅格子元素之间有4.8rem的间隙 */
/* [自适应类名] */
/*   _xxs-*                       手机端中一行（列）显示几个，*的值范围为1~24 */
/*   _xs-*                        普通平板端中一行（列）显示几个，*的值范围为1~24 */
/*   _sm-*                        平板端横屏中一行（列）显示几个，*的值范围为1~24 */
/*   _md-*                        高清平板端中一行（列）显示几个，*的值范围为1~24 */
/*   _lg-*                        桌面电脑端中一行（列）显示几个，*的值范围为1~24 */
/*   _xl-*                        2k桌面电脑端中一行（列）显示几个，*的值范围为1~24 */
/*   _xxl-*                       4k桌面电脑端中一行（列）显示几个，*的值范围为1~24 */
/*   _hh-*                        手持移动设备中一行（列）显示几个，*的值范围为1~24 */
/*   _tb-*                        平板设备中一行（列）显示几个，*的值范围为1~24 */
/*   _dt-*                        桌面设备中一行（列）显示几个，*的值范围为1~24 */
/* [其他类名] */
/*   _clip                        在终端自适应中（使用了_xxs-*~_lg-*）只显示第一行 */

/* ---- Margin & Padding 外边距和内边距（native/111.html） ---- */
/* [Margin关联类名] */
/*   _m                           默认14px外边距 */
/*   _m-0                         无外边距，margin:0 */
/*   _m-[xxs~xxl]                 外边距尺寸xxs~xxl（4px~44px） */
/*   _m-s                         左侧外边距 */
/*   _m-e                         右侧外边距 */
/*   _m-t                         上方外边距 */
/*   _m-b                         下方外边距 */
/*   _m-se                        左右外边距 */
/*   _m-tb                        上下外边距 */
/* [Padding关联类名] */
/*   _p                           默认14px内边距 */
/*   _p-0                         无内边距，margin:0 */
/*   _p-[xxs~xxl]                 内边距尺寸xxs~xxl（4px~44px） */
/*   _p-s                         左侧内边距 */
/*   _p-e                         右侧内边距 */
/*   _p-t                         上方内边距 */
/*   _p-b                         下方内边距 */
/*   _p-se                        左右内边距 */
/*   _p-tb                        上下内边距 */

/* ---- Radius 圆角（native/105.html） ---- */
/* [关联类名] */
/*   _r                           默认3px圆角 */
/*   _r-0                         无圆角，border-radius:0 */
/*   _r-[1~10]                    _r圆角的1~10倍大小 */
/*   _r-s                         左侧圆角 */
/*   _r-e                         右侧圆角 */
/*   _r-t                         上方圆角 */
/*   _r-b                         下方圆角 */
/*   _r-xxs                       4px圆角 */
/*   _r-xs                        8px圆角 */
/*   _r-sm                        14px圆角 */
/*   _r-md                        24px圆角 */
/*   _r-lg                        32px圆角 */
/*   _r-xl                        40px圆角 */
/*   _r-xxl                       48px圆角 */
/*   _r-input                     6px圆角 */
/*   _r-input-sm                  3px圆角 */
/*   _r-input-lg                  9px圆角 */
/*   _r-bubble                    9px圆角 */
/*   _r-full                      99999px圆角（全圆角） */

/* ---- Shadow 阴影（native/106.html） ---- */
/* [关联类名] */
/*   _sd                          普通阴影 */
/*   _sd-lt                       浅色阴影（lt=light缩写） */
/*   _sd-dk                       深色阴影（dk=dark缩写） */
/*   _sd-soft                     漫射阴影 */
/*   _sd-hard                     突出阴影 */
/*   _sd-bd                       使用阴影制造边框 */
/*   _sd-prim                     prim主题阴影 */
/*   _sd-error                    error主题阴影 */
/*   _sd-succ                     succ主题阴影 */
/*   _sd-info                     info主题阴影 */
/*   _sd-warn                     warn主题阴影 */
/*   _sd-issue                    issue主题阴影 */

/* ---- Width & height元素宽度和高度（native/113.html） ---- */
/* [宽度样式] */
/*   _w-0                         宽0px */
/*   _w-1                         宽4rem(40px) */
/*   _w-2                         宽4rem(40px)*2 */
/*   _w-3                         宽4rem(40px)*3 */
/*   _w-4                         宽4rem(40px)*4 */
/*   _w-5                         宽4rem(40px)*5 */
/*   _w-6                         宽4rem(40px)*6 */
/*   _w-7                         宽4rem(40px)*7 */
/*   _w-8                         宽4rem(40px)*8 */
/*   _w-9                         宽4rem(40px)*9 */
/*   _w-10                        宽4rem(40px)*10 */
/*   _w-neat                      宽4rem(40px)*10 ，与表单域input宽度一致 */
/*   _w-half                      宽50% */
/*   _w-full                      宽100% */
/*   _w-auto                      auto */
/* [高度样式] */
/*   _h-0                         高0px */
/*   _h-1                         高4rem(40px) */
/*   _h-2                         高4rem(40px)*2 */
/*   _h-3                         高4rem(40px)*3 */
/*   _h-4                         高4rem(40px)*4 */
/*   _h-5                         高4rem(40px)*5 */
/*   _h-6                         高4rem(40px)*6 */
/*   _h-7                         高4rem(40px)*7 */
/*   _h-8                         高4rem(40px)*8 */
/*   _h-9                         高4rem(40px)*9 */
/*   _h-10                        高4rem(40px)*10 */
/*   _h-neat                      高4rem(40px) ，与表单域input高度一致 */
/*   _h-half                      高50% */
/*   _h-full                      高100% */
/*   _h-auto                      auto */
