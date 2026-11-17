/* 九阴角色模型浏览器 — three.js 前端
 * 数据流: /api/catalog → 列表; /api/model → SkinnedMesh; /api/anim → 骨架+轨道
 * 装配: 多部件 (身体/脸/发型/头饰…) 共享同一骨架, 各自 xmod 按骨名绑定;
 *       性别总标签 (男/女) 驱动部件过滤 + 默认骨架 + 默认身体。
 * 蒙皮: 动画文件内嵌 bind 变换 → three.Skeleton (bind 时自动取逆);
 *       每帧按轨道 slerp/lerp 驱动骨骼局部变换。
 */
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { DDSLoader } from 'three/addons/loaders/DDSLoader.js';

/* ================= 状态 ================= */
const S = {
  catalog: null,
  parts: new Map(),     // id → {data, meshes[]} 已装配部件 (共享骨架)
  gender: '男',         // 总标签: 驱动部件/动画过滤与默认装配
  anim: null,           // /api/anim 载荷
  rig: null,            // {bones[], root, skeleton, matched, meshBones}
  playing: false,
  frame: 0,
  speed: 1,
  loop: true,
  showTexture: true,
  bindOnly: false,      // true = 只装配骨架停在绑定姿势, 不应用轨道
};

/* 性别默认身体+脸型 (总标签切换 / 初始装配) */
const DEFAULT_BODY = {
  '男': 'b_beiyuan06/b_beiyuan06.xmod',
  '女': 'g_beiyuan06/g_beiyuan06.xmod',
};
const DEFAULT_FACE = {
  '男': 'b_face/b_face.xmod',
  '女': 'g_face/g_face.xmod',
};

/* 互斥槽位: 同槽装配新部件自动替换旧件 (武器/发型/胡子/脸/面饰/面具/头饰/衣/裤/鞋/披风 各一件;
   未知类别按类别名互斥). 时装按类型 (上衣/下装/鞋…) 各占一槽. */
function slotOf(m) {
  const cat = m.category || '';
  const stem = (m.name || '').toLowerCase();
  if (cat.startsWith('武器·')) return 'weapon';
  if (cat === '发型') return 'hair';
  if (cat === '特殊脸(s_face)' || cat === '蒙面头') return 'face';
  if (cat === '脸型/胡子') return stem.includes('huzi') ? 'beard' : 'face';
  if (cat === '脸型') return 'face';
  if (cat === '脸部饰件') return 'facecloth';
  if (cat === '面具') return 'mask';
  if (cat === '头饰') return 'helmet';
  if (cat === '身体参考' || cat === '身体/套装') return 'body';
  if (cat === '上衣/时装') return 'top';
  if (cat === '下装') return 'pants';
  if (cat === '鞋') return 'shoes';
  if (cat === '披风') return 'cloak';
  return cat || 'misc';
}

const $ = (id) => document.getElementById(id);

/* ================= three 场景 ================= */
const viewport = $('viewport');
const renderer = new THREE.WebGLRenderer({ antialias: true });
renderer.outputColorSpace = THREE.SRGBColorSpace;
viewport.appendChild(renderer.domElement);

const scene = new THREE.Scene();
scene.background = new THREE.Color(0x14161a);
scene.fog = null;

const camera = new THREE.PerspectiveCamera(50, 1, 0.01, 200);
camera.position.set(1.8, 1.5, 2.6);
const controls = new OrbitControls(camera, renderer.domElement);
controls.target.set(0, 1.0, 0);
controls.enableDamping = true;

scene.add(new THREE.HemisphereLight(0xffffff, 0x444466, 1.5));
const sun = new THREE.DirectionalLight(0xffffff, 1.6);
sun.position.set(2, 4, 3);
scene.add(sun);
const sun2 = new THREE.DirectionalLight(0xaabbff, 0.6);
sun2.position.set(-3, 2, -2);
scene.add(sun2);

const grid = new THREE.GridHelper(10, 40, 0x3a4050, 0x262b34);
grid.position.y = 0;
scene.add(grid);

let skeletonHelper = null;

function resize() {
  const w = viewport.clientWidth, h = viewport.clientHeight;
  renderer.setSize(w, h, false);
  renderer.setPixelRatio(window.devicePixelRatio);
  camera.aspect = w / h;
  camera.updateProjectionMatrix();
}
new ResizeObserver(resize).observe(viewport);
resize();

/* ================= 目录载入 ================= */
async function fetchJSON(url) {
  const r = await fetch(url);
  if (!r.ok) throw new Error((await r.json().catch(() => ({}))).error || r.statusText);
  return r.json();
}

async function loadCatalog() {
  $('lib-info').textContent = '加载目录… (首次启动需全量解析, 约 4 分钟, 请勿关闭页面)';
  try {
    S.catalog = await fetchJSON('/api/catalog');
  } catch (e) {
    $('lib-info').textContent = `目录加载失败: ${e.message} — 服务未启动或已退出, 重启后刷新本页`;
    throw e;
  }
  const c = S.catalog.counts;
  $('lib-info').textContent = '';
  $('anim-count').textContent = `${c.animsOk}/${c.anims}`;
  buildModelGroups();
  buildAnimList();
  await setGender('男', true);
}

// 调试/自动化: 目录+默认装配就绪信号
window.__ready = loadCatalog();

/* ---------- 性别总标签 ---------- */
async function setGender(g, initial = false) {
  if (S.gender === g && !initial) return;
  S.gender = g;
  document.querySelectorAll('.gtab').forEach((b) =>
    b.classList.toggle('active', b.dataset.gender === g));
  buildModelGroups();        // 重建分组数据 (M.groups 按性别过滤) — 只 renderModelList 的话列表还是旧性别
  refreshAnimList();
  // 卸下异性别部件 (武器不分性别, 保留); body/face 槽缺省时自动装配默认件
  for (const [id, p] of [...S.parts]) {
    if (id.startsWith('weapon/')) continue;
    if (p.cat.gender !== g) removePart(id);
  }
  const slots = new Set([...S.parts.values()].map((p) => p.slot));
  if (!slots.has('body')) {
    const body = S.catalog.models.find((m) => m.id === DEFAULT_BODY[g] && m.status === 'ok');
    if (body) await addPart(body);
  }
  if (!slots.has('face')) {
    const face = S.catalog.models.find((m) => m.id === DEFAULT_FACE[g] && m.status === 'ok');
    if (face) await addPart(face);
  }
  if (!S.anim) await loadDefaultSkeleton();
  else attachAnim(S.anim);
  frameCamera();
  updateModelHud();
}

/* 性别默认骨架 (无动画时 bind 展示) */
async function loadDefaultSkeleton() {
  const tpose = S.gender === '男' ? 'b_action/tpose.xskt' : 'g_action/tpose.xskt';
  const entry = S.catalog.anims.find((a) => a.id === tpose);
  if (entry) await selectAnim(entry.id, { bindOnly: true });
}

/* ---------- 模型列表 (分组折叠 + 搜索; 点击条目装配/卸下) ---------- */
const M = { groups: [], expanded: new Set() };
const W = { groups: [], expanded: new Set() };   // 武器 (不受性别标签影响)

const isWeapon = (m) => m.id.startsWith('weapon/');
const dispName = (m) => m.label || m.name;

function modelMatches(m, q) {
  if (isWeapon(m)) return false;                 // 武器走独立面板
  if (m.gender !== S.gender && m.gender !== '通用') return false;
  if (!q) return true;
  return m.name.toLowerCase().includes(q) || m.id.toLowerCase().includes(q)
    || (m.label || '').toLowerCase().includes(q);
}

function buildModelGroups() {
  const groups = new Map();
  for (const m of S.catalog.models) {
    if (isWeapon(m) || (m.gender !== S.gender && m.gender !== '通用')) continue;
    if (!groups.has(m.category)) groups.set(m.category, []);
    groups.get(m.category).push(m);
  }
  M.groups = [...groups.entries()].sort((a, b) => a[0].localeCompare(b[0], 'zh'));
  M.expanded = new Set(M.groups.length <= 8 ? M.groups.map(([k]) => k) : []);
  buildWeaponGroups();
  renderModelList();
  renderWeaponList();
}

/* ---------- 武器面板 ---------- */
function buildWeaponGroups() {
  const groups = new Map();
  let n = 0;
  for (const m of S.catalog.models) {
    if (!isWeapon(m)) continue;
    const key = m.category.replace('武器·', '') || '其它';
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(m);
    n++;
  }
  W.groups = [...groups.entries()].sort((a, b) => b[1].length - a[1].length);
  $('weapon-count').textContent = `${n}`;
}

function renderWeaponList() {
  const q = $('weapon-search').value.trim().toLowerCase();
  const searching = !!q;
  const box = $('weapon-groups');
  box.innerHTML = '';
  for (const [key, items] of W.groups) {
    const vis = searching
      ? items.filter((m) => m.name.toLowerCase().includes(q)
        || m.id.toLowerCase().includes(q) || (m.label || '').toLowerCase().includes(q))
      : items;
    if (searching && !vis.length) continue;
    const open = searching || W.expanded.has(key);
    const h = document.createElement('div');
    h.className = 'mgroup-header' + (open ? ' open' : '');
    h.innerHTML = `<span class="tw">${open ? '▾' : '▸'}</span>${key}
      <span class="cnt">${vis.length}</span>`;
    h.onclick = () => {
      if (W.expanded.has(key)) W.expanded.delete(key); else W.expanded.add(key);
      renderWeaponList();
    };
    box.appendChild(h);
    if (!open) continue;
    const cap = 200;
    for (const m of vis.slice(0, cap)) {
      box.appendChild(weaponItem(m));
    }
    if (vis.length > cap) {
      const more = document.createElement('div');
      more.className = 'mgroup-more';
      more.textContent = `… 其余 ${vis.length - cap} 项请用搜索定位`;
      box.appendChild(more);
    }
  }
}

function weaponItem(m) {
  const el = document.createElement('div');
  el.className = 'mitem';
  el.dataset.id = m.id;
  const badge = m.status !== 'ok'
    ? `<span class="badge unsup" title="${m.error || ''}">${m.status === 'unsupported' ? '版本' : '错误'}</span>` : '';
  const on = S.parts.has(m.id) ? '<span class="on">✓</span>' : '';
  el.innerHTML = `${on}<span class="nm">${m.label
    ? `<span class="lbl">${m.label}</span><span class="orig">${m.name}</span>`
    : m.name}</span>${badge}
    <span class="meta">${m.vertices}v</span>`;
  el.title = `${m.label || ''}\n${m.id}`;
  el.classList.toggle('active', S.parts.has(m.id));
  el.onclick = () => togglePart(m.id);
  return el;
}

function renderModelList() {
  if (!S.catalog) return;
  const q = $('model-search').value.trim().toLowerCase();
  const searching = !!q;
  const box = $('model-groups');
  box.innerHTML = '';
  let shown = 0;
  for (const [key, items] of M.groups) {
    const vis = searching ? items.filter((m) => modelMatches(m, q)) : items;
    if (searching && !vis.length) continue;
    const open = searching || M.expanded.has(key);
    const h = document.createElement('div');
    h.className = 'mgroup-header' + (open ? ' open' : '');
    h.innerHTML = `<span class="tw">${open ? '▾' : '▸'}</span>${key}
      <span class="cnt">${vis.length}${vis.length !== items.length ? `/${items.length}` : ''}</span>`;
    h.onclick = () => {
      if (M.expanded.has(key)) M.expanded.delete(key); else M.expanded.add(key);
      renderModelList();
    };
    box.appendChild(h);
    if (!open) continue;
    // 大组内截断显示 (搜索时放行全部命中)
    const cap = searching ? Infinity : 400;
    for (const m of vis.slice(0, cap)) {
      const el = document.createElement('div');
      el.className = 'mitem';
      el.dataset.id = m.id;
      const badge = m.status !== 'ok'
        ? `<span class="badge unsup" title="${m.error || ''}">${m.status === 'unsupported' ? '版本' : '错误'}</span>` : '';
      const on = S.parts.has(m.id) ? '<span class="on">✓</span>' : '';
      el.innerHTML = `${on}<span class="nm">${m.label
        ? `<span class="lbl">${m.label}</span><span class="orig">${m.name}</span>`
        : m.name}</span>${badge}
        <span class="meta">${m.vertices}v ${m.triangles}t</span>`;
      el.title = `${m.label || ''}\n${m.id}` + (m.status !== 'ok' ? `\n${m.error || ''}` : '');
      el.classList.toggle('active', S.parts.has(m.id));
      el.onclick = () => togglePart(m.id);
      box.appendChild(el);
      shown++;
    }
    if (vis.length > cap) {
      const more = document.createElement('div');
      more.className = 'mgroup-more';
      more.textContent = `… 其余 ${vis.length - cap} 项请用搜索定位`;
      box.appendChild(more);
    }
  }
  const c = S.catalog.counts;
  $('model-list-status').textContent =
    `${shown} 项显示 · 已装配 ${S.parts.size} 部件 (可解析 ${c.modelsOk}/${c.models})`;
}

$('model-search').addEventListener('input', renderModelList);
$('weapon-search').addEventListener('input', renderWeaponList);
$('weapons-head').onclick = () => $('weapons-panel').classList.toggle('closed');
document.querySelectorAll('.gtab').forEach((b) => { b.onclick = () => setGender(b.dataset.gender); });
$('btn-clear-parts').onclick = () => { clearParts(); updateModelHud(); renderModelList(); renderWeaponList(); };

/* ---------- 动画列表: 分类折叠分组 + 搜索 (虚拟滚动, 行高恒定 26px) ---------- */
const A = { rows: [], itemH: 26, rendered: new Map(), opened: new Set() };

function animMatches(a, q) {
  if (a.gender !== S.gender) return false;
  if (!q) return true;
  return a.id.toLowerCase().includes(q) || a.names.some((n) => n.toLowerCase().includes(q));
}

function buildAnimList() {
  refreshAnimList();
}

function refreshAnimList() {
  const q = $('anim-search').value.trim().toLowerCase();
  const searching = !!q;
  const groups = new Map();
  let total = 0;
  for (const a of S.catalog.anims) {
    if (!animMatches(a, q)) continue;
    if (!groups.has(a.category)) groups.set(a.category, []);
    groups.get(a.category).push(a);
    total++;
  }
  const rows = [];
  for (const cat of [...groups.keys()].sort((x, y) => x.localeCompare(y, "zh"))) {
    const items = groups.get(cat);
    const open = searching || A.opened.has(cat);   // 默认全折叠, 只展开用户点开的
    rows.push({ type: 'h', cat, n: items.length, open });
    if (open) for (const a of items) rows.push({ type: 'i', a });
  }
  A.rows = rows;
  $('anim-list-spacer').style.height = `${rows.length * A.itemH}px`;
  $('anim-list-status').textContent =
    `${total} 条 · ${groups.size} 分类 · ${S.gender}性${searching ? ' (搜索中)' : ''}`;
  renderAnimWindow();
}

function renderAnimWindow() {
  const wrap = $('anim-list-wrap');
  const list = $('anim-list');
  const top = wrap.scrollTop, h = wrap.clientHeight;
  const first = Math.max(0, Math.floor(top / A.itemH) - 5);
  const last = Math.min(A.rows.length, Math.ceil((top + h) / A.itemH) + 5);
  const keep = new Set();
  for (let i = first; i < last; i++) {
    const row = A.rows[i];
    const key = row.type === 'h' ? `h:${row.cat}` : row.a.id;
    keep.add(key);
    let el = A.rendered.get(key);
    if (!el) {
      if (row.type === 'h') {
        el = document.createElement('div');
        el.className = 'agroup-header';
        el.innerHTML = `<span class="tw">${row.open ? '▾' : '▸'}</span>${row.cat}
          <span class="cnt">${row.n}</span>`;
        el.onclick = () => {
          if (A.opened.has(row.cat)) A.opened.delete(row.cat);
          else A.opened.add(row.cat);
          refreshAnimList();
        };
      } else {
        const a = row.a;
        el = document.createElement('div');
        el.className = 'aitem';
        el.dataset.id = a.id;
        const badge = a.status !== 'ok'
          ? `<span class="badge unsup" title="${a.error || ''}">✕</span>` : '';
        el.innerHTML = `<span class="nm" style="flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
            ${a.names[0]}${a.names.length > 1 ? ` <span style="color:var(--fg-dim)">+${a.names.length - 1}</span>` : ''}</span>
          ${badge}<span class="meta">${a.frames}f</span>
          <span class="meta">${a.bones || '-'}骨</span>`;
        el.title = `${a.names.join(' / ')}\n${a.id}\n${a.frames}帧 ${a.duration}s ${a.bones}骨` +
          (a.status !== 'ok' ? `\n${a.error}` : '');
        el.onclick = () => selectAnim(a.id);
      }
      A.rendered.set(key, el);
    }
    el.style.top = `${i * A.itemH}px`;
    if (el.parentNode !== list) list.appendChild(el);
  }
  for (const [key, el] of A.rendered) {
    if (!keep.has(key)) { el.remove(); A.rendered.delete(key); }
  }
}

$('anim-list-wrap').addEventListener('scroll', renderAnimWindow);
$('anim-search').addEventListener('input', refreshAnimList);

/* ================= 部件装配 (多 xmod 共享骨架) ================= */
let modelGroup = null;   // 场景内全部部件的容器
let currentMeshes = [];  // 全部部件的网格平铺 (骨架换装时统一转 SkinnedMesh)

function ensureGroup() {
  if (!modelGroup) {
    modelGroup = new THREE.Group();
    scene.add(modelGroup);
  }
  return modelGroup;
}

function disposeMesh(mesh) {
  mesh.geometry?.dispose();
  const mats = Array.isArray(mesh.material) ? mesh.material : [mesh.material];
  for (const m of mats) {
    for (const k of ['map', 'specularMap', 'emissiveMap', 'normalMap']) m[k]?.dispose?.();
    m.dispose();
  }
}

function rebuildMeshList() {
  currentMeshes = [];
  for (const p of S.parts.values()) currentMeshes.push(...p.meshes);
}

const ddsLoader = new DDSLoader();

function loadTexture(url) {
  return new Promise((resolve) => {
    if (!url) return resolve(null);
    ddsLoader.load(url, (tex) => {
      tex.colorSpace = THREE.SRGBColorSpace;
      tex.flipY = false;          // DDS 上 V 向与引擎一致
      resolve(tex);
    }, undefined, () => resolve(null));
  });
}

/** 装配一个部件 (可叠加: 身体+脸+发型+头饰…全部绑到同一骨架). */
async function addPart(catEntry) {
  if (S.parts.has(catEntry.id)) return;
  const slot = slotOf(catEntry);
  // 同槽互斥: 自动卸下旧件 (武器/发型/衣裤鞋/脸… 每槽一件)
  for (const [id, p] of [...S.parts]) {
    if (p.slot === slot) removePart(id);
  }
  $('hud-model').innerHTML = `<span class="t">装配</span>${catEntry.name} 载入中…`;
  try {
    const data = await fetchJSON(`/api/model?id=${encodeURIComponent(catEntry.id)}`);
    const group = ensureGroup();
    const meshes = [];
    for (const sm of data.submeshes) {
      const geo = new THREE.BufferGeometry();
      geo.setAttribute('position', new THREE.Float32BufferAttribute(sm.positions, 3));
      geo.setAttribute('normal', new THREE.Float32BufferAttribute(sm.normals, 3));
      // DDS 压缩纹理块序与 D3D 一致 (v=0 = 顶行), UV 原样使用, 不做翻转
      geo.setAttribute('uv', new THREE.Float32BufferAttribute(sm.uvs, 2));
      geo.setIndex(sm.indices);
      const mat = new THREE.MeshStandardMaterial({
        color: 0xb8b8c0, roughness: 0.75, metalness: 0.05,
        side: THREE.DoubleSide,
      });
      const mesh = new THREE.Mesh(geo, mat);
      mesh.userData.sm = sm;
      mesh.userData.partId = catEntry.id;
      group.add(mesh);
      meshes.push(mesh);
      if (sm.textures.diffuse) {
        loadTexture(sm.textures.diffuse).then((tex) => {
          if (tex) { mat.map = tex; mat.color.set(0xffffff); mat.needsUpdate = true; }
        });
      }
    }
    S.parts.set(catEntry.id, { id: catEntry.id, data, cat: catEntry, meshes, slot });
    rebuildMeshList();
    if (S.anim) attachAnim(S.anim);        // 已有动画 → 全部件重建蒙皮
    else await loadDefaultSkeleton();
    updateModelHud();
    renderModelList();
    renderWeaponList();
    frameCamera();
  } catch (e) {
    $('hud-model').innerHTML = `<span class="t">装配</span>${catEntry.name} 失败: ${e.message}`;
  }
}

function removePart(id) {
  const p = S.parts.get(id);
  if (!p) return;
  for (const mesh of p.meshes) {
    mesh.parent?.remove(mesh);      // 可能在 modelGroup 或手骨 (武器挂接)
    disposeMesh(mesh);
  }
  S.parts.delete(id);
  rebuildMeshList();
  if (S.anim) attachAnim(S.anim);
  else if (!S.parts.size) {                // 空了 → 连骨架一起清
    if (S.rig?.root && modelGroup) modelGroup.remove(S.rig.root);
    S.rig = null;
    if (skeletonHelper) { scene.remove(skeletonHelper); skeletonHelper = null; }
  }
  updateModelHud();
  renderModelList();
  renderWeaponList();
  frameCamera();
}

function clearParts() {
  for (const id of [...S.parts.keys()]) removePart(id);
}

async function togglePart(id) {
  if (S.parts.has(id)) { removePart(id); return; }
  const cat = S.catalog.models.find((m) => m.id === id);
  if (!cat) return;
  await addPart(cat);
}

/** 视口内已装配部件 chips (× 卸下) */
function updateAssemblyBar() {
  const bar = $('assembly-bar');
  bar.innerHTML = '';
  for (const p of S.parts.values()) {
    const chip = document.createElement('div');
    chip.className = 'part-chip';
    chip.innerHTML = `<span>${p.cat.label || p.cat.name}</span><span class="x" title="卸下">✕</span>`;
    chip.querySelector('.x').onclick = () => removePart(p.id);
    bar.appendChild(chip);
  }
}

function frameCamera() {
  // 用全部件文件 AABB 并集 (bind 空间) 取景
  if (!S.parts.size) return;
  const box = new THREE.Box3();
  for (const p of S.parts.values()) {
    const b = p.data.aabb;
    box.union(new THREE.Box3(
      new THREE.Vector3(b[0], b[1], b[2]), new THREE.Vector3(b[3], b[4], b[5])));
  }
  const size = box.getSize(new THREE.Vector3()).length() || 1.8;
  const center = box.getCenter(new THREE.Vector3());
  controls.target.copy(center);
  // 模型正面朝 -Z (b_face 法线实测), 相机放 -Z 侧看正面
  const dir = new THREE.Vector3(-0.6, 0.3, -1).normalize();
  camera.position.copy(center).addScaledVector(dir, size * 1.4 + 0.5);
  camera.near = Math.max(0.01, size / 200);
  camera.far = size * 50;
  camera.updateProjectionMatrix();
  controls.update();
}

function updateModelHud() {
  updateAssemblyBar();
  if (!S.parts.size) {
    $('hud-model').innerHTML = `<span class="t">装配</span>空 — 点击左侧部件装载 (可多选叠加)`;
    return;
  }
  let v = 0, t = 0, subs = 0;
  const names = [];
  for (const p of S.parts.values()) {
    for (const sm of p.data.submeshes) {
      v += sm.vertexCount;
      t += sm.indices.length / 3;
      subs++;
    }
    names.push(p.cat.label || p.cat.name);
  }
  $('hud-model').innerHTML =
    `<span class="t">装配</span>${S.parts.size} 部件 · ${subs} 子网格 · ${v} 顶点 · ${Math.round(t)} 三角` +
    `<br><span style="color:var(--fg-dim)">${names.join(' + ')}</span>`;
}

/* ---------- 动画 → 骨架装配 ---------- */
async function selectAnim(id, opts = {}) {
  document.querySelectorAll('.aitem').forEach((el) =>
    el.classList.toggle('active', el.dataset.id === id));
  try {
    const data = await fetchJSON(`/api/anim?id=${encodeURIComponent(id)}`);
    S.anim = data;
    S.bindOnly = !!opts.bindOnly;
    S.frame = 0;
    attachAnim(data);
    updateAnimHud();
    syncPlaybar(true);
  } catch (e) {
    $('hud-anim').innerHTML = `<span class="t">动画</span>载入失败: ${e.message}`;
  }
}

function attachAnim(data) {
  if (!S.parts.size) return;
  convertToSkinned(data);
  if (S.bindOnly) {
    // 绑定姿势: 骨架已停在 bind, 不应用轨道帧
    S.playing = false;
    $('btn-play').textContent = '▶';
  } else {
    setFrame(0);
  }
}

function convertToSkinned(data) {
  // 换骨架前清旧
  if (skeletonHelper) { scene.remove(skeletonHelper); skeletonHelper = null; }
  if (S.rig?.root && modelGroup) modelGroup.remove(S.rig.root);
  rebuildMeshList();   // 防御: 一律从部件重建网格清单, 杜绝陈旧引用

  const bones = data.bones.map((b) => {
    const bone = new THREE.Bone();
    bone.name = b.name;
    bone.userData.b = b;
    return bone;
  });
  let root = null;
  const idxMap = new Map();
  data.bones.forEach((b, i) => idxMap.set(b.name.toLowerCase(), i));
  data.bones.forEach((b, i) => {
    if (b.parent >= 0) bones[b.parent].add(bones[i]);
    else if (!root) root = bones[i];
  });
  setBindPose(bones);
  modelGroup.add(root);
  root.updateMatrixWorld(true);       // bind 姿态世界矩阵 → Skeleton 逆
  const skeleton = new THREE.Skeleton(bones);
  S.rig = { bones, root, data, idxMap };

  // 骨架匹配率 (全部件骨名并集 ∩ 动画骨架)
  const meshBoneNames = new Set();
  for (const p of S.parts.values()) {
    for (const n of p.data.boneNames) meshBoneNames.add(n.toLowerCase());
  }
  let hit = 0;
  for (const n of meshBoneNames) if (idxMap.has(n)) hit++;
  const ratio = meshBoneNames.size ? hit / meshBoneNames.size : 0;
  S.rig.ratio = ratio;
  $('hud-warn').hidden = ratio >= 0.5;
  $('hud-warn').innerHTML = ratio < 0.5
    ? `⚠ 骨骼匹配率 ${Math.round(ratio * 100)}% — 该模型与所选动画骨架不同源, 保持绑定姿态显示`
    : '';

  // 子网格 → SkinnedMesh (root 挂 modelGroup, 多网格共享一个骨架)
  // 武器部件例外: 握柄建模在原点, 直接挂到手骨 Bip01 R Hand (不蒙皮, 跟手刚体运动)
  const handBone = bones.find((b) => b.name.toLowerCase() === 'bip01 r hand');
  // 缺失骨回退目标: 骨名不在动画骨架 (如脸部表情骨 Bone01-43) 时挂到该子网格
  // 权重最大的已匹配骨; 子网格全未匹配 (如眼球/眼皮 submesh 只认表情骨) 再取
  // 整个部件的最大已匹配骨 (脸→Bip01 Head); 部件也全未匹配才落骨架根 (index 0)
  const fbOf = new Map();            // mesh → 回退骨 index | null
  {
    const partBest = new Map();      // partId → [bestW, bestIdx]
    for (const mesh of currentMeshes) {
      const sm = mesh.userData.sm;
      let best = null, bestW = -1;
      if (sm?.boneNames && sm.weights) {
        const bw = new Map();        // 骨名(小写) → 累计权重
        let wp = 0;
        for (let vi = 0; vi < sm.vertexCount; vi++) {
          for (let k = 0; k < sm.weightCounts[vi]; k++) {
            const pr = sm.weights[wp++];
            const bn = sm.boneNames[pr[0]]?.toLowerCase();
            if (bn) bw.set(bn, (bw.get(bn) || 0) + pr[1]);
          }
        }
        for (const [bn, w] of bw) {
          const bi = idxMap.get(bn);
          if (bi !== undefined && w > bestW) { bestW = w; best = bi; }
        }
      }
      fbOf.set(mesh, best);
      const pid = mesh.userData.partId || '';
      if (best !== null && (!partBest.has(pid) || bestW > partBest.get(pid)[0]))
        partBest.set(pid, [bestW, best]);
    }
    for (const [mesh, best] of fbOf) {
      if (best === null)
        fbOf.set(mesh, partBest.get(mesh.userData.partId || '')?.[1] ?? null);
    }
  }
  window.__fb = () => currentMeshes.map((m) => ({   // 调试: 各网格回退骨
    part: (m.userData.partId || '(none)').split('/').pop(),
    sm: m.userData.sm?.name, nVerts: m.userData.sm?.vertexCount,
    fb: m.userData.fb ?? '?',
  }));
  const newMeshes = [];
  // 头骨: 近头刚体件 (脸饰/零骨件) 的挂载目标
  const headBone = bones.find((b) => b.name.toLowerCase() === 'bip01 head');
  for (const mesh of currentMeshes) {
    const sm = mesh.userData.sm;
    if (mesh.userData.partId?.startsWith('weapon/')) {
      mesh.parent?.remove(mesh);
      (handBone || modelGroup).add(mesh);
      newMeshes.push(mesh);
      continue;
    }
    // 刚体网格 (脸饰 facecloth 等无骨名 blob 件, 解析器占位权重[0,1.0]): 蒙皮无从谈起.
    // 原点建模挂件 (同武器语义: 几何在局部原点, 引擎运行时挂骨) 且部件属头部类
    // → 挂 Bip01 Head (add 原点语义, 幂等); 其余 (特效/测试件) 留原地
    if (sm && (!sm.boneNames?.length || !sm.weightCounts?.some((c) => c > 0))) {
      const pid = (mesh.userData.partId || '').toLowerCase();
      const headPart = !pid.startsWith('effect') && /face|hair|helmet|mask/.test(pid);
      const bb = mesh.geometry.boundingBox
        || (mesh.geometry.computeBoundingBox(), mesh.geometry.boundingBox);
      const atOrigin = bb.getCenter(new THREE.Vector3()).length() < 0.6;
      if (headPart && atOrigin && headBone) {
        mesh.parent?.remove(mesh);
        headBone.add(mesh);
        mesh.userData.fb = '(rigid→' + headBone.name + ')';
      } else {
        mesh.userData.fb = '(rigid, kept)';
      }
      newMeshes.push(mesh);
      continue;
    }
    const geo = mesh.geometry;
    const n = sm.vertexCount;
    const si = new Uint16Array(n * 4);
    const sw = new Float32Array(n * 4);
    const fb = fbOf.get(mesh) ?? 0;   // 缺失骨回退目标 (子网格→部件→根)
    let wpos = 0;
    for (let vi = 0; vi < n; vi++) {
      const cnt = sm.weightCounts[vi];
      for (let k = 0; k < 4; k++) {
        if (k < cnt) {
          const pair = sm.weights[wpos++];
          const bn = sm.boneNames[pair[0]]?.toLowerCase();   // 刚体网格无骨名 blob → undefined
          si[vi * 4 + k] = bn && idxMap.has(bn) ? idxMap.get(bn) : fb;
          sw[vi * 4 + k] = pair[1];
        }
      }
    }
    geo.setAttribute('skinIndex', new THREE.Uint16BufferAttribute(si, 4));
    geo.setAttribute('skinWeight', new THREE.Float32BufferAttribute(sw, 4));

    const skinned = new THREE.SkinnedMesh(geo, mesh.material);
    skinned.userData.sm = sm;
    skinned.userData.partId = mesh.userData.partId;   // 必须传递: 丢失会使后续转换把全部件并成一组 (回退骨/武器挂手全失效)
    skinned.userData.fb = bones[fb]?.name ?? '(root)';
    skinned.bind(skeleton);
    modelGroup.remove(mesh);
    modelGroup.add(skinned);
    newMeshes.push(skinned);
  }
  currentMeshes = newMeshes;
  // 写回各部件的网格引用: convertToSkinned 用新 SkinnedMesh 替换了构建时的普通 Mesh,
  // 不写回的话 removePart/rebuildMeshList 会拿到已脱离场景的旧网格 (卸不掉/重复渲染)
  {
    let k = 0;
    for (const p of S.parts.values()) {
      p.meshes = newMeshes.slice(k, k + p.meshes.length);
      k += p.meshes.length;
    }
  }
  window.__skel = skeleton;   // 调试: 手动检查骨架状态

  skeletonHelper = new THREE.SkeletonHelper(root);
  skeletonHelper.visible = $('opt-skeleton').checked;
  scene.add(skeletonHelper);
}

function setBindPose(bones) {
  for (const bone of bones) {
    const b = bone.userData.b;
    bone.position.set(b.bindPos[0], b.bindPos[1], b.bindPos[2]);
    bone.quaternion.set(b.bindQuat[0], b.bindQuat[1], b.bindQuat[2], b.bindQuat[3]);
    bone.scale.set(b.bindScale[0], b.bindScale[1], b.bindScale[2]);
  }
}

// 调试出口: 同步渲染后直读画布 (绕过合成器截图)
window.__cap = (q) => {
  renderer.render(scene, camera);
  return renderer.domElement.toDataURL('image/jpeg', q === undefined ? 0.7 : q);
};
// 调试: 去蒙皮直渲 (静态 bind 形态) — 二分 GPU 蒙皮问题
window.__unskin = () => {
  const plain = [];
  for (const m of currentMeshes) {
    const p = new THREE.Mesh(m.geometry, m.material);
    p.userData.sm = m.userData.sm;
    modelGroup.remove(m);
    modelGroup.add(p);
    plain.push(p);
  }
  currentMeshes = plain;
  renderer.render(scene, camera);
  return 'unskinned';
};
// 调试: 模块作用域桥接 (自动化测试用)
window.__cam = {
  get: () => ({ pos: camera.position.toArray(), target: controls.target.toArray() }),
  set: (pos, target) => {
    camera.position.set(...pos);
    controls.target.set(...target);
    controls.update();
    renderer.render(scene, camera);
  },
};
window.__test = {
  // 蒙皮跟随探针: 逐帧量 眼球/脸部皮肤 质心到头骨(或指定骨)的世界距离
  // 距离随帧波动大 = 该网格没跟住头骨 (错位/外冒); 刚体跟随时应近似恒定
  eyeProbe: (boneName = 'Bip01 Head', frames = [0, 5, 10, 15, 20]) => {
    if (!S.rig) return 'no rig';
    const bone = S.rig.bones.find((b) => b.name.toLowerCase() === boneName.toLowerCase());
    if (!bone) return 'no bone ' + boneName;
    const v = new THREE.Vector3();
    const centroid = (mesh) => {
      const g = mesh.geometry, pos = g.getAttribute('position');
      v.set(0, 0, 0);
      const tmp = new THREE.Vector3();
      for (let i = 0; i < pos.count; i += 7) {   // 抽样 1/7
        tmp.fromBufferAttribute(pos, i);
        (mesh.boneTransform || mesh.applyBoneTransform).call(mesh, i, tmp);
        v.add(tmp);
      }
      v.divideScalar(Math.ceil(pos.count / 7));
      return v;
    };
    const out = [];
    for (const f of frames) {
      setFrame(f);
      S.rig.root.updateMatrixWorld(true);
      const bw = bone.getWorldPosition(new THREE.Vector3());
      const row = { f, boneY: +bw.y.toFixed(3) };
      for (const mesh of currentMeshes) {
        if (!mesh.isSkinnedMesh) continue;   // 刚体挂件无蒙皮变换, 质心走世界坐标无意义
        const nm = mesh.userData.sm?.name;
        if (nm === 'Standard_3' || nm === 'Standard_1' || nm === '01 - Default')
          row[nm] = +centroid(mesh).sub(bw).length().toFixed(4);
      }
      out.push(row);
    }
    return out;
  },
  loadModel: async (id) => {          // 独占装载 (清空其余部件)
    clearParts();
    const cat = S.catalog.models.find((m) => m.id === id);
    if (!cat) throw new Error(`目录中无 ${id}`);
    await addPart(cat);
    let verts = 0;
    for (const p of S.parts.values()) {
      for (const sm of p.data.submeshes) verts += sm.vertexCount;
    }
    return { verts };
  },
  addPart: async (id) => {            // 叠加装配
    const cat = S.catalog.models.find((m) => m.id === id);
    if (!cat) throw new Error(`目录中无 ${id}`);
    await addPart(cat);
    return { parts: S.parts.size };
  },
  parts: () => [...S.parts.keys()],
  gender: (g) => setGender(g),
  loadAnim: async (id) => {
    const d = await (await fetch(`/api/anim?id=${encodeURIComponent(id)}`)).json();
    S.anim = d; S.frame = 0;
    convertToSkinned(d);
    setFrame(0);
    return { frames: d.frames, bones: d.bones.length };
  },
  setFrame: (f) => setFrame(f),
  frame: () => S.frame,
  sil: () => window.__sil(),
};
// 调试: GPU 像素回读 → 剪影包围盒 (客观验证渲染结果)
window.__sil = () => {
  const gridWas = grid.visible;
  grid.visible = false;
  if (skeletonHelper) skeletonHelper.visible = false;
  renderer.render(scene, camera);
  const gl = renderer.getContext();
  const w = gl.drawingBufferWidth, h = gl.drawingBufferHeight;
  const px = new Uint8Array(w * h * 4);
  gl.readPixels(0, 0, w, h, gl.RGBA, gl.UNSIGNED_BYTE, px);
  grid.visible = gridWas;
  if (skeletonHelper) skeletonHelper.visible = $('opt-skeleton').checked;
  let x0 = w, y0 = h, x1 = -1, y1 = -1, cnt = 0;
  for (let y = 0; y < h; y += 2) {
    for (let x = 0; x < w; x += 2) {
      const i = (y * w + x) * 4;
      const bg = Math.abs(px[i] - 20) < 14 && Math.abs(px[i + 1] - 22) < 14 && Math.abs(px[i + 2] - 26) < 14;
      if (!bg) {
        cnt++;
        if (x < x0) x0 = x; if (x > x1) x1 = x;
        if (y < y0) y0 = y; if (y > y1) y1 = y;
      }
    }
  }
  return { w, h, bboxPx: [x0, y0, x1, y1], fill: Math.round((cnt / (w * h / 4)) * 1000) / 1000 };
};
window.__dbg2 = () => {
  return {
    info: { frame: renderer.info.render.frame, calls: renderer.info.render.calls, triangles: renderer.info.render.triangles },
    canvasSize: [renderer.domElement.width, renderer.domElement.height],
    clientSize: [viewport.clientWidth, viewport.clientHeight],
    groupChildren: modelGroup ? modelGroup.children.map((c) => ({
      type: c.type, name: c.name, skinned: !!c.isSkinnedMesh,
      hasSkel: !!c.skeleton, boneTex: !!c.skeleton?.boneTexture,
      visible: c.visible, matSkin: null,
    })) : null,
    rigRootInGroup: !!(S.rig && modelGroup && S.rig.root.parent === modelGroup),
    sceneKids: scene.children.map((c) => c.type),
  };
};
// 调试: 贴图装载状态探针
window.__tex = () => currentMeshes.map((m) => ({
  sm: m.userData.sm?.name,
  url: m.userData.sm?.textures?.diffuse,
  hasMap: !!m.material.map,
  mapOk: !!m.material.map?.image,
  mips: m.material.map?.image?.mipmaps?.length,
  color: m.material.color.getHexString(),
}));
// 调试: CPU 蒙皮直渲 — 把蒙皮结果烘进 position, 换成静态 Mesh (对比 GPU 蒙皮)
window.__cpuskin = () => {
  if (S.rig?.root) S.rig.root.updateMatrixWorld(true);
  const m = new THREE.Matrix4();
  for (const sk of currentMeshes) {
    if (!sk.isSkinnedMesh) return 'already cpuskinned';
    const sm = sk.userData.sm;
    const pos = new Float32Array(sm.positions.length);
    for (let vi = 0; vi < sm.vertexCount; vi++) {
      const v = new THREE.Vector3(sm.positions[vi * 3], sm.positions[vi * 3 + 1], sm.positions[vi * 3 + 2]);
      const a = new THREE.Vector3();
      const woff = sm.weightCounts.slice(0, vi).reduce((x, y) => x + y, 0);
      for (let k = 0; k < sm.weightCounts[vi]; k++) {
        const [bi, w] = sm.weights[woff + k];
        const si = S.rig.idxMap.get(sm.boneNames[bi].toLowerCase());
        m.multiplyMatrices(sk.skeleton.bones[si].matrixWorld, sk.skeleton.boneInverses[si]);
        a.addScaledVector(v.clone().applyMatrix4(m), w);
      }
      pos[vi * 3] = a.x; pos[vi * 3 + 1] = a.y; pos[vi * 3 + 2] = a.z;
    }
    const geo = sk.geometry.clone();
    geo.setAttribute('position', new THREE.BufferAttribute(pos, 3));
    geo.deleteAttribute('skinIndex');
    geo.deleteAttribute('skinWeight');
    const plain = new THREE.Mesh(geo, sk.material);
    plain.userData.sm = sm;
    modelGroup.remove(sk);
    modelGroup.add(plain);
    const idx = currentMeshes.indexOf(sk);
    currentMeshes[idx] = plain;
  }
  renderer.render(scene, camera);
  return 'cpuskinned';
};
// 调试: bind 姿态恒等性探针 (最高顶点的权重/蒙皮结果 + CPU 包围盒)
window.__probe = () => {
  const firstPart = S.parts.values().next().value;
  if (!firstPart) return { noParts: true };
  const sm = firstPart.data.submeshes[0];
  const sk = currentMeshes[0];
  const idxMap = S.rig.idxMap;
  let hi = 0;
  for (let i = 1; i < sm.vertexCount; i++) {
    if (sm.positions[i * 3 + 1] > sm.positions[hi * 3 + 1]) hi = i;
  }
  const woff = sm.weightCounts.slice(0, hi).reduce((a, b) => a + b, 0);
  const wps = [];
  for (let k = 0; k < sm.weightCounts[hi]; k++) {
    const [bi, w] = sm.weights[woff + k];
    wps.push([sm.boneNames[bi], w]);
  }
  const inBind = S.anim.bones.every((b, i) => {
    const q = S.rig.bones[i].quaternion;
    return q.x === b.bindQuat[0] && q.y === b.bindQuat[1]
      && q.z === b.bindQuat[2] && q.w === b.bindQuat[3];
  });
  const m = new THREE.Matrix4();
  const box = new THREE.Box3();
  const rawBox = new THREE.Box3();
  for (let vi = 0; vi < sm.vertexCount; vi++) {
    const v = new THREE.Vector3(sm.positions[vi * 3], sm.positions[vi * 3 + 1], sm.positions[vi * 3 + 2]);
    rawBox.expandByPoint(v);
    const a = new THREE.Vector3();
    let wsum = 0;
    for (let k = 0; k < sm.weightCounts[vi]; k++) {
      const [bi, w] = sm.weights[sm.weightCounts.slice(0, vi).reduce((x, y) => x + y, 0) + k];
      const si = idxMap.get(sm.boneNames[bi].toLowerCase());
      wsum += w;
      if (si === undefined) continue;
      m.multiplyMatrices(sk.skeleton.bones[si].matrixWorld, sk.skeleton.boneInverses[si]);
      a.addScaledVector(v.clone().applyMatrix4(m), w);
    }
    if (Math.abs(wsum - 1) > 1e-3) a.multiplyScalar(1 / wsum);   // 归一化后再比较
    box.expandByPoint(a);
  }
  return {
    hi, origY: sm.positions[hi * 3 + 1], wps, inBind,
    rawBox: rawBox.min.toArray().concat(rawBox.max.toArray()).map((x) => +x.toFixed(3)),
    cpuBox: box.min.toArray().concat(box.max.toArray()).map((x) => +x.toFixed(3)),
  };
};
window.__dbg = () => {
  const out = { hasRig: !!S.rig, anim: S.anim?.id, frame: S.frame, ratio: S.rig?.ratio };
  const sk = currentMeshes[0];
  if (sk?.isSkinnedMesh && sk.skeleton) {
    out.inv0 = sk.skeleton.boneInverses[0].elements.map((x) => +x.toFixed(3));
    out.inv0IsIdentity = [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]
      .every((x, i) => Math.abs(x - out.inv0[i]) < 1e-6);
    out.bone0 = sk.skeleton.bones[0].name;
    out.bone0World = sk.skeleton.bones[0].matrixWorld.elements.map((x) => +x.toFixed(3));
    // 手算首顶点蒙皮 (与 Python 模拟对拍)
    const sm = sk.userData.sm;
    out.si0 = Array.from(sk.geometry.attributes.skinIndex.array.slice(0, 8));
    out.sw0 = Array.from(sk.geometry.attributes.skinWeight.array.slice(0, 8)).map((x) => +x.toFixed(3));
    out.meshBoneNames = sm.boneNames.slice(0, 5);
    const m = new THREE.Matrix4(), v = new THREE.Vector3(sm.positions[0], sm.positions[1], sm.positions[2]);
    const acc = new THREE.Vector3();
    const cnt = sm.weightCounts[0];
    for (let k = 0; k < cnt; k++) {
      const [bi, w] = sm.weights[k];
      const bn = sm.boneNames[bi].toLowerCase();
      const si2 = S.rig.idxMap.get(bn);
      m.multiplyMatrices(sk.skeleton.bones[si2].matrixWorld, sk.skeleton.boneInverses[si2]);
      const tv = v.clone().applyMatrix4(m);
      acc.addScaledVector(tv, w);
    }
    out.v0skinned = acc.toArray().map((x) => +x.toFixed(3));
    out.v0orig = [sm.positions[0], sm.positions[1], sm.positions[2]].map((x) => +x.toFixed(3));
    // GPU 实际用的 boneMatrices (skeleton.update() 的产物)
    const bm = sk.skeleton.boneMatrices;
    out.boneMat0 = Array.from(bm.slice(0, 16)).map((x) => +x.toFixed(3));
    // 全顶点 CPU 蒙皮包围盒
    const box = new THREE.Box3();
    for (let vi = 0; vi < sm.vertexCount; vi++) {
      const vv = new THREE.Vector3(sm.positions[vi * 3], sm.positions[vi * 3 + 1], sm.positions[vi * 3 + 2]);
      const a = new THREE.Vector3();
      const c2 = sm.weightCounts[vi];
      let wp = 0;
      for (let k = 0; k < c2; k++) {
        const [bi, w] = sm.weights[wp++];
        const si3 = S.rig.idxMap.get(sm.boneNames[bi].toLowerCase());
        m.multiplyMatrices(sk.skeleton.bones[si3].matrixWorld, sk.skeleton.boneInverses[si3]);
        a.addScaledVector(vv.clone().applyMatrix4(m), w);
      }
      box.expandByPoint(a);
    }
    out.cpuSkinnedBox = box.min.toArray().concat(box.max.toArray()).map((x) => +x.toFixed(2));
  }
  return out;
};

/* ---------- 逐帧求值 ---------- */
const _qa = new THREE.Quaternion(), _qb = new THREE.Quaternion();

function setFrame(f) {
  const data = S.anim;
  if (!data || !S.rig) return;
  const N = data.frames;
  if (S.loop) f = ((f % N) + N) % N;
  else f = Math.max(0, Math.min(N - 1e-4, f));
  S.frame = f;
  const i0 = Math.floor(f);
  const i1 = S.loop ? (i0 + 1) % N : Math.min(i0 + 1, N - 1);
  const t = f - i0;

  const animate = S.rig.ratio === undefined || S.rig.ratio >= 0.5;
  for (const [key, tr] of Object.entries(data.tracks)) {
    const i = +key;
    const bone = S.rig.bones[i];
    if (!bone || !animate) continue;
    const q = tr.q;                    // [x,y,z,w × N]
    const a = i0 * 4, b2 = i1 * 4;
    _qa.set(q[a], q[a + 1], q[a + 2], q[a + 3]);
    _qb.set(q[b2], q[b2 + 1], q[b2 + 2], q[b2 + 3]);
    _qa.slerp(_qb, t);
    bone.quaternion.copy(_qa);
    const b = bone.userData.b;
    if (tr.p) {
      bone.position.set(
        tr.p[i0 * 3] + (tr.p[i1 * 3] - tr.p[i0 * 3]) * t,
        tr.p[i0 * 3 + 1] + (tr.p[i1 * 3 + 1] - tr.p[i0 * 3 + 1]) * t,
        tr.p[i0 * 3 + 2] + (tr.p[i1 * 3 + 2] - tr.p[i0 * 3 + 2]) * t);
    } else {
      bone.position.set(...b.bindPos);
    }
    if (tr.s) {
      bone.scale.set(
        tr.s[i0 * 3] + (tr.s[i1 * 3] - tr.s[i0 * 3]) * t,
        tr.s[i0 * 3 + 1] + (tr.s[i1 * 3 + 1] - tr.s[i0 * 3 + 1]) * t,
        tr.s[i0 * 3 + 2] + (tr.s[i1 * 3 + 2] - tr.s[i0 * 3 + 2]) * t);
    } else {
      bone.scale.set(...b.bindScale);
    }
  }
  // SkeletonHelper 在渲染循环中自动跟随骨骼
  syncPlaybar();
}

function updateAnimHud() {
  const a = S.anim;
  if (S.bindOnly) {
    $('hud-anim').innerHTML = `<span class="t">动画</span>绑定姿势 (在动画列表选择任意动画播放)`;
    return;
  }
  $('hud-anim').innerHTML =
    `<span class="t">动画</span>${a.name} · ${a.frames} 帧 · ${a.duration}s · ${a.bones.length} 骨`;
}

/* ---------- 播放控制 ---------- */
let _lastBarUpdate = 0;
function syncPlaybar(force) {
  const now = performance.now();
  if (!force && now - _lastBarUpdate < 100) return;   // 播放中 DOM 写入节流至 10/s
  _lastBarUpdate = now;
  const N = S.anim ? S.anim.frames : 0;
  const slider = $('frame-slider');
  slider.max = Math.max(0, N - 1e-4);
  slider.value = S.frame;
  $('frame-label').textContent = `${S.frame.toFixed(1)} / ${N}`;
  $('time-label').textContent = `${(S.frame / (S.anim?.fps || 30)).toFixed(2)}s`;
}

$('btn-play').onclick = () => {
  S.playing = !S.playing;
  $('btn-play').textContent = S.playing ? '⏸' : '▶';
};
$('speed').onchange = (e) => { S.speed = +e.target.value; };
$('opt-loop').onchange = (e) => { S.loop = e.target.checked; };
$('frame-slider').oninput = (e) => {
  S.playing = false; $('btn-play').textContent = '▶';
  setFrame(+e.target.value);
};
window.addEventListener('keydown', (e) => {
  if (e.target.tagName === 'INPUT' || e.target.tagName === 'SELECT') return;
  if (e.code === 'Space') { e.preventDefault(); $('btn-play').click(); }
});

/* ---------- 显示开关 ---------- */
$('opt-skeleton').onchange = (e) => { if (skeletonHelper) skeletonHelper.visible = e.target.checked; };
$('opt-wireframe').onchange = (e) => {
  for (const m of currentMeshes) m.material.wireframe = e.target.checked;
};
$('opt-texture').onchange = (e) => {
  S.showTexture = e.target.checked;
  for (const m of currentMeshes) {
    const sm = m.userData.sm;
    if (!sm?.textures.diffuse) continue;
    if (!e.target.checked) { m.material.map = null; m.material.color.set(0xb8b8c0); }
    else {
      loadTexture(sm.textures.diffuse).then((tex) => {
        if (tex && S.showTexture) { m.material.map = tex; m.material.color.set(0xffffff); m.material.needsUpdate = true; }
      });
    }
    m.material.needsUpdate = true;
  }
};
$('opt-rotate').onchange = (e) => { controls.autoRotate = e.target.checked; controls.autoRotateSpeed = 1.2; };
$('opt-dark-bg').onchange = (e) => {
  scene.background = new THREE.Color(e.target.checked ? 0x14161a : 0x9aa3b0);
  grid.visible = true;
};

/* ---------- 主循环 ---------- */
const clock = new THREE.Clock();
let _lastRender = 0;
const NOLOOP = new URLSearchParams(location.search).has('noloop');   // 无头截图模式: 500ms 一帧
function renderFrame() {
  const now = performance.now();
  if (now - _lastRender < (NOLOOP ? 400 : 10)) return;   // 防重复渲染
  _lastRender = now;
  const dt = clock.getDelta();
  if (S.playing && S.anim && S.rig && !S.bindOnly) {
    setFrame(S.frame + dt * S.anim.fps * S.speed);
  }
  controls.update();
  try {
    renderer.render(scene, camera);
  } catch (e) {
    if (!window.__renderErr) {
      window.__renderErr = String(e && e.stack || e).slice(0, 600);
      console.error(e);
    }
  }
}
// 调度: 单 rAF 链为主; 看门狗链仅在 rAF 长时间不触发时兜底 (嵌入式浏览器 rAF 可能停摆).
// 注意不能在回调里同时挂 rAF+setTimeout —— 两条链会互相繁殖导致回调指数膨胀 (页面卡死).
let _rafAlive = 0;
function rafChain() {
  _rafAlive = performance.now();
  renderFrame();
  requestAnimationFrame(rafChain);
}
function stoChain() {
  setTimeout(stoChain, NOLOOP ? 500 : 60);
  if (performance.now() - _rafAlive > 150) renderFrame();
}
rafChain();
stoChain();

/* ---------- 启动 ---------- */
window.__ready.then(async () => {
  // loadCatalog → setGender('男') 已装配默认身体; 再挂跑步循环 (贴图/蒙皮/动画全链路即刻可验)
  try {
    const run = S.catalog.anims.find((a) => a.id === 'b_action/base/run.xskt');
    if (run) {
      await selectAnim(run.id);
      S.playing = true;
      $('btn-play').textContent = '⏸';
    }
  } catch (e) {
    console.warn('默认场景失败', e);
  }
});
