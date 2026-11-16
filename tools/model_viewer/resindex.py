"""资源目录扫描: 模型清单 + 动画索引链 (GBK ini) → catalog.

数据源: obj_char_unpacked\\res\\obj\\char (可用 NINEYIN_CHAR_ROOT 覆盖)
- 模型: 全部 .xmod (性别/类别归类 + 配套贴图/材质 + 默认骨架解析)
- 动画: b_action/g_action 的 action.ini → 分类 index.ini → [小节] Skeleton= 路径
  (GBK、反斜杠、大小写混杂 → 归一化 + 大小写不敏感文件映射)
- 每个动画文件快速解析头部 (不解码轨道) 取 N/fps/骨数; 失败标记 status

CLI: python resindex.py [--root PATH] [--out catalog.json]
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

DEFAULT_ROOT = os.environ.get(
    "NINEYIN_CHAR_ROOT",
    r"F:\Game\Online\9yin\九阴资源\obj_char_unpacked\res\obj\char")


def norm_key(rel_path: str) -> str:
    """归一化路径键: 正斜杠 + 小写 (Windows 大小写不敏感)."""
    return rel_path.replace("\\", "/").lower().lstrip("/")


class FileMap:
    """char 树内大小写不敏感的文件映射."""

    def __init__(self, root: str):
        self.root = root
        self.by_key = {}
        for dirpath, _, files in os.walk(root):
            for fn in files:
                full = os.path.join(dirpath, fn)
                self.by_key[norm_key(os.path.relpath(full, root))] = full

    def resolve(self, ini_path: str) -> str | None:
        """ini 内路径 (obj\\char\\... 或相对 char) → 实际文件路径."""
        p = ini_path.replace("\\", "/")
        p = re.sub(r"^obj/char/", "", p, count=1, flags=re.IGNORECASE)
        return self.by_key.get(norm_key(p))

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


def classify_model(rel: str) -> tuple:
    """(性别, 类别) 归类."""
    d = os.path.dirname(rel).replace("\\", "/")
    stem = os.path.basename(rel)
    if "s_face" in rel:
        gender = "男" if "/b_" in f"/{stem}" or stem.startswith("b_") else "女"
        return gender, "特殊脸(s_face)"
    if d.startswith("tab_"):
        return ("男", "蒙面头") if d.startswith("tab_b") else ("女", "蒙面头")
    if d == "helper":
        return "通用", "辅助件"
    if d == "test":
        return "女", "测试件"
    if d in ("b_action", "g_action") and stem.lower().startswith("tpose"):
        return ("男", "身体参考") if d == "b_action" else ("女", "身体参考")
    if d.startswith("b_face"):
        return "男", "脸型/胡子"
    if d.startswith("g_face"):
        return "女", "脸型"
    base = os.path.basename(d) or stem
    gender = "男" if base.startswith("b_") or stem.startswith("b_") else "女"
    return gender, "其它"


def default_skeleton(fm: FileMap, rel_model: str) -> str | None:
    """模型默认骨架: 同名 .xskt → 同目录 tpose.xskt → 性别全身 tpose."""
    d = os.path.dirname(rel_model)
    stem = os.path.splitext(os.path.basename(rel_model))[0]
    for cand in (f"{d}/{stem}.xskt" if d else f"{stem}.xskt",
                 f"{d}/tpose.xskt" if d else "tpose.xskt",
                 "b_action/tpose.xskt", "g_action/tpose.xskt"):
        if fm.exists(cand):
            return fm.resolve(cand)
    return None


def build_models(fm: FileMap) -> list:
    models = []
    for key in sorted(fm.by_key):
        if not key.endswith(".xmod"):
            continue
        full = fm.by_key[key]
        rel = os.path.relpath(full, fm.root).replace("\\", "/")
        gender, cat = classify_model(rel)
        entry = {
            "id": rel,
            "name": os.path.splitext(os.path.basename(rel))[0],
            "dir": os.path.dirname(rel),
            "gender": gender,
            "category": cat,
            "skeleton": None,
            "vertices": 0, "triangles": 0, "submeshes": 0,
            "textures": [], "mtl": None,
            "status": "ok", "error": None,
        }
        # 配套贴图/材质 (同目录)
        mtl_key = key[:-5] + ".mtl"
        if mtl_key in fm.by_key:
            entry["mtl"] = os.path.relpath(fm.by_key[mtl_key], fm.root).replace("\\", "/")
        dds_dir = os.path.dirname(full)
        for fn in sorted(os.listdir(dds_dir)):
            if fn.lower().endswith(".dds"):
                entry["textures"].append(
                    os.path.relpath(os.path.join(dds_dir, fn), fm.root).replace("\\", "/"))
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
            entry["skeleton"] = os.path.relpath(skt, fm.root).replace("\\", "/")
        models.append(entry)
    return models


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


def build_catalog(root: str = None) -> dict:
    root = root or DEFAULT_ROOT
    fm = FileMap(root)
    t0 = time.time()
    models = build_models(fm)
    anims = build_anims(fm)
    return {
        "root": root,
        "generated": time.strftime("%Y-%m-%d %H:%M:%S"),
        "buildSeconds": round(time.time() - t0, 1),
        "models": models,
        "anims": anims,
        "counts": {
            "models": len(models),
            "modelsOk": sum(1 for m in models if m["status"] == "ok"),
            "anims": len(anims),
            "animsOk": sum(1 for a in anims if a["status"] == "ok"),
        },
    }


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", default=DEFAULT_ROOT)
    ap.add_argument("--out", default=os.path.join(os.path.dirname(
        os.path.abspath(__file__)), "catalog.json"))
    args = ap.parse_args()
    cat = build_catalog(args.root)
    with open(args.out, "w", encoding="utf-8") as f:
        json.dump(cat, f, ensure_ascii=False)
    c = cat["counts"]
    print(f"目录完成: 模型 {c['modelsOk']}/{c['models']} 可解析, "
          f"动画 {c['animsOk']}/{c['anims']} 可解析, "
          f"耗时 {cat['buildSeconds']}s → {args.out}")


if __name__ == "__main__":
    main()
