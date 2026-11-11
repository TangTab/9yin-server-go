/**
 * ============================================================================
 *  AXUI v3 · ax-flag 角标组件（FlagElem）TypeScript 使用文档
 * ----------------------------------------------------------------------------
 *  整理自官方文档 : https://axui.cn/v3/components/131.php
 *  对照本地构建   : ./ax.esm.js  （class FlagElem extends CompBaseComm）
 *  样式依赖       : ./ax.css     （必须一同引入，否则无样式）
 *
 *  组件形态 : Web Component（自定义元素 ax-flag），悬浮于父元素角部。
 *  定位要求 : 父元素需 position: relative / absolute / fixed。
 *
 * 【快速上手】
 *  1. 页面引入（ESM 导入即自动执行 customElements.define('ax-flag', FlagElem)）:
 *
 *       <link rel="stylesheet" href="ax.css">
 *       <script type="module" src="ax.esm.js"></script>
 *
 *  2. HTML 使用:
 *
 *       <div style="position:relative">
 *         <img src="cover.jpg" alt="">
 *         <ax-flag type="ribbon" theme="error" placement="right" label="新品"></ax-flag>
 *       </div>
 *
 *  3. TS/JS 使用（本文档类型即为此服务）:
 *
 *       const flag = document.querySelector('ax-flag');   // 类型自动为 AxFlagElem
 *       flag?.set({ label: '推荐', theme: 'succ' });
 *       flag?.on('set', (resp) => console.log('已设置:', resp));
 */

/* ============================================================================
 * 1. 属性（Attributes）取值类型
 * ========================================================================== */

/** 角标类型（官方文档 Attributes 表） */
export type AxFlagType =
  | 'triangle' // 三角（默认）
  | 'circle'   // 圆形
  | 'horn'     // 羊角书签
  | 'ribbon';  // 绸带

/** 颜色主题（官方文档 Attributes 表） */
export type AxFlagTheme =
  | 'prim'    // 主色（默认）
  | 'error'   // 错误/危险
  | 'succ'    // 成功
  | 'info'    // 信息
  | 'warn'    // 警告
  | 'issue'   // 议题
  | 'text'    // 文本色
  | 'caption';// 说明文字色

/** 角标位置（官方文档 Attributes 表） */
export type AxFlagPlacement =
  | 'left'   // 左侧（默认）
  | 'right'; // 右侧

/* ============================================================================
 * 2. HTML 属性接口
 *    —— 全部可作为 <ax-flag> 上的字面属性使用；改动会被
 *       attributeChangedCallback 监听并即时生效（observedAttributes）。
 * ========================================================================== */

export interface AxFlagAttributes {
  /** 角标文本；缺省时回退取元素原始 innerHTML（源码 getRawData） */
  label?: string;
  /** 角标类型，默认 'triangle' */
  type?: AxFlagType;
  /** 颜色主题，默认 'prim' */
  theme?: AxFlagTheme;
  /** 角标位置，默认 'left' */
  placement?: AxFlagPlacement;

  /* ---- 以下为基类 CompBase 通用属性（源码 baseAttrs / jsonAttrs）---- */

  /** 以 HTML 属性形式注册事件回调，值为字符串函数体，如 on-set="console.log" */
  'on-connected'?: string;
  'on-disconnected'?: string;
  'on-adopted'?: string;
  'on-reset'?: string;
  'on-set'?: string;
  /** 语言包 JSON（基类 jsonAttr，flag 本身未消费，预留） */
  lang?: string;
}

/* ============================================================================
 * 3. 响应式属性对象（propsProxy）
 *    —— setAttribute / set() 改动属性后，组件内部经由 Observe 代理同步；
 *       label 的变更通过 changedMaps.label 即时刷新 labelEl.innerHTML。
 * ========================================================================== */

export interface AxFlagProperties {
  label: string;
  type: AxFlagType;
  theme: AxFlagTheme;
  placement: AxFlagPlacement;
}

/* ============================================================================
 * 4. set() 参数（源码 CompBaseComm.set → setAttrs 实测支持四种形态）
 * ========================================================================== */

/** 单个键值对 */
export interface AxFlagSetPair {
  key: keyof AxFlagAttributes;
  value: string;
}

/** 官方文档口径：键值对 {key,value} 或其数组；源码另支持普通对象与 JSON 字符串 */
export type AxFlagSetData =
  | AxFlagSetPair              // { key: 'theme', value: 'warn' }
  | AxFlagSetPair[]            // [{ key: 'theme', value: 'warn' }, ...]
  | Partial<AxFlagAttributes>  // { theme: 'warn', label: '热销' }
  | string;                    // '{"theme":"warn"}'（JSON 字符串）

/* ============================================================================
 * 5. 事件（Events）载荷
 *    —— 通过 elem.on(name, cb) 注册，或 HTML 属性 on-<name> 注册。
 * ========================================================================== */

export interface AxFlagSetResultItem extends AxFlagSetPair {
  /** setAttr 写入结果明细（ax.esm.js setAttrs 返回，已过滤空项） */
  [extra: string]: unknown;
}

export interface AxFlagEventMap {
  /** 组件连接成功（源码传参：组件实例引用 this.ins） */
  connected: (instance?: unknown) => void;
  /** 组件断开连接 */
  disconnected: () => void;
  /** 组件被转移至其他文档 */
  adopted: () => void;
  /** set() 完成；参数为实际生效项数组（空数据不触发） */
  set: (result: AxFlagSetResultItem[]) => void;
  /** clear() 完成；参数为被清除的属性名 */
  cleared: (attr: string) => void;
  /** reset() 完成 */
  reset: () => void;
}

export type AxFlagEventName = keyof AxFlagEventMap;

/* ============================================================================
 * 6. 组件实例接口
 *    —— ax.esm.js 无类型声明，此接口即 <ax-flag> 元素的 TS 门面；
 *       末尾的 HTMLElementTagNameMap 增强让 querySelector('ax-flag')
 *       自动获得本类型。
 * ========================================================================== */

export interface AxFlagElem extends HTMLElement {
  /* ---------- 方法（官方文档 Methods + 源码 CompBaseComm）---------- */

  /** 设置属性；触发 'set' 事件；空数据直接返回 */
  set(data: AxFlagSetData): void;

  /**
   * 重置组件：恢复初始属性并重渲染；触发 'reset' 事件。
   * 官方示例：CompElem.reset();
   */
  reset(): void;

  /**
   * 清除指定 HTML 属性；触发 'cleared' 事件（参数为属性名）。
   * 源码默认值为 'value'（flag 组件无该属性，实际使用建议显式传 'label' 等）。
   */
  clear(attr?: string): void;

  /* ---------- 事件三件套（源码 ModBaseListen）---------- */

  /** 注册事件监听；官方示例：CompElem.on('connected', () => {})；返回 this 可链式 */
  on<K extends AxFlagEventName>(type: K, handler: AxFlagEventMap[K]): this;
  on(type: string, handler: (...args: unknown[]) => void): this;

  /** 主动触发事件（一般用不到） */
  emit<K extends AxFlagEventName>(type: K, ...params: Parameters<AxFlagEventMap[K]>): this;

  /** 注销事件监听 */
  off<K extends AxFlagEventName>(type: K, handler: AxFlagEventMap[K]): this;

  /* ---------- 内部结构（源码字段，只读参考，勿直接改写）---------- */

  /** 响应式属性代理（读值用，写值请走 set()/setAttribute） */
  readonly propsProxy: AxFlagProperties;
  /** 首次连接时的原始属性快照（reset 的回退基准） */
  readonly propsRaw: Readonly<AxFlagAttributes>;
  /** 文本节点 <i>（渲染结构: span[wrap] > i[label]） */
  readonly labelEl: HTMLElement;
  /** 外层容器 span[ax-alias="wrap"] */
  readonly wrapEl: HTMLSpanElement;
}

/* 让 document.querySelector / getElementById 等拿到正确类型 */
declare global {
  interface HTMLElementTagNameMap {
    'ax-flag': AxFlagElem;
  }
}

/* ============================================================================
 * 7. 完整用法示例
 * ========================================================================== */

/**
 * 示例 1 —— 四种角标形态:
 *
 *   <ax-flag label="1"></ax-flag>                                  <!-- 三角 -->
 *   <ax-flag type="circle" theme="error" label="99+"></ax-flag>    <!-- 圆形 -->
 *   <ax-flag type="horn"   theme="warn"  label="顶"></ax-flag>     <!-- 羊角 -->
 *   <ax-flag type="ribbon" theme="succ"  placement="right"
 *            label="已完成"></ax-flag>                              <!-- 绸带·右侧 -->
 */

/**
 * 示例 2 —— 方法:
 *
 *   const flag = document.querySelector('ax-flag');
 *   flag?.set({ key: 'theme', value: 'error' });   // 官方口径：键值对
 *   flag?.set([{ key: 'theme', value: 'error' },
 *              { key: 'label', value: '爆' }]);    // 键值对数组
 *   flag?.set({ theme: 'error' });                 // 普通对象（源码支持）
 *   flag?.set('{"theme":"error"}');                // JSON 字符串（源码支持）
 *   flag?.clear('label');                          // 清除文本
 *   flag?.reset();                                 // 恢复初始属性并重渲染
 */

/**
 * 示例 3 —— 事件:
 *
 *   const flag = document.querySelector('ax-flag');
 *   flag?.on('connected', () => console.log('connected'));
 *   flag?.on('set',      (resp) => console.log('set:', resp));
 *   flag?.on('cleared',  (attr) => console.log('cleared:', attr));
 *   flag?.on('reset',    () => console.log('reset'));
 *
 *   // 或 HTML 属性回调形式（值为字符串函数体）:
 *   // <ax-flag label="热" on-set="console.log(arguments[0])"></ax-flag>
 */

/**
 * 示例 4 —— ESM 按需导入（拿类本身，如自定义继承/类型引用）:
 *
 *   import { FlagElem } from './ax.esm.js';
 *   console.log(FlagElem.observedAttributes);
 *   // → ['label', 'type', 'theme', 'placement', 'on-connected', ...]
 */
