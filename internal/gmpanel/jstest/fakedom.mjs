// fakedom.mjs —— 前端子页逻辑回归共用的最小假 DOM（scene_subpage.mjs /
// debug_subpage.mjs 由 Go 测试 jstest_test.go 在 Node 可用时调起）。
// installDOM() 挂 globalThis.document；El 只实现 panels/test.js 用到的面
//（属性 / 类名 / 子树 / 事件 / 取值），不带样式与布局。

export class Text {
  constructor(text) {
    this.nodeType = 3;
    this.text = String(text);
  }
}

export class El {
  constructor(tag) {
    this.nodeType = 1;
    this.tagName = tag.toUpperCase();
    this.children = [];
    this.attrs = {};
    this.listeners = {};
    this.style = {};
    this._text = "";
    this._class = "";
    this._props = {};
    this._forcedValue = undefined;
  }
  get className() {
    return this._class;
  }
  set className(v) {
    this._class = v;
  }
  setAttribute(k, v) {
    this.attrs[k] = String(v);
  }
  getAttribute(k) {
    return this.attrs[k];
  }
  appendChild(child) {
    this.children.push(child);
    child.parent = this;
    return child;
  }
  get textContent() {
    return this._text + this.children.map((c) => (c.nodeType === 3 ? c.text : c.textContent)).join("");
  }
  set textContent(v) {
    this._text = String(v);
    this.children = [];
    this._forcedValue = undefined;
  }
  set innerHTML(v) {
    this.textContent = v;
  }
  addEventListener(type, fn) {
    (this.listeners[type] ||= []).push(fn);
  }
  fire(type) {
    for (const fn of this.listeners[type] || []) fn({ target: this });
  }
  // <select>：显式设值则选中匹配 option（无匹配清空），否则跟随首个 option。
  get value() {
    if (this.tagName !== "SELECT") return this._props.value ?? this.attrs.value ?? "";
    if (this._forcedValue !== undefined) return this._forcedValue;
    const first = this.options()[0];
    return first ? first.attrs.value : "";
  }
  set value(v) {
    if (this.tagName === "SELECT") {
      const has = this.options().some((o) => o.attrs.value === String(v));
      this._forcedValue = has ? String(v) : "";
      return;
    }
    this._props.value = String(v);
  }
  options() {
    const out = [];
    const walk = (n) => {
      for (const c of n.children) {
        if (c.nodeType === 3) continue;
        if (c.tagName === "OPTION") out.push(c);
        walk(c);
      }
    };
    walk(this);
    return out;
  }
  find(pred) {
    let hit = null;
    const walk = (n) => {
      for (const c of n.children) {
        if (c.nodeType === 3 || hit) continue;
        if (pred(c)) hit = c;
        walk(c);
      }
    };
    walk(this);
    return hit;
  }
  collect(tag, out = []) {
    const walk = (n) => {
      for (const c of n.children) {
        if (c.nodeType === 3) continue;
        if (c.tagName === tag) out.push(c);
        walk(c);
      }
    };
    walk(this);
    return out;
  }
}

export function installDOM() {
  globalThis.document = {
    createElement: (tag) => new El(tag),
    createTextNode: (text) => new Text(text),
    querySelector: () => null,
    contains: () => true,
  };
}
