# FsCollide.dll 全解与 Go 复刻（internal/fscollide）

> 权威文档。官方快照：`BGserverCJ\Debug\FsCollide.dll`
> （2014-03-22 构建，VS2003/08 Debug，配 PDB+全符号 map）。
> Go 复刻：`internal/fscollide`（测试 `go test ./internal/fscollide`，
> 探针 `go run ./cmd/fscollide-probe -map <地图目录>`）。
> 反汇编工作目录：`tmp_research/fscollide/`（fdis.py 反汇编器 + funcs.json
> 函数表 + 各函数转储）。

## 一、模块定位

Fs 框架碰撞插件（源码目录 `server\fscollide\`），`FxModule_*` 八导出挂入
FsLoader，实体名 **CollideManager**（space=system）。职责：装载服务端地图
碰撞数据并供游戏逻辑查询可行走/高度/多层地板/墙面/水域/区域。

- `CollideManager::Init(varlist)`：空实现恒 true。
- `CollideManager::CreateSceneCollide(mapDir)`：`walkmap.ini` 存在 →
  CSceneCollideFlat（**部署内无任何样本**，Flat 块内布局未逆向，Go 版显式
  报错不臆造）；否则 `terrain.ini` → CSceneCollide3D。

源文件清单（自 .rdata 路径串）：blockarea/blockchunk/blockfile/
blockheight2/blockwalk/collidemanager/heightcompress/scenecollide3d/
scenecollideflat/scenemap/terrainheight/terrainregion/terrainwalkable/
terrainzone/walkbitcompress/walkbitmap/walkflagcompress/walkflagmap/
walkflagrle/walkheightcompress/walkheightmap/walkheightrle。

## 二、容器格式（blockfile.cpp CBlockFile）

```
char signature[8]        "FxTer\n\n\n"
loop:                    （读到 EOF 自然结束）
  u32 size               数据区字节数（有符号须 ≥0）
  u32 id                 四字符小端：ARE0/HGT2/WLK1/CHK0…
  byte data[size]        由 id 对应 reader 解析
  u32 crc                官方 Load 读走但不校验
```

未知 id 块被跳过（打印 block unknown）。数据在磁盘的分布
（`Res/map/<地图>/`）：

| 文件 | 块 | 内容 |
|---|---|---|
| `zone_<row>_<col>.ground` | HGT2 + CHK0 | 地形高度四叉树 + 编辑器 chunk（服务端跳过，56B/格） |
| `walk/zone_<row>_<col>.walk` | WLK1 | 可行走/多层地板数据 |
| `zone_<row>_<col>.area` | ARE0 | 命名区域位图（LoadArea 栅格化类型图） |
| `zone_<row>_<col>.region` | ARE0 | 同格式（LoadRegion 保留命名位图，部署内未见使用） |
| `terrain.ini` | — | [system] 网格参数 |

## 三、四叉树族（核心压缩格式）

### 3.1 共同骨架（walkbit/walkflag/walkheightcompress）

节点首字节低 nibble = 类型：

- **0 分支**：高 nibble = 子树尺寸字段宽度（1=u8/2=u16/4=u32），随后
  4 个该宽度子树尺寸；子树按 **TL,TR,BL,BR** 顺序紧挨，第 n 子起点 =
  分支数据首 + 前面子树尺寸之和。查询象限：`q=(row≥half)*2+(col≥half)`。
- **1 全叶**：高 nibble 必须等于本节点 scale；随后 scale² 格原始数据。
- **2 统一**：值就地存放。

值编码差异：

| 类 | 统一(tag2) | 全叶(tag1) | 用途 |
|---|---|---|---|
| CWalkBitCompress | 0x10 位 | scale²/8 字节位图（第 i 位=i&7） | .area 区域位图 |
| CWalkFlagCompress | 高 nibble(0..15) | scale²/2 字节 nibble 对 | walk/floor 标记 |
| CWalkHeightCompress | 跟随 u16 | u16[scale²] | walk 高度数组 |
| CHeightCompress | 跟随 f32 | 裸 f32[scale²]；另有 3=u8 量化叶（f32 base+scale² u8）、4=u16 量化叶（h=base+q*0.001 毫米量化） | .ground 地形高度 |

CWalkHeightCompress 头部 `{u32 min, u32 delta}`（offset 从 8 起）；
u16 量化值解码：`0xffff→1e8(空高)、0xfffe→1e6、否则 (min+q*delta)*0.001`。
**CHeightCompress 分支无尺寸表**（四子顺序递归消费；walk 族才有表——
walk 族支持随机点查，地形族总是整树解压）。

`height_is_null(h) = h > 1e8`。

### 3.2 行 RLE（walkflagrle/walkheightrle.cpp，仅内存/校验用）

文件里 walk 数组**全部为四叉树直存**；官方 Debug 装载时另做 RLE 回压
memcmp 校验（可忽略）。RLE 格式（Go 版保留实现供完整性）：

```
u32 offsets[scale]      bit31=字面行，低 31 位=行数据偏移（行宽=下偏移差）
字面行: flag=scale/2 字节（低/高 nibble 各格）；height=scale 个裸 f32
RLE 行 : flag 记录 {u8 run, u8 val&0xf}；height 记录 {u8 run, f32 val}
```

## 四、块内布局

### HGT2（.ground）

```
u32 scale ×2            = ZoneScale+1（高度网格 257=256+1 边缘采样）
u32 comp_size
CHeightCompress 树      填 values[col*scale+row]（注意转置：第一下标=列）
```

CHK0：u32 rows, u32 cols, rows×cols×56 字节（跳过）。

### WLK1（.walk，字符串地址+数组计数双重实证）

```
u32 scale ×2            = ZoneScale*CollidePerUnit（csm001=512）
u32 floorCount          ≤15
单数组: 水面高（高度四叉树）
单数组: 行走标记（标记四叉树，值=可行走层号+1，0=阻挡）
分组×floorCount: 地板高（高度四叉树）
分组×floorCount: 空间高（高度四叉树）
分组×floorCount: 地板标记（标记四叉树）
```

**按类型分组而非按层交错**（zone_58_55 六层：H,F,H×12,F×6 实证）。
地板标记位：bit0=存在 bit1=可行走 bit2=可站立 bit3=有墙。
多层分布实测 1..15 层（5138/8358 文件为多层）。

### ARE0（.area/.region）

```
u32 scale ×2            = ZoneScale
u32 areaCount           ≤256（栅格化时 ≤254）
每区域: u32 nameLen(≤256) + name + u32 dataLen(≤1MB) + CWalkBitCompress 树
```

LoadArea 把各区域位图栅格化成类型图（区域号+1）；GetAreaType(row,col)
= 图值-1 或 -1；GetRegionEnable(名,row,col) 按名（hash+strcmp）查位图。

## 五、CSceneCollide3D（scenecollide3d.cpp）

### 5.1 terrain.ini [system] 与派生量

```
ZoneRows/ZoneCols(100) OriginRow/OriginCol(50) ZoneScale(256)
CollidePerUnit(2, ∈{1,2,4,8}) InitHeight(0) UnitSize(1, 整数∈{1,2,4,8})
WalkablePath(walk 子目录)
zoneWorld=ZoneScale*UnitSize      cell=UnitSize/CollidePerUnit
mapLeft=-(OriginCol+0.5)*zoneWorld   mapTop=-(OriginRow+0.5)*zoneWorld
```

x→列、z→行；`zones[row*ZoneCols+col]`；zone 文件名
`zone_<row>_<col>`。装载循环对每个 (row,col) 调 LoadZone（双文件皆缺
则静默跳过），随后 UpdateMaxArea 把地图包络收缩到实际 zone 集
（GetMapLeft/Top/SizeX/SizeZ 返回收缩值），再对每 zone ConnectValues：
**左邻居东缘列=本区西缘列、上邻居南缘行=本区北缘行、左上角点同步**
（本区覆写邻居；257 网格的第 257 行/列即由缝合供给）。

### 5.2 查询语义（虚接口逐条复刻）

坐标换算：碰撞格 `col=trunc((x-zoneLeft)/cell)`、
`row=trunc((z-zoneTop)/cell)`；区域格用 UnitSize 分辨率。
`FloatToInt=截断（cvttss2si）`。

- GetInZone：fx=(x-mapLeft)/zoneWorld；fx≤0 或越网格 → nil
- GetCollideEnable：zone 存在且有地形高度
- GetGroundHeight：无 zone→0；否则地形 GetY
- GetWalkHeight：walkMarker(=层号+1)>0 → 该层地板高；空高(>1e8)→地形
- GetApexHeight：自顶层向下首个 bit0 层的高度；空高→地形
- GetApexFloor：自顶层向下首个 bit0 层号；无→-1
- GetWalkEnable：无行走数据→**true**；walkMarker≠0
- GetWalkType=GetWalkEnable?1:0；GetMapType 恒 0；GetMapHeight=GetWalkHeight
- GetFloorExists/CanMove/CanStand：floorMarker bit0/1/2
- GetFloorHeight：空高→地形 GetY；GetFloorSpace：空高→1e8
- GetWallExists(row,col,y)：任一层 bit3 且该层地板高<y
- GetWalkWaterExists/Height：水面高（空高=1e8）
- GetMapRegion(名,x,z)/GetMapArea(x,z)：区域位/区域名
- LineCanWalk(mask,step,x1,z1,x2,z2)：按 step 采样，逐点 walkType&mask

地形 GetY（双线性，对角翻转）：

```
col=trunc((x-left)/unit) row=trunc((z-top)/unit)；越 scale-1 界→0
tx,tz=格内小数；tx+tz≥1 → 以 (col,row) 为基插 v1(右),v2(下)
否则以 (col+1,row+1) 为基（tx,tz 取 1-）插 v2,v1
```

### 5.3 追踪三件套

公共：格心离散（calc_line_points）——起点所在格中心起步，按"出边方向"
逐步挪格；方向分类器（0x1005B1B0，eps=FLT_EPSILON）：

- dx<0 考察西缘：贴缘平行（|ex-ez|≤eps 且 |ex|≤eps）→ 直接西行；
  否则两端点均在缘内侧（ex,ez≤0）时求直线延展与缘线交点，
  z∈[top,bottom] → 西行。东/南/北对称。
- dir 位：1=西(x-cell) 2=北(z-cell) 4=东 8=南；组合 3/6/9/12 斜向走
  L 形三采样点 ((新x,curZ),(curX,新z),(新x,新z))；5/7/10/11 官方断言不可能。
- 前置闸：|trunc(dx)|+|trunc(dz)|>256 → 失败；采样点数 <2 → 失败。

单点地板判定（traceFloorAt，三 Trace 共用）：自顶层向下
bit0 存在 → |h-lastY|≤step → bit1 可行走 → 命中；存在且高度合适但
不可行走 → 直接判死（不试低层）。

- **TraceLineWalk**：全程可走 → true 出 (x2, curY, z2)；中途无地板 →
  false 出最后合法点（首点即断 → 出起点）。
- **TraceNextFloor**：同骨架但输出层号；中途地板消失＝**成功落点**
  （报告最后合法点与层），首点即无地板才失败。
- **TraceNextSlide**：NextFloor + 地形兜底：地形高≤lastY+step → 落地；
  >lastY+2*step → 失败；其间 → 继续滑（此函数为结构级复刻，
  待 oracle 对拍收紧）。

## 六、Go 复刻清单（internal/fscollide）

| 文件 | 对应官方 |
|---|---|
| blockfile.go | CBlockFile 容器 |
| rle.go | 行 RLE（DecodeFlagRLE/DecodeHeightRLE） |
| quadtree.go | 四类四叉树整树解压+点查（QuadGetBit/Flag/Height、TerrainHeightQuad） |
| terrain.go | CTerrainHeight（HGT2+GetY）/CTerrainWalkable（WLK1）/CTerrainRegion（ARE0） |
| scene.go | CTerrainZone+CSceneCollide3D（装载/缝合/包络/全部查询） |
| trace.go | 方向分类器+格心离散+三 Trace |
| manager.go | CollideManager 路由 + SceneCollide 接口 + ini 解析 |

验证（`go test ./internal/fscollide`，全绿）：

- **66 张官方地图全量装载**（含 15 层 extreme 多层区）
- csm001 采样 18 万点：可行走 49.4%，无 NaN/空高违例
- WLK1 结构：块内字节数恰好耗尽；多层分组解析
- GetY 格点恒等；区域位图栅格化（zone_45_55 区域"991300000"）
- Manager 路由/缓存；TraceLineWalk/TraceNextFloor 冒烟

## 七、已知边界与后续

- SceneCollideFlat：部署无 walkmap.ini 样本，块内布局未逆向（Go 版
  路由命中即显式报错）。
- TraceNextSlide：结构级复刻（2×step 滑落窗口来自反汇编尾部），
  待 unicorn oracle（tools/fsverify_oracle.py 模式）逐点对拍。
- CBlockFile 写侧（SaveBegin/SaveBlock/SaveEnd）未复刻（编辑器侧，
  服务端只读）。
- .region 文件与 .area 同为 ARE0；部署内只有 .area（LoadZone 对
  .region 缺失静默容错）。
- **现代线数据源**（2016，格式未变实测兼容）：
  `九阴资源/map_ter_battle_unpacked/res/map/ter`＝战场地图 12 张
  （battle02/banghui/gmp/kof/myz/myz2/new/ngm/pingheng/wd/ym/zb），
  四件套齐全，TestModernBattleMaps 全量装载零错；
  `九阴资源/resources/map/ter`＝422 张图但只有 terrain.ini+.area
  （客户端/编辑器树，无 .ground/.walk——区域查询可用，行走/高度需
  服务端导出）。`resources\modern`＝纯客户端树，零碰撞文件。
- **整包解压与服务端整理集**（2026-11-19）：`九阴资源/map` 下 10 个
  .package（PCK0 v15）已用 jiuyin-unpack 全解至 `map/map_ter_unpacked/`
  （约 7.0GB）；其中 `map_ter_city.package`
  索引加密 key_id **0xCE95** 不在原 31 键表，已用 2023 明文同名包
  （`20230423/9yin/res/map_ter_city.package`，cnt 5185）做已知明文攻击
  破出 149 字节周期键并入两张 jiuyin-keys.txt。服务端碰撞集已整理到
  `九阴资源/map/server_ter/`：**422 张地图 / 47994 文件 / 4.26GB**
  （terrain.ini + zone_*.ground/.area/.region + walk/zone_*.walk），
  TestServerTerAllMaps 全量装载验证。
