// api.js —— 面板全部网络访问的唯一出口。本地面板（127.0.0.1）免鉴权；
// GET 走 no-store，POST 返回 {ok, message} 语义（失败也绝不抛异常到 UI 层）。

async function getJSON(url) {
  const response = await fetch(url, { cache: "no-store", headers: { Accept: "application/json" } });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(body.error || "请求失败（" + response.status + "）");
  }
  return body;
}

async function postJSON(url, payload = {}) {
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    return { ok: false, message: body.error || "操作失败（" + response.status + "）" };
  }
  return { ok: true, message: (body.messages || [body.message]).filter(Boolean).join("；") };
}

export const overview = () => getJSON("/api/overview");
export const logs = (name, tail) => getJSON("/api/logs?name=" + encodeURIComponent(name) + "&tail=" + tail);
export const players = () => getJSON("/api/players");
export const roles = (account) => getJSON("/api/roles?account=" + encodeURIComponent(account));
export const roleDetail = (account, name, server) =>
  getJSON("/api/role?account=" + encodeURIComponent(account) + "&name=" + encodeURIComponent(name) + "&server=" + server);
export const alerts = (limit) => getJSON("/api/alerts?limit=" + limit);

export const svcStart = (name) => postJSON("/api/svc/start", { name });
export const svcStop = (name) => postJSON("/api/svc/stop", { name });
export const svcRestart = (name) => postJSON("/api/svc/restart", { name });
export const svcStartAll = () => postJSON("/api/svc/startall");
export const svcStopAll = () => postJSON("/api/svc/stopall");

export const gmKick = (name) => postJSON("/api/gm/kick", { name });
export const gmRaw = (line) => postJSON("/api/gm/raw", { line });
export const gmScenes = () => getJSON("/api/gm/scenes");
export const gmBan = (account) => postJSON("/api/gm/ban", { account });
export const gmUnban = (account) => postJSON("/api/gm/unban", { account });
export const worldOpen = () => postJSON("/api/world/open");
export const worldClose = () => postJSON("/api/world/close");
