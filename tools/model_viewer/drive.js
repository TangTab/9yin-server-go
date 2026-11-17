/* 无头 Chrome CDP 驱动 — 逐步执行页面表达式后截图 (渲染问题分步诊断)
 * 用法: node drive.js <url> <out.png> '<expr1>' '<expr2>' ...
 * 每个表达式 awaitPromise 执行; 最后统一截图并回传 JSON 汇总 */
import { spawn } from 'node:child_process';
import { writeFileSync, mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const CHROME = process.env.CHROME_EXE || 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe';
const [, , url, out, ...steps] = process.argv;
if (!url || !out) {
  console.error('usage: node drive.js <url> <out.png> [expr...]');
  process.exit(1);
}
const PORT = 9323 + Math.floor(Math.random() * 100);
const profile = mkdtempSync(join(tmpdir(), 'mvdrive-'));
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

async function evalExpr(cdp, expr) {
  const r = await cdp.send('Runtime.evaluate', {
    expression: expr, awaitPromise: true, returnByValue: true,
  });
  if (r.exceptionDetails) {
    return { __error: r.exceptionDetails.exception?.description || r.exceptionDetails.text };
  }
  return r.result?.value;
}

try {
  const cdp = await connect(await getTarget());
  await cdp.send('Page.enable');
  await cdp.send('Emulation.setDeviceMetricsOverride',
    { width: 1400, height: 900, deviceScaleFactor: 1, mobile: false });
  await cdp.send('Page.navigate', { url });
  // 等 __test 钩子就绪 (目录+默认场景载完)
  for (let i = 0; i < 120; i++) {
    const ok = await evalExpr(cdp, '!!window.__test && !!window.__sil').catch(() => false);
    if (ok === true) break;
    await sleep(500);
  }
  const results = [];
  for (const s of steps) {
    results.push(await evalExpr(cdp, s));
  }
  await sleep(300);
  const shot = await cdp.send('Page.captureScreenshot', { format: 'png' });
  writeFileSync(out, Buffer.from(shot.data, 'base64'));
  console.log('saved', out);
  for (const r of results) console.log(JSON.stringify(r));
  cdp.close();
} finally {
  chrome.kill();
}
