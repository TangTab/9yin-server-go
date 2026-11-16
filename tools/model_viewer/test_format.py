"""解析器回归测试 — 真实资源样本, 解析必须精确到 EOF 且满足几何不变量.

资源根默认 F:\\Game\\Online\\9yin\\九阴资源\\obj_char_unpacked\\res\\obj\\char,
可用 NINEYIN_CHAR_ROOT 覆盖; 不存在时整组跳过.
"""
from __future__ import annotations

import math
import os
import unittest

import sys
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from format import xmod, xskt, ddsinfo

CHAR_ROOT = os.environ.get(
    "NINEYIN_CHAR_ROOT",
    r"F:\Game\Online\9yin\九阴资源\obj_char_unpacked\res\obj\char")

HAVE_RES = os.path.isdir(CHAR_ROOT)


def p(*parts):
    return os.path.join(CHAR_ROOT, *parts)


@unittest.skipUnless(HAVE_RES, "角色资源目录不存在")
class TestXMod(unittest.TestCase):

    def _parse(self, rel):
        m = xmod.parse_file(p(*rel.split("/")))
        # 全部子网格几何不变量
        total_v = total_t = 0
        for sm in m.submeshes:
            vc = sm.vertex_count
            self.assertGreater(vc, 0)
            total_v += vc
            total_t += sm.triangle_count
            self.assertTrue(all(0 <= i < vc for i in sm.indices),
                            f"{rel}/{sm.name}: 索引越界")
            self.assertEqual(len(sm.indices) % 3, 0, f"{rel}/{sm.name}: 非三角列表")
            self.assertEqual(len(sm.weights), vc, f"{rel}/{sm.name}: 权重数 != 顶点数")
            for j in range(vc):
                wsum = sum(w for _, w in sm.weights[j])
                self.assertAlmostEqual(wsum, 1.0, delta=1e-3,
                                       msg=f"{rel}/{sm.name}: 顶点 {j} 权重和 {wsum}")
            for k in range(0, len(sm.uvs), 2):
                # 平铺贴图允许 UV 超出 [0,1], 只做宽 sanity 校验
                self.assertGreaterEqual(sm.uvs[k], -1.0)
                self.assertLessEqual(sm.uvs[k], 4.0)
                self.assertGreaterEqual(sm.uvs[k + 1], -1.0)
                self.assertLessEqual(sm.uvs[k + 1], 4.0)
            for k in range(0, len(sm.normals), 3):
                n = (sm.normals[k], sm.normals[k + 1], sm.normals[k + 2])
                self.assertAlmostEqual(math.sqrt(sum(c * c for c in n)), 1.0,
                                       delta=0.02,
                                       msg=f"{rel}/{sm.name}: 法线非单位向量 {n}")
            self.assertGreater(len(sm.bone_names), 0)
        self.assertGreater(total_t, 0)
        return m, total_v, total_t

    def test_helper_minimal(self):
        # 522B 最小样本: 1 对象 1 子网格 3 顶点 1 三角
        m, v, t = self._parse("helper/helper.xmod")
        self.assertEqual(v, 3)
        self.assertEqual(t, 1)

    def test_b_face(self):
        m, v, t = self._parse("b_face/b_face.xmod")
        self.assertEqual(len(list(m.submeshes)), 2)
        self.assertEqual(v, 712 + 632)
        self.assertEqual(t, 1058 + 888)

    def test_tab_b_head(self):
        m, v, t = self._parse("tab_b_head_148_b/tab_b_head_148_b.xmod")
        self.assertEqual(len(list(m.submeshes)), 2)
        self.assertEqual(v, 1227 + 736)
        # AABB 与顶点包围盒一致性 (头模高度 ~2m 内)
        self.assertLess(m.aabb[4], 2.2)

    def test_tpose_body(self):
        # 全身参考体 (动画播放载体): 低模 181 顶点, 高度是硬指标
        m, v, t = self._parse("b_action/tpose.xmod")
        self.assertGreater(v, 100)
        self.assertEqual(len(list(m.submeshes)), 1)
        # 人体高度: AABB y 范围应约 0~1.9m
        self.assertGreater(m.aabb[4], 1.5)
        self.assertLess(m.aabb[1], 0.2)

    def test_tpose_g_body(self):
        xmod.parse_file(p("g_action/tpose.xmod"))

    def test_face_variants(self):
        # 各版本脸型抽样
        for rel in ("b_face/face_59/b_face_59.xmod",
                    "g_face/face_16/g_face_16.xmod",
                    "b_face/b_s_face01/b_s_face01.xmod"):
            xmod.parse_file(p(*rel.split("/")))

    def test_0x1000_unsupported(self):
        with self.assertRaises(xmod.UnsupportedVersion):
            xmod.parse_file(p("test/g_hair.xmod"))


@unittest.skipUnless(HAVE_RES, "角色资源目录不存在")
class TestXSkt(unittest.TestCase):

    def _parse(self, rel, min_bones=1):
        s = xskt.parse_file(p(*rel.split("/")))
        self.assertGreaterEqual(len(s.bones), min_bones)
        self.assertGreater(s.frame_count, 0)
        self.assertAlmostEqual(s.fps, 30.0, delta=1e-4)
        for b in s.bones:
            if b.kind == 4:
                self.assertEqual(len(b.quats), s.frame_count,
                                 f"{rel}/{b.name}: 旋转帧数 != N")
                if b.has_pos:
                    self.assertEqual(len(b.poss), s.frame_count)
                q = b.bind_quat
                norm = math.sqrt(sum(c * c for c in q))
                self.assertAlmostEqual(norm, 1.0, delta=0.01,
                                       msg=f"{rel}/{b.name}: bind 四元数非单位 {q}")
        return s

    def test_tpose_trivial(self):
        s = self._parse("helper/tpose.xskt")
        self.assertEqual(s.frame_count, 3)
        self.assertEqual([b.name for b in s.bones],
                         ["Bone01", "BoneGW01", "GW01"])
        self.assertEqual(s.bones[2].kind, 3)  # 挂点

    def test_catch_r(self):
        s = self._parse("helper/catch_r_01.xskt")
        self.assertEqual(s.frame_count, 47)

    def test_fi_0h_back(self):
        s = self._parse("b_action/base/0h/fi_0h_back.xskt", min_bones=100)
        self.assertEqual(s.frame_count, 51)
        bip = s.bones[0]
        self.assertTrue(bip.has_pos)
        self.assertEqual(bip.name, "Bip01")

    def test_run(self):
        s = self._parse("b_action/base/run.xskt", min_bones=100)
        self.assertGreater(s.frame_count, 10)

    def test_v1003_unsupported(self):
        # v0x1003 (diedown 系列 8 文件) 布局变体未全解 — 应明确报不支持
        with self.assertRaises(xskt.XSktError):
            xskt.parse_file(p("b_action/base/0h/fi_0h_diedownloop.xskt"))

    def test_face_skeleton(self):
        s = self._parse("b_face/b_face.xskt", min_bones=30)

    def test_root_level_files(self):
        # 根级散置动画 (含曾被误判截断的 stand001) — 全部应可解析或明确报错
        for fn in ("stand001.xskt", "stand001_s.xskt", "run_dodge_l.xskt",
                   "run_dodge_r.xskt", "walk_dodge_l.xskt", "walk_dodge_r.xskt",
                   "tpose.xskt"):
            path = p("b_action", fn)
            if os.path.exists(path):
                with self.subTest(fn=fn):
                    xskt.parse_file(path)

    def test_blink_anim(self):
        # 眨眼动画驱动脸部
        path = p("b_face/b_face_zhayan.xskt")
        if os.path.exists(path):
            xskt.parse_file(path)


def struct_error():
    import struct
    return struct.error


@unittest.skipUnless(HAVE_RES, "角色资源目录不存在")
class TestDDS(unittest.TestCase):

    def test_samples(self):
        for rel, fmt in (("b_face/b_face.dds", "DXT3"),
                         ("b_action/tpose.dds", None),
                         ("tab_b_head_148_b/tab_b_head_148_b_1_n.dds", "DXT1")):
            info = ddsinfo.probe_file(p(*rel.split("/")))
            self.assertGreater(info["width"], 0)
            if fmt:
                self.assertEqual(info["fourcc"], fmt)


if __name__ == "__main__":
    unittest.main(verbosity=2)
