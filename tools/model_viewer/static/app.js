/* 九阴角色模型浏览器 — three.js 前端
 * 数据流: /api/catalog → 列表; /api/model → SkinnedMesh; /api/anim → 骨架+轨道
 * 蒙皮: 动画文件内嵌 bind 变换 → three.Skeleton (bind 时自动取逆);
 *       每帧按轨道 slerp/lerp 驱动骨骼局部变换。
 */
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { DDSLoader } from 'three/addons/loaders/DDSLoader.js';

/* ================= 状态 ================= */
const S = {
  catalog: null,
  model: null,          // /api/model 载荷
  anim: null,           // /api/anim 载荷
  rig: null,            // {bones[], root, skeleton, matched, meshBones}
  playing: false,
  frame: 0,
  speed: 1,
  loop: true,
  showTexture: true,
  bindOnly: false,      // true = 只装配骨架停在绑定姿势, 不应用轨道
};

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
  $('lib-info').textContent = '加载目录…';
  S.catalog = await fetchJSON('/api/catalog');
  const c = S.catalog.counts;
  $('lib-info').textContent = '';
  $('model-count').textContent = `${c.modelsOk}/${c.models}`;
  $('anim-count').textContent = `${c.animsOk}/${c.anims}`;
  buildModelGroups();
  buildAnimList();
}

/* ---------- 模型列表 ---------- */
function buildModelGroups() {
  const groups = new Map();
  for (const m of S.catalog.models) {
    const key = `${m.gender} · ${m.category}`;
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(m);
  }
  const box = $('model-groups');
  box.innerHTML = '';
  for (const [key, items] of groups) {
    const h = document.createElement('div');
    h.className = 'mgroup-header';
    h.textContent = `${key} (${items.length})`;
    box.appendChild(h);
    for (const m of items) {
      const el = document.createElement('div');
      el.className = 'mitem';
      el.dataset.id = m.id;
      const badge = m.status !== 'ok'
        ? `<span class="badge unsup" title="${m.error || ''}">${m.status === 'unsupported' ? '版本' : '错误'}</span>` : '';
      el.innerHTML = `<span class="nm">${m.name}</span>${badge}
        <span class="meta">${m.vertices}v ${m.triangles}t</span>`;
      el.title = m.id + (m.status !== 'ok' ? `\n${m.error || ''}` : '');
      el.onclick = () => selectModel(m.id);
      box.appendChild(el);
    }
  }
}

/* ---------- 动画虚拟列表 ---------- */
const A = { filtered: [], itemH: 26, rendered: new Map() };

function animFilter() {
  const q = $('anim-search').value.trim().toLowerCase();
  const g = $('anim-gender').value;
  const cat = $('anim-cat').value;
  return S.catalog.anims.filter((a) => {
    if (g && a.gender !== g) return false;
    if (cat && a.category !== cat) return false;
    if (!q) return true;
    return a.id.toLowerCase().includes(q) || a.names.some((n) => n.toLowerCase().includes(q));
  });
}

function buildAnimList() {
  const cats = [...new Set(S.catalog.anims.map((a) => a.category))].sort();
  const sel = $('anim-cat');
  for (const c of cats) {
    const o = document.createElement('option');
    o.textContent = c; o.value = c;
    sel.appendChild(o);
  }
  refreshAnimList();
}

function refreshAnimList() {
  A.filtered = animFilter();
  $('anim-list-spacer').style.height = `${A.filtered.length * A.itemH}px`;
  $('anim-list-status').textContent =
    `${A.filtered.length} 条 / 共 ${S.catalog.anims.length}${$('anim-search').value ? ' (已过滤)' : ''}`;
  renderAnimWindow();
}

function renderAnimWindow() {
  const wrap = $('anim-list-wrap');
  const list = $('anim-list');
  const top = wrap.scrollTop, h = wrap.clientHeight;
  const first = Math.max(0, Math.floor(top / A.itemH) - 5);
  const last = Math.min(A.filtered.length, Math.ceil((top + h) / A.itemH) + 5);
  const keep = new Set();
  for (let i = first; i < last; i++) {
    const a = A.filtered[i];
    keep.add(a.id);
    let el = A.rendered.get(a.id);
    if (!el) {
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
      A.rendered.set(a.id, el);
    }
    el.style.top = `${i * A.itemH}px`;
    if (el.parentNode !== list) list.appendChild(el);
  }
  for (const [id, el] of A.rendered) {
    if (!keep.has(id)) { el.remove(); A.rendered.delete(id); }
  }
}

$('anim-list-wrap').addEventListener('scroll', renderAnimWindow);
$('anim-search').addEventListener('input', refreshAnimList);
$('anim-gender').addEventListener('change', refreshAnimList);
$('anim-cat').addEventListener('change', refreshAnimList);

/* ---------- Tab 切换 ---------- */
document.querySelectorAll('.tab').forEach((t) => {
  t.onclick = () => {
    document.querySelectorAll('.tab').forEach((x) => x.classList.remove('active'));
    document.querySelectorAll('.tabpane').forEach((x) => x.classList.remove('active'));
    t.classList.add('active');
    $(`tab-${t.dataset.tab}`).classList.add('active');
    if (t.dataset.tab === 'anims') renderAnimWindow();
  };
});

/* ================= 模型/动画装载 ================= */
let modelGroup = null;   // 场景内当前模型容器
let currentMeshes = [];

function disposeModel() {
  if (modelGroup) {
    scene.remove(modelGroup);
    modelGroup.traverse((o) => {
      if (o.geometry) o.geometry.dispose();
      if (o.material) {
        for (const m of Array.isArray(o.material) ? o.material : [o.material]) {
          for (const k of ['map', 'specularMap', 'emissiveMap']) m[k]?.dispose?.();
          m.dispose();
        }
      }
    });
  }
  modelGroup = null;
  currentMeshes = [];
  S.rig = null;
  if (skeletonHelper) { scene.remove(skeletonHelper); skeletonHelper = null; }
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

async function selectModel(id, keepAnim = true) {
  document.querySelectorAll('.mitem').forEach((el) =>
    el.classList.toggle('active', el.dataset.id === id));
  $('hud-model').innerHTML = `<span class="t">模型</span>载入中…`;
  try {
    const data = await fetchJSON(`/api/model?id=${encodeURIComponent(id)}`);
    disposeModel();
    S.model = data;
    buildModel(data);
    updateModelHud();
    if (!keepAnim || !S.anim) {
      const cat = S.catalog.models.find((m) => m.id === id);
      if (cat?.skeleton) await selectAnim(cat.skeleton, { bindOnly: true });
    } else if (S.anim) {
      attachAnim(S.anim);   // 用现有动画重建骨架
    }
  } catch (e) {
    $('hud-model').innerHTML = `<span class="t">模型</span>载入失败: ${e.message}`;
  }
}

function buildModel(data) {
  modelGroup = new THREE.Group();
  scene.add(modelGroup);
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
    modelGroup.add(mesh);
    currentMeshes.push(mesh);
    if (sm.textures.diffuse) {
      loadTexture(sm.textures.diffuse).then((tex) => {
        if (tex) { mat.map = tex; mat.color.set(0xffffff); mat.needsUpdate = true; }
      });
    }
  }
  frameCamera();
}

function frameCamera() {
  // 用文件 AABB (bind 空间) 取景
  const b = S.model.aabb;
  const box = new THREE.Box3(
    new THREE.Vector3(b[0], b[1], b[2]), new THREE.Vector3(b[3], b[4], b[5]));
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
  const d = S.model;
  const v = d.submeshes.reduce((s, x) => s + x.vertexCount, 0);
  const t = d.submeshes.reduce((s, x) => s + x.indices.length / 3, 0);
  $('hud-model').innerHTML =
    `<span class="t">模型</span>${d.name} · ${d.submeshes.length} 子网格 · ${v} 顶点 · ${Math.round(t)} 三角`;
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
  if (!S.model) return;
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
  if (S.rig?.root) modelGroup.remove(S.rig.root);

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

  // 骨架匹配率 (网格骨名 ∩ 动画骨架)
  const meshBoneNames = new Set(S.model.boneNames.map((n) => n.toLowerCase()));
  let hit = 0;
  for (const n of meshBoneNames) if (idxMap.has(n)) hit++;
  const ratio = meshBoneNames.size ? hit / meshBoneNames.size : 0;
  S.rig.ratio = ratio;
  $('hud-warn').hidden = ratio >= 0.5;
  $('hud-warn').innerHTML = ratio < 0.5
    ? `⚠ 骨骼匹配率 ${Math.round(ratio * 100)}% — 该模型与所选动画骨架不同源, 保持绑定姿态显示`
    : '';

  // 子网格 → SkinnedMesh (root 挂 modelGroup, 多网格共享一个骨架)
  const newMeshes = [];
  for (const mesh of currentMeshes) {
    const sm = mesh.userData.sm;
    const geo = mesh.geometry;
    const n = sm.vertexCount;
    const si = new Uint16Array(n * 4);
    const sw = new Float32Array(n * 4);
    let wpos = 0;
    for (let vi = 0; vi < n; vi++) {
      const cnt = sm.weightCounts[vi];
      for (let k = 0; k < 4; k++) {
        if (k < cnt) {
          const pair = sm.weights[wpos++];
          const bn = sm.boneNames[pair[0]].toLowerCase();
          si[vi * 4 + k] = idxMap.has(bn) ? idxMap.get(bn) : 0;
          sw[vi * 4 + k] = pair[1];
        }
      }
    }
    geo.setAttribute('skinIndex', new THREE.Uint16BufferAttribute(si, 4));
    geo.setAttribute('skinWeight', new THREE.Float32BufferAttribute(sw, 4));

    const skinned = new THREE.SkinnedMesh(geo, mesh.material);
    skinned.userData.sm = sm;
    skinned.bind(skeleton);
    modelGroup.remove(mesh);
    modelGroup.add(skinned);
    newMeshes.push(skinned);
  }
  currentMeshes = newMeshes;
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
window.__test = {
  loadModel: async (id) => {
    await selectModel(id, false);
    return { verts: S.model.submeshes.reduce((a, x) => a + x.vertexCount, 0) };
  },
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
loadCatalog().then(async () => {
  // 默认载入男全身参考体 + 跑步循环 (第0帧=站立姿势, 播放时最易辨认)
  try {
    await selectModel('b_action/tpose.xmod', false);
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
