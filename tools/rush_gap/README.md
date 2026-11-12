# rush_gap —— 冲锋自走差集取证工具链（2026-11-12）

官服 `技能.pcapng` + `登陆-all.pcapng` 与本服对照的帧级取证工具。定版结论与
差集清单见 `docs/冲锋自走触发链官服定版与差集.md`；背景文档
`docs/技能pcapng移动交互与冲锋官服分析.md`、`docs/官服玩家档案0x28全量帧.md`。

复用：`tools/pcap_analysis/`（pcaplib 帧解码 + decode 描述器）、`tools/op28/`
（lzdec LZ 解压器）。全部脚本 `python -I` 运行、路径自举（不依赖 cwd）。

## 管线

```
pcapng --pcap_frames--> frames.pkl --pcap_events--> events.json --walk_correlation--> 按下/自走/簇对齐表
                          |--dump_frames--> 时间窗帧字节 hexdump
                          |--npc28_decode--> 0x27-0x2a 族 LZ 会话池解码 + 0x28 字段走查
fxgamelogic.dll --xdis--> 明文 .text 反汇编（属性名 xref 注释）
```

## 用法

```bash
cd 本目录（输出物落在当前工作目录）

# 1. pcapng → 重组帧库（TCP seq 去重 + ISN 回退>1MB 切 epoch——登陆-all 双连接必需）
python -I pcap_frames.py E:\Workspace\project\game\adata\official\技能.pcapng jn.pkl
python -I pcap_frames.py E:\Workspace\project\game\adata\official\登陆-all.pcapng login.pkl --ports 7074
#    时间锚=保留流的最早 TCP 载荷包；--ports 过滤 HTTPS 噪声并固定锚点。

# 2. 事件流（老 tools/pcap_analysis/decode.py 对双连接包会崩，此处为加固版）
python -I pcap_events.py jn.pkl jn_events.json

# 3. 冲锋按下/自走/受理簇三方对齐（定版：按下即走/簇到即走双模式）
python -I walk_correlation.py jn_events.json

# 4. 时间窗帧字节（例：受理簇四帧 + 0x21 回显）
python -I dump_frames.py jn.pkl 32.64 32.65 0x10,0x1e s2c
python -I dump_frames.py jn.pkl 32.79 32.81 0x21 s2c

# 5. 0x28 详情帧 LZ 解码（会话池：全帧明文滑窗种子）
python -I npc28_decode.py jn.pkl                    # 概览：全部 0x28 对象/位姿/字段名
python -I npc28_decode.py jn.pkl --at 30.456        # 逐字段值
python -I npc28_decode.py login.pkl --obj 0x1100163b

# 6. fxgamelogic.dll 明文带反汇编（加壳段不可静态解）
python -I xdis.py 0xc891f5 100 --dll <fxgamelogic.dll 路径>
```

## 脚本一览

| 脚本 | 作用 |
|---|---|
| pcap_frames.py | pcapng → TCP seq 重组抽帧 → frames pkl（epoch 切分/重叠去重/帧时间戳） |
| pcap_events.py | frames pkl → 事件 json（0x09 表先行、describe 异常容忍） |
| dump_frames.py | frames pkl + 时间窗 + op 过滤 → 帧字节 hexdump |
| npc28_decode.py | 0x27-0x2a 族 LZ 会话池解码（World2）+ 0x28 字段走查（概览/--at/--obj） |
| walk_correlation.py | 每次出招的 211→首条 m15 state7→受理簇时延对齐与模式分类 |
| xdis.py | fxgamelogic.dll 明文 .text 反汇编，.rdata 字符串立即数自动注释 |

## 已知边界

- npc28_decode 的 LZ 会话池对官方 0x47 帧通过率非 100%（残余 miss=客户端本地
  种子）；部分解码输出尾部字段带伪影，对象归属/位姿/字段名可靠，字段值以
  `--at` 全解样本为准。
- xdis 仅覆盖明文 .text/.rdata 带（字符串/技能表加载/轻功闸）；接近状态机
  主体在 themida 壳段内，静态不可达。
