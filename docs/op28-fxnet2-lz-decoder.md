# fxnet2.dll 0x28 四连崩根因定案（2026-11-11 exe 轮）

## 铁证链（dump → 反汇编 → 帧字节 三重闭环）

### 1. 崩溃现场（bin64 五个 minidump 全部同址）
- `[]DUMP20261110204728468/205402312/234019115.dmp`（=3 模板实验）+ `[]DUMP20261111002744515/003319757.dmp`（v2 实验）
- 全部：**fxnet2.dll+0x10b0，AV READ at 0x0**，栈：+0x10b0 ← +0x125c ← +0x1bdae ← fxgame.exe+0x65978
- RBX=0x10/3（=匹配拷贝长度），RCX=堆缓冲，RDI=-RCX ⇒ 源基址=NULL
- 工具：~/.zcode/analysis/fxgamelogic-0x28/parse_dumps.py（纯 py minidump 解析器）

### 2. 崩溃函数 = LZ77 式字典压缩解压器（fxnet2.dll RVA 0x1000-0x1260，明文可反汇编）
```asm
入口(rcx=压缩流, rdx=输出):
  mode = src[0]; hdrlen = (mode&2)?9:3      ; 0x1018-0x104b
  r11 = 1 (标志位哨兵); rsi = dst+outlen-1; rbp = rsi-0xa (尾部阈值)
循环:
  r11==1 → r11=u32[r9]; r9+=4                ; 标志位重填
  r11&1==1 → 匹配 token:                      ; ★池回引
      edx=u32[r9]; r11>>=1
      idx=(edx>>4)&0xfff; ln=(edx&0xf)+2     ; 12位索引+4位长度(2..17)
      (edx&0xf)==0 → 扩展长度 ln=byte[r9+2]; r9+=3  else r9+=2
      rdi=pool[idx] (r13+idx*8+0x502e0)      ; ★字典查表
      rdi-=r10; 3字节步进拷贝 ln 字节         ; ★pool[idx]=NULL → mov eax,[rdi+rcx] → AV 0x0
  r11&1==0 → 字面量: n=r11&0xf; 步长查表@[r13+n*4+0x40310]
      表=[4,0,1,0,2,0,1,0,3,0,1,0,2,0,1,0]; u32[r10]=u32[r9]; 双游标按步长推进; r11>>=步长
  每次输出推进后窗口注册: idx=((u32[pos]>>12)^u32[pos])&0xfff; pool[idx]=pos; flags@0x582e0[idx]=1
  dst≥rbf(尾部) → 0x1160: 字节直拷到 rsi + 最后一次窗口注册
包装层 0x11f0(被记录字段 switch 0x1bda9 调用):
  mode=blob[0]; hdrlen=(mode&2)?4:1
  length=u32[blob+hdrlen+1] & (0xffffffff>>((4-hdrlen)*8))
  mode&1 → call 0x1000 解压; else → 0x126d 直拷(未压缩)
```
- **字典池**：.data@0x502e0（4096×qword）+ flags@0x582e0；**文件内预置 .rdata RVA（DLL 静态串）+ 运行时窗口注册混合**
- 0x28 记录字段 switch：0x1bdb7 带（tag=byte[rbp]-1，55 case，跳转表@0x1c26c）

### 3. 触发字节（崩溃帧实解，60809 会话 00:27）
解码：EE 去转义 → XOR 密钥 0xFA6D3BCC（4字节循环，internal/transport/key.go）→ op=plaintext[0]
最后一帧 0x28（227B，v2 生成器）字段全解：
```
b7 00 (State=183)  06 01 00 "stand\0"
b5 00 (Type=181)   =4
a8 05 (Resource=1448) 0e 01 00 "npc\cj_102005\0"
1d 06 (1565)       "0.0,4.767,"
a7 05 (ConfigID=1447) 13 01 00 "kuangshi_1001npc01\0"
b8 00 (Photo=184)  1b 01 00 "icon\funcnpc\cj_102005.png\0"
c0 05 (Level=1472) ...
46 06 (WeaponMode=1606) 08 01 00 "hand1_1\0"
d7 01 (HPRatio=471) =100; e4 01 (HitHPRatio=484) 64 00 00 00
★ 0x00c5-0x00c8: ...c8 08 | 80 02 | ...  ← 模板保真段携带的串池回引 token（l.pcapng a2 80/81 02 族）
```
**池回引 token 在我方会话字典中无对应 → pool[idx]=NULL → 崩。**

## 终审结论

1. **fork 客户端 0x28 路径天生是好的**——fxnet2 解码器在正常处理我们的帧（=3 与 v2 都崩在同一处=压缩串段，非帧头/时序问题）。
2. 四连崩（身份两极×时序两极全灭）的解释：四组实验都保留了模板的压缩串段 → 都带池回引 → 都崩。剩余变量 f6/模板号/事件u32 无关。
3. **v3 生成器方向（唯一正路）**：
   - 模板中所有压缩串段必须**解码后以干净形重编码**（全字面量，无匹配 token）；
   - 干净串定律 [u16 idx][u8 outlen][01 00][GBK+NUL] 已知可行（官方 108/218 帧自带干净形）；
   - 绝不重放 `80 02`/`a2 80` 族字节；
   - 下一步工具：按 0x1000-0x1260 反汇编写解码模拟器 → 跑 218 官方帧提取全部压缩段明文 → 重编码进生成器。
4. 同理警示：**任何**携带压缩串的消息都有此雷（0x0A/0x10/0x18 若官服方言也用压缩串）。

## 模拟器轮（同日续）——模型校准定版

### record 解析器全解（fxnet2 RVA 0x1bc90 起，.pdata 被 Themida 抹、CC 边界定位）
```
parse(this, cursor=rsi, end=rdi):
  tag = [cursor]
  if tag-0x27 <= 9:                    # ★ tag 0x27..0x30 = 压缩值族（op 0x28 本身就是成员！）
      scratch = this->[0x88]; scratch[0] = tag
      mode = [cursor+1]; hdrlen = (mode&2)?4:1
      outlen = u32[cursor+2+hdrlen] & mask((4-hdrlen)*8)     # ≤0xffff
      decoded = 0x11f0(blob=cursor+1, dst=scratch+1)         # blob=[mode][头][LZ流]
      assert len(decoded) == outlen
      cursor = scratch(+1)
  switch [cursor] 55 案 → 按值类型解析
```
- **0x28 帧体 = 一个完整压缩值**：`[op=0x28][mode=0x47][len-1:u32@2][LZ输出长度:u32@6][LZ流@10]`
- **[6:10] 勘误：是 LZ 解压输出长度，不是模板号**（f6=290 系巧合）
- 解压输出 = `[obj/own][4×f32][零区/嵌套压缩值...]`（官方 frame#2: 24B 头+311B 零区；帧内可嵌套 [tag 0x27-0x30][mode]... 子值）
- v2 生成器的明文头+模板段被客户端当 LZ 流解压：明文部分=字面量侥幸通过、模板拷贝段的池回引 token=`80 02` 族 → 崩

### LZ 位流格式（0x1000 全解，模拟器 lzdec.py）
```
流=[flags u32 内联重填(哨兵值1)] 交替 [token 载荷]
bit 0=字面量: 后续 unary 计数 "1"→1B "01"→2B "001"→3B "000"→4B（查表@[0x40310]）
bit 1=匹配:   u16: idx=(v>>4)&0xfff, nib=v&0xf; nib≠0→len=nib+2 消耗2B; nib=0→len=byte[+2] 消耗3B
池: 4096槽@0x502e0 全局持久, 注册 idx=((u32@pos>>12)^u32@pos)&0xfff（窗口滞后 write-3）
尾部: write≥outlen-0xb 后逐字节直拷（refill 丢 u32 置 0x80000000）
```

### 实证结果（validate_official.py）
- **崩溃帧（v2 227B）+ 官方字典 → 完整解出**（outlen=0x122, [obj/own][4f32][0]）——帧结构合法，死因=字典状态
- 官方 224 帧顺序解码：**141 OK / 83 FAIL（跨帧池回引）**——官方压缩器跨帧引用，重放需全量会话时序 → v3 必须全字面量
- op 0x27 也在崩溃抓包出现——0x27..0x30 整族=压缩值消息

### v3 生成器规格（定稿）
- **方案A（首选待验）**: mode=0x46（0x47 清 bit0）→ 纯拷贝路径：`[28][46][len-1][outlen][明文]`，客户端 memcpy 直收。真机需验无他处硬查 0x47。
- **方案B（保底）**: mode=0x47 + 全字面量 LZ 流（flags u32 全 0 位=4B 字面量串联）。
- 明文构造：[obj:u32][own:u32][4×f32:16B][记录字段区: [idx u16][tag][值]...，串值=[len u8][01 00][GBK+NUL]]
- 剩余工作：全量时序解码 l.pcapng（含非 0x28 消息的压缩值）建完整字典 → 提取官方明文语法全貌 → 生成器实现

## 资产
- minidump 解析器：parse_dumps.py；capture 解码器：decode_capture.py（EE+XOR+帧抽取）
- LZ 模拟器：lzdec.py（含 World 池模型）；官方帧验证：validate_official.py
- fxnet2 x64 副本：G:/pojie/ai/9yin-server-go/artifacts/fxgame/fxnet2_x64.dll（.text 明文；.themida 6MB 壳段不影响 0x1000-0x12000 带）
- 崩溃 dump ×5：F:/Game/Online/9yin/9yin/bin64/[]DUMP2026111*.dmp
- 崩溃会话抓包：F:/.../captures/local/127-0-0-1-{60809,51770,61547}_s2c.bin（裸 TCP，EE+XOR 编码）
- fxgame.exe（VMP 薄壳）加载链：FxCore→FxTool→FxNet2→FxGui→FxRender→FxTerrain→FxSpecial→FxWorld→FxSound→FxGameLogic→FxModel→FxModelAdv→FxPackage→FxGnugo→FxPhysics8→FxFFMpeg→FxProtect→FxMovie→fx_chat→fx_event

## v3 已落地（feat/op28-official 分支，2026-11-11）

- `cmd/protocol-probe/npc_detail_v3.go`：mode=0x46 明文方言程序化构造
  （零 LZ/零字典依赖）。字段集 `officialNPCEntityProperties`：官方 0x46 帧
  字段序，Level/BossLevel/NpcType/RotatePara/State 取 schema 真值，其余官服
  常量；串值 [len u32][GBK+NUL]，列宽按 0x09 表线型。
- 调用点 scene_lifecycle.go：`NINEYIN_NPC_DETAIL_0X28=1` 走 v3，物化(0x0D)
  之后下发（时序不变）。
- v2（0x47 模板切片）退役存档；测试 npc_detail_v3_test.go 按表列型回走
  零失步 + 身份/常量逐项断言。
- 真机待验：official2433 + NINEYIN_NPC_DETAIL_0X28=1 → NPC 渲染。
