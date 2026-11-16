# 九阴角色模型浏览器

浏览 `obj_char_unpacked` 解包资源中的角色网格 / 贴图 / 骨骼动画，任意模型 × 任意动画自由配对播放。
纯 Python 标准库后端 + three.js 前端，无 npm 依赖。

## 启动

```bash
cd tools/model_viewer
python server.py            # 默认 127.0.0.1:8901
# 打开 http://127.0.0.1:8901
```

可选参数：`--port 8901`、`--host 0.0.0.0`、`--root <char目录>`
（默认 `F:\Game\Online\9yin\九阴资源\obj_char_unpacked\res\obj\char`，或环境变量 `NINEYIN_CHAR_ROOT`）。

首次访问 `/api/catalog` 时构建目录（扫 12k+ 文件，约 12s），之后常驻内存。

## 界面

- **模型页**：按 性别 × 类别 分组列出全部 79 个 .xmod（脸型/胡子/特殊脸/蒙面头/tpose 全身参考体…），
  显示顶点/三角数；选中后自动装配该目录的默认骨架。
- **动画页**：12,000+ 动画（b_action/g_action 索引链 + 根级散置 + 脸部表情），搜索框 + 性别/分类过滤 +
  虚拟滚动；显示帧数/骨数/时长，未支持版本标 ✕。
- **视口**：轨道相机；骨架叠加 / 线框 / 贴图开关；网格与所选动画骨架按骨名匹配，
  匹配率 <50% 时保持绑定姿态并提示。
- **播放条**：播放/暂停（空格）、0.1–4× 倍速、帧拖动、循环开关。
  任意模型可与任意动画配对（例：tpose 全身 × `skill/` 下任意招式）。

## 逆向格式支持

| 格式 | 版本 | 状态 |
|---|---|---|
| XVAM (.xmod) | 0x1002 | 全解（顶点 pos/nrm/uv 32B、u16 三角列表、4 槽蒙皮权重、骨名 blob、材质区自适应） |
| XVAM (.xmod) | 0x1000 | test/ 目录 5 件，顶点布局未解，标记不支持 |
| XSAM (.xskt) | 0x1005 | 全解（递归骨树、bind 变换、B 轨道 0/1/2 三型、C 轨道开关、挂点 44B+18B） |
| XSAM (.xskt) | 0x1003 / 0x2000 / 0x3000 / 0x5000 | 布局变体未全解（约 250 个 diedown 等），标记不支持 |
| DDS | DXT1/3/5 | 传统格式全支持（three.js DDSLoader） |

关键定版细节（详见 `format/xmod.py` / `format/xskt.py` 文件头注释）：

- **i16×3 轨道旋转 = 欧拉角，非四元数量化**（引擎逆向铁证：fxmodeladv_rebuilt.dll @0x52a2
  常量 π/2/65535 + `d3dx9!D3DXQuaternionRotationYawPitchRoll(out, Yaw=s1, Pitch=s0, Roll=s2)`）：
  `angle = i16 × 2π/65535`，`quat = qy(yaw)·qx(pitch)·qz(roll)`。
  验证：tpose/run/stand/jump 全 112 骨 f0 与 bind 偏差 0.0°
- **B 轨道型**：0=6B 纯欧拉；2=18B `[欧拉 i16×3][pos f32×3]`；1=18B `[欧拉 i16×3][scale f32×3]`
  （欧拉在前，f32 尾巴在后）；A=3 B=6 挂点 44B bind + 18B 轨道；C=轨道开关（half 系 0=无轨道）
- xmod 权重的 boneRef = 骨名 blob 内**字节偏移**（非序号）
- 材质区 = 6×0.588 浮点 + 0~4 组附加浮点 + 2~4 个 .dds 串（或零串，按下一结构回溯识别）
- 对象尾部绑定表三形态：纯对 / 条目式（tpose 的 264 条穴位挂点，骨→穴名+40B 变换）/ 不透明跳过
- **贴图方向**：DDS 压缩块序与 D3D 一致（v=0=顶行），UV 原样使用不翻转；模型正面朝 **−Z**（b_face 法线实测），默认相机在 −Z 侧

引擎逆向工具见 `re_tools/`（capstone 反汇编 disa.py、riprel 交叉引用 xref.py、导入表 imports.py、pdata 函数表 funcs.py）；
引擎内部格式为 .gm2/.bon/.an2（fxmodel），XVAM/XSAM 为资源流格式。

## 组件

```
format/xmod.py     XVAM 解析器        format/xskt.py   XSAM 解析器
format/ddsinfo.py  DDS 头探测         resindex.py      目录/索引链扫描 (catalog.json)
server.py          HTTP 服务+API      static/          three.js 前端 (vendor 已内置)
test_format.py     解析器回归测试 (16 用例)
shot.js            无头 Chrome CDP 截图工具 (node shot.js <url> <out.png> [waitMs] [状态表达式])
```

## API

- `GET /api/catalog` — 模型+动画目录
- `GET /api/model?id=<相对路径>` — 子网格几何/权重/贴图 URL
- `GET /api/anim?id=<相对路径>` — 骨架 + 逐帧轨道（q 已含 w）
- `GET /res/<相对路径>` — 原始资源（DDS 等，大小写不敏感）

## 已知边界

- 身体/时装模型大库仍在 obj_char_b/g.package（未解包），解包后重扫即入目录。
- `app.js` 末尾的 `window.__test/__sil/__dbg` 为自动化验证钩子（CDP 驱动），生产无副作用。
- 嵌入式浏览器可能不触发 rAF —— 渲染循环用 rAF+setTimeout 双保险；`?noloop` 参数降为 500ms/帧（无头截图用）。
