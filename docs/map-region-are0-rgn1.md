# 地图区域数据(ARE0/RGN1)提取与格式逆向

> 2026-11-03 完成。地区范围数据全量提取 + 二进制格式破解到链式语义一步之遥。
> 状态:**数据 100% 到位,解码完成度 ~80%**,差多边形边界环的拼装语义。

## 1. 数据资产

### 1.1 区域数据全集:`F:\Game\Online\9yin\九阴资源\map\ter\<地图名>\`

来源:客户端 12 个 `map_ter_*.package`(battle/born/city/clone/misc/misc_1/scene/scene_1~4/school,共 3.9GB)逐包解压提取,临时目录已清理。

| 数据 | 数量 | 说明 |
|---|---|---|
| 地图目录 | 422 | scene01~64 主世界、born01~04 出生地、city/school/adv 副本、battle 战场等 |
| `zone_XX_YY.area` | 7,635 | 命名地区多边形(ARE0),如 `area_born01_jimingyi`(鸡鸣驿) |
| `zone_XX_YY.region` | 5,565 | 规则区域多边形(RGN1),如 `permit_stall`(摆摊)/`permit_challenge`(切磋) |
| `<图>_npc_area.ini` | 338 | 场景物件的 GUID → 地区名映射 |
| `terrain.ini` | 422 | zone 网格 → 世界坐标换算参数 |
| `map/combinemap*.ini` | 1,363 | 整图世界范围(如 born01 = 起点(-1152,-1152),3072×3072) |
| `area_color.ini` | 347 | 地区名 → 小地图 RGBA 配色 |
| `spot_text.ini` | 241 | 地图标注文本 |

每张图按 zone 瓦片(256 世界单位/格)切分,`.area`/`.region` 内含被裁剪到该 zone 的多边形,坐标为 zone 内局部坐标。

### 1.2 规则定义:`share/area/` 三件套(modern 与 九阴资源\res\share 各一份)

- `areascene.ini`:1,877 个命名地区段(`[CANGKU] info=29 name=仓库` 格式)
- `areatype.ini`:1,186 段,PK 系数/音乐/is_child 等行为参数
- `areainfo.ini`:94 套 forbid/permit 规则模板(`forbid_ride=Y`、`forbid_qinggong`、`forbid_trade`、`permit_challenge` 等)

### 1.3 解包工具

`九阴资源\jiuyin-unpack.exe --keys jiuyin-keys.txt <package> -o <输出目录>`,用法详见 `九阴资源\jiuyin-unpack.md`。

## 2. 二进制格式(已验证部分)

### 2.1 容器与记录框架 —— 100% 破译

```
偏移  内容
0x00  魔数 "FxTer\n\n\n" (8B)
0x08  payload 大小 (u32 LE)
0x0C  标签 "ARE0"(.area) / "RGN1"(.region) / "WLK1"(.walk)
0x10  u32 = 256 (常量)
0x14  u32 = 256 (常量)
0x18  u32 = 多边形计数 N
0x1C  u32 = 第一个区名的长度(含 \0)   ← 注意:它属于第一条记录,不是容器字段
0x20  记录区,每条多边形:
        名长 nlen (u32, 含\0; 首条复用 0x1C 处的值)
        名字 (nlen 字节, GBK)
        数据长 dsz (u32)
        令牌流 (dsz 字节)
      nlen=0 为空记录(仅 4B,无后续字段)
文件尾可能另有 4B(13,112 个文件解析到 len 或 len-4 完全吻合)
```

**验证**:全部 13,201 个 .area/.region 按此框架解析,消费字节数与文件长完全对齐(88 个 count=N×全零记录的退化文件单独吻合)。

### 2.2 令牌流 —— 已识别的指令

| 令牌 | 长度 | 语义 |
|---|---|---|
| `0x12`(整个流只有这一个字节) | 1 | **整 zone 覆盖**(如行宫类副本图,已证实) |
| `0x41 hi lo` | 3 | **精确坐标**:u16=(hi<<8)\|lo,坐标 = u16/256,即 zone 内 [0,256) 世界单位。连续两个 = 一个顶点 (x,y) |
| `0x10 a b c d` | 5 | 网格锚点记录,参数为奇数字节(半格量化) |
| `0x02` | 1 | 链式步进 A(全语料 60.5 万次) |
| `0x12` | 1 | 链式步进 B(53.3 万次),常成段出现如 `12 12 12 12` |
| `0x20 ...` | - | 出现在部分流开头(约 7k 次),疑为流级头部 |

**坐标系铁证**:`0x41` 曾被误读为"3 字节截断大端浮点"(算出的值恒在 [8,32),其实是解读假象);改按 u16/256 解读后,坐标完美分布于 [0,256) 且贴边值(0.996、255.9)大量出现。**zone 内坐标 + zone 文件名网格 ×256 + terrain.ini(OriginRow/Col=50) = 世界坐标**,已用 born01.path 寻路节点(x≈1421, z≈-1026)反向验证吻合。

### 2.3 未解部分(剩余 ~20%)

多边形边界环的**拼装语义**:如何把 `10` 锚点、`41` 精确坐标、`02/12` 步进序列拼成闭合边界。证据指向 Freeman 链码式压缩(步进指令占绝对多数、成段运行、锚点偶尔校准)。观察到的典型模式:

```
10 01 21 01 01 02   ← 锚点记录 + 步进
10 01 19 01 01 02
...
12 02 02 02 02 02 02 02 02 02 02   ← 长步进段收尾

10 03 03 01 01 41 00 ff 41 00 ff 12 12   ← 锚点 + 精确顶点(0.996, 0.996) + 步进
```

锚点参数为 0x01/0x03/0x09/0x0b/0x11/0x13/0x19/0x1b/0x21/0x2f…(奇数,即 2k+1),量化含义与链式方向的对应关系待定。收尾路径:①继续逆向(对拍法:找单地区跨相邻 zone 的多边形做连续性对拍);②进游戏实测(跨地区边界时客户端有地区名提示,可反推)。

## 3. 解析器参考实现

提取脚本可复用(已在全语料验证):

```python
def parse(d: bytes):
    """返回 [(地区名, 令牌流 bytes)]"""
    assert d[:8] == b'FxTer\n\n\n'
    _, _, count, nlen = struct.unpack_from('<4I', d, 16)  # 0x10 处依次为 256,256,N,首名长
    off, polys = 32, []
    for i in range(count):
        if i:
            nlen = struct.unpack_from('<I', d, off)[0]; off += 4
        if nlen == 0:
            continue                    # 空记录
        name = d[off:off+nlen-1].decode('gbk', 'replace'); off += nlen
        dsz = struct.unpack_from('<I', d, off)[0]; off += 4
        polys.append((name, d[off:off+dsz])); off += dsz
    return polys
```

## 4. res\map 清理记录(2026-11-03)

- 清理对象:`九阴资源\res\map`(旧的 map_ini+map_path 客户端解包)
- **删前安全验证**:与 `9yin_server_new\resources\modern\map` 逐文件 md5 全等(7,444 文件,0 差异),且原始 .package 包都在,误删可随时重解
- 已删:`path\`(214MB 寻路,modern 有同份)、`ini\` 下 effect/group/mdl/particles/saberarc 目录及 particles×14、role×5、lights、scene_particles 等渲染配置
- 保留(天气三件):`ini\model_weather.ini`、`ini\particles_weather.ini`、`ini\weather\`(6,626 文件,14MB)
- 结果:235MB/7,444 文件 → 14MB/6,628 文件;`scene_particles.ini`(花瓣/树叶/水花)判定为场景氛围非天气,已删,可从 modern 秒拷回
- 区域数据在独立的 `九阴资源\map\ter\`,与本次清理无交集,已复核完好

## 5. 用途与接入设想

解码完成后服务端可落地:

1. **地区判定**:玩家坐标 → 所在 zone → 遍历该 zone 的 .area 多边形做点内判定 → 得到地区名(born01 实测有 ~30 个命名地区:鸡鸣驿、杏花村、流冰河、南白山庄…)
2. **区域规则**:地区名 → areainfo 模板 → 禁骑马/禁轻功/禁交易/切磋许可/摆摊许可等(permit_* region 多边形可直接做规则区)
3. **整图范围**:combinemap + zone 网格可用于出生点合法性校验、越界检测

## 6. 相关文件

- 数据:`F:\Game\Online\9yin\九阴资源\map\ter\`(区域)、`...\res\share\area\`(规则)、`...\res\map\ini\`(仅天气)
- 工具:`F:\Game\Online\9yin\九阴资源\jiuyin-unpack.exe` + `jiuyin-keys.txt`
- 客户端场景表:`九阴资源\res\ini\scenes.ini`(场景号 → Config 路径、Grouping/MaxPlayer)
- 寻路数据(对照用):`modern\map\path\`(560 文件,FxTer 无关的独立二进制,节点数+float3+邻接表)
