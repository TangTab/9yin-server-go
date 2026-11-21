/* 无头 Chrome CDP 截图 — 验证模型浏览器渲染
 * 用法: node shot.js <url> <out.png> [等待ms] [页面状态JS表达式]
 * 依赖 Node 22+ (内置 fetch/WebSocket), 无需 npm 包 */
import { spawn } from 'node:child_process';
import { writeFileSync, mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const CHROME = process.env.CHROME_EXE || 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const [, , url, out, waitMsArg, stateExpr] = process.argv;
const waitMs = +(waitMsArg || 15000);
if (!url || !out) {
  console.error('usage: node shot.js <url> <out.png> [waitMs] [stateExpr]');
  process.exit(1);
}
const PORT = 9223 + Math.floor(Math.random() * 100);
const profile = mkdtempSync(join(tmpdir(), 'mvshot-'));
const chrome = spawn(CHROME, [
  '--headless=new', '--no-sandbox', '--disable-gpu-sandbox',
  `--remote-debugging-port=${PORT}`,
  `--user-data-dir=${profile}`,
  '--window-size=1400,900', 'about:blank',
], { stdio: 'ignore' });

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function getTarget() {
  for (let i = 0; i < 40; i++) {
    try {
      const list = await (await fetch(`http://127.0.0.1:${PORT}/json/list`)).json();
      const page = list.find((t) => t.type === 'page');
      if (page) return page.webSocketDebuggerUrl;
    } catch { /* retry */ }
    await sleep(250);
  }
  throw new Error('chrome devtools 未就绪');
}

function connect(wsUrl) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(wsUrl);
    let id = 0;
    const pending = new Map();
    ws.onopen = () => resolve({
      send(method, params = {}) {
        return new Promise((res, rej) => {
          const mid = ++id;
          pending.set(mid, { res, rej });
          ws.send(JSON.stringify({ id: mid, method, params }));
        });
      },
      close: () => ws.close(),
    });
    ws.onerror = () => reject(new Error('ws 连接失败'));
    ws.onmessage = (ev) => {
      const msg = JSON.parse(ev.data);
      if (msg.id && pending.has(msg.id)) {
        const p = pending.get(msg.id);
        pending.delete(msg.id);
        if (msg.error) p.rej(new Error(msg.error.message));
        else p.res(msg.result);
      }
    };
  });
}

try {
  const cdp = await connect(await getTarget());
  await cdp.send('Page.enable');
  await cdp.send('Emulation.setDeviceMetricsOverride',
    { width: 1400, height: 900, deviceScaleFactor: 1, mobile: false });
  await cdp.send('Page.navigate', { url });
  await sleep(waitMs);
  let state = null;
  if (stateExpr) {
    try {
      const r = await cdp.send('Runtime.evaluate', { expression: stateExpr, returnByValue: true });
      state = r.result?.value;
    } catch (e) { state = String(e); }
  }
  const shot = await cdp.send('Page.captureScreenshot', { format: 'png' });
  writeFileSync(out, Buffer.from(shot.data, 'base64'));
  console.log('saved', out, state !== null ? 'state=' + JSON.stringify(state) : '');
  cdp.close();
} finally {
  chrome.kill();
}
