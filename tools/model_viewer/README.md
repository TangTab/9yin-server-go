# 九阴角色模型浏览器

浏览游戏资源根（`obj\char` / `obj\weapon` / `obj\effect*`）中的角色/武器/特效网格、贴图、骨骼动画，
任意部件 × 任意动画自由配对，带服务器资源表中文名映射。
纯 Python 标准库后端 + three.js 前端，无 npm 依赖。

## 启动

```bash
cd tools/model_viewer
python server.py            # 默认 127.0.0.1:8901
# 打开 http://127.0.0.1:8901
```

**资源根 `--res`**（默认 `F:\Game\Online\9yin\res`，或环境变量 `NINEYIN_RES_ROOT`）：
其余所有路径都相对它推导 — `obj\char`=角色、`obj\weapon`=武器、
`obj\effect` / `obj\effect_sns` / `obj\effect_weapon`=特效（三树进目录，id 加 `effect*/` 前缀，
分类「特效 / 特效·SNS / 特效·武器」；obj/effect 平铺 1.7 万文件中约 1 成可解析为网格，
其余粒子/物理件按状态标记 ✕）。
单个子根仍可显式覆盖：`--root <角色目录>`、`--weapons <武器目录>`
（或 `NINEYIN_CHAR_ROOT` / `NINEYIN_WEAPON_ROOT`）。
中文名映射表默认取本仓库 `resources/`（`--resources` / `NINEYIN_RESOURCES` 覆盖）。

首次访问 `/api/catalog` 时构建目录（char+weapon ~1.1 万 + effect ~1 万全量解析 + 名称映射，约 3–6 分钟），之后常驻内存。

## 界面

三栏布局：**左 = 部件库+武器 · 中 = 视口 · 右 = 骨骼动画**。

- **性别总标签**（左栏顶部 男/女）：一键切换部件库、动画库过滤与默认装配；
  切换时自动卸下异性别部件并装配该性别**默认身体+默认脸型**（男 b_beiyuan06+b_face，女 g_beiyuan06+g_face）。
  性别以**文件名前缀**判定（b_=男 g_=女，优先于目录名——b_xiyu10/g_face.xmod 这类目录名会骗人）。武器不受性别影响。
- **互斥槽位**：武器/发型/胡子/脸(含特殊脸·蒙面头)/面饰/面具/头饰/上衣/下装/鞋/披风 各一件，
  同槽装配新件**自动替换旧件**（时装按类型各占一槽）；未知类别按类别名互斥。
- **部件库**（左栏上部）：按部件类别分组（身体/套装、上衣/时装、下装、鞋、头饰、发型、脸型…；
  `_h` 后缀 = 发型，实测全 BoneHAIR* 骨），分组默认折叠 + 搜索；
  条目显示**中文名**（服务器资源表映射，橙色）+ 灰色文件名 + 顶点数。
- **武器栏**（左栏底部，独立于性别标签）：obj_weapon 包 2243 件，按类型分组
  （剑 439 / 刀 319 / 长兵 212 / 双刺 / 斧 / 拳套 / 刺 / 短刀 / 环 / 爪 / 扇 …），标题栏可整体收起；
  装配后武器**挂接在角色右手骨 `Bip01 R Hand`**（握柄建模在原点，随摆臂刚体运动）。
- **多部件装配**：点击任意条目即装配/卸下（✓ 标记），身体+脸+发型+头饰+武器可同时渲染 ——
  部件共享同一套标准骨架（按骨名绑定），视口底部 chips 列出已装部件（✕ 卸下，显示中文名），
  "清空"一键卸下全部。
- **骨骼动画栏**（右栏）：12,000+ 动画按**分类折叠分组**（默认全折叠）+ 搜索，虚拟滚动；
  显示帧数/骨数，未支持版本标 ✕；
  **播放控制固定在右栏底部**（播放/暂停空格、0.1–4× 倍速、帧拖动、循环、帧号/时间）。
- **视口**：轨道相机；骨架叠加 / 线框 / 贴图开关；网格与所选动画骨架按骨名匹配，
  匹配率 <50% 时保持绑定姿态并提示。贴图按 `_d`/`_s`/`_n` 后缀映射 diffuse/specular/normal。
  **缺失骨回退**：动画骨架没有的骨名（如面部表情骨 Bone01-43 只存在于脸专属骨架）
  回退挂到该子网格权重最大的已匹配骨；子网格全未匹配（眼球 submesh）再取整个部件的
  最大已匹配骨（脸→Bip01 Head）——否则这些顶点只跟根位移，跑步时脸会滞后错位、眼球飞出眼眶
  （蒙皮重建必须传递 `userData.partId`，丢失会把全部件并成一组、回退骨退化成全身最大权重骨）。
  **刚体挂件**：无骨名 blob 网格（脸饰 facecloth 等占位权重 `[0,1.0]`）与武器同为原点建模——
  头部件类（face/hair/helmet/mask）挂 Bip01 Head 随头刚体运动，特效/测试件留原地。
  任意装配 × 任意动画自由配对（例：脸+面饰+身体+武器 × `skill/` 下任意招式）。

## 中文名映射链（resindex.build_name_index）

```
equipment.ini [QName] ──ArtPack──▶ itemartstatic.ini [id]
                                      │ MaleModel/FemaleModel = obj\char\<目录>\<文件>   (部件)
                                      │                或     = <武器目录名>            (武器/复合引用)
                                      ▼
catalog 模型 id / 武器目录 ──匹配──▶ 中文名 (文本表 QName/Name 键, UTF-8, 去 HTML 标签)
```

文本表 = `resources/modern/text/chineses/*`（UTF-8，86 万条）；命中 5425/8892 模型
（部件 4087 + 武器 1338，如 天迹 / 玄机 / 古墓仙域服 / 开刃剑 / 戮天斧·我杰）。
未命中条目回退显示文件名。

## 逆向格式支持

| 格式 | 版本 | 状态 |
|---|---|---|
| XVAM (.xmod) | 0x1002 | 全解 8666/8892（char 6522 + weapon 2144；顶点步长 32/56B 自适应、UV 恒在记录末 8B、u16 三角列表、4 槽蒙皮权重或无权重刚体、骨名 blob、材质区自适应） |
| XVAM (.xmod) | classId=3 节点链 | 武器 dummy/粒子点（w0_start/pose_*/L_Weapon*）：节点布局不规则，按 cls∈{1,2}+tag 候选扫描+整文件验证跳过（ rescued 766 件）；纯节点文件不支持 |
| XVAM (.xmod) | 0x1000 | test/ 目录 5 件，顶点布局未解，标记不支持 |
| XSAM (.xskt) | 0x1005 | 全解（递归骨树、bind 变换、B 轨道 0/1/2 三型、C 轨道开关、挂点 44B+18B） |
| XSAM (.xskt) | 0x1003 / 0x2000 / 0x3000 / 0x5000 | 布局变体未全解（约 250 个 diedown 等），标记不支持 |
| DDS | DXT1/3/5 | 传统格式全支持（three.js DDSLoader） |

关键定版细节（详见 `format/xmod.py` / `format/xskt.py` 文件头注释）：

- **i16×3 轨道旋转 = 欧拉角，非四元数量化**（引擎逆向铁证：fxmodeladv_rebuilt.dll @0x52a2
  常量 π/2/65535 + `d3dx9!D3DXQuaternionRotationYawPitchRoll(out, Yaw=s1, Pitch=s0, Roll=s2)`）：
  `angle = i16 × 2π/65535`，`quat = qy(yaw)·qx(pitch)·qz(roll)`。
  验证：tpose/run/stand/jump 全 112 骨 f0 与 bind 偏差 0.0°（f0 即 bind 参考帧，f1..N−1 为动画本体）
- **B 轨道型**：0=6B 纯欧拉；2=18B `[欧拉 i16×3][pos f32×3]`；1=18B `[欧拉 i16×3][scale f32×3]`
  （欧拉在前，f32 尾巴在后）；A=3 B=6 挂点 44B bind + 18B 轨道；C=轨道开关（half 系 0=无轨道）
- **顶点步长**（身体/时装大库定版）：vflags(首 u16) bit0x2000 → 56B，否则 32B（另有 36/24B 罕见变体）；
  **UV 恒在记录末 8 字节**（32B→@24、56B→@48；56B 的 f6..f10 为切线/次级通道，
  以三角形 UV 连续性实测定位 —— 用错通道会渲染成平灰色）
- **对象 classId**：1/2=网格（结构相同，A 字段实为对象 ID 可为任意整数）；3=ragdoll 物理骨架不支持
- **子网格 sflags=0x20 系**（facecloth 等）**无权重块**，索引后直接材质区（刚体网格）
- 解析为**校验驱动**：步长×权重假设候选逐个回溯重试，校验点 = 索引数/越界、blob 偏移、
  材质浮点、子网格边界（非末尾子网格后必须紧跟下一子网格头）、对象尾到 EOF 精确
- xmod 权重的 boneRef = 骨名 blob 内**字节偏移**（非序号）；空槽 = 0xFFFFFFFF
- 材质区 = 6×0.588 浮点 + 0~5 组附加浮点（specular(1,1,1)/gloss 等）+ 0~4 个 .dds 串
  （串常成对重复；槽位语义按后缀：`_d`=diffuse `_s`=specular `_n`=normal）
- 对象尾部绑定表三形态：纯对 / 条目式（tpose 的 264 条穴位挂点，骨→穴名+40B 变换）/ 不透明跳过
- **贴图方向**：DDS 压缩块序与 D3D 一致（v=0=顶行），UV 原样使用不翻转；模型正面朝 **−Z**（b_face 法线实测），默认相机在 −Z 侧

引擎逆向工具见 `re_tools/`（capstone 反汇编 disa.py、riprel 交叉引用 xref.py、导入表 imports.py、pdata 函数表 funcs.py）；
引擎内部格式为 .gm2/.bon/.an2（fxmodel），XVAM/XSAM 为资源流格式。

## 组件

```
format/xmod.py     XVAM 解析器        format/xskt.py   XSAM 解析器
format/ddsinfo.py  DDS 头探测         resindex.py      目录/索引链扫描 (catalog.json)
server.py          HTTP 服务+API      static/          three.js 前端 (vendor 已内置)
test_format.py     解析器回归测试 (16 用例, NINEYIN_CHAR_ROOT 指向资源根即激活)
shot.js            无头 Chrome CDP 截图 (node shot.js <url> <out.png> [waitMs] [状态表达式])
drive.js           无头 Chrome CDP 驱动 (node drive.js <url> <out.png> '<expr>'… 逐步执行+截图)
```

## API

- `GET /api/catalog` — 模型+动画目录
- `GET /api/model?id=<相对路径>` — 子网格几何/权重/贴图 URL
- `GET /api/anim?id=<相对路径>` — 骨架 + 逐帧轨道（q 已含 w）
- `GET /res/<相对路径>` — 原始资源（DDS 等，大小写不敏感）

## 已知边界

- 约 110 件 char + 99 件武器 .xmod 尚未解出（多布局混排/纯节点/0x1001），目录中标错误徽章。
- 武器挂手骨为握柄原点假设（正确的挂点骨 `Tpose_handR_01`/`Tpose_back_01` 只在 434 骨的
  tpose.xskt 里、动作骨架没有），剑类握柄在原点基本正确，斧等个别武器角度/位置会偏；
  故**不默认装配武器**，需要时手动点选。武器自身的 .xskt/composite.ini 染色变体未接入。
- 动画 f0 为 bind 参考帧（f1..N−1 为播放区间）；run 等大摆幅动作的中段帧观感夸张为数据原貌
  （CPU/GPU 蒙皮逐顶点对拍 + 独立 Python 复算三方一致）。
- 部件卸载/换骨架的关键约定：`convertToSkinned` 重建蒙皮网格后必须把新网格**写回各部件**，
  否则卸载拿到陈旧引用（模型删不掉/新旧骨架混挂动作不同步）——见 app.js 内注释。
- `app.js` 末尾的 `window.__ready/__test/__sil/__dbg/__probe/__tex/__cpuskin` 为自动化验证钩子（CDP 驱动），
  生产无副作用；`__ready` 为目录+默认装配完成信号，自动化先 `await` 它。
- 嵌入式浏览器可能不触发 rAF —— 渲染循环用 rAF+setTimeout 双保险；`?noloop` 参数降为 500ms/帧（无头截图用）。
