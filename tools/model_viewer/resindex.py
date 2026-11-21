"""资源目录扫描: 模型清单 + 动画索引链 (GBK ini) → catalog.

资源根 (--res / NINEYIN_RES_ROOT, 默认 F:\\Game\\Online\\9yin\\res):
  其余路径都相对它推导 — obj\\char=角色  obj\\weapon=武器
  obj\\effect / obj\\effect_sns / obj\\effect_weapon=特效
  (各子根仍可用 --root/--weapons 或 NINEYIN_CHAR_ROOT/NINEYIN_WEAPON_ROOT 显式覆盖)
- 模型: 全部 .xmod (性别/类别归类 + 配套贴图/材质 + 默认骨架解析 + 中文显示名)
- 武器: 按目录前缀归类 (剑/刀/长兵/…), 不分性别
- 特效: obj_effect* 三树, catalog id 加 effect*/ 前缀, 不分性别
- 动画: b_action/g_action 的 action.ini → 分类 index.ini → [小节] Skeleton= 路径
  (GBK、反斜杠、大小写混杂 → 归一化 + 大小写不敏感文件映射)
- 每个动画文件快速解析头部 (不解码轨道) 取 N/fps/骨数; 失败标记 status

CLI: python resindex.py [--res PATH] [--out catalog.json]
"""
from __future__ import annotations

import argparse
import json
import os
import re
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from format import xmod, xskt

# 资源根: 其余路径都相对它 (obj/char obj/weapon obj/effect*)
DEFAULT_RES_ROOT = os.environ.get("NINEYIN_RES_ROOT", r"F:\Game\Online\9yin\res")
# 特效三树 (obj/ 下) → FileMap 前缀与 catalog 分类
EFFECT_PREFIXES = ("effect", "effect_sns", "effect_weapon")
EFFECT_CATEGORIES = {"effect": "特效", "effect_sns": "特效·SNS", "effect_weapon": "特效·武器"}
# 显式覆盖 / 资源根不存在时的回退 (历史解包目录)
DEFAULT_ROOT = os.environ.get(
    "NINEYIN_CHAR_ROOT",
    r"E:\Workspace\project\game\adata\obj_char_unpacked\res\obj\char")
DEFAULT_WEAPON_ROOT = os.environ.get(
    "NINEYIN_WEAPON_ROOT",
    r"E:\Workspace\project\game\adata\obj_weapon_unpacked\res\obj\weapon")
_REPO_RESOURCES = os.path.normpath(os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "..", "..", "resources"))
DEFAULT_RESOURCES = os.environ.get(
    "NINEYIN_RESOURCES",
    _REPO_RESOURCES if os.path.isdir(_REPO_RESOURCES)
    else r"E:\Workspace\project\game\9yin_server-go\resources")


def sub_roots(res_root: str) -> dict:
    """资源根 → {子树名: 物理路径} (只含 obj/ 下实际存在的目录)."""
    out = {}
    for name in ("char", "weapon", *EFFECT_PREFIXES):
        p = os.path.join(res_root, "obj", name)
        if os.path.isdir(p):
            out[name] = p
    return out


def derive_resources(res_root: str | None) -> str | None:
    """资源根 → 同级服务器资源表 (<res 父>/9yin-server-go/resources, 中文名映射).
    找不到返回 None (调用方回退 --resources/环境变量默认)."""
    parent = os.path.dirname(os.path.abspath(res_root).rstrip("\\/"))
    cand = os.path.join(parent, "9yin-server-go", "resources")
    if os.path.isdir(os.path.join(cand, "modern", "share", "item")):
        return cand
    return None


def resolve_roots(res_root: str | None = None, char_root: str | None = None,
                  weapons_root: str | None = None) -> tuple:
    """资源根推导 + 显式覆盖 → (char, weapon, {特效名: 路径}, 实际资源根).
    优先级: 显式 char/weapons 参数 > 资源根下 obj/char|weapon > 旧默认."""
    res_root = res_root or DEFAULT_RES_ROOT
    subs = sub_roots(res_root)
    char = char_root or subs.get("char") or DEFAULT_ROOT
    weapons = (weapons_root if weapons_root is not None
               else subs.get("weapon") or DEFAULT_WEAPON_ROOT)
    effects = {k: subs[k] for k in EFFECT_PREFIXES if k in subs}
    return char, weapons, effects, res_root


def norm_key(rel_path: str) -> str:
    """归一化路径键: 正斜杠 + 小写 (Windows 大小写不敏感)."""
    return rel_path.replace("\\", "/").lower().lstrip("/")


class FileMap:
    """char 树 (+ 附加前缀树, 如 weapon/) 大小写不敏感的文件映射."""

    def __init__(self, root: str, extra_roots=()):
        # extra_roots: [(url 前缀, 物理根)] — 条目 key = 前缀 + 相对路径
        self.root = root
        self.by_key = {}
        self._key_of = {}          # 物理全路径(小写) → key
        for dirpath, _, files in os.walk(root):
            for fn in files:
                full = os.path.join(dirpath, fn)
                k = norm_key(os.path.relpath(full, root))
                self.by_key[k] = full
                self._key_of[full.lower()] = k
        for prefix, proot in extra_roots:
            for dirpath, _, files in os.walk(proot):
                for fn in files:
                    full = os.path.join(dirpath, fn)
                    k = norm_key(prefix + os.path.relpath(full, proot))
                    self.by_key[k] = full
                    self._key_of[full.lower()] = k

    def url_key(self, full: str) -> str:
        """物理路径 → catalog/URL 用的 key."""
        return self._key_of.get(full.lower())

    def resolve(self, ini_path: str) -> str | None:
        """ini 内路径 → 实际文件路径.
        支持: obj\\char\\… (char 树相对键) / obj\\weapon|effect*/… (附加前缀键) /
        相对 char 的裸路径."""
        p = ini_path.replace("\\", "/")
        for cand in (p,
                     re.sub(r"^obj/char/", "", p, count=1, flags=re.IGNORECASE),
                     re.sub(r"^obj/", "", p, count=1, flags=re.IGNORECASE)):
            hit = self.by_key.get(norm_key(cand))
            if hit:
                return hit
        return None

    def exists(self, rel: str) -> bool:
        return norm_key(rel) in self.by_key


def read_gbk(path: str) -> str:
    with open(path, "r", encoding="gbk", errors="replace") as f:
        return f.read()


def parse_ini_sections(text: str) -> dict:
    """极简 ini: {section: {key: [values]}} (同名键聚合, 保留顺序)."""
    out: dict = {}
    cur = None
    for line in text.splitlines():
        line = line.strip()
        if not line or line.startswith("//") or line.startswith(";"):
            continue
        if line.startswith("[") and line.endswith("]"):
            cur = line[1:-1].strip()
            out.setdefault(cur, {})
            continue
        if "=" in line and cur is not None:
            k, _, v = line.partition("=")
            out[cur].setdefault(k.strip(), []).append(v.strip())
    return out


# ---------------- 中文名映射 (服务器资源表 → 模型路径/武器目录) ----------------

_TAG_RE = re.compile(r"<[^>]*>")


def _has_cjk(s: str) -> bool:
    return any("\u4e00" <= c <= "\u9fff" for c in (s or ""))


def build_name_index(resources_root: str) -> dict:
    """equipment.ini (QName/Name→ArtPack) ∘ itemartstatic.ini (ArtPack→模型路径)
    ∘ 文本表 (key→中文) → {归一化模型路径或武器目录: 中文名}.

    武器条目的 MaleModel 是不带 obj\\ 前缀的武器目录名 (如 axe_leader_xt_002),
    同时按 "目录级" 键收录, 使该目录下全部 xmod 命中."""
    if not resources_root or not os.path.isdir(resources_root):
        return {}
    text: dict = {}
    tdir = os.path.join(resources_root, "modern", "text", "chineses")
    if os.path.isdir(tdir):
        for fn in os.listdir(tdir):
            p = os.path.join(tdir, fn)
            if not os.path.isfile(p):
                continue
            try:
                raw = open(p, "rb").read()
                try:
                    body = raw.decode("utf-8")          # 文本表为 UTF-8
                except UnicodeDecodeError:
                    body = raw.decode("gbk", errors="replace")
                for line in body.splitlines():
                    line = line.strip()
                    if not line or line.startswith("//") or "=" not in line:
                        continue
                    k, _, v = line.partition("=")
                    v = _TAG_RE.sub("", v).strip()
                    if k and v and _has_cjk(v) and k not in text:
                        text[k] = v
            except OSError:
                pass

    def sections(rel):
        p = os.path.join(resources_root, rel)
        if not os.path.isfile(p):
            return {}
        with open(p, "r", encoding="gbk", errors="replace") as f:
            return parse_ini_sections(f.read())

    equip = sections(os.path.join("modern", "share", "item", "equipment.ini"))
    art = sections(os.path.join("modern", "share", "item", "itemartstatic.ini"))

    # artid → 中文名 (文本表 QName 命中率最高; Name 字段多为生成串)
    art_name: dict = {}
    for _key, kv in equip.items():
        artid = kv.get("ArtPack", [None])[0]
        if not artid or artid in art_name:
            continue
        qname = kv.get("QName", [None])[0]
        name = kv.get("Name", [None])[0]
        label = None
        for cand in (qname, name, _key):
            if cand and cand in text:
                label = text[cand]
                break
        if not label and name and _has_cjk(name):
            label = name
        if label:
            art_name[artid] = label

    path_label: dict = {}
    for artid, kv in art.items():
        label = art_name.get(artid)
        if not label:
            continue
        for f in ("MaleModel", "FemaleModel"):
            mp = kv.get(f, [None])[0]
            if not mp:
                continue
            norm = mp.replace("\\", "/").lower().strip("/")
            norm = re.sub(r"^obj/(char|weapon)/", "", norm)
            if not norm or "/" in norm and not re.match(r"^[a-z0-9_/.]+$", norm):
                continue
            path_label.setdefault(norm, label)
            if "/" in norm:                    # 目录级键 (目录下全部部件命中)
                path_label.setdefault(norm.rsplit("/", 1)[0], label)
    return path_label


# 武器目录前缀 → 中文类型 (obj_weapon_unpacked 实测分布)
WEAPON_TYPES = {
    "sword": "剑", "ssword": "短剑", "sblade": "短刃", "blade": "刀", "saxe": "小斧",
    "knife": "短刀", "axe": "斧", "mace": "锤", "claymore": "重剑", "naginata": "薙刀",
    "lstuff": "长兵", "stuff": "异兵", "spear": "枪", "gun": "铳", "bow": "弓",
    "arrow": "箭", "thorn": "刺", "sthorn": "双刺", "cosh": "拳套", "katar": "爪",
    "zither": "琴", "qin": "琴", "flute": "笛", "fan": "扇", "pen": "笔",
    "ruler": "尺", "loop": "环", "wrap": "软兵", "token": "令牌",
}


def weapon_category(rel: str) -> str:
    stem = os.path.basename(rel)
    d = os.path.basename(os.path.dirname(rel))
    for src in (d, stem):
        m = re.match(r"([a-z]+)_", src.lower())
        if m and m.group(1) in WEAPON_TYPES:
            return WEAPON_TYPES[m.group(1)]
    m = re.match(r"([a-z]+)_", stem.lower())
    return WEAPON_TYPES.get(m.group(1) if m else "", "其它")


def classify_model(rel: str) -> tuple:
    """(性别, 类别) 归类 — 性别以文件名前缀优先 (b_xiyu10/g_face.xmod 之类目录名会骗人),
    部件按关键词分类, _h 后缀 = 发型 (实测全 BoneHAIR* 骨)."""
    d = os.path.dirname(rel).replace("\\", "/")
    stem = os.path.basename(rel)
    low = stem.lower()
    # 性别: 文件名前缀 > 目录前缀
    if low.startswith("b_"):
        gender = "男"
    elif low.startswith("g_"):
        gender = "女"
    else:
        base = os.path.basename(d)
        gender = "男" if base.startswith("b_") else "女"
    if "s_face" in rel:
        return gender, "特殊脸(s_face)"
    if d.startswith("tab_"):
        return gender, "蒙面头"
    if d == "helper":
        return "通用", "辅助件"
    if d == "test":
        return "女", "测试件"
    if d in ("b_action", "g_action") and low.startswith("tpose"):
        return gender, "身体参考"
    # _h 后缀 = 发型 (配套的 *_h.xmod)
    if low.endswith("_h.xmod"):
        return gender, "发型"
    # b_facecloth/g_facecloth 以 b_face/g_face 开头但语义独立 (面纱/眼罩等), 先于前缀判定
    if "facecloth" in d.lower() or "facecloth" in low:
        return gender, "脸部饰件"
    if d.startswith("b_face"):
        return gender, "脸型/胡子"
    if d.startswith("g_face"):
        return gender, "脸型"
    # 部件关键词 (文件名优先于目录名)
    part = None
    for src in (low, (os.path.basename(d) or "").lower()):
        for kw, cat in (("facecloth", "脸部饰件"), ("mask", "面具"), ("cloak", "披风"),
                        ("hair", "发型"), ("helmet", "头饰"), ("shoes", "鞋"),
                        ("pants", "下装"), ("cloth", "上衣/时装"), ("weapon", "武器挂件")):
            if kw in src:
                part = cat
                break
        if part:
            break
    if part:
        return gender, part
    if "_physx" in low or "ragdoll" in low:
        return gender, "物理件"
    return gender, "身体/套装"


def default_skeleton(fm: FileMap, rel_model: str) -> str | None:
    """模型默认骨架: 同名 .xskt → 同目录 tpose.xskt → 同性别全身 tpose → 另一性别."""
    d = os.path.dirname(rel_model)
    stem = os.path.splitext(os.path.basename(rel_model))[0]
    gender_tpose = []
    base = os.path.basename(d) or stem
    if base.startswith("g_"):
        gender_tpose = ["g_action/tpose.xskt", "b_action/tpose.xskt"]
    else:
        gender_tpose = ["b_action/tpose.xskt", "g_action/tpose.xskt"]
    for cand in (f"{d}/{stem}.xskt" if d else f"{stem}.xskt",
                 f"{d}/tpose.xskt" if d else "tpose.xskt",
                 *gender_tpose):
        if fm.exists(cand):
            return fm.resolve(cand)
    return None


def build_models(fm: FileMap, path_label: dict | None = None) -> list:
    path_label = path_label or {}
    models = []
    skip = ("weapon/", *(f"{n}/" for n in EFFECT_PREFIXES))
    for key in sorted(fm.by_key):
        if not key.endswith(".xmod") or key.startswith(skip):
            continue
        full = fm.by_key[key]
        rel = os.path.relpath(full, fm.root).replace("\\", "/")
        gender, cat = classify_model(rel)
        entry = _model_entry(fm, key, rel, gender, cat, path_label)
        models.append(entry)
    return models


def build_weapons(fm: FileMap, path_label: dict | None = None) -> list:
    """obj_weapon 树 (catalog id 带 weapon/ 前缀, 由 FileMap 附加根提供)."""
    path_label = path_label or {}
    models = []
    for key in sorted(fm.by_key):
        if not key.startswith("weapon/") or not key.endswith(".xmod"):
            continue
        rel = key                      # weapon/<dir>/<file>.xmod
        entry = _model_entry(fm, key, rel, "通用", f"武器·{weapon_category(rel)}", path_label)
        models.append(entry)
    return models


def build_effects(fm: FileMap, path_label: dict | None = None) -> list:
    """obj_effect* 三树 (catalog id 带 effect*/ 前缀, 由 FileMap 附加根提供)."""
    path_label = path_label or {}
    models = []
    for key in sorted(fm.by_key):
        if not key.endswith(".xmod"):
            continue
        m = re.match(r"(effect(?:_sns|_weapon)?)/", key)
        if not m:
            continue
        models.append(_model_entry(fm, key, key, "通用", EFFECT_CATEGORIES[m.group(1)],
                                   path_label))
    return models


_DDS_CACHE: dict = {}          # 目录 → 排序 dds 文件名 (obj/effect 平铺 1.7 万文件)


def _dir_dds(d: str) -> list:
    lst = _DDS_CACHE.get(d)
    if lst is None:
        try:
            lst = sorted(fn for fn in os.listdir(d) if fn.lower().endswith(".dds"))
        except OSError:
            lst = []
        _DDS_CACHE[d] = lst
    return lst


def _model_entry(fm: FileMap, key: str, rel: str, gender: str, cat: str,
                 path_label: dict) -> dict:
    full = fm.by_key[key]
    stem = os.path.splitext(os.path.basename(rel))[0]
    entry = {
        "id": rel,
        "name": stem,
        "label": None,                 # 中文名 (服务器资源表映射, 可缺)
        "dir": os.path.dirname(rel),
        "gender": gender,
        "category": cat,
        "skeleton": None,
        "vertices": 0, "triangles": 0, "submeshes": 0,
        "textures": [], "mtl": None,
        "status": "ok", "error": None,
    }
    # 中文名: 文件级 → 目录级 (武器再试去掉 weapon/ 前缀的目录名 —
    # itemartstatic 武器引用是不带路径的武器目录名)
    no_ext = key[:-5]
    dir_key = no_ext.rsplit("/", 1)[0]
    label = path_label.get(no_ext) or path_label.get(dir_key)
    if not label and dir_key.startswith("weapon/"):
        label = path_label.get(dir_key[len("weapon/"):])
    entry["label"] = label
    # 配套贴图/材质 (同目录; 平铺大目录只收同 stem 前若干张, 防目录爆体积)
    mtl_key = key[:-5] + ".mtl"
    if mtl_key in fm.by_key:
        entry["mtl"] = fm.url_key(fm.by_key[mtl_key])
    dds_dir = os.path.dirname(full)
    dds_list = _dir_dds(dds_dir)
    big_dir = len(dds_list) > 64
    for fn in dds_list:
        if len(entry["textures"]) >= 8:
            break
        if big_dir and not fn.lower().startswith(stem[:6].lower()):
            continue        # 大目录 (如 obj/effect 平铺): 只收 stem 相近的贴图
        uk = fm.url_key(os.path.join(dds_dir, fn))
        if uk:
            entry["textures"].append(uk)
    try:
        m = xmod.parse_file(full)
        entry["vertices"] = sum(sm.vertex_count for sm in m.submeshes)
        entry["triangles"] = sum(sm.triangle_count for sm in m.submeshes)
        entry["submeshes"] = len(list(m.submeshes))
    except Exception as e:
        entry["status"] = "unsupported" if isinstance(e, xmod.UnsupportedVersion) else "error"
        entry["error"] = str(e)[:120]
    skt = default_skeleton(fm, rel)
    if skt:
        entry["skeleton"] = fm.url_key(skt)
    return entry


def build_anims(fm: FileMap) -> list:
    """action.ini 索引链 + 根级散置 → 动画目录 (按文件去重, 一文件多名)."""
    anims = {}   # id → entry
    for gender, act_ini in (("男", "b_action/action.ini"), ("女", "g_action/action.ini")):
        top = fm.resolve(act_ini)
        if not top:
            continue
        sections = parse_ini_sections(read_gbk(top))
        cat_files = []
        for k, vals in sections.get("ACTION_CHILD_FILE", {}).items():
            cat_files.extend(vals)
        # 顶层 [Tpose] 等自带小节也扫
        own = {s: kv for s, kv in sections.items() if s != "ACTION_CHILD_FILE"}
        _load_cat(fm, anims, gender, "(顶层)", own)
        for cf in cat_files:
            path = fm.resolve(cf)
            if not path:
                continue
            cat_name = os.path.basename(os.path.dirname(path))
            try:
                _load_cat(fm, anims, gender, cat_name,
                          parse_ini_sections(read_gbk(path)))
            except OSError:
                pass
    # 根级散置 + 脸部动画等未索引文件兜底收录
    for key in sorted(fm.by_key):
        if not key.endswith(".xskt"):
            continue
        if key in anims:
            continue
        d = os.path.dirname(key)
        if d in ("b_action", "g_action"):
            gender = "男" if d == "b_action" else "女"
            _add_anim(fm, anims, key, [os.path.basename(key)], gender, "(根级散置)")
        elif d.startswith(("b_face", "g_face")):
            gender = "男" if d.startswith("b_face") else "女"
            _add_anim(fm, anims, key, [os.path.basename(key)], gender, "脸部/表情")
    return sorted(anims.values(), key=lambda e: e["id"])


def _load_cat(fm, anims, gender, cat, sections):
    for sec, kv in sections.items():
        if sec.upper() == "TPOSE" and "Skeleton" not in kv:
            continue
        skel = kv.get("Skeleton", [None])[0]
        if not skel:
            continue
        full = fm.resolve(skel)
        if not full:
            continue
        rel = os.path.relpath(full, fm.root).replace("\\", "/").lower()
        meta = {
            "enterTime": _f(kv.get("EnterTime", [None])[0]),
            "leaveTime": _f(kv.get("LeaveTime", [None])[0]),
            "baseSpeed": _f(kv.get("BaseSpeed", [None])[0]),
            "triggers": sum(1 for k in kv if k.upper().startswith("TRIGGER_CUSTOM")),
        }
        _add_anim(fm, anims, rel, [sec], gender, cat, meta)


def _f(v):
    try:
        return round(float(v), 4)
    except (TypeError, ValueError):
        return None


def _add_anim(fm, anims, key, names, gender, cat, meta=None):
    e = anims.get(key)
    if e is None:
        e = {
            "id": key,
            "names": [],
            "gender": gender,
            "category": cat,
            "frames": 0, "duration": 0.0, "bones": 0,
            "status": "ok", "error": None,
            "enterTime": None, "leaveTime": None, "baseSpeed": None, "triggers": 0,
        }
        try:
            s = xskt.parse_file(fm.by_key[key], decode_tracks=False)
            e["frames"] = s.frame_count
            e["duration"] = round(s.duration, 2)
            e["bones"] = len(s.bones)
        except Exception as ex:
            e["status"] = "unsupported" if "版本" in str(ex) or "v0x1003" in str(ex) else "error"
            e["error"] = str(ex)[:120]
        anims[key] = e
    for n in names:
        if n not in e["names"]:
            e["names"].append(n)
    if meta:
        for k, v in meta.items():
            if v is not None or e.get(k) is None:
                e[k] = v if v is not None else e.get(k)
    # 多名字时按性别保持一致
    return e


def build_catalog(root: str = None, weapons_root: str = None, resources_root: str = None,
                  res_root: str = None, effect_roots: dict | None = None) -> dict:
    """root/weapons_root=None 时由资源根推导 (resolve_roots);
    effect_roots=None 时同样取资源根下 obj/effect* 实际存在的目录."""
    res_root = res_root or DEFAULT_RES_ROOT
    root, weapons_root, derived_effects, res_root = resolve_roots(
        res_root, char_root=root, weapons_root=weapons_root)
    resources_root = (resources_root if resources_root is not None
                      else derive_resources(res_root) or DEFAULT_RESOURCES)
    effect_roots = effect_roots if effect_roots is not None else derived_effects
    extra = []
    if weapons_root and os.path.isdir(weapons_root):
        extra.append(("weapon/", weapons_root))
    for name in EFFECT_PREFIXES:
        p = effect_roots.get(name)
        if p and os.path.isdir(p):
            extra.append((f"{name}/", p))
    fm = FileMap(root, extra)
    t0 = time.time()
    path_label = build_name_index(resources_root) if os.path.isdir(resources_root) else {}
    models = build_models(fm, path_label) + build_weapons(fm, path_label) \
        + build_effects(fm, path_label)
    anims = build_anims(fm)
    labeled = sum(1 for m in models if m["label"])
    return {
        "root": root,
        "resRoot": res_root,
        "weaponsRoot": weapons_root,
        "effectRoots": {k: effect_roots[k] for k in EFFECT_PREFIXES if k in effect_roots},
        "generated": time.strftime("%Y-%m-%d %H:%M:%S"),
        "buildSeconds": round(time.time() - t0, 1),
        "models": models,
        "anims": anims,
        "counts": {
            "models": len(models),
            "modelsOk": sum(1 for m in models if m["status"] == "ok"),
            "modelsLabeled": labeled,
            "effects": sum(1 for m in models if m["category"] in EFFECT_CATEGORIES.values()),
            "anims": len(anims),
            "animsOk": sum(1 for a in anims if a["status"] == "ok"),
        },
    }


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--res", default=DEFAULT_RES_ROOT,
                    help=f"资源根, 其余路径相对它 (默认 {DEFAULT_RES_ROOT})")
    ap.add_argument("--root", default=None, help="角色根 (默认 <res>/obj/char)")
    ap.add_argument("--weapons", default=None, help="武器根 (默认 <res>/obj/weapon)")
    ap.add_argument("--resources", default=DEFAULT_RESOURCES)
    ap.add_argument("--out", default=os.path.join(os.path.dirname(
        os.path.abspath(__file__)), "catalog.json"))
    args = ap.parse_args()
    cat = build_catalog(root=args.root, weapons_root=args.weapons,
                        resources_root=args.resources, res_root=args.res)
    with open(args.out, "w", encoding="utf-8") as f:
        json.dump(cat, f, ensure_ascii=False)
    c = cat["counts"]
    print(f"目录完成: 模型 {c['modelsOk']}/{c['models']} 可解析 ({c['modelsLabeled']} 有中文名), "
          f"动画 {c['animsOk']}/{c['anims']} 可解析, "
          f"耗时 {cat['buildSeconds']}s → {args.out}")


if __name__ == "__main__":
    main()
