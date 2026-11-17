"""九阴角色模型浏览工具 — 本地 HTTP 服务 (纯标准库).

用法: python server.py [--port 8901] [--root CHAR_ROOT]
打开 http://127.0.0.1:8901

API:
  GET /api/catalog           模型+动画目录 (首次构建后缓存)
  GET /api/model?id=<rel>    解析 xmod → 子网格几何/权重/贴图 URL
  GET /api/anim?id=<rel>     解析 xskt → 骨架 + 逐帧轨道 (已解码 f32)
  GET /res/<rel>             原样伺服资源文件 (DDS 等, 大小写不敏感)
"""
from __future__ import annotations

import argparse
import gzip
import io
import json
import mimetypes
import os
import re
import struct
import sys
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, unquote, urlparse

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import resindex
from format import xmod, xskt

TOOL_DIR = os.path.dirname(os.path.abspath(__file__))
STATIC_DIR = os.path.join(TOOL_DIR, "static")

_state = {"catalog": None, "lock": threading.Lock()}


def get_catalog(root: str):
    with _state["lock"]:
        if _state["catalog"] is None:
            _state["catalog"] = resindex.build_catalog(root)
        return _state["catalog"]


def _round_list(xs, nd=4):
    return [round(v, nd) for v in xs]


def _resolve_texture(fm, root: str, rel_base: str, texture_ref: str):
    """贴图引用 (相对 xmod/mtl 目录) → /res/<key> URL 或 None."""
    if not texture_ref:
        return None
    p = texture_ref.replace("\\", "/")
    cand = fm.resolve(p) if p.lower().startswith("obj/") else fm.resolve(f"{rel_base}/{p}")
    if cand:
        return "/res/" + fm.url_key(cand)
    return None


def _map_texture_slots(dds_strs, fm, root, rel_dir):
    """贴图串列表 → {diffuse, specular, normal} URL.
    串内常见重复 (槽位成对); 按后缀语义分配: _d=diffuse _s=specular _n=normal,
    无后缀标记时首串作 diffuse."""
    resolved = []
    for s in dds_strs:
        u = _resolve_texture(fm, root, rel_dir, s)
        if u and u not in resolved:
            resolved.append(u)
    slots = {"diffuse": None, "specular": None, "normal": None}
    for u in resolved:
        stem = u.rsplit("/", 1)[-1].rsplit(".", 1)[0].lower()
        if stem.endswith("_d") and not slots["diffuse"]:
            slots["diffuse"] = u
        elif stem.endswith("_s") and not slots["specular"]:
            slots["specular"] = u
        elif stem.endswith("_n") and not slots["normal"]:
            slots["normal"] = u
    if not slots["diffuse"] and resolved:
        slots["diffuse"] = resolved[0]
    return slots


def build_model_payload(fm, root: str, rel: str) -> dict:
    full = fm.by_key.get(rel.lower())
    if not full or not full.lower().endswith(".xmod"):
        raise FileNotFoundError(rel)
    m = xmod.parse_file(full)
    rel_dir = os.path.dirname(rel).replace("\\", "/")

    # mtl 材质映射: 小节名标准化 (Standard_1 → standard_1 / material #0)
    mtl_sections = {}
    mtl_rel = None
    mtl_key = rel.lower()[:-5] + ".mtl"
    if mtl_key in fm.by_key:
        mtl_full = fm.by_key[mtl_key]
        mtl_rel = fm.url_key(mtl_full)
        for sec, kv in resindex.parse_ini_sections(resindex.read_gbk(mtl_full)).items():
            mtl_sections[sec.strip().lower()] = kv

    submeshes = []
    bone_names_all = []
    for sm in m.submeshes:
        for bn in sm.bone_names:
            if bn not in bone_names_all:
                bone_names_all.append(bn)
        name_key = sm.name.strip().lower()
        kv = mtl_sections.get(name_key)
        if kv is None and len(mtl_sections) == 1:
            kv = next(iter(mtl_sections.values()))
        textures = {"diffuse": None, "specular": None, "emissive": None}
        if kv:
            for field, slot in (("diffusemap", "diffuse"), ("specularlevelmap", "specular"),
                                ("specularmap", "specular"), ("emissivemap", "emissive")):
                v = kv.get(field, [None])[0] if kv else None
                if v and not textures[slot]:
                    textures[slot] = _resolve_texture(fm, root, rel_dir, v)
        dds_strs = [s for s in sm.textures if s.lower().endswith(".dds")]
        slots = _map_texture_slots(dds_strs, fm, root, rel_dir)
        if not textures["diffuse"]:
            textures["diffuse"] = slots["diffuse"]
        if not textures["specular"]:
            textures["specular"] = slots["specular"]
        textures["normal"] = slots["normal"]
        submeshes.append({
            "name": sm.name,
            "vertexCount": sm.vertex_count,
            "positions": _round_list(sm.positions),
            "normals": _round_list(sm.normals, 3),
            "uvs": _round_list(sm.uvs),
            "indices": sm.indices,
            "weights": [[bi, round(w, 4)] for v in sm.weights for bi, w in v],
            "weightCounts": [len(v) for v in sm.weights],
            "boneNames": sm.bone_names,
            "textures": textures,
        })
    return {
        "id": rel,
        "name": os.path.splitext(os.path.basename(rel))[0],
        "aabb": _round_list(m.aabb, 3),
        "warnings": m.warnings,
        "boneNames": bone_names_all,
        "submeshes": submeshes,
        "mtl": mtl_rel,
    }


def build_anim_payload(fm, root: str, rel: str) -> dict:
    full = fm.by_key.get(rel.lower())
    if not full or not full.lower().endswith(".xskt"):
        raise FileNotFoundError(rel)
    s = xskt.parse_file(full)
    bones = []
    for b in s.bones:
        bones.append({
            "name": b.name,
            "parent": b.parent,
            "kind": b.kind,
            "hasPos": b.has_pos,
            "hasScale": b.has_scale,
            "bindPos": _round_list(b.bind_pos, 5),
            "bindQuat": _round_list(b.bind_quat, 5),
            "bindScale": _round_list(b.bind_scale, 5),
            "track": bool(b.quats),
        })
    tracks = {}
    for i, b in enumerate(s.bones):
        if not b.quats:
            continue
        # 展平数组: q=[x,y,z,w × N] (w 服务端已算), p/s=[x,y,z × N]
        t = {"q": _round_list([c for q in b.quats for c in q], 5)}
        if b.poss:
            t["p"] = _round_list([c for p in b.poss for c in p], 4)
        if b.scales:
            t["s"] = _round_list([c for sc in b.scales for c in sc], 4)
        tracks[i] = t
    return {
        "id": rel,
        "name": s.name,
        "frames": s.frame_count,
        "fps": s.fps,
        "duration": round(s.duration, 3),
        "warnings": s.warnings,
        "bones": bones,
        "tracks": tracks,
    }


class Handler(BaseHTTPRequestHandler):
    root = DEFAULT_ROOT = resindex.DEFAULT_ROOT
    fm = None

    def log_message(self, fmt, *args):
        pass  # 静默访问日志

    def _send(self, code, body: bytes, ctype: str, gzip_ok=True):
        accept = self.headers.get("Accept-Encoding", "")
        if gzip_ok and "gzip" in accept and len(body) > 2048:
            buf = io.BytesIO()
            with gzip.GzipFile(fileobj=buf, mode="wb", compresslevel=6) as g:
                g.write(body)
            body = buf.getvalue()
            self.send_response(code)
            self.send_header("Content-Type", ctype)
            self.send_header("Content-Encoding", "gzip")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
        else:
            self.send_response(code)
            self.send_header("Content-Type", ctype)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
        self.wfile.write(body)

    def _json(self, obj, code=200):
        self._send(code, json.dumps(obj, ensure_ascii=False).encode("utf-8"),
                   "application/json; charset=utf-8")

    def do_GET(self):
        try:
            self._route()
        except (FileNotFoundError, xmod.XModError, xskt.XSktError) as e:
            self._json({"error": str(e)}, 404)
        except BrokenPipeError:
            pass
        except Exception as e:  # noqa: BLE001
            self._json({"error": f"{type(e).__name__}: {e}"}, 500)

    def _route(self):
        u = urlparse(self.path)
        path = unquote(u.path)
        if path in ("/", "/index.html"):
            self._file(os.path.join(STATIC_DIR, "index.html"))
            return
        if path.startswith("/static/"):
            self._file(os.path.join(STATIC_DIR, *path[len("/static/"):].split("/")))
            return
        if path.startswith("/api/"):
            self._api(path, parse_qs(u.query))
            return
        if path.startswith("/res/"):
            self._res(path[len("/res/"):])
            return
        self._json({"error": "not found"}, 404)

    def _api(self, path, q):
        cat = get_catalog(self.root)
        if path == "/api/catalog":
            self._json(cat)
            return
        rel = (q.get("id") or [""])[0]
        if not rel:
            self._json({"error": "missing id"}, 400)
            return
        if path == "/api/model":
            self._json(build_model_payload(self.fm, self.root, rel))
        elif path == "/api/anim":
            self._json(build_anim_payload(self.fm, self.root, rel))
        else:
            self._json({"error": "unknown api"}, 404)

    def _file(self, full):
        if not os.path.isfile(full) or ".." in full:
            self._json({"error": "not found"}, 404)
            return
        ctype = mimetypes.guess_type(full)[0] or "application/octet-stream"
        if full.endswith(".js"):
            ctype = "text/javascript"
        with open(full, "rb") as f:
            self._send(200, f.read(), ctype)

    def _res(self, rel):
        rel = rel.replace("\\", "/")
        full = self.fm.by_key.get(rel.lower())
        if not full:
            self._json({"error": f"资源不存在: {rel}"}, 404)
            return
        ctype = "application/octet-stream"
        if rel.lower().endswith((".html", ".js", ".css")):
            self._json({"error": "forbidden"}, 403)
            return
        with open(full, "rb") as f:
            self._send(200, f.read(), ctype, gzip_ok=False)


def main():
    ap = argparse.ArgumentParser(description="九阴角色模型浏览工具")
    ap.add_argument("--port", type=int, default=int(os.environ.get("NINEYIN_VIEWER_PORT", 8901)))
    ap.add_argument("--host", default="127.0.0.1")
    ap.add_argument("--root", default=resindex.DEFAULT_ROOT)
    ap.add_argument("--weapons", default=resindex.DEFAULT_WEAPON_ROOT,
                    help="obj_weapon 解包根 (catalog id 前缀 weapon/)")
    ap.add_argument("--resources", default=resindex.DEFAULT_RESOURCES,
                    help="服务器资源根 (中文名映射表)")
    args = ap.parse_args()

    extra = []
    if args.weapons and os.path.isdir(args.weapons):
        extra.append(("weapon/", args.weapons))
    print(f"资源根: {args.root}  武器根: {args.weapons}  资源表: {args.resources}")
    Handler.root = args.root
    Handler.fm = resindex.FileMap(args.root, extra)
    srv = ThreadingHTTPServer((args.host, args.port), Handler)
    url = f"http://{args.host}:{args.port}"
    print(f"模型浏览工具就绪: {url}  (目录在首次访问 /api/catalog 时构建, 约 12s)")
    try:
        srv.serve_forever()
    except KeyboardInterrupt:
        print("\nbye")


if __name__ == "__main__":
    main()
