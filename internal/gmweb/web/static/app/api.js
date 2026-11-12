// api.js —— 面板全部网络访问的唯一出口（docs/gm-console-axui-redesign-design.md §6）。
// 设计要点：
//   1. 目录请求走并发闸（同时在飞 ≤2），杜绝旧页 18 并发打挂游戏服的加载风暴；
//   2. 目录三层缓存：内存 + sessionStorage（键含 manifest 版本）+ If-None-Match/304
//      协商（P0 落地后端到端生效；旧游戏服无 ETag 时自动退化为纯 TTL 缓存）；
//   3. 会话轮询带 If-None-Match，未变化 304 空体近零成本；
//   4. GM 命令全局互斥（同一时刻仅 1 条在飞）。
// 业务面板不允许直接 fetch。

const CATALOG_TTL = 5 * 60 * 1000; // 304 缺席时的兜底 TTL
const MANIFEST_TTL = 60 * 1000; // manifest 版本静默校验间隔
const MAX_CATALOG_INFLIGHT = 2;

const memCache = new Map(); // key -> {at, data, etag}
const inflight = new Map(); // key -> Promise（去重并发同请求）
let gateQueue = [];
let gateActive = 0;

/* ---------------- manifest（目录版本；旧游戏服无此端点时回退 null） ---------------- */

let manifestData = null;
let manifestAt = 0;
let manifestJob = null;

async function fetchManifest() {
  if (manifestData && Date.now() - manifestAt < MANIFEST_TTL) return manifestData;
  if (manifestJob) return manifestJob;
  manifestJob = fetch("/api/catalog/manifest", { cache: "no-store" })
    .then((response) => (response.ok ? response.json() : null))
    .then((data) => {
      if (data && data.version) {
        if (manifestData && manifestData.version !== data.version) {
          // 游戏服重启/名称缓存重载 → 全部目录缓存失效
          memCache.clear();
        }
        manifestData = data;
        manifestAt = Date.now();
      }
      return manifestData;
    })
    .catch(() => manifestData)
    .finally(() => {
      manifestJob = null;
    });
  return manifestJob;
}

async function catalogVersion() {
  const manifest = await fetchManifest();
  return manifest ? manifest.version : "noversion";
}

/* ---------------- 并发闸 ---------------- */

function acquireGate() {
  return new Promise((resolve) => {
    if (gateActive < MAX_CATALOG_INFLIGHT) {
      gateActive++;
      resolve();
    } else {
      gateQueue.push(resolve);
    }
  });
}

function releaseGate() {
  const next = gateQueue.shift();
  if (next) next();
  else gateActive--;
}

/* ---------------- 缓存（键 = url@版本） ---------------- */

function sessionKey(key) {
  return "gm-ax-cat:" + key;
}

function readEntry(key) {
  return memCache.get(key) || null;
}

function loadEntry(key) {
  const hit = readEntry(key);
  if (hit) return hit;
  try {
    const raw = sessionStorage.getItem(sessionKey(key));
    if (raw) {
      const entry = JSON.parse(raw);
      memCache.set(key, entry);
      return entry;
    }
  } catch {
    /* sessionStorage 不可用（隐私模式等）则只用内存缓存 */
  }
  return null;
}

function saveEntry(key, entry) {
  memCache.set(key, entry);
  try {
    sessionStorage.setItem(sessionKey(key), JSON.stringify(entry));
  } catch {
    /* 超额时静默放弃持久层 */
  }
}

/* ---------------- 目录 GET ---------------- */

/** 目录 GET：并发闸 + 去重 + 版本化双层缓存 + 304 协商。失败抛 Error（中文提示）。 */
export async function catalog(url, { cache = true } = {}) {
  if (!cache) return rawCatalogFetch(url, "");
  const version = await catalogVersion();
  const key = url + "@" + version;
  const entry = loadEntry(key);
  if (entry && Date.now() - entry.at < CATALOG_TTL) return entry.data;
  if (inflight.has(key)) return inflight.get(key);
  const task = (async () => {
    await acquireGate();
    try {
      // TTL 过期但有 ETag → 协商刷新；304 则缓存续期，空体零传输
      const etag = entry && entry.etag ? entry.etag : "";
      const fresh = await rawCatalogFetch(url, etag);
      if (fresh.notModified) {
        const renewed = Object.assign({}, entry, { at: Date.now() });
        saveEntry(key, renewed);
        return renewed.data;
      }
      saveEntry(key, { at: Date.now(), data: fresh.data, etag: fresh.etag });
      return fresh.data;
    } finally {
      releaseGate();
    }
  })();
  inflight.set(key, task);
  try {
    return await task;
  } finally {
    inflight.delete(key);
  }
}

async function rawCatalogFetch(url, etag) {
  const headers = { Accept: "application/json" };
  if (etag) headers["If-None-Match"] = etag;
  const response = await fetch(url, { headers });
  if (response.status === 304) return { notModified: true };
  if (!response.ok) {
    throw new Error(response.status === 503 ? "游戏服不可达（目录暂不可用）" : "目录请求失败（" + response.status + "）");
  }
  return { data: await response.json(), etag: response.headers.get("ETag") || "" };
}

/** 即时搜索（目录快道）：不占并发闸、不缓存、可被 AbortController 取消。 */
export async function search(url, signal) {
  const response = await fetch(url, { signal, headers: { Accept: "application/json" } });
  if (!response.ok) {
    throw new Error(response.status === 503 ? "游戏服不可达（搜索暂不可用）" : "搜索失败（" + response.status + "）");
  }
  return response.json();
}

/* ---------------- 会话快照（ETag 轮询） ---------------- */

let sessionsState = { etag: "", data: null };

/** 会话快照：带 If-None-Match，未变化返回上次数据（304）；不可达返回 null。 */
export async function sessions() {
  try {
    const headers = { Accept: "application/json" };
    if (sessionsState.etag) headers["If-None-Match"] = sessionsState.etag;
    const response = await fetch("/api/sessions", { cache: "no-store", headers });
    if (response.status === 304) {
      return sessionsState.data === null ? [] : sessionsState.data;
    }
    if (!response.ok) return null;
    const data = await response.json();
    sessionsState = { etag: response.headers.get("ETag") || "", data };
    return data;
  } catch {
    return null;
  }
}

/* ---------------- 命令通道：全局互斥 ---------------- */

let cmdInFlight = null;

/** 发 GM 命令。返回 {ok, message}，绝不抛异常。 */
export function command(payload) {
  if (cmdInFlight) {
    return Promise.resolve({ ok: false, message: "上一条命令仍在执行，请稍候" });
  }
  cmdInFlight = (async () => {
    try {
      const list = await sessions();
      if (!list) return { ok: false, message: "游戏服不可达，命令未发送" };
      if (!list.length) return { ok: false, message: "没有在线角色——请先登录游戏" };
      const response = await fetch("/api/sessions/" + list[0].sessionId + "/command", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      let body = {};
      try {
        body = await response.json();
      } catch {
        /* 非 JSON 响应按空体处理 */
      }
      if (!response.ok) {
        if (body.error === "unsupported action") {
          return { ok: false, message: "当前服暂未实现该命令（" + payload.action + "），等待服务端支持" };
        }
        return { ok: false, message: body.error || "命令失败（" + response.status + "）" };
      }
      return { ok: true, message: body.result || body.message || "已发送" };
    } catch (err) {
      return { ok: false, message: "命令发送失败：" + (err && err.message ? err.message : err) };
    } finally {
      cmdInFlight = null;
    }
  })();
  return cmdInFlight;
}

/** 去抖（搜索输入用）。 */
export function debounce(fn, wait = 300) {
  let timer = null;
  return function (...args) {
    clearTimeout(timer);
    timer = setTimeout(() => fn.apply(this, args), wait);
  };
}
