## a.pcapng — 49.4.14.150:7074 侠骨丹心

场景：冲锋/范围技打怪 + 区域boss进度 + 打坐 + 末尾设置上报。事件 3177 条 → 合并为 2268 段。抓包起点 2026-11-08 13:24:39（本地墙钟）

| 时间 | 方向 | 交互 |
|---|---|---|
| 0.0s | S2C | S2C m294 MSG_DELAY回显[client ms, obj, 服务器unix秒, uptime] [72290612, obj(0xea540de1,0xc38), 1794115479, 46334.559] |
| 0.0–137.2s | PAIR | C2S ping(0x18) → S2C 0x38 回显 ×127 |
| 1.0–137.1s | PAIR | C2S m924 MSG_DELAY(client ms) → S2C m294 回显[client ms, obj, 服务器unix秒, …] ×136 |
| 3.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 3.8s | C2S | m15 移动上报 自动 state=5 pos=(864.6,2.3,1118.2) ×2 |
| 4.5s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x701e5a4] |
| 5.1s | C2S | m54 格挡状态(值2679078047) |
| 5.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 5.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 5.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 5.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 5.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 5.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 5.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 5.3s | C2S | m15 移动上报 手动 state=4 pos=(864.6,2.3,1118.2) |
| 5.4–8.6s | S2C | 对象同步0x21[0x701e5a4] 码43 len=47 ×6 |
| 5.7s | C2S | m54 格挡状态(值2455038072) |
| 5.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 5.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 5.7s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 5.7s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 5.7s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 5.7s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 5.7s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 5.7s | C2S | m54 格挡状态(值2455038072) ×3 |
| 5.9s | C2S | m15 移动上报 自动 state=4 pos=(864.6,2.3,1118.2) |
| 6.4s | C2S | m54 格挡状态(值2679078047) |
| 6.4s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 6.4s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 6.4s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 6.4s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 6.4s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 6.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 6.4s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 6.8–8.4s | C2S | m15 移动上报 手动 state=4 pos=(864.6,2.3,1118.2) ×4 |
| 8.8s | C2S | m15 移动上报 手动 state=7 pos=(864.6,2.3,1118.2) |
| 8.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 9.0s | S2C | 对象同步0x21[0x701e5a4] 码155 len=47 |
| 9.1s | C2S | m54 格挡状态(值2455038072) |
| 9.2s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 9.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 9.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 9.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 9.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 9.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 9.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 9.2s | C2S | m82 ? [3] |
| 9.2s | C2S | m54 格挡状态(值2455038072) ×2 |
| 9.3s | C2S | m15 移动上报 手动 state=7 pos=(864.3,2.2,1117.6) |
| 9.3s | C2S | m54 格挡状态(值2455038072) |
| 9.4s | C2S | m15 移动上报 手动 state=8 pos=(864.0,2.0,1116.7) |
| 9.4s | S2C | 对象同步0x21[0x701e5a4] 码1 len=47 |
| 9.7s | S2C | 对象同步0x21[0x701e5a4] 码207 len=47 |
| 9.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 9.7s | C2S | m15 移动上报 手动 state=5 pos=(863.5,3.2,1115.5) ×2 |
| 9.8s | C2S | m15 移动上报 手动 state=6 pos=(863.4,3.4,1115.3) |
| 9.9s | S2C | 对象同步0x21[0x701e5a4] 码36 len=47 |
| 9.9s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [863.312, 3.482, 1114.976, -2.77] |
| 9.9s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x701e5a4] |
| 9.9s | S2C | 位置0x1F(短len=25) |
| 9.9s | S2C | S2C m337 施法回显[0x3ad5fb0701e5a4] [863.089, 3.489, 1114.405, 3.828] |
| 9.9s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID=CS_jh_hyjf07 |
| 9.9s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 冲锋动画 [858.229, 3.277, 1108.473, 2, "qg_fastruning_s_2h_01"] |
| 9.9s | S2C | 属性更新0x10[O#0x3ad5fb] CantMove=1 |
| 9.9s | S2C | 属性更新0x10[O#0x3ad5fb] CantParry=1 |
| 9.9s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +93行 |
| 9.9s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =23808 0x701e5a4] ×4 |
| 9.9s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 9.9s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +94行 |
| 9.9s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x701e5a4] ×4 |
| 9.9s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=99 |
| 9.9s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10298 |
| 9.9s | S2C | S2C m707 受理族变体 [15] |
| 9.9s | C2S | m15 移动上报 手动 state=6 pos=(863.1,3.5,1114.4) |
| 10.1s | S2C | 对象同步0x21[0x701e5a4] 码163 len=47 |
| 10.4s | C2S | m212 冲锋/续招落地 [858.335, 3.283, 1108.602, -2.455] |
| 10.4s | S2C | 属性更新0x10[O#0x3ad5fb] CantMove=0 |
| 10.4s | S2C | 属性更新0x10[O#0x3ad5fb] CantParry=0 |
| 10.4s | S2C | 位置0x1F(短len=25) |
| 10.4s | S2C | S2C m26 技能释放 [] |
| 10.4s | S2C | 位置0x1F(短len=25) |
| 10.4s | S2C | S2C m337 施法回显[0x3ad5fb0701e5a4] [858.335, 3.283, 1108.602, 3.828] |
| 10.4s | S2C | S2C m20 范围施放标记[0x3ad5fb0701e5a4] ["CS_jh_hyjf07", 0, 0, 0.0, 857.278, 2.428] |
| 10.4s | C2S | m15 移动上报 手动 state=8 pos=(858.2,3.3,1108.5) |
| 10.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 10.5s | C2S | m15 移动上报 手动 state=4 pos=(858.3,3.3,1108.6) |
| 10.5s | C2S | m15 移动上报 手动 state=7 pos=(858.3,3.3,1108.6) |
| 10.5s | S2C | S2C m586 状态推送(簇首) [5] |
| 10.6s | S2C | 对象同步0x21[0x701e5a4] 码170 len=135 |
| 10.8s | S2C | op31(sub1) 0x703afff |
| 10.8s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 10.8s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID= |
| 11.6s | C2S | m15 移动上报 自动 state=8 pos=(858.3,2.8,1108.6) |
| 11.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 11.8s | S2C | 对象同步0x21[0x701e5a4] 码112 len=47 |
| 12.0–12.2s | C2S | m15 移动上报 自动 state=4 pos=(858.3,2.3,1108.6) ×2 |
| 12.2–12.5s | S2C | 对象同步0x21[0x701e5a4] 码111 len=47 ×2 |
| 12.3s | C2S | m15 移动上报 手动 state=7 pos=(858.3,2.3,1108.6) |
| 12.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 12.5s | S2C | op06 len=1 06 |
| 13.0s | C2S | m211 使用技能(出招) CS_jh_hyjf05 10值 [857.196, 2.626, 1105.684, -2.77] |
| 13.1s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x701e5a4] |
| 13.1s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID=CS_jh_hyjf05 |
| 13.1s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +94行 |
| 13.1s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x701e5a4] ×4 |
| 13.1s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 13.1s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =16801024 0x701e5a4] ×2 |
| 13.1s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=98 |
| 13.1s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10199 |
| 13.1s | S2C | S2C m20 范围施放标记[0x3ad5fb0701e5a4] ["CS_jh_hyjf05", 0, 0, 0.0] |
| 13.1s | S2C | 对象同步0x21[0x7007c67] 码50 len=223 |
| 13.1s | S2C | op2E len=230 2e47e5000000a4010000000000c002005fc20307 |
| 13.1s | C2S | m15 移动上报 手动 state=4 pos=(857.1,2.6,1105.4) |
| 13.3s | S2C | 对象同步0x21[0x701e5a4] 码118 len=47 |
| 13.4s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 13.4s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID= |
| 13.8s | C2S | m15 移动上报 手动 state=7 pos=(857.1,2.6,1105.4) |
| 13.8s | S2C | op28(子码47) len=242 ×3 |
| 13.9s | S2C | 对象同步0x21[0x701e5a4] 码79 len=47 |
| 13.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 14.2s | S2C | 属性更新0x10[O#0x3ad5fb] Sat=39 |
| 14.3s | C2S | m298 刷新NPC任务特效 [0, obj(0x70012d8,0x12f1)] |
| 14.3s | S2C | S2C m209 ? [15, obj(0x70012d8,0x12f1), 0] |
| 14.4s | S2C | 属性更新0x10[O#0x3ad5fb] FightData=19088929605 |
| 14.6s | S2C | 属性更新0x10[V#0x2] LeftTime=603378 |
| 14.6s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=99 |
| 14.6s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10299 |
| 14.7s | C2S | m211 使用技能(出招) wuji_CS_jh_hyjf04 10值 [855.685, 3.297, 1101.813, -2.77] |
| 14.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x701e5a4] |
| 14.7s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID=wuji_CS_jh_hyjf04 |
| 14.7s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillTarget=None |
| 14.7s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +94行 |
| 14.7s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x701e5a4] ×4 |
| 14.7s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 14.7s | S2C | 记录写格0x13[cooldown_rec#109 行92 列1 =16800768 0x701e5a4] ×2 |
| 14.7s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=98 |
| 14.7s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10267 |
| 14.7s | S2C | S2C m20 范围施放标记[0x3ad5fb0701e5a4] ["wuji_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb), 0, 0.0] |
| 14.8s | C2S | m15 移动上报 手动 state=4 pos=(855.6,3.3,1101.5) |
| 14.8s | S2C | 对象同步0x21[0x701e5a4] 码45 len=47 |
| 14.9s | S2C | op31(sub1) 0x700f894 |
| 15.1s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.7 |
| 15.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.8 |
| 15.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA546FE9,1 |
| 15.1s | S2C | S2C m169 buff添加 ["buf_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb)] |
| 15.1s | S2C | S2C m169 buff刷新? ["buf_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb)] |
| 15.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA546FE9,A |
| 15.1s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=6.0 |
| 15.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=2.55 |
| 15.1s | S2C | S2C m205 伤害目标列表[0x3ad5fb0701e5a4] ["wuji_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb)] |
| 15.1s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID= |
| 15.1s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillTarget=None |
| 15.6s | C2S | m15 移动上报 手动 state=7 pos=(855.6,3.3,1101.5) |
| 15.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 15.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 15.8s | C2S | m15 移动上报 手动 state=5 pos=(855.4,3.4,1101.1) ×2 |
| 15.9s | S2C | 对象同步0x21[0x701e5a4] 码6 len=47 |
| 16.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA5473DE,9 |
| 16.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=2.36 |
| 16.5s | C2S | m15 移动上报 自动 state=4 pos=(853.8,3.6,1097.0) |
| 16.5s | S2C | 对象同步0x21[0x701e5a4] 码245 len=47 |
| 16.9s | S2C | op28(子码47) len=219 ×3 |
| 17.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA5473DD,8 |
| 17.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=2.16 |
| 17.6s | C2S | m15 移动上报 手动 state=7 pos=(853.8,3.6,1097.0) |
| 17.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 17.8s | S2C | 对象同步0x21[0x701e5a4] 码28 len=47 |
| 17.8s | C2S | m15 移动上报 手动 state=7 pos=(854.3,3.6,1098.3) |
| 18.0s | S2C | 对象同步0x21[0x701e5a4] 码210 len=47 |
| 18.1s | C2S | m15 移动上报 自动 state=4 pos=(855.0,3.5,1100.1) |
| 18.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA5473DD,7 |
| 18.1s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.87 |
| 18.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.97 |
| 18.3s | S2C | 对象同步0x21[0x701e5a4] 码70 len=47 |
| 18.4s | C2S | m15 移动上报 手动 state=7 pos=(855.0,3.5,1100.1) |
| 18.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 18.5s | S2C | 对象同步0x21[0x701e5a4] 码31 len=47 |
| 18.6s | C2S | m15 移动上报 手动 state=7 pos=(854.6,3.6,1099.0) |
| 18.7s | S2C | 对象同步0x21[0x701e5a4] 码150 len=47 |
| 18.7s | C2S | m211 使用技能(出招) CS_jh_hyjf05 10值 [854.34, 3.59, 1098.368, -2.77] |
| 18.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x701e5a4] |
| 18.7s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID=CS_jh_hyjf05 |
| 18.7s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +94行 |
| 18.7s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x701e5a4] ×4 |
| 18.7s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 18.7s | S2C | 记录写格0x13[cooldown_rec#109 行92 列1 =16800768 0x701e5a4] ×2 |
| 18.7s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=97 |
| 18.7s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10168 |
| 18.7s | S2C | S2C m20 范围施放标记[0x3ad5fb0701e5a4] ["CS_jh_hyjf05", 0, 0, 0.0] |
| 18.8s | C2S | m15 移动上报 手动 state=4 pos=(854.2,3.6,1098.0) |
| 18.9s | S2C | 对象同步0x21[0x701e5a4] 码193 len=47 |
| 19.1s | S2C | 属性更新0x10[V#0x28] CurTargetCount=2 |
| 19.1s | S2C | 属性更新0x10[O#0x3ad5fb] SP=2 |
| 19.1s | S2C | 属性更新0x10[O#0x3b015d] LogicState=1 |
| 19.1s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 19.1s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=1 |
| 19.1s | S2C | 属性更新0x10[O#0x3afb5f] LogicState=1 |
| 19.1s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +1行 |
| 19.1s | S2C | S2C m1 系统提示 [2, "8066"] |
| 19.1s | S2C | S2C m24 通用事件[0x3b015d0702ccb3] 伤害飘字 [1500, 0, 2, "CS_jh_hyjf05", "SkillNormal", "monm031304"] |
| 19.1s | S2C | S2C m18 战斗伤害帧[0x3b015d0702ccb3] ["hurt_half", 0, obj(0x701e5a4,0x3ad5fb)] |
| 19.1s | S2C | 属性更新0x10[O#0x3ad5fb] SP=4 |
| 19.1s | S2C | S2C m24 通用事件[0x3afb5f07013793] 伤害飘字 [1500, 0, 2, "CS_jh_hyjf05", "SkillNormal", "monm031304"] |
| 19.1s | S2C | S2C m18 战斗伤害帧[0x3afb5f07013793] ["hurt_half", 0, obj(0x701e5a4,0x3ad5fb)] |
| 19.1s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 怒气段 [2] |
| 19.1s | S2C | S2C m205 伤害目标列表[0x3ad5fb0701e5a4] ["CS_jh_hyjf05", obj(0x702ccb3,0x3b015d), obj(0x7013793,0x3afb5f)] |
| 19.1s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID= |
| 19.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA5473F1,6 |
| 19.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.67 |
| 19.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.77 |
| 19.3s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=98 |
| 19.3s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10268 |
| 19.4s | S2C | 对象同步0x21[0x702ccb3] 码102 len=179 |
| 19.4s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 动作开始 ["buff_ng_xyhxj01", obj(0x701e5a4,0x3ad5fb)] |
| 19.4s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo5=4BF9,701E5A4,3AD5FB,5,C38EA549417,1 |
| 19.4s | S2C | S2C m169 buff添加 ["buf_ng_jz_006_01", obj(0x701e5a4,0x3ad5fb)] |
| 19.4s | S2C | S2C m169 buff刷新? ["buf_ng_jz_006_01", obj(0x701e5a4,0x3ad5fb)] |
| 19.4s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo5=4BF9,701E5A4,3AD5FB,5,C38EA549417,5 |
| 19.5s | S2C | 属性更新0x10[O#0x3ad5fb] PhyDamageMul=1.5 |
| 19.5s | S2C | 属性更新0x10[O#0x3ad5fb] StiDamageMul=1.5 |
| 19.5s | S2C | 属性更新0x10[O#0x3ad5fb] JujDamageMul=1.5 |
| 19.5s | S2C | 属性更新0x10[O#0x3ad5fb] NegDamageMul=1.5 |
| 19.5s | S2C | 属性更新0x10[O#0x3ad5fb] MasDamageMul=1.5 |
| 19.5s | S2C | 属性更新0x10[O#0x3ad5fb] UnderWaterModifySpeed=1.0 ×2 |
| 19.6s | S2C | 多属性0x2D(非标子码02) len=67 2d0200933701075ffb3a000300a001a4 |
| 19.8s | C2S | m54 格挡状态(值4489905844) |
| 19.8s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=2.94 |
| 19.8s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-0.96 |
| 19.8s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 19.8s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 19.8s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo6=11CE,701E5A4,3AD5FB,1,0,0 |
| 19.8s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 19.8s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 20.0s | S2C | 属性更新0x10[O#0x3b015d] State= |
| 20.2s | S2C | 对象同步0x21[0x702ccb3] 码42 len=47 |
| 20.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA5473E3,5 |
| 20.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=2.75 |
| 20.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-1.15 |
| 20.2s | C2S | m54 格挡状态(值4265865869) |
| 20.2s | C2S | m15 移动上报 手动 state=7 pos=(854.2,3.6,1098.0) |
| 20.2s | C2S | m54 格挡状态(值4265865869) |
| 20.2s | C2S | m15 移动上报 手动 state=7 pos=(854.2,3.6,1098.0) |
| 20.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 20.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo6= |
| 20.3s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 20.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.48 |
| 20.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.58 |
| 20.3s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 20.3s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 20.3s | S2C | 属性更新0x10[O#0x3ad5fb] LiveGroove_5=42786 |
| 20.3s | S2C | S2C m1 系统提示 [1, "8052", 149] |
| 20.3s | S2C | S2C m349 ? [149] |
| 20.3s | S2C | 属性更新0x10[O#0x3ad5fb] Faculty=114282922 |
| 20.3s | S2C | 属性更新0x10[O#0x3ad5fb] HoldFaculty=1500644 |
| 20.3s | C2S | m54 格挡状态(值4265865869) ×2 |
| 20.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 20.3s | C2S | m54 格挡状态(值4265865869) |
| 20.3s | C2S | m15 移动上报 手动 state=7 pos=(854.3,3.6,1098.2) |
| 20.4s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 20.4s | S2C | op2C len=156 2c479b0000000a010000000000e00600b3cc0207 |
| 20.8s | S2C | 对象同步0x21[0x7013793] 码24 len=91 |
| 20.9s | C2S | m15 移动上报 自动 state=4 pos=(855.5,3.4,1101.3) |
| 21.0–23.0s | S2C | 对象同步0x21[0x701e5a4] 码63 len=47 ×3 |
| 21.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA5473D8,4 |
| 21.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.28 |
| 21.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.38 |
| 21.3s | C2S | m15 移动上报 自动 state=8 pos=(855.5,3.4,1101.3) |
| 21.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 21.5s | S2C | 属性更新0x10[O#0x3b015d] CurSkillID=default_normal_skill |
| 21.5s | S2C | 属性更新0x10[O#0x3b015d] CurSkillTarget=None |
| 21.5s | S2C | 位置0x1F(短len=25) |
| 21.5s | S2C | S2C m337 施法回显[0x3b015d0702ccb3] [854.577, 3.556, 1099.437, 0.451, 0] |
| 21.5s | S2C | S2C m20 范围施放标记[0x3b015d0702ccb3] ["atkhit1", obj(0x701e5a4,0x3ad5fb), 0, 0.0] |
| 21.7s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [855.504, 4.686, 1101.35, 0.372] |
| 21.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x701e5a4] |
| 21.7s | S2C | 位置0x1F(短len=25) |
| 21.7s | S2C | S2C m337 施法回显[0x3ad5fb0701e5a4] [855.504, 4.378, 1101.35, 3.568] |
| 21.7s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID=CS_jh_hyjf07 |
| 21.7s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +94行 |
| 21.7s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x701e5a4] ×4 |
| 21.7s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 21.7s | S2C | 记录写格0x13[cooldown_rec#109 行91 列1 =16800512 0x701e5a4] ×2 |
| 21.7s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=97 |
| 21.7s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10178 |
| 21.7s | S2C | 位置0x1F(短len=25) |
| 21.7s | S2C | S2C m337 施法回显[0x3ad5fb0701e5a4] [855.504, 4.378, 1101.35, 0.372, 0] |
| 21.7s | S2C | S2C m20 范围施放标记[0x3ad5fb0701e5a4] ["CS_jh_hyjf07", 0, 0, 0.0, 854.905, 3.506] |
| 21.7s | C2S | m15 移动上报 自动 state=4 pos=(855.5,4.4,1101.3) |
| 21.7s | S2C | 对象同步0x21[0x702ccb3] 码241 len=135 |
| 21.7s | S2C | 属性更新0x10[O#0x3afb5f] State= |
| 21.7s | S2C | 属性更新0x10[O#0x3afb5f] CurSkillID=default_normal_skill |
| 21.7s | S2C | 属性更新0x10[O#0x3afb5f] CurSkillTarget=None |
| 21.7s | S2C | 位置0x1F(短len=25) |
| 21.7s | S2C | S2C m337 施法回显[0x3afb5f07013793] [855.08, 3.552, 1099.282, 0.202, 0] |
| 21.7s | S2C | S2C m20 范围施放标记[0x3afb5f07013793] ["atkhit3", obj(0x701e5a4,0x3ad5fb), 0, 0.0] |
| 21.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 21.7s | C2S | m15 移动上报 自动 state=5 pos=(855.5,4.4,1101.3) ×2 |
| 21.7s | C2S | m15 移动上报 自动 state=7 pos=(855.5,4.4,1101.3) |
| 21.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 21.8s | C2S | 对象请求 len=33 |
| 21.8s | C2S | m431 选择目标 [obj(0x702ccb3,0x3b015d)] |
| 21.8s | C2S | 对象请求 len=33 |
| 21.8s | C2S | m431 选择目标 [obj(0x702ccb3,0x3b015d)] |
| 21.8s | S2C | 属性更新0x10[O#0x3ad5fb] LastObject=None |
| 21.8s | S2C | 属性更新0x10[O#0x3ad5fb] AutoSelectObject=None |
| 21.9s | S2C | op2C len=147 2c47920000000a01000000000080060093370107 |
| 22.1s | S2C | 属性更新0x10[V#0x28] CurTargetCount=2 |
| 22.1s | S2C | 属性更新0x10[O#0x3ad5fb] FP=2 |
| 22.1s | S2C | 属性更新0x10[O#0x3ad5fb] SP=6 |
| 22.1s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 22.1s | S2C | S2C m24 通用事件[0x3b015d0702ccb3] 伤害飘字 [732, 1825, 2, "CS_jh_hyjf07", "SkillNormal", "monm031304"] |
| 22.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo5=4BF9,701E5A4,3AD5FB,5,C38EA5496C7,4 |
| 22.1s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 22.1s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 22.1s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x701e5a4] ×2 |
| 22.1s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 35, 160, "ui_jh_001"] |
| 22.1s | S2C | S2C m24 通用事件[0x3b015d0702ccb3] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031304", "徐傠囃辭睼 "] |
| 22.1s | S2C | S2C m18 战斗伤害帧[0x3b015d0702ccb3] ["hurt_half", 0, obj(0x701e5a4,0x3ad5fb)] |
| 22.1s | S2C | S2C m24 通用事件[0x3b015d0702ccb3] 命中? [] |
| 22.1s | S2C | 属性更新0x10[O#0x3ad5fb] FP=4 |
| 22.1s | S2C | 属性更新0x10[O#0x3ad5fb] SP=8 |
| 22.1s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 22.1s | S2C | S2C m24 通用事件[0x3afb5f07013793] 伤害飘字 [732, 1825, 2, "CS_jh_hyjf07", "SkillNormal", "monm031304"] |
| 22.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo5=4BF9,701E5A4,3AD5FB,5,C38EA5496C8,3 |
| 22.1s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 22.1s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 22.1s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x701e5a4] |
| 22.1s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 42, 160, "ui_jh_001"] |
| 22.1s | S2C | S2C m24 通用事件[0x3afb5f07013793] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031304", "徐傠囃辭睼 "] |
| 22.1s | S2C | S2C m18 战斗伤害帧[0x3afb5f07013793] ["hurt_half", 0, obj(0x701e5a4,0x3ad5fb)] |
| 22.1s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 怒气段 [2] |
| 22.1s | S2C | S2C m24 通用事件[0x3afb5f07013793] 命中? [] |
| 22.1s | S2C | S2C m205 伤害目标列表[0x3ad5fb0701e5a4] ["CS_jh_hyjf07", obj(0x702ccb3,0x3b015d), obj(0x7013793,0x3afb5f)] |
| 22.1s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID= |
| 22.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA5473CA,3 |
| 22.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.09 |
| 22.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.19 |
| 22.3s | S2C | 属性更新0x10[O#0x3ad5fb] PhyDamageMul=1.3 |
| 22.3s | S2C | 属性更新0x10[O#0x3ad5fb] StiDamageMul=1.3 |
| 22.3s | S2C | 属性更新0x10[O#0x3ad5fb] JujDamageMul=1.3 |
| 22.3s | S2C | 属性更新0x10[O#0x3ad5fb] NegDamageMul=1.3 |
| 22.3s | S2C | 属性更新0x10[O#0x3ad5fb] MasDamageMul=1.3 |
| 22.3s | S2C | 属性更新0x10[O#0x3ad5fb] UnderWaterModifySpeed=1.0 ×2 |
| 22.3s | S2C | S2C m1 系统提示 [6, "0", "尊敬的大侠：为保护广大玩家利益，营造绿色公平游戏环境，请勿在公共频道或公开场合上"] |
| 22.3s | S2C | S2C m249 ? [1, 1, "尊敬的大侠：为保护广大玩家利益，营造绿色公平游戏环境，请勿在公共频道或公开场合上"] |
| 22.4s | S2C | 多属性0x2D(非标子码02) len=115 2d0200933701075ffb3a000600d70100 |
| 22.5s | S2C | 属性更新0x10[O#0x3ad5fb] SP=9 |
| 22.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=99 |
| 22.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=28255 |
| 22.5s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 22.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=28264 |
| 22.5s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [21, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 22.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=99 |
| 22.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=28275 |
| 22.5s | S2C | S2C m205 伤害目标列表[0x3b015d0702ccb3] ["default_normal_skill", obj(0x701e5a4,0x3ad5fb)] |
| 22.5s | S2C | 属性更新0x10[O#0x3b015d] CurSkillID= |
| 22.5s | S2C | 属性更新0x10[O#0x3b015d] CurSkillTarget=None |
| 22.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] c1=7 [1, obj(0x7013793,0x3afb5f), "default_normal_skill", "徐傠囃辭睼 ", "monm031304", 100] |
| 22.7s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +1行 |
| 22.7s | S2C | 属性更新0x10[O#0x3afb5f] CurSkillID= |
| 22.7s | S2C | 属性更新0x10[O#0x3afb5f] CurSkillTarget=None |
| 22.9s | C2S | m15 移动上报 自动 state=8 pos=(855.5,3.9,1101.3) |
| 22.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 23.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA5473D8,2 |
| 23.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.89 |
| 23.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.99 |
| 23.3s | C2S | m15 移动上报 自动 state=4 pos=(855.5,3.4,1101.3) |
| 23.4s | S2C | 对象同步0x21[0x701e5a4] 码62 len=47 |
| 23.9s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=98 |
| 23.9s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10278 |
| 24.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA5473FC,1 |
| 24.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.7 |
| 24.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.8 |
| 25.3s | S2C | S2C m169 buff移除 ["buf_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb)] |
| 25.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 25.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 25.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 26.1s | S2C | op0E len=9 0eb3cc02075d013b00 ×2 |
| 26.1s | C2S | m440 ? [] |
| 26.2s | S2C | 属性更新0x10[O#0x3ad5fb] LastObject=None |
| 26.2s | C2S | m431 选择目标 [0] |
| 26.2s | S2C | 属性更新0x10[O#0x3ad5fb] AutoSelectObject=None |
| 27.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=28275 |
| 27.6s | S2C | op06 len=1 06 |
| 27.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 27.7s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] ×2 |
| 27.7s | C2S | m15 移动上报 自动 state=5 pos=(855.5,3.4,1101.3) ×2 |
| 28.6s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=99 |
| 28.6s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10378 |
| 29.2s | C2S | m82 ? [3] |
| 29.5s | C2S | m15 移动上报 ?1000 state=7 pos=(855.5,3.4,1101.3) |
| 29.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 29.7s | S2C | 对象同步0x21[0x701e5a4] 码183 len=47 |
| 29.7s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=2 |
| 29.9s | C2S | m54 格挡状态(值4489905844) |
| 29.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 29.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 29.9s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 29.9s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 29.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 29.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 29.9s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 29.9–30.1s | C2S | m15 移动上报 手动 state=7 pos=(856.9,3.4,1100.8) ×3 |
| 30.1s | S2C | 对象同步0x21[0x701e5a4] 码135 len=47 |
| 30.1s | C2S | m54 格挡状态(值4265865869) ×3 |
| 30.1s | C2S | m15 移动上报 手动 state=7 pos=(857.5,3.4,1100.6) |
| 30.2s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 30.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 30.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 30.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 30.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 30.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 30.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 30.2s | C2S | m54 格挡状态(值4265865869) ×2 |
| 30.3s | C2S | m15 移动上报 手动 state=7 pos=(857.5,3.4,1100.5) |
| 30.3s | C2S | m54 格挡状态(值4265865869) |
| 30.3s | S2C | 对象同步0x21[0x701e5a4] 码134 len=91 |
| 30.4s | C2S | m54 格挡状态(值4489905844) |
| 30.4s | C2S | m15 移动上报 手动 state=7 pos=(857.6,3.5,1099.7) |
| 30.6s | S2C | 对象同步0x21[0x701e5a4] 码255 len=47 |
| 30.6s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 30.6s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 30.6s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 30.6s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 30.6s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 30.6s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 30.6s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 30.6–30.9s | C2S | m15 移动上报 手动 state=7 pos=(857.8,3.6,1098.9) ×3 |
| 30.7s | S2C | 对象同步0x21[0x701e5a4] 码189 len=91 |
| 31.1s | S2C | 对象同步0x21[0x701e5a4] 码78 len=47 |
| 31.2s | C2S | m54 格挡状态(值4265865869) |
| 31.2s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 31.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 31.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 31.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 31.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 31.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 31.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 31.2s | C2S | m54 格挡状态(值4265865869) |
| 31.2s | S2C | 属性更新0x10[O#0x3ad5fb] FP=0 |
| 31.2s | C2S | m54 格挡状态(值4265865869) |
| 31.2s | C2S | m15 移动上报 手动 state=7 pos=(858.4,3.6,1097.9) |
| 31.3s | S2C | 属性更新0x10[O#0x3ad5fb] SP=0 |
| 31.4s | S2C | 对象同步0x21[0x701e5a4] 码66 len=47 |
| 31.4s | C2S | m15 移动上报 手动 state=8 pos=(859.0,3.6,1097.1) |
| 31.5s | S2C | 对象同步0x21[0x701e5a4] 码119 len=47 |
| 31.5s | C2S | m54 格挡状态(值4489905844) |
| 31.6s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 31.6s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 31.6s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 31.6s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 31.6s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 31.6s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 31.6s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 31.6s | C2S | m15 移动上报 手动 state=6 pos=(859.3,4.2,1096.8) ×2 |
| 31.7s | C2S | m54 格挡状态(值4265865869) ×2 |
| 31.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 31.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 31.7s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 31.7s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 31.7s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 31.7s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 31.7s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 31.7s | C2S | m54 格挡状态(值4265865869) ×2 |
| 31.8s | S2C | 对象同步0x21[0x701e5a4] 码160 len=47 |
| 31.8s | C2S | m15 移动上报 手动 state=6 pos=(859.5,4.8,1096.5) |
| 31.8s | C2S | m54 格挡状态(值4265865869) |
| 32.0s | S2C | 对象同步0x21[0x701e5a4] 码187 len=47 |
| 32.2–32.5s | C2S | m15 移动上报 手动 state=6 pos=(860.9,5.0,1095.1) ×2 |
| 32.4s | S2C | 对象同步0x21[0x701e5a4] 码201 len=47 |
| 32.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=100 |
| 32.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=28285 |
| 32.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=100 |
| 32.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=28285 |
| 32.6s | C2S | m15 移动上报 手动 state=4 pos=(862.2,3.6,1093.7) |
| 32.6s | C2S | m15 移动上报 手动 state=7 pos=(862.2,3.6,1093.7) |
| 32.7s | S2C | 对象同步0x21[0x701e5a4] 码236 len=91 |
| 32.7s | S2C | 属性更新0x10[O#0x3afad4] State=stand ×2 |
| 32.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 32.7s | C2S | m54 格挡状态(值4489905844) |
| 32.8s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 32.8s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 32.8s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 32.8s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 32.8s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 32.8s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 32.8s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 32.8s | C2S | m15 移动上报 手动 state=7 pos=(862.8,3.6,1093.0) |
| 32.9s | S2C | op2C len=174 2c47ad00000036010000000000e00700677c0007 |
| 32.9s | S2C | 多属性0x2D[0xfad40700](布局变体) len=23 2d0100677c0007d4fa3a000100540004 |
| 33.0s | C2S | m54 格挡状态(值4265865869) |
| 33.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 33.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 33.0s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 33.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 33.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 33.0s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 33.0s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 33.0s | C2S | m54 格挡状态(值4265865869) ×2 |
| 33.1s | C2S | m15 移动上报 手动 state=7 pos=(863.1,3.7,1092.8) |
| 33.2s | C2S | m15 移动上报 手动 state=8 pos=(863.3,3.8,1092.5) |
| 33.4s | S2C | 对象同步0x21[0x701e5a4] 码163 len=91 |
| 33.4s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=100 |
| 33.4s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10388 |
| 33.4s | S2C | 多属性0x2D[0x1c70701](布局变体) len=207 2d010093370107c7013b001000b50004 |
| 33.4s | C2S | m15 移动上报 手动 state=6 pos=(864.3,5.1,1091.5) |
| 33.5s | S2C | 对象同步0x21[0x701e5a4] 码83 len=47 |
| 33.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 33.8s | C2S | m15 移动上报 手动 state=5 pos=(865.3,5.3,1090.5) ×2 |
| 33.9s | C2S | m15 移动上报 手动 state=6 pos=(865.7,5.0,1090.0) |
| 34.0s | S2C | 对象同步0x21[0x701e5a4] 码172 len=47 |
| 34.0s | C2S | m15 移动上报 手动 state=4 pos=(866.1,4.6,1089.6) |
| 34.1s | S2C | op31(sub1) 0x70012d8 |
| 34.1s | S2C | op28(子码47) len=218 |
| 34.1s | C2S | 对象请求 len=33 |
| 34.1s | C2S | m431 选择目标 [obj(0x7013781,0x3afa82)] |
| 34.1s | C2S | 对象请求 len=33 |
| 34.1s | C2S | m431 选择目标 [obj(0x7013781,0x3afa82)] |
| 34.1s | C2S | m15 移动上报 手动 state=7 pos=(866.1,4.6,1089.6) |
| 34.1s | S2C | 属性更新0x10[O#0x3ad5fb] LastObject=None |
| 34.1s | S2C | 属性更新0x10[O#0x3ad5fb] AutoSelectObject=None |
| 34.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 34.2s | S2C | 对象同步0x21[0x701e5a4] 码14 len=91 |
| 34.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo5=4BF9,701E5A4,3AD5FB,5,C38EA549570,1 |
| 34.7s | S2C | S2C m169 buff移除 ["buf_ng_jz_006_01", obj(0x701e5a4,0x3ad5fb)] |
| 34.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo5= |
| 34.9s | S2C | 属性更新0x10[O#0x3ad5fb] PhyDamageMul=1.0 |
| 34.9s | S2C | 属性更新0x10[O#0x3ad5fb] StiDamageMul=1.0 |
| 34.9s | S2C | 属性更新0x10[O#0x3ad5fb] JujDamageMul=1.0 |
| 34.9s | S2C | 属性更新0x10[O#0x3ad5fb] NegDamageMul=1.0 |
| 34.9s | S2C | 属性更新0x10[O#0x3ad5fb] MasDamageMul=1.0 |
| 34.9s | S2C | 属性更新0x10[O#0x3ad5fb] UnderWaterModifySpeed=1.0 ×2 |
| 35.5s | C2S | m15 移动上报 自动 state=4 pos=(870.5,5.7,1085.1) |
| 35.6s | C2S | m211 使用技能(出招) CS_jh_hyjf05 10值 [870.46, 5.704, 1085.073, 2.378] |
| 35.6s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x701e5a4] ×2 |
| 35.6s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID=CS_jh_hyjf05 |
| 35.6s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +93行 |
| 35.6s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =23808 0x701e5a4] ×4 |
| 35.6s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 35.6s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +94行 |
| 35.6s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x701e5a4] ×4 |
| 35.6s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=99 |
| 35.6s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10289 |
| 35.6s | S2C | 位置0x1F(短len=25) |
| 35.6s | S2C | S2C m337 施法回显[0x3ad5fb0701e5a4] [870.46, 5.703, 1085.073, 0.266, 0] |
| 35.6s | S2C | S2C m20 范围施放标记[0x3ad5fb0701e5a4] ["CS_jh_hyjf05", 0, 0, 0.0] |
| 35.6s | C2S | m15 移动上报 自动 state=4 pos=(870.5,5.7,1085.1) |
| 35.7s | S2C | 对象同步0x21[0x701e5a4] 码118 len=91 |
| 36.0s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 36.0s | S2C | 属性更新0x10[O#0x3ad5fb] LastNGAttackNpc=monm031303 |
| 36.0s | S2C | 属性更新0x10[O#0x3ad5fb] SP=2 |
| 36.0s | S2C | 属性更新0x10[O#0x3afa82] LogicState=1 |
| 36.0s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 36.0s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=1 |
| 36.0s | S2C | 属性更新0x10[O#0x3afa82] CurSkillID=default_normal_skill |
| 36.0s | S2C | 属性更新0x10[O#0x3afa82] CurSkillTarget=None |
| 36.0s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =16801024 0x701e5a4] ×2 |
| 36.0s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 36.0s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =16801280 0x701e5a4] ×2 |
| 36.0s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=98 |
| 36.0s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10190 |
| 36.0s | S2C | S2C m1 系统提示 [4, "fight_kezhi_info_2", "徐傠囃辭睼 ", "CS_jh_hyjf05", "monm031303", "default_normal_skill"] |
| 36.0s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] c1=11 [obj(0x7013781,0x3afa82)] |
| 36.0s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID= |
| 36.0s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] c1=56 [] |
| 36.0s | S2C | 位置0x1F(短len=25) |
| 36.0s | S2C | S2C m337 施法回显[0x3afa8207013781] [870.844, 5.335, 1086.483, 3.407, 0] |
| 36.0s | S2C | S2C m20 范围施放标记[0x3afa8207013781] ["atkhit3", obj(0x701e5a4,0x3ad5fb), 0, 0.0] |
| 36.0s | S2C | S2C m24 通用事件[0x3afa8207013781] 伤害飘字 [678, 806, 2, "CS_jh_hyjf05", "SkillNormal", "monm031303"] |
| 36.0s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 36.0s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 36.0s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x701e5a4] ×2 |
| 36.0s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 49, 160, "ui_jh_001"] |
| 36.0s | S2C | S2C m24 通用事件[0x3afa8207013781] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031303", "徐傠囃辭睼 "] |
| 36.0s | S2C | S2C m707 受理族变体 [13] |
| 36.1s | S2C | 对象同步0x21[0x7013781] 码4 len=223 |
| 36.1s | S2C | op2E len=218 2e47d9000000ae010000000000c0020008de0007 |
| 36.4s | S2C | 多属性0x2D[0xfa820701](布局变体) len=59 2d01008137010782fa3a000600d70100 |
| 37.0s | S2C | 属性更新0x10[O#0x3ad5fb] LastAttackerNpc=monm031303 |
| 37.0s | S2C | 属性更新0x10[O#0x3ad5fb] LastDodge=32.07 |
| 37.0s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] c1=7 [1, obj(0x7013781,0x3afa82), "default_normal_skill", "徐傠囃辭睼 ", "monm031303", 100] |
| 37.0s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 37.0s | S2C | 属性更新0x10[O#0x3afa82] State= |
| 37.0s | S2C | 属性更新0x10[O#0x3afa82] CurSkillID= |
| 37.0s | S2C | 属性更新0x10[O#0x3afa82] CurSkillTarget=None |
| 37.1s | C2S | m15 移动上报 手动 state=7 pos=(870.5,5.7,1085.1) |
| 37.1s | S2C | op28(子码47) len=230 ×2 |
| 37.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 37.3s | S2C | 对象同步0x21[0x701e5a4] 码179 len=47 |
| 37.7s | C2S | m15 移动上报 自动 state=4 pos=(872.6,5.8,1082.8) |
| 37.9s | S2C | 对象同步0x21[0x701e5a4] 码52 len=47 |
| 37.9s | C2S | 对象请求 len=33 |
| 37.9s | S2C | 属性更新0x10[O#0x3ad5fb] LastObject=None |
| 37.9s | C2S | m431 选择目标 [obj(0x7013779,0x3afa94)] |
| 37.9s | C2S | 对象请求 len=33 |
| 37.9s | C2S | m431 选择目标 [obj(0x7013779,0x3afa94)] |
| 38.0s | S2C | 属性更新0x10[O#0x3ad5fb] AutoSelectObject=None |
| 38.1s | C2S | 对象请求 len=33 |
| 38.2s | C2S | m431 选择目标 [obj(0x7013779,0x3afa94)] |
| 39.4s | C2S | m211 使用技能(出招) wuji_CS_jh_hyjf04 10值 [872.613, 5.794, 1082.826, 2.378] |
| 39.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x701e5a4] |
| 39.5s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID=wuji_CS_jh_hyjf04 |
| 39.5s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillTarget=None |
| 39.5s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +94行 |
| 39.5s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x701e5a4] ×4 |
| 39.5s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 39.5s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =16801024 0x701e5a4] ×2 |
| 39.5s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=97 |
| 39.5s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10158 |
| 39.5s | S2C | S2C m20 范围施放标记[0x3ad5fb0701e5a4] ["wuji_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb), 0, 0.0] |
| 39.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 39.8s | C2S | m15 移动上报 自动 state=5 pos=(872.6,5.8,1082.8) ×2 |
| 39.9s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 39.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.7 |
| 39.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.8 |
| 39.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D0A1,1 |
| 39.9s | S2C | S2C m169 buff添加 ["buf_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb)] |
| 39.9s | S2C | S2C m169 buff刷新? ["buf_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb)] |
| 39.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D0A1,A |
| 39.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=6.0 |
| 39.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=2.55 |
| 39.9s | S2C | S2C m205 伤害目标列表[0x3ad5fb0701e5a4] ["wuji_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb)] |
| 39.9s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID= |
| 39.9s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillTarget=None |
| 40.3s | S2C | 记录写格0x13[Count_Limit_Form_Rec#363 行5 列1 =33555712 0x0] |
| 40.3s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 40.3s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 40.3s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=98 |
| 40.3s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10258 |
| 40.3s | S2C | S2C m252 ? [] |
| 40.3s | S2C | S2C m250 ? [] ×3 |
| 40.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D4B1,9 |
| 40.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=2.36 |
| 41.1s | S2C | op28(子码47) len=222 ×3 |
| 41.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D49A,8 |
| 41.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=2.16 |
| 42.3s | C2S | m15 移动上报 手动 state=7 pos=(872.6,5.8,1082.8) |
| 42.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 42.4s | S2C | 对象同步0x21[0x701e5a4] 码247 len=47 |
| 42.6s | C2S | m15 移动上报 手动 state=7 pos=(871.7,5.8,1083.8) |
| 42.6s | S2C | 对象同步0x21[0x701e5a4] 码127 len=47 |
| 42.9s | S2C | op06 len=1 06 |
| 42.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D492,7 |
| 42.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.87 |
| 42.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.97 |
| 43.0s | C2S | m15 移动上报 手动 state=7 pos=(869.9,5.6,1085.7) |
| 43.1s | S2C | 对象同步0x21[0x701e5a4] 码224 len=47 |
| 43.2s | S2C | op31(sub3) 0x700e687 →0x703c260 |
| 43.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D49E,6 |
| 43.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.67 |
| 43.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.77 |
| 44.0s | C2S | m15 移动上报 手动 state=7 pos=(865.9,4.5,1089.8) |
| 44.1s | S2C | 对象同步0x21[0x701e5a4] 码163 len=47 |
| 44.2s | S2C | op31(sub1) 0x700de08 |
| 44.8s | S2C | 属性更新0x10[O#0x3afad4] State= |
| 44.8s | C2S | m15 移动上报 手动 state=7 pos=(862.5,3.6,1093.4) |
| 44.9s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 44.9s | C2S | m15 移动上报 手动 state=7 pos=(862.1,3.6,1093.4) |
| 45.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D486,5 |
| 45.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.48 |
| 45.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.58 |
| 45.0s | C2S | m15 移动上报 ?2000 state=7 pos=(861.8,3.6,1093.0) |
| 45.0s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=99 |
| 45.0s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10358 |
| 45.0s | S2C | op2C len=154 2c47990000000a010000000000e00600677c0007 |
| 45.0s | S2C | 多属性0x2D[0xfad40700](布局变体) len=20 2d0100677c0007d4fa3a000100540001 |
| 45.2s | S2C | 对象同步0x21[0x701e5a4] 码138 len=47 |
| 45.2s | S2C | op31(sub1) 0x701e56e |
| 45.5s | C2S | m15 移动上报 手动 state=7 pos=(859.8,4.0,1091.2) |
| 45.6s | S2C | 对象同步0x21[0x701e5a4] 码27 len=47 |
| 45.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 45.8s | C2S | m15 移动上报 ?4000 state=5 pos=(859.8,5.0,1089.7) ×2 |
| 45.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D4A7,4 |
| 45.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.28 |
| 45.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.38 |
| 46.0s | C2S | m15 移动上报 ?4000 state=7 pos=(859.8,5.8,1088.4) |
| 46.0s | S2C | 对象同步0x21[0x701e5a4] 码91 len=47 |
| 46.2s | S2C | op28(子码47) len=242 |
| 46.5s | C2S | m298 刷新NPC任务特效 [0, obj(0x70012d8,0x12f1)] |
| 46.5s | S2C | S2C m209 ? [15, obj(0x70012d8,0x12f1), 0] |
| 46.9s | S2C | 属性更新0x10[O#0x3afa82] LogicState=0 |
| 46.9s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=2 |
| 47.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D4B0,3 |
| 47.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=5.09 |
| 47.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=1.19 |
| 47.0s | C2S | m15 移动上报 ?4000 state=7 pos=(859.7,9.2,1082.9) |
| 47.1s | S2C | 对象同步0x21[0x701e5a4] 码176 len=47 |
| 47.2s | S2C | op28(子码47) len=222 ×2 |
| 47.3s | C2S | m211 使用技能(出招) CS_jh_tmjt06 10值 [859.631, 9.247, 1081.568, -3.12] |
| 47.3s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x701e5a4] ×2 |
| 47.3s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID=CS_jh_tmjt06 |
| 47.3s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillTarget=None |
| 47.3s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +93行 |
| 47.3s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =23808 0x701e5a4] ×4 |
| 47.3s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 47.3s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +94行 |
| 47.3s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x701e5a4] ×4 |
| 47.3s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10319 |
| 47.3s | S2C | S2C m20 范围施放标记[0x3ad5fb0701e5a4] ["CS_jh_tmjt06", obj(0x701e5a4,0x3ad5fb), 0, 0.0] |
| 47.3s | C2S | m15 移动上报 ?4000 state=4 pos=(859.6,9.2,1081.2) |
| 47.5s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo5=1AAA,701E5A4,3AD5FB,8,0,0 |
| 47.5s | S2C | S2C m169 buff添加 ["buf_CS_jh_tmjt06", obj(0x701e5a4,0x3ad5fb)] |
| 47.5s | S2C | S2C m205 伤害目标列表[0x3ad5fb0701e5a4] ["CS_jh_tmjt06", obj(0x701e5a4,0x3ad5fb)] |
| 47.5s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID= |
| 47.5s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillTarget=None |
| 47.6s | S2C | 对象同步0x21[0x701e5a4] 码233 len=47 |
| 48.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D4A5,2 |
| 48.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.89 |
| 48.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.99 |
| 48.1s | C2S | m54 格挡状态(值4489905844) |
| 48.1s | S2C | 属性更新0x10[O#0x3ad5fb] SP=0 |
| 48.2s | C2S | m15 移动上报 手动 state=7 pos=(859.6,9.2,1081.2) |
| 48.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=2.16 |
| 48.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-1.74 |
| 48.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 48.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 48.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo6=11CE,701E5A4,3AD5FB,1,0,0 |
| 48.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 48.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 48.3s | C2S | m15 移动上报 手动 state=7 pos=(859.5,9.2,1080.8) |
| 48.3s | S2C | op28(子码47) len=230 ×3 |
| 48.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 48.3–48.5s | C2S | m15 移动上报 手动 state=7 pos=(859.4,9.2,1080.5) ×2 |
| 48.4s | S2C | 对象同步0x21[0x701e5a4] 码144 len=91 |
| 48.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=98 |
| 48.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=28001 |
| 48.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 48.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=99 |
| 48.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=28143 |
| 48.6s | C2S | m54 格挡状态(值4265865869) ×2 |
| 48.7–50.1s | S2C | 对象同步0x21[0x701e5a4] 码0 len=47 ×2 |
| 48.7s | C2S | m15 移动上报 手动 state=7 pos=(859.1,9.2,1079.9) |
| 48.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 48.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo6= |
| 48.7s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 48.7s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.89 |
| 48.7s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.99 |
| 48.7s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 48.7s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 48.7s | C2S | m54 格挡状态(值4265865869) ×3 |
| 48.8s | C2S | m15 移动上报 ?4000 state=7 pos=(859.0,9.2,1079.8) |
| 48.9s | S2C | 对象同步0x21[0x701e5a4] 码19 len=91 |
| 49.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=229D,701E5A4,3AD5FB,8,C38EA54D4AA,1 |
| 49.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.7 |
| 49.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.8 |
| 49.1s | C2S | m15 移动上报 ?4000 state=7 pos=(857.9,9.6,1078.5) |
| 49.2s | C2S | m82 ? [3] |
| 49.2s | S2C | op31(sub2) 0x7040899 →0x700de08 |
| 49.3s | S2C | 对象同步0x21[0x701e5a4] 码254 len=47 |
| 49.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=97 |
| 49.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=27717 |
| 49.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 49.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=98 |
| 49.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=28001 |
| 49.7s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=100 |
| 49.7s | S2C | 属性更新0x10[O#0x3ad5fb] MP=10388 |
| 50.1s | S2C | S2C m169 buff移除 ["buf_CS_jh_hyjf04", obj(0x701e5a4,0x3ad5fb)] |
| 50.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 50.1s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 50.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 50.1s | C2S | m15 移动上报 ?4000 state=7 pos=(854.7,12.0,1075.0) |
| 50.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=96 |
| 50.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=27433 |
| 50.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 50.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=27859 |
| 50.6s | C2S | m15 移动上报 手动 state=7 pos=(853.3,12.1,1073.3) |
| 50.8s | S2C | 对象同步0x21[0x701e5a4] 码253 len=47 |
| 51.3s | S2C | op31(sub2) 0x70058ec →0x7007c67 |
| 51.3s | C2S | m54 格挡状态(值4489905844) |
| 51.4s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 51.4s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 51.4s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 51.4s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 51.4s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 51.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 51.4s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 51.4s | C2S | m15 移动上报 自动 state=4 pos=(853.5,12.1,1070.0) |
| 51.5–52.1s | S2C | 对象同步0x21[0x701e5a4] 码55 len=47 ×2 |
| 51.6s | S2C | S2C m169 buff添加 ["buf_CS_jh_tmjt06_range", obj(0x703891a,0x3afac8)] ×2 |
| 51.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=95 |
| 51.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=27149 |
| 51.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 51.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=97 |
| 51.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=27717 |
| 51.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 51.8s | C2S | m15 移动上报 自动 state=5 pos=(853.5,12.1,1070.0) ×2 |
| 51.8s | C2S | m15 移动上报 手动 state=4 pos=(853.5,12.1,1070.0) |
| 51.9s | S2C | 多属性0x2D(非标子码02) len=87 2d02002fc00307c3fa3a00010077021a |
| 52.6s | S2C | 属性更新0x10[O#0x3afac8] LogicState=1 |
| 52.6s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 52.6s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=1 |
| 52.6s | S2C | 属性更新0x10[O#0x3afac3] LogicState=1 |
| 52.6s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +1行 |
| 52.6s | S2C | S2C m24 通用事件[0x3afac80703891a] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] ×2 |
| 52.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=94 |
| 52.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=26865 |
| 52.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 52.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=27575 |
| 52.8s | S2C | 对象同步0x21[0x703891a] 码176 len=179 |
| 52.9s | S2C | 多属性0x2D(非标子码02) len=67 2d02002fc00307c3fa3a000300a001a4 |
| 53.1s | C2S | m15 移动上报 手动 state=7 pos=(853.5,12.1,1070.0) |
| 53.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 53.3s | S2C | op28(子码47) len=230 ×2 |
| 53.4–53.6s | C2S | m15 移动上报 手动 state=7 pos=(853.3,12.1,1069.6) ×2 |
| 53.4s | S2C | 对象同步0x21[0x701e5a4] 码107 len=47 |
| 53.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=95 |
| 53.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=26947 |
| 53.6s | S2C | 对象同步0x21[0x701e5a4] 码84 len=47 |
| 53.6s | S2C | 属性更新0x10[O#0x3afac8] State= |
| 53.6s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 53.6s | S2C | S2C m24 通用事件[0x3afac80703891a] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] |
| 53.6s | S2C | 记录写格0x13[fight_relation_rec#114 行1 列1 =16777472 0x701e5a4] |
| 53.6s | S2C | S2C m24 通用事件[0x3afac30703c02f] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] |
| 53.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=94 |
| 53.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=26663 |
| 53.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 53.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=96 |
| 53.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=27433 |
| 53.8–54.0s | C2S | m15 移动上报 手动 state=7 pos=(853.1,12.1,1069.0) ×2 |
| 53.8s | S2C | 对象同步0x21[0x703891a] 码121 len=223 |
| 54.0s | S2C | 多属性0x2D(非标子码02) len=47 2d02002fc00307c3fa3a000200d7011e |
| 54.2–54.6s | C2S | m15 移动上报 手动 state=7 pos=(852.8,12.1,1068.3) ×3 |
| 54.2s | S2C | 对象同步0x21[0x701e5a4] 码12 len=91 |
| 54.4s | S2C | 对象同步0x21[0x701e5a4] 码162 len=47 |
| 54.6s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 54.6s | S2C | S2C m24 通用事件[0x3afac80703891a] 伤害飘字 [682, 93, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] |
| 54.6s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 54.6s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 54.6s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x703891a,0x3afac8)] |
| 54.6s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x701e5a4] ×2 |
| 54.6s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 56, 160, "ui_jh_001"] |
| 54.6s | S2C | S2C m24 通用事件[0x3afac80703891a] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031304", "徐傠囃辭睼 "] |
| 54.6s | S2C | 属性更新0x10[O#0x3afac3] InParry=1 |
| 54.6s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 54.6s | S2C | S2C m24 通用事件[0x3afac30703c02f] 伤害飘字 [682, 93, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] |
| 54.6s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 54.6s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 54.6s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x703c02f,0x3afac3)] |
| 54.6s | S2C | 属性更新0x10[O#0x3afac3] InParry=0 |
| 54.6s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x701e5a4] |
| 54.6s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 63, 160, "ui_jh_001"] |
| 54.6s | S2C | S2C m24 通用事件[0x3afac30703c02f] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031304", "徐傠囃辭睼 "] |
| 54.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=93 |
| 54.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=26379 |
| 54.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 54.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=27291 |
| 54.8s | C2S | m15 移动上报 手动 state=7 pos=(852.4,12.1,1067.2) |
| 54.9s | S2C | 对象同步0x21[0x701e5a4] 码13 len=91 |
| 54.9s | S2C | 多属性0x2D(非标子码02) len=136 2d02002fc00307c3fa3a000800d70100 |
| 55.0s | C2S | m54 格挡状态(值4265865869) |
| 55.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 55.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 55.0s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 55.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 55.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 55.0s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 55.0s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 55.0s | C2S | m15 移动上报 手动 state=7 pos=(852.3,12.1,1066.9) |
| 55.0s | C2S | m54 格挡状态(值4265865869) ×2 |
| 55.1s | C2S | m15 移动上报 手动 state=7 pos=(852.3,12.1,1066.8) |
| 55.1s | C2S | m54 格挡状态(值4265865869) |
| 55.3s | S2C | 对象同步0x21[0x701e5a4] 码10 len=91 |
| 55.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=92 |
| 55.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=26095 |
| 55.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 55.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=95 |
| 55.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=27149 |
| 56.0s | S2C | 属性更新0x10[O#0x3ad5fb] Faculty=114282750 |
| 56.0s | S2C | 属性更新0x10[V#0x28] CurFillValue=350912 |
| 56.0s | S2C | 属性更新0x10[O#0x3ad5fb] CurFillValue=350912 |
| 56.3s | S2C | op31(sub1) 0x701e56e |
| 56.5s | S2C | op2E len=218 2e47d9000000ae010000000000c00200d3b30307 |
| 56.7s | S2C | 属性更新0x10[O#0x3afac8] LogicState=0 |
| 56.7s | S2C | 属性更新0x10[O#0x3afac3] LogicState=0 |
| 56.7s | C2S | m54 格挡状态(值4489905844) |
| 56.7s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 56.7s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 56.7s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 56.7s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 56.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 56.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 56.7s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 56.7s | C2S | m15 移动上报 手动 state=7 pos=(849.8,12.1,1059.8) |
| 56.8s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=91 |
| 56.8s | S2C | 属性更新0x10[O#0x3ad5fb] HP=25811 |
| 56.8s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 56.8s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=27007 |
| 56.8–57.1s | C2S | m15 移动上报 手动 state=7 pos=(849.8,12.1,1059.7) ×3 |
| 57.0s | S2C | 对象同步0x21[0x701e5a4] 码198 len=91 |
| 57.2s | S2C | 对象同步0x21[0x701e5a4] 码29 len=47 |
| 57.2s | C2S | m54 格挡状态(值4265865869) ×3 |
| 57.2s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 57.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 57.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 57.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 57.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 57.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 57.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 57.2s | C2S | m54 格挡状态(值4265865869) ×2 |
| 57.3s | C2S | m15 移动上报 手动 state=7 pos=(849.1,12.1,1059.0) |
| 57.3s | S2C | op28(子码47) len=230 ×2 |
| 57.4s | S2C | 对象同步0x21[0x701e5a4] 码220 len=47 |
| 57.7s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=2 |
| 57.7s | C2S | m15 移动上报 手动 state=8 pos=(847.6,12.1,1057.7) |
| 57.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=90 |
| 57.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=25527 |
| 57.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 57.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=94 |
| 57.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=26865 |
| 57.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 57.8s | C2S | m15 移动上报 手动 state=5 pos=(847.4,12.4,1057.5) ×2 |
| 57.8s | S2C | 对象同步0x21[0x701e5a4] 码60 len=47 |
| 58.1s | C2S | m15 移动上报 手动 state=6 pos=(846.5,13.4,1056.6) |
| 58.1s | S2C | op06 len=1 06 |
| 58.2s | S2C | 对象同步0x21[0x701e5a4] 码253 len=47 |
| 58.5s | C2S | m15 移动上报 手动 state=6 pos=(845.0,13.4,1055.3) |
| 58.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=25609 |
| 58.7s | S2C | 对象同步0x21[0x701e5a4] 码145 len=47 |
| 58.8s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=89 |
| 58.8s | S2C | 属性更新0x10[O#0x3ad5fb] HP=25325 |
| 58.8s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 58.8s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=26723 |
| 58.8s | C2S | m15 移动上报 手动 state=6 pos=(843.9,12.1,1054.3) |
| 58.8s | C2S | m15 移动上报 手动 state=4 pos=(843.9,12.1,1054.3) |
| 58.9s | C2S | m15 移动上报 手动 state=7 pos=(843.9,12.1,1054.3) |
| 58.9s | S2C | 对象同步0x21[0x701e5a4] 码121 len=91 |
| 59.0s | C2S | m15 移动上报 手动 state=8 pos=(843.6,12.1,1054.1) |
| 59.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 59.1s | S2C | 对象同步0x21[0x701e5a4] 码75 len=91 |
| 59.3s | C2S | m15 移动上报 手动 state=6 pos=(842.4,13.4,1053.0) |
| 59.4s | S2C | op28(子码47) len=226 ×2 |
| 59.4s | S2C | 对象同步0x21[0x701e5a4] 码149 len=47 |
| 59.4s | S2C | op2E len=284 2e471b010000a2010000000000c0020024580207 |
| 59.5s | S2C | op0E len=9 0e2fc00307c3fa3a00 ×2 |
| 59.7s | C2S | m15 移动上报 手动 state=6 pos=(841.0,13.4,1051.7) |
| 59.8s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=88 |
| 59.8s | S2C | 属性更新0x10[O#0x3ad5fb] HP=25041 |
| 59.8s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 59.8s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=93 |
| 59.8s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=26581 |
| 59.9s | S2C | 对象同步0x21[0x701e5a4] 码191 len=47 |
| 60.1s | C2S | m15 移动上报 手动 state=6 pos=(839.9,12.1,1050.7) |
| 60.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 60.1s | C2S | m15 移动上报 手动 state=4 pos=(839.8,12.1,1050.7) |
| 60.1s | C2S | m15 移动上报 手动 state=8 pos=(839.8,12.1,1050.7) |
| 60.1s | S2C | S2C m586 状态推送(簇首) [5] |
| 60.3s | S2C | 对象同步0x21[0x701e5a4] 码93 len=91 |
| 60.4s | S2C | op28(子码47) len=226 ×2 |
| 60.4s | C2S | m15 移动上报 手动 state=6 pos=(838.7,13.4,1049.6) |
| 60.5s | S2C | 对象同步0x21[0x701e5a4] 码61 len=47 |
| 60.8s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=87 |
| 60.8s | S2C | 属性更新0x10[O#0x3ad5fb] HP=24757 |
| 60.8s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 60.8s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=26439 |
| 60.8s | C2S | m54 格挡状态(值4489905844) |
| 60.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 60.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 60.9s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 60.9s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 60.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 60.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 60.9s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 60.9s | C2S | m15 移动上报 手动 state=6 pos=(837.3,13.4,1048.4) ×2 |
| 61.1s | C2S | m54 格挡状态(值4265865869) |
| 61.1s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 61.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 61.1s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 61.1s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 61.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 61.1s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 61.1s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 61.1s | C2S | m54 格挡状态(值4265865869) ×2 |
| 61.1s | S2C | 对象同步0x21[0x701e5a4] 码2 len=47 |
| 61.1s | C2S | m15 移动上报 手动 state=6 pos=(836.9,12.6,1047.9) |
| 61.2s | C2S | m15 移动上报 手动 state=4 pos=(836.6,12.1,1047.6) |
| 61.3s | C2S | m15 移动上报 手动 state=8 pos=(836.6,12.1,1047.6) |
| 61.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 61.4s | S2C | 对象同步0x21[0x701e5a4] 码71 len=135 |
| 61.4s | S2C | op31(sub3) 0x7013781 →0x703b3d3 |
| 61.6s | C2S | m15 移动上报 手动 state=6 pos=(835.6,13.4,1046.4) |
| 61.8s | S2C | 对象同步0x21[0x701e5a4] 码69 len=47 |
| 61.9s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=86 |
| 61.9s | S2C | 属性更新0x10[O#0x3ad5fb] HP=24473 |
| 61.9s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 61.9s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=92 |
| 61.9s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=26297 |
| 62.0s | C2S | m15 移动上报 手动 state=6 pos=(834.4,13.4,1044.9) |
| 62.2s | C2S | m54 格挡状态(值4489905844) |
| 62.2s | S2C | 对象同步0x21[0x701e5a4] 码227 len=47 |
| 62.3s | C2S | m15 移动上报 手动 state=6 pos=(833.6,12.4,1043.9) |
| 62.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 62.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 62.3s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 62.3s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 62.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 62.3s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 62.3s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 62.4s | C2S | m15 移动上报 手动 state=6 pos=(833.5,12.1,1043.7) |
| 62.4s | C2S | m15 移动上报 手动 state=4 pos=(833.5,12.1,1043.7) |
| 62.4s | C2S | m15 移动上报 手动 state=7 pos=(833.5,12.1,1043.7) |
| 62.4s | S2C | op31(sub1) 0x7013779 |
| 62.4s | S2C | op28(子码47) len=223 |
| 62.5s | C2S | m54 格挡状态(值4265865869) |
| 62.5s | C2S | m440 ? [] |
| 62.5s | C2S | m431 选择目标 [0] |
| 62.5s | C2S | m54 格挡状态(值4265865869) ×2 |
| 62.5s | S2C | 对象同步0x21[0x701e5a4] 码34 len=135 |
| 62.5s | S2C | 属性更新0x10[O#0x3ad5fb] LastObject=None |
| 62.5s | S2C | 属性更新0x10[O#0x3ad5fb] AutoSelectObject=None |
| 62.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 62.5s | C2S | m15 移动上报 手动 state=8 pos=(833.4,12.1,1043.5) |
| 62.5s | C2S | m54 格挡状态(值4265865869) ×2 |
| 62.5s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 62.5s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 62.5s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 62.5s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 62.5s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 62.5s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 62.5s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 62.5s | C2S | m54 格挡状态(值4265865869) |
| 62.6s | C2S | m15 移动上报 手动 state=6 pos=(833.3,12.5,1043.4) |
| 62.6s | S2C | 对象同步0x21[0x701e5a4] 码6 len=47 |
| 62.6s | S2C | 多属性0x2D[0x3f30700](布局变体) len=220 2d010072910007f3033b001000b50004 |
| 62.8s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=85 |
| 62.8s | S2C | 属性更新0x10[O#0x3ad5fb] HP=24189 |
| 62.8s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 62.8s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=26155 |
| 62.8s | C2S | m54 格挡状态(值4489905844) |
| 62.9s | S2C | 对象同步0x21[0x701e5a4] 码230 len=47 |
| 62.9s | C2S | m15 移动上报 手动 state=6 pos=(832.6,13.5,1042.3) |
| 62.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 62.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 62.9s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 62.9s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 62.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 62.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 62.9s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 62.9s | C2S | m15 移动上报 手动 state=6 pos=(832.4,13.6,1042.0) |
| 63.0s | C2S | m54 格挡状态(值4265865869) ×2 |
| 63.0s | S2C | 对象同步0x21[0x701e5a4] 码114 len=47 |
| 63.1s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 63.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 63.1s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 63.1s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 63.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 63.1s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 63.1s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 63.1s | C2S | m54 格挡状态(值4265865869) ×2 |
| 63.2s | C2S | m15 移动上报 手动 state=6 pos=(832.3,13.6,1041.7) |
| 63.3s | S2C | 对象同步0x21[0x701e5a4] 码189 len=47 |
| 63.4s | S2C | op31(sub1) 0x7013793 |
| 63.4s | S2C | op28(子码47) len=233 |
| 63.5s | C2S | m15 移动上报 手动 state=6 pos=(831.6,12.6,1040.1) |
| 63.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=24271 |
| 63.6s | C2S | m15 移动上报 手动 state=4 pos=(831.4,12.1,1039.6) |
| 63.7s | C2S | m15 移动上报 手动 state=7 pos=(831.4,12.1,1039.6) |
| 63.7s | S2C | 对象同步0x21[0x701e5a4] 码129 len=91 |
| 63.7s | S2C | S2C m169 buff添加 ["buf_CS_jh_tmjt06_range", obj(0x703b3de,0x3b03c4)] ×2 |
| 63.8s | C2S | m15 移动上报 手动 state=8 pos=(831.3,12.1,1039.2) |
| 63.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 63.8s | C2S | m15 移动上报 手动 state=5 pos=(831.2,12.4,1039.0) ×2 |
| 63.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 63.9s | C2S | m54 格挡状态(值4489905844) |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=84 |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] HP=23987 |
| 63.9s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=91 |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=26013 |
| 63.9s | S2C | 对象同步0x21[0x701e5a4] 码25 len=91 |
| 63.9s | S2C | 多属性0x2D(非标子码02) len=87 2d020062410307bf033b00010077021a |
| 63.9s | C2S | m15 移动上报 手动 state=6 pos=(831.0,12.9,1038.5) |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 63.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 63.9s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 64.0s | C2S | m15 移动上报 手动 state=6 pos=(830.9,13.2,1038.2) |
| 64.0s | C2S | m54 格挡状态(值4265865869) |
| 64.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 64.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 64.0s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 64.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 64.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 64.0s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 64.0s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 64.0s | C2S | m54 格挡状态(值4265865869) ×2 |
| 64.1s | C2S | m15 移动上报 手动 state=6 pos=(830.8,13.4,1038.1) |
| 64.1s | C2S | m54 格挡状态(值4265865869) |
| 64.1s | S2C | 对象同步0x21[0x701e5a4] 码86 len=47 |
| 64.3s | S2C | 对象同步0x21[0x701e5a4] 码62 len=47 |
| 64.4s | S2C | op28(子码47) len=230 ×2 |
| 64.5s | C2S | m15 移动上报 手动 state=6 pos=(830.3,13.4,1036.2) |
| 64.8s | S2C | 属性更新0x10[O#0x3b03c4] LogicState=1 |
| 64.8s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 64.8s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=1 |
| 64.8s | S2C | 属性更新0x10[O#0x3b03bf] LogicState=1 |
| 64.8s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +1行 |
| 64.8s | S2C | 属性更新0x10[O#0x3b03c4] CurSkillID=default_normal_skill |
| 64.8s | S2C | 属性更新0x10[O#0x3b03c4] CurSkillTarget=None |
| 64.8s | S2C | 位置0x1F(短len=25) |
| 64.8s | S2C | S2C m337 施法回显[0x3b03c40703b3de] [828.654, 12.101, 1035.849, 1.381, 0] |
| 64.8s | S2C | S2C m20 范围施放标记[0x3b03c40703b3de] ["atkhit1", obj(0x701e5a4,0x3ad5fb), 0, 0.0] |
| 64.8s | S2C | S2C m24 通用事件[0x3b03c40703b3de] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] ×2 |
| 64.8s | S2C | op2C len=180 2c47b300000036010000000000800700a4e50107 |
| 64.8s | C2S | m15 移动上报 手动 state=6 pos=(829.9,12.1,1034.6) |
| 64.9s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=83 |
| 64.9s | S2C | 属性更新0x10[O#0x3ad5fb] HP=23703 |
| 64.9s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 64.9s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25871 |
| 64.9s | C2S | m15 移动上报 手动 state=4 pos=(829.9,12.1,1034.6) |
| 64.9s | C2S | m15 移动上报 手动 state=8 pos=(829.9,12.1,1034.6) |
| 64.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 65.0s | S2C | 对象同步0x21[0x701e5a4] 码30 len=135 |
| 65.0s | S2C | 多属性0x2D(非标子码02) len=67 2d020062410307bf033b000300a001a4 |
| 65.2s | S2C | S2C m758 玩家名片 [6, "云依君 ", "79DD834856A8402580EC32B8154634EC", 6, 2, " "] |
| 65.2s | C2S | m15 移动上报 手动 state=6 pos=(829.5,13.4,1033.2) |
| 65.4–66.8s | S2C | 对象同步0x21[0x701e5a4] 码251 len=47 ×2 |
| 65.5s | S2C | op31(sub3) 0x70012d8 →0x703c25f |
| 65.5s | S2C | op28(子码47) len=230 |
| 65.6s | C2S | m15 移动上报 手动 state=6 pos=(829.3,13.4,1032.6) |
| 65.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 65.7s | C2S | m15 移动上报 手动 state=4 pos=(829.3,13.1,1032.6) |
| 65.8s | S2C | 对象同步0x21[0x701e5a4] 码144 len=47 |
| 65.8s | S2C | 属性更新0x10[O#0x3ad5fb] LastAttackerNpc=monm031304 |
| 65.8s | S2C | 属性更新0x10[O#0x3ad5fb] LastDodge=32.29 |
| 65.8s | S2C | 属性更新0x10[O#0x3ad5fb] SP=1 |
| 65.8s | S2C | 属性更新0x10[O#0x3ad5fb] HP=23671 |
| 65.8s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 65.8s | S2C | 属性更新0x10[O#0x3ad5fb] LastObject=None |
| 65.8s | S2C | 属性更新0x10[O#0x3ad5fb] AutoSelectObject=None |
| 65.8s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [32, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 65.8s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25855 |
| 65.8s | S2C | S2C m205 伤害目标列表[0x3b03c40703b3de] ["default_normal_skill", obj(0x701e5a4,0x3ad5fb)] |
| 65.8s | S2C | 属性更新0x10[O#0x3b03c4] State= |
| 65.8s | S2C | 属性更新0x10[O#0x3b03c4] CurSkillID= |
| 65.8s | S2C | 属性更新0x10[O#0x3b03c4] CurSkillTarget=None |
| 65.8s | S2C | S2C m189 对象通知 [obj(0x703b3de,0x3b03c4)] |
| 65.8s | S2C | S2C m24 通用事件[0x3b03c40703b3de] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] |
| 65.8s | S2C | 记录写格0x13[fight_relation_rec#114 行1 列1 =16777472 0x701e5a4] |
| 65.8s | S2C | S2C m24 通用事件[0x3b03bf07034162] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] |
| 65.8s | C2S | m15 移动上报 手动 state=7 pos=(829.3,13.1,1032.6) |
| 65.8s | S2C | S2C m586 状态推送(簇首) [5] |
| 65.9s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=82 |
| 65.9s | S2C | 属性更新0x10[O#0x3ad5fb] HP=23387 |
| 65.9s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 65.9s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=90 |
| 65.9s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25713 |
| 66.0s | S2C | 对象同步0x21[0x701e5a4] 码64 len=47 |
| 66.0s | S2C | 多属性0x2D(非标子码02) len=47 2d020062410307bf033b000200d7011e |
| 66.2s | C2S | m54 格挡状态(值4489905844) |
| 66.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 66.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 66.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 66.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 66.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 66.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 66.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 66.2s | C2S | m15 移动上报 手动 state=7 pos=(828.7,15.1,1030.5) |
| 66.2s | C2S | m15 移动上报 手动 state=8 pos=(828.7,15.0,1030.4) |
| 66.2s | C2S | m15 移动上报 手动 state=4 pos=(828.7,15.1,1030.4) |
| 66.3s | C2S | m15 移动上报 手动 state=7 pos=(828.7,15.1,1030.4) |
| 66.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 66.4s | C2S | m15 移动上报 手动 state=7 pos=(828.7,15.1,1030.1) |
| 66.5s | S2C | 对象同步0x21[0x701e5a4] 码235 len=135 |
| 66.5s | S2C | op28(子码47) len=226 |
| 66.6s | C2S | m15 移动上报 手动 state=7 pos=(828.6,15.5,1029.7) |
| 66.7s | S2C | 对象同步0x21[0x701e5a4] 码40 len=47 |
| 66.7s | C2S | m54 格挡状态(值4265865869) |
| 66.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 66.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 66.7s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 66.7s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 66.7s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 66.7s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 66.7s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 66.7s | C2S | m54 格挡状态(值4265865869) ×2 |
| 66.8s | C2S | m15 移动上报 手动 state=7 pos=(828.6,15.5,1029.5) |
| 66.8s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 66.8s | S2C | S2C m24 通用事件[0x3b03c40703b3de] 伤害飘字 [682, 93, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] |
| 66.8s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 66.8s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 66.8s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x703b3de,0x3b03c4)] |
| 66.8s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x701e5a4] ×2 |
| 66.8s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 70, 160, "ui_jh_001"] |
| 66.8s | S2C | S2C m24 通用事件[0x3b03c40703b3de] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031304", "徐傠囃辭睼 "] |
| 66.8s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 66.8s | S2C | S2C m24 通用事件[0x3b03bf07034162] 伤害飘字 [682, 93, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031304"] |
| 66.8s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 66.8s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 66.8s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x7034162,0x3b03bf)] |
| 66.8s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x701e5a4] |
| 66.8s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 77, 160, "ui_jh_001"] |
| 66.8s | S2C | S2C m24 通用事件[0x3b03bf07034162] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031304", "徐傠囃辭睼 "] |
| 66.9s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=81 |
| 66.9s | S2C | 属性更新0x10[O#0x3ad5fb] HP=23103 |
| 66.9s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 66.9s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25571 |
| 67.1s | S2C | 对象同步0x21[0x701e5a4] 码206 len=47 |
| 67.1s | S2C | 多属性0x2D(非标子码02) len=129 2d020062410307bf033b000700d70100 |
| 67.6s | C2S | m15 移动上报 手动 state=8 pos=(828.3,18.6,1025.8) |
| 67.7s | S2C | 对象同步0x21[0x701e5a4] 码110 len=47 |
| 67.9s | C2S | m15 移动上报 手动 state=6 pos=(828.2,20.0,1024.7) |
| 68.0s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=80 |
| 68.0s | S2C | 属性更新0x10[O#0x3ad5fb] HP=22819 |
| 68.0s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 68.0s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=89 |
| 68.0s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25429 |
| 68.0s | C2S | m54 格挡状态(值4489905844) |
| 68.1s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 68.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 68.1s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 68.1s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 68.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 68.1s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 68.1s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 68.1s | C2S | m15 移动上报 手动 state=6 pos=(828.1,20.1,1024.2) ×2 |
| 68.1s | S2C | 对象同步0x21[0x701e5a4] 码253 len=47 |
| 68.2s | C2S | m54 格挡状态(值4265865869) |
| 68.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 68.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 68.3s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 68.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 68.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 68.3s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 68.3s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 68.3s | C2S | m15 移动上报 手动 state=6 pos=(828.2,20.1,1023.7) |
| 68.3s | C2S | m54 格挡状态(值4265865869) ×2 |
| 68.3s | S2C | 对象同步0x21[0x701e5a4] 码227 len=91 |
| 68.3s | C2S | m15 移动上报 手动 state=6 pos=(828.2,20.1,1023.6) |
| 68.5s | C2S | m15 移动上报 手动 state=4 pos=(828.3,19.6,1022.8) |
| 68.5s | S2C | op28(子码47) len=234 ×2 |
| 68.5s | S2C | 对象同步0x21[0x701e5a4] 码17 len=91 |
| 68.5s | C2S | m15 移动上报 手动 state=7 pos=(828.3,19.6,1022.8) |
| 68.5s | C2S | m54 格挡状态(值4489905844) |
| 68.6s | S2C | 属性更新0x10[O#0x3ad5fb] LastObject=None |
| 68.6s | S2C | 属性更新0x10[O#0x3ad5fb] AutoSelectObject=None |
| 68.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 68.6s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 68.6s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 68.6s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 68.6s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 68.6s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 68.6s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 68.6s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 68.6s | C2S | m15 移动上报 手动 state=7 pos=(828.4,19.6,1022.5) |
| 68.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=22901 |
| 68.6s | C2S | m15 移动上报 手动 state=8 pos=(828.4,19.6,1022.4) |
| 68.7s | C2S | m54 格挡状态(值4265865869) ×6 |
| 68.7s | S2C | 对象同步0x21[0x701e5a4] 码127 len=135 |
| 68.8s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 68.8s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 68.8s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 68.8s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 68.8s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 68.8s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 68.8s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 68.9s | S2C | 属性更新0x10[O#0x3b03c4] LogicState=0 |
| 68.9s | S2C | 属性更新0x10[O#0x3b03bf] LogicState=0 |
| 68.9s | C2S | m54 格挡状态(值4265865869) |
| 68.9s | C2S | m15 移动上报 手动 state=6 pos=(828.5,20.5,1022.0) |
| 69.0s | S2C | 对象同步0x21[0x701e5a4] 码63 len=47 |
| 69.0s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=79 |
| 69.0s | S2C | 属性更新0x10[O#0x3ad5fb] HP=22617 |
| 69.0s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 69.0s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25287 |
| 69.2s | C2S | m82 ? [3] |
| 69.3s | C2S | m54 格挡状态(值4489905844) |
| 69.3s | C2S | m15 移动上报 手动 state=6 pos=(829.2,21.2,1020.3) |
| 69.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 69.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 69.3s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 69.3s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 69.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 69.3s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 69.3s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 69.3s | C2S | m15 移动上报 手动 state=6 pos=(829.2,21.1,1020.2) ×2 |
| 69.5s | C2S | m54 格挡状态(值4265865869) |
| 69.5s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 69.5s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 69.5s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 69.5s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 69.5s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 69.5s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 69.5s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 69.5s | C2S | m54 格挡状态(值4265865869) ×2 |
| 69.6s | C2S | m15 移动上报 手动 state=6 pos=(829.5,20.6,1019.6) |
| 69.6s | C2S | m54 格挡状态(值4265865869) |
| 69.6s | S2C | 对象同步0x21[0x701e5a4] 码124 len=91 |
| 69.8s | C2S | m15 移动上报 手动 state=4 pos=(829.9,19.6,1018.7) |
| 69.8s | S2C | 对象同步0x21[0x701e5a4] 码199 len=47 |
| 69.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 69.8s | C2S | m15 移动上报 手动 state=5 pos=(829.9,19.6,1018.7) ×2 |
| 69.8s | C2S | m15 移动上报 手动 state=8 pos=(829.9,19.6,1018.7) |
| 69.9s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=2 |
| 69.9s | C2S | m54 格挡状态(值4489905844) |
| 70.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 70.0s | C2S | m15 移动上报 手动 state=6 pos=(830.2,20.3,1018.1) |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 70.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=78 |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] HP=22333 |
| 70.0s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=88 |
| 70.0s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25145 |
| 70.0–70.4s | C2S | m15 移动上报 手动 state=6 pos=(830.4,20.6,1017.8) ×3 |
| 70.1s | S2C | 对象同步0x21[0x701e5a4] 码243 len=47 |
| 70.2s | S2C | 对象同步0x21[0x701e5a4] 码126 len=91 |
| 70.4s | S2C | 对象同步0x21[0x701e5a4] 码112 len=47 |
| 70.5s | C2S | m54 格挡状态(值4265865869) |
| 70.5s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 70.5s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 70.5s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 70.5s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 70.5s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 70.5s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 70.5s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 70.5s | C2S | m54 格挡状态(值4265865869) ×2 |
| 70.5s | S2C | op28(子码47) len=239 ×2 |
| 70.5s | C2S | m15 移动上报 手动 state=6 pos=(831.0,21.0,1017.1) |
| 70.5s | C2S | m54 格挡状态(值4265865869) |
| 70.7s | S2C | 对象同步0x21[0x701e5a4] 码4 len=47 |
| 70.9s | C2S | m15 移动上报 手动 state=6 pos=(832.2,19.8,1016.1) |
| 71.0s | C2S | m15 移动上报 手动 state=4 pos=(832.4,19.6,1016.0) |
| 71.0s | C2S | m15 移动上报 手动 state=8 pos=(832.4,19.6,1016.0) |
| 71.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 71.0s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=77 |
| 71.0s | S2C | 属性更新0x10[O#0x3ad5fb] HP=22049 |
| 71.0s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 71.0s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25003 |
| 71.1s | S2C | 对象同步0x21[0x701e5a4] 码159 len=135 |
| 71.2s | S2C | op0E len=9 0e62410307bf033b00 ×2 |
| 71.3s | C2S | m15 移动上报 手动 state=6 pos=(833.6,20.9,1015.0) |
| 71.5s | S2C | 对象同步0x21[0x701e5a4] 码228 len=47 |
| 71.5s | S2C | op31(sub3) 0x7011e70 →0x700ad14 |
| 71.7s | C2S | m15 移动上报 手动 state=6 pos=(835.1,20.9,1013.8) |
| 71.8s | S2C | S2C m169 buff添加 ["buf_CS_jh_tmjt06_range", obj(0x7034160,0x3b03df)] ×2 |
| 71.9s | S2C | 属性更新0x10[O#0x3ad5fb] SP=0 |
| 72.0s | S2C | 对象同步0x21[0x701e5a4] 码110 len=47 |
| 72.0s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=76 |
| 72.0s | S2C | 属性更新0x10[O#0x3ad5fb] HP=21765 |
| 72.0s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 72.0s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=87 |
| 72.0s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=24861 |
| 72.1s | C2S | m15 移动上报 手动 state=6 pos=(836.3,19.7,1012.9) |
| 72.1s | C2S | m15 移动上报 手动 state=4 pos=(836.3,19.6,1012.8) |
| 72.2s | C2S | m15 移动上报 手动 state=7 pos=(836.3,19.6,1012.8) |
| 72.2s | S2C | 对象同步0x21[0x701e5a4] 码50 len=91 |
| 72.2s | S2C | 多属性0x2D(非标子码02) len=87 2d020083fd0107da033b00010077021a |
| 72.2s | C2S | m15 移动上报 手动 state=8 pos=(836.6,19.6,1012.6) |
| 72.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 72.4s | S2C | 对象同步0x21[0x701e5a4] 码0 len=91 |
| 72.5s | C2S | m15 移动上报 手动 state=6 pos=(837.8,20.9,1011.6) |
| 72.6s | S2C | 对象同步0x21[0x701e5a4] 码46 len=47 |
| 72.8s | C2S | m15 移动上报 手动 state=8 pos=(838.7,21.2,1010.9) |
| 72.8s | S2C | 属性更新0x10[O#0x3b03df] LogicState=1 |
| 72.8s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 72.8s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=1 |
| 72.8s | S2C | 属性更新0x10[O#0x3b03da] LogicState=1 |
| 72.8s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +1行 |
| 72.8s | S2C | S2C m24 通用事件[0x3b03df07034160] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031301"] |
| 72.8s | S2C | 属性更新0x10[O#0x3b03da] InParry=1 |
| 72.8s | S2C | S2C m24 通用事件[0x3b03da0701fd83] 伤害飘字 [678, 97, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031300"] |
| 72.8s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 72.8s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 72.8s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x701fd83,0x3b03da)] |
| 72.8s | S2C | 属性更新0x10[O#0x3b03da] InParry=0 |
| 72.8s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x701e5a4] ×2 |
| 72.8s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 82, 160, "ui_jh_001"] |
| 72.8s | S2C | S2C m24 通用事件[0x3b03da0701fd83] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031300", "徐傠囃辭睼 "] |
| 72.9s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=158 |
| 72.9s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=1 |
| 73.0s | C2S | m15 移动上报 手动 state=6 pos=(839.3,23.1,1010.3) |
| 73.0–74.0s | S2C | 对象同步0x21[0x7034160] 码100 len=179 ×2 |
| 73.1s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=75 |
| 73.1s | S2C | 属性更新0x10[O#0x3ad5fb] HP=21481 |
| 73.1s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 73.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=24719 |
| 73.2s | C2S | m15 移动上报 手动 state=6 pos=(840.2,24.9,1009.7) |
| 73.2s | S2C | 对象同步0x21[0x701e5a4] 码18 len=47 |
| 73.2s | S2C | 多属性0x2D(非标子码02) len=105 2d020083fd0107da033b000800a00100 |
| 73.3s | S2C | op06 len=1 06 |
| 73.4s | C2S | m15 移动上报 手动 state=8 pos=(840.9,26.0,1009.1) |
| 73.4s | S2C | 对象同步0x21[0x701e5a4] 码90 len=47 |
| 73.5s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=152 |
| 73.6s | C2S | m15 移动上报 手动 state=6 pos=(841.7,27.9,1008.4) |
| 73.7s | S2C | 对象同步0x21[0x701e5a4] 码42 len=47 |
| 73.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=76 |
| 73.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=21563 |
| 73.7s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=162 |
| 73.8s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x7034160,0x3b03df)] |
| 73.9–76.0s | S2C | 对象同步0x21[0x701e5a4] 码114 len=47 ×2 |
| 73.9s | S2C | 属性更新0x10[O#0x3b03df] State= |
| 73.9s | S2C | S2C m189 对象通知 [obj(0x7034160,0x3b03df)] |
| 73.9s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 73.9s | S2C | S2C m24 通用事件[0x3b03df07034160] HP治疗飘字 [775.0, 1457, 0, "buff_clone_recovery_5", "Buffer", "monm031301"] |
| 73.9s | S2C | 属性更新0x10[O#0x3b03df] State=ty_talk3 |
| 73.9s | C2S | m15 移动上报 手动 state=6 pos=(842.7,29.4,1007.6) |
| 74.0s | S2C | 多属性0x2D[0x3df0703](布局变体) len=42 2d010060410307df033b000400770201 |
| 74.1s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=75 |
| 74.1s | S2C | 属性更新0x10[O#0x3ad5fb] HP=21279 |
| 74.1s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 74.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=86 |
| 74.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=24577 |
| 74.3s | C2S | m15 移动上报 手动 state=6 pos=(844.2,30.2,1006.4) |
| 74.5s | S2C | 对象同步0x21[0x701e5a4] 码177 len=47 |
| 74.6s | S2C | op28(子码47) len=219 |
| 74.6s | S2C | 属性更新0x10[V#0x2] LeftTime=603318 |
| 74.7s | C2S | m15 移动上报 手动 state=6 pos=(845.6,29.3,1005.2) |
| 74.8s | S2C | 属性更新0x10[O#0x3b03da] LogicState=0 |
| 74.9s | S2C | 对象同步0x21[0x701e5a4] 码81 len=47 |
| 75.0s | C2S | m15 移动上报 手动 state=6 pos=(846.6,27.7,1004.4) |
| 75.1s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=74 |
| 75.1s | S2C | 属性更新0x10[O#0x3ad5fb] HP=20995 |
| 75.1s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 75.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=24435 |
| 75.3s | C2S | m15 移动上报 手动 state=6 pos=(847.3,25.9,1003.8) |
| 75.3s | S2C | 对象同步0x21[0x701e5a4] 码205 len=91 |
| 75.3s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=0 |
| 75.3s | S2C | 属性更新0x10[O#0x14b] State=stand ×2 |
| 75.3s | C2S | m15 移动上报 手动 state=7 pos=(847.4,25.4,1003.7) |
| 75.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 75.5s | S2C | 对象同步0x21[0x7000146] 码65 len=223 |
| 75.5s | S2C | 多属性0x2D[0x14b0700](布局变体) len=23 2d0100460100074b0100000100540004 |
| 75.6s | S2C | op28(子码47) len=225 |
| 75.8s | C2S | m54 格挡状态(值4489905844) |
| 75.8s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 75.8s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 75.8s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 75.8s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 75.8s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 75.8s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 75.8s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 75.8s | C2S | m15 移动上报 手动 state=7 pos=(849.0,25.9,1002.5) |
| 75.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 75.8s | C2S | m15 移动上报 手动 state=5 pos=(849.1,26.3,1002.4) ×2 |
| 75.9s | S2C | 属性更新0x10[O#0x3b03df] LogicState=0 |
| 75.9s | C2S | m15 移动上报 手动 state=7 pos=(849.2,26.2,1002.3) |
| 76.0s | C2S | m54 格挡状态(值4265865869) ×3 |
| 76.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 76.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 76.0s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 76.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 76.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 76.0s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 76.0s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 76.0s | C2S | m54 格挡状态(值4265865869) ×3 |
| 76.1s | C2S | m15 移动上报 手动 state=7 pos=(849.6,26.5,1002.3) |
| 76.1s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=73 |
| 76.1s | S2C | 属性更新0x10[O#0x3ad5fb] HP=20711 |
| 76.1s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 76.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=85 |
| 76.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=24293 |
| 76.2s | S2C | 对象同步0x21[0x701e5a4] 码147 len=47 |
| 76.4s | C2S | m15 移动上报 手动 state=7 pos=(851.0,27.3,1002.4) |
| 76.5s | S2C | op28(子码47) len=226 ×2 |
| 76.6s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=165 |
| 76.6s | S2C | 对象同步0x21[0x701e5a4] 码69 len=47 |
| 76.9s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=2 |
| 77.1s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=72 |
| 77.1s | S2C | 属性更新0x10[O#0x3ad5fb] HP=20427 |
| 77.1s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 77.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=24151 |
| 77.1s | C2S | m15 移动上报 手动 state=7 pos=(853.7,28.7,1000.2) |
| 77.2s | S2C | 对象同步0x21[0x701e5a4] 码61 len=47 |
| 77.9s | S2C | S2C m169 buff添加 ["buf_CS_jh_tmjt06_range", obj(0x7007559,0xf5e7f)] ×2 |
| 78.1s | S2C | 多属性0x2D(非标子码02) len=87 2d02008cae00077a5e0f00010077021a |
| 78.2s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=71 |
| 78.2s | S2C | 属性更新0x10[O#0x3ad5fb] HP=20143 |
| 78.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 78.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=84 |
| 78.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=24009 |
| 78.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=20225 |
| 78.9s | S2C | 属性更新0x10[O#0xf5e7f] LogicState=1 |
| 78.9s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 78.9s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=1 |
| 78.9s | S2C | 属性更新0x10[O#0xf5e7a] LogicState=1 |
| 78.9s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +1行 |
| 78.9s | S2C | S2C m24 通用事件[0xf5e7f07007559] 伤害飘字 [678, 97, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031302"] |
| 78.9s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 78.9s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 78.9s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x7007559,0xf5e7f)] |
| 78.9s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x701e5a4] ×2 |
| 78.9s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 87, 160, "ui_jh_001"] |
| 78.9s | S2C | S2C m24 通用事件[0xf5e7f07007559] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031302", "徐傠囃辭睼 "] |
| 78.9s | S2C | 属性更新0x10[O#0xf5e7a] InParry=1 |
| 78.9s | S2C | S2C m24 通用事件[0xf5e7a0700ae8c] 伤害飘字 [678, 97, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031302"] |
| 78.9s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 78.9s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 78.9s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x700ae8c,0xf5e7a)] |
| 78.9s | S2C | 属性更新0x10[O#0xf5e7a] InParry=0 |
| 78.9s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x701e5a4] |
| 78.9s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 92, 160, "ui_jh_001"] |
| 78.9s | S2C | S2C m24 通用事件[0xf5e7a0700ae8c] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031302", "徐傠囃辭睼 "] |
| 79.0s | S2C | 对象同步0x21[0x7007559] 码205 len=223 |
| 79.1s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=70 |
| 79.1s | S2C | 属性更新0x10[O#0x3ad5fb] HP=19941 |
| 79.1s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 79.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=23867 |
| 79.1s | S2C | 多属性0x2D(非标子码02) len=136 2d02008cae00077a5e0f000800a00100 |
| 79.4s | C2S | m15 移动上报 手动 state=7 pos=(863.7,29.8,1000.9) |
| 79.5s | S2C | 对象同步0x21[0x701e5a4] 码85 len=47 |
| 80.2s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=69 |
| 80.2s | S2C | 属性更新0x10[O#0x3ad5fb] HP=19657 |
| 80.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 80.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=83 |
| 80.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=23725 |
| 80.3s | S2C | S2C m566 令牌应答串 ["sE/L/GpgoRMbjw0+86/ZJ7Qbn1qvN2lzU"] |
| 80.4s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 80.9s | S2C | 属性更新0x10[O#0xf5e7f] LogicState=0 |
| 80.9s | S2C | 属性更新0x10[O#0xf5e7a] LogicState=0 |
| 81.2s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=68 |
| 81.2s | S2C | 属性更新0x10[O#0x3ad5fb] HP=19373 |
| 81.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 81.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=23583 |
| 81.6s | S2C | op31(sub3) 0x7005971 →0x70336bb |
| 81.6s | C2S | m15 移动上报 手动 state=7 pos=(873.7,29.8,1001.6) |
| 81.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 81.9s | S2C | 对象同步0x21[0x701e5a4] 码17 len=47 |
| 81.9s | C2S | m15 移动上报 手动 state=5 pos=(874.6,29.8,1001.7) ×2 |
| 81.9s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=2 |
| 82.2s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=67 |
| 82.2s | S2C | 属性更新0x10[O#0x3ad5fb] HP=19089 |
| 82.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 82.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=82 |
| 82.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=23441 |
| 82.5s | S2C | 属性更新0x10[O#0x14b] State= |
| 82.7s | C2S | m15 移动上报 手动 state=7 pos=(878.7,29.8,1002.0) |
| 82.7s | S2C | 对象同步0x21[0x7000146] 码237 len=223 |
| 82.7s | S2C | 多属性0x2D[0x14b0700](布局变体) len=20 2d0100460100074b0100000100540001 |
| 82.9s | C2S | m15 移动上报 手动 state=7 pos=(879.4,29.8,1001.4) |
| 83.0s | S2C | 对象同步0x21[0x701e5a4] 码152 len=47 |
| 83.0s | C2S | m15 移动上报 手动 state=8 pos=(879.8,29.8,1001.4) |
| 83.2s | S2C | 对象同步0x21[0x701e5a4] 码92 len=91 |
| 83.3s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=66 |
| 83.3s | S2C | 属性更新0x10[O#0x3ad5fb] HP=18805 |
| 83.3s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 83.3s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=23299 |
| 83.4s | C2S | m15 移动上报 手动 state=6 pos=(881.4,31.1,1001.5) |
| 83.5s | C2S | m15 移动上报 手动 state=8 pos=(882.1,31.3,1001.6) |
| 83.6s | S2C | op28(子码47) len=234 |
| 83.6s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=158 |
| 83.6s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=1 |
| 83.6s | S2C | 对象同步0x21[0x701e5a4] 码212 len=91 |
| 83.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=18887 |
| 83.7–84.0s | C2S | m15 移动上报 手动 state=6 pos=(883.1,33.3,1001.6) ×2 |
| 83.8s | S2C | 对象同步0x21[0x701e5a4] 码228 len=47 |
| 84.1s | S2C | 对象同步0x21[0x701e5a4] 码75 len=47 |
| 84.2s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=65 |
| 84.2s | S2C | 属性更新0x10[O#0x3ad5fb] HP=18603 |
| 84.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 84.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=81 |
| 84.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=23157 |
| 84.3s | C2S | m15 移动上报 手动 state=6 pos=(885.5,36.6,1001.8) |
| 84.5s | S2C | 对象同步0x21[0x701e5a4] 码184 len=47 |
| 84.6s | S2C | op31(sub8) 0x7011265 →0x70336c3 |
| 84.6s | S2C | op0D len=251 0d9fb603072b043b009e676444b88b2f42c3b275 |
| 84.7s | C2S | m15 移动上报 手动 state=6 pos=(887.5,37.2,1001.9) |
| 84.9s | S2C | 对象同步0x21[0x701e5a4] 码128 len=47 |
| 85.0s | C2S | m54 格挡状态(值4489905844) |
| 85.1s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 85.1s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 85.1s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 85.1s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 85.1s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 85.1s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 85.1s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 85.1s | C2S | m15 移动上报 手动 state=6 pos=(889.0,36.4,1002.0) ×2 |
| 85.2s | C2S | m54 格挡状态(值4265865869) |
| 85.2s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=64 |
| 85.2s | S2C | 属性更新0x10[O#0x3ad5fb] HP=18319 |
| 85.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 85.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=23015 |
| 85.3s | C2S | m54 格挡状态(值4265865869) ×3 |
| 85.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 85.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 85.3s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 85.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 85.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 85.3s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 85.3s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 85.3s | S2C | 对象同步0x21[0x701e5a4] 码24 len=47 |
| 85.3s | C2S | m54 格挡状态(值4265865869) |
| 85.3s | C2S | m15 移动上报 手动 state=6 pos=(889.5,34.8,1001.9) |
| 85.4s | C2S | m216 使用轻功 ["qinggong_9", 1] |
| 85.5s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=148 |
| 85.5s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=DA,701E5A4,3AD5FB,1,0,0 |
| 85.5s | S2C | S2C m169 buff添加 ["buf_qg_light", obj(0x701e5a4,0x3ad5fb)] |
| 85.5s | C2S | m15 移动上报 手动 state=7 pos=(890.2,33.6,1001.7) |
| 85.6–86.9s | S2C | 对象同步0x21[0x701e5a4] 码203 len=47 ×2 |
| 85.6s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=0 |
| 85.6s | C2S | m216 使用轻功 ["qinggong_9", 0] |
| 85.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 85.6s | S2C | op28(子码47) len=230 ×2 |
| 85.6s | C2S | m216 使用轻功 ["qinggong_9", 0] |
| 85.6s | C2S | m15 移动上报 手动 state=7 pos=(890.4,33.2,1001.7) |
| 85.6s | S2C | S2C m169 buff移除 ["buf_qg_light", obj(0x701e5a4,0x3ad5fb)] |
| 85.6s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 85.7s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=158 |
| 85.8s | S2C | 对象同步0x21[0x701e5a4] 码72 len=91 |
| 85.8s | C2S | m15 移动上报 手动 state=8 pos=(891.1,33.8,1001.4) |
| 86.0s | S2C | 对象同步0x21[0x701e5a4] 码143 len=47 |
| 86.1s | C2S | m15 移动上报 手动 state=6 pos=(892.4,35.1,1001.0) |
| 86.2s | S2C | 对象同步0x21[0x701e5a4] 码200 len=47 |
| 86.3s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=63 |
| 86.3s | S2C | 属性更新0x10[O#0x3ad5fb] HP=18035 |
| 86.3s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 86.3s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=80 |
| 86.3s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=22873 |
| 86.4s | C2S | m15 移动上报 手动 state=8 pos=(893.4,35.3,1000.7) |
| 86.4s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=151 |
| 86.4s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=1 |
| 86.5s | S2C | 对象同步0x21[0x701e5a4] 码95 len=47 |
| 86.6s | C2S | m15 移动上报 手动 state=6 pos=(894.2,37.3,1000.5) |
| 86.7s | S2C | 对象同步0x21[0x701e5a4] 码42 len=47 |
| 86.7s | S2C | 多属性0x2D[0x4420700](布局变体) len=189 2d0100a75d000742043b000f00b50004 |
| 86.8s | C2S | m15 移动上报 手动 state=6 pos=(895.3,39.1,1000.1) |
| 86.9s | C2S | m54 格挡状态(值4489905844) |
| 86.9s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 86.9s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 86.9s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 86.9s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 86.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 86.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 86.9s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 86.9–87.1s | C2S | m15 移动上报 手动 state=6 pos=(895.8,39.9,1000.0) ×3 |
| 87.2s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=62 |
| 87.2s | S2C | 属性更新0x10[O#0x3ad5fb] HP=17751 |
| 87.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 87.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=22731 |
| 87.3s | S2C | 对象同步0x21[0x701e5a4] 码63 len=91 |
| 87.4s | C2S | m54 格挡状态(值4265865869) |
| 87.4s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 87.4s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 87.4s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 87.4s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 87.4s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 87.4s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 87.4s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 87.4s | C2S | m54 格挡状态(值4265865869) ×2 |
| 87.5s | C2S | m15 移动上报 手动 state=6 pos=(896.9,41.3,999.8) |
| 87.5s | C2S | m54 格挡状态(值4265865869) |
| 87.6s | S2C | op0D len=251 0d65d900073d043b002b776944e8fb4242a15277 ×2 |
| 87.8s | S2C | 对象同步0x21[0x701e5a4] 码159 len=47 |
| 87.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 87.9s | C2S | m15 移动上报 手动 state=5 pos=(898.5,40.7,999.7) ×2 |
| 87.9–88.1s | C2S | m15 移动上报 手动 state=6 pos=(898.7,40.6,999.7) ×2 |
| 87.9s | S2C | 对象同步0x21[0x701e5a4] 码127 len=47 |
| 88.2s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=0 |
| 88.2s | C2S | m15 移动上报 手动 state=4 pos=(900.3,38.3,999.6) |
| 88.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 88.3s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=61 |
| 88.3s | S2C | 属性更新0x10[O#0x3ad5fb] HP=17467 |
| 88.3s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 88.3s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=79 |
| 88.3s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=22589 |
| 88.3s | C2S | m15 移动上报 手动 state=7 pos=(900.3,38.3,999.6) |
| 88.3s | S2C | S2C m586 状态推送(簇首) [5] |
| 88.4s | S2C | 对象同步0x21[0x701e5a4] 码91 len=91 |
| 88.4s | C2S | m15 移动上报 手动 state=8 pos=(901.1,39.0,999.5) |
| 88.6s | S2C | op06 len=1 06 |
| 88.6s | S2C | 对象同步0x21[0x701e5a4] 码107 len=47 |
| 88.6s | S2C | op28(子码47) len=235 |
| 88.6s | C2S | m15 移动上报 手动 state=6 pos=(901.8,41.0,999.5) |
| 88.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=62 |
| 88.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=17549 |
| 88.7s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=161 |
| 88.8s | S2C | 对象同步0x21[0x701e5a4] 码134 len=47 |
| 88.8s | S2C | op2E len=216 2e47d7000000b4010000000000c00200519f0007 |
| 88.8s | C2S | m15 移动上报 手动 state=8 pos=(902.7,42.6,999.4) |
| 88.9s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=154 |
| 88.9s | S2C | 属性更新0x10[O#0x3ad5fb] LiveGroove_4=1000 |
| 88.9s | S2C | S2C m1 系统提示 [1, "30504", 1] |
| 88.9s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] c1=1 [1] |
| 88.9s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 动作开始 ["huodelilian"] |
| 88.9s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=1 |
| 89.0–89.2s | S2C | 对象同步0x21[0x701e5a4] 码202 len=47 ×2 |
| 89.0s | C2S | m15 移动上报 手动 state=6 pos=(903.6,44.6,999.3) |
| 89.2s | C2S | m82 ? [3] |
| 89.3s | C2S | m15 移动上报 手动 state=6 pos=(904.7,46.3,999.3) |
| 89.3s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=61 |
| 89.3s | S2C | 属性更新0x10[O#0x3ad5fb] HP=17265 |
| 89.3s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 89.3s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=22447 |
| 89.4s | S2C | 对象同步0x21[0x701e5a4] 码20 len=47 |
| 89.6s | C2S | m15 移动上报 手动 state=6 pos=(906.1,47.9,999.2) |
| 89.6s | S2C | op31(sub4) 0x700ae8c →0x7007559 |
| 89.6s | S2C | op28(子码47) len=232 |
| 89.6s | S2C | op0D len=251 0d629f00074e043b00e9366b44e8fb4242c0267c |
| 89.6s | S2C | op28(子码47) len=233 ×2 |
| 89.6s | S2C | 对象同步0x21[0x701e5a4] 码92 len=47 |
| 89.6s | S2C | 多属性0x2D[0x4620700](布局变体) len=209 2d0100569f000762043b001000b50004 |
| 89.7s | C2S | m54 格挡状态(值4489905844) |
| 89.7s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 89.7s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 89.7s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 89.7s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 89.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 89.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 89.7s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 89.8s | C2S | m15 移动上报 手动 state=6 pos=(906.8,48.4,999.1) |
| 89.9s | S2C | 对象同步0x21[0x701e5a4] 码114 len=47 |
| 89.9s | C2S | m54 格挡状态(值4265865869) |
| 90.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 90.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 90.0s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 90.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 90.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 90.0s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 90.0s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 90.0s | C2S | m54 格挡状态(值4265865869) ×2 |
| 90.1s | C2S | m15 移动上报 手动 state=6 pos=(907.3,48.5,999.1) |
| 90.1s | C2S | m54 格挡状态(值4265865869) |
| 90.3s | S2C | 对象同步0x21[0x701e5a4] 码28 len=47 |
| 90.4s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=60 |
| 90.4s | S2C | 属性更新0x10[O#0x3ad5fb] HP=16981 |
| 90.4s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 90.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=78 |
| 90.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=22305 |
| 90.4s | C2S | m15 移动上报 手动 state=6 pos=(909.0,47.6,999.2) |
| 90.5s | S2C | op2C len=142 2c478d0000000a010000000000e0060046010007 |
| 90.6s | S2C | op31(sub1) 0x7000146 |
| 90.6s | S2C | op28(子码47) len=235 ×2 |
| 90.7s | C2S | m216 使用轻功 ["qinggong_9", 1] |
| 90.7s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=144 |
| 90.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=DA,701E5A4,3AD5FB,1,0,0 |
| 90.7s | S2C | S2C m169 buff添加 ["buf_qg_light", obj(0x701e5a4,0x3ad5fb)] |
| 90.7s | C2S | m15 移动上报 手动 state=6 pos=(910.3,45.9,999.3) |
| 90.8s | C2S | m15 移动上报 手动 state=7 pos=(910.5,45.5,999.3) |
| 90.9s | C2S | m216 使用轻功 ["qinggong_9", 0] |
| 90.9s | S2C | S2C m169 buff移除 ["buf_qg_light", obj(0x701e5a4,0x3ad5fb)] |
| 90.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 90.9s | C2S | m216 使用轻功 ["qinggong_9", 0] |
| 91.0s | S2C | 对象同步0x21[0x701e5a4] 码144 len=91 |
| 91.0s | C2S | m15 移动上报 手动 state=8 pos=(911.3,44.5,999.3) |
| 91.1s | S2C | 对象同步0x21[0x701e5a4] 码125 len=47 |
| 91.2s | C2S | m15 移动上报 手动 state=4 pos=(911.3,44.1,999.3) |
| 91.3s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=0 |
| 91.3s | C2S | m15 移动上报 手动 state=7 pos=(911.3,44.1,999.3) |
| 91.3s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=59 |
| 91.3s | S2C | 属性更新0x10[O#0x3ad5fb] HP=16697 |
| 91.3s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 91.3s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=22163 |
| 91.4s | S2C | 对象同步0x21[0x701e5a4] 码124 len=91 |
| 91.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 91.4s | C2S | m15 移动上报 手动 state=8 pos=(912.1,44.7,999.3) |
| 91.6s | S2C | 对象同步0x21[0x701e5a4] 码51 len=47 |
| 91.7s | S2C | op31(sub1) 0x703b3d5 |
| 91.7s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=154 |
| 91.8s | C2S | m15 移动上报 手动 state=6 pos=(913.4,46.0,999.4) |
| 92.0s | C2S | m15 移动上报 手动 state=8 pos=(914.2,46.2,999.4) |
| 92.0s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=147 |
| 92.0s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=1 |
| 92.1s | S2C | 对象同步0x21[0x701e5a4] 码157 len=91 |
| 92.2s | C2S | m15 移动上报 手动 state=6 pos=(915.1,48.2,999.5) |
| 92.2s | S2C | 对象同步0x21[0x701e5a4] 码231 len=47 |
| 92.3s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=58 |
| 92.3s | S2C | 属性更新0x10[O#0x3ad5fb] HP=16413 |
| 92.3s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 92.3s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=77 |
| 92.3s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=22021 |
| 92.4s | C2S | m15 移动上报 手动 state=6 pos=(916.1,50.0,999.5) |
| 92.4s | S2C | 对象同步0x21[0x701e5a4] 码233 len=47 |
| 92.6s | S2C | op31(sub2) 0x70211c6 →0x7007d62 |
| 92.7–93.1s | C2S | m15 移动上报 手动 state=6 pos=(917.5,51.6,999.6) ×2 |
| 92.9s | S2C | 对象同步0x21[0x701e5a4] 码79 len=47 |
| 93.3s | S2C | 对象同步0x21[0x701e5a4] 码130 len=47 |
| 93.4s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=57 |
| 93.4s | S2C | 属性更新0x10[O#0x3ad5fb] HP=16129 |
| 93.4s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 93.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=21879 |
| 93.5s | C2S | m15 移动上报 手动 state=6 pos=(921.1,51.2,999.8) |
| 93.7s | S2C | op0D len=63 0df3330007bb34000093007044db792f429c2c7e |
| 93.7s | S2C | op28(子码47) len=222 ×4 |
| 93.7s | S2C | 对象同步0x21[0x701e5a4] 码243 len=47 |
| 93.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=16211 |
| 93.8s | C2S | m15 移动上报 手动 state=6 pos=(922.4,49.6,999.8) |
| 93.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 93.8s | C2S | m15 移动上报 手动 state=5 pos=(922.5,49.4,999.8) ×2 |
| 93.9s | S2C | 对象同步0x21[0x701e5a4] 码136 len=47 |
| 93.9s | S2C | 属性更新0x10[O#0x3ad5fb] JumpTimes=0 |
| 93.9s | C2S | m15 移动上报 手动 state=4 pos=(922.9,48.7,999.8) |
| 93.9s | C2S | m15 移动上报 手动 state=7 pos=(922.9,48.7,999.8) |
| 94.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 94.0s | C2S | m15 移动上报 自动 state=4 pos=(923.3,48.7,999.9) |
| 94.1s | S2C | 对象同步0x21[0x701e5a4] 码127 len=135 |
| 94.1s | S2C | S2C m169 buff添加 ["buf_CS_jh_tmjt06_range", obj(0x700e282,0x3b0435)] ×2 |
| 94.1s | S2C | 多属性0x2D(非标子码02) len=87 2d02008af2020730043b00010077021a |
| 94.1s | C2S | m54 格挡状态(值4489905844) |
| 94.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 94.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 94.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 94.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 94.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 94.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 94.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 94.4s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=56 |
| 94.4s | S2C | 属性更新0x10[O#0x3ad5fb] HP=15927 |
| 94.4s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 94.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=76 |
| 94.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=21737 |
| 94.5s | C2S | m15 移动上报 手动 state=4 pos=(923.3,48.7,999.9) |
| 94.6s | S2C | 对象同步0x21[0x701e5a4] 码167 len=47 |
| 94.8s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=157 |
| 95.0s | C2S | m15 移动上报 手动 state=7 pos=(923.3,48.7,999.9) |
| 95.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 95.1s | S2C | 属性更新0x10[O#0x3b0435] LogicState=1 |
| 95.1s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 95.1s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=1 |
| 95.1s | S2C | 属性更新0x10[O#0x3b0430] LogicState=1 |
| 95.1s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +1行 |
| 95.1s | S2C | S2C m24 通用事件[0x3b04350700e282] 伤害飘字 [678, 97, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031302"] |
| 95.1s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 95.1s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 95.1s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x700e282,0x3b0435)] |
| 95.1s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x701e5a4] ×2 |
| 95.1s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 97, 160, "ui_jh_001"] |
| 95.1s | S2C | S2C m24 通用事件[0x3b04350700e282] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031302", "徐傠囃辭睼 "] |
| 95.1s | S2C | 属性更新0x10[O#0x3b0430] InParry=1 |
| 95.1s | S2C | S2C m24 通用事件[0x3b04300702f28a] 伤害飘字 [678, 97, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031302"] |
| 95.1s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 95.1s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 95.1s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x702f28a,0x3b0430)] |
| 95.1s | S2C | 属性更新0x10[O#0x3b0430] InParry=0 |
| 95.1s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x701e5a4] |
| 95.1s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 102, 160, "ui_jh_001"] |
| 95.1s | S2C | S2C m24 通用事件[0x3b04300702f28a] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031302", "徐傠囃辭睼 "] |
| 95.2s | C2S | m54 格挡状态(值4265865869) |
| 95.2s | S2C | op2C len=184 2c47b70000000a010000000000800600a4e50107 |
| 95.2s | S2C | 多属性0x2D(非标子码02) len=136 2d02008af2020730043b000800a00100 |
| 95.3s | C2S | m54 格挡状态(值4265865869) ×3 |
| 95.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 95.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 95.3s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 95.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 95.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 95.3s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 95.3s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 95.3s | C2S | m54 格挡状态(值4265865869) ×2 |
| 95.3s | C2S | m15 移动上报 手动 state=7 pos=(923.9,48.7,999.9) |
| 95.4s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=55 |
| 95.4s | S2C | 属性更新0x10[O#0x3ad5fb] HP=15643 |
| 95.4s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 95.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=21595 |
| 95.5s | S2C | 对象同步0x21[0x701e5a4] 码234 len=47 |
| 95.7s | S2C | op28(子码47) len=237 ×2 |
| 96.4s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=54 |
| 96.4s | S2C | 属性更新0x10[O#0x3ad5fb] HP=15359 |
| 96.4s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 96.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=75 |
| 96.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=21453 |
| 96.7s | S2C | op31(sub2) 0x7009041 →0x700c97d |
| 96.9–97.1s | S2C | S2C m386 UI开关推送 [1, 1, "ui_guild_kill"] ×4 |
| 97.2s | S2C | 属性更新0x10[O#0x3b0435] LogicState=0 |
| 97.2s | S2C | 属性更新0x10[O#0x3b0430] LogicState=0 |
| 97.4s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=53 |
| 97.4s | S2C | 属性更新0x10[O#0x3ad5fb] HP=15075 |
| 97.4s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 97.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=21311 |
| 97.6s | C2S | m15 移动上报 手动 state=7 pos=(934.0,48.7,999.8) |
| 97.7s | S2C | 属性更新0x10[O#0x3ad5fb] QingGongPoint=165 |
| 97.8s | S2C | 对象同步0x21[0x701e5a4] 码0 len=47 |
| 98.2s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=2 |
| 98.2s | S2C | S2C m169 buff添加 ["buf_CS_jh_tmjt06_range", obj(0x7009f55,0x3b0458)] ×2 |
| 98.4s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=52 |
| 98.4s | S2C | 属性更新0x10[O#0x3ad5fb] HP=14791 |
| 98.4s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 98.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=74 |
| 98.4s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=21169 |
| 98.4s | S2C | 多属性0x2D(非标子码02) len=87 2d0200519f000753043b00010077021a |
| 98.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=14873 |
| 99.2s | S2C | 属性更新0x10[O#0x3b0458] LogicState=1 |
| 99.2s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +0行 |
| 99.2s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=1 |
| 99.2s | S2C | 属性更新0x10[O#0x3b0458] CurSkillID=default_normal_skill |
| 99.2s | S2C | 属性更新0x10[O#0x3b0458] CurSkillTarget=None |
| 99.2s | S2C | 位置0x1F(短len=25) |
| 99.2s | S2C | S2C m337 施法回显[0x3b045807009f55] [939.809, 48.746, 998.046, 0.49, 0] |
| 99.2s | S2C | S2C m20 范围施放标记[0x3b045807009f55] ["atkhit1", obj(0x701e5a4,0x3ad5fb), 0, 0.0] |
| 99.2s | S2C | S2C m24 通用事件[0x3b045807009f55] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031301"] |
| 99.2s | S2C | 属性更新0x10[O#0x3b0453] InParry=1 |
| 99.2s | S2C | 属性更新0x10[O#0x3b0453] LogicState=1 |
| 99.2s | S2C | 记录加行0x11[fight_relation_rec#114 0x701e5a4/0x3ad5fb] +1行 |
| 99.2s | S2C | S2C m24 通用事件[0x3b045307009f51] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031301"] |
| 99.3s | S2C | op2C len=109 2c476c00000062010000000000800800559f0007 |
| 99.5s | S2C | 多属性0x2D(非标子码02) len=99 2d0200519f000753043b000400d70141 |
| 99.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=51 |
| 99.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=14589 |
| 99.5s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 99.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=21027 |
| 99.7s | C2S | m15 移动上报 自动 state=4 pos=(942.5,49.7,999.8) |
| 99.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 99.8s | C2S | m15 移动上报 自动 state=5 pos=(942.5,49.7,999.8) ×2 |
| 99.9–100.1s | S2C | 对象同步0x21[0x701e5a4] 码24 len=47 ×2 |
| 99.9s | C2S | m15 移动上报 手动 state=7 pos=(942.5,49.7,999.8) |
| 100.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 100.2s | S2C | 属性更新0x10[O#0x3ad5fb] LastAttackerNpc=monm031301 |
| 100.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] c1=7 [1, obj(0x7009f55,0x3b0458), "default_normal_skill", "徐傠囃辭睼 ", "monm031301", 100] |
| 100.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 100.2s | S2C | 属性更新0x10[O#0x3b0458] State= |
| 100.2s | S2C | 属性更新0x10[O#0x3b0458] CurSkillID= |
| 100.2s | S2C | 属性更新0x10[O#0x3b0458] CurSkillTarget=None |
| 100.2s | S2C | S2C m189 对象通知 [obj(0x7009f55,0x3b0458)] |
| 100.2s | S2C | 属性更新0x10[O#0x3b0458] InParry=1 |
| 100.2s | S2C | S2C m24 通用事件[0x3b045807009f55] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031301"] |
| 100.2s | S2C | 记录写格0x13[fight_relation_rec#114 行1 列1 =16777472 0x701e5a4] |
| 100.2s | S2C | S2C m24 通用事件[0x3b045307009f51] 伤害飘字 [775, 0, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031301"] |
| 100.3s | S2C | 多属性0x2D(非标子码02) len=79 2d0200519f000753043b000200d7011e |
| 100.4s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 100.4s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 100.4s | S2C | S2C m250 ? [] ×2 |
| 100.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=50 |
| 100.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=14305 |
| 100.5s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 100.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=73 |
| 100.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=20885 |
| 100.9s | C2S | m15 移动上报 自动 state=4 pos=(941.5,48.7,999.8) |
| 101.0–102.7s | S2C | 对象同步0x21[0x701e5a4] 码162 len=47 ×2 |
| 101.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 101.2s | S2C | S2C m24 通用事件[0x3b045807009f55] 伤害飘字 [682, 93, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031301"] |
| 101.2s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 101.2s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 101.2s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x7009f55,0x3b0458)] |
| 101.2s | S2C | 属性更新0x10[O#0x3b0458] InParry=0 |
| 101.2s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x701e5a4] ×2 |
| 101.2s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 109, 160, "ui_jh_001"] |
| 101.2s | S2C | S2C m24 通用事件[0x3b045807009f55] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031301", "徐傠囃辭睼 "] |
| 101.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x701e5a4] |
| 101.2s | S2C | S2C m24 通用事件[0x3b045307009f51] 伤害飘字 [682, 93, 3, "buf_CS_jh_tmjt06_range", "Buffer", "monm031301"] |
| 101.2s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x701e5a4] |
| 101.2s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x701e5a4] |
| 101.2s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06_range", obj(0x7009f51,0x3b0453)] |
| 101.2s | S2C | 属性更新0x10[O#0x3b0453] InParry=0 |
| 101.2s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x701e5a4] |
| 101.2s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei4", 116, 160, "ui_jh_001"] |
| 101.2s | S2C | S2C m24 通用事件[0x3b045307009f51] 击退 [obj(0x701e5a4,0x3ad5fb), "monm031301", "徐傠囃辭睼 "] |
| 101.4s | S2C | 多属性0x2D(非标子码02) len=143 2d0200519f000753043b000800d70100 |
| 101.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=49 |
| 101.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=14021 |
| 101.5s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 101.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=20743 |
| 102.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=48 |
| 102.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=13737 |
| 102.5s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 102.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=72 |
| 102.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=20601 |
| 102.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=47 |
| 102.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=13453 |
| 102.5s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 102.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=20459 |
| 102.5s | C2S | m15 移动上报 手动 state=7 pos=(941.5,48.7,999.8) |
| 102.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 102.9s | C2S | m15 移动上报 自动 state=4 pos=(940.1,48.7,999.8) |
| 103.1s | S2C | 对象同步0x21[0x701e5a4] 码135 len=47 |
| 103.2s | S2C | 属性更新0x10[O#0x3b0458] LogicState=0 |
| 103.2s | S2C | 属性更新0x10[O#0x3b0453] LogicState=0 |
| 103.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=46 |
| 103.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=13169 |
| 103.5s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 103.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=71 |
| 103.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=20317 |
| 103.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=13251 |
| 103.8s | S2C | op06 len=1 06 |
| 104.3s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=2 |
| 104.4s | S2C | 属性更新0x10[O#0x3ad5fb] FightData=98091892830 |
| 104.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=45 |
| 104.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=12967 |
| 104.5s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 104.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=20175 |
| 105.5s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=44 |
| 105.5s | S2C | 属性更新0x10[O#0x3ad5fb] HP=12683 |
| 105.5s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 105.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=70 |
| 105.5s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=20033 |
| 105.6s | S2C | op0E len=9 0e519f000753043b00 ×2 |
| 105.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 105.9s | C2S | m15 移动上报 自动 state=5 pos=(940.1,48.7,999.8) ×2 |
| 106.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=43 |
| 106.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=12399 |
| 106.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 106.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=19891 |
| 107.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=42 |
| 107.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=12115 |
| 107.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 107.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=69 |
| 107.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=19749 |
| 107.7s | S2C | op28(子码47) len=230 |
| 108.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=41 |
| 108.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=11831 |
| 108.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 108.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=19607 |
| 108.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=42 |
| 108.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=11913 |
| 108.7s | S2C | op28(子码47) len=231 |
| 109.0s | C2S | m54 格挡状态(值33369417533) |
| 109.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 109.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 109.0s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 109.0s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 109.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 109.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 109.0s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 109.1s | C2S | m54 格挡状态(值33145377558) |
| 109.2s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 109.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 109.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 109.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 109.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 109.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 109.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 109.2s | C2S | m54 格挡状态(值33145377558) ×2 |
| 109.3s | C2S | m82 ? [3] |
| 109.3s | C2S | m54 格挡状态(值33145377558) |
| 109.3s | C2S | m15 移动上报 自动 state=4 pos=(940.1,48.7,999.8) |
| 109.5–110.0s | S2C | 对象同步0x21[0x701e5a4] 码135 len=47 ×2 |
| 109.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=41 |
| 109.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=11629 |
| 109.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 109.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=68 |
| 109.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=19465 |
| 109.8s | C2S | m15 移动上报 自动 state=4 pos=(940.1,48.7,999.8) |
| 110.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=40 |
| 110.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=11345 |
| 110.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 110.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=19323 |
| 111.1–112.5s | S2C | 对象同步0x21[0x7000152] 码149 len=91 ×2 |
| 111.2s | C2S | m15 移动上报 自动 state=7 pos=(940.1,48.7,999.8) |
| 111.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 111.2s | C2S | m15 移动上报 自动 state=7 pos=(940.1,48.7,999.8) |
| 111.4–113.6s | S2C | 对象同步0x21[0x701e5a4] 码0 len=91 ×3 |
| 111.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=39 |
| 111.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=11061 |
| 111.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 111.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=67 |
| 111.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=19181 |
| 111.6s | S2C | 属性更新0x10[O#0x150] State=stand ×2 |
| 111.7s | C2S | m54 格挡状态(值33369417533) |
| 111.7s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 111.7s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 111.7s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 111.7s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 111.7s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 111.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 111.7s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 111.7s | C2S | m15 移动上报 自动 state=7 pos=(940.2,48.7,997.4) |
| 111.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 111.9s | S2C | op2C len=145 2c47900000000a010000000000e006004b010007 |
| 111.9s | S2C | 多属性0x2D[0x1500700](布局变体) len=23 2d01004b010007500100000100540004 |
| 111.9s | S2C | 属性更新0x10[O#0x15c] State=stand |
| 111.9s | C2S | m15 移动上报 自动 state=5 pos=(940.2,48.8,997.2) ×2 |
| 112.1s | S2C | 对象同步0x21[0x7000157] 码97 len=91 |
| 112.1s | C2S | m54 格挡状态(值33145377558) |
| 112.2s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 112.2s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 112.2s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 112.2s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 112.2s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 112.2s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 112.2s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 112.2s | C2S | m15 移动上报 手动 state=7 pos=(940.2,48.7,996.7) |
| 112.2s | C2S | m54 格挡状态(值33145377558) ×2 |
| 112.3s | C2S | m15 移动上报 手动 state=7 pos=(940.2,48.7,996.8) |
| 112.3s | S2C | 对象同步0x21[0x701e5a4] 码98 len=91 |
| 112.3s | S2C | 属性更新0x10[O#0x157] State=stand |
| 112.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=38 |
| 112.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=10777 |
| 112.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 112.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=19039 |
| 112.7s | C2S | m54 格挡状态(值33369417533) |
| 112.8s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 112.8s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 112.8s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 112.8s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 112.8s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 112.8s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 112.8s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 112.8s | C2S | m15 移动上报 手动 state=7 pos=(938.8,48.7,998.9) ×2 |
| 113.0s | C2S | m54 格挡状态(值33145377558) ×2 |
| 113.0s | S2C | 对象同步0x21[0x701e5a4] 码215 len=47 |
| 113.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 113.0s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 113.0s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 113.0s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 113.0s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 113.0s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 113.0s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 113.0s | C2S | m15 移动上报 自动 state=4 pos=(938.8,48.7,999.3) |
| 113.0s | C2S | m54 格挡状态(值33145377558) ×3 |
| 113.1s | S2C | 属性更新0x10[O#0x157] State=walk |
| 113.1s | S2C | 属性更新0x10[O#0x15c] State=walk |
| 113.1s | C2S | m54 格挡状态(值33145377558) |
| 113.2s | C2S | m15 移动上报 手动 state=7 pos=(938.8,48.7,999.3) |
| 113.2s | S2C | op2C len=173 2c47ac00000036010000000000e0070052010007 |
| 113.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 113.3s | C2S | m54 格挡状态(值33369417533) |
| 113.3s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=1.77 |
| 113.3s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=-2.13 |
| 113.3s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=1 |
| 113.3s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=-93 |
| 113.3s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4=11CE,701E5A4,3AD5FB,1,0,0 |
| 113.3s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 113.3s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=1 |
| 113.3s | C2S | m15 移动上报 手动 state=7 pos=(938.2,48.7,998.8) ×2 |
| 113.6s | C2S | m54 格挡状态(值33145377558) |
| 113.6s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x701e5a4,0x3ad5fb)] |
| 113.6s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo4= |
| 113.6s | S2C | 属性更新0x10[O#0x3ad5fb] InParry=0 |
| 113.6s | S2C | 属性更新0x10[O#0x3ad5fb] MoveSpeed=4.5 |
| 113.6s | S2C | 属性更新0x10[O#0x3ad5fb] RunSpeedAdd=0.6 |
| 113.6s | S2C | 属性更新0x10[O#0x3ad5fb] ParryUpSpeedAdd=10 |
| 113.6s | S2C | 属性更新0x10[O#0x3ad5fb] InManualParry=0 |
| 113.6s | C2S | m54 格挡状态(值33145377558) ×2 |
| 113.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=37 |
| 113.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=10493 |
| 113.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 113.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=66 |
| 113.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=18897 |
| 113.7s | C2S | m15 移动上报 手动 state=7 pos=(937.7,48.7,998.7) |
| 113.7s | C2S | m54 格挡状态(值33145377558) |
| 113.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=10575 |
| 113.8s | S2C | 对象同步0x21[0x701e5a4] 码55 len=47 |
| 114.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=36 |
| 114.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=10291 |
| 114.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 114.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=18755 |
| 114.9–115.1s | S2C | S2C m386 UI开关推送 [1, 1, "ui_guild_kill"] ×4 |
| 115.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=35 |
| 115.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=10007 |
| 115.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 115.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=65 |
| 115.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=18613 |
| 115.8s | C2S | m15 移动上报 自动 state=4 pos=(928.3,48.7,998.9) |
| 115.8s | S2C | op31(sub1) 0x7000157 |
| 116.0s | S2C | 对象同步0x21[0x701e5a4] 码254 len=47 |
| 116.1s | S2C | 属性更新0x10[O#0x3ad5fb] Faculty=114282578 |
| 116.1s | S2C | 属性更新0x10[V#0x28] CurFillValue=351084 |
| 116.1s | S2C | 属性更新0x10[O#0x3ad5fb] CurFillValue=351084 |
| 116.7s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=34 |
| 116.7s | S2C | 属性更新0x10[O#0x3ad5fb] HP=9723 |
| 116.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 伤害飘字 [284, 0, 3, "buf_CS_jh_tmjt06", "Buffer", "徐傠囃辭睼 "] |
| 116.7s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=18471 |
| 116.8s | S2C | op31(sub1) 0x7009041 |
| 116.9s | C2S | m211 使用技能(出招) CS_wd_tjj02 10值 [928.328, 48.748, 998.877, -1.551] |
| 116.9s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x701e5a4] ×2 |
| 116.9s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID=CS_wd_tjj02 |
| 116.9s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillTarget=None |
| 116.9s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +93行 |
| 116.9s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =23808 0x701e5a4] ×4 |
| 116.9s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x701e5a4] ×2 |
| 116.9s | S2C | 记录加行0x11[cooldown_rec#109 0x701e5a4/0x3ad5fb] +94行 |
| 116.9s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x701e5a4] ×4 |
| 116.9s | S2C | S2C m20 范围施放标记[0x3ad5fb0701e5a4] ["CS_wd_tjj02", obj(0x701e5a4,0x3ad5fb), 0, 0.0] |
| 116.9s | S2C | S2C m169 buff移除 ["buf_CS_jh_tmjt06", obj(0x701e5a4,0x3ad5fb)] |
| 116.9s | S2C | 属性更新0x10[O#0x3ad5fb] BufferInfo5= |
| 117.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=89 |
| 117.1s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25456 |
| 117.1s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=89 |
| 117.1s | S2C | 属性更新0x10[O#0x3ad5fb] HP=25456 |
| 117.1s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=30 |
| 117.1s | S2C | 属性更新0x10[O#0x3ad5fb] MP=3214 |
| 117.1s | S2C | S2C m205 伤害目标列表[0x3ad5fb0701e5a4] ["CS_wd_tjj02", obj(0x701e5a4,0x3ad5fb)] |
| 117.1s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillID= |
| 117.1s | S2C | 属性更新0x10[O#0x3ad5fb] CurSkillTarget=None |
| 117.8s | S2C | op28(子码47) len=231 |
| 117.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 117.9s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 118.8s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=90 |
| 118.8s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=25538 |
| 118.8s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=90 |
| 118.8s | S2C | 属性更新0x10[O#0x3ad5fb] HP=25538 |
| 119.1s | S2C | op06 len=1 06 |
| 120.0s | C2S | m240 打坐 [1] |
| 120.1s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=102 |
| 120.1s | S2C | S2C m1 系统提示 [2, "9400"] |
| 120.1s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 动作开始 ["dazuo_old"] |
| 120.3s | S2C | 属性更新0x10[O#0x3ad5fb] LiveGroove_4=893 |
| 120.3s | S2C | 属性更新0x10[O#0x3ad5fb] LiveGroove_5=42572 |
| 120.3s | S2C | S2C m1 系统提示 [1, "8052", 223] |
| 120.3s | S2C | S2C m349 ? [223] |
| 120.3s | S2C | 属性更新0x10[O#0x3ad5fb] Faculty=114282801 |
| 120.3s | S2C | 属性更新0x10[O#0x3ad5fb] HoldFaculty=1500740 |
| 121.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=95 |
| 121.6s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=26952 |
| 121.6s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=95 |
| 121.6s | S2C | 属性更新0x10[O#0x3ad5fb] HP=26952 |
| 121.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] HP治疗飘字 [1414, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 121.6s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=35 |
| 121.6s | S2C | 属性更新0x10[O#0x3ad5fb] MP=3733 |
| 121.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] MP回蓝飘字 [519, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 121.7s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=36 |
| 121.7s | S2C | 属性更新0x10[O#0x3ad5fb] MP=3833 |
| 123.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHPRatio=100 |
| 123.2s | S2C | 属性更新0x10[O#0x3ad5fb] HitHP=28285 |
| 123.2s | S2C | 属性更新0x10[O#0x3ad5fb] HPRatio=100 |
| 123.2s | S2C | 属性更新0x10[O#0x3ad5fb] HP=28285 |
| 123.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] HP治疗飘字 [1333, 81, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 123.2s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=41 |
| 123.2s | S2C | 属性更新0x10[O#0x3ad5fb] MP=4352 |
| 123.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] MP回蓝飘字 [519, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 123.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 123.9s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 124.6s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=46 |
| 124.6s | S2C | 属性更新0x10[O#0x3ad5fb] MP=4871 |
| 124.6s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] MP回蓝飘字 [519, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 126.2s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=51 |
| 126.2s | S2C | 属性更新0x10[O#0x3ad5fb] MP=5390 |
| 126.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] MP回蓝飘字 [519, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 126.4s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=52 |
| 126.4s | S2C | 属性更新0x10[O#0x3ad5fb] MP=5490 |
| 127.7s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=57 |
| 127.7s | S2C | 属性更新0x10[O#0x3ad5fb] MP=6009 |
| 127.7s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] MP回蓝飘字 [519, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 129.2s | C2S | m82 ? [3] |
| 129.2s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=62 |
| 129.2s | S2C | 属性更新0x10[O#0x3ad5fb] MP=6528 |
| 129.2s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] MP回蓝飘字 [519, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 129.4s | C2S | m350 ? [0] |
| 129.5s | C2S | m366 ? [0.0] |
| 129.5s | S2C | S2C m580 ? [0] |
| 129.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 130.0s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 130.7s | C2S | m353 ? ["2DA0B90CAA204E659A7145AF0AD2812B"] |
| 130.7s | S2C | 记录写格0x13[RecvLetterRec#319 行31 列1 =151002880 0x701e5a4] |
| 130.8s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=67 |
| 130.8s | S2C | 属性更新0x10[O#0x3ad5fb] MP=7047 |
| 130.8s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] MP回蓝飘字 [519, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 131.2s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=68 |
| 131.2s | S2C | 属性更新0x10[O#0x3ad5fb] MP=7147 |
| 131.5s | S2C | 对象同步0x21[0x7000152] 码177 len=91 |
| 131.7s | C2S | m357 ? [1.0, "2DA0B90CAA204E659A7145AF0AD2812B", 1.0] |
| 131.7s | S2C | 记录写格0x13[RecvLetterRec#319 行31 列1 =201334528 0x701e5a4] |
| 132.3s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=73 |
| 132.3s | S2C | 属性更新0x10[O#0x3ad5fb] MP=7666 |
| 132.3s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] MP回蓝飘字 [519, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 132.5s | C2S | m356 ? [0.0, 2.0] |
| 132.5s | S2C | 记录删行0x12[RecvLetterRec#319 行319 0x701e5a4] |
| 133.7s | S2C | 属性更新0x10[O#0x15c] State=stand |
| 133.8s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=78 |
| 133.8s | S2C | 属性更新0x10[O#0x3ad5fb] MP=8185 |
| 133.8s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] MP回蓝飘字 [519, 0, 0, "zs_default_01", "SkillNormal", "徐傠囃辭睼 "] |
| 133.9s | S2C | 对象同步0x21[0x7000157] 码81 len=91 |
| 134.3s | S2C | op06 len=1 06 |
| 134.5s | S2C | 属性更新0x10[O#0x157] State=stand |
| 134.6s | S2C | 属性更新0x10[V#0x2] LeftTime=603258 |
| 134.7s | S2C | 对象同步0x21[0x7000152] 码177 len=91 |
| 134.8s | C2S | m240 打坐 [2] |
| 134.8s | S2C | 属性更新0x10[O#0x3ad5fb] LogicState=0 |
| 134.8s | S2C | S2C m1 系统提示 [2, "9401"] |
| 134.8s | S2C | S2C m24 通用事件[0x3ad5fb0701e5a4] 动作结束 ["dazuo_old"] |
| 134.9s | C2S | m240 打坐 [2] |
| 135.4s | S2C | 属性更新0x10[O#0x157] State=walk |
| 135.4s | S2C | 属性更新0x10[O#0x15c] State=walk |
| 135.6s | S2C | op2C len=130 2c47810000000a010000000000e0060052010007 |
| 135.8s | S2C | S2C m1 系统提示 [2, "555504", "徐徐荼之 ", "tableitem_JQ_02_SZ"] |
| 135.9s | S2C | 属性更新0x10[O#0x3ad5fb] MPRatio=79 |
| 135.9s | S2C | 属性更新0x10[O#0x3ad5fb] MP=8285 |
| 135.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 136.0s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 136.0s | S2C | op28(子码47) len=233 ×2 |
| 136.8s | S2C | 属性更新0x10[O#0x150] State= |
| 137.0s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 137.0s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |

## b.pcapng — 103.143.19.173:7071

场景：kfkj02/03/04/05 冲锋连打（含空中/无目标形态）。事件 1067 条 → 合并为 667 段。抓包起点 2026-11-08 13:55:13（本地墙钟）

| 时间 | 方向 | 交互 |
|---|---|---|
| 0.0–66.6s | PAIR | C2S m924 MSG_DELAY(client ms) → S2C m294 回显[client ms, obj, 服务器unix秒, …] ×66 |
| 0.0–65.6s | PAIR | C2S ping(0x18) → S2C 0x38 回显 ×61 |
| 0.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 0.6s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 5.7s | C2S | m54 格挡状态(值6902349270) |
| 5.7s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 5.7s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 5.7s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 5.7s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 5.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 5.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 5.7s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 5.8s | C2S | m54 格挡状态(值6678309295) |
| 5.8s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 5.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 5.8s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 5.8s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 5.8s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 5.8s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 5.8s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 5.8s | C2S | m54 格挡状态(值6678309295) ×2 |
| 6.0s | C2S | m15 移动上报 自动 state=4 pos=(928.3,48.7,998.9) |
| 6.3–7.2s | S2C | 对象同步0x21[0x700ca8c] 码254 len=47 ×3 |
| 6.4s | C2S | m54 格挡状态(值6902349270) |
| 6.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 6.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 6.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 6.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 6.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 6.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 6.4s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 6.4s | C2S | m15 移动上报 手动 state=4 pos=(928.3,48.7,998.9) |
| 6.5s | S2C | op06 len=1 06 |
| 6.5s | C2S | m54 格挡状态(值6678309295) ×2 |
| 6.5s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 6.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 6.5s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 6.5s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 6.5s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 6.5s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 6.5s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 6.5s | C2S | m54 格挡状态(值6678309295) |
| 6.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 6.5s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 6.6s | C2S | m54 格挡状态(值6678309295) |
| 6.9s | C2S | m15 移动上报 自动 state=4 pos=(928.3,48.7,998.9) |
| 10.9s | C2S | m107 外观自定义保存 ["e$"] |
| 12.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 12.6s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 15.5s | C2S | m82 ? [3] |
| 15.7s | C2S | m211 使用技能(出招) CS_hs_kfkj05 10值 [928.328, 48.748, 998.877, -1.584] |
| 15.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 15.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj05 |
| 15.7s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +93行 |
| 15.7s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =23808 0x700ca8c] ×4 |
| 15.7s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 15.7s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +94行 |
| 15.7s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x700ca8c] ×4 |
| 15.7s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 15.7s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10342 |
| 15.7s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_hs_kfkj05", 0, 0, 0.0] |
| 15.7s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 16.4s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 17.5s | S2C | op28(子码47) len=231 |
| 17.5s | C2S | m211 使用技能(出招) CS_hs_kfkj04 10值 [928.328, 48.748, 998.877, -1.584] |
| 17.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 17.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj04 |
| 17.5s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +94行 |
| 17.5s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x700ca8c] ×4 |
| 17.5s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =16801024 0x700ca8c] ×2 |
| 17.5s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10301 |
| 17.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=48AB,701C0B9,3B7117,1,C38EA706249,1 |
| 17.5s | S2C | S2C m169 buff添加 ["buf_CS_hs_kfkj04", obj(0x700ca8c,0x3b6fb8)] |
| 17.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo5=48AC,700CA8C,3B6FB8,1,C38EA706249,0 |
| 17.5s | S2C | S2C m169 buff添加 ["buf_CS_hs_kfkj04_01", obj(0x700ca8c,0x3b6fb8)] |
| 17.5s | S2C | 记录写格0x13[cooldown_rec#109 行94 列4 =24064 0x700ca8c] |
| 17.5s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_hs_kfkj04", 0, 0, 0.0] |
| 17.6s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 17.6s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 冲锋动画 [920.329, 48.746, 998.775, 1, ""] |
| 17.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 17.6s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=6.0 |
| 17.6s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=7.6 |
| 17.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 17.7s | S2C | 对象同步0x21[0x700ca8c] 码9 len=47 |
| 17.8s | S2C | 属性更新0x10[O#0x157] State=stand |
| 17.9s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 17.9s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 17.9s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 17.9s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 18.0s | C2S | m212 冲锋/续招落地 [920.376, 48.746, 998.775, -1.584] |
| 18.1s | S2C | 对象同步0x21[0x7000152] 码149 len=91 |
| 18.2s | S2C | 属性更新0x10[V#0x2] LeftTime=601540 |
| 18.5s | S2C | op31(sub1) 0x7000152 |
| 18.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 18.6s | C2S | m15 移动上报 自动 state=5 pos=(920.3,48.7,998.8) ×2 |
| 19.5s | S2C | op28(子码47) len=226 ×2 |
| 19.6s | C2S | m15 移动上报 手动 state=7 pos=(920.3,48.7,998.8) |
| 19.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 19.8s | S2C | 对象同步0x21[0x700ca8c] 码0 len=47 |
| 19.8s | S2C | 属性更新0x10[O#0x150] State= |
| 19.8s | C2S | m15 移动上报 手动 state=7 pos=(921.2,48.7,998.8) |
| 20.0s | S2C | op2C len=131 2c47820000000a010000000000e006004b010007 |
| 20.0s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 20.7s | C2S | m15 移动上报 自动 state=4 pos=(925.0,48.7,998.8) |
| 20.9s | S2C | 对象同步0x21[0x700ca8c] 码134 len=47 |
| 21.0s | C2S | m15 移动上报 手动 state=7 pos=(925.0,48.7,998.8) |
| 21.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 21.1–21.5s | C2S | m15 移动上报 自动 state=4 pos=(924.6,48.7,998.8) ×2 |
| 21.3s | S2C | 对象同步0x21[0x700ca8c] 码150 len=91 |
| 21.6s | C2S | m211 使用技能(出招) CS_hs_kfkj03 10值 [924.554, 48.748, 998.828, -1.584] |
| 21.7s | S2C | op06 len=1 06 |
| 21.7s | S2C | 对象同步0x21[0x700ca8c] 码121 len=47 |
| 21.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 21.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj03 |
| 21.7s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +94行 |
| 21.7s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x700ca8c] ×4 |
| 21.7s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 21.7s | S2C | 记录写格0x13[cooldown_rec#109 行92 列1 =16800768 0x700ca8c] ×2 |
| 21.7s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 21.7s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10247 |
| 21.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantBeSkillLocked=1 |
| 21.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantBeSkillMustLocked=1 |
| 21.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo6=1A14,700CA8C,3B6FB8,1,C38EA70AD20,0 |
| 21.7s | S2C | S2C m169 buff添加 ["buf_CS_em_lbc06_1", obj(0x700ca8c,0x3b6fb8)] |
| 21.7s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_hs_kfkj03", 0, 0, 0.0] |
| 21.9s | S2C | 属性更新0x10[V#0x28] CurTargetCount=2 |
| 21.9s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 21.9s | S2C | 属性更新0x10[O#0x3b6fb8] LastWGAttackNpc=monm031302 |
| 21.9s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 21.9s | S2C | 属性更新0x10[O#0x3b0a93] LogicState=1 |
| 21.9s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 21.9s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 21.9s | S2C | 属性更新0x10[O#0x3b0a8e] LogicState=1 |
| 21.9s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +1行 |
| 21.9s | S2C | S2C m1 系统提示 [2, "8066"] |
| 21.9s | S2C | S2C m24 通用事件[0x3b0a9307020e3a] 伤害飘字 [363, 0, 0, "CS_hs_kfkj03", "SkillNormal", "monm031302"] |
| 21.9s | S2C | S2C m18 战斗伤害帧[0x3b0a9307020e3a] ["hurt_half", 0, obj(0x700ca8c,0x3b6fb8)] |
| 21.9s | S2C | S2C m169 buff添加 ["buf_CS_hs_kfkj03", obj(0x7020e3a,0x3b0a93)] |
| 21.9s | S2C | S2C m24 通用事件[0x3b0a9307020e3a] 动作开始 ["kfkj_hurt_01_self", obj(0x700ca8c,0x3b6fb8)] |
| 21.9s | S2C | 属性更新0x10[O#0x3b6fb8] FP=4 |
| 21.9s | S2C | 属性更新0x10[O#0x3b6fb8] SP=4 |
| 21.9s | S2C | S2C m24 通用事件[0x3b0a8e07033e09] 伤害飘字 [363, 0, 0, "CS_hs_kfkj03", "SkillNormal", "monm031302"] |
| 21.9s | S2C | S2C m18 战斗伤害帧[0x3b0a8e07033e09] ["hurt_half", 0, obj(0x700ca8c,0x3b6fb8)] |
| 21.9s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [2] |
| 21.9s | S2C | S2C m169 buff添加 ["buf_CS_hs_kfkj03", obj(0x7033e09,0x3b0a8e)] |
| 21.9s | S2C | S2C m24 通用事件[0x3b0a8e07033e09] 动作开始 ["kfkj_hurt_01_self", obj(0x700ca8c,0x3b6fb8)] |
| 21.9s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj03", obj(0x7020e3a,0x3b0a93), obj(0x7033e09,0x3b0a8e)] |
| 22.0s | S2C | 对象同步0x21[0x7020e3a] 码217 len=179 |
| 22.2s | S2C | 对象同步0x21[0x7020e3a] 码105 len=47 |
| 22.2s | S2C | op79 len=41 795cac36f57e9736f7508536f756fc509c3cc85e |
| 22.2s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj03", obj(0x7020e3a,0x3b0a93), obj(0x7033e09,0x3b0a8e)] |
| 22.2s | S2C | 多属性0x2D(非标子码02) len=151 2d0200093e03078e0a3b000400a0018c |
| 22.2s | S2C | 属性更新0x10[O#0x3b6fb8] FP=6 |
| 22.2s | S2C | 属性更新0x10[O#0x3b6fb8] SP=6 |
| 22.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 22.2s | S2C | S2C m24 通用事件[0x3b0a9307020e3a] 伤害飘字 [315, 314, 0, "CS_hs_kfkj03", "SkillNormal", "monm031302"] |
| 22.2s | S2C | 记录加行0x11[JhBoss_Accept_Rec#22 0x700ca8c/0x3b6fb8] +0行 |
| 22.2s | S2C | S2C m446 区域boss进度 [1, 146, "quyuboss_emei2", 20, 160, "ui_jh_001"] |
| 22.2s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 22.2s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 22.2s | S2C | S2C m169 buff移除 ["buf_CS_hs_kfkj03", obj(0x7020e3a,0x3b0a93)] |
| 22.2s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 22.2s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 25, 160, "ui_jh_001"] |
| 22.2s | S2C | S2C m24 通用事件[0x3b0a9307020e3a] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031302", "徐傠囃辭睼 "] |
| 22.2s | S2C | S2C m18 战斗伤害帧[0x3b0a9307020e3a] ["hurt_half", 0, obj(0x700ca8c,0x3b6fb8)] |
| 22.2s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [3] |
| 22.2s | S2C | S2C m24 通用事件[0x3b0a9307020e3a] 动作开始 ["kfkj_hurt_01_self", obj(0x700ca8c,0x3b6fb8)] |
| 22.2s | S2C | 属性更新0x10[O#0x3b6fb8] FP=8 |
| 22.2s | S2C | 属性更新0x10[O#0x3b6fb8] SP=8 |
| 22.2s | S2C | 属性更新0x10[O#0x3b0a8e] InParry=1 |
| 22.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 22.2s | S2C | S2C m24 通用事件[0x3b0a8e07033e09] 伤害飘字 [315, 314, 0, "CS_hs_kfkj03", "SkillNormal", "monm031302"] |
| 22.2s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 22.2s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 22.2s | S2C | S2C m169 buff移除 ["buf_CS_hs_kfkj03", obj(0x7033e09,0x3b0a8e)] |
| 22.2s | S2C | 属性更新0x10[O#0x3b0a8e] InParry=0 |
| 22.2s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x700ca8c] |
| 22.2s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 30, 160, "ui_jh_001"] |
| 22.2s | S2C | S2C m24 通用事件[0x3b0a8e07033e09] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031302", "徐傠囃辭睼 "] |
| 22.2s | S2C | S2C m18 战斗伤害帧[0x3b0a8e07033e09] ["hurt_half", 0, obj(0x700ca8c,0x3b6fb8)] |
| 22.2s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [4] |
| 22.2s | S2C | custom0x1E(无法解TVar) |
| 22.4s | S2C | 多属性0x2D(非标子码02) len=136 2d0200093e03078e0a3b000800d70100 |
| 22.4s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 22.6s | S2C | op28(子码47) len=231 |
| 22.6s | S2C | S2C m169 buff移除 ["buf_CS_hs_kfkj04", obj(0x700ca8c,0x3b6fb8)] |
| 22.6s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =16801024 0x700ca8c] ×3 |
| 22.6s | S2C | 记录写格0x13[cooldown_rec#109 行92 列1 =16800768 0x700ca8c] ×2 |
| 22.6s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 22.8s | S2C | S2C m169 buff移除 ["buf_CS_hs_kfkj04_01", obj(0x700ca8c,0x3b6fb8)] |
| 22.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo5= |
| 23.3s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 23.3s | S2C | S2C m169 buff移除 ["buf_CS_em_lbc06_1", obj(0x700ca8c,0x3b6fb8)] |
| 23.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo6= |
| 23.3s | S2C | 属性更新0x10[O#0x3b6fb8] CantBeSkillLocked=0 |
| 23.3s | S2C | 属性更新0x10[O#0x3b6fb8] CantBeSkillMustLocked=0 |
| 24.2s | S2C | 属性更新0x10[O#0x3b0a93] LogicState=0 |
| 24.2s | S2C | 属性更新0x10[O#0x3b0a8e] LogicState=0 |
| 24.3s | S2C | S2C m566 令牌应答串 ["Md368R8mHLS8/iK7/auVW1G3WXBL5Lptfp9ZAvg5"] |
| 24.4s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 24.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 24.6s | C2S | m15 移动上报 自动 state=5 pos=(924.6,48.7,998.8) ×2 |
| 26.2s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 26.4s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 26.4s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10347 |
| 26.5s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 26.6s | S2C | op28(子码47) len=231 |
| 27.9s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 28.7s | S2C | S2C m1 系统提示 [3, "15917", "徐元迟 ", "ini\scene\clone047", 2000] |
| 29.4s | C2S | m15 移动上报 手动 state=7 pos=(924.6,48.7,998.8) |
| 29.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 29.7s | C2S | m15 移动上报 手动 state=7 pos=(925.5,48.7,998.8) |
| 29.7s | S2C | 对象同步0x21[0x700ca8c] 码105 len=47 |
| 29.9–30.3s | S2C | 对象同步0x21[0x700ca8c] 码233 len=47 ×2 |
| 30.0s | C2S | m15 移动上报 自动 state=4 pos=(927.0,48.7,998.9) |
| 30.1s | S2C | 对象同步0x21[0x700ca8c] 码217 len=47 |
| 30.2s | C2S | m15 移动上报 手动 state=7 pos=(927.0,48.7,998.9) |
| 30.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 30.3s | C2S | m15 移动上报 自动 state=4 pos=(926.5,48.7,998.9) |
| 30.5–30.8s | S2C | 对象同步0x21[0x700ca8c] 码25 len=47 ×2 |
| 30.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 30.6s | C2S | m15 移动上报 自动 state=5 pos=(926.5,48.7,998.9) ×2 |
| 30.6s | C2S | m15 移动上报 自动 state=4 pos=(926.5,48.7,998.9) |
| 31.1s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 31.1s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 34.1s | C2S | m54 格挡状态(值6902349270) |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 34.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 34.2s | C2S | m15 移动上报 手动 state=4 pos=(926.5,48.7,998.9) |
| 34.5–37.9s | S2C | 对象同步0x21[0x700ca8c] 码25 len=47 ×7 |
| 34.5s | C2S | m54 格挡状态(值6678309295) |
| 34.5s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 34.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 34.5s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 34.5s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 34.5s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 34.5s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 34.5s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 34.5s | C2S | m54 格挡状态(值6678309295) ×2 |
| 34.7s | C2S | m15 移动上报 自动 state=4 pos=(926.5,48.7,998.9) |
| 35.2s | C2S | m54 格挡状态(值6902349270) |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 35.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 35.2s | C2S | m15 移动上报 手动 state=4 pos=(926.5,48.7,998.9) |
| 35.4s | C2S | m54 格挡状态(值6678309295) |
| 35.4s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 35.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 35.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 35.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 35.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 35.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 35.4s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 35.4s | C2S | m54 格挡状态(值6678309295) ×3 |
| 35.5s | C2S | m82 ? [3] |
| 35.7s | C2S | m15 移动上报 自动 state=4 pos=(926.5,48.7,998.9) |
| 36.2s | C2S | m54 格挡状态(值6902349270) |
| 36.2s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 36.2s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 36.2s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 36.2s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 36.2s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 36.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 36.2s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 36.2s | C2S | m15 移动上报 手动 state=4 pos=(926.5,48.7,998.9) |
| 36.5s | C2S | m54 格挡状态(值6678309295) |
| 36.6s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 36.6s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 36.6s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 36.6s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 36.6s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 36.6s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 36.6s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 36.6s | C2S | m54 格挡状态(值6678309295) ×2 |
| 36.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 36.6s | C2S | m15 移动上报 自动 state=5 pos=(926.5,48.7,998.9) ×2 |
| 36.7s | C2S | m15 移动上报 自动 state=4 pos=(926.5,48.7,998.9) |
| 36.9s | S2C | op06 len=1 06 |
| 37.0s | S2C | 对象同步0x21[0x7000152] 码177 len=135 |
| 37.1s | C2S | m54 格挡状态(值6902349270) |
| 37.1s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 37.1s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 37.1s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 37.1s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 37.1s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 37.1s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 37.1s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 37.3–37.8s | C2S | m15 移动上报 手动 state=4 pos=(926.5,48.7,998.9) ×2 |
| 38.0s | C2S | m54 格挡状态(值6678309295) |
| 38.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 38.0s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 38.0s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 38.0s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 38.0s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 38.0s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 38.0s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 38.0s | C2S | m54 格挡状态(值6678309295) ×2 |
| 38.1s | C2S | m15 移动上报 手动 state=7 pos=(926.5,48.7,998.9) ×2 |
| 38.1s | C2S | m54 格挡状态(值6678309295) |
| 38.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 38.3s | S2C | 对象同步0x21[0x700ca8c] 码181 len=91 |
| 38.7s | S2C | op31(sub2) 0x7009041 →0x700c97d |
| 39.0s | C2S | 对象请求 len=33 |
| 39.0s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 39.0s | C2S | m431 选择目标 [obj(0x7036540,0x3b04aa)] |
| 39.0s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [930.608, 48.763, 999.103, 1.511] |
| 39.0s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 39.0s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 39.0s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj02 |
| 39.0s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 39.0s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 39.0s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill=CS_hs_kfkj02 |
| 39.0s | S2C | S2C m707 受理族变体 [15] |
| 39.1s | S2C | S2C m386 UI开关推送 [1, 1, "ui_guild_kill"] ×2 |
| 39.1s | S2C | 属性更新0x10[O#0x15c] State=stand |
| 39.2s | C2S | m15 移动上报 自动 state=4 pos=(931.7,48.7,999.2) |
| 39.4s | S2C | 对象同步0x21[0x7000157] 码81 len=135 |
| 39.6s | C2S | m15 移动上报 手动 state=7 pos=(931.7,48.7,999.2) |
| 39.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 39.8–40.6s | S2C | 对象同步0x21[0x700ca8c] 码0 len=47 ×2 |
| 39.8s | S2C | S2C m386 UI开关推送 [1, 1, "ui_guild_kill"] ×2 |
| 39.9s | C2S | m15 移动上报 自动 state=4 pos=(932.9,48.7,999.2) |
| 40.0s | S2C | 对象同步0x21[0x700ca8c] 码23 len=47 |
| 40.0s | S2C | 属性更新0x10[O#0x157] State=stand |
| 40.2s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [932.861, 48.748, 999.239, 1.511] |
| 40.2s | S2C | 对象同步0x21[0x7000152] 码177 len=91 |
| 40.3s | C2S | m15 移动上报 自动 state=7 pos=(932.9,48.7,999.2) ×2 |
| 40.3s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 40.3s | S2C | S2C m26 技能释放 [] |
| 40.3s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill= |
| 40.3s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj02 |
| 40.3s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 40.3s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill=CS_hs_kfkj02 |
| 40.3s | S2C | S2C m707 受理族变体 [15] |
| 40.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 40.4s | C2S | m220 冲锋接近上报 [933.506, 48.748, 999.256, 1.545] |
| 40.4s | C2S | m15 移动上报 手动 state=7 pos=(933.6,48.7,999.3) |
| 40.4s | S2C | 对象同步0x21[0x700ca8c] 码205 len=91 |
| 40.4s | C2S | m220 冲锋接近上报 [933.92, 48.748, 999.278, 1.511] |
| 40.5s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 冲锋动画 [938.326, 49.746, 999.38, 1, "qg_fastrun_2h_01"] |
| 40.5s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 40.5s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 40.5s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +93行 |
| 40.5s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =23808 0x700ca8c] ×4 |
| 40.5s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 40.5s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +94行 |
| 40.5s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x700ca8c] ×4 |
| 40.5s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 40.5s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10247 |
| 40.5s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=155 |
| 40.5s | C2S | m15 移动上报 自动 state=4 pos=(934.1,48.7,999.3) |
| 40.8s | C2S | m212 冲锋/续招落地 [938.223, 48.746, 999.377, 1.549] |
| 40.9s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 40.9s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 40.9s | S2C | 位置0x1F(短len=25) |
| 40.9s | S2C | 属性更新0x10[O#0x3b6fb8] CurBreathState=0 |
| 40.9s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill= |
| 40.9s | S2C | S2C m26 技能释放 [] |
| 40.9s | S2C | 属性更新0x10[O#0x3b04aa] LogicState=1 |
| 40.9s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 40.9s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 40.9s | S2C | 位置0x1F(短len=25) |
| 40.9s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [938.223, 48.746, 999.377, 1.545, 0] |
| 40.9s | S2C | 属性更新0x10[O#0x3b6fb8] LastWGAttackNpc=monm031301 |
| 40.9s | S2C | S2C m24 通用事件[0x3b04aa07036540] 动作开始 [1, obj(0x700ca8c,0x3b6fb8)] |
| 40.9s | S2C | S2C m19 锁定伤害段[0x3b6fb80700ca8c] [obj(0x7036540,0x3b04aa), "CS_hs_kfkj02", 3000, 3000, 935.786, 48.746] |
| 40.9s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 40.9s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 40.9s | C2S | m15 移动上报 自动 state=4 pos=(938.3,48.7,999.4) |
| 41.0s | C2S | m15 移动上报 自动 state=7 pos=(938.2,48.7,999.4) ×2 |
| 41.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 41.1s | S2C | 属性更新0x10[O#0x157] State=walk |
| 41.1s | S2C | 属性更新0x10[O#0x15c] State=walk |
| 41.1s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 41.1s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 41.1s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 41.1s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 41.1s | S2C | S2C m24 通用事件[0x3b04aa07036540] 伤害飘字 [508, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 41.1s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj02", obj(0x7036540,0x3b04aa)] |
| 41.2s | S2C | op2C len=246 2c47f5000000e6010000000000800b0040650307 |
| 41.3s | S2C | 多属性0x2D[0x4aa0703](布局变体) len=35 2d010040650307aa043b000300a0018c |
| 41.5s | S2C | 属性更新0x10[O#0x3b6fb8] FP=4 |
| 41.5s | S2C | 属性更新0x10[O#0x3b6fb8] SP=4 |
| 41.5s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 41.5s | S2C | S2C m24 通用事件[0x3b04aa07036540] 伤害飘字 [508, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 41.5s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [2] |
| 41.5s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj02", obj(0x7036540,0x3b04aa)] |
| 41.6s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=165 |
| 41.8s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 动作开始 ["buff_ng_xyhxj01", obj(0x700ca8c,0x3b6fb8), obj(0x7036540,0x3b04aa)] |
| 41.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=4BF9,700CA8C,3B6FB8,5,C38EA70E820,1 |
| 41.8s | S2C | S2C m169 buff添加 ["buf_ng_jz_006_01", obj(0x700ca8c,0x3b6fb8)] |
| 41.8s | S2C | S2C m169 buff刷新? ["buf_ng_jz_006_01", obj(0x700ca8c,0x3b6fb8)] |
| 41.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=4BF9,700CA8C,3B6FB8,5,C38EA70E820,5 |
| 41.8s | S2C | 属性更新0x10[O#0x3b6fb8] FP=6 |
| 41.8s | S2C | 属性更新0x10[O#0x3b6fb8] SP=6 |
| 41.8s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 41.8s | S2C | S2C m24 通用事件[0x3b04aa07036540] 伤害飘字 [508, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 41.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=4BF9,700CA8C,3B6FB8,5,C38EA70E821,4 |
| 41.8s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [3] |
| 41.8s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj02", obj(0x7036540,0x3b04aa)] |
| 41.9s | S2C | 属性更新0x10[O#0x3b6fb8] PhyDamageMul=1.4 |
| 41.9s | S2C | 属性更新0x10[O#0x3b6fb8] StiDamageMul=1.4 |
| 41.9s | S2C | 属性更新0x10[O#0x3b6fb8] JujDamageMul=1.4 |
| 41.9s | S2C | 属性更新0x10[O#0x3b6fb8] NegDamageMul=1.4 |
| 41.9s | S2C | 属性更新0x10[O#0x3b6fb8] MasDamageMul=1.4 |
| 41.9s | S2C | 属性更新0x10[O#0x3b6fb8] UnderWaterModifySpeed=1.0 ×2 |
| 42.0s | S2C | 多属性0x2D[0x4aa0703](布局变体) len=25 2d010040650307aa043b000200d70136 |
| 42.2s | S2C | 属性更新0x10[O#0x3b6fb8] FP=8 |
| 42.2s | S2C | 属性更新0x10[O#0x3b6fb8] SP=8 |
| 42.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 42.2s | S2C | S2C m24 通用事件[0x3b04aa07036540] 伤害飘字 [674, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 42.2s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=4BF9,700CA8C,3B6FB8,5,C38EA70E9A1,3 |
| 42.2s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [4] |
| 42.2s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj02", obj(0x7036540,0x3b04aa)] |
| 42.2s | S2C | 多属性0x2D[0x4aa0703](布局变体) len=25 2d010040650307aa043b000200d7011f |
| 42.3s | S2C | 属性更新0x10[O#0x3b6fb8] PhyDamageMul=1.3 |
| 42.3s | S2C | 属性更新0x10[O#0x3b6fb8] StiDamageMul=1.3 |
| 42.3s | S2C | 属性更新0x10[O#0x3b6fb8] JujDamageMul=1.3 |
| 42.3s | S2C | 属性更新0x10[O#0x3b6fb8] NegDamageMul=1.3 |
| 42.3s | S2C | 属性更新0x10[O#0x3b6fb8] MasDamageMul=1.3 |
| 42.3s | S2C | 属性更新0x10[O#0x3b6fb8] UnderWaterModifySpeed=1.0 ×2 |
| 42.4s | S2C | 多属性0x2D[0x4aa0703](布局变体) len=25 2d010040650307aa043b000200d70101 |
| 42.5s | S2C | 属性更新0x10[O#0x3b6fb8] FP=10 |
| 42.5s | S2C | 属性更新0x10[O#0x3b6fb8] SP=10 |
| 42.5s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 42.5s | S2C | S2C m24 通用事件[0x3b04aa07036540] 伤害飘字 [34, 599, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 42.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=4BF9,700CA8C,3B6FB8,5,C38EA70EAF0,2 |
| 42.5s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 42.5s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 42.5s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 42.5s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 37, 160, "ui_jh_001"] |
| 42.5s | S2C | S2C m24 通用事件[0x3b04aa07036540] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031301", "徐傠囃辭睼 "] |
| 42.5s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [5] |
| 42.5s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj02", obj(0x7036540,0x3b04aa)] |
| 42.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 42.6s | S2C | 属性更新0x10[O#0x3b6fb8] PhyDamageMul=1.2 |
| 42.6s | S2C | 属性更新0x10[O#0x3b6fb8] StiDamageMul=1.2 |
| 42.6s | S2C | 属性更新0x10[O#0x3b6fb8] JujDamageMul=1.2 |
| 42.6s | S2C | 属性更新0x10[O#0x3b6fb8] NegDamageMul=1.2 |
| 42.6s | S2C | 属性更新0x10[O#0x3b6fb8] MasDamageMul=1.2 |
| 42.6s | S2C | 属性更新0x10[O#0x3b6fb8] UnderWaterModifySpeed=1.0 ×2 |
| 42.6s | C2S | m15 移动上报 自动 state=5 pos=(939.7,48.8,999.4) ×2 |
| 42.8s | S2C | 多属性0x2D[0x4aa0703](布局变体) len=59 2d010040650307aa043b000600d70100 |
| 43.1s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 43.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 43.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 43.8s | S2C | op31(sub1) 0x7000152 |
| 43.8s | C2S | m990 ? [obj(0x7036540,0x3b04aa), 938.768, 48.756, 999.391] |
| 43.9s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 43.9s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 43.9s | C2S | m914 招式结束异步位置 [936.16, 48.746, 999.325, 1.545, 12, 1] |
| 43.9s | S2C | 对象同步0x21[0x7036540] 码205 len=91 |
| 43.9s | S2C | 位置0x1F(短len=25) |
| 44.1s | S2C | 对象同步0x21[0x700ca8c] 码56 len=91 |
| 44.5s | S2C | 属性更新0x10[O#0x3b04aa] LogicState=0 |
| 44.6s | C2S | m15 移动上报 手动 state=7 pos=(936.2,48.7,999.3) |
| 44.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 44.8s | S2C | 对象同步0x21[0x700ca8c] 码156 len=47 |
| 44.8s | C2S | m15 移动上报 手动 state=7 pos=(935.1,48.7,999.3) |
| 45.0s | S2C | 对象同步0x21[0x700ca8c] 码0 len=47 |
| 45.0s | S2C | 属性更新0x10[O#0x150] State=stand ×2 |
| 45.1s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 45.1s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10347 |
| 45.2s | S2C | 对象同步0x21[0x700014b] 码96 len=223 |
| 45.2s | S2C | 多属性0x2D[0x1500700](布局变体) len=23 2d01004b010007500100000100540004 |
| 45.6s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 45.6s | C2S | m15 移动上报 自动 state=4 pos=(931.5,48.7,999.0) |
| 45.7s | S2C | S2C m386 UI开关推送 [1, 1, "ui_guild_kill"] ×2 |
| 45.7s | C2S | m15 移动上报 手动 state=7 pos=(931.5,48.7,999.0) |
| 45.8s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 45.8s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 45.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 45.9s | S2C | 对象同步0x21[0x700ca8c] 码165 len=91 |
| 45.9s | S2C | S2C m386 UI开关推送 [1, 1, "ui_guild_kill"] ×2 |
| 45.9–46.1s | C2S | m15 移动上报 自动 state=4 pos=(932.1,48.7,999.1) ×2 |
| 46.0–46.3s | S2C | 对象同步0x21[0x700ca8c] 码110 len=47 ×2 |
| 46.5s | S2C | op0E len=9 0e40650307aa043b00 |
| 47.1s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 47.7s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 47.8s | S2C | op31(sub1) 0x7000157 |
| 48.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 48.7s | C2S | m15 移动上报 自动 state=5 pos=(932.1,48.7,999.1) ×2 |
| 49.6s | C2S | m15 移动上报 ?2000 state=7 pos=(932.1,48.7,999.1) |
| 49.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 49.8–50.1s | S2C | 对象同步0x21[0x700ca8c] 码0 len=47 ×3 |
| 49.8s | C2S | m15 移动上报 手动 state=7 pos=(932.1,48.7,998.4) |
| 49.8s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 49.8s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 49.9s | C2S | m15 移动上报 手动 state=7 pos=(932.4,48.7,998.2) |
| 50.5s | C2S | m15 移动上报 自动 state=4 pos=(935.4,48.7,998.3) |
| 50.8s | S2C | 对象同步0x21[0x700ca8c] 码11 len=47 |
| 51.0s | C2S | 对象请求 len=33 |
| 51.0s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 51.0s | C2S | m431 选择目标 [obj(0x70067ff,0x3b04af)] |
| 51.0s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [935.407, 48.748, 998.343, 1.511] |
| 51.0s | C2S | m15 移动上报 自动 state=7 pos=(935.4,48.7,998.3) ×2 |
| 51.1s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 51.1s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 51.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj02 |
| 51.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 51.1s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 51.1s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill=CS_hs_kfkj02 |
| 51.1s | S2C | S2C m707 受理族变体 [15] |
| 51.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 51.2s | C2S | m220 冲锋接近上报 [936.205, 48.748, 998.289, 1.638] ×2 |
| 51.2s | S2C | 对象同步0x21[0x700ca8c] 码199 len=91 |
| 51.3s | C2S | m212 冲锋/续招落地 [937.077, 48.748, 998.23, 1.638] |
| 51.4s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill= |
| 51.4s | S2C | S2C m26 技能释放 [] |
| 51.4s | S2C | 属性更新0x10[O#0x3b04af] LogicState=1 |
| 51.4s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 51.4s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 51.4s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +93行 |
| 51.4s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =23808 0x700ca8c] ×4 |
| 51.4s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 51.4s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +94行 |
| 51.4s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x700ca8c] ×4 |
| 51.4s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 51.4s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10247 |
| 51.4s | S2C | 位置0x1F(短len=25) |
| 51.4s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [936.39, 48.746, 998.276, 1.638, 0] |
| 51.4s | S2C | S2C m24 通用事件[0x3b04af070067ff] 动作开始 [1, obj(0x700ca8c,0x3b6fb8)] |
| 51.4s | S2C | S2C m19 锁定伤害段[0x3b6fb80700ca8c] [obj(0x70067ff,0x3b04af), "CS_hs_kfkj02", 3000, 3000, 935.778, 48.746] |
| 51.4s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 51.4s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 51.4s | C2S | m15 移动上报 自动 state=7 pos=(936.4,48.7,998.3) |
| 51.4s | S2C | 对象同步0x21[0x70067ff] 码199 len=91 |
| 51.4s | C2S | m212 冲锋/续招落地 [937.381, 48.756, 998.21, 1.638] |
| 51.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 51.5s | C2S | m15 移动上报 自动 state=7 pos=(938.3,48.8,998.1) |
| 51.5s | C2S | m915 命中确认上报 [1, obj(0x70067ff,0x3b04af), 935.204, 48.756, 998.356, 1.442] |
| 51.5s | C2S | m212 冲锋/续招落地 [938.464, 48.756, 998.137, 1.638] |
| 51.6s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 51.6s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 51.6s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 51.6s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 51.6s | S2C | S2C m24 通用事件[0x3b04af070067ff] 伤害飘字 [593, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 51.6s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=4BF9,700CA8C,3B6FB8,5,C38EA70EA96,1 |
| 51.6s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj02", obj(0x70067ff,0x3b04af)] |
| 51.6s | C2S | m212 冲锋/续招落地 [939.247, 48.756, 998.084, 1.638] |
| 51.7s | S2C | 对象同步0x21[0x700ca8c] 码113 len=135 |
| 51.7s | S2C | 多属性0x2D[0x4af0700](布局变体) len=23 2d0100ff670007af043b000100a0018c |
| 51.7s | C2S | m212 冲锋/续招落地 [939.291, 48.756, 998.081, 1.638] |
| 51.7s | S2C | 属性更新0x10[O#0x3b6fb8] PhyDamageMul=1.1 |
| 51.7s | S2C | 属性更新0x10[O#0x3b6fb8] StiDamageMul=1.1 |
| 51.7s | S2C | 属性更新0x10[O#0x3b6fb8] JujDamageMul=1.1 |
| 51.7s | S2C | 属性更新0x10[O#0x3b6fb8] NegDamageMul=1.1 |
| 51.7s | S2C | 属性更新0x10[O#0x3b6fb8] MasDamageMul=1.1 |
| 51.7s | S2C | 属性更新0x10[O#0x3b6fb8] UnderWaterModifySpeed=1.0 ×2 |
| 51.8s | C2S | m212 冲锋/续招落地 [939.453, 48.756, 998.07, 1.638] |
| 51.9s | S2C | 多属性0x2D[0x4af0700](布局变体) len=25 2d0100ff670007af043b000200d70149 |
| 51.9s | C2S | m212 冲锋/续招落地 [941.498, 48.756, 997.932, 1.638] ×2 |
| 52.1s | S2C | 属性更新0x10[O#0x3b6fb8] FP=4 |
| 52.1s | S2C | 属性更新0x10[O#0x3b6fb8] SP=4 |
| 52.1s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 52.1s | S2C | S2C m24 通用事件[0x3b04af070067ff] 伤害飘字 [553, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 52.1s | S2C | S2C m169 buff移除 ["buf_ng_jz_006_01", obj(0x700ca8c,0x3b6fb8)] |
| 52.1s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 52.1s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [2] |
| 52.1s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj02", obj(0x70067ff,0x3b04af)] |
| 52.1s | C2S | m212 冲锋/续招落地 [941.498, 48.756, 997.932, 1.638] |
| 52.2s | S2C | op06 len=1 06 |
| 52.2s | S2C | 属性更新0x10[O#0x3b6fb8] PhyDamageMul=1.0 |
| 52.2s | S2C | 属性更新0x10[O#0x3b6fb8] StiDamageMul=1.0 |
| 52.2s | S2C | 属性更新0x10[O#0x3b6fb8] JujDamageMul=1.0 |
| 52.2s | S2C | 属性更新0x10[O#0x3b6fb8] NegDamageMul=1.0 |
| 52.2s | S2C | 属性更新0x10[O#0x3b6fb8] MasDamageMul=1.0 |
| 52.2s | S2C | 属性更新0x10[O#0x3b6fb8] UnderWaterModifySpeed=1.0 ×2 |
| 52.2s | C2S | m212 冲锋/续招落地 [941.455, 48.756, 997.935, 1.638] |
| 52.3s | S2C | 多属性0x2D[0x4af0700](布局变体) len=25 2d0100ff670007af043b000200d70130 |
| 52.3s | C2S | m212 冲锋/续招落地 [941.116, 48.756, 997.958, 1.638] |
| 52.4s | S2C | 属性更新0x10[O#0x3b6fb8] FP=6 |
| 52.4s | S2C | 属性更新0x10[O#0x3b6fb8] SP=6 |
| 52.4s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 52.4s | S2C | S2C m24 通用事件[0x3b04af070067ff] 伤害飘字 [925, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 52.4s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [3] |
| 52.4s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj02", obj(0x70067ff,0x3b04af)] |
| 52.5–52.6s | C2S | m212 冲锋/续招落地 [940.815, 48.756, 997.978, 1.638] ×3 |
| 52.7s | S2C | 属性更新0x10[O#0x3b6fb8] FP=8 |
| 52.7s | S2C | 属性更新0x10[O#0x3b6fb8] SP=8 |
| 52.7s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 52.7s | S2C | S2C m24 通用事件[0x3b04af070067ff] 伤害飘字 [161, 350, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 52.7s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 52.7s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 52.7s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 52.7s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 44, 160, "ui_jh_001"] |
| 52.7s | S2C | S2C m24 通用事件[0x3b04af070067ff] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031301", "徐傠囃辭睼 "] |
| 52.7s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [4] |
| 52.7s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj02", obj(0x70067ff,0x3b04af)] |
| 52.7s | C2S | m212 冲锋/续招落地 [939.945, 48.756, 998.037, 1.638] |
| 52.8s | S2C | 对象同步0x21[0x70067ff] 码124 len=47 |
| 52.8s | S2C | 多属性0x2D[0x4af0700](布局变体) len=25 2d0100ff670007af043b000200d70107 |
| 52.8s | C2S | m212 冲锋/续招落地 [940.542, 48.756, 997.997, 1.638] |
| 52.9s | S2C | 多属性0x2D[0x4af0700](布局变体) len=59 2d0100ff670007af043b000600d70100 |
| 53.1s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 53.8s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 53.8s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 54.4s | C2S | m990 ? [obj(0x70067ff,0x3b04af), 938.799, 48.756, 998.114] |
| 54.4s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 54.4s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 54.4s | C2S | m914 招式结束异步位置 [936.151, 48.746, 998.293, 1.638, 12, 2] |
| 54.4s | S2C | 位置0x1F(短len=25) |
| 54.5s | S2C | 对象同步0x21[0x700ca8c] 码167 len=91 |
| 54.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 54.6s | C2S | m15 移动上报 自动 state=5 pos=(936.2,48.7,998.3) ×2 |
| 54.7s | S2C | 属性更新0x10[O#0x3b04af] LogicState=0 |
| 55.5s | C2S | m82 ? [3] |
| 56.0s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 56.0s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10347 |
| 56.7s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 56.9s | S2C | op0E len=9 0eff670007af043b00 |
| 56.9s | C2S | m440 ? [] |
| 56.9s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 56.9s | C2S | m431 选择目标 [0] |
| 57.0s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 57.6s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 59.8s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 59.8s | C2S | m15 移动上报 手动 state=7 pos=(936.2,48.7,998.3) |
| 59.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 59.9s | S2C | 对象同步0x21[0x700ca8c] 码11 len=47 |
| 60.0s | C2S | m15 移动上报 手动 state=7 pos=(935.1,48.7,998.2) |
| 60.1s | S2C | 对象同步0x21[0x700ca8c] 码69 len=47 |
| 60.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 60.7s | C2S | m15 移动上报 手动 state=5 pos=(932.4,48.8,998.1) ×2 |
| 60.7s | S2C | S2C m386 UI开关推送 [1, 1, "ui_guild_kill"] ×2 |
| 60.7s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 60.7s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 60.9s | S2C | S2C m386 UI开关推送 [1, 1, "ui_guild_kill"] ×2 |
| 61.3s | C2S | m15 移动上报 自动 state=4 pos=(929.4,48.7,997.9) |
| 61.4s | S2C | 对象同步0x21[0x700ca8c] 码188 len=47 |
| 62.1s | C2S | m15 移动上报 手动 state=7 pos=(929.4,48.7,997.9) |
| 62.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 62.2s | S2C | 对象同步0x21[0x700ca8c] 码88 len=47 |
| 62.2–62.9s | C2S | m15 移动上报 自动 state=4 pos=(930.1,48.7,997.9) ×3 |
| 62.4–65.1s | S2C | 对象同步0x21[0x700ca8c] 码129 len=91 ×3 |
| 64.8s | C2S | m54 格挡状态(值6902349270) |
| 64.8s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 64.8s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 64.8s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 64.8s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 64.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 64.8s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 64.8s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 64.9s | C2S | m54 格挡状态(值6678309295) |
| 64.9s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 64.9s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 64.9s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 64.9s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 64.9s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 64.9s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 64.9s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 64.9s | C2S | m54 格挡状态(值6678309295) |
| 64.9s | C2S | m15 移动上报 自动 state=4 pos=(930.1,48.7,997.9) |
| 65.0s | C2S | m54 格挡状态(值6678309295) |
| 66.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 1794117318.1s | C2S | 6001 大厅心跳 04 00 00 00 10 EE 11 11 → 01 FF 00 00 ×7 |

## c.pcapng — 49.4.14.150:7074（与 a 同客户端口 60179，重连会话）

场景：重连全量快照 + UI/系统功能遍历探测（属性商城等表单事件）。事件 2853 条 → 合并为 1202 段。抓包起点 2026-11-08 13:39:21（本地墙钟）

| 时间 | 方向 | 交互 |
|---|---|---|
| 0.0s | C2S | op04 len=84 04905fa050c356ad8f7c7700000000000000000000000000 |
| 0.1s | S2C | op32 len=1026 3231383a23494e4e455237453444354336353232 |
| 0.1s | PAIR | C2S ping(0x18) → S2C 0x38 回显 |
| 0.1s | S2C | 属性表0x09 |
| 0.1s | S2C | op0A len=10429 0abe014e6f6e6500000073656c665f686f6d655f |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] PhyParryDamRes=50.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] MagicParryDamRes=50.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] HPHeartSpeed=5000 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] MPHeartSpeed=5000 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] GRushRange=5.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] ARushRange=8.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] WRushRange=8.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] UnderWaterSpeed=10.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] UnderWaterModifySpeed=1.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] WalkSpeed=1.28 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] RunSpeed=3.9 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] ActionSpeed=1.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] MoveSpeed=3.9 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] JumpSpeed=5.5 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] Gravity=9.8 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] DropHeightPub=10.0 |
| 0.1s | S2C | 建视图0x15 view=165(?) cap=6 |
| 0.1s | S2C | S2C m386 UI开关推送 [1, 18, "ui_leitai_info0"] ×14 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] DrunkLevel=2 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +1行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +2行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +3行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +4行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +5行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +6行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +7行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +8行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +9行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +10行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +11行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +12行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +13行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +14行 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +15行 |
| 0.1s | S2C | S2C m458 ? [3, 11] |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] FightData=114859754505 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] ClimbSpeed=7.5 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] DriftSpeed=7.5 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] AdvancedJumpSpeed=23.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] AdvancedGlideSpeed=5.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] AdvancedFastfallSpeed=30.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] AdvancedFastfallDist=40.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] UnderSandSpeed=5.8 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] WaterRushSpeed=25.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] WaterRushDist=6.5 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] DriftJumpSpeed=6.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] SndJumpSpeed=6.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] ThdJumpSpeed=6.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] AirRushSpeed=15.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] AirRushDist=12.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] LandRushSpeed=25.0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] LandRushDist=6.5 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] MaxQingGongPoint=150 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] MaxGJQingGongPoint=150 |
| 0.1s | S2C | 记录加行0x11[BufferGroupPublicRec#59 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] BufferInfo1=6747,7020FA4,3B3CC3,1,0,0 |
| 0.1s | S2C | S2C m169 buff添加 ["buff_group_flag_101", obj(0x7020fa4,0x3b3cc3)] |
| 0.1s | S2C | 记录加行0x11[BufferGroupPublicRec#59 0x7020fa4/0x3b3cc3] +1行 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] ActionSet=2h |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] Hat=obj\char\b_festival_1_62\b_helmet_1_62 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] Cloth=obj\char\b_festival_1_62\b_cloth_1_62 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] Pants=obj\char\b_festival_1_62\b_pants_1_62 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] Shoes=obj\char\b_festival_1_62\b_shoes_1_62 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] Weapon=sword_card_004 |
| 0.1s | S2C | S2C m683 ? [1] |
| 0.1s | S2C | 建视图0x15 view=128 cap=5 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] GroupID=-1 |
| 0.1s | S2C | S2C m1 系统提示 [0, "sys_donghai_prize_switch"] |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] VisualPlayers=80 |
| 0.1s | S2C | S2C m705 ? ["e$"] |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] MaxFaculty=2000000000 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] LevelTitle=title025 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] CurLevel=8 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] CurFillValue=353148 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] TotalFillValue=555700 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] Type=2 |
| 0.1s | S2C | S2C m1 系统提示 [0, "2543"] |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] InteractStatus=-1 |
| 0.1s | S2C | S2C m395 ? [4] |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] LastChuShiTime=0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] ShiTu_WeekIncFacultyDate=0 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] LastChuShiTime=11 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] ShiTu_WeekIncFacultyDate=44 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] TotalCardPrize=494 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] AccountIDPlaysnail=415016589 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] OffLineTime=None |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] LastOffLineTime=None |
| 0.1s | S2C | 记录加行0x11[InteractTraceRec#101 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] InSuperTransMode=1 |
| 0.1s | S2C | S2C m366 ? ["171.222.254.180"] |
| 0.1s | S2C | S2C m623 ? [0, 0, 3] |
| 0.1s | S2C | S2C m655 ? [7100037, 7100037, "侠骨丹心 ", 7100037, "lizy86", 415016589] |
| 0.1s | S2C | custom0x27(无法解TVar) ×2 |
| 0.1s | S2C | S2C m414 ? ["gamestep", 6, 1] |
| 0.1s | S2C | custom0x27(无法解TVar) |
| 0.1s | S2C | 属性更新0x10[O#0x3b3cc3] TodayCapitalLeave1=360000 ×2 |
| 0.1s | S2C | op0B len=83 0b21150000a40f0207c33c3b000400b200080000 |
| 0.1s | S2C | op14 len=12 140200000000000000006901 |
| 0.1s | S2C | 记录加行0x11[WeatherRec#361 0x0/0x0] +0行 |
| 0.1s | S2C | op14 len=12 140200000000000000006a01 |
| 0.1s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 0.1s | S2C | op14 len=12 140200000000000000006b01 |
| 0.1s | S2C | op29(子码47) len=151 |
| 0.1s | S2C | op14 len=12 140200000000000000006d01 |
| 0.1s | S2C | op29(子码47) len=174 |
| 0.1s | S2C | op28(子码47) len=3187 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b000300 |
| 0.1s | S2C | 记录加行0x11[HomeFurnitureRec#3 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b000600 |
| 0.1s | S2C | 记录加行0x11[SsfHurtRec#6 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b000900 |
| 0.1s | S2C | 记录加行0x11[worldwar_honor_rec#9 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b000d00 |
| 0.1s | S2C | 记录加行0x11[TiGuanRec#13 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001000 |
| 0.1s | S2C | op29(子码47) len=672 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001100 |
| 0.1s | S2C | 记录加行0x11[JHTour_Record#17 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001200 |
| 0.1s | S2C | 记录加行0x11[JHFavorable_Record#18 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001300 |
| 0.1s | S2C | 记录加行0x11[JHRenown_Record#19 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001400 |
| 0.1s | S2C | op29(子码47) len=112 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001700 |
| 0.1s | S2C | 记录加行0x11[JhBoss_Round_Rec#23 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001800 |
| 0.1s | S2C | op29(子码47) len=245 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001d00 |
| 0.1s | S2C | op29(子码47) len=987 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001e00 |
| 0.1s | S2C | op29(子码47) len=736 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b001f00 |
| 0.1s | S2C | op29(子码47) len=274 ×21 |
| 0.1s | S2C | 记录加行0x11[Task_Completed#31 0x7020fa4/0x3b3cc3] +2688行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002200 |
| 0.1s | S2C | 记录加行0x11[Plot_Record#34 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002400 |
| 0.1s | S2C | op29(子码47) len=143 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002500 |
| 0.1s | S2C | 记录加行0x11[Origin_History_Rec#37 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002600 |
| 0.1s | S2C | 记录加行0x11[Origin_Top_Rec#38 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002700 |
| 0.1s | S2C | 记录加行0x11[DailyLPRec#39 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002800 |
| 0.1s | S2C | 记录加行0x11[consign_history#40 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002900 |
| 0.1s | S2C | 记录加行0x11[vip_info_rec#41 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002a00 |
| 0.1s | S2C | 记录加行0x11[DramaRec#42 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002b00 |
| 0.1s | S2C | 记录加行0x11[SchoolDramaRec#43 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002c00 |
| 0.1s | S2C | 记录加行0x11[Repute_Record#44 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002d00 |
| 0.1s | S2C | op29(子码47) len=146 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002e00 |
| 0.1s | S2C | op29(子码47) len=154 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b002f00 |
| 0.1s | S2C | op29(子码47) len=120 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b003000 |
| 0.1s | S2C | op29(子码47) len=120 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b003300 |
| 0.1s | S2C | 记录加行0x11[OpenMapRec#51 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b003600 |
| 0.1s | S2C | 记录加行0x11[ForgetSkillRec#54 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b003800 |
| 0.1s | S2C | 记录加行0x11[MapMaskRec#56 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b003900 |
| 0.1s | S2C | 记录加行0x11[ShiJiaJiangLi#57 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b003b00 |
| 0.1s | S2C | 记录加行0x11[BufferGroupPublicRec#59 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b004700 |
| 0.1s | S2C | op29(子码47) len=409 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b004900 |
| 0.1s | S2C | 记录加行0x11[AnnotateCombineRec#73 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b004a00 |
| 0.1s | S2C | op29(子码47) len=388 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b005500 |
| 0.1s | S2C | op29(子码47) len=1789 ×2 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b005700 |
| 0.1s | S2C | 记录加行0x11[rec_npc_attention#87 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b005800 |
| 0.1s | S2C | op29(子码47) len=577 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b005900 |
| 0.1s | S2C | 记录加行0x11[CharacterValueRec#89 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b005d00 |
| 0.1s | S2C | op29(子码47) len=1391 ×2 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b005f00 |
| 0.1s | S2C | op29(子码47) len=653 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b006000 |
| 0.1s | S2C | op29(子码47) len=656 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b006200 |
| 0.1s | S2C | op29(子码47) len=476 ×2 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b006500 |
| 0.1s | S2C | 记录加行0x11[InteractTraceRec#101 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b006700 |
| 0.1s | S2C | 记录加行0x11[consum_capital_limit#103 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b006800 |
| 0.1s | S2C | op29(子码47) len=733 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b006900 |
| 0.1s | S2C | op29(子码47) len=362 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b006b00 |
| 0.1s | S2C | op29(子码47) len=1209 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b006c00 |
| 0.1s | S2C | op29(子码47) len=433 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b006d00 |
| 0.1s | S2C | op29(子码47) len=1495 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b007400 |
| 0.1s | S2C | 记录加行0x11[QingGongRec#116 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b007500 |
| 0.1s | S2C | 记录加行0x11[ActiveQGSkillRec#117 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b007800 |
| 0.1s | S2C | 记录加行0x11[data_record#120 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b007900 |
| 0.1s | C2S | m301 ? [4, 1, "A036BCAD7F5D", "192.168.3.19"] |
| 0.1s | S2C | op29(子码47) len=296 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b007a00 |
| 0.1s | S2C | op29(子码47) len=139 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b007d00 |
| 0.1s | S2C | op29(子码47) len=272 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b008400 |
| 0.1s | S2C | 记录加行0x11[SafeLoginActivityRec#132 0x7020fa4/0x3b3cc3] +0行 |
| 0.1s | S2C | op14 len=12 1400a40f0207c33c3b008500 |
| 0.1s | S2C | op29(子码47) len=746 ×2 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009000 |
| 0.2s | S2C | 记录加行0x11[job_skill_share_rec#144 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009200 |
| 0.2s | S2C | 记录加行0x11[clone_rec_not_save#146 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009400 |
| 0.2s | S2C | op29(子码47) len=281 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009500 |
| 0.2s | S2C | op29(子码47) len=367 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009700 |
| 0.2s | S2C | op29(子码47) len=260 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009800 |
| 0.2s | S2C | op29(子码47) len=207 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009900 |
| 0.2s | S2C | op29(子码47) len=194 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009a00 |
| 0.2s | S2C | op29(子码47) len=244 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009b00 |
| 0.2s | S2C | op29(子码47) len=163 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009d00 |
| 0.2s | S2C | op29(子码47) len=565 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b009e00 |
| 0.2s | S2C | op29(子码47) len=1164 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00a000 |
| 0.2s | S2C | 记录加行0x11[Sub_Kapai_State#160 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00a200 |
| 0.2s | S2C | 记录加行0x11[active_jingmai_rec#162 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00a300 |
| 0.2s | S2C | 记录加行0x11[cancel_rule_rec#163 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00a600 |
| 0.2s | S2C | op29(子码47) len=120 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00a700 |
| 0.2s | S2C | op29(子码47) len=2750 ×4 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00a800 |
| 0.2s | S2C | op29(子码47) len=2984 ×2 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00ab00 |
| 0.2s | S2C | 记录加行0x11[KarmaExpRec#171 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00ac00 |
| 0.2s | S2C | 记录加行0x11[HuaShanPointRec#172 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00b200 |
| 0.2s | S2C | 记录加行0x11[binglu_rank_rec#178 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00b300 |
| 0.2s | S2C | op29(子码47) len=684 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00b400 |
| 0.2s | S2C | 记录加行0x11[blend_collect_suit#180 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00b600 |
| 0.2s | S2C | op29(子码47) len=269 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00b700 |
| 0.2s | S2C | op29(子码47) len=4098 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00b800 |
| 0.2s | S2C | 记录加行0x11[itemprotect_rec#184 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00b900 |
| 0.2s | S2C | 记录加行0x11[rec_activate_zs#185 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00ba00 |
| 0.2s | S2C | 记录加行0x11[rec_pet_show#186 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00bc00 |
| 0.2s | S2C | op29(子码47) len=4187 ×2 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00bd00 |
| 0.2s | S2C | 记录加行0x11[sweet_pet_eq_base_prop#189 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00be00 |
| 0.2s | S2C | 记录加行0x11[sweet_pet_eq_normal_prop#190 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00bf00 |
| 0.2s | S2C | op29(子码47) len=1357 ×3 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00c000 |
| 0.2s | S2C | op29(子码47) len=437 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00c100 |
| 0.2s | S2C | op29(子码47) len=213 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00c200 |
| 0.2s | S2C | op29(子码47) len=157 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00c400 |
| 0.2s | S2C | 记录加行0x11[SecondWearWuJiRec#196 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00c500 |
| 0.2s | S2C | 记录加行0x11[rec_dailytask_date#197 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00c600 |
| 0.2s | S2C | op29(子码47) len=172 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00ca00 |
| 0.2s | S2C | op29(子码47) len=460 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00cc00 |
| 0.2s | S2C | op29(子码47) len=259 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00cd00 |
| 0.2s | S2C | 记录加行0x11[BadGuyVallyAwardRec#205 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00d700 |
| 0.2s | S2C | 记录加行0x11[ActivityCollectPiecesRec#215 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00db00 |
| 0.2s | S2C | 记录加行0x11[GrowGiftRec#219 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00dc00 |
| 0.2s | S2C | 记录加行0x11[AssistantCompleteRec#220 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00df00 |
| 0.2s | S2C | 记录加行0x11[SchoolIntegralInfoRec#223 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00ed00 |
| 0.2s | S2C | 记录加行0x11[ActiveWWZJRec#237 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00ef00 |
| 0.2s | S2C | op29(子码47) len=122 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b00f200 |
| 0.2s | S2C | 记录加行0x11[ActiveJGTBRec#242 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b000101 |
| 0.2s | S2C | op29(子码47) len=588 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b000201 |
| 0.2s | S2C | 记录加行0x11[HomeWorkshopDDShowRec#258 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b000301 |
| 0.2s | S2C | op29(子码47) len=490 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b000401 |
| 0.2s | S2C | 记录加行0x11[HomeWorkshopHSDDShowRec#260 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b001a01 |
| 0.2s | S2C | 记录加行0x11[TBBattleStatisticsRecord#282 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b001e01 |
| 0.2s | S2C | 记录加行0x11[JJGGHCTransRec#286 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b001f01 |
| 0.2s | S2C | 记录加行0x11[JJGGHCTransCrossRec#287 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b002001 |
| 0.2s | S2C | 记录加行0x11[JJGGHCTransCrossSpecailRec#288 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b003701 |
| 0.2s | S2C | 记录加行0x11[SceneSchoolDayTaskRec#311 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b003801 |
| 0.2s | S2C | op29(子码47) len=171 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b003b01 |
| 0.2s | S2C | 记录加行0x11[FashionSuitRec#315 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b003d01 |
| 0.2s | S2C | 记录加行0x11[ActivityRec#317 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b003f01 |
| 0.2s | S2C | op29(子码47) len=2841 ×2 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b004201 |
| 0.2s | S2C | 记录加行0x11[CardRec#322 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b004401 |
| 0.2s | S2C | 记录加行0x11[UseCardRec#324 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b004501 |
| 0.2s | S2C | 记录加行0x11[UseSuitCardsRec#325 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b004701 |
| 0.2s | S2C | op29(子码47) len=459 ×2 |
| 0.2s | S2C | 记录加行0x11[ForceRec#327 0x7020fa4/0x3b3cc3] +256行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b004801 |
| 0.2s | S2C | 记录加行0x11[Gossip_Record#328 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b004b01 |
| 0.2s | S2C | 记录加行0x11[OffLineJob#331 0x7020fa4/0x3b3cc3] +0行 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b004c01 |
| 0.2s | S2C | op29(子码47) len=1371 |
| 0.2s | S2C | op14 len=12 1400a40f0207c33c3b004d01 |
| 0.2s | S2C | op29(子码47) len=1177 |
| 0.2s | S2C | 记录写格0x13[vip_info_rec#41 行0 列1 =33554432 0x7020fa4] |
| 0.2s | S2C | S2C m98 数值推送 [274] |
| 0.2s | S2C | 属性更新0x10[O#0x3b3cc3] SafeLoginFlag=6 |
| 0.2s | S2C | S2C m98 数值推送 [274] |
| 0.3s | C2S | m107 外观自定义保存 ["e$"] |
| 0.4s | C2S | m185 ? [17] |
| 0.4s | C2S | m421 ? [0] |
| 0.4s | C2S | m423 ? [7] |
| 0.4s | C2S | m224 ? [1] |
| 0.4s | C2S | m185 ? [17] |
| 3.1s | S2C | 属性更新0x10[O#0x3b3cc3] QingGongPoint=150 |
| 3.1–144.3s | PAIR | C2S ping(0x18) → S2C 0x38 回显 ×117 |
| 3.5s | C2S | m8 ? [21, 701, 1] ×2 |
| 3.7s | C2S | m770 九阴志步骤查询 ["", 6] |
| 3.7–4.0s | C2S | m107 外观自定义保存 ["e$"] ×2 |
| 4.1s | C2S | m293 ? [5.0, 1.0] ×5 |
| 4.5s | C2S | m61 ? [0] |
| 4.7s | C2S | op09 len=1 09 |
| 4.7s | S2C | S2C m561 ? [7] |
| 4.7s | S2C | S2C m681 挑战令牌推送 [3, 39, "czZpE6UmzSYMlAAhoD3m43XIWgcgX5MbiQ6fVUIe"] |
| 4.7s | S2C | S2C m756 ? [4] |
| 4.7s | S2C | S2C m681 挑战令牌推送 [4, 33] |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] VisualPlayers=210 |
| 4.7s | S2C | 建视图0x15 view=116 cap=1 |
| 4.7s | S2C | 视图加行0x18 view=116 obj#1 +22属性 |
| 4.7s | S2C | op14 len=12 14017400000001000000a701 |
| 4.7s | S2C | op29(子码47) len=98 |
| 4.7s | S2C | op14 len=12 14017400000001000000a801 |
| 4.7s | S2C | 记录加行0x11[TempGemSkillRec#424 0x74/0x1] +0行 |
| 4.7s | S2C | S2C m416 ? ["DomainJiuYinZhi ", 6] |
| 4.7s | S2C | S2C m419 ? [6] |
| 4.7s | S2C | S2C m259 ? [11] |
| 4.7s | S2C | S2C m386 UI开关推送 [1, 112, "time_wudaoyanwu_fight"] |
| 4.7s | S2C | S2C m256 ? [1, 112] |
| 4.7s | S2C | S2C m386 UI开关推送 [1, 105, "time_jianghudaluandou_ready"] ×12 |
| 4.7s | S2C | S2C m256 ? [1, 105] |
| 4.7s | S2C | S2C m386 UI开关推送 [1, 104, "time_wudaodahui_pipei"] ×12 |
| 4.7s | S2C | S2C m256 ? [1, 104] |
| 4.7s | S2C | S2C m386 UI开关推送 [1, 103, "time_balance_queue"] ×4 |
| 4.7s | S2C | S2C m256 ? [1, 103] |
| 4.7s | S2C | S2C m386 UI开关推送 [2, 103, "Battle_balance_ownside"] ×3 |
| 4.7s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 4.7s | S2C | S2C m386 UI开关推送 [3, 131, "ltc_challenger_wait_time"] ×4 |
| 4.7s | S2C | S2C m496 ? [2, 10370, 4, 12, 7, 5] |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] BufferInfo2=1E37,7020FA4,3B3CC3,0,0,0 |
| 4.7s | C2S | m1016 ? [140] |
| 4.7s | C2S | m758 ? [10] |
| 4.7s | C2S | m949 ? [3] |
| 4.7s | C2S | m858 ? [6] |
| 4.7s | S2C | S2C m169 buff添加 ["buf_baosd_02", obj(0x7020fa4,0x3b3cc3)] |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] IsGuildCaptain=1 |
| 4.7s | S2C | S2C m505 ? [1] |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] MaxWuJiPoint=95 |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] WuJiPoint=81 |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] MaxSP=50 ×2 |
| 4.7s | S2C | 建视图0x15 view=47(?) cap=512 |
| 4.7s | S2C | 视图加行0x18 view=47(?) obj#1 +13属性 ×15 |
| 4.7s | S2C | 建视图0x15 view=48(?) cap=128 |
| 4.7s | S2C | 视图加行0x18 view=48(?) obj#1 +8属性 ×15 |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] ZQPLValue=2673440 |
| 4.7s | S2C | 建视图0x15 view=44(?) cap=512 |
| 4.7s | S2C | 视图加行0x18 view=44(?) obj#1 +5属性 ×70 |
| 4.7s | S2C | 建视图0x15 view=45(经脉激活) cap=128 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#1 +9属性 |
| 4.7s | S2C | op14 len=12 14012d00000001000000a301 |
| 4.7s | S2C | 记录加行0x11[plug_record#419 0x2d/0x1] +0行 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#2 +9属性 |
| 4.7s | S2C | op14 len=12 14012d00000002000000a301 |
| 4.7s | S2C | 记录加行0x11[plug_record#419 0x2d/0x2] +0行 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#3 +9属性 |
| 4.7s | S2C | op14 len=12 14012d00000003000000a301 |
| 4.7s | S2C | 记录加行0x11[plug_record#419 0x2d/0x3] +0行 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#4 +9属性 |
| 4.7s | S2C | op14 len=12 14012d00000004000000a301 |
| 4.7s | S2C | 记录加行0x11[plug_record#419 0x2d/0x4] +0行 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#5 +9属性 |
| 4.7s | S2C | op14 len=12 14012d00000005000000a301 |
| 4.7s | S2C | 记录加行0x11[plug_record#419 0x2d/0x5] +0行 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#6 +9属性 |
| 4.7s | S2C | op14 len=12 14012d00000006000000a301 |
| 4.7s | S2C | 记录加行0x11[plug_record#419 0x2d/0x6] +0行 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#7 +9属性 |
| 4.7s | S2C | op14 len=12 14012d00000007000000a301 |
| 4.7s | S2C | 记录加行0x11[plug_record#419 0x2d/0x7] +0行 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#8 +9属性 |
| 4.7s | S2C | op14 len=12 14012d00000008000000a301 |
| 4.7s | S2C | 记录加行0x11[plug_record#419 0x2d/0x8] +0行 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#9 +9属性 |
| 4.7s | S2C | op14 len=12 14012d00000009000000a301 |
| 4.7s | S2C | 记录加行0x11[plug_record#419 0x2d/0x9] +0行 |
| 4.7s | S2C | 视图加行0x18 view=45(经脉激活) obj#10 +7属性 ×13 |
| 4.7s | S2C | 建视图0x15 view=43(内功) cap=512 |
| 4.7s | S2C | 视图加行0x18 view=43(内功) obj#1 +18属性 ×40 |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] CurNeiGongWuXing=6 |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] BufferInfo3=4BF8,7020FA4,3B3CC3,5,0,0 |
| 4.7s | S2C | S2C m169 buff添加 ["buf_ng_jz_006", obj(0x7020fa4,0x3b3cc3)] |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] YinLevel=1089 |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] YangLevel=623 |
| 4.7s | S2C | 属性更新0x10[O#0x3b3cc3] TaijiLevel=299 |
| 4.7s | S2C | 建视图0x15 view=40(武学技能) cap=800 |
| 4.7s | S2C | 视图加行0x18 view=40(武学技能) obj#1 +20属性 ×503 |
| 4.8s | S2C | 建视图0x15 view=41(装备?) cap=512 |
| 4.8s | S2C | 视图加行0x18 view=41(装备?) obj#1 +19属性 ×16 |
| 4.8s | S2C | 建视图0x15 view=170(?) cap=10 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] ModifySkillLockTime=200 |
| 4.8s | S2C | 建视图0x15 view=29 cap=40 |
| 4.8s | S2C | 建视图0x15 view=1(主属性) cap=52 |
| 4.8s | S2C | op2A len=325 2a47440100004f01000000000080010001002200 |
| 4.8s | S2C | op14 len=12 140101000000010000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0x1] +0行 |
| 4.8s | S2C | op2A len=290 2a472101000029010000000000c0010003002000 |
| 4.8s | S2C | op14 len=12 140101000000030000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0x3] +0行 |
| 4.8s | S2C | op2A len=283 2a471a0100002b010000000000c0010004002000 |
| 4.8s | S2C | op14 len=12 140101000000040000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0x4] +0行 |
| 4.8s | S2C | op2A len=290 2a47210100002a010000000000c0010006002000 |
| 4.8s | S2C | op14 len=12 140101000000060000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0x6] +0行 |
| 4.8s | S2C | op2A len=261 2a47040100000b01000000000090010007001e00 |
| 4.8s | S2C | op14 len=12 140101000000070000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0x7] +0行 |
| 4.8s | S2C | op2A len=315 2a473a01000055010000000000c0010008002000 |
| 4.8s | S2C | op14 len=12 140101000000080000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0x8] +0行 |
| 4.8s | S2C | op2A len=265 2a47080100000f0100000000009001000b001e00 |
| 4.8s | S2C | op14 len=12 1401010000000b0000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0xb] +0行 |
| 4.8s | S2C | op2A len=263 2a47060100000f0100000000009001000c001e00 |
| 4.8s | S2C | op14 len=12 1401010000000c0000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0xc] +0行 |
| 4.8s | S2C | op2A len=263 2a4706010000130100000000008001000d001e00 |
| 4.8s | S2C | op14 len=12 1401010000000d0000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0xd] +0行 |
| 4.8s | S2C | op2A len=287 2a471e0100002c010000000000a001000e002000 |
| 4.8s | S2C | op14 len=12 1401010000000e0000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0xe] +0行 |
| 4.8s | S2C | op2A len=278 2a47150100002c010000000000a001000f002000 |
| 4.8s | S2C | op14 len=12 1401010000000f0000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0xf] +0行 |
| 4.8s | S2C | op2A len=303 2a472e01000045010000000000c0010016002200 |
| 4.8s | S2C | op14 len=12 140101000000160000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x1/0x16] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#23 +23属性 ×2 |
| 4.8s | S2C | op14 len=12 140101000000180000008b01 |
| 4.8s | S2C | 记录加行0x11[SkillModifyPackRec#395 0x1/0x18] +0行 |
| 4.8s | S2C | op14 len=12 140101000000180000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x1/0x18] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#25 +23属性 |
| 4.8s | S2C | op14 len=12 140101000000190000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x19] +0行 |
| 4.8s | S2C | op14 len=12 140101000000190000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x1/0x19] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#26 +23属性 |
| 4.8s | S2C | op14 len=12 1401010000001a0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x1a] +0行 |
| 4.8s | S2C | op14 len=12 1401010000001a0000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x1/0x1a] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#27 +23属性 |
| 4.8s | S2C | op14 len=12 1401010000001b0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x1b] +0行 |
| 4.8s | S2C | op14 len=12 1401010000001b0000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x1/0x1b] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#28 +23属性 |
| 4.8s | S2C | op14 len=12 1401010000001c0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x1c] +0行 |
| 4.8s | S2C | op14 len=12 1401010000001c0000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x1/0x1c] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#29 +23属性 |
| 4.8s | S2C | op14 len=12 1401010000001d0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x1d] +0行 |
| 4.8s | S2C | op14 len=12 1401010000001d0000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x1/0x1d] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#30 +21属性 |
| 4.8s | S2C | op14 len=12 1401010000001e0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x1e] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#31 +21属性 |
| 4.8s | S2C | op14 len=12 1401010000001f0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x1f] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#32 +21属性 |
| 4.8s | S2C | op14 len=12 140101000000200000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x20] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#33 +21属性 |
| 4.8s | S2C | op14 len=12 140101000000210000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x21] +0行 |
| 4.8s | S2C | 视图加行0x18 view=1(主属性) obj#34 +21属性 |
| 4.8s | S2C | op14 len=12 140101000000220000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x1/0x22] +0行 |
| 4.8s | S2C | S2C m648 ? [1, -1, 6, 1] |
| 4.8s | S2C | 建视图0x15 view=64 cap=6 |
| 4.8s | S2C | S2C m586 状态推送(簇首) [5] |
| 4.8s | S2C | S2C m504 ? [1, 2] |
| 4.8s | S2C | S2C m386 UI开关推送 [1, 28, "ui_jh_002"] |
| 4.8s | S2C | S2C m144 ? ["qiandengzhen_02"] |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] AutoTrace=1 |
| 4.8s | S2C | S2C m52 服务器时间 [1, 1794116366] |
| 4.8s | S2C | S2C m758 玩家名片 [1] |
| 4.8s | S2C | S2C m430 ? [5, 0] |
| 4.8s | S2C | 建视图0x15 view=174(?) cap=65626 |
| 4.8s | S2C | 建视图0x15 view=175(?) cap=2 |
| 4.8s | S2C | 视图加行0x18 view=175(?) obj#1 +18属性 |
| 4.8s | S2C | 建视图0x15 view=176(?) cap=65626 |
| 4.8s | S2C | 建视图0x15 view=177(?) cap=2 |
| 4.8s | S2C | 建视图0x15 view=178(?) cap=65626 |
| 4.8s | S2C | 建视图0x15 view=179(?) cap=2 |
| 4.8s | S2C | 建视图0x15 view=180(?) cap=65626 |
| 4.8s | S2C | 建视图0x15 view=181(?) cap=2 |
| 4.8s | S2C | 建视图0x15 view=2(背包) cap=65626 |
| 4.8s | S2C | 视图加行0x18 view=2(背包) obj#1 +19属性 ×40 |
| 4.8s | S2C | 建视图0x15 view=3(任务?) cap=2 |
| 4.8s | S2C | 视图加行0x18 view=3(任务?) obj#1 +17属性 ×2 |
| 4.8s | S2C | 建视图0x15 view=121(背包-装备类) cap=65626 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#1 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000010000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x1] +0行 |
| 4.8s | S2C | op14 len=12 140179000000010000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x1] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#2 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000020000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x2] +0行 |
| 4.8s | S2C | op14 len=12 140179000000020000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x2] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#3 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000030000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x3] +0行 |
| 4.8s | S2C | op14 len=12 140179000000030000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x3] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#4 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000040000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x4] +0行 |
| 4.8s | S2C | op14 len=12 140179000000040000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x4] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#5 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000050000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x5] +0行 |
| 4.8s | S2C | op14 len=12 140179000000050000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x5] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#6 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000060000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x6] +0行 |
| 4.8s | S2C | op14 len=12 140179000000060000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x6] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#7 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000070000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x7] +0行 |
| 4.8s | S2C | op14 len=12 140179000000070000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x7] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#8 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000080000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x8] +0行 |
| 4.8s | S2C | op14 len=12 140179000000080000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x8] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#9 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000090000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x9] +0行 |
| 4.8s | S2C | op14 len=12 140179000000090000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x9] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#10 +23属性 |
| 4.8s | S2C | op14 len=12 1401790000000a0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0xa] +0行 |
| 4.8s | S2C | op14 len=12 1401790000000a0000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0xa] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#11 +22属性 |
| 4.8s | S2C | op14 len=12 1401790000000b0000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0xb] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#12 +20属性 |
| 4.8s | S2C | op14 len=12 1401790000000c0000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0xc] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#13 +20属性 |
| 4.8s | S2C | op14 len=12 1401790000000d0000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0xd] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#14 +24属性 ×3 |
| 4.8s | S2C | op14 len=12 140179000000100000008b01 |
| 4.8s | S2C | 记录加行0x11[SkillModifyPackRec#395 0x79/0x10] +0行 |
| 4.8s | S2C | op14 len=12 140179000000100000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0x10] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#17 +24属性 |
| 4.8s | S2C | op14 len=12 140179000000110000008b01 |
| 4.8s | S2C | 记录加行0x11[SkillModifyPackRec#395 0x79/0x11] +0行 |
| 4.8s | S2C | op14 len=12 140179000000110000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0x11] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#18 +24属性 |
| 4.8s | S2C | op14 len=12 140179000000120000008b01 |
| 4.8s | S2C | 记录加行0x11[SkillModifyPackRec#395 0x79/0x12] +0行 |
| 4.8s | S2C | op14 len=12 140179000000120000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0x12] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#19 +24属性 |
| 4.8s | S2C | op14 len=12 140179000000130000008b01 |
| 4.8s | S2C | 记录加行0x11[SkillModifyPackRec#395 0x79/0x13] +0行 |
| 4.8s | S2C | op14 len=12 140179000000130000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0x13] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#20 +24属性 |
| 4.8s | S2C | op14 len=12 140179000000140000008b01 |
| 4.8s | S2C | 记录加行0x11[SkillModifyPackRec#395 0x79/0x14] +0行 |
| 4.8s | S2C | op14 len=12 140179000000140000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0x14] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#21 +20属性 |
| 4.8s | S2C | op14 len=12 140179000000150000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0x15] +0行 |
| 4.8s | S2C | op2A len=263 2a47060100001501000000804080790016001f00 |
| 4.8s | S2C | op14 len=12 140179000000160000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x79/0x16] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#23 +24属性 ×2 |
| 4.8s | S2C | op2A len=303 2a472e0100004101000000000080790019002100 |
| 4.8s | S2C | op14 len=12 140179000000190000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x79/0x19] +0行 |
| 4.8s | S2C | op14 len=12 140179000000190000008f01 |
| 4.8s | S2C | 记录加行0x11[PoisonRec#399 0x79/0x19] +0行 |
| 4.8s | S2C | op2A len=302 2a472d0100003d0100000000008079001a002100 |
| 4.8s | S2C | op14 len=12 1401790000001a0000008a01 |
| 4.8s | S2C | 记录加行0x11[PropModifyPackRec#394 0x79/0x1a] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#27 +21属性 |
| 4.8s | S2C | op14 len=12 1401790000001b0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x1b] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#28 +21属性 |
| 4.8s | S2C | op14 len=12 1401790000001c0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x1c] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#29 +23属性 |
| 4.8s | S2C | op14 len=12 1401790000001d0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x1d] +0行 |
| 4.8s | S2C | op14 len=12 1401790000001d0000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x1d] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#30 +23属性 |
| 4.8s | S2C | op14 len=12 1401790000001e0000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x1e] +0行 |
| 4.8s | S2C | op14 len=12 1401790000001e0000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x1e] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#31 +22属性 |
| 4.8s | S2C | op14 len=12 1401790000001f0000008b01 |
| 4.8s | S2C | 记录加行0x11[SkillModifyPackRec#395 0x79/0x1f] +0行 |
| 4.8s | S2C | op14 len=12 1401790000001f0000009701 |
| 4.8s | S2C | 记录加行0x11[AddFaculty#407 0x79/0x1f] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#32 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000200000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x20] +0行 |
| 4.8s | S2C | op14 len=12 140179000000200000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x20] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#33 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000210000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x21] +0行 |
| 4.8s | S2C | op14 len=12 140179000000210000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x21] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#34 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000220000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x22] +0行 |
| 4.8s | S2C | op14 len=12 140179000000220000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x22] +0行 |
| 4.8s | S2C | 视图加行0x18 view=121(背包-装备类) obj#35 +23属性 |
| 4.8s | S2C | op14 len=12 140179000000230000009001 |
| 4.8s | S2C | 记录加行0x11[RandomPropRec#400 0x79/0x23] +0行 |
| 4.8s | S2C | op14 len=12 140179000000230000009101 |
| 4.8s | S2C | 记录加行0x11[RandomSkillRec#401 0x79/0x23] +0行 |
| 4.8s | S2C | 建视图0x15 view=122 cap=2 |
| 4.8s | S2C | 视图加行0x18 view=122 obj#1 +20属性 ×2 |
| 4.8s | S2C | 建视图0x15 view=123(背包-类3) cap=65626 |
| 4.8s | S2C | 视图加行0x18 view=123(背包-类3) obj#1 +13属性 ×28 |
| 4.8s | S2C | 建视图0x15 view=124 cap=2 |
| 4.8s | S2C | 视图加行0x18 view=124 obj#1 +20属性 |
| 4.8s | S2C | 建视图0x15 view=125(背包-类4) cap=65626 |
| 4.8s | S2C | 视图加行0x18 view=125(背包-类4) obj#1 +11属性 ×25 |
| 4.8s | S2C | 建视图0x15 view=126 cap=2 |
| 4.8s | S2C | 视图加行0x18 view=126 obj#1 +20属性 |
| 4.8s | S2C | S2C m100 ? [obj(0x0,0x7), 1] |
| 4.8s | S2C | S2C m24 通用事件[0x3b3cc307020fa4] c1=33 [] |
| 4.8s | S2C | S2C m2 ? [obj(0xea61974c,0xc38)] |
| 4.8s | S2C | S2C m607 ? ["lizy86", "徐傠囃辭睼 "] |
| 4.8s | S2C | custom0x27(无法解TVar) |
| 4.8s | S2C | S2C m681 挑战令牌推送 [3, 2, "NJPLTLVwdtQoe6H2MdKMsF7NkdVjHhA2EJqKK0EX"] ×6 |
| 4.8s | S2C | S2C m438 ? [5] |
| 4.8s | S2C | 建视图0x15 view=159 cap=24 |
| 4.8s | S2C | S2C m386 UI开关推送 [1, 1, "ui_guild_kill"] ×2 |
| 4.8s | S2C | S2C m64 ? [12, 0] |
| 4.8s | S2C | S2C m281 ? [1, 7100037, 7100037] |
| 4.8s | S2C | S2C m382 ? [2, 0] |
| 4.8s | S2C | 位置0x1F(短len=25) |
| 4.8s | S2C | 属性更新0x10[V#0x28] PhyDamageMul=1.1 |
| 4.8s | S2C | 属性更新0x10[V#0x28] JujDamageAdd=35 ×3 |
| 4.8s | S2C | 属性更新0x10[V#0x28] StiDamageAdd=130 ×4 |
| 4.8s | S2C | 属性更新0x10[V#0x28] PhyDamageMul=1.1 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] StrAdd=341 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] DexAdd=160 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] IngAdd=931 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] SpiAdd=254 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] StaAdd=483 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MaxHPAdd=24125 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MaxMPAdd=6355 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] HPUpSpeedAdd=38 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MPUpSpeedAdd=57 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MeleePowerAdd=247 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] ShotPowerAdd=181 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MagicPowerAdd=206 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] PhyHitAdd=182 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MagicHitAdd=314 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] DodgeAdd=109 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] PhyVaAdd=110 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MagicVaAdd=288 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] VaDefAdd=164 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MinMagicDefAdd=802 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MaxParryAdd=9486 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] ParryUpSpeedAdd=10 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] PhyDamageMul=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] StiDamageMul=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] JujDamageMul=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] NegDamageMul=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MasDamageMul=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] StiDefAdd=9 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] JujDefAdd=235 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] NegDefAdd=510 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MasDefAdd=9 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] DodgeProbAdd=8.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MagicVaProbAdd=4.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MagicVaDamageAdd=30.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] VaDamageDefAdd=76.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] AIPhyDefAdd=480 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] PIPhyDefAdd=2.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] AIMagicDefAdd=456 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MaxQingGongPointAdd=15 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] LiveSpeedPower=7 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] HPResumeMul=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MPResumeMul=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] AIPhyParryAdd=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] AIMagicParryAdd=30.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] PhyParryDamResAdd=9.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MagicParryDamResAdd=25.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] AIStiDefAdd=9 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] AIJujDefAdd=32 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MPHeartSpeedAdd=-350 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] HPResumeOnceMul=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MPResumeOnceMul=1.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] PhyFinalDamageAdd=1344 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] StiFinalDamageAdd=128 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] JujFinalDamageAdd=144 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MaxHP=4160 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MaxMP=4033 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] HPUpSpeed=44 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MPUpSpeed=43 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MeleePower=344 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] ShotPower=163 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MagicPower=934 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] PhyHit=163 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MagicHit=257 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] Dodge=163 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] PhyVa=344 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MagicVa=257 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] VaDef=243 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MinMagicDef=51 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] ParryRatio=100 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MaxParry=10372 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] ParryUpSpeed=104 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] PhyDef=7448 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MaxMeleeDamage=307 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MinMeleeDamage=280 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MaxShotDamage=4 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MinShotDamage=4 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MoveSpeed=4.5 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] RunSpeedAdd=0.6 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] DriftJumpSpeedAdd=3.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] SndJumpSpeedAdd=4.8 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] ThdJumpSpeedAdd=3.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] AirRushDistAdd=5.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] DropHeightPub=24.0 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] UnderWaterModifySpeed=1.0 ×2 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] HPRatio=100 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] HitHPRatio=100 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] MPRatio=100 |
| 4.8s | S2C | 属性更新0x10[O#0x3b3cc3] ProRefreshFlag=1 |
| 4.8s | S2C | S2C m647 ? [7] |
| 4.8s | S2C | 建视图0x15 view=46(轻功) cap=512 |
| 4.8s | S2C | 视图加行0x18 view=46(轻功) obj#1 +9属性 ×4 |
| 4.8s | S2C | S2C m98 数值推送 [1] |
| 4.8s | S2C | S2C m228 ? [53] |
| 4.8s | S2C | S2C m98 数值推送 [12] ×2 |
| 4.8s | S2C | S2C m136 ? [0] |
| 4.8s | S2C | S2C m137 ? [0] |
| 4.9s | S2C | S2C m228 ? [53] |
| 4.9s | S2C | S2C m98 数值推送 [274] ×2 |
| 4.9s | S2C | S2C m229 小数值推送 [130, obj(0x7001c94,0x1cb3), "npc\itemnpc057", -924.789, 444.0, 1186.123] ×62 |
| 4.9s | S2C | S2C m596 ? [2] |
| 4.9s | S2C | 属性更新0x10[O#0x3b3cc3] IsMobileIDBind=1 |
| 4.9s | S2C | S2C m235 ? [5] |
| 5.0s | C2S | m327 ? [6, "83,85,70068,20549,70501,20798,81,39331,5"] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行0 列1 =117440512 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行1 列1 =117440768 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行2 列1 =117441024 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行3 列1 =117441280 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行4 列1 =117441536 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行5 列1 =117441792 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行6 列1 =117442048 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行7 列1 =117442304 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行8 列1 =117442560 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行0 列1 =117440512 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行1 列1 =117440768 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行2 列1 =117441024 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行3 列1 =117441280 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行4 列1 =117441536 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行5 列1 =117441792 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行6 列1 =117442048 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行7 列1 =117442304 0x7020fa4] |
| 5.0s | S2C | 记录写格0x13[Task_Accepted#29 行8 列1 =117442560 0x7020fa4] |
| 5.2s | C2S | m301 ? [4, 1, "00155D3490D8"] |
| 5.3–144.3s | PAIR | C2S m924 MSG_DELAY(client ms) → S2C m294 回显[client ms, obj, 服务器unix秒, …] ×139 |
| 5.3s | C2S | m760 ? [16.0] |
| 5.3s | C2S | m761 ? [15] |
| 5.3s | C2S | m182 ? [1] ×2 |
| 5.3s | S2C | S2C m382 ? [8, 0] |
| 5.3s | S2C | S2C m390 ? [5, 0] |
| 5.3s | C2S | m409 ? [55] |
| 5.4s | S2C | S2C m681 挑战令牌推送 [5, 1] |
| 5.7s | S2C | 属性更新0x10[O#0x3b3cc3] AIPhyParryAdd=11.0 |
| 5.7s | S2C | op28(子码47) len=234 |
| 5.7s | S2C | op0D len=251 0d65d900073d043b002b776944e8fb4242a15277 ×2 |
| 5.7s | S2C | op28(子码47) len=233 ×4 |
| 6.1s | S2C | 属性更新0x10[O#0x3b3cc3] QingGongPoint=160 |
| 6.7s | S2C | op28(子码47) len=255 ×3 |
| 6.7s | S2C | op0D len=63 0df3330007bb34000093007044db792f429c2c7e ×3 |
| 6.7s | S2C | op28(子码47) len=235 ×4 |
| 7.2s | C2S | m82 ? [3] |
| 7.4s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 7.5s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 7.8s | S2C | op28(子码47) len=222 ×5 |
| 9.1s | S2C | 属性更新0x10[O#0x3b3cc3] QingGongPoint=165 |
| 11.9s | S2C | 属性更新0x10[O#0x150] State= |
| 12.0s | C2S | m760 ? [1, "821610 "] |
| 12.0s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 12.0s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 12.0s | S2C | 属性更新0x10[O#0x3b3cc3] IsCheckPass=1 |
| 12.0s | S2C | S2C m1 系统提示 [2, "24001"] |
| 13.4s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 13.5s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 14.5s | S2C | 对象同步0x21[0x7000152] 码177 len=91 |
| 14.6s | S2C | op06 len=1 06 |
| 14.8s | S2C | S2C m98 数值推送 [145] |
| 14.9s | C2S | m54 格挡状态(值23644070824) |
| 14.9s | S2C | 属性更新0x10[O#0x3b3cc3] MoveSpeed=1.77 |
| 14.9s | S2C | 属性更新0x10[O#0x3b3cc3] RunSpeedAdd=-2.13 |
| 14.9s | S2C | 属性更新0x10[O#0x3b3cc3] InParry=1 |
| 14.9s | S2C | 属性更新0x10[O#0x3b3cc3] ParryUpSpeedAdd=-93 |
| 14.9s | S2C | 属性更新0x10[O#0x3b3cc3] BufferInfo4=11CE,7020FA4,3B3CC3,1,0,0 |
| 14.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7020fa4,0x3b3cc3)] |
| 14.9s | S2C | 属性更新0x10[O#0x3b3cc3] InManualParry=1 |
| 15.0s | C2S | m54 格挡状态(值23420030849) |
| 15.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7020fa4,0x3b3cc3)] |
| 15.0s | S2C | 属性更新0x10[O#0x3b3cc3] BufferInfo4= |
| 15.0s | S2C | 属性更新0x10[O#0x3b3cc3] InParry=0 |
| 15.0s | S2C | 属性更新0x10[O#0x3b3cc3] MoveSpeed=4.5 |
| 15.0s | S2C | 属性更新0x10[O#0x3b3cc3] RunSpeedAdd=0.6 |
| 15.0s | S2C | 属性更新0x10[O#0x3b3cc3] ParryUpSpeedAdd=10 |
| 15.0s | S2C | 属性更新0x10[O#0x3b3cc3] InManualParry=0 |
| 15.0s | C2S | m54 格挡状态(值23420030849) ×3 |
| 15.4s | C2S | m15 移动上报 自动 state=4 pos=(928.3,48.7,998.9) |
| 15.6s | S2C | 对象同步0x21[0x7020fa4] 码254 len=47 |
| 16.8s | S2C | 属性更新0x10[O#0x15c] State=stand |
| 16.9s | S2C | 对象同步0x21[0x7000157] 码81 len=91 |
| 17.7–18.8s | S2C | 属性更新0x10[O#0x157] State=stand ×2 |
| 17.8s | S2C | 对象同步0x21[0x7000152] 码177 len=91 |
| 18.8s | S2C | 属性更新0x10[O#0x15c] State=walk |
| 18.8s | S2C | op2C len=130 2c47810000000a010000000000e0060052010007 |
| 19.4s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 19.5s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 19.6s | C2S | m464 ? [0] |
| 21.8s | S2C | op31(sub1) 0x7000152 |
| 23.1s | C2S | m187 ? [2, "form_stage_main\form_attribute_mall\form", "DesertPointNHSBShaHe,DesertPointNHSBXunB"] |
| 23.2s | S2C | custom0x27(无法解TVar) |
| 23.2s | C2S | m187 ? [1, 101.0] |
| 23.3s | C2S | m522 ? [5] |
| 23.4s | C2S | m187 ? [2, "form_stage_main\form_attribute_mall\form", "ZLCLCLZBPointLoop"] ×2 |
| 23.4s | C2S | m514 ? [0] |
| 23.4s | C2S | m198 ? [1, "rank_nhzf_WarRes_001", -2147483648] |
| 23.4s | S2C | S2C m672 ? [2, "form_stage_main\form_attribute_mall\form", "ZLCLCLZBPointLoop,0,20000"] |
| 23.4s | S2C | custom0x27(无法解TVar) |
| 23.4s | S2C | S2C m199 ? [2, "rank_nhzf_WarRes_001"] |
| 23.4s | C2S | m187 ? [1, 102.0] |
| 23.4s | S2C | S2C m199 ? [3, "rank_nhzf_WarRes_001", 0, obj(0xea2bca94,0xc38), " "] |
| 23.4s | S2C | S2C m772 ? [1, "侠骨丹心 ", "侠骨丹心 ", "群雄逐鹿 ", "群雄逐鹿 ", " "] |
| 23.4s | C2S | m187 ? [1, 101.0] ×3 |
| 24.8s | S2C | op31(sub1) 0x7000157 |
| 25.4s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 25.5s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 27.2s | C2S | m82 ? [3] |
| 29.8s | C2S | m187 ? [2, "form_stage_main\form_attribute_mall\form", "SceneSchoolTaskPoint"] |
| 29.8s | S2C | op06 len=1 06 |
| 29.8s | C2S | m187 ? [1, 96.0] |
| 29.9s | S2C | S2C m672 ? [2, "form_stage_main\form_attribute_mall\form", "SceneSchoolTaskPoint,0,3000"] |
| 29.9s | C2S | m187 ? [1, 97.0] ×2 |
| 31.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 31.5s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 34.8s | S2C | S2C m1 系统提示 [1, "11403"] ×100 |
| 35.8s | C2S | m187 ? [2, "form_stage_main\form_attribute_mall\form", "MyrPointC, MyrPointD, MyrPointE"] |
| 35.8s | S2C | S2C m672 ? [2, "form_stage_main\form_attribute_mall\form", "MyrPointC,0,0; MyrPointD,0,0; MyrPointE,"] |
| 35.8–36.0s | C2S | m187 ? [1, 98.0] ×3 |
| 37.2s | S2C | 对象同步0x21[0x700014b] 码96 len=223 |
| 37.2s | S2C | 属性更新0x10[O#0x150] State=stand ×2 |
| 37.2s | S2C | 多属性0x2D[0x1500700](布局变体) len=23 2d01004b010007500100000100540004 |
| 37.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 37.6s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 37.8s | C2S | m110 ? [2] |
| 37.9s | S2C | S2C m759 ? [2, 0, 100, 1, 0, 0] |
| 37.9s | C2S | m110 ? [2] |
| 37.9s | S2C | S2C m759 ? [2, 0, 100, 1, 0, 0] |
| 38.4s | C2S | m986 ? [39.0, 8.0] |
| 38.4s | S2C | S2C m673 ? [41, 0, 45, 300] |
| 39.1s | C2S | m187 ? [2, "form_stage_main\form_attribute_mall\form", ""] |
| 39.2s | S2C | S2C m672 ? [2, "form_stage_main\form_attribute_mall\form", ""] |
| 39.2s | C2S | m187 ? [1, 67.0] |
| 39.6s | C2S | m167 ? [3.0] |
| 39.7s | S2C | custom0x27(无法解TVar) |
| 39.7s | C2S | m187 ? [2, "form_stage_main\form_attribute_mall\form", ""] ×2 |
| 39.7s | C2S | m167 ? [5.0] |
| 39.7s | S2C | S2C m672 ? [2, "form_stage_main\form_attribute_mall\form", ""] |
| 39.7s | S2C | S2C m59 ? [6, 4, 4, 1] |
| 40.6s | C2S | m187 ? [1, 13.0] |
| 40.9s | S2C | S2C m1 系统提示 [1, "sys_smzb_018", 17] |
| 40.9s | C2S | m876 ? [3, 0.0, 0.0] |
| 41.0s | S2C | S2C m287 面板数据 [6, 3, 1, 2, 2, 2] |
| 41.0s | C2S | m876 ? [0] ×2 |
| 41.0s | C2S | m865 ? [2, obj(0x7020fa4,0x3b3cc3)] |
| 41.0s | C2S | m876 ? [2, 1.0] ×6 |
| 41.0s | S2C | S2C m287 面板数据 [6, 0, 1, 900, 405, 0] ×4 |
| 41.0s | S2C | custom0x27(无法解TVar) ×3 |
| 41.0s | S2C | S2C m287 面板数据 [6, 5, 29, " ", 0, 0] |
| 41.0s | S2C | custom0x27(无法解TVar) |
| 41.0–41.4s | S2C | S2C m287 面板数据 [9, 1, 0, 5, "", 0] ×15 |
| 41.5s | C2S | m789 ? [3] |
| 41.6s | S2C | S2C m287 面板数据 [1, 1, 1, 16, 1, 192] |
| 41.6s | C2S | m781 ? [0] |
| 41.6s | S2C | S2C m644 ? [2, 10, 180] |
| 41.6s | S2C | 记录写格0x13[Count_Limit_Form_Rec#363 行5 列1 =33555712 0x0] |
| 41.6s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 41.6s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 41.6s | S2C | S2C m252 ? [] |
| 41.6s | S2C | S2C m250 ? [] ×3 |
| 41.7s | S2C | S2C m634 ? [0, 0, 0, 0, 0, 0] |
| 41.7–41.9s | S2C | S2C m287 面板数据 [1, 1, 1, 21, 0, 0] ×3 |
| 42.4s | C2S | m965 ? [9] |
| 42.5s | S2C | S2C m447 ? [8, 0] |
| 42.5s | C2S | m965 ? [12] ×2 |
| 42.5s | S2C | S2C m447 ? [11, 1, 6] ×2 |
| 43.0s | C2S | m398 ? [3] |
| 43.0s | S2C | S2C m751 ? [2, 0, 0, 0, "5,6", 0] |
| 43.0s | C2S | m198 ? [2, "rank_clone_441", 0] |
| 43.0s | C2S | m435 ? [2, "ini\scene\clone048", "rank_clone_441"] ×2 |
| 43.0s | S2C | S2C m679 ? [1, 10, 12] |
| 43.0s | C2S | m187 ? [2, "form_stage_main\form_attribute_mall\form", "equip_second_yu_split_point,LG_reload,eq"] ×2 |
| 43.0s | C2S | m766 ? [8] |
| 43.1s | S2C | S2C m199 ? [3, "rank_clone_441", 0, obj(0xea5c9c17,0xc38), " "] |
| 43.1s | C2S | m398 ? [0] |
| 43.1s | S2C | S2C m672 ? [2, "form_stage_main\form_attribute_mall\form", "equip_second_yu_split_point,0,0;LG_reloa"] |
| 43.1s | C2S | m719 ? [4] ×3 |
| 43.2s | S2C | S2C m751 ? [0, "5,6"] |
| 43.3s | S2C | S2C m614 ? [2, 19, 0, 0, 0, 3315] |
| 43.3s | C2S | m447 ? [4] |
| 43.3s | S2C | S2C m735 ? [1, 3, 0, "0"] |
| 43.3s | C2S | m766 ? [4] |
| 43.4s | C2S | m447 ? [7] |
| 43.4s | C2S | m398 ? [1, 1.0] |
| 43.5s | S2C | S2C m735 ? [3, 3, 0, "0"] |
| 43.5s | S2C | custom0x27(无法解TVar) |
| 43.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 43.8s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 44.7s | C2S | m905 ? [1, "rank_global_mind_player_hard", -2147483648] |
| 44.7s | C2S | m198 ? [1, "rank_mind_player_hard", obj(0xea3c105b,0xc38)] |
| 44.7s | C2S | m438 ? [0] |
| 44.7s | S2C | custom0x27(无法解TVar) |
| 44.8s | S2C | S2C m199 ? [1, "rank_mind_player_hard"] |
| 44.8s | S2C | S2C m726 ? [11, 0, 0] ×2 |
| 45.1s | S2C | op06 len=1 06 |
| 46.9s | C2S | m911 ? [1.0, "task_tyg"] |
| 46.9s | S2C | S2C m770 ? [0, 75601, 0, 1, 1, 75602] |
| 47.2s | C2S | m82 ? [3] |
| 47.3s | S2C | op02 len=369 0200000000000000000000000000000000000000 |
| 47.4s | C2S | op01 len=53 010000000000000000000000000000000000000000000000 |
| 47.5s | C2S | m187 ? [2, "form_stage_main\form_attribute_mall\form", "ittaketwo_price"] |
| 47.5s | S2C | S2C m672 ? [2, "form_stage_main\form_attribute_mall\form", "ittaketwo_price,0,0"] |
| 47.5s | C2S | m187 ? [1, 95.0] |
| 47.9s | C2S | m504 ? [0] |
| 48.0s | C2S | m187 ? [1, 95.0] ×2 |
| 48.3s | C2S | m449 ? [2] |
| 48.4s | S2C | S2C m740 ? [1] |
| 48.4s | C2S | m449 ? [3] |
| 48.4s | S2C | S2C m740 ? [2, " ", ""] |
| 49.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 49.6s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 50.5s | C2S | m503 ? [3] |
| 50.5s | S2C | S2C m352 ? [4, 0, "徐傠囃辭睼 ", 2, ""] |
| 53.6s | C2S | m477 ? [7] ×2 |
| 53.8s | S2C | S2C m571 ? [0, 0] |
| 55.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 55.7s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 56.1s | C2S | m693 ? [1] |
| 56.2s | S2C | custom0x27(无法解TVar) |
| 57.3s | C2S | m477 ? [7] ×2 |
| 57.4s | S2C | S2C m571 ? [0, 0] |
| 59.5s | C2S | m479 ? [2, 2, "rank_1_RevengeIntegral,1"] |
| 59.5s | S2C | custom0x27(无法解TVar) |
| 59.5s | C2S | m479 ? [2, 1, "rank_1_khbd"] |
| 59.5s | C2S | m965 ? [7, 2020.0] ×2 |
| 59.6s | S2C | custom0x27(无法解TVar) ×3 |
| 59.6s | C2S | m108 ? [] |
| 59.6s | S2C | S2C m171 ? ["万长青 ", "李饼 ", "徐轩辕悟剑 ", "元気菲酱 ", "刘季一万钱 ", "林珞宸 "] |
| 60.1s | S2C | 属性更新0x10[V#0x2] LeftTime=602450 |
| 60.3s | S2C | op06 len=1 06 |
| 61.1s | C2S | m784 ? [10, 3] |
| 61.1s | S2C | S2C m640 ? [10, 3] |
| 61.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 61.7s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 61.9s | C2S | m198 ? [2, "rank_1_SkillUseValue", 0] |
| 62.0s | S2C | custom0x27(无法解TVar) |
| 62.5s | S2C | 属性更新0x10[O#0x150] State= |
| 62.7s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 62.7s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 64.4s | C2S | m302 ? [4] ×4 |
| 64.4–66.6s | S2C | 对象同步0x21[0x7000152] 码149 len=47 ×3 |
| 64.4s | S2C | custom0x27(无法解TVar) ×2 |
| 64.5s | S2C | S2C m368 ? [13, 22, 0] ×2 |
| 64.7s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 65.2s | S2C | op28(子码47) len=231 |
| 65.3s | C2S | m302 ? [18] |
| 65.4s | S2C | S2C m597 ? ["1,1,2,1,5,1,"] |
| 65.8–66.5s | S2C | 属性更新0x10[O#0x157] State=stand ×2 |
| 67.2s | C2S | m82 ? [3] |
| 67.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 67.6s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 72.2s | S2C | op28(子码47) len=231 |
| 73.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 73.7s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 74.9s | C2S | m151 ? [93.0] ×3 |
| 75.0s | S2C | custom0x27(无法解TVar) |
| 75.1s | C2S | m151 ? [5.0, "24C333CA005740879B7CEE501A7AFC09"] |
| 75.1s | S2C | custom0x27(无法解TVar) |
| 75.1s | C2S | m151 ? [25.0, "24C333CA005740879B7CEE501A7AFC09"] ×2 |
| 75.1s | S2C | S2C m200 ? [16, "24C333CA005740879B7CEE501A7AFC09", 0, 1, 0, 1] ×2 |
| 75.5s | S2C | op06 len=1 06 |
| 79.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 79.7s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 84.9s | S2C | 对象同步0x21[0x7000152] 码177 len=91 |
| 85.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 85.8s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 86.8s | S2C | 属性更新0x10[O#0x15c] State=stand |
| 87.0s | S2C | 对象同步0x21[0x7000157] 码81 len=91 |
| 87.2s | C2S | m82 ? [3] |
| 87.3s | S2C | S2C m1 系统提示 [3, "15917", "徐一拳出货 ", "ini\scene\clone047", 2000] |
| 87.9s | S2C | 属性更新0x10[O#0x150] State=stand ×2 |
| 87.9s | S2C | 属性更新0x10[O#0x157] State=stand |
| 88.0s | S2C | op2C len=142 2c478d00000036010000000000e007004b010007 |
| 88.0s | S2C | 多属性0x2D[0x1500700](布局变体) len=23 2d01004b010007500100000100540004 |
| 88.9s | S2C | 属性更新0x10[O#0x157] State=walk |
| 88.9s | S2C | 属性更新0x10[O#0x15c] State=walk |
| 88.9s | S2C | op2C len=130 2c47810000000a010000000000e0060052010007 |
| 89.7s | C2S | m423 ? [0, 0.0] |
| 89.8s | S2C | S2C m603 ? [2, " ", 0, 0] |
| 89.8s | S2C | S2C m681 挑战令牌推送 [3, 45, "J4IK05V0TuMKzOk4/GfhMaNZ5f2OJoEq2YhcLQ/0"] |
| 89.8s | C2S | m423 ? [0, 100.0] |
| 89.8s | S2C | S2C m603 ? [2, " ", 0, 100] |
| 89.8s | S2C | S2C m681 挑战令牌推送 [3, 46, "+DQdBDGHFw0LKKbr6YJjMqvmbY/GLX5jNIDbMW5o"] |
| 90.8s | S2C | op06 len=1 06 |
| 90.8s | C2S | m133 ? [5] |
| 90.8s | S2C | 建视图0x15 view=149 cap=4 |
| 91.5s | S2C | op31(sub1) 0x7000152 |
| 91.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 91.8s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 91.8s | C2S | m651 ? [] |
| 91.8s | C2S | m667 ? [] |
| 91.8s | S2C | 建视图0x15 view=110(?) cap=30 |
| 91.8s | S2C | 建视图0x15 view=108 cap=30 |
| 91.8s | S2C | S2C m183 ? [-2] |
| 92.9s | C2S | m560 ? [1, 0, 0.0] |
| 93.0s | S2C | custom0x27(无法解TVar) |
| 93.0s | S2C | S2C m98 数值推送 [274] |
| 93.4s | C2S | m390 ? [30] |
| 93.5s | S2C | S2C m477 ? [30] |
| 94.5s | S2C | op31(sub1) 0x7000157 |
| 94.6s | C2S | m560 ? [1, 0, 1] |
| 97.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 97.8s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 101.6s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 101.6s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 101.7s | S2C | S2C m250 ? [] ×2 |
| 102.4s | C2S | m965 ? [12] |
| 102.4s | S2C | S2C m447 ? [11, 1, 6] |
| 103.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 103.8s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 104.7s | S2C | 属性更新0x10[O#0x3b3cc3] LiveGroove_4=37 |
| 104.7s | S2C | 属性更新0x10[O#0x3b3cc3] LiveGroove_5=40860 |
| 104.7s | S2C | S2C m1 系统提示 [1, "8052", 223] |
| 104.7s | S2C | S2C m349 ? [223] |
| 104.7s | S2C | 属性更新0x10[O#0x3b3cc3] Faculty=114282521 |
| 104.7s | S2C | 属性更新0x10[O#0x3b3cc3] HoldFaculty=1501508 |
| 106.0s | S2C | op06 len=1 06 |
| 107.2s | C2S | m82 ? [3] |
| 109.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 109.8s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 110.1s | S2C | 属性更新0x10[O#0x3b3cc3] FightData=1343859035614 |
| 113.2s | S2C | 属性更新0x10[O#0x150] State= |
| 113.4s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 113.4s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 115.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 115.8s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 120.1s | S2C | 属性更新0x10[V#0x2] LeftTime=602390 |
| 121.2s | S2C | op06 len=1 06 |
| 121.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 121.8s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 124.7s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 124.7s | S2C | S2C m566 令牌应答串 ["PGjlSPiz+np3JjCiQ2Jg1veCH4QTo7eLoS8c"] |
| 127.2s | C2S | m82 ? [3] |
| 127.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 127.8s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 133.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 133.9s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 134.1s | C2S | m663 ? [] |
| 134.2s | C2S | m666 ? [] |
| 134.9s | C2S | m133 ? [6] |
| 134.9s | S2C | 删视图0x16 view=149 |
| 134.9s | C2S | m423 ? [6, 0.0] ×2 |
| 135.0s | S2C | S2C m681 挑战令牌推送 [4, 45] ×2 |
| 135.8s | S2C | 属性更新0x10[O#0x157] State=stand |
| 135.9s | C2S | m784 ? [10, 3] |
| 135.9s | S2C | op28(子码47) len=225 |
| 136.0s | S2C | S2C m640 ? [10, 3] |
| 136.1–136.9s | S2C | 对象同步0x21[0x7000152] 码149 len=91 ×2 |
| 136.5s | S2C | op06 len=1 06 |
| 136.8s | S2C | 属性更新0x10[O#0x157] State=walk |
| 138.4s | S2C | 属性更新0x10[O#0x150] State=stand ×2 |
| 138.6s | S2C | 对象同步0x21[0x700014b] 码96 len=223 |
| 138.6s | S2C | 多属性0x2D[0x1500700](布局变体) len=23 2d01004b010007500100000100540004 |
| 139.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "56F34EEDFC654A0F8C49745664DD9446"] |
| 139.9s | C2S | m15 移动上报 自动 state=5 pos=(928.3,48.7,998.9) ×2 |
| 141.5s | C2S | m819 ? [] |
| 141.5s | S2C | 属性更新0x10[O#0x3b3cc3] JobID1=15 |
| 141.5s | S2C | 属性更新0x10[O#0x3b3cc3] JobID2=7 |
| 141.5s | S2C | 属性更新0x10[O#0x3b3cc3] JobID3=4 |
| 141.5s | S2C | 属性更新0x10[O#0x3b3cc3] LastGenDate=46334 |
| 141.5s | S2C | S2C m111 ? [46333, 0, 15, 7, 4, 0] |
| 141.9s | S2C | op28(子码47) len=231 |

## d.pcapng — 103.143.19.173:7071（与 b/e 同客户端口 52396）

场景：短会话：mzjf02/06 出招两次。事件 284 条 → 合并为 192 段。抓包起点 2026-11-08 14:02:26（本地墙钟）

| 时间 | 方向 | 交互 |
|---|---|---|
| 0.0s | S2C | op06 len=1 06 |
| 0.0–13.6s | PAIR | C2S ping(0x18) → S2C 0x38 回显 ×14 |
| 0.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 0.2s | C2S | m15 移动上报 自动 state=5 pos=(924.0,48.7,1001.5) ×2 |
| 0.5–13.6s | PAIR | C2S m924 MSG_DELAY(client ms) → S2C m294 回显[client ms, obj, 服务器unix秒, …] ×14 |
| 1.6s | C2S | m15 移动上报 自动 state=7 pos=(924.0,48.7,1001.5) |
| 1.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 1.6s | C2S | m15 移动上报 自动 state=7 pos=(924.0,48.7,1001.5) |
| 1.8s | S2C | 对象同步0x21[0x700ca8c] 码171 len=91 |
| 1.9s | C2S | m15 移动上报 自动 state=4 pos=(923.2,48.7,1000.2) |
| 2.0s | S2C | 对象同步0x21[0x700ca8c] 码174 len=47 |
| 2.3s | C2S | m54 格挡状态(值53999866395) |
| 2.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 2.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 2.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 2.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 2.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 2.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 2.4s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 2.5s | C2S | m54 格挡状态(值53775826420) |
| 2.5s | C2S | 对象请求 len=33 |
| 2.5s | C2S | m431 选择目标 [obj(0x7022e58,0x3b8a14)] |
| 2.6s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 2.6s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 2.6s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 2.6s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 2.6s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 2.6s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 2.6s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 2.6s | C2S | m54 格挡状态(值53775826420) ×3 |
| 2.7s | C2S | m15 移动上报 自动 state=4 pos=(923.2,48.7,1000.2) |
| 2.7s | C2S | m211 使用技能(出招) CS_jh_mzjf02 14值 [923.167, 48.748, 1000.165, -1.442] |
| 2.7s | C2S | m82 ? [3] |
| 2.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 2.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_mzjf02 |
| 2.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 2.7s | S2C | S2C m284 空值标记帧 [] |
| 2.7s | S2C | 属性更新0x10[O#0x3b8a14] LogicState=1 |
| 2.7s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 2.7s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 2.7s | S2C | 属性更新0x10[O#0x3b8a19] LogicState=1 |
| 2.7s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +1行 |
| 2.7s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +93行 |
| 2.7s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =23808 0x700ca8c] ×4 |
| 2.7s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 2.7s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +94行 |
| 2.7s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x700ca8c] ×4 |
| 2.7s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 2.7s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10287 |
| 2.7s | S2C | 位置0x1F(短len=25) |
| 2.7s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [923.167, 48.746, 1000.165, -1.442, 0] |
| 2.7s | S2C | 属性更新0x10[O#0x3b6fb8] LastWGAttackNpc=monm031302 |
| 2.7s | S2C | S2C m24 通用事件[0x3b8a1407022e58] 动作开始 [1, obj(0x700ca8c,0x3b6fb8)] |
| 2.7s | S2C | S2C m19 锁定伤害段[0x3b6fb80700ca8c] [obj(0x7022e58,0x3b8a14), "CS_jh_mzjf02", 1800, 1800, 919.92, 48.746] |
| 2.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 2.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 2.7s | C2S | m15 移动上报 自动 state=7 pos=(923.2,48.7,1000.2) |
| 2.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 2.8s | C2S | m15 移动上报 自动 state=7 pos=(923.3,48.8,1000.2) |
| 2.9s | S2C | op2C len=183 2c47b60000000a010000000000a006008cca0007 |
| 2.9s | S2C | 多属性0x2D[0x8a140702](布局变体) len=23 2d0100582e0207148a3b000100a0018c |
| 3.1s | S2C | 多属性0x2D[0x8a190704](布局变体) len=23 2d010039040407198a3b000100a0018c |
| 3.2s | S2C | S2C m169 buff添加 ["eqp_wp_06a", obj(0x7022e58,0x3b8a14)] |
| 3.2s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 3.2s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 3.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 3.2s | S2C | S2C m1 系统提示 [2, "8066"] |
| 3.2s | S2C | S2C m24 通用事件[0x3b8a1407022e58] 伤害飘字 [678, 548, 0, "CS_jh_mzjf02", "SkillLock", "monm031302"] |
| 3.2s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 3.2s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 3.2s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 3.2s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 26, 160, "ui_jh_001"] |
| 3.2s | S2C | S2C m24 通用事件[0x3b8a1407022e58] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031302", "徐傠囃辭睼 "] |
| 3.2s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jh_mzjf02", obj(0x7022e58,0x3b8a14)] |
| 3.3s | S2C | 多属性0x2D[0x8a140702](布局变体) len=101 2d0100582e0207148a3b000700770224 |
| 4.4s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 4.4s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 4.4s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 4.5s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 4.5s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 5.0s | C2S | m914 招式结束异步位置 [919.916, 48.747, 1000.588, -1.442, 12, 5] |
| 5.0s | C2S | m990 ? [obj(0x7022e58,0x3b8a14), 921.723, 48.756, 1000.353] |
| 5.0s | C2S | 对象请求 len=33 |
| 5.0s | C2S | m431 选择目标 [obj(0x7040439,0x3b8a19)] |
| 5.0s | C2S | 对象请求 len=33 |
| 5.0s | C2S | m431 选择目标 [obj(0x7040439,0x3b8a19)] |
| 5.1s | S2C | 位置0x1F(短len=25) |
| 5.1s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 5.1s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 5.2s | S2C | S2C m169 buff移除 ["eqp_wp_06a", obj(0x7022e58,0x3b8a14)] |
| 5.2s | S2C | 属性更新0x10[O#0x3b8a14] LogicState=0 |
| 5.2s | S2C | 对象同步0x21[0x700ca8c] 码158 len=91 |
| 5.2s | S2C | 多属性0x2D[0x8a140702](布局变体) len=20 2d0100582e0207148a3b000100770201 |
| 5.4s | S2C | 属性更新0x10[V#0x2] LeftTime=601120 |
| 5.8s | C2S | m163 ? [2] |
| 5.9s | S2C | S2C m639 ? [2] |
| 6.2s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 6.3s | C2S | m15 移动上报 自动 state=5 pos=(919.9,48.7,1000.6) ×2 |
| 6.5–8.2s | S2C | 对象同步0x21[0x7040439] 码119 len=47 ×2 |
| 7.1s | S2C | 属性更新0x10[O#0x3b8a19] State= |
| 7.1s | S2C | 属性更新0x10[O#0x3b8a19] CurSkillID=default_normal_skill |
| 7.1s | S2C | 属性更新0x10[O#0x3b8a19] CurSkillTarget=None |
| 7.1s | S2C | 位置0x1F(短len=25) |
| 7.1s | S2C | S2C m337 施法回显[0x3b8a1907040439] [920.335, 48.746, 999.565, 5.894, 0] |
| 7.1s | S2C | S2C m20 范围施放标记[0x3b8a1907040439] ["atkhit3", obj(0x700ca8c,0x3b6fb8), 0, 0.0] |
| 7.4s | C2S | m15 移动上报 手动 state=7 pos=(919.9,48.7,1000.6) |
| 7.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 7.5s | C2S | m15 移动上报 自动 state=4 pos=(919.4,48.4,1000.6) |
| 7.5s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10387 |
| 7.5s | S2C | 对象同步0x21[0x700ca8c] 码252 len=47 |
| 7.8s | S2C | 对象同步0x21[0x700ca8c] 码32 len=47 |
| 8.0s | C2S | m211 使用技能(出招) CS_jh_mzjf06 10值 [919.393, 48.418, 1000.604, -1.54] |
| 8.1s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 8.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_mzjf06 |
| 8.1s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +94行 |
| 8.1s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =24064 0x700ca8c] ×4 |
| 8.1s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 8.1s | S2C | 记录写格0x13[cooldown_rec#109 行93 列1 =16801024 0x700ca8c] ×2 |
| 8.1s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10297 |
| 8.1s | S2C | 位置0x1F(短len=25) |
| 8.1s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [919.393, 48.416, 1000.604, -3.878, 0] |
| 8.1s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_jh_mzjf06", 0, 0, 0.0] |
| 8.1s | C2S | m15 移动上报 自动 state=4 pos=(919.4,48.4,1000.6) |
| 8.1s | S2C | 属性更新0x10[O#0x3b6fb8] LastAttackerNpc=monm031302 |
| 8.1s | S2C | 属性更新0x10[O#0x3b6fb8] LastDodge=32.07 |
| 8.1s | S2C | 属性更新0x10[O#0x3b6fb8] SP=3 |
| 8.1s | S2C | 属性更新0x10[O#0x3b6fb8] HPRatio=99 |
| 8.1s | S2C | 属性更新0x10[O#0x3b6fb8] HP=28253 |
| 8.1s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 8.1s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 伤害飘字 [32, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 8.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHPRatio=99 |
| 8.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=28269 |
| 8.1s | S2C | S2C m205 伤害目标列表[0x3b8a1907040439] ["default_normal_skill", obj(0x700ca8c,0x3b6fb8)] |
| 8.1s | S2C | 属性更新0x10[O#0x3b8a19] CurSkillID= |
| 8.1s | S2C | 属性更新0x10[O#0x3b8a19] CurSkillTarget=None |
| 8.1s | S2C | S2C m189 对象通知 [obj(0x7040439,0x3b8a19)] |
| 8.5s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 8.5s | S2C | 属性更新0x10[O#0x3b6fb8] SP=5 |
| 8.5s | S2C | 属性更新0x10[O#0x3b8a19] InParry=1 |
| 8.5s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 8.5s | S2C | S2C m24 通用事件[0x3b8a1907040439] 伤害飘字 [678, 590, 0, "CS_jh_mzjf06", "SkillNormal", "monm031302"] |
| 8.5s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 8.5s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 8.5s | S2C | 属性更新0x10[O#0x3b8a19] InParry=0 |
| 8.5s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 8.5s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 31, 160, "ui_jh_001"] |
| 8.5s | S2C | S2C m24 通用事件[0x3b8a1907040439] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031302", "徐傠囃辭睼 "] |
| 8.5s | S2C | S2C m18 战斗伤害帧[0x3b8a1907040439] ["hurt_half", 0, obj(0x700ca8c,0x3b6fb8)] |
| 8.5s | S2C | S2C m24 通用事件[0x3b8a1907040439] 动作开始 ["mzjf_hurt_01_self", obj(0x700ca8c,0x3b6fb8)] |
| 8.5s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jh_mzjf06", obj(0x7040439,0x3b8a19)] |
| 8.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 8.8s | S2C | 多属性0x2D[0x8a190704](布局变体) len=66 2d010039040407198a3b000700d70100 |
| 9.7s | S2C | op28(子码47) len=231 |
| 10.1s | C2S | m15 移动上报 手动 state=7 pos=(919.4,48.4,1000.6) |
| 10.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 10.3s | S2C | 对象同步0x21[0x700ca8c] 码194 len=47 |
| 10.6s | S2C | 属性更新0x10[O#0x3b8a19] LogicState=0 |
| 11.1s | C2S | m15 移动上报 自动 state=4 pos=(923.7,48.7,1000.5) |
| 11.2–11.8s | S2C | 对象同步0x21[0x700ca8c] 码130 len=47 ×2 |
| 11.5s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 11.5s | C2S | m54 格挡状态(值53999866395) |
| 11.5s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 11.5s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 11.5s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 11.5s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 11.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 11.5s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 11.5s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 11.6s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 11.7s | C2S | m54 格挡状态(值53775826420) |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 11.7s | S2C | op28(子码47) len=231 |
| 11.7s | C2S | m54 格挡状态(值53775826420) ×2 |
| 11.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 11.7s | C2S | m15 移动上报 自动 state=4 pos=(923.7,48.7,1000.5) |
| 11.7s | C2S | m54 格挡状态(值53775826420) ×2 |
| 12.1s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 12.1s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 12.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 12.2s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 12.2s | C2S | m15 移动上报 自动 state=5 pos=(923.7,48.7,1000.5) ×2 |
| 13.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHPRatio=100 |
| 13.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=28285 |
| 13.1s | S2C | 属性更新0x10[O#0x3b6fb8] HPRatio=100 |
| 13.1s | S2C | 属性更新0x10[O#0x3b6fb8] HP=28285 |
| 1794117748.2s | C2S | 6001 大厅心跳 04 00 00 00 10 EE 11 11 → 01 FF 00 00 ×2 |

## e.pcapng — 103.143.19.173:7071

场景：长战斗会话：mzjf/kfkj/lyfh/hyjf 全家桶 + 冲锋落地循环。事件 3255 条 → 合并为 1979 段。抓包起点 2026-11-08 14:15:55（本地墙钟）

| 时间 | 方向 | 交互 |
|---|---|---|
| 0.0s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26431 |
| 0.1s | S2C | 多属性0x2D[0xbb570700](布局变体) len=19 2d0100f68f000757bb3b000100e0015c |
| 0.2s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 0.2s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 0.8–207.4s | PAIR | C2S m924 MSG_DELAY(client ms) → S2C m294 回显[client ms, obj, 服务器unix秒, …] ×205 |
| 0.9–207.6s | PAIR | C2S ping(0x18) → S2C 0x38 回显 ×190 |
| 1.2–2.3s | S2C | 多属性0x2D[0xbb570700](布局变体) len=19 2d0100f68f000757bb3b000100e00160 ×2 |
| 2.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 2.7s | C2S | m15 移动上报 自动 state=5 pos=(943.1,43.9,1036.2) ×2 |
| 3.6s | C2S | m15 移动上报 手动 state=7 pos=(943.1,43.9,1036.2) |
| 3.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 3.8s | S2C | 对象同步0x21[0x700ca8c] 码217 len=47 |
| 4.4s | C2S | m54 格挡状态(值9560079747) |
| 4.5s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 4.5s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 4.5s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 4.5s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 4.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 4.5s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 4.5s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 4.5–4.9s | C2S | m15 移动上报 手动 state=7 pos=(939.4,43.9,1037.1) ×4 |
| 4.6s | S2C | 对象同步0x21[0x700ca8c] 码33 len=47 |
| 4.8s | S2C | 对象同步0x21[0x700ca8c] 码147 len=47 |
| 5.0s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26513 |
| 5.1s | S2C | 对象同步0x21[0x700ca8c] 码206 len=47 |
| 5.2s | C2S | m54 格挡状态(值9336039772) |
| 5.2s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 5.2s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 5.2s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 5.2s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 5.2s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 5.2s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 5.2s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 5.2s | C2S | m54 格挡状态(值9336039772) ×2 |
| 5.3s | C2S | m15 移动上报 手动 state=7 pos=(939.9,43.9,1038.1) |
| 5.3s | C2S | m54 格挡状态(值9336039772) |
| 5.4s | C2S | m54 格挡状态(值9560079747) |
| 5.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 5.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 5.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 5.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 5.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 5.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 5.4s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 5.4s | C2S | m15 移动上报 手动 state=7 pos=(940.6,43.9,1038.7) ×2 |
| 5.5s | S2C | 对象同步0x21[0x700ca8c] 码68 len=47 |
| 5.6s | C2S | m54 格挡状态(值9336039772) |
| 5.6s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 5.6s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 5.6s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 5.6s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 5.6s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 5.6s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 5.6s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 5.6s | C2S | m54 格挡状态(值9336039772) ×2 |
| 5.7s | C2S | m15 移动上报 手动 state=7 pos=(940.5,43.9,1039.0) |
| 5.7s | S2C | 对象同步0x21[0x700ca8c] 码38 len=47 |
| 5.8s | C2S | m54 格挡状态(值9560079747) |
| 5.9s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 5.9s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 5.9s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 5.9s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 5.9s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 5.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 5.9s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 5.9s | C2S | m15 移动上报 手动 state=7 pos=(939.5,43.9,1039.2) ×2 |
| 5.9s | S2C | 对象同步0x21[0x700ca8c] 码66 len=47 |
| 6.0s | C2S | m54 格挡状态(值9336039772) ×3 |
| 6.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 6.0s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 6.0s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 6.0s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 6.0s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 6.0s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 6.0s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 6.0s | C2S | m54 格挡状态(值9336039772) ×2 |
| 6.1s | S2C | 对象同步0x21[0x700ca8c] 码214 len=47 |
| 6.1s | C2S | m15 移动上报 手动 state=7 pos=(939.1,43.9,1038.9) |
| 6.3s | S2C | 对象同步0x21[0x700ca8c] 码102 len=47 |
| 6.5s | C2S | m15 移动上报 手动 state=8 pos=(938.1,43.9,1037.2) |
| 6.8s | S2C | 对象同步0x21[0x700ca8c] 码228 len=47 |
| 6.9s | C2S | m15 移动上报 手动 state=6 pos=(937.2,45.2,1035.8) |
| 7.0s | S2C | 对象同步0x21[0x700ca8c] 码20 len=47 |
| 7.2s | C2S | m15 移动上报 手动 state=8 pos=(936.3,45.3,1034.4) |
| 7.3s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=158 |
| 7.3s | S2C | 属性更新0x10[O#0x3b6fb8] JumpTimes=1 |
| 7.4s | S2C | 对象同步0x21[0x700ca8c] 码138 len=47 |
| 7.4–7.7s | C2S | m15 移动上报 手动 state=6 pos=(935.8,47.2,1033.6) ×2 |
| 7.6s | S2C | 对象同步0x21[0x700ca8c] 码18 len=47 |
| 7.8s | S2C | 对象同步0x21[0x700ca8c] 码116 len=47 |
| 7.9s | C2S | m15 移动上报 手动 state=8 pos=(934.8,50.2,1031.9) |
| 7.9s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=152 |
| 8.1s | S2C | 对象同步0x21[0x700ca8c] 码253 len=47 |
| 8.1–8.4s | C2S | m15 移动上报 手动 state=6 pos=(934.2,52.1,1031.0) ×2 |
| 8.2s | S2C | 对象同步0x21[0x700ca8c] 码243 len=47 |
| 8.5s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=162 |
| 8.6s | C2S | m15 移动上报 手动 state=6 pos=(933.5,54.1,1029.9) |
| 8.6s | S2C | 对象同步0x21[0x700ca8c] 码37 len=91 |
| 8.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 8.6s | C2S | m15 移动上报 手动 state=5 pos=(933.4,54.2,1029.8) ×2 |
| 8.7s | C2S | m15 移动上报 手动 state=6 pos=(933.3,54.3,1029.6) |
| 8.8s | C2S | m15 移动上报 手动 state=7 pos=(933.3,54.4,1029.6) |
| 8.9s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=152 |
| 8.9s | S2C | 对象同步0x21[0x700ca8c] 码5 len=91 |
| 8.9–9.7s | C2S | m15 移动上报 手动 state=7 pos=(932.1,54.4,1027.7) ×7 |
| 9.1–9.9s | S2C | 对象同步0x21[0x700ca8c] 码188 len=91 ×5 |
| 9.9s | C2S | m54 格挡状态(值9560079747) |
| 9.9s | C2S | m15 移动上报 手动 state=8 pos=(924.3,54.4,1015.2) |
| 9.9s | S2C | 属性更新0x10[O#0x3b6fb8] JumpTimes=0 |
| 9.9s | C2S | m54 格挡状态(值9336039772) |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 10.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 10.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 10.0s | C2S | m15 移动上报 手动 state=6 pos=(924.0,54.3,1014.8) |
| 10.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] HPRatio=94 |
| 10.0s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26595 |
| 10.1–10.4s | C2S | m15 移动上报 手动 state=6 pos=(924.0,54.2,1014.7) ×2 |
| 10.1s | S2C | 对象同步0x21[0x700ca8c] 码131 len=135 |
| 10.5s | C2S | m54 格挡状态(值9560079747) |
| 10.5s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 10.5s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 10.5s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 10.5s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 10.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 10.5s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 10.5s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 10.6s | C2S | m54 格挡状态(值9336039772) |
| 10.6s | S2C | 对象同步0x21[0x700ca8c] 码141 len=47 |
| 10.6s | C2S | m15 移动上报 手动 state=6 pos=(922.8,52.4,1013.0) |
| 10.6s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 10.6s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 10.6s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 10.6s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 10.6s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 10.6s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 10.6s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 10.7–10.8s | C2S | m15 移动上报 手动 state=6 pos=(922.7,51.7,1012.8) ×2 |
| 10.8s | S2C | 对象同步0x21[0x700ca8c] 码231 len=91 |
| 10.9s | C2S | m54 格挡状态(值9560079747) |
| 11.0s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 11.0s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 11.0s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 11.0s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 11.0s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 11.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 11.0s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 11.0s | C2S | m15 移动上报 手动 state=7 pos=(921.9,48.7,1011.5) |
| 11.0s | S2C | 对象同步0x21[0x700ca8c] 码102 len=47 |
| 11.0s | S2C | S2C m446 区域boss进度 [6] |
| 11.0s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] c1=48 [935.818, 43.886, 1029.985, 20.0, 1] |
| 11.0s | C2S | m15 移动上报 手动 state=7 pos=(921.9,48.7,1011.5) |
| 11.1s | C2S | m54 格挡状态(值9336039772) |
| 11.1s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 11.1s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 11.1s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 11.1s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 11.1s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 11.1s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 11.1s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 11.1s | C2S | m54 格挡状态(值9336039772) ×2 |
| 11.2s | C2S | m15 移动上报 手动 state=7 pos=(922.0,48.7,1011.3) |
| 11.2s | S2C | 对象同步0x21[0x700ca8c] 码103 len=47 |
| 11.4s | S2C | 对象同步0x21[0x700ca8c] 码77 len=47 |
| 11.6s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=162 |
| 11.7s | C2S | m54 格挡状态(值9560079747) |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 11.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 11.7s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 11.7–12.2s | C2S | m15 移动上报 自动 state=4 pos=(922.2,48.7,1008.5) ×2 |
| 11.8–12.3s | S2C | 对象同步0x21[0x700ca8c] 码224 len=47 ×2 |
| 12.3s | C2S | m54 格挡状态(值9336039772) ×2 |
| 12.3s | C2S | m15 移动上报 手动 state=7 pos=(922.2,48.7,1008.5) |
| 12.3s | C2S | m54 格挡状态(值9336039772) |
| 12.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 12.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 12.3s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 12.3s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 12.3s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 12.3s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 12.3s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 12.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 12.3s | C2S | m54 格挡状态(值9336039772) ×3 |
| 12.3s | C2S | m15 移动上报 手动 state=7 pos=(922.3,48.7,1008.7) |
| 12.5s | S2C | 对象同步0x21[0x700ca8c] 码26 len=91 |
| 13.7s | C2S | m15 移动上报 自动 state=4 pos=(925.6,48.7,1014.0) |
| 13.8s | S2C | 对象同步0x21[0x700ca8c] 码174 len=47 |
| 13.8s | C2S | m54 格挡状态(值9560079747) |
| 13.8s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 13.8s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 13.8s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 13.8s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 13.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 13.8s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 13.8s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 13.9s | S2C | op06 len=1 06 |
| 14.0s | C2S | m54 格挡状态(值9336039772) |
| 14.1s | S2C | S2C m446 区域boss进度 [7] |
| 14.1s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] c1=48 [935.818, 43.886, 1029.985, 20.0, 0] |
| 14.1s | C2S | m54 格挡状态(值9336039772) |
| 14.1s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 14.1s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 14.1s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 14.1s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 14.1s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 14.1s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 14.1s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 14.1s | C2S | m54 格挡状态(值9336039772) |
| 14.1s | C2S | m15 移动上报 ?2000 state=7 pos=(925.6,48.7,1014.0) |
| 14.1s | C2S | m54 格挡状态(值9336039772) |
| 14.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 14.2s | C2S | m54 格挡状态(值9336039772) |
| 14.2s | C2S | m15 移动上报 ?2000 state=7 pos=(925.5,48.7,1014.0) |
| 14.2s | S2C | 对象同步0x21[0x700ca8c] 码0 len=91 |
| 14.3s | C2S | m82 ? [3] |
| 14.4s | C2S | m15 移动上报 手动 state=7 pos=(924.4,48.7,1013.2) |
| 14.6s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=165 |
| 14.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 14.7s | S2C | 对象同步0x21[0x700ca8c] 码144 len=47 |
| 14.7s | C2S | m15 移动上报 ?4000 state=5 pos=(924.3,48.7,1012.4) ×2 |
| 15.1s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26677 |
| 15.1s | S2C | S2C m446 区域boss进度 [6] |
| 15.1s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] c1=48 [935.818, 43.886, 1029.985, 20.0, 1] |
| 16.7s | C2S | m54 格挡状态(值9560079747) |
| 16.7s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 16.7s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 16.7s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 16.7s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 16.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 16.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 16.7s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 16.7–18.7s | C2S | m15 移动上报 ?4000 state=7 pos=(923.2,48.7,1003.2) ×9 |
| 17.0s | S2C | 对象同步0x21[0x700ca8c] 码231 len=91 |
| 17.2–18.7s | S2C | 对象同步0x21[0x700ca8c] 码0 len=47 ×3 |
| 17.5s | S2C | 对象同步0x21[0x700ca8c] 码120 len=47 |
| 19.5s | C2S | m15 移动上报 手动 state=8 pos=(922.3,48.7,999.6) |
| 19.6s | S2C | 对象同步0x21[0x700ca8c] 码99 len=47 |
| 20.1s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26759 |
| 20.2s | C2S | m15 移动上报 手动 state=6 pos=(921.2,50.3,999.6) |
| 20.2s | S2C | 对象同步0x21[0x700ca8c] 码84 len=47 |
| 20.4s | C2S | m15 移动上报 手动 state=8 pos=(920.9,50.0,999.5) |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=158 |
| 20.4s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] JumpTimes=1 |
| 20.4s | C2S | m54 格挡状态(值9560079747) |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 20.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 20.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 20.4s | C2S | m15 移动上报 手动 state=6 pos=(920.7,50.7,999.5) |
| 20.5–22.9s | S2C | 对象同步0x21[0x700ca8c] 码78 len=47 ×2 |
| 20.5s | C2S | m54 格挡状态(值9336039772) |
| 20.5s | C2S | m15 移动上报 手动 state=6 pos=(920.5,51.2,999.5) |
| 20.5s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 20.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 20.5s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 20.5s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 20.5s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 20.5s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 20.5s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 20.5s | C2S | m54 格挡状态(值9336039772) ×3 |
| 20.6s | C2S | m15 移动上报 手动 state=6 pos=(920.4,51.8,999.5) |
| 20.7s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=165 |
| 20.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 20.7s | C2S | m15 移动上报 手动 state=5 pos=(920.1,52.5,999.5) ×2 |
| 20.7s | S2C | 对象同步0x21[0x700ca8c] 码45 len=91 |
| 20.8–22.5s | C2S | m15 移动上报 手动 state=6 pos=(919.4,53.6,999.4) ×7 |
| 20.9s | S2C | 对象同步0x21[0x700ca8c] 码163 len=47 |
| 21.1s | S2C | 对象同步0x21[0x700ca8c] 码81 len=47 |
| 21.6s | S2C | 对象同步0x21[0x700ca8c] 码62 len=47 |
| 21.8s | S2C | 对象同步0x21[0x700ca8c] 码188 len=47 |
| 22.2s | S2C | 对象同步0x21[0x700ca8c] 码200 len=47 |
| 22.4s | S2C | 对象同步0x21[0x700ca8c] 码108 len=47 |
| 22.6s | S2C | 对象同步0x21[0x700ca8c] 码176 len=47 |
| 22.6s | S2C | 属性更新0x10[O#0x3b6fb8] JumpTimes=0 |
| 22.7s | C2S | m15 移动上报 手动 state=4 pos=(920.4,48.7,999.5) |
| 22.7s | C2S | m15 移动上报 手动 state=7 pos=(920.4,48.7,999.5) |
| 22.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 24.4s | C2S | m15 移动上报 自动 state=4 pos=(928.1,48.7,1000.1) |
| 24.6s | S2C | 对象同步0x21[0x700ca8c] 码186 len=47 |
| 24.6s | C2S | m15 移动上报 ?1000 state=7 pos=(928.1,48.7,1000.1) |
| 24.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 24.8–26.3s | S2C | 对象同步0x21[0x700ca8c] 码0 len=47 ×2 |
| 25.2s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26841 |
| 25.3s | S2C | S2C m504 ? [1, 2] |
| 25.7s | C2S | m15 移动上报 自动 state=4 pos=(928.0,48.7,1001.4) |
| 25.8s | C2S | m15 移动上报 手动 state=7 pos=(928.0,48.7,1001.4) |
| 25.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 25.9s | S2C | 对象同步0x21[0x700ca8c] 码136 len=91 |
| 26.2s | C2S | m15 移动上报 手动 state=7 pos=(926.1,48.7,1001.3) ×2 |
| 26.5s | S2C | 对象同步0x21[0x700ca8c] 码216 len=47 |
| 26.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 26.7s | C2S | m15 移动上报 ?1000 state=5 pos=(925.8,48.7,1003.3) ×2 |
| 27.3s | C2S | m15 移动上报 自动 state=4 pos=(925.6,48.7,1006.2) |
| 27.4s | S2C | 对象同步0x21[0x700ca8c] 码169 len=47 |
| 27.4s | S2C | 属性更新0x10[O#0x3b6fb8] GroupID=-1 |
| 27.4s | S2C | S2C m504 ? [1, 2] |
| 27.4s | S2C | S2C m386 UI开关推送 [1, 28, "ui_jh_002"] |
| 27.4s | S2C | op0E len=9 0e7ec9000769bb3b00 ×2 |
| 27.4s | S2C | S2C m504 ? [1, 2] |
| 27.4s | S2C | 属性更新0x10[O#0x3b6fb8] InteractStatus=-1 |
| 27.4s | S2C | S2C m368 ? [5, 27, 0] |
| 27.4s | S2C | 记录删行0x12[InteractTraceRec#101 行101 0x700ca8c] |
| 27.4s | S2C | S2C m446 区域boss进度 [3, 0, 0, 0, 0, 0] |
| 27.4s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] c1=48 [0, 0, 0, 0, 0] |
| 27.4s | S2C | S2C m446 区域boss进度 [7] ×2 |
| 27.4s | C2S | m15 移动上报 ?2000 state=7 pos=(925.6,48.7,1006.2) |
| 27.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 27.5s | S2C | op0D len=251 0d65d900073d043b002b776944e8fb4242a15277 ×2 |
| 27.5s | S2C | op28(子码47) len=233 ×5 |
| 27.6–27.9s | C2S | m15 移动上报 ?2000 state=7 pos=(925.6,48.7,1005.2) ×2 |
| 27.6s | S2C | 对象同步0x21[0x700ca8c] 码129 len=47 |
| 27.8s | S2C | 对象同步0x21[0x700ca8c] 码19 len=47 |
| 27.9s | C2S | m15 移动上报 自动 state=4 pos=(925.7,48.7,1003.9) |
| 27.9s | C2S | m15 移动上报 ?1000 state=7 pos=(925.7,48.7,1003.9) |
| 28.0s | S2C | 对象同步0x21[0x700ca8c] 码181 len=47 |
| 28.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 28.1s | C2S | m15 移动上报 ?1000 state=7 pos=(925.7,48.7,1004.9) |
| 28.3s | S2C | 对象同步0x21[0x700ca8c] 码52 len=135 |
| 28.5s | S2C | op8B len=231 8bc0a1c1a1c1a150daf7a2c6dfc59ac12838ce85 |
| 28.5s | S2C | op28(子码47) len=234 |
| 28.5s | S2C | op0D len=251 0d16d70007e5b63b009e676444b88b2f42c3b275 |
| 28.5s | S2C | op28(子码47) len=235 ×2 |
| 28.5s | S2C | op0D len=251 0da75d000742043b00a6136744b88b2f428bbb72 |
| 28.5–29.0s | S2C | op28(子码47) len=222 ×5 |
| 29.1s | C2S | m15 移动上报 自动 state=4 pos=(925.4,48.7,1009.0) |
| 29.1s | S2C | 对象同步0x21[0x700ca8c] 码94 len=47 |
| 29.1s | C2S | m15 移动上报 ?2000 state=7 pos=(925.4,48.7,1009.0) |
| 29.1s | S2C | op06 len=1 06 |
| 29.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 29.3s | C2S | m15 移动上报 自动 state=4 pos=(925.4,48.7,1008.0) |
| 29.3s | S2C | 对象同步0x21[0x700ca8c] 码54 len=47 |
| 29.5s | S2C | op28(子码47) len=235 ×4 |
| 29.6–30.6s | S2C | 对象同步0x21[0x700ca8c] 码36 len=47 ×3 |
| 29.6s | S2C | S2C m1 系统提示 [2, "85201", "胡煜暻 ", "box_dsdh_dazaotai003", "tableitem_TJ_01_YJ"] |
| 30.0s | C2S | m15 移动上报 自动 state=4 pos=(925.4,48.7,1008.0) |
| 30.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHPRatio=95 |
| 30.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=26923 |
| 30.1s | S2C | 属性更新0x10[O#0x3b6fb8] HPRatio=95 |
| 30.1s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26923 |
| 30.1s | C2S | 对象请求 len=33 |
| 30.1s | C2S | m431 选择目标 [obj(0x701027b,0x3bbbc0)] |
| 30.1s | C2S | 对象请求 len=33 |
| 30.1s | C2S | m431 选择目标 [obj(0x701027b,0x3bbbc0)] |
| 30.2s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 30.2s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 30.5s | C2S | m211 使用技能(出招) CS_jh_hyjf05 10值 [925.44, 48.748, 1007.995, 3.069] |
| 30.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 30.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_hyjf05 |
| 30.5s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +97行 |
| 30.5s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x700ca8c] ×4 |
| 30.5s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 30.5s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 30.5s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 30.5s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 30.5s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10289 |
| 30.5s | S2C | 位置0x1F(短len=25) |
| 30.5s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [925.44, 48.746, 1007.995, 3.588, 0] |
| 30.5s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_jh_hyjf05", 0, 0, 0.0] |
| 30.5s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 30.5s | C2S | m15 移动上报 自动 state=4 pos=(925.4,48.7,1008.0) |
| 30.8s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 30.8s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 31.4s | C2S | m54 格挡状态(值9560079747) |
| 31.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 31.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 31.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 31.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 31.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 31.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 31.4s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 32.1s | C2S | m15 移动上报 手动 state=7 pos=(925.4,48.7,1008.0) |
| 32.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 32.2s | C2S | m54 格挡状态(值9336039772) |
| 32.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 32.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 32.3s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 32.3s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 32.3s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 32.3s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 32.3s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 32.3s | C2S | m54 格挡状态(值9336039772) ×2 |
| 32.3s | S2C | 对象同步0x21[0x700ca8c] 码30 len=47 |
| 32.3s | C2S | m15 移动上报 手动 state=7 pos=(925.4,48.7,1007.6) |
| 32.5s | S2C | 对象同步0x21[0x700ca8c] 码58 len=47 |
| 32.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 32.7s | C2S | m15 移动上报 手动 state=5 pos=(925.4,48.7,1006.2) ×2 |
| 32.8s | C2S | m15 移动上报 自动 state=4 pos=(925.3,48.7,1005.6) |
| 32.9s | S2C | 对象同步0x21[0x700ca8c] 码70 len=47 |
| 33.4s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [925.348, 48.748, 1005.65, -3.102] |
| 33.4s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 33.4s | S2C | 位置0x1F(短len=25) |
| 33.4s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [925.348, 48.746, 1005.65, 3.707] |
| 33.4s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_hyjf07 |
| 33.4s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 冲锋动画 [923.011, 49.746, 1001.965, 1, "qg_fastrun_2h_01"] |
| 33.4s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 33.4s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 33.4s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 33.4s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 33.4s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 33.4s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =16802048 0x700ca8c] ×2 |
| 33.4s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 33.4s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10199 |
| 33.4s | S2C | S2C m707 受理族变体 [15] |
| 33.6s | S2C | 对象同步0x21[0x700ca8c] 码176 len=135 |
| 33.7s | C2S | m212 冲锋/续招落地 [923.167, 48.746, 1002.211, -2.576] |
| 33.8s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 33.8s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 33.8s | S2C | 位置0x1F(短len=25) |
| 33.8s | S2C | S2C m26 技能释放 [] |
| 33.8s | S2C | 位置0x1F(短len=25) |
| 33.8s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [923.167, 48.746, 1002.211, 3.707] |
| 33.8s | S2C | 位置0x1F(短len=25) |
| 33.8s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [923.167, 48.746, 1002.211, 3.783, 0] |
| 33.8s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_jh_hyjf07", 0, 0, 0.0, 922.207, 48.746] |
| 33.8s | C2S | m15 移动上报 自动 state=4 pos=(923.0,48.7,1002.0) ×2 |
| 33.8s | S2C | 对象同步0x21[0x700ca8c] 码174 len=47 |
| 34.0s | S2C | 对象同步0x21[0x700ca8c] 码175 len=47 |
| 34.2s | S2C | op5E len=19 5e963af136f5569736f75e1bfcf7512f59cc56 |
| 34.2s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [2] |
| 34.2s | S2C | S2C m24 通用事件[0x3bbb790703dcef] 命中? [] |
| 34.2s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jh_hyjf07", obj(0x701027b,0x3bbbc0), obj(0x703dcef,0x3bbb79)] |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 34.2s | S2C | 属性更新0x10[V#0x28] CurTargetCount=2 |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] LastNGAttackNpc=monm031302 |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 34.2s | S2C | 属性更新0x10[O#0x3bbbc0] LogicState=1 |
| 34.2s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 34.2s | S2C | 属性更新0x10[O#0x3bbb79] LogicState=1 |
| 34.2s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +1行 |
| 34.2s | S2C | 属性更新0x10[O#0x3bbbc0] CurSkillID=default_normal_skill |
| 34.2s | S2C | 属性更新0x10[O#0x3bbbc0] CurSkillTarget=None |
| 34.2s | S2C | 位置0x1F(短len=25) |
| 34.2s | S2C | S2C m337 施法回显[0x3bbbc00701027b] [921.774, 48.746, 1000.346, 6.925, 0] |
| 34.2s | S2C | S2C m20 范围施放标记[0x3bbbc00701027b] ["atkhit3", obj(0x700ca8c,0x3b6fb8), 0, 0.0] |
| 34.2s | S2C | S2C m1 系统提示 [2, "8066"] |
| 34.2s | S2C | S2C m24 通用事件[0x3bbbc00701027b] 伤害飘字 [678, 1186, 2, "CS_jh_hyjf07", "SkillNormal", "monm031302"] |
| 34.2s | S2C | 记录加行0x11[JhBoss_Accept_Rec#22 0x700ca8c/0x3b6fb8] +0行 |
| 34.2s | S2C | S2C m446 区域boss进度 [1, 146, "quyuboss_emei1", 20, 160, "ui_jh_001"] |
| 34.2s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 34.2s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 34.2s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 34.2s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei1", 25, 160, "ui_jh_001"] |
| 34.2s | S2C | S2C m24 通用事件[0x3bbbc00701027b] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031302", "徐傠囃辭睼 "] |
| 34.2s | S2C | S2C m18 战斗伤害帧[0x3bbbc00701027b] ["hurt_half", 0, obj(0x700ca8c,0x3b6fb8)] |
| 34.2s | S2C | S2C m24 通用事件[0x3bbbc00701027b] 命中? [] |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] FP=4 |
| 34.2s | S2C | 属性更新0x10[O#0x3b6fb8] SP=4 |
| 34.2s | S2C | 属性更新0x10[O#0x3bbb79] InParry=1 |
| 34.2s | S2C | S2C m24 通用事件[0x3bbb790703dcef] 伤害飘字 [678, 1186, 2, "CS_jh_hyjf07", "SkillNormal", "monm031302"] |
| 34.2s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 34.2s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 34.2s | S2C | 属性更新0x10[O#0x3bbb79] InParry=0 |
| 34.2s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x700ca8c] |
| 34.2s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei1", 30, 160, "ui_jh_001"] |
| 34.2s | S2C | S2C m24 通用事件[0x3bbb790703dcef] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031302", "徐傠囃辭睼 "] |
| 34.2s | S2C | custom0x1E(无法解TVar) |
| 34.3s | C2S | m82 ? [3] |
| 34.4s | S2C | 多属性0x2D(非标子码02) len=122 2d0200efdc030779bb3b000700a00100 |
| 34.7s | S2C | 属性更新0x10[O#0x3b6fb8] FightData=17097474714 |
| 35.0s | C2S | m15 移动上报 手动 state=7 pos=(923.2,48.7,1002.2) |
| 35.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 35.1s | S2C | 对象同步0x21[0x700ca8c] 码181 len=47 |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10299 |
| 35.2s | C2S | m15 移动上报 手动 state=7 pos=(923.2,48.7,1003.1) |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] LastAttackerNpc=monm031302 |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] LastDodge=32.07 |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] SP=5 |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26895 |
| 35.2s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 35.2s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 伤害飘字 [28, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 35.2s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=26909 |
| 35.2s | S2C | S2C m205 伤害目标列表[0x3bbbc00701027b] ["default_normal_skill", obj(0x700ca8c,0x3b6fb8)] |
| 35.2s | S2C | 属性更新0x10[O#0x3bbbc0] State= |
| 35.2s | S2C | 属性更新0x10[O#0x3bbbc0] CurSkillID= |
| 35.2s | S2C | 属性更新0x10[O#0x3bbbc0] CurSkillTarget=None |
| 35.3s | S2C | 对象同步0x21[0x700ca8c] 码234 len=47 |
| 35.5s | S2C | op28(子码47) len=226 ×2 |
| 35.7s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 35.8s | S2C | 对象同步0x21[0x702ade2] 码81 len=91 |
| 36.2s | S2C | 属性更新0x10[O#0x3bbb79] LogicState=0 |
| 36.5s | S2C | op31(sub2) 0x7009041 →0x700c97d |
| 36.7–36.9s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 36.8s | S2C | 对象同步0x21[0x70295df] 码177 len=91 |
| 36.9s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 37.0s | S2C | op2C len=147 2c47920000000a010000000000e00600df950207 |
| 37.1s | S2C | 属性更新0x10[V#0x2] LeftTime=600280 |
| 37.4s | C2S | m15 移动上报 手动 state=7 pos=(923.6,48.7,1013.2) |
| 37.5s | S2C | op28(子码47) len=235 |
| 37.6s | S2C | 对象同步0x21[0x700ca8c] 码0 len=47 |
| 38.3s | C2S | m15 移动上报 自动 state=4 pos=(923.8,48.7,1017.1) |
| 38.5s | S2C | 对象同步0x21[0x700ca8c] 码252 len=47 |
| 38.5s | S2C | op31(sub2) 0x700802b →0x7008316 |
| 38.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 38.7s | C2S | m15 移动上报 自动 state=5 pos=(923.8,48.7,1017.1) ×2 |
| 39.9s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 39.9s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 40.1s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26909 |
| 40.3s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 40.8s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 40.8s | S2C | 属性更新0x10[O#0x150] State= |
| 40.8s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 41.6s | S2C | op31(sub1) 0x7000146 |
| 42.3s | S2C | 属性更新0x10[O#0x3bbbc0] LogicState=0 |
| 42.3s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 43.0s | S2C | 属性更新0x10[O#0x3b6fb8] LiveGroove_4=84786 |
| 43.0s | S2C | 属性更新0x10[O#0x3b6fb8] LiveGroove_5=36650 |
| 43.0s | S2C | S2C m1 系统提示 [1, "8052", 223] |
| 43.0s | S2C | S2C m349 ? [223] |
| 43.0s | S2C | 属性更新0x10[O#0x3b6fb8] Faculty=114285629 |
| 43.0s | S2C | 属性更新0x10[O#0x3b6fb8] HoldFaculty=1502843 |
| 43.1s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 43.3s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 44.4s | S2C | op06 len=1 06 |
| 44.6s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 44.7s | C2S | m15 移动上报 自动 state=5 pos=(923.8,48.7,1017.1) ×2 |
| 44.8s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 45.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=26991 |
| 45.1s | S2C | 属性更新0x10[O#0x3b6fb8] HP=26991 |
| 50.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27073 |
| 50.1s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27073 |
| 50.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 50.8s | C2S | m15 移动上报 自动 state=5 pos=(923.8,48.7,1017.1) ×2 |
| 52.9–54.0s | S2C | 对象同步0x21[0x70295df] 码217 len=91 ×2 |
| 53.5s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 53.5s | S2C | 对象同步0x21[0x702ade2] 码52 len=91 |
| 53.9–54.1s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 54.1s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 54.2s | S2C | op2C len=151 2c47960000000a010000000000e00600df950207 |
| 54.3s | C2S | m82 ? [3] |
| 55.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHPRatio=96 |
| 55.1s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27155 |
| 55.1s | S2C | 属性更新0x10[O#0x3b6fb8] HPRatio=96 |
| 55.1s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27155 |
| 56.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 56.7s | C2S | m15 移动上报 自动 state=5 pos=(923.8,48.7,1017.1) ×2 |
| 59.4s | C2S | m54 格挡状态(值4946488601) |
| 59.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 59.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 59.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 59.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 59.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 59.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 59.4s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 59.6s | C2S | m15 移动上报 手动 state=4 pos=(923.8,48.7,1017.1) |
| 59.6s | S2C | op06 len=1 06 |
| 59.7s | S2C | 对象同步0x21[0x700ca8c] 码252 len=47 |
| 60.1s | C2S | m54 格挡状态(值4722448626) |
| 60.1s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 60.1s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 60.1s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 60.1s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 60.1s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 60.1s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 60.1s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 60.1s | C2S | m54 格挡状态(值4722448626) ×2 |
| 60.2s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27237 |
| 60.2s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27237 |
| 60.2s | C2S | m54 格挡状态(值4722448626) |
| 60.8s | S2C | op31(sub1) 0x70295df |
| 62.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 62.8s | C2S | m15 移动上报 自动 state=5 pos=(923.8,48.7,1017.1) ×2 |
| 62.8s | S2C | op31(sub1) 0x702ade2 |
| 63.2s | S2C | 记录写格0x13[Count_Limit_Form_Rec#363 行5 列1 =33555712 0x0] |
| 63.2s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 63.2s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 63.2s | S2C | S2C m252 ? [] |
| 63.2s | S2C | S2C m250 ? [] ×3 |
| 64.3s | C2S | m54 格挡状态(值4946488601) |
| 64.3s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 64.3s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 64.3s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 64.3s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 64.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 64.3s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 64.3s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 64.6s | C2S | m54 格挡状态(值4722448626) |
| 64.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 64.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 64.7s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 64.7s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 64.7s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 64.7s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 64.7s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 64.7s | C2S | m54 格挡状态(值4722448626) ×3 |
| 65.2s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27319 |
| 65.2s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27319 |
| 65.2s | C2S | m15 移动上报 自动 state=8 pos=(923.8,48.7,1017.1) |
| 65.3s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 65.3s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 65.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 65.4s | S2C | 对象同步0x21[0x700ca8c] 码252 len=47 |
| 65.4–65.7s | C2S | m15 移动上报 手动 state=6 pos=(923.8,49.5,1017.1) ×2 |
| 65.6–69.4s | S2C | 对象同步0x21[0x700ca8c] 码89 len=47 ×3 |
| 65.8–66.4s | S2C | 对象同步0x21[0x700ca8c] 码122 len=47 ×2 |
| 65.8s | C2S | m54 格挡状态(值4946488601) |
| 65.8s | S2C | op31(sub1) 0x7009f56 |
| 65.9s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 65.9s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 65.9s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 65.9s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 65.9s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 65.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 65.9s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 65.9s | C2S | m15 移动上报 手动 state=6 pos=(923.2,50.2,1018.3) |
| 66.0s | S2C | 属性更新0x10[O#0x150] State=stand ×2 |
| 66.0s | S2C | op2C len=143 2c478e0000000a010000000000a006008cca0007 |
| 66.0s | S2C | 多属性0x2D[0x1500700](布局变体) len=23 2d01004b010007500100000100540004 |
| 66.4s | C2S | m15 移动上报 自动 state=4 pos=(923.2,48.7,1018.3) |
| 66.6s | C2S | m54 格挡状态(值4722448626) |
| 66.6s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 66.6s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 66.6s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 66.6s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 66.6s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 66.6s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 66.6s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 66.6s | C2S | m54 格挡状态(值4722448626) ×2 |
| 66.7s | C2S | m15 移动上报 手动 state=7 pos=(923.2,48.7,1018.3) |
| 66.7s | C2S | m54 格挡状态(值4722448626) |
| 66.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 66.9s | S2C | 对象同步0x21[0x700ca8c] 码0 len=47 |
| 66.9s | C2S | m15 移动上报 手动 state=8 pos=(922.0,48.7,1017.6) |
| 67.1s | S2C | 对象同步0x21[0x700ca8c] 码226 len=47 |
| 67.3s | C2S | m15 移动上报 手动 state=6 pos=(920.6,50.1,1016.9) |
| 67.5s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [919.808, 50.292, 1016.388, -2.091] |
| 67.5s | S2C | 对象同步0x21[0x700ca8c] 码200 len=47 |
| 67.6s | C2S | m15 移动上报 手动 state=6 pos=(919.3,50.2,1016.1) |
| 67.6s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 67.6s | S2C | 位置0x1F(短len=25) |
| 67.6s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [919.78, 50.261, 1016.372, 4.583] |
| 67.6s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_hyjf07 |
| 67.6s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 冲锋动画 [914.78, 44.886, 1015.724, 2, "qg_fastruning_s_2h_01"] |
| 67.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 67.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 67.6s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +97行 |
| 67.6s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x700ca8c] ×4 |
| 67.6s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 67.6s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 67.6s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 67.6s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 67.6s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10298 |
| 67.6s | S2C | S2C m707 受理族变体 [15] |
| 67.7s | S2C | 对象同步0x21[0x700ca8c] 码239 len=47 |
| 67.7s | C2S | m15 移动上报 自动 state=4 pos=(919.8,50.2,1016.4) |
| 67.8s | S2C | op28(子码47) len=235 |
| 67.9s | S2C | 对象同步0x21[0x700ca8c] 码230 len=47 |
| 68.0s | C2S | m212 冲锋/续招落地 [914.802, 44.91, 1015.727, -1.7] |
| 68.1s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 68.1s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 68.1s | S2C | 位置0x1F(短len=25) |
| 68.1s | S2C | S2C m26 技能释放 [] |
| 68.1s | S2C | 位置0x1F(短len=25) |
| 68.1s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [914.802, 44.91, 1015.727, 4.583] |
| 68.1s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_jh_hyjf07", 0, 0, 0.0, 913.293, 43.886] |
| 68.1s | C2S | m15 移动上报 手动 state=8 pos=(914.8,44.9,1015.7) |
| 68.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 68.1s | C2S | m15 移动上报 自动 state=4 pos=(914.8,44.9,1015.7) |
| 68.1s | C2S | m15 移动上报 自动 state=7 pos=(914.8,44.9,1015.7) |
| 68.1s | S2C | 对象同步0x21[0x700ca8c] 码235 len=47 |
| 68.2s | S2C | S2C m586 状态推送(簇首) [5] |
| 68.5s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 68.5s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 68.5s | S2C | 属性更新0x10[O#0x3b6fb8] LastNGAttackNpc=monm031300 |
| 68.5s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 68.5s | S2C | 属性更新0x10[O#0x3bb6fd] LogicState=1 |
| 68.5s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 68.5s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 68.5s | S2C | 属性更新0x10[O#0x3bb6fd] CurSkillID=default_normal_skill |
| 68.5s | S2C | 属性更新0x10[O#0x3bb6fd] CurSkillTarget=None |
| 68.5s | S2C | 位置0x1F(短len=25) |
| 68.5s | S2C | S2C m337 施法回显[0x3bb6fd0701e430] [913.113, 43.886, 1014.283, 7.147, 0] |
| 68.5s | S2C | S2C m20 范围施放标记[0x3bb6fd0701e430] ["atkhit3", obj(0x700ca8c,0x3b6fb8), 0, 0.0] |
| 68.5s | S2C | S2C m24 通用事件[0x3bb6fd0701e430] 伤害飘字 [678, 1209, 2, "CS_jh_hyjf07", "SkillNormal", "monm031300"] |
| 68.5s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 68.5s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 68.5s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 68.5s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei1", 35, 160, "ui_jh_001"] |
| 68.5s | S2C | S2C m24 通用事件[0x3bb6fd0701e430] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031300", "徐傠囃辭睼 "] |
| 68.5s | S2C | S2C m18 战斗伤害帧[0x3bb6fd0701e430] ["hurt_half", 0, obj(0x700ca8c,0x3b6fb8)] |
| 68.5s | S2C | S2C m24 通用事件[0x3bb6fd0701e430] 命中? [] |
| 68.5s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jh_hyjf07", obj(0x701e430,0x3bb6fd)] |
| 68.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 68.6s | S2C | 对象同步0x21[0x701e430] 码59 len=223 |
| 68.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 68.7s | S2C | 多属性0x2D[0xb6fd0701](布局变体) len=59 2d010030e40107fdb63b000600d70100 |
| 68.7s | C2S | m15 移动上报 自动 state=5 pos=(914.8,44.4,1015.7) ×2 |
| 68.8s | S2C | op31(sub1) 0x7009f56 |
| 69.2s | C2S | m15 移动上报 自动 state=8 pos=(914.8,44.4,1015.7) |
| 69.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 69.3s | C2S | m54 格挡状态(值4722448626) |
| 69.5s | S2C | 属性更新0x10[O#0x3b6fb8] LastAttackerNpc=monm031300 |
| 69.5s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] c1=7 [1, obj(0x701e430,0x3bb6fd), "default_normal_skill", "徐傠囃辭睼 ", "monm031300", 100] |
| 69.5s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 69.5s | S2C | 属性更新0x10[O#0x3bb6fd] State= |
| 69.5s | S2C | 属性更新0x10[O#0x3bb6fd] CurSkillID= |
| 69.5s | S2C | 属性更新0x10[O#0x3bb6fd] CurSkillTarget=None |
| 69.6s | C2S | m15 移动上报 自动 state=4 pos=(914.8,43.9,1015.7) |
| 69.8s | S2C | 对象同步0x21[0x700ca8c] 码88 len=47 |
| 69.9s | S2C | op28(子码47) len=226 |
| 71.1s | S2C | 多属性0x2D[0xcd530702](布局变体) len=220 2d01009842020753cd3b001000b50004 |
| 71.9s | S2C | op28(子码47) len=233 |
| 72.2s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 72.2s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 74.3s | C2S | m82 ? [3] |
| 74.4s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 74.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 74.8s | C2S | m15 移动上报 自动 state=5 pos=(914.8,43.9,1015.7) ×2 |
| 74.8s | S2C | op06 len=1 06 |
| 76.5s | S2C | 属性更新0x10[O#0x3bb6fd] LogicState=0 |
| 76.5s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 77.5s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 78.0s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 79.0s | S2C | op28(子码47) len=230 ×2 |
| 80.2s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27401 |
| 80.2s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27401 |
| 80.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 80.8s | C2S | m15 移动上报 自动 state=5 pos=(914.8,43.9,1015.7) ×2 |
| 82.2s | S2C | 属性更新0x10[O#0x14b] State=stand ×2 |
| 82.2s | S2C | op2C len=157 2c479c0000000a010000000000800600df950207 |
| 82.2s | S2C | 多属性0x2D[0x14b0700](布局变体) len=23 2d0100460100074b0100000100540004 |
| 82.9s | S2C | S2C m758 玩家名片 [6, "虎皮喵大人 ", "2F8CAB039C8E458EA7FB69E091E5C2A1", 6, 2, " "] |
| 83.3s | S2C | 对象同步0x21[0x702ade2] 码97 len=91 |
| 83.3s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 83.7s | S2C | 对象同步0x21[0x70295df] 码149 len=91 |
| 83.7–84.5s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 84.5s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 84.6s | S2C | op2C len=136 2c47870000000a010000000000e00600df950207 |
| 85.2s | S2C | 属性更新0x10[O#0x3b6fb8] HitHPRatio=97 |
| 85.2s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27483 |
| 85.2s | S2C | 属性更新0x10[O#0x3b6fb8] HPRatio=97 |
| 85.2s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27483 |
| 86.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 86.8s | C2S | m15 移动上报 自动 state=5 pos=(914.8,43.9,1015.7) ×2 |
| 89.5s | S2C | 对象同步0x21[0x7000146] 码237 len=223 |
| 89.5s | S2C | 属性更新0x10[O#0x14b] State= |
| 89.5s | S2C | 多属性0x2D[0x14b0700](布局变体) len=20 2d0100460100074b0100000100540001 |
| 90.1s | S2C | op06 len=1 06 |
| 90.2s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27565 |
| 90.2s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27565 |
| 91.2s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 91.2s | S2C | 属性更新0x10[O#0x150] State= |
| 91.2s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 92.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 92.8s | C2S | m15 移动上报 自动 state=5 pos=(914.8,43.9,1015.7) ×2 |
| 94.2s | S2C | op0E len=9 0e7b020107c0bb3b00 |
| 94.3s | C2S | m82 ? [3] |
| 94.5s | S2C | op0E len=9 0eefdc030779bb3b00 |
| 94.8s | C2S | m15 移动上报 自动 state=7 pos=(914.8,43.9,1015.7) |
| 94.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 94.8s | C2S | m15 移动上报 自动 state=7 pos=(914.8,43.9,1015.7) |
| 95.1s | S2C | 对象同步0x21[0x700ca8c] 码24 len=91 |
| 95.1s | C2S | m54 格挡状态(值4946488601) |
| 95.1s | S2C | op31(sub1) 0x7009041 |
| 95.1s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 95.1s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 95.1s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 95.1s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 95.1s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 95.1s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 95.1s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 95.1s | C2S | m15 移动上报 自动 state=7 pos=(916.4,43.9,1015.7) |
| 95.3s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27647 |
| 95.3s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27647 |
| 95.3s | C2S | m15 移动上报 ?1000 state=7 pos=(916.7,43.9,1015.7) |
| 95.3s | S2C | 对象同步0x21[0x700ca8c] 码23 len=47 |
| 95.4s | C2S | m54 格挡状态(值4722448626) |
| 95.4s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 95.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 95.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 95.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 95.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 95.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 95.4s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 95.4s | C2S | m54 格挡状态(值4722448626) ×3 |
| 95.5s | S2C | 对象同步0x21[0x700ca8c] 码30 len=47 |
| 95.5s | C2S | m15 移动上报 ?1000 state=7 pos=(916.5,43.9,1015.9) ×2 |
| 95.8s | S2C | 对象同步0x21[0x700ca8c] 码62 len=91 |
| 97.1s | S2C | 属性更新0x10[V#0x2] LeftTime=600220 |
| 97.1s | S2C | op31(sub3) 0x7005da7 →0x7022212 |
| 97.3s | S2C | 对象同步0x21[0x7000146] 码133 len=223 |
| 97.4–98.0s | C2S | m15 移动上报 自动 state=4 pos=(915.2,43.9,1024.6) ×3 |
| 97.7–98.1s | S2C | 对象同步0x21[0x700ca8c] 码209 len=91 ×2 |
| 98.4s | C2S | m15 移动上报 自动 state=7 pos=(915.2,43.9,1024.6) |
| 98.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 98.5s | C2S | m15 移动上报 自动 state=7 pos=(915.2,43.9,1024.6) |
| 98.5s | S2C | 对象同步0x21[0x700ca8c] 码0 len=91 |
| 98.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 98.8s | C2S | m15 移动上报 自动 state=5 pos=(914.4,43.9,1023.4) ×2 |
| 98.8s | C2S | m15 移动上报 自动 state=4 pos=(914.2,43.9,1023.2) |
| 98.8s | C2S | m15 移动上报 自动 state=7 pos=(914.2,43.9,1023.2) ×2 |
| 98.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 98.9s | C2S | m15 移动上报 手动 state=7 pos=(914.3,43.9,1022.8) |
| 99.0s | S2C | 对象同步0x21[0x700ca8c] 码4 len=179 |
| 99.1s | C2S | m15 移动上报 自动 state=4 pos=(914.9,43.9,1023.6) |
| 99.1s | S2C | op28(子码47) len=230 ×2 |
| 99.1–100.2s | C2S | m107 外观自定义保存 ["e$"] ×2 |
| 99.2s | S2C | 对象同步0x21[0x700ca8c] 码183 len=47 |
| 100.3s | S2C | 属性更新0x10[O#0x3b6fb8] HitHPRatio=98 |
| 100.3s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27729 |
| 100.3s | S2C | 属性更新0x10[O#0x3b6fb8] HPRatio=98 |
| 100.3s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27729 |
| 101.1s | C2S | m15 移动上报 手动 state=7 pos=(914.9,43.9,1023.6) |
| 101.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 101.3s | S2C | 对象同步0x21[0x700ca8c] 码164 len=47 |
| 101.6s | C2S | m54 格挡状态(值4946488601) |
| 101.7s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 101.7s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 101.7s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 101.7s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 101.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 101.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 101.7s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 101.7s | C2S | m15 移动上报 手动 state=7 pos=(916.6,43.9,1025.6) ×2 |
| 101.8s | S2C | 对象同步0x21[0x700ca8c] 码56 len=47 |
| 101.8s | C2S | m54 格挡状态(值4722448626) |
| 101.9s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 101.9s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 101.9s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 101.9s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 101.9s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 101.9s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 101.9s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 101.9s | C2S | m54 格挡状态(值4722448626) ×2 |
| 102.0s | C2S | m15 移动上报 手动 state=7 pos=(917.0,43.9,1025.9) |
| 102.0s | C2S | m54 格挡状态(值4722448626) |
| 102.2s | S2C | 对象同步0x21[0x700ca8c] 码52 len=47 |
| 102.2s | S2C | op31(sub1) 0x7024298 |
| 102.3s | C2S | m15 移动上报 自动 state=4 pos=(918.3,43.9,1026.9) |
| 102.4–103.9s | S2C | 对象同步0x21[0x700ca8c] 码76 len=47 ×3 |
| 102.6s | S2C | 对象同步0x21[0x70295df] 码177 len=91 |
| 102.9s | S2C | S2C m566 令牌应答串 ["K/tU0Q3gE3fpipmku15rbdjOJsTnRJwJ6nS7iSiK"] |
| 103.0s | C2S | m15 移动上报 自动 state=8 pos=(918.3,43.9,1026.9) |
| 103.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 103.2s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 103.8s | C2S | m15 移动上报 自动 state=6 pos=(918.3,44.9,1026.9) |
| 104.0s | C2S | m15 移动上报 自动 state=4 pos=(918.3,43.9,1026.9) |
| 104.1s | S2C | 对象同步0x21[0x700ca8c] 码75 len=47 |
| 104.2s | S2C | op28(子码47) len=222 |
| 104.4s | C2S | 对象请求 len=33 |
| 104.4s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 104.4s | C2S | m431 选择目标 [obj(0x7006822,0x3bb9f8)] |
| 104.4s | C2S | m211 使用技能(出招) wuji_CS_jz_lyfh05 14值 [918.333, 43.887, 1026.871, 0.965] |
| 104.4s | C2S | m15 移动上报 自动 state=7 pos=(918.3,43.9,1026.9) ×2 |
| 104.5s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 104.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 104.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=wuji_CS_jz_lyfh05 |
| 104.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 104.5s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 104.5s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill=wuji_CS_jz_lyfh05 |
| 104.5s | S2C | S2C m707 受理族变体 [15] |
| 104.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 104.5s | C2S | m220 冲锋接近上报 [918.971, 43.887, 1027.079, 1.257] |
| 104.6s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 冲锋动画 [923.054, 44.886, 1028.405, 1, "qg_fastrun_2h_01"] |
| 104.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 104.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 104.6s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +97行 |
| 104.6s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x700ca8c] ×4 |
| 104.6s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 104.6s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 104.6s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 104.6s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 104.6s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10264 |
| 104.6s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=155 |
| 104.6s | C2S | m15 移动上报 自动 state=4 pos=(919.2,43.9,1027.1) |
| 104.6s | S2C | 对象同步0x21[0x700ca8c] 码201 len=135 |
| 104.6s | C2S | m15 移动上报 自动 state=7 pos=(919.2,43.9,1027.1) ×2 |
| 104.7s | C2S | m212 冲锋/续招落地 [922.863, 43.886, 1028.343, 1.257] |
| 104.7s | C2S | m15 移动上报 自动 state=4 pos=(922.9,43.9,1028.3) |
| 104.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 104.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 104.7s | S2C | 位置0x1F(短len=25) |
| 104.7s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill= |
| 104.7s | S2C | S2C m26 技能释放 [] |
| 104.7s | S2C | 属性更新0x10[O#0x3bb9f8] LogicState=1 |
| 104.7s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 104.7s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 104.7s | S2C | 位置0x1F(短len=25) |
| 104.7s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [922.863, 43.886, 1028.343, 1.257, 0] |
| 104.7s | S2C | 属性更新0x10[O#0x3b6fb8] LastWGAttackNpc=monm031301 |
| 104.7s | S2C | S2C m24 通用事件[0x3bb9f807006822] 动作开始 [1, obj(0x700ca8c,0x3b6fb8)] |
| 104.7s | S2C | S2C m19 锁定伤害段[0x3b6fb80700ca8c] [obj(0x7006822,0x3bb9f8), "wuji_CS_jz_lyfh05", 1500, 1500, 923.12, 43.886] |
| 104.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 104.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 104.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 104.7s | C2S | m15 移动上报 自动 state=5 pos=(922.9,43.9,1028.3) ×2 |
| 104.8s | C2S | m15 移动上报 自动 state=7 pos=(922.9,43.9,1028.3) |
| 104.8s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 104.8s | S2C | op2C len=171 2c47aa0000000a010000000000a006008cca0007 |
| 104.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 105.1s | S2C | 对象同步0x21[0x700ca8c] 码60 len=47 |
| 105.1s | S2C | 多属性0x2D[0xb9f80700](布局变体) len=23 2d010022680007f8b93b000100a0018c |
| 105.2s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=165 |
| 105.3s | S2C | S2C m169 buff添加 ["eqp_wp_06a", obj(0x7006822,0x3bb9f8)] |
| 105.3s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 105.3s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 105.3s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 105.3s | S2C | S2C m1 系统提示 [2, "8066"] |
| 105.3s | S2C | S2C m24 通用事件[0x3bb9f807006822] 伤害飘字 [1022, 0, 0, "wuji_CS_jz_lyfh05", "SkillLock", "monm031301"] |
| 105.3s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["wuji_CS_jz_lyfh05", obj(0x7006822,0x3bb9f8)] |
| 105.3s | C2S | m15 移动上报 自动 state=7 pos=(923.1,43.9,1028.4) |
| 105.4s | S2C | op06 len=1 06 |
| 105.5s | S2C | 对象同步0x21[0x700ca8c] 码15 len=47 |
| 105.5s | S2C | 多属性0x2D[0xb9f80700](布局变体) len=67 2d010022680007f8b93b000300770224 |
| 105.8s | S2C | 对象同步0x21[0x70295df] 码177 len=91 |
| 105.8s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 105.9s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 105.9s | S2C | 属性更新0x10[O#0x3b6fb8] FP=4 |
| 105.9s | S2C | 属性更新0x10[O#0x3b6fb8] SP=4 |
| 105.9s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 105.9s | S2C | S2C m24 通用事件[0x3bb9f807006822] 伤害飘字 [1022, 0, 0, "wuji_CS_jz_lyfh05", "SkillLock", "monm031301"] |
| 105.9s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [2] |
| 105.9s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["wuji_CS_jz_lyfh05", obj(0x7006822,0x3bb9f8)] |
| 105.9s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 105.9s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 105.9s | S2C | S2C m189 对象通知 [obj(0x7006822,0x3bb9f8)] |
| 105.9s | S2C | op2C len=147 2c47920000000a010000000000e00600df950207 |
| 106.2s | S2C | 多属性0x2D[0xb9f80700](布局变体) len=25 2d010022680007f8b93b000200d70108 |
| 106.2s | C2S | m914 招式结束异步位置 [922.928, 43.886, 1028.362, 1.257, 12, 29] |
| 106.2s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 106.2s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 106.2s | S2C | op28(子码47) len=222 |
| 106.3s | C2S | m54 格挡状态(值4946488601) |
| 106.3s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 106.3s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 106.3s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 106.3s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 106.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 106.3s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 106.3s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 106.4s | S2C | 对象同步0x21[0x700ca8c] 码99 len=91 |
| 106.8s | C2S | m54 格挡状态(值4722448626) |
| 106.8s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 106.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 106.8s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 106.8s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 106.8s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 106.8s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 106.8s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 106.8s | C2S | m54 格挡状态(值4722448626) ×2 |
| 107.3s | C2S | m15 移动上报 手动 state=7 pos=(922.9,43.9,1028.4) |
| 107.3s | S2C | S2C m169 buff移除 ["eqp_wp_06a", obj(0x7006822,0x3bb9f8)] |
| 107.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 107.5s | S2C | 对象同步0x21[0x700ca8c] 码124 len=47 |
| 107.5s | C2S | m15 移动上报 手动 state=7 pos=(921.9,43.9,1028.4) |
| 107.7s | S2C | 属性更新0x10[O#0x3bb9f8] CurSkillID=default_normal_skill |
| 107.7s | S2C | 属性更新0x10[O#0x3bb9f8] CurSkillTarget=None |
| 107.7s | S2C | 位置0x1F(短len=25) |
| 107.7s | S2C | S2C m337 施法回显[0x3bb9f807006822] [924.481, 43.886, 1028.868, 4.549, 0] |
| 107.7s | S2C | S2C m20 范围施放标记[0x3bb9f807006822] ["atkhit3", obj(0x700ca8c,0x3b6fb8), 0, 0.0] |
| 107.8s | S2C | 对象同步0x21[0x700ca8c] 码146 len=135 |
| 107.8s | S2C | 多属性0x2D[0xb9f80700](布局变体) len=20 2d010022680007f8b93b000100770201 |
| 108.3s | S2C | op31(sub2) 0x7022212 →0x700fed1 |
| 108.3s | S2C | op28(子码47) len=221 ×3 |
| 108.8s | S2C | 属性更新0x10[O#0x3b6fb8] LastAttackerNpc=monm031301 |
| 108.8s | S2C | 属性更新0x10[O#0x3b6fb8] LastDodge=32.29 |
| 108.8s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] c1=7 [1, obj(0x7006822,0x3bb9f8), "default_normal_skill", "徐傠囃辭睼 ", "monm031301", 100] |
| 108.8s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 108.8s | S2C | 属性更新0x10[O#0x3bb9f8] State= |
| 108.8s | S2C | 属性更新0x10[O#0x3bb9f8] CurSkillID= |
| 108.8s | S2C | 属性更新0x10[O#0x3bb9f8] CurSkillTarget=None |
| 108.8s | S2C | S2C m189 对象通知 [obj(0x7006822,0x3bb9f8)] |
| 108.9s | C2S | m15 移动上报 自动 state=4 pos=(915.9,43.9,1028.9) |
| 109.0s | S2C | 对象同步0x21[0x700ca8c] 码224 len=47 |
| 109.1s | C2S | m15 移动上报 手动 state=7 pos=(915.9,43.9,1028.9) |
| 109.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 109.2s | C2S | m15 移动上报 自动 state=4 pos=(916.3,43.9,1028.9) |
| 109.2s | S2C | 对象同步0x21[0x700ca8c] 码199 len=91 |
| 109.2s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 109.2s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10364 |
| 109.2s | C2S | m15 移动上报 自动 state=4 pos=(916.3,43.9,1028.9) |
| 109.3s | S2C | op31(sub1) 0x7036191 |
| 109.5s | S2C | 对象同步0x21[0x700ca8c] 码34 len=47 |
| 109.5s | C2S | m211 使用技能(出招) CS_jz_lyfh02 14值 [916.283, 43.887, 1028.857, 1.645] |
| 109.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 109.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jz_lyfh02 |
| 109.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 109.5s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 109.5s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill=CS_jz_lyfh02 |
| 109.5s | S2C | S2C m707 受理族变体 [15] |
| 109.5s | C2S | m15 移动上报 自动 state=7 pos=(916.3,43.9,1028.9) ×2 |
| 109.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 109.7–110.1s | S2C | 对象同步0x21[0x700ca8c] 码201 len=91 ×2 |
| 109.9s | C2S | m220 冲锋接近上报 [918.308, 43.887, 1028.861, 1.57] |
| 110.0s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 冲锋动画 [922.981, 44.886, 1028.866, 1, "qg_fastrun_2h_01"] |
| 110.0s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 110.0s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 110.0s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 110.0s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 110.0s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 110.0s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =16802048 0x700ca8c] ×2 |
| 110.0s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 110.0s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10240 |
| 110.0s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=155 |
| 110.0s | C2S | m15 移动上报 自动 state=4 pos=(918.5,43.9,1028.9) |
| 110.1s | C2S | m15 移动上报 自动 state=7 pos=(918.5,43.9,1028.9) ×2 |
| 110.2s | C2S | m212 冲锋/续招落地 [922.139, 43.886, 1028.866, 1.57] |
| 110.3s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 110.3s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 110.3s | S2C | 位置0x1F(短len=25) |
| 110.3s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill= |
| 110.3s | S2C | S2C m26 技能释放 [] |
| 110.3s | S2C | 位置0x1F(短len=25) |
| 110.3s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [922.139, 43.886, 1028.866, 1.57, 0] |
| 110.3s | S2C | S2C m24 通用事件[0x3bb9f807006822] 动作开始 [1, obj(0x700ca8c,0x3b6fb8)] |
| 110.3s | S2C | S2C m19 锁定伤害段[0x3b6fb80700ca8c] [obj(0x7006822,0x3bb9f8), "wuji_CS_jz_lyfh02", 1066, 1066, 922.139, 43.886] |
| 110.3s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 110.3s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 110.3s | C2S | m15 移动上报 自动 state=4 pos=(922.1,43.9,1028.9) |
| 110.3s | S2C | 对象同步0x21[0x700ca8c] 码233 len=47 |
| 110.3s | C2S | m15 移动上报 自动 state=7 pos=(922.1,43.9,1028.9) |
| 110.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 110.5s | S2C | 对象同步0x21[0x700ca8c] 码44 len=47 |
| 110.7s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 110.7s | S2C | 属性更新0x10[O#0x3b6fb8] FP=6 |
| 110.7s | S2C | 属性更新0x10[O#0x3b6fb8] SP=6 |
| 110.7s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 110.7s | S2C | S2C m24 通用事件[0x3bb9f807006822] 伤害飘字 [188, 824, 0, "CS_jz_lyfh02", "SkillLock", "monm031301"] |
| 110.7s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 110.7s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 110.7s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 110.7s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei1", 42, 160, "ui_jh_001"] |
| 110.7s | S2C | S2C m24 通用事件[0x3bb9f807006822] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031301", "徐傠囃辭睼 "] |
| 110.7s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jz_lyfh02", obj(0x7006822,0x3bb9f8)] |
| 110.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 110.8s | C2S | m15 移动上报 自动 state=5 pos=(922.2,43.9,1028.9) ×2 |
| 110.9s | C2S | m15 移动上报 自动 state=7 pos=(922.2,43.9,1028.9) |
| 111.0s | S2C | 对象同步0x21[0x700ca8c] 码105 len=47 |
| 111.0s | S2C | 多属性0x2D[0xb9f80700](布局变体) len=59 2d010022680007f8b93b000600d70100 |
| 111.3s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=165 |
| 111.3s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 111.3s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 111.3s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 111.3s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 111.3s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 111.3s | S2C | op28(子码47) len=222 ×3 |
| 111.7s | C2S | m914 招式结束异步位置 [922.048, 43.886, 1028.866, 1.57, 12, 30] |
| 111.7s | C2S | m990 ? [obj(0x7006822,0x3bb9f8), 924.481, 43.886, 1028.868] |
| 111.9s | C2S | m15 移动上报 手动 state=7 pos=(922.0,43.9,1028.9) |
| 111.9s | S2C | 对象同步0x21[0x700ca8c] 码22 len=91 |
| 112.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 112.0s | S2C | 对象同步0x21[0x700ca8c] 码47 len=47 |
| 112.1s | C2S | m15 移动上报 手动 state=7 pos=(921.0,43.9,1028.9) |
| 112.3s | S2C | 对象同步0x21[0x700ca8c] 码0 len=47 |
| 112.4s | S2C | op31(sub2) 0x700014b →0x700fed1 |
| 112.7s | S2C | 属性更新0x10[O#0x3bb9f8] LogicState=0 |
| 112.9s | C2S | m15 移动上报 自动 state=4 pos=(917.4,43.9,1029.2) |
| 113.1s | S2C | 对象同步0x21[0x700ca8c] 码138 len=47 |
| 113.3s | S2C | op31(sub1) 0x7022212 |
| 113.7s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 113.9s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 113.9s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10340 |
| 114.3s | C2S | m82 ? [3] |
| 114.4s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 114.4s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 115.3s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27811 |
| 115.3s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27811 |
| 115.6s | S2C | op0E len=9 0e22680007f8b93b00 |
| 115.6s | C2S | m440 ? [] |
| 115.6s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 115.6s | C2S | m431 选择目标 [0] |
| 115.7s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 116.4s | S2C | 对象同步0x21[0x7000146] 码115 len=223 |
| 116.7s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 116.8s | C2S | m15 移动上报 自动 state=5 pos=(917.4,43.9,1029.2) ×2 |
| 118.6s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 118.6s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 120.3s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27893 |
| 120.3s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27893 |
| 120.5s | S2C | op06 len=1 06 |
| 121.7–122.8s | S2C | 对象同步0x21[0x70295df] 码217 len=91 ×2 |
| 122.4s | S2C | 对象同步0x21[0x702ade2] 码52 len=91 |
| 122.4s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 122.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 122.8s | C2S | m15 移动上报 自动 state=5 pos=(917.4,43.9,1029.2) ×2 |
| 122.8–123.1s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 123.1s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 123.2s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 123.2s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 123.2s | S2C | S2C m250 ? [] ×2 |
| 123.3s | S2C | op2C len=139 2c478a0000000a010000000000e00600df950207 |
| 124.5s | S2C | 多属性0x2D[0xce360700](布局变体) len=217 2d01005d83000736ce3b001000b50004 |
| 125.4s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=27975 |
| 125.4s | S2C | 属性更新0x10[O#0x3b6fb8] HP=27975 |
| 125.5s | S2C | op28(子码47) len=255 ×2 |
| 126.2s | C2S | m15 移动上报 自动 state=7 pos=(917.4,43.9,1029.2) |
| 126.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 126.3s | C2S | m15 移动上报 自动 state=7 pos=(917.4,43.9,1029.2) ×2 |
| 126.5s | S2C | 对象同步0x21[0x700ca8c] 码128 len=135 |
| 126.5s | S2C | op28(子码47) len=230 |
| 126.6s | C2S | m15 移动上报 手动 state=7 pos=(918.0,43.9,1029.5) |
| 126.7s | S2C | 对象同步0x21[0x700ca8c] 码244 len=47 |
| 127.5s | C2S | m15 移动上报 手动 state=8 pos=(922.2,43.9,1029.2) |
| 127.7s | S2C | 对象同步0x21[0x700ca8c] 码13 len=47 |
| 127.9s | C2S | m15 移动上报 手动 state=6 pos=(923.7,45.2,1029.1) |
| 127.9s | S2C | 对象同步0x21[0x700ca8c] 码121 len=47 |
| 128.0s | C2S | m15 移动上报 手动 state=8 pos=(924.4,45.4,1029.0) |
| 128.1s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=158 |
| 128.1s | S2C | 属性更新0x10[O#0x3b6fb8] JumpTimes=1 |
| 128.2s | S2C | 对象同步0x21[0x700ca8c] 码165 len=47 |
| 128.2–128.5s | C2S | m15 移动上报 手动 state=6 pos=(925.3,47.4,1029.0) ×2 |
| 128.4s | S2C | 对象同步0x21[0x700ca8c] 码53 len=47 |
| 128.5s | S2C | op31(sub1) 0x7000146 |
| 128.5s | S2C | op28(子码47) len=222 |
| 128.6s | S2C | 对象同步0x21[0x700ca8c] 码93 len=47 |
| 128.6s | C2S | 对象请求 len=33 |
| 128.6s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 128.6s | C2S | m431 选择目标 [obj(0x7024885,0x3bbc11)] |
| 128.6s | C2S | m211 使用技能(出招) wuji_CS_jz_lyfh05 14值 [926.882, 49.875, 1028.841, 1.645] |
| 128.7s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 128.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 128.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=wuji_CS_jz_lyfh05 |
| 128.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurIsAirSkill=4 |
| 128.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 128.7s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 128.7s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill=wuji_CS_jz_lyfh05 |
| 128.7s | S2C | S2C m707 受理族变体 [15] |
| 128.7s | C2S | m220 冲锋接近上报 [927.557, 50.576, 1028.792, 1.645] |
| 128.7s | S2C | op0E len=9 0e30e40107fdb63b00 |
| 128.8s | C2S | m15 移动上报 手动 state=6 pos=(927.7,50.7,1028.8) |
| 128.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 128.8s | C2S | m15 移动上报 手动 state=5 pos=(927.8,50.8,1028.8) ×2 |
| 128.8s | S2C | 对象同步0x21[0x700ca8c] 码49 len=47 |
| 128.8s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 冲锋动画 [934.378, 44.886, 1028.837, 2, "qg_fastruning_s_2h_01"] |
| 128.8s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 128.8s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 128.8s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +97行 |
| 128.8s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x700ca8c] ×4 |
| 128.8s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 128.8s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 128.8s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 128.8s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 128.8s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10264 |
| 128.8s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=148 |
| 128.8s | S2C | 属性更新0x10[O#0x3b6fb8] JumpTimes=0 |
| 128.8s | C2S | m220 冲锋接近上报 [928.007, 50.92, 1028.759, 1.645] |
| 128.9s | S2C | S2C m707 受理族变体 [29] |
| 129.0s | C2S | m212 冲锋/续招落地 [934.029, 45.223, 1028.833, 1.558] |
| 129.1s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 129.1s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 129.1s | S2C | 位置0x1F(短len=25) |
| 129.1s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill= |
| 129.1s | S2C | S2C m26 技能释放 [] |
| 129.1s | S2C | 属性更新0x10[O#0x3bbc11] LogicState=1 |
| 129.1s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 129.1s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 129.1s | S2C | 位置0x1F(短len=25) |
| 129.1s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [934.029, 45.223, 1028.833, 1.566, 0] |
| 129.1s | S2C | S2C m24 通用事件[0x3bbc1107024885] 动作开始 [1, obj(0x700ca8c,0x3b6fb8)] |
| 129.1s | S2C | S2C m19 锁定伤害段[0x3b6fb80700ca8c] [obj(0x7024885,0x3bbc11), "wuji_CS_jz_lyfh05", 1500, 1500, 934.299, 45.223] |
| 129.1s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 129.1s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 129.1s | C2S | m15 移动上报 手动 state=8 pos=(934.4,44.9,1028.8) |
| 129.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 129.1s | C2S | m15 移动上报 手动 state=7 pos=(934.0,45.2,1028.8) |
| 129.1s | S2C | 位置0x1F(短len=25) |
| 129.3s | S2C | 对象同步0x21[0x7024885] 码49 len=135 |
| 129.5s | S2C | 多属性0x2D[0xbc110702](布局变体) len=23 2d01008548020711bc3b000100a0018c |
| 129.5s | S2C | op31(sub3) 0x700d716 →0x70095b5 |
| 129.5s | S2C | op28(子码47) len=222 |
| 129.6s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=158 |
| 129.6s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 129.6s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 129.6s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 129.6s | S2C | S2C m24 通用事件[0x3bbc1107024885] 伤害飘字 [1028, 0, 0, "wuji_CS_jz_lyfh05", "SkillLock", "monm031301"] |
| 129.6s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["wuji_CS_jz_lyfh05", obj(0x7024885,0x3bbc11)] |
| 129.7s | C2S | m15 移动上报 手动 state=7 pos=(934.3,45.2,1028.8) |
| 129.7s | S2C | 位置0x1F(短len=25) |
| 129.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 130.0s | S2C | 多属性0x2D[0xbc110702](布局变体) len=25 2d01008548020711bc3b000200d70135 |
| 130.2s | S2C | 属性更新0x10[O#0x3b6fb8] FP=4 |
| 130.2s | S2C | 属性更新0x10[O#0x3b6fb8] SP=4 |
| 130.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 130.2s | S2C | S2C m24 通用事件[0x3bbc1107024885] 伤害飘字 [1028, 0, 0, "wuji_CS_jz_lyfh05", "SkillLock", "monm031301"] |
| 130.2s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [2] |
| 130.2s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["wuji_CS_jz_lyfh05", obj(0x7024885,0x3bbc11)] |
| 130.2s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 130.2s | S2C | 属性更新0x10[O#0x3b6fb8] CurIsAirSkill=0 |
| 130.2s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 130.2s | S2C | S2C m189 对象通知 [obj(0x7024885,0x3bbc11)] |
| 130.5s | S2C | 多属性0x2D[0xbc110702](布局变体) len=25 2d01008548020711bc3b000200d70107 |
| 130.6s | C2S | m914 招式结束异步位置 [934.102, 45.223, 1028.834, 1.566, 12, 31] |
| 130.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 130.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 130.6s | C2S | m15 移动上报 手动 state=8 pos=(934.1,45.2,1028.8) |
| 130.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 130.7–131.0s | C2S | m15 移动上报 手动 state=6 pos=(934.1,45.2,1028.8) ×2 |
| 130.8s | S2C | 对象同步0x21[0x700ca8c] 码136 len=179 |
| 131.1s | C2S | m15 移动上报 手动 state=4 pos=(932.1,43.9,1029.0) |
| 131.1s | C2S | m15 移动上报 手动 state=7 pos=(932.1,43.9,1029.0) |
| 131.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 131.2s | S2C | 对象同步0x21[0x700ca8c] 码171 len=135 |
| 132.2s | C2S | m15 移动上报 自动 state=4 pos=(927.1,43.9,1029.3) |
| 132.4s | C2S | m15 移动上报 手动 state=7 pos=(927.1,43.9,1029.3) |
| 132.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 132.5s | S2C | 对象同步0x21[0x700ca8c] 码236 len=91 |
| 132.5s | S2C | op0D len=251 0d16d70007e5b63b009e676444b88b2f42c3b275 |
| 132.5s | S2C | op28(子码47) len=230 |
| 132.6s | C2S | m15 移动上报 手动 state=8 pos=(928.1,43.9,1029.3) |
| 132.6s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=165 |
| 132.7s | S2C | 对象同步0x21[0x700ca8c] 码22 len=47 |
| 132.9s | C2S | m15 移动上报 手动 state=6 pos=(929.7,45.2,1029.2) |
| 133.0s | C2S | m211 使用技能(出招) CS_jz_lyfh02 14值 [930.132, 45.369, 1029.128, 1.645] |
| 133.1s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 133.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jz_lyfh02 |
| 133.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurIsAirSkill=4 |
| 133.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 133.1s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 133.1s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill=CS_jz_lyfh02 |
| 133.1s | S2C | S2C m707 受理族变体 [15] |
| 133.1s | C2S | m220 冲锋接近上报 [930.65, 45.431, 1029.09, 1.645] ×2 |
| 133.1s | S2C | 对象同步0x21[0x700ca8c] 码142 len=47 |
| 133.2s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 冲锋动画 [934.38, 44.886, 1028.913, 2, "qg_fastruning_s_2h_01"] |
| 133.2s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 133.2s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 133.2s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 133.2s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 133.2s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 133.2s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =16802048 0x700ca8c] ×2 |
| 133.2s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=97 |
| 133.2s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10140 |
| 133.2s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=155 |
| 133.3s | C2S | m15 移动上报 手动 state=6 pos=(931.2,45.4,1029.1) |
| 133.3s | S2C | 对象同步0x21[0x700ca8c] 码76 len=47 |
| 133.3s | S2C | S2C m707 受理族变体 [29] |
| 133.4s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 133.4s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10240 |
| 133.5s | C2S | m212 冲锋/续招落地 [934.186, 44.907, 1028.922, 1.614] |
| 133.6s | S2C | op31(sub3) 0x700d716 →0x70095b5 |
| 133.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 133.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 133.6s | S2C | 位置0x1F(短len=25) |
| 133.6s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill= |
| 133.6s | S2C | S2C m26 技能释放 [] |
| 133.6s | S2C | 位置0x1F(短len=25) |
| 133.6s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [934.186, 44.907, 1028.922, 1.618, 0] |
| 133.6s | S2C | S2C m24 通用事件[0x3bbc1107024885] 动作开始 [1, obj(0x700ca8c,0x3b6fb8)] |
| 133.6s | S2C | S2C m19 锁定伤害段[0x3b6fb80700ca8c] [obj(0x7024885,0x3bbc11), "wuji_CS_jz_lyfh02", 1066, 1066, 934.186, 44.907] |
| 133.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 133.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 133.6s | C2S | m15 移动上报 手动 state=8 pos=(934.4,44.9,1028.9) |
| 133.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 133.7s | C2S | m15 移动上报 自动 state=7 pos=(934.2,44.9,1028.9) |
| 133.7s | S2C | 位置0x1F(短len=25) |
| 133.8s | S2C | 对象同步0x21[0x700ca8c] 码75 len=47 |
| 134.0s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 134.0s | S2C | 属性更新0x10[O#0x3b6fb8] FP=6 |
| 134.0s | S2C | 属性更新0x10[O#0x3b6fb8] SP=6 |
| 134.0s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 134.0s | S2C | S2C m24 通用事件[0x3bbc1107024885] 伤害飘字 [176, 844, 0, "CS_jz_lyfh02", "SkillLock", "monm031301"] |
| 134.0s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 134.0s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 134.0s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 134.0s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei1", 49, 160, "ui_jh_001"] |
| 134.0s | S2C | S2C m24 通用事件[0x3bbc1107024885] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031301", "徐傠囃辭睼 "] |
| 134.0s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jz_lyfh02", obj(0x7024885,0x3bbc11)] |
| 134.3s | C2S | m15 移动上报 自动 state=7 pos=(934.2,44.9,1028.9) |
| 134.3s | S2C | 位置0x1F(短len=25) |
| 134.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 134.4s | C2S | m82 ? [3] |
| 134.4s | S2C | 多属性0x2D[0xbc110702](布局变体) len=59 2d01008548020711bc3b000600d70100 |
| 134.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 134.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 134.6s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 134.6s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 134.6s | S2C | 属性更新0x10[O#0x3b6fb8] CurIsAirSkill=0 |
| 134.6s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 134.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 134.9s | C2S | m15 移动上报 自动 state=5 pos=(934.0,44.9,1028.9) ×2 |
| 135.0s | C2S | m914 招式结束异步位置 [934.075, 44.907, 1028.927, 1.618, 12, 32] |
| 135.1s | C2S | m990 ? [obj(0x7024885,0x3bbc11), 935.878, 43.886, 1028.842] |
| 135.1s | C2S | m15 移动上报 自动 state=8 pos=(934.1,44.9,1028.9) |
| 135.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 135.2s | S2C | 对象同步0x21[0x700ca8c] 码207 len=135 |
| 135.4–138.4s | S2C | 对象同步0x21[0x70295df] 码32 len=91 ×2 |
| 135.5s | C2S | m15 移动上报 自动 state=4 pos=(934.1,43.9,1028.9) |
| 135.6s | S2C | 属性更新0x10[O#0x3b6fb8] QingGongPoint=165 |
| 135.6s | C2S | m15 移动上报 手动 state=7 pos=(934.1,43.9,1028.9) |
| 135.6s | S2C | 对象同步0x21[0x700ca8c] 码206 len=47 |
| 135.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 135.9s | S2C | 对象同步0x21[0x700ca8c] 码231 len=47 |
| 135.9s | S2C | op06 len=1 06 |
| 135.9s | C2S | m15 移动上报 手动 state=7 pos=(933.0,43.9,1029.0) |
| 136.0s | S2C | 属性更新0x10[O#0x3bbc11] LogicState=0 |
| 136.1s | S2C | 对象同步0x21[0x700ca8c] 码157 len=47 |
| 137.0s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 137.2s | C2S | m15 移动上报 自动 state=4 pos=(927.0,43.9,1029.5) |
| 137.4s | S2C | 对象同步0x21[0x700ca8c] 码37 len=47 |
| 137.5s | S2C | op0D len=251 0d16d70007e5b63b009e676444b88b2f42c3b275 |
| 137.5s | S2C | op28(子码47) len=230 ×2 |
| 137.6s | S2C | 对象同步0x21[0x702ade2] 码49 len=91 |
| 137.6s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 138.1s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 138.1s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10340 |
| 138.4s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 138.5s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 138.6s | S2C | op28(子码47) len=258 |
| 138.7s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 138.7s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 138.7s | S2C | op2C len=148 2c47930000000a010000000000e00600df950207 |
| 138.7s | S2C | op0E len=9 0e8548020711bc3b00 |
| 138.7s | C2S | m440 ? [] |
| 138.7s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 138.7s | C2S | m431 选择目标 [0] |
| 138.7s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 140.4s | S2C | 属性更新0x10[O#0x3b6fb8] HitHPRatio=99 |
| 140.4s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=28057 |
| 140.4s | S2C | 属性更新0x10[O#0x3b6fb8] HPRatio=99 |
| 140.4s | S2C | 属性更新0x10[O#0x3b6fb8] HP=28057 |
| 140.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 140.9s | C2S | m15 移动上报 自动 state=5 pos=(927.0,43.9,1029.5) ×2 |
| 141.7s | S2C | 属性更新0x10[O#0x3b6fb8] FightData=15662707389 |
| 141.9s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 141.9s | S2C | 属性更新0x10[O#0x150] State= |
| 141.9s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 142.7s | C2S | m15 移动上报 自动 state=7 pos=(927.0,43.9,1029.5) |
| 142.7s | S2C | 对象同步0x21[0x7000146] 码237 len=223 |
| 142.7s | S2C | 属性更新0x10[O#0x14b] State= |
| 142.7s | S2C | 多属性0x2D[0x14b0700](布局变体) len=20 2d0100460100074b0100000100540001 |
| 142.7s | C2S | m15 移动上报 自动 state=7 pos=(927.0,43.9,1029.5) |
| 142.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 142.8s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 142.8s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 142.9s | S2C | 对象同步0x21[0x700ca8c] 码0 len=91 |
| 143.0s | S2C | 属性更新0x10[O#0x3b6fb8] LiveGroove_4=84679 |
| 143.0s | S2C | 属性更新0x10[O#0x3b6fb8] LiveGroove_5=36436 |
| 143.0s | S2C | S2C m1 系统提示 [1, "8052", 223] |
| 143.0s | S2C | S2C m349 ? [223] |
| 143.0s | S2C | 属性更新0x10[O#0x3b6fb8] Faculty=114285852 |
| 143.0s | S2C | 属性更新0x10[O#0x3b6fb8] HoldFaculty=1502939 |
| 143.1s | C2S | m54 格挡状态(值2412420744) |
| 143.1s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 143.1s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 143.1s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 143.1s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 143.1s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=11CE,700CA8C,3B6FB8,1,0,0 |
| 143.1s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 143.1s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 143.1s | C2S | m15 移动上报 自动 state=7 pos=(925.1,43.9,1028.5) ×2 |
| 143.3s | C2S | m54 格挡状态(值2188380769) ×3 |
| 143.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 143.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 143.3s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 143.3s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 143.3s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 143.3s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 143.3s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 143.3s | C2S | m54 格挡状态(值2188380769) ×3 |
| 143.4s | S2C | 对象同步0x21[0x700ca8c] 码176 len=91 |
| 143.4s | C2S | m15 移动上报 手动 state=7 pos=(925.3,43.9,1028.7) |
| 143.6s | S2C | 对象同步0x21[0x700ca8c] 码40 len=47 |
| 143.8s | C2S | m15 移动上报 自动 state=4 pos=(927.2,43.9,1029.7) |
| 144.1s | S2C | 对象同步0x21[0x700ca8c] 码232 len=47 |
| 144.4s | C2S | m107 外观自定义保存 ["e$"] |
| 145.4s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=28139 |
| 145.4s | S2C | 属性更新0x10[O#0x3b6fb8] HP=28139 |
| 146.5s | C2S | m15 移动上报 手动 state=7 pos=(927.2,43.9,1029.7) |
| 146.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 146.6s | S2C | op28(子码47) len=235 |
| 146.7s | S2C | 对象同步0x21[0x700ca8c] 码219 len=47 |
| 146.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 146.9s | C2S | m15 移动上报 手动 state=5 pos=(928.4,43.9,1030.4) ×2 |
| 147.4s | C2S | m15 移动上报 自动 state=4 pos=(930.7,43.9,1031.7) |
| 147.5s | S2C | 对象同步0x21[0x700ca8c] 码89 len=47 |
| 147.6s | S2C | op31(sub2) 0x7000146 →0x700d716 |
| 147.8s | S2C | S2C m1 系统提示 [2, "85201", "为伊执剑 ", "qy_box_tableitem_JQ_01", "tableitem_JQ_01_LY"] |
| 148.3s | C2S | m211 使用技能(出招) CS_jh_mzjf05 10值 [930.662, 43.887, 1031.747, 1.047] |
| 148.4s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 148.4s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_mzjf05 |
| 148.4s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 148.4s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +97行 |
| 148.4s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x700ca8c] ×4 |
| 148.4s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 148.4s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 148.4s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 148.4s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 148.4s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10343 |
| 148.4s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_jh_mzjf05", obj(0x700ca8c,0x3b6fb8), 0, 0.0] |
| 148.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=2EBC,700CA8C,3B6FB8,9,C38EA857C69,0 |
| 148.7s | S2C | S2C m169 buff添加 ["buf_CS_jh_mzjf05_01", obj(0x700ca8c,0x3b6fb8)] |
| 148.7s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jh_mzjf05", obj(0x700ca8c,0x3b6fb8)] |
| 148.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 148.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 150.4s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=28221 |
| 150.4s | S2C | 属性更新0x10[O#0x3b6fb8] HP=28221 |
| 150.6–152.1s | S2C | 对象同步0x21[0x70295df] 码149 len=91 ×2 |
| 151.0s | S2C | op06 len=1 06 |
| 151.4s | C2S | 对象请求 len=33 |
| 151.4s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 151.4s | C2S | m431 选择目标 [obj(0x7025c40,0x3bce7e)] |
| 151.5s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 151.7s | S2C | 对象同步0x21[0x702ade2] 码97 len=91 |
| 151.7s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 152.1s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand |
| 152.1s | C2S | m211 使用技能(出招) CS_jh_mzjf01 14值 [930.662, 43.887, 1031.747, 1.047] |
| 152.2s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 152.2s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_mzjf01 |
| 152.2s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 152.2s | S2C | 属性更新0x10[O#0x3bce7e] LogicState=1 |
| 152.2s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 152.2s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 152.2s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 152.2s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 152.2s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 152.2s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =16802048 0x700ca8c] ×2 |
| 152.2s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 152.2s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10242 |
| 152.2s | S2C | 位置0x1F(短len=25) |
| 152.2s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [930.662, 43.886, 1031.747, -2.007, 0] |
| 152.2s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_jh_mzjf01", obj(0x7025c40,0x3bce7e), 0, 0.0] |
| 152.2s | C2S | m15 移动上报 自动 state=4 pos=(930.7,43.9,1031.7) |
| 152.3s | S2C | 对象同步0x21[0x7025c40] 码201 len=91 |
| 152.3s | S2C | 多属性0x2D[0xce7e0702](布局变体) len=23 2d0100405c02077ece3b000100a0018c |
| 152.6s | S2C | S2C m169 buff添加 ["buf_CS_jh_mzjf05_02", obj(0x7025c40,0x3bce7e)] |
| 152.6s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 152.6s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 152.6s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 152.6s | S2C | S2C m24 通用事件[0x3bce7e07025c40] 伤害飘字 [1641, 0, 0, "CS_jh_mzjf01", "SkillNormal", "monm031301"] |
| 152.6s | S2C | 属性更新0x10[O#0x3b6fb8] SP=6 |
| 152.6s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jh_mzjf01", obj(0x7025c40,0x3bce7e)] |
| 152.6s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 152.6s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 152.7s | S2C | 属性更新0x10[O#0x3bbc1c] State=walk |
| 152.7s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 152.8s | S2C | op2C len=136 2c47870000000a010000000000e00600df950207 |
| 152.8s | S2C | 多属性0x2D[0xce7e0702](布局变体) len=67 2d0100405c02077ece3b000300770224 |
| 152.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 152.8s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 动作开始 ["buff_ng_xyhxj01", obj(0x700ca8c,0x3b6fb8)] |
| 152.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo5=4BF9,700CA8C,3B6FB8,5,C38EA858C8F,1 |
| 152.8s | S2C | S2C m169 buff添加 ["buf_ng_jz_006_01", obj(0x700ca8c,0x3b6fb8)] |
| 152.8s | S2C | S2C m169 buff刷新? ["buf_ng_jz_006_01", obj(0x700ca8c,0x3b6fb8)] |
| 152.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo5=4BF9,700CA8C,3B6FB8,5,C38EA858C8F,5 |
| 152.8s | C2S | m15 移动上报 自动 state=5 pos=(930.7,43.9,1031.7) ×2 |
| 153.0s | S2C | 属性更新0x10[O#0x3b6fb8] PhyDamageMul=1.5 |
| 153.0s | S2C | 属性更新0x10[O#0x3b6fb8] StiDamageMul=1.5 |
| 153.0s | S2C | 属性更新0x10[O#0x3b6fb8] JujDamageMul=1.5 |
| 153.0s | S2C | 属性更新0x10[O#0x3b6fb8] NegDamageMul=1.5 |
| 153.0s | S2C | 属性更新0x10[O#0x3b6fb8] MasDamageMul=1.5 |
| 153.0s | S2C | 属性更新0x10[O#0x3b6fb8] UnderWaterModifySpeed=1.0 ×2 |
| 153.0s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 153.0s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10342 |
| 154.0s | C2S | m211 使用技能(出招) CS_jh_mzjf06 10值 [930.662, 43.887, 1031.747, -2.007] |
| 154.1s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 154.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_mzjf06 |
| 154.1s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 154.1s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 154.1s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 154.1s | S2C | 记录写格0x13[cooldown_rec#109 行96 列1 =16801792 0x700ca8c] ×2 |
| 154.1s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 154.1s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10252 |
| 154.1s | S2C | 位置0x1F(短len=25) |
| 154.1s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [930.662, 43.886, 1031.747, -2.007, 0] |
| 154.1s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_jh_mzjf06", 0, 0, 0.0] |
| 154.4s | C2S | m82 ? [3] |
| 154.6s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 154.6s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 155.1s | S2C | 对象同步0x21[0x7025c40] 码220 len=47 |
| 155.9s | C2S | m211 使用技能(出招) CS_jh_mzjf07 10值 [930.662, 43.887, 1031.747, -2.007] |
| 156.0s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 156.0s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_mzjf07 |
| 156.0s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 156.0s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 156.0s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 156.0s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 156.0s | S2C | 记录写格0x13[cooldown_rec#109 行95 列1 =16801536 0x700ca8c] ×2 |
| 156.0s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10224 |
| 156.0s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_jh_mzjf07", obj(0x700ca8c,0x3b6fb8), 0, 0.0] |
| 156.2s | S2C | 属性更新0x10[O#0x3bce7e] State= |
| 156.2s | S2C | 属性更新0x10[O#0x3bce7e] CurSkillID=default_normal_skill |
| 156.2s | S2C | 属性更新0x10[O#0x3bce7e] CurSkillTarget=None |
| 156.2s | S2C | 位置0x1F(短len=25) |
| 156.2s | S2C | S2C m337 施法回显[0x3bce7e07025c40] [929.795, 43.886, 1031.343, 1.135, 0] |
| 156.2s | S2C | S2C m20 范围施放标记[0x3bce7e07025c40] ["atkhit3", obj(0x700ca8c,0x3b6fb8), 0, 0.0] |
| 156.2s | S2C | S2C m169 buff移除 ["buf_CS_jh_mzjf05_02", obj(0x7025c40,0x3bce7e)] |
| 156.2s | S2C | 多属性0x2D[0xce7e0702](布局变体) len=20 2d0100405c02077ece3b000100770201 |
| 156.3s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 156.3s | S2C | S2C m169 buff移除 ["buf_CS_jh_mzjf07_01", obj(0x700ca8c,0x3b6fb8)] |
| 156.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo7= ×2 |
| 156.3s | S2C | S2C m169 buff添加 ["buf_CS_jh_mzjf07_01", obj(0x700ca8c,0x3b6fb8)] |
| 156.3s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jh_mzjf07", obj(0x700ca8c,0x3b6fb8)] |
| 156.3s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 156.3s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 156.6s | S2C | 属性更新0x10[O#0x3b6fb8] Sat=26 |
| 157.2s | S2C | 属性更新0x10[V#0x2] LeftTime=600159 |
| 157.3s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] c1=7 [1, obj(0x7025c40,0x3bce7e), "default_normal_skill", "徐傠囃辭睼 ", "monm031301", 100] |
| 157.3s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 157.3s | S2C | 属性更新0x10[O#0x3bce7e] CurSkillID= |
| 157.3s | S2C | 属性更新0x10[O#0x3bce7e] CurSkillTarget=None |
| 157.3s | S2C | S2C m189 对象通知 [obj(0x7025c40,0x3bce7e)] |
| 157.7s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 157.7s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10324 |
| 158.6s | C2S | m211 使用技能(出招) CS_jh_mzjf03 14值 [930.662, 43.887, 1031.747, -2.007] |
| 158.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 158.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_jh_mzjf03 |
| 158.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 158.7s | S2C | S2C m284 空值标记帧 [] |
| 158.7s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 158.7s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 158.7s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 158.7s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =16801280 0x700ca8c] ×2 |
| 158.7s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 158.7s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10223 |
| 158.7s | S2C | 位置0x1F(短len=25) |
| 158.7s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [930.662, 43.886, 1031.747, 4.277, 0] |
| 158.7s | S2C | S2C m24 通用事件[0x3bce7e07025c40] 动作开始 [1, obj(0x700ca8c,0x3b6fb8)] |
| 158.7s | S2C | S2C m19 锁定伤害段[0x3b6fb80700ca8c] [obj(0x7025c40,0x3bce7e), "CS_jh_mzjf03", 1900, 1900, 930.701, 43.886] |
| 158.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 158.7s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 158.7s | C2S | m15 移动上报 自动 state=7 pos=(930.7,43.9,1031.7) |
| 158.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 158.7s | C2S | m15 移动上报 自动 state=7 pos=(930.7,43.9,1031.8) |
| 158.8s | S2C | 对象同步0x21[0x700ca8c] 码224 len=91 |
| 158.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 158.9s | C2S | m15 移动上报 自动 state=5 pos=(930.9,43.9,1031.8) ×2 |
| 159.0s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 159.0s | S2C | S2C m169 buff添加 ["buf_CS_jh_mzjf05_02", obj(0x7025c40,0x3bce7e)] |
| 159.0s | S2C | 属性更新0x10[O#0x3b6fb8] FP=4 |
| 159.0s | S2C | 属性更新0x10[O#0x3b6fb8] SP=8 |
| 159.0s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 159.0s | S2C | S2C m24 通用事件[0x3bce7e07025c40] 伤害飘字 [591, 1240, 0, "CS_jh_mzjf03", "SkillLock", "monm031301"] |
| 159.0s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo5=4BF9,700CA8C,3B6FB8,5,C38EA858D03,4 |
| 159.0s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 159.0s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 159.0s | S2C | S2C m169 buff移除 ["buf_CS_jh_mzjf05_02", obj(0x7025c40,0x3bce7e)] |
| 159.0s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 159.0s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei1", 56, 160, "ui_jh_001"] |
| 159.0s | S2C | S2C m24 通用事件[0x3bce7e07025c40] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031301", "徐傠囃辭睼 "] |
| 159.0s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_jh_mzjf03", obj(0x7025c40,0x3bce7e)] |
| 159.2s | S2C | 属性更新0x10[O#0x3b6fb8] PhyDamageMul=1.4 |
| 159.2s | S2C | 属性更新0x10[O#0x3b6fb8] StiDamageMul=1.4 |
| 159.2s | S2C | 属性更新0x10[O#0x3b6fb8] JujDamageMul=1.4 |
| 159.2s | S2C | 属性更新0x10[O#0x3b6fb8] NegDamageMul=1.4 |
| 159.2s | S2C | 属性更新0x10[O#0x3b6fb8] MasDamageMul=1.4 |
| 159.2s | S2C | 属性更新0x10[O#0x3b6fb8] UnderWaterModifySpeed=1.0 ×2 |
| 159.2s | S2C | 多属性0x2D[0xce7e0702](布局变体) len=66 2d0100405c02077ece3b000700770201 |
| 159.7s | S2C | op28(子码47) len=234 |
| 160.0s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 160.0s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 160.0s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 160.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 160.6s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 161.1s | S2C | 属性更新0x10[O#0x3bce7e] LogicState=0 |
| 161.2s | C2S | m914 招式结束异步位置 [929.649, 43.886, 1031.275, -2.007, 12, 33] |
| 161.3s | C2S | m990 ? [obj(0x7025c40,0x3bce7e), 928.638, 43.896, 1030.804] |
| 161.3s | C2S | m54 格挡状态(值2412420744) |
| 161.3s | S2C | 对象同步0x21[0x700ca8c] 码139 len=91 |
| 161.4s | S2C | 位置0x1F(短len=25) |
| 161.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 161.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 161.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 161.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 161.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8=11CE,700CA8C,3B6FB8,1,0,0 |
| 161.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 161.4s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 161.7s | C2S | m54 格挡状态(值2188380769) |
| 161.8s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 161.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8= |
| 161.8s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 161.8s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 161.8s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 161.8s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 161.8s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 161.8s | C2S | m54 格挡状态(值2188380769) ×3 |
| 162.1s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 162.2s | C2S | m54 格挡状态(值2412420744) |
| 162.3s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 162.3s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 162.3s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 162.3s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 162.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8=11CE,700CA8C,3B6FB8,1,0,0 |
| 162.3s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 162.3s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 162.4s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 162.4s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10323 |
| 162.4s | C2S | m54 格挡状态(值2188380769) |
| 162.4s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 162.4s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8= |
| 162.4s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 162.4s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 162.4s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 162.4s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 162.4s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 162.4s | C2S | m54 格挡状态(值2188380769) ×3 |
| 162.9s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 163.0s | C2S | m54 格挡状态(值2412420744) |
| 163.0s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 163.0s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 163.0s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 163.0s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 163.0s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8=11CE,700CA8C,3B6FB8,1,0,0 |
| 163.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 163.0s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 163.2s | C2S | m54 格挡状态(值2188380769) |
| 163.2s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 163.2s | C2S | m54 格挡状态(值2188380769) |
| 163.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 163.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8= |
| 163.3s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 163.3s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 163.3s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 163.3s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 163.3s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 163.3s | C2S | m54 格挡状态(值2188380769) ×3 |
| 163.5s | C2S | m54 格挡状态(值2412420744) |
| 163.5s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=1.77 |
| 163.5s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=-2.13 |
| 163.5s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=1 |
| 163.5s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=-93 |
| 163.5s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8=11CE,700CA8C,3B6FB8,1,0,0 |
| 163.5s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 163.5s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=1 |
| 163.7s | C2S | m54 格挡状态(值2188380769) |
| 163.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700ca8c,0x3b6fb8)] |
| 163.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8= |
| 163.7s | S2C | 属性更新0x10[O#0x3b6fb8] InParry=0 |
| 163.7s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 163.7s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 163.7s | S2C | 属性更新0x10[O#0x3b6fb8] ParryUpSpeedAdd=10 |
| 163.7s | S2C | 属性更新0x10[O#0x3b6fb8] InManualParry=0 |
| 163.7s | C2S | m54 格挡状态(值2188380769) |
| 163.8s | S2C | S2C m169 buff移除 ["buf_CS_jh_mzjf05_01", obj(0x700ca8c,0x3b6fb8)] |
| 163.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 163.8s | C2S | m54 格挡状态(值2188380769) |
| 163.8s | S2C | op0E len=9 0e405c02077ece3b00 |
| 163.8s | C2S | m440 ? [] |
| 163.9s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 163.9s | C2S | m431 选择目标 [0] |
| 163.9s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 164.0s | C2S | m15 移动上报 手动 state=7 pos=(929.6,43.9,1031.3) |
| 164.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 164.2s | S2C | 对象同步0x21[0x700ca8c] 码79 len=47 |
| 164.2s | C2S | m15 移动上报 自动 state=4 pos=(930.5,43.9,1031.2) |
| 164.4–164.8s | S2C | 对象同步0x21[0x700ca8c] 码8 len=47 ×2 |
| 164.7s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 164.7s | C2S | m15 移动上报 自动 state=4 pos=(930.5,43.9,1031.2) |
| 164.8s | C2S | m15 移动上报 手动 state=7 pos=(930.5,43.9,1031.2) |
| 164.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 164.9s | C2S | m15 移动上报 手动 state=5 pos=(931.0,43.9,1031.1) ×2 |
| 164.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 165.0s | S2C | 对象同步0x21[0x700ca8c] 码204 len=47 |
| 165.1s | C2S | m15 移动上报 自动 state=4 pos=(932.3,43.9,1031.0) |
| 165.2s | S2C | 对象同步0x21[0x700ca8c] 码36 len=47 |
| 165.5s | S2C | 属性更新0x10[O#0x3b6fb8] HitHPRatio=100 |
| 165.5s | S2C | 属性更新0x10[O#0x3b6fb8] HitHP=28285 |
| 165.5s | S2C | 属性更新0x10[O#0x3b6fb8] HPRatio=100 |
| 165.5s | S2C | 属性更新0x10[O#0x3b6fb8] HP=28285 |
| 165.7s | S2C | op31(sub2) 0x700014b →0x70095b5 |
| 166.3s | S2C | op06 len=1 06 |
| 167.1s | C2S | m107 外观自定义保存 ["e$"] |
| 167.1s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 167.1s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 168.1s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo5=4BF9,700CA8C,3B6FB8,5,C38EA858D71,1 |
| 168.1s | S2C | S2C m169 buff移除 ["buf_ng_jz_006_01", obj(0x700ca8c,0x3b6fb8)] |
| 168.1s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo5= |
| 168.2s | C2S | m15 移动上报 手动 state=7 pos=(932.3,43.9,1031.0) |
| 168.2s | S2C | 属性更新0x10[O#0x3b6fb8] PhyDamageMul=1.0 |
| 168.2s | S2C | 属性更新0x10[O#0x3b6fb8] StiDamageMul=1.0 |
| 168.2s | S2C | 属性更新0x10[O#0x3b6fb8] JujDamageMul=1.0 |
| 168.2s | S2C | 属性更新0x10[O#0x3b6fb8] NegDamageMul=1.0 |
| 168.2s | S2C | 属性更新0x10[O#0x3b6fb8] MasDamageMul=1.0 |
| 168.2s | S2C | 属性更新0x10[O#0x3b6fb8] UnderWaterModifySpeed=1.0 ×2 |
| 168.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 168.3s | S2C | 对象同步0x21[0x700ca8c] 码232 len=47 |
| 168.6s | C2S | m15 移动上报 自动 state=4 pos=(934.3,43.9,1030.8) |
| 168.7–172.0s | S2C | 对象同步0x21[0x700ca8c] 码74 len=47 ×4 |
| 168.7s | S2C | op28(子码47) len=235 |
| 169.0s | S2C | 多属性0x2D[0xced40702](布局变体) len=221 2d0100bea40207d4ce3b001000b50004 |
| 169.1s | C2S | m107 外观自定义保存 ["e$"] |
| 169.8s | S2C | op28(子码47) len=255 |
| 170.4s | C2S | m15 移动上报 自动 state=8 pos=(934.3,43.9,1030.8) |
| 170.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 170.7s | S2C | 对象同步0x21[0x70295df] 码177 len=91 |
| 170.8s | C2S | m211 使用技能(出招) CS_hs_kfkj01 10值 [934.301, 45.206, 1030.785, 1.676] |
| 170.8s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 170.8s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj01 |
| 170.8s | S2C | 属性更新0x10[O#0x3b6fb8] CurIsAirSkill=4 |
| 170.8s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +97行 |
| 170.8s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x700ca8c] ×4 |
| 170.8s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 170.8s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 170.8s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 170.8s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 170.8s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10347 |
| 170.8s | S2C | 位置0x1F(短len=25) |
| 170.8s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [934.301, 45.336, 1030.785, 1.676, 0] |
| 170.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4=489C,700CA8C,3B6FB8,1,C38EA860D7A,1 |
| 170.8s | S2C | S2C m169 buff添加 ["buf_CS_hs_kfkj01", obj(0x700ca8c,0x3b6fb8)] |
| 170.8s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_hs_kfkj01", 0, 0, 0.0] |
| 170.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 170.8s | C2S | m15 移动上报 自动 state=5 pos=(934.3,45.3,1030.8) ×2 |
| 170.9s | C2S | m15 移动上报 自动 state=4 pos=(934.3,45.3,1030.8) |
| 170.9s | C2S | m15 移动上报 自动 state=7 pos=(934.3,45.3,1030.8) |
| 171.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 171.3s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 171.3s | S2C | S2C m169 buff添加 ["eqp_wp_06a", obj(0x702a4be,0x3bced4)] |
| 171.3s | S2C | 属性更新0x10[O#0x3b6fb8] FP=2 |
| 171.3s | S2C | 属性更新0x10[O#0x3b6fb8] SP=2 |
| 171.3s | S2C | 属性更新0x10[O#0x3bced4] LogicState=1 |
| 171.3s | S2C | 记录加行0x11[fight_relation_rec#114 0x700ca8c/0x3b6fb8] +0行 |
| 171.3s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=1 |
| 171.3s | S2C | 属性更新0x10[O#0x3bced4] CurSkillID=default_normal_skill |
| 171.3s | S2C | 属性更新0x10[O#0x3bced4] CurSkillTarget=None |
| 171.3s | S2C | 位置0x1F(短len=25) |
| 171.3s | S2C | S2C m337 施法回显[0x3bced40702a4be] [935.878, 43.886, 1028.842, 5.601, 0] |
| 171.3s | S2C | S2C m20 范围施放标记[0x3bced40702a4be] ["atkhit2", obj(0x700ca8c,0x3b6fb8), 0, 0.0] |
| 171.3s | S2C | S2C m1 系统提示 [2, "8066"] |
| 171.3s | S2C | S2C m24 通用事件[0x3bced40702a4be] 伤害飘字 [1225, 0, 0, "CS_hs_kfkj01", "SkillNormal", "monm031301"] |
| 171.3s | S2C | S2C m18 战斗伤害帧[0x3bced40702a4be] ["hurt_half", 0, obj(0x700ca8c,0x3b6fb8)] |
| 171.3s | S2C | S2C m24 通用事件[0x3bced40702a4be] 动作开始 ["kfkj_hurt_01_self", obj(0x700ca8c,0x3b6fb8)] |
| 171.3s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj01", obj(0x702a4be,0x3bced4)] |
| 171.3s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 171.3s | S2C | 属性更新0x10[O#0x3b6fb8] CurIsAirSkill=0 |
| 171.3s | S2C | 对象同步0x21[0x702a4be] 码49 len=223 |
| 171.3s | S2C | 多属性0x2D[0xced40702](布局变体) len=77 2d0100bea40207d4ce3b000400770224 |
| 171.8s | C2S | m15 移动上报 自动 state=8 pos=(934.3,44.4,1030.8) |
| 171.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 172.1s | C2S | m15 移动上报 自动 state=4 pos=(934.3,43.9,1030.8) |
| 172.2s | S2C | 对象同步0x21[0x700ca8c] 码73 len=47 |
| 172.3s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] c1=7 [1, obj(0x702a4be,0x3bced4), "default_normal_skill", "徐傠囃辭睼 ", "monm031301", 100] |
| 172.3s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 172.3s | S2C | 属性更新0x10[O#0x3bced4] State= |
| 172.3s | S2C | 属性更新0x10[O#0x3bced4] CurSkillID= |
| 172.3s | S2C | 属性更新0x10[O#0x3bced4] CurSkillTarget=None |
| 172.3s | S2C | S2C m189 对象通知 [obj(0x702a4be,0x3bced4)] |
| 172.4s | C2S | m211 使用技能(出招) CS_hs_kfkj07 10值 [934.301, 43.887, 1030.785, 1.676] |
| 172.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 172.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj07 |
| 172.5s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 172.5s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 172.5s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 172.5s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =16802048 0x700ca8c] ×2 |
| 172.5s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10301 |
| 172.5s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_hs_kfkj07", 0, 0, 0.0] |
| 172.6s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 172.6s | S2C | 属性更新0x10[O#0x3b6fb8] FP=4 |
| 172.6s | S2C | 属性更新0x10[O#0x3b6fb8] SP=4 |
| 172.6s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700ca8c] |
| 172.6s | S2C | S2C m24 通用事件[0x3bced40702a4be] 伤害飘字 [1007, 95, 0, "CS_hs_kfkj07", "SkillNormal", "monm031301"] |
| 172.6s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700ca8c] |
| 172.6s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700ca8c] |
| 172.6s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700ca8c] ×2 |
| 172.6s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei1", 63, 160, "ui_jh_001"] |
| 172.6s | S2C | S2C m24 通用事件[0x3bced40702a4be] 击退 [obj(0x700ca8c,0x3b6fb8), "monm031301", "徐傠囃辭睼 "] |
| 172.6s | S2C | S2C m18 战斗伤害帧[0x3bced40702a4be] ["hurt_half", 0, obj(0x700ca8c,0x3b6fb8)] |
| 172.6s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 怒气段 [2] |
| 172.6s | S2C | S2C m24 通用事件[0x3bced40702a4be] 命中? [] |
| 172.6s | S2C | S2C m24 通用事件[0x3bced40702a4be] 动作开始 ["kfkj_hurt_01_self", obj(0x700ca8c,0x3b6fb8)] |
| 172.6s | S2C | S2C m205 伤害目标列表[0x3b6fb80700ca8c] ["CS_hs_kfkj07", obj(0x702a4be,0x3bced4)] |
| 172.8s | S2C | 多属性0x2D[0xced40702](布局变体) len=59 2d0100bea40207d4ce3b000600d70100 |
| 173.0s | S2C | 对象同步0x21[0x702ade2] 码81 len=91 |
| 173.0s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 173.3s | S2C | S2C m169 buff移除 ["eqp_wp_06a", obj(0x702a4be,0x3bced4)] |
| 173.5s | S2C | 多属性0x2D[0xced40702](布局变体) len=20 2d0100bea40207d4ce3b000100770201 |
| 173.6s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 173.6s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 173.9s | S2C | 对象同步0x21[0x70295df] 码177 len=91 |
| 173.9s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 174.0s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 174.2s | S2C | op2C len=147 2c47920000000a010000000000e00600df950207 |
| 174.4s | C2S | m82 ? [3] |
| 174.7s | S2C | 属性更新0x10[O#0x3bced4] LogicState=0 |
| 175.5s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 175.5s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 175.5s | C2S | m15 移动上报 自动 state=8 pos=(934.3,43.9,1030.8) |
| 175.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 175.7–176.5s | S2C | 对象同步0x21[0x700ca8c] 码73 len=47 ×2 |
| 175.7s | S2C | 属性更新0x10[O#0x3b6fb8] LogicState=2 |
| 176.4s | C2S | m15 移动上报 自动 state=6 pos=(934.3,44.9,1030.8) |
| 176.6s | C2S | m15 移动上报 自动 state=4 pos=(934.3,43.9,1030.8) |
| 176.7–180.0s | S2C | 对象同步0x21[0x700ca8c] 码72 len=47 ×4 |
| 176.7s | S2C | op0E len=9 0ebea40207d4ce3b00 |
| 176.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 176.9s | C2S | m15 移动上报 自动 state=5 pos=(934.3,43.9,1030.8) ×2 |
| 177.3s | C2S | m15 移动上报 自动 state=8 pos=(934.3,43.9,1030.8) |
| 177.3s | S2C | 属性更新0x10[O#0x3b6fb8] SP=0 |
| 177.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 177.7s | C2S | m211 使用技能(出招) CS_hs_kfkj04 10值 [934.301, 45.28, 1030.784, 1.676] |
| 177.7s | S2C | S2C m1 系统提示 [2, "9026"] |
| 177.7s | S2C | S2C m707 受理族变体 [8] |
| 178.1s | C2S | m211 使用技能(出招) CS_hs_kfkj03 10值 [934.301, 45.071, 1030.784, 1.676] |
| 178.2s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 178.2s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj03 |
| 178.2s | S2C | 属性更新0x10[O#0x3b6fb8] CurIsAirSkill=4 |
| 178.2s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 178.2s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 178.2s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 178.2s | S2C | 记录写格0x13[cooldown_rec#109 行96 列1 =16801792 0x700ca8c] ×2 |
| 178.2s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 178.2s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10247 |
| 178.2s | S2C | 位置0x1F(短len=25) |
| 178.2s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [934.301, 44.993, 1030.784, 1.676, 0] |
| 178.2s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_hs_kfkj03", 0, 0, 0.0] |
| 178.2s | C2S | m15 移动上报 自动 state=6 pos=(934.3,44.9,1030.8) |
| 178.2s | S2C | 属性更新0x10[O#0x3b6fb8] FP=0 |
| 178.2s | C2S | m15 移动上报 自动 state=4 pos=(934.3,45.0,1030.8) |
| 178.2s | C2S | m15 移动上报 自动 state=7 pos=(934.3,45.0,1030.8) |
| 178.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 178.3s | S2C | S2C m586 状态推送(簇首) [5] |
| 179.8s | C2S | m15 移动上报 自动 state=8 pos=(934.3,44.4,1030.8) |
| 179.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 179.9s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 179.9s | S2C | 属性更新0x10[O#0x3b6fb8] CurIsAirSkill=0 |
| 180.1s | C2S | m15 移动上报 自动 state=4 pos=(934.3,43.9,1030.8) |
| 180.2s | S2C | 对象同步0x21[0x700ca8c] 码71 len=47 |
| 180.5s | C2S | m211 使用技能(出招) CS_hs_kfkj05 10值 [934.301, 43.887, 1030.784, 1.676] |
| 180.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 180.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj05 |
| 180.5s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 180.5s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 180.5s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 180.5s | S2C | 记录写格0x13[cooldown_rec#109 行95 列1 =16801536 0x700ca8c] ×2 |
| 180.5s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10201 |
| 180.5s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_hs_kfkj05", 0, 0, 0.0] |
| 181.2s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 181.5s | S2C | op06 len=1 06 |
| 181.6s | C2S | m211 使用技能(出招) CS_hs_kfkj04 10值 [934.301, 43.887, 1030.784, 1.676] |
| 181.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 181.7s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj04 |
| 181.7s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +98行 |
| 181.7s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700ca8c] ×4 |
| 181.7s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =16801280 0x700ca8c] ×2 |
| 181.7s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=97 |
| 181.7s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10160 |
| 181.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo5=48AB,701C0B9,3B7117,1,C38EA85D629,1 |
| 181.7s | S2C | S2C m169 buff添加 ["buf_CS_hs_kfkj04", obj(0x700ca8c,0x3b6fb8)] |
| 181.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8=48AC,700CA8C,3B6FB8,1,C38EA85D629,0 |
| 181.7s | S2C | S2C m169 buff添加 ["buf_CS_hs_kfkj04_01", obj(0x700ca8c,0x3b6fb8)] |
| 181.7s | S2C | 记录写格0x13[cooldown_rec#109 行98 列4 =25088 0x700ca8c] |
| 181.7s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_hs_kfkj04", 0, 0, 0.0] |
| 181.8s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 181.8s | S2C | S2C m24 通用事件[0x3b6fb80700ca8c] 冲锋动画 [942.257, 43.886, 1029.945, 1, ""] |
| 181.8s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=1 |
| 181.8s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=6.0 |
| 181.8s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=7.6 |
| 181.8s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=1 |
| 181.9s | S2C | 对象同步0x21[0x700ca8c] 码116 len=47 |
| 182.1s | S2C | 属性更新0x10[O#0x3b6fb8] CantMove=0 |
| 182.1s | S2C | 属性更新0x10[O#0x3b6fb8] MoveSpeed=4.5 |
| 182.1s | S2C | 属性更新0x10[O#0x3b6fb8] RunSpeedAdd=0.6 |
| 182.1s | S2C | 属性更新0x10[O#0x3b6fb8] CantParry=0 |
| 182.2s | C2S | m212 冲锋/续招落地 [941.844, 43.886, 1029.988, 1.676] |
| 182.8s | S2C | op31(sub1) 0x70242e8 |
| 182.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 182.9s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 182.9s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10260 |
| 182.9s | C2S | m15 移动上报 自动 state=5 pos=(942.3,43.9,1029.9) ×2 |
| 183.4s | C2S | m15 移动上报 手动 state=7 pos=(942.3,43.9,1029.9) |
| 183.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 183.6s | S2C | 对象同步0x21[0x700ca8c] 码178 len=47 |
| 183.6s | C2S | m15 移动上报 手动 state=7 pos=(941.2,43.9,1030.1) |
| 183.8s | S2C | 对象同步0x21[0x700ca8c] 码12 len=47 |
| 185.0s | C2S | m15 移动上报 自动 state=4 pos=(935.2,43.9,1030.7) |
| 185.1s | S2C | 对象同步0x21[0x700ca8c] 码244 len=47 |
| 186.7s | S2C | S2C m169 buff移除 ["buf_CS_hs_kfkj04_01", obj(0x700ca8c,0x3b6fb8)] |
| 186.7s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo8= |
| 186.8s | S2C | S2C m169 buff移除 ["buf_CS_hs_kfkj04", obj(0x700ca8c,0x3b6fb8)] |
| 186.8s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =16802304 0x700ca8c] ×3 |
| 186.8s | S2C | 记录写格0x13[cooldown_rec#109 行94 列1 =16801280 0x700ca8c] ×2 |
| 186.8s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo5= |
| 186.9s | S2C | op28(子码47) len=230 |
| 187.6s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 187.6s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10360 |
| 188.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 188.9s | C2S | m15 移动上报 自动 state=5 pos=(935.2,43.9,1030.7) ×2 |
| 189.3s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 189.3s | S2C | S2C m250 ? [] |
| 189.4s | S2C | 记录写格0x13[Count_Limit_Form_Rec#363 行5 列1 =33555712 0x0] |
| 189.4s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 189.4s | S2C | S2C m252 ? [] |
| 189.4s | S2C | S2C m250 ? [] |
| 190.0–191.1s | S2C | 对象同步0x21[0x70295df] 码217 len=91 ×2 |
| 190.7s | S2C | 对象同步0x21[0x702ade2] 码52 len=91 |
| 190.7s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 191.1s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 191.3s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 191.3s | S2C | op2C len=151 2c47960000000a010000000000e00600df950207 |
| 192.3s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 192.3s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 192.4s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 192.4s | S2C | 属性更新0x10[O#0x150] State= |
| 192.4s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 193.9s | S2C | op28(子码47) len=235 |
| 194.2s | S2C | 多属性0x2D[0xcf110703](布局变体) len=221 2d0100984e030711cf3b001000b50004 |
| 194.4s | C2S | m82 ? [3] |
| 194.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 194.9s | C2S | m15 移动上报 自动 state=5 pos=(935.2,43.9,1030.7) ×2 |
| 195.4s | C2S | m15 移动上报 自动 state=7 pos=(935.2,43.9,1030.7) |
| 195.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 195.5s | C2S | m15 移动上报 自动 state=7 pos=(935.2,43.9,1030.7) |
| 195.5s | S2C | 对象同步0x21[0x700ca8c] 码48 len=47 |
| 195.7s | S2C | 对象同步0x21[0x700ca8c] 码244 len=47 |
| 196.0s | C2S | m15 移动上报 自动 state=4 pos=(934.2,43.9,1033.2) |
| 196.1s | S2C | 对象同步0x21[0x700ca8c] 码54 len=47 |
| 196.7s | S2C | op06 len=1 06 |
| 196.8s | C2S | 对象请求 len=33 |
| 196.9s | S2C | 属性更新0x10[O#0x3b6fb8] LastObject=None |
| 196.9s | C2S | m431 选择目标 [obj(0x7034e98,0x3bcf11)] |
| 196.9s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [934.206, 43.887, 1033.23, -0.379] |
| 196.9s | C2S | m15 移动上报 自动 state=7 pos=(934.2,43.9,1033.2) ×2 |
| 196.9s | S2C | 属性更新0x10[O#0x3b6fb8] AutoSelectObject=None |
| 196.9s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] ×2 |
| 196.9s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj02 |
| 196.9s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 196.9s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 196.9s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill=CS_hs_kfkj02 |
| 196.9s | S2C | S2C m707 受理族变体 [15] |
| 196.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 197.0s | S2C | 对象同步0x21[0x700ca8c] 码201 len=91 |
| 197.4s | C2S | m15 移动上报 自动 state=4 pos=(931.8,43.9,1032.2) |
| 197.5s | C2S | m15 移动上报 自动 state=7 pos=(931.8,43.9,1032.2) ×2 |
| 197.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 197.6s | S2C | 对象同步0x21[0x700ca8c] 码14 len=135 |
| 198.3s | C2S | m15 移动上报 自动 state=4 pos=(933.3,43.9,1036.1) |
| 198.4s | C2S | m211 使用技能(出招) CS_hs_kfkj03 10值 [933.348, 43.887, 1036.129, 0.366] |
| 198.4–198.8s | S2C | 对象同步0x21[0x700ca8c] 码73 len=47 ×2 |
| 198.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 198.5s | S2C | S2C m26 技能释放 [] |
| 198.5s | S2C | 属性更新0x10[O#0x3b6fb8] PauseSkill= |
| 198.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700ca8c] |
| 198.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID=CS_hs_kfkj03 |
| 198.5s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillTarget=None |
| 198.5s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +96行 |
| 198.5s | S2C | 记录写格0x13[cooldown_rec#109 行96 列1 =24576 0x700ca8c] ×4 |
| 198.5s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700ca8c] ×2 |
| 198.5s | S2C | 记录加行0x11[cooldown_rec#109 0x700ca8c/0x3b6fb8] +97行 |
| 198.5s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x700ca8c] ×4 |
| 198.5s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=98 |
| 198.5s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10247 |
| 198.5s | S2C | 位置0x1F(短len=25) |
| 198.5s | S2C | S2C m337 施法回显[0x3b6fb80700ca8c] [933.348, 43.886, 1036.129, -2.257, 0] |
| 198.5s | S2C | S2C m20 范围施放标记[0x3b6fb80700ca8c] ["CS_hs_kfkj03", 0, 0, 0.0] |
| 198.6s | C2S | m15 移动上报 自动 state=4 pos=(933.3,43.9,1036.1) |
| 198.9s | S2C | op28(子码47) len=230 |
| 200.1s | S2C | 属性更新0x10[O#0x3b6fb8] CurSkillID= |
| 200.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 201.0s | C2S | m15 移动上报 自动 state=5 pos=(933.3,43.9,1036.1) ×2 |
| 201.3s | S2C | S2C m169 buff移除 ["buf_CS_hs_kfkj01", obj(0x700ca8c,0x3b6fb8)] |
| 201.3s | S2C | 属性更新0x10[O#0x3b6fb8] BufferInfo4= |
| 203.2s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=99 |
| 203.2s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10347 |
| 203.6s | S2C | 对象同步0x21[0x70295df] 码32 len=91 |
| 205.8s | S2C | 对象同步0x21[0x702ade2] 码49 len=91 |
| 205.8s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 206.6s | S2C | 对象同步0x21[0x70295df] 码32 len=91 |
| 206.6s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand |
| 206.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "8370F8BA6B66412A9EE0D8F990D54109"] |
| 207.0s | C2S | m15 移动上报 自动 state=5 pos=(933.3,43.9,1036.1) ×2 |
| 207.0s | S2C | op28(子码47) len=235 |
| 207.6s | S2C | 属性更新0x10[O#0x3bbc1c] State=walk |
| 207.6s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 207.7s | S2C | op2C len=147 2c47920000000a010000000000e00600df950207 |
| 207.8s | S2C | 属性更新0x10[O#0x3b6fb8] MPRatio=100 |
| 207.8s | S2C | 属性更新0x10[O#0x3b6fb8] MP=10388 |
| 1794118558.2s | C2S | 6001 大厅心跳 04 00 00 00 10 EE 11 11 → 01 FF 00 00 ×21 |

## f.pcapng — 49.4.14.150:7074

场景：kfkj02 十连按拉回剧本（docs 已详析：选怪→受理→拉回→冲刺→命中）。事件 873 条 → 合并为 541 段。抓包起点 2026-11-08 18:33:41（本地墙钟）

| 时间 | 方向 | 交互 |
|---|---|---|
| 0.0–49.4s | PAIR | C2S m924 MSG_DELAY(client ms) → S2C m294 回显[client ms, obj, 服务器unix秒, …] ×50 |
| 0.1–49.5s | PAIR | C2S ping(0x18) → S2C 0x38 回显 ×46 |
| 0.2s | S2C | op31(sub1) 0x700014b |
| 1.4s | C2S | m54 格挡状态(值4539351178) |
| 1.5s | S2C | 属性更新0x10[O#0x3db441] MoveSpeed=1.77 |
| 1.5s | S2C | 属性更新0x10[O#0x3db441] RunSpeedAdd=-2.13 |
| 1.5s | S2C | 属性更新0x10[O#0x3db441] InParry=1 |
| 1.5s | S2C | 属性更新0x10[O#0x3db441] ParryUpSpeedAdd=-93 |
| 1.5s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4=11CE,700991D,3DB441,1,0,0 |
| 1.5s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700991d,0x3db441)] |
| 1.5s | S2C | 属性更新0x10[O#0x3db441] InManualParry=1 |
| 1.7s | C2S | m15 移动上报 手动 state=4 pos=(930.8,43.9,1034.9) |
| 1.8–4.6s | S2C | 对象同步0x21[0x700991d] 码6 len=47 ×4 |
| 2.1s | C2S | m54 格挡状态(值4315311203) |
| 2.1s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700991d,0x3db441)] |
| 2.1s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4= |
| 2.1s | S2C | 属性更新0x10[O#0x3db441] InParry=0 |
| 2.1s | S2C | 属性更新0x10[O#0x3db441] MoveSpeed=4.5 |
| 2.1s | S2C | 属性更新0x10[O#0x3db441] RunSpeedAdd=0.6 |
| 2.1s | S2C | 属性更新0x10[O#0x3db441] ParryUpSpeedAdd=10 |
| 2.1s | S2C | 属性更新0x10[O#0x3db441] InManualParry=0 |
| 2.1s | C2S | m54 格挡状态(值4315311203) ×2 |
| 2.2s | C2S | m15 移动上报 自动 state=4 pos=(930.8,43.9,1034.9) |
| 2.7s | C2S | m54 格挡状态(值4539351178) |
| 2.8s | S2C | 属性更新0x10[O#0x3db441] MoveSpeed=1.77 |
| 2.8s | S2C | 属性更新0x10[O#0x3db441] RunSpeedAdd=-2.13 |
| 2.8s | S2C | 属性更新0x10[O#0x3db441] InParry=1 |
| 2.8s | S2C | 属性更新0x10[O#0x3db441] ParryUpSpeedAdd=-93 |
| 2.8s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4=11CE,700991D,3DB441,1,0,0 |
| 2.8s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700991d,0x3db441)] |
| 2.8s | S2C | 属性更新0x10[O#0x3db441] InManualParry=1 |
| 2.9s | C2S | m54 格挡状态(值4315311203) |
| 2.9s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700991d,0x3db441)] |
| 2.9s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4= |
| 2.9s | S2C | 属性更新0x10[O#0x3db441] InParry=0 |
| 2.9s | S2C | 属性更新0x10[O#0x3db441] MoveSpeed=4.5 |
| 2.9s | S2C | 属性更新0x10[O#0x3db441] RunSpeedAdd=0.6 |
| 2.9s | S2C | 属性更新0x10[O#0x3db441] ParryUpSpeedAdd=10 |
| 2.9s | S2C | 属性更新0x10[O#0x3db441] InManualParry=0 |
| 2.9s | C2S | m54 格挡状态(值4315311203) ×3 |
| 3.2s | C2S | m15 移动上报 自动 state=4 pos=(930.8,43.9,1034.9) |
| 3.3s | S2C | 记录写格0x13[Count_Limit_Form_Rec#363 行5 列1 =33555712 0x0] |
| 3.3s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 3.3s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 3.3s | S2C | S2C m252 ? [] |
| 3.3s | S2C | S2C m250 ? [] ×3 |
| 3.4s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "9EEB2019797B42B1A66B7C74557CDB1A"] |
| 3.5s | C2S | m15 移动上报 自动 state=5 pos=(930.8,43.9,1034.9) ×2 |
| 3.6s | C2S | 对象请求 len=33 |
| 3.7s | S2C | 属性更新0x10[O#0x3db441] LastObject=None |
| 3.7s | C2S | m431 选择目标 [obj(0x704054e,0x3db879)] |
| 3.7s | C2S | 对象请求 len=33 |
| 3.7s | C2S | m431 选择目标 [obj(0x704054e,0x3db879)] |
| 3.7s | S2C | 属性更新0x10[O#0x3db441] AutoSelectObject=None |
| 4.0s | C2S | m54 格挡状态(值4539351178) |
| 4.0s | S2C | 属性更新0x10[O#0x3db441] MoveSpeed=1.77 |
| 4.0s | S2C | 属性更新0x10[O#0x3db441] RunSpeedAdd=-2.13 |
| 4.0s | S2C | 属性更新0x10[O#0x3db441] InParry=1 |
| 4.0s | S2C | 属性更新0x10[O#0x3db441] ParryUpSpeedAdd=-93 |
| 4.0s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4=11CE,700991D,3DB441,1,0,0 |
| 4.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700991d,0x3db441)] |
| 4.0s | S2C | 属性更新0x10[O#0x3db441] InManualParry=1 |
| 4.1s | C2S | m54 格挡状态(值4315311203) |
| 4.1s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700991d,0x3db441)] |
| 4.1s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4= |
| 4.1s | S2C | 属性更新0x10[O#0x3db441] InParry=0 |
| 4.1s | S2C | 属性更新0x10[O#0x3db441] MoveSpeed=4.5 |
| 4.1s | S2C | 属性更新0x10[O#0x3db441] RunSpeedAdd=0.6 |
| 4.1s | S2C | 属性更新0x10[O#0x3db441] ParryUpSpeedAdd=10 |
| 4.1s | S2C | 属性更新0x10[O#0x3db441] InManualParry=0 |
| 4.1s | C2S | m54 格挡状态(值4315311203) ×3 |
| 4.3s | C2S | m15 移动上报 自动 state=4 pos=(930.8,43.9,1034.9) |
| 4.5s | C2S | m15 移动上报 手动 state=7 pos=(930.8,43.9,1034.9) |
| 4.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 4.7s | S2C | 对象同步0x21[0x70295df] 码177 len=91 |
| 4.9s | C2S | m54 格挡状态(值4539351178) |
| 5.0s | S2C | 属性更新0x10[O#0x3db441] MoveSpeed=1.77 |
| 5.0s | S2C | 属性更新0x10[O#0x3db441] RunSpeedAdd=-2.13 |
| 5.0s | S2C | 属性更新0x10[O#0x3db441] InParry=1 |
| 5.0s | S2C | 属性更新0x10[O#0x3db441] ParryUpSpeedAdd=-93 |
| 5.0s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4=11CE,700991D,3DB441,1,0,0 |
| 5.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x700991d,0x3db441)] |
| 5.0s | S2C | 属性更新0x10[O#0x3db441] InManualParry=1 |
| 5.0s | C2S | m15 移动上报 手动 state=7 pos=(931.0,43.9,1037.2) ×2 |
| 5.2s | S2C | 对象同步0x21[0x700991d] 码253 len=47 |
| 5.2s | C2S | m54 格挡状态(值4315311203) ×2 |
| 5.2s | C2S | m15 移动上报 手动 state=7 pos=(931.2,43.9,1037.6) |
| 5.2s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x700991d,0x3db441)] |
| 5.2s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4= |
| 5.2s | S2C | 属性更新0x10[O#0x3db441] InParry=0 |
| 5.2s | S2C | 属性更新0x10[O#0x3db441] MoveSpeed=4.5 |
| 5.2s | S2C | 属性更新0x10[O#0x3db441] RunSpeedAdd=0.6 |
| 5.2s | S2C | 属性更新0x10[O#0x3db441] ParryUpSpeedAdd=10 |
| 5.2s | S2C | 属性更新0x10[O#0x3db441] InManualParry=0 |
| 5.2s | C2S | m54 格挡状态(值4315311203) ×2 |
| 5.3s | C2S | m15 移动上报 手动 state=7 pos=(931.3,43.9,1037.7) |
| 5.3s | C2S | m54 格挡状态(值4315311203) |
| 5.4s | S2C | 对象同步0x21[0x700991d] 码133 len=91 |
| 6.0s | C2S | m15 移动上报 自动 state=4 pos=(933.2,43.9,1040.5) |
| 6.2s | C2S | m211 使用技能(出招) CS_hs_kfkj01 10值 [933.217, 43.887, 1040.522, 0.602] |
| 6.2s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700991d] ×2 |
| 6.2s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj01 |
| 6.2s | S2C | 记录加行0x11[cooldown_rec#109 0x700991d/0x3db441] +97行 |
| 6.2s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x700991d] ×4 |
| 6.2s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700991d] ×2 |
| 6.2s | S2C | 记录加行0x11[cooldown_rec#109 0x700991d/0x3db441] +98行 |
| 6.2s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700991d] ×4 |
| 6.2s | S2C | 属性更新0x10[O#0x3db441] MPRatio=99 |
| 6.2s | S2C | 属性更新0x10[O#0x3db441] MP=10347 |
| 6.2s | S2C | 位置0x1F(短len=25) |
| 6.2s | S2C | S2C m337 施法回显[0x3db4410700991d] [933.217, 43.886, 1040.522, -2.498, 0] |
| 6.2s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4=489C,700991D,3DB441,1,C38EB6F8C38,1 |
| 6.2s | S2C | S2C m169 buff添加 ["buf_CS_hs_kfkj01", obj(0x700991d,0x3db441)] |
| 6.2s | S2C | S2C m20 范围施放标记[0x3db4410700991d] ["CS_hs_kfkj01", 0, 0, 0.0] |
| 6.3–6.5s | S2C | 对象同步0x21[0x700991d] 码234 len=47 ×2 |
| 6.3s | C2S | m15 移动上报 自动 state=4 pos=(933.2,43.9,1040.5) |
| 6.3s | S2C | op31(sub2) 0x700d965 →0x700fed1 |
| 6.6s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 6.7s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 6.9s | S2C | 对象同步0x21[0x702ade2] 码81 len=91 |
| 7.6s | S2C | 多属性0x2D[0xbf710700](布局变体) len=210 2d010000ca000771bf3d001000b50004 |
| 7.7s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [933.217, 43.887, 1040.522, -2.498] |
| 7.8s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700991d] |
| 7.8s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 7.8s | S2C | 属性更新0x10[O#0x3db441] CurSkillTarget=None |
| 7.8s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 7.8s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 7.8s | S2C | S2C m707 受理族变体 [15] |
| 7.8s | C2S | m15 移动上报 自动 state=7 pos=(933.2,43.9,1040.5) ×2 |
| 7.8s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand |
| 7.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 8.0s | S2C | 对象同步0x21[0x70295df] 码177 len=179 |
| 8.3s | S2C | op28(子码47) len=225 |
| 8.6s | C2S | m15 移动上报 手动 state=7 pos=(931.0,43.9,1037.5) |
| 8.6s | S2C | 对象同步0x21[0x700991d] 码151 len=47 |
| 8.8s | S2C | 属性更新0x10[O#0x3bbc1c] State=walk |
| 8.8s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 8.8s | C2S | m15 移动上报 手动 state=7 pos=(931.6,43.9,1038.4) |
| 8.9s | S2C | op2C len=165 2c47a400000036010000000000e00700df950207 |
| 9.1s | S2C | op06 len=1 06 |
| 9.3s | S2C | op31(sub1) 0x700d965 |
| 9.4s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "9EEB2019797B42B1A66B7C74557CDB1A"] |
| 9.5s | C2S | m15 移动上报 手动 state=5 pos=(933.2,43.9,1040.8) ×2 |
| 9.9–10.7s | C2S | m15 移动上报 手动 state=7 pos=(934.5,43.9,1042.6) ×4 |
| 10.0s | S2C | 对象同步0x21[0x700991d] 码155 len=47 |
| 10.3s | S2C | 对象同步0x21[0x700991d] 码202 len=47 |
| 10.6s | S2C | 对象同步0x21[0x700991d] 码203 len=47 |
| 10.8s | C2S | m15 移动上报 自动 state=4 pos=(936.7,43.9,1040.3) |
| 10.8s | S2C | 对象同步0x21[0x700991d] 码121 len=47 |
| 10.9s | S2C | 属性更新0x10[O#0x3db441] MPRatio=100 |
| 10.9s | S2C | 属性更新0x10[O#0x3db441] MP=10388 |
| 10.9s | C2S | m15 移动上报 ?2000 state=7 pos=(936.7,43.9,1040.3) |
| 11.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 11.1s | S2C | 对象同步0x21[0x700991d] 码254 len=91 |
| 11.2s | C2S | m15 移动上报 自动 state=4 pos=(935.8,43.9,1040.9) |
| 11.2s | S2C | 对象同步0x21[0x700991d] 码32 len=47 |
| 11.4s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [935.768, 43.887, 1040.913, -0.969] |
| 11.4s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 11.4s | S2C | S2C m26 技能释放 [] |
| 11.4s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 11.4s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 11.4s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 11.4s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 11.4s | S2C | S2C m707 受理族变体 [15] |
| 11.4s | C2S | m15 移动上报 自动 state=7 pos=(935.8,43.9,1040.9) ×2 |
| 11.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 11.5s | S2C | 对象同步0x21[0x700991d] 码201 len=91 |
| 12.4s | S2C | op31(sub1) 0x700ca00 |
| 12.5s | C2S | m15 移动上报 ?2000 state=7 pos=(932.2,43.9,1037.1) |
| 12.8s | S2C | 对象同步0x21[0x700991d] 码135 len=47 |
| 12.9s | C2S | m15 移动上报 自动 state=4 pos=(930.6,43.9,1038.2) |
| 13.0s | C2S | m15 移动上报 手动 state=7 pos=(930.6,43.9,1038.2) |
| 13.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 13.3s | S2C | 对象同步0x21[0x700991d] 码195 len=91 |
| 13.4s | S2C | op31(sub2) 0x700d965 →0x700fed1 |
| 13.4s | S2C | op28(子码47) len=255 |
| 13.4s | C2S | m15 移动上报 手动 state=7 pos=(931.6,43.9,1039.6) ×2 |
| 13.4s | S2C | 对象同步0x21[0x700991d] 码248 len=47 |
| 13.7s | S2C | 对象同步0x21[0x700991d] 码142 len=47 |
| 14.1s | C2S | m15 移动上报 自动 state=4 pos=(934.4,43.9,1038.0) |
| 14.4s | S2C | 对象同步0x21[0x700991d] 码219 len=47 |
| 14.5s | S2C | op31(sub1) 0x700ca00 |
| 14.6s | C2S | m15 移动上报 手动 state=7 pos=(934.4,43.9,1038.0) |
| 14.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 14.7s | S2C | 对象同步0x21[0x700991d] 码214 len=47 |
| 15.4s | S2C | op31(sub2) 0x700d965 →0x700fed1 |
| 15.4s | C2S | m15 移动上报 自动 state=4 pos=(936.5,43.9,1041.0) |
| 15.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "9EEB2019797B42B1A66B7C74557CDB1A"] |
| 15.5s | C2S | m15 移动上报 自动 state=5 pos=(936.5,43.9,1041.0) ×2 |
| 15.6s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [936.519, 43.887, 1041.001, 0.602] |
| 15.6s | S2C | 对象同步0x21[0x700991d] 码59 len=47 |
| 15.6s | C2S | m15 移动上报 自动 state=7 pos=(936.5,43.9,1041.0) ×2 |
| 15.6s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 15.6s | S2C | S2C m26 技能释放 [] |
| 15.6s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 15.6s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x700991d] |
| 15.6s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 15.6s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 15.6s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 15.6s | S2C | S2C m707 受理族变体 [15] |
| 15.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 15.8s | S2C | 对象同步0x21[0x700991d] 码201 len=91 |
| 16.4s | S2C | op31(sub1) 0x700ca00 |
| 16.8–17.1s | C2S | m15 移动上报 手动 state=7 pos=(932.7,43.9,1037.1) ×2 |
| 17.1s | S2C | 对象同步0x21[0x700991d] 码11 len=47 |
| 17.3s | S2C | 对象同步0x21[0x700991d] 码20 len=47 |
| 17.4s | S2C | op0D len=251 0d65d900073d043b002b776944e8fb4242a15277 |
| 17.4s | S2C | op28(子码47) len=222 |
| 18.2s | C2S | m82 ? [3] |
| 18.4s | S2C | op31(sub2) 0x700d965 →0x700fed1 |
| 18.4s | S2C | op28(子码47) len=225 |
| 18.7s | C2S | m15 移动上报 自动 state=4 pos=(937.4,43.9,1044.1) |
| 18.8s | S2C | 对象同步0x21[0x700991d] 码7 len=47 |
| 19.0s | C2S | m15 移动上报 手动 state=7 pos=(937.4,43.9,1044.1) |
| 19.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 19.1s | C2S | m15 移动上报 自动 state=4 pos=(937.3,43.9,1043.8) |
| 19.1s | S2C | 对象同步0x21[0x700991d] 码12 len=47 |
| 19.3s | S2C | 对象同步0x21[0x700991d] 码51 len=47 |
| 19.4s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [937.269, 43.887, 1043.808, -2.539] |
| 19.4s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 19.4s | S2C | S2C m26 技能释放 [] |
| 19.4s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 19.4s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 19.4s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 19.4s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 19.4s | S2C | S2C m707 受理族变体 [15] |
| 19.4s | C2S | m15 移动上报 自动 state=7 pos=(937.3,43.9,1043.8) ×2 |
| 19.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 19.7s | S2C | 对象同步0x21[0x700991d] 码201 len=91 |
| 20.9–21.1s | C2S | m15 移动上报 手动 state=7 pos=(932.9,43.9,1038.7) ×2 |
| 20.9s | S2C | 对象同步0x21[0x700991d] 码159 len=47 |
| 21.1s | S2C | 对象同步0x21[0x700991d] 码237 len=47 |
| 21.4s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "9EEB2019797B42B1A66B7C74557CDB1A"] |
| 21.5s | C2S | m15 移动上报 手动 state=5 pos=(934.5,43.9,1041.0) ×2 |
| 22.4s | S2C | op31(sub2) 0x7022212 →0x7007d1f |
| 22.5s | C2S | m15 移动上报 自动 state=4 pos=(937.0,43.9,1044.7) |
| 22.6s | C2S | m15 移动上报 手动 state=7 pos=(937.0,43.9,1044.7) |
| 22.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 22.7–23.1s | C2S | m15 移动上报 自动 state=4 pos=(936.8,43.9,1044.4) ×2 |
| 22.7s | S2C | 对象同步0x21[0x700991d] 码66 len=91 |
| 22.9–23.1s | S2C | 对象同步0x21[0x700991d] 码180 len=47 ×2 |
| 23.4s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [936.808, 43.887, 1044.375, -2.539] |
| 23.4s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 23.4s | S2C | S2C m26 技能释放 [] |
| 23.4s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 23.4s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 23.4s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 23.4s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 23.4s | S2C | S2C m707 受理族变体 [15] |
| 23.4s | C2S | m15 移动上报 自动 state=7 pos=(936.8,43.9,1044.4) ×2 |
| 23.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 23.5s | S2C | 对象同步0x21[0x700991d] 码201 len=91 |
| 23.7s | S2C | 多属性0x2D[0xc00b0703](布局变体) len=210 2d01008be703070bc03d001000b50004 |
| 24.3s | S2C | op06 len=1 06 |
| 24.5s | S2C | op31(sub1) 0x703e78b |
| 24.8–25.8s | S2C | 对象同步0x21[0x70295df] 码217 len=91 ×2 |
| 25.3s | C2S | m15 移动上报 手动 state=7 pos=(931.3,43.9,1037.4) |
| 25.4s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 25.4s | S2C | 对象同步0x21[0x702ade2] 码52 len=135 |
| 25.5s | S2C | op31(sub1) 0x700ca00 |
| 25.5s | S2C | op28(子码47) len=222 ×2 |
| 25.6s | C2S | m15 移动上报 手动 state=7 pos=(931.9,43.9,1038.3) |
| 25.6s | S2C | 对象同步0x21[0x700991d] 码94 len=47 |
| 25.8–26.1s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 26.1s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 26.3s | S2C | op2C len=139 2c478a0000000a010000000000e00600df950207 |
| 26.5s | S2C | op31(sub1) 0x700d965 |
| 26.7s | C2S | m15 移动上报 自动 state=4 pos=(934.9,43.9,1042.6) |
| 26.9s | S2C | 对象同步0x21[0x700991d] 码97 len=47 |
| 27.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "9EEB2019797B42B1A66B7C74557CDB1A"] |
| 27.5s | S2C | op28(子码47) len=225 |
| 27.5s | C2S | m15 移动上报 自动 state=5 pos=(934.9,43.9,1042.6) ×2 |
| 27.7s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [934.865, 43.887, 1042.625, 0.602] |
| 27.7s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 27.7s | S2C | S2C m26 技能释放 [] |
| 27.7s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 27.7s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 27.7s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 27.7s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 27.7s | S2C | S2C m707 受理族变体 [15] |
| 27.7s | C2S | m15 移动上报 自动 state=7 pos=(934.9,43.9,1042.6) ×2 |
| 27.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 28.0–32.7s | S2C | 对象同步0x21[0x700991d] 码201 len=91 ×3 |
| 28.5s | C2S | m15 移动上报 手动 state=7 pos=(932.7,43.9,1039.8) |
| 28.5s | S2C | op31(sub1) 0x703e78b |
| 28.6s | S2C | 对象同步0x21[0x700991d] 码1 len=47 |
| 28.7s | C2S | m15 移动上报 手动 state=7 pos=(933.3,43.9,1040.7) |
| 28.8s | S2C | 对象同步0x21[0x700991d] 码217 len=47 |
| 29.4s | C2S | m15 移动上报 自动 state=4 pos=(935.0,43.9,1043.1) |
| 29.5s | S2C | 对象同步0x21[0x700991d] 码218 len=47 |
| 29.6s | C2S | m15 移动上报 手动 state=7 pos=(935.0,43.9,1043.1) |
| 29.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 29.7s | S2C | 属性更新0x10[O#0x150] State= |
| 29.7s | S2C | op2C len=147 2c47920000000a0100000000008006001d990007 |
| 29.7s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 29.7s | C2S | m15 移动上报 自动 state=4 pos=(934.8,43.9,1042.8) ×2 |
| 29.9s | S2C | 对象同步0x21[0x700991d] 码121 len=91 |
| 30.0s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [934.82, 43.887, 1042.837, -2.539] |
| 30.0s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 30.0s | S2C | S2C m26 技能释放 [] |
| 30.0s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 30.0s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 30.0s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 30.0s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 30.0s | S2C | S2C m707 受理族变体 [15] |
| 30.0s | C2S | m15 移动上报 自动 state=7 pos=(934.8,43.9,1042.8) ×2 |
| 30.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 30.5s | S2C | op31(sub1) 0x703e78b |
| 31.0–31.3s | C2S | m15 移动上报 手动 state=7 pos=(932.0,43.9,1039.0) ×2 |
| 31.2s | S2C | 对象同步0x21[0x700991d] 码118 len=47 |
| 31.4s | S2C | 对象同步0x21[0x700991d] 码208 len=47 |
| 32.1s | C2S | m15 移动上报 自动 state=4 pos=(934.7,43.9,1043.0) |
| 32.3s | S2C | 对象同步0x21[0x700991d] 码196 len=47 |
| 32.4s | C2S | m15 移动上报 手动 state=7 pos=(934.7,43.9,1043.0) |
| 32.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 32.6s | S2C | op31(sub1) 0x7022212 |
| 32.7s | C2S | m15 移动上报 手动 state=7 pos=(934.1,43.9,1042.1) |
| 32.9s | S2C | 对象同步0x21[0x700991d] 码27 len=47 |
| 33.1s | C2S | m15 移动上报 自动 state=4 pos=(932.9,43.9,1040.4) |
| 33.4s | S2C | 对象同步0x21[0x700991d] 码137 len=47 |
| 33.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "9EEB2019797B42B1A66B7C74557CDB1A"] |
| 33.5s | C2S | m15 移动上报 自动 state=5 pos=(932.9,43.9,1040.4) ×2 |
| 33.6s | S2C | op31(sub1) 0x703e78b |
| 33.6s | C2S | m15 移动上报 手动 state=7 pos=(932.9,43.9,1040.4) |
| 33.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 33.7s | S2C | 对象同步0x21[0x700991d] 码132 len=47 |
| 33.8s | C2S | m15 移动上报 手动 state=7 pos=(933.5,43.9,1041.3) |
| 34.0s | S2C | 对象同步0x21[0x700991d] 码197 len=47 |
| 34.6s | S2C | op31(sub1) 0x7022212 |
| 34.6s | C2S | m15 移动上报 自动 state=4 pos=(935.6,43.9,1044.3) |
| 34.7s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [935.593, 43.887, 1044.256, 0.602] |
| 34.8s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 34.8s | S2C | S2C m26 技能释放 [] |
| 34.8s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 34.8s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 34.8s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 34.8s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 34.8s | S2C | S2C m707 受理族变体 [15] |
| 34.8s | C2S | m15 移动上报 自动 state=7 pos=(935.6,43.9,1044.3) ×2 |
| 34.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 34.9s | S2C | 对象同步0x21[0x700991d] 码249 len=135 |
| 35.0s | S2C | 属性更新0x10[O#0x3db441] FightData=1493178652289 |
| 35.6s | S2C | op31(sub1) 0x703e78b |
| 35.8–36.0s | C2S | m15 移动上报 手动 state=7 pos=(932.8,43.9,1040.4) ×2 |
| 35.9s | S2C | 对象同步0x21[0x700991d] 码216 len=47 |
| 36.1s | S2C | 对象同步0x21[0x700991d] 码237 len=47 |
| 36.6s | S2C | op31(sub1) 0x7022212 |
| 36.8s | S2C | S2C m169 buff移除 ["buf_CS_hs_kfkj01", obj(0x700991d,0x3db441)] |
| 36.8s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4= |
| 37.0s | C2S | m15 移动上报 自动 state=4 pos=(935.9,43.9,1044.8) |
| 37.1s | S2C | 对象同步0x21[0x700991d] 码241 len=47 |
| 37.3s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [935.858, 43.887, 1044.836, 0.602] |
| 37.3s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 37.3s | S2C | S2C m26 技能释放 [] |
| 37.3s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 37.3s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 37.3s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 37.3s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 37.3s | S2C | S2C m707 受理族变体 [15] |
| 37.3s | C2S | m15 移动上报 自动 state=7 pos=(935.9,43.9,1044.8) ×2 |
| 37.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 37.4–40.3s | S2C | 对象同步0x21[0x700991d] 码201 len=91 ×2 |
| 37.6s | S2C | op28(子码47) len=225 |
| 38.2s | C2S | m82 ? [3] |
| 38.4s | C2S | m15 移动上报 手动 state=7 pos=(932.9,43.9,1040.7) |
| 38.5–41.4s | S2C | 对象同步0x21[0x70295df] 码32 len=91 ×2 |
| 38.6s | S2C | op31(sub1) 0x703e78b |
| 38.7s | C2S | m15 移动上报 手动 state=7 pos=(933.5,43.9,1041.5) |
| 38.7s | S2C | 对象同步0x21[0x700991d] 码143 len=47 |
| 38.9s | S2C | 对象同步0x21[0x700991d] 码145 len=47 |
| 39.4s | C2S | m15 移动上报 自动 state=4 pos=(935.4,43.9,1044.3) |
| 39.5s | S2C | 对象同步0x21[0x700991d] 码193 len=47 |
| 39.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "9EEB2019797B42B1A66B7C74557CDB1A"] |
| 39.5s | C2S | m15 移动上报 自动 state=5 pos=(935.4,43.9,1044.3) ×2 |
| 39.6s | S2C | op06 len=1 06 |
| 39.6s | C2S | m15 移动上报 手动 state=7 pos=(935.4,43.9,1044.3) |
| 39.6s | S2C | op31(sub1) 0x7022212 |
| 39.7s | C2S | m15 移动上报 自动 state=4 pos=(935.2,43.9,1044.0) |
| 39.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 39.8s | S2C | 对象同步0x21[0x700991d] 码198 len=47 |
| 39.9s | S2C | 对象同步0x21[0x700991d] 码46 len=47 |
| 40.1s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [935.19, 43.887, 1044.017, -2.539] |
| 40.1s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 40.1s | S2C | S2C m26 技能释放 [] |
| 40.1s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 40.1s | S2C | 属性更新0x10[O#0x3db441] CurSkillID=CS_hs_kfkj02 |
| 40.1s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 40.1s | S2C | 属性更新0x10[O#0x3db441] PauseSkill=CS_hs_kfkj02 |
| 40.1s | S2C | S2C m707 受理族变体 [15] |
| 40.1s | C2S | m15 移动上报 自动 state=7 pos=(935.2,43.9,1044.0) ×2 |
| 40.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 40.6s | S2C | 对象同步0x21[0x702ade2] 码49 len=91 |
| 40.6s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 40.6s | S2C | op31(sub1) 0x703e78b |
| 41.4s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand ×2 |
| 41.5s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 41.6s | S2C | op2C len=148 2c47930000000a010000000000e00600df950207 |
| 41.7s | S2C | op28(子码47) len=222 |
| 41.8s | S2C | 对象同步0x21[0x7000146] 码237 len=223 |
| 41.8s | S2C | 属性更新0x10[O#0x14b] State= |
| 41.8s | S2C | 多属性0x2D[0x14b0700](布局变体) len=20 2d0100460100074b0100000100540001 |
| 42.3s | C2S | m15 移动上报 自动 state=7 pos=(929.4,43.9,1035.8) |
| 42.5s | S2C | 对象同步0x21[0x700991d] 码158 len=47 |
| 42.7s | S2C | op31(sub1) 0x700ca00 |
| 42.7s | S2C | op28(子码47) len=235 |
| 42.7s | S2C | op0D len=251 0d65d900073d043b002b776944e8fb4242a15277 |
| 42.8s | C2S | m220 冲锋接近上报 [928.041, 43.887, 1033.898, -2.526] ×2 |
| 42.9s | S2C | S2C m24 通用事件[0x3db4410700991d] 冲锋动画 [925.348, 44.886, 1030.092, 1, "qg_fastrun_2h_01"] |
| 42.9s | S2C | 属性更新0x10[O#0x3db441] CantMove=1 |
| 42.9s | S2C | 属性更新0x10[O#0x3db441] CantParry=1 |
| 42.9s | S2C | 记录加行0x11[cooldown_rec#109 0x700991d/0x3db441] +97行 |
| 42.9s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x700991d] ×4 |
| 42.9s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x700991d] ×2 |
| 42.9s | S2C | 记录加行0x11[cooldown_rec#109 0x700991d/0x3db441] +98行 |
| 42.9s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x700991d] ×4 |
| 42.9s | S2C | 属性更新0x10[O#0x3db441] MPRatio=98 |
| 42.9s | S2C | 属性更新0x10[O#0x3db441] MP=10247 |
| 42.9s | S2C | 属性更新0x10[O#0x3db441] QingGongPoint=155 |
| 43.0s | C2S | m15 移动上报 自动 state=4 pos=(927.7,43.9,1033.3) |
| 43.1s | C2S | m15 移动上报 自动 state=7 pos=(927.7,43.9,1033.3) ×2 |
| 43.1s | S2C | 对象同步0x21[0x700991d] 码64 len=91 |
| 43.2s | C2S | m212 冲锋/续招落地 [925.877, 43.886, 1030.84, -2.525] |
| 43.2s | S2C | 属性更新0x10[O#0x3db441] CantMove=0 |
| 43.2s | S2C | 属性更新0x10[O#0x3db441] CantParry=0 |
| 43.2s | S2C | 位置0x1F(短len=25) |
| 43.2s | S2C | 属性更新0x10[O#0x3db441] PauseSkill= |
| 43.2s | S2C | S2C m26 技能释放 [] |
| 43.2s | S2C | 属性更新0x10[O#0x3db879] LogicState=1 |
| 43.2s | S2C | 记录加行0x11[fight_relation_rec#114 0x700991d/0x3db441] +0行 |
| 43.2s | S2C | 属性更新0x10[O#0x3db441] LogicState=1 |
| 43.2s | S2C | 位置0x1F(短len=25) |
| 43.2s | S2C | S2C m337 施法回显[0x3db4410700991d] [925.877, 43.886, 1030.84, 3.758, 0] |
| 43.2s | S2C | S2C m24 通用事件[0x3db8790704054e] 动作开始 [1, obj(0x700991d,0x3db441)] |
| 43.2s | S2C | S2C m19 锁定伤害段[0x3db4410700991d] [obj(0x704054e,0x3db879), "CS_hs_kfkj02", 3000, 3000, 926.815, 43.886] |
| 43.2s | S2C | 属性更新0x10[O#0x3db441] CantMove=1 |
| 43.2s | S2C | 属性更新0x10[O#0x3db441] CantParry=1 |
| 43.2s | C2S | m15 移动上报 自动 state=4 pos=(925.9,43.9,1030.8) |
| 43.3s | S2C | 对象同步0x21[0x704054e] 码201 len=135 |
| 43.3s | S2C | 多属性0x2D[0xb8790704](布局变体) len=23 2d01004e05040779b83d000100a0011d |
| 43.3s | C2S | m15 移动上报 自动 state=7 pos=(925.9,43.9,1030.8) ×2 |
| 43.3s | C2S | m915 命中确认上报 [1, obj(0x704054e,0x3db879), 927.148, 43.896, 1032.635, 1.442] |
| 43.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 43.4s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 43.4s | S2C | 属性更新0x10[O#0x3db441] FP=2 |
| 43.4s | S2C | 属性更新0x10[O#0x3db441] SP=2 |
| 43.4s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700991d] |
| 43.4s | S2C | S2C m1 系统提示 [2, "8066"] |
| 43.4s | S2C | S2C m24 通用事件[0x3db8790704054e] 伤害飘字 [512, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 43.4s | S2C | S2C m205 伤害目标列表[0x3db4410700991d] ["CS_hs_kfkj02", obj(0x704054e,0x3db879)] |
| 43.5s | S2C | 对象同步0x21[0x700991d] 码188 len=135 |
| 43.5s | S2C | 多属性0x2D[0xb8790704](布局变体) len=25 2d01004e05040779b83d000200d7014d |
| 43.7s | S2C | S2C m24 通用事件[0x3db4410700991d] 动作开始 ["buff_ng_xyhxj01", obj(0x700991d,0x3db441), obj(0x704054e,0x3db879)] |
| 43.7s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4=4BF9,700991D,3DB441,5,C38EB6FE3DD,1 |
| 43.7s | S2C | S2C m169 buff添加 ["buf_ng_jz_006_01", obj(0x700991d,0x3db441)] |
| 43.7s | S2C | S2C m169 buff刷新? ["buf_ng_jz_006_01", obj(0x700991d,0x3db441)] |
| 43.7s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4=4BF9,700991D,3DB441,5,C38EB6FE3DD,5 |
| 43.7s | S2C | op28(子码47) len=222 |
| 43.8s | S2C | 属性更新0x10[O#0x3db441] PhyDamageMul=1.5 |
| 43.8s | S2C | 属性更新0x10[O#0x3db441] StiDamageMul=1.5 |
| 43.8s | S2C | 属性更新0x10[O#0x3db441] JujDamageMul=1.5 |
| 43.8s | S2C | 属性更新0x10[O#0x3db441] NegDamageMul=1.5 |
| 43.8s | S2C | 属性更新0x10[O#0x3db441] MasDamageMul=1.5 |
| 43.8s | S2C | 属性更新0x10[O#0x3db441] UnderWaterModifySpeed=1.0 ×2 |
| 43.9s | S2C | 属性更新0x10[O#0x3db441] FP=4 |
| 43.9s | S2C | 属性更新0x10[O#0x3db441] SP=4 |
| 43.9s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700991d] |
| 43.9s | S2C | S2C m24 通用事件[0x3db8790704054e] 伤害飘字 [721, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 43.9s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4=4BF9,700991D,3DB441,5,C38EB6FE4E6,4 |
| 43.9s | S2C | S2C m24 通用事件[0x3db4410700991d] 怒气段 [2] |
| 43.9s | S2C | S2C m205 伤害目标列表[0x3db4410700991d] ["CS_hs_kfkj02", obj(0x704054e,0x3db879)] |
| 44.0s | S2C | 多属性0x2D[0xb8790704](布局变体) len=25 2d01004e05040779b83d000200d7012c |
| 44.1s | S2C | 属性更新0x10[O#0x3db441] PhyDamageMul=1.4 |
| 44.1s | S2C | 属性更新0x10[O#0x3db441] StiDamageMul=1.4 |
| 44.1s | S2C | 属性更新0x10[O#0x3db441] JujDamageMul=1.4 |
| 44.1s | S2C | 属性更新0x10[O#0x3db441] NegDamageMul=1.4 |
| 44.1s | S2C | 属性更新0x10[O#0x3db441] MasDamageMul=1.4 |
| 44.1s | S2C | 属性更新0x10[O#0x3db441] UnderWaterModifySpeed=1.0 ×2 |
| 44.2s | S2C | 属性更新0x10[O#0x3db441] FP=6 |
| 44.2s | S2C | 属性更新0x10[O#0x3db441] SP=6 |
| 44.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700991d] |
| 44.2s | S2C | S2C m24 通用事件[0x3db8790704054e] 伤害飘字 [679, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 44.2s | S2C | 属性更新0x10[O#0x3db441] BufferInfo4=4BF9,700991D,3DB441,5,C38EB6FE5DF,3 |
| 44.2s | S2C | S2C m24 通用事件[0x3db4410700991d] 怒气段 [3] |
| 44.2s | S2C | S2C m205 伤害目标列表[0x3db4410700991d] ["CS_hs_kfkj02", obj(0x704054e,0x3db879)] |
| 44.2s | S2C | 多属性0x2D[0xb8790704](布局变体) len=25 2d01004e05040779b83d000200d7010e |
| 44.3s | S2C | 属性更新0x10[O#0x3db441] QingGongPoint=165 |
| 44.3s | S2C | 属性更新0x10[O#0x3db441] PhyDamageMul=1.3 |
| 44.3s | S2C | 属性更新0x10[O#0x3db441] StiDamageMul=1.3 |
| 44.3s | S2C | 属性更新0x10[O#0x3db441] JujDamageMul=1.3 |
| 44.3s | S2C | 属性更新0x10[O#0x3db441] NegDamageMul=1.3 |
| 44.3s | S2C | 属性更新0x10[O#0x3db441] MasDamageMul=1.3 |
| 44.3s | S2C | 属性更新0x10[O#0x3db441] UnderWaterModifySpeed=1.0 ×2 |
| 44.5s | S2C | 属性更新0x10[O#0x3db441] FP=8 |
| 44.5s | S2C | 属性更新0x10[O#0x3db441] SP=8 |
| 44.5s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x700991d] |
| 44.5s | S2C | S2C m24 通用事件[0x3db8790704054e] 伤害飘字 [320, 317, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 44.5s | S2C | 记录加行0x11[JhBoss_Accept_Rec#22 0x700991d/0x3db441] +0行 |
| 44.5s | S2C | S2C m446 区域boss进度 [1, 146, "quyuboss_emei2", 20, 160, "ui_jh_001"] |
| 44.5s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x700991d] |
| 44.5s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x700991d] |
| 44.5s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x700991d] ×2 |
| 44.5s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 27, 160, "ui_jh_001"] |
| 44.5s | S2C | S2C m24 通用事件[0x3db8790704054e] 击退 [obj(0x700991d,0x3db441), "monm031301", "徐傠囃辭睼 "] |
| 44.5s | S2C | S2C m24 通用事件[0x3db4410700991d] 怒气段 [4] |
| 44.5s | S2C | S2C m205 伤害目标列表[0x3db4410700991d] ["CS_hs_kfkj02", obj(0x704054e,0x3db879)] |
| 44.7s | S2C | 对象同步0x21[0x704054e] 码72 len=47 |
| 44.7s | S2C | 多属性0x2D[0xb8790704](布局变体) len=59 2d01004e05040779b83d000600d70100 |
| 44.7s | S2C | op0D len=251 0d16d70007e5b63b009e676444b88b2f42c3b275 |
| 44.9s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 45.5s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "9EEB2019797B42B1A66B7C74557CDB1A"] |
| 45.6s | C2S | m15 移动上报 自动 state=5 pos=(926.5,43.9,1031.7) ×2 |
| 45.6s | S2C | 属性更新0x10[O#0x3db441] CurSkillID= |
| 45.6s | S2C | 属性更新0x10[O#0x3db441] CurSkillTarget=None |
| 45.7s | S2C | op31(sub1) 0x700d716 |
| 46.2s | S2C | 属性更新0x10[O#0x3db441] CantMove=0 |
| 46.2s | S2C | 属性更新0x10[O#0x3db441] CantParry=0 |
| 46.2s | C2S | m990 ? [obj(0x704054e,0x3db879), 925.004, 43.896, 1029.607] |
| 46.2s | S2C | 位置0x1F(短len=25) |
| 46.2s | C2S | m914 招式结束异步位置 [926.6, 43.886, 1031.861, -2.526, 12, 2] |
| 46.2s | C2S | m15 移动上报 手动 state=7 pos=(926.6,43.9,1031.9) |
| 46.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 46.4s | S2C | 对象同步0x21[0x700991d] 码95 len=135 |
| 46.4s | C2S | m15 移动上报 手动 state=4 pos=(927.1,43.9,1032.6) |
| 46.5s | C2S | m15 移动上报 手动 state=7 pos=(927.1,43.9,1032.6) |
| 46.5s | S2C | 对象同步0x21[0x700991d] 码144 len=47 |
| 46.6s | S2C | 属性更新0x10[O#0x3db879] LogicState=0 |
| 46.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 46.8s | S2C | 对象同步0x21[0x700991d] 码139 len=47 |
| 47.6s | C2S | m15 移动上报 自动 state=4 pos=(930.1,43.9,1036.9) |
| 47.6s | S2C | 属性更新0x10[O#0x3db441] MPRatio=99 |
| 47.6s | S2C | 属性更新0x10[O#0x3db441] MP=10347 |
| 47.7s | S2C | op31(sub2) 0x7000146 →0x700fed1 |
| 47.8s | S2C | 对象同步0x21[0x700991d] 码160 len=47 |
| 48.6s | S2C | 属性更新0x10[O#0x3db441] LogicState=2 |
| 48.7s | S2C | op0E len=9 0e4e05040779b83d00 |
| 48.7s | S2C | op31(sub1) 0x700014b |
| 48.7s | C2S | m440 ? [] |
| 48.7s | S2C | 属性更新0x10[O#0x3db441] LastObject=None |
| 48.7s | C2S | m431 选择目标 [0] |
| 48.8s | S2C | 属性更新0x10[O#0x3db441] AutoSelectObject=None |
| 49.5s | S2C | 属性更新0x10[O#0x3db441] SP=0 |

## g.pcapng — 103.143.19.173:7072（与 i 同端口，前半段）

场景：kfkj 战斗 + 击杀 + NPC 对话（ui_npcclonetl）。事件 1627 条 → 合并为 1173 段。抓包起点 2026-11-08 21:15:48（本地墙钟）

| 时间 | 方向 | 交互 |
|---|---|---|
| 0.0–67.6s | PAIR | C2S m924 MSG_DELAY(client ms) → S2C m294 回显[client ms, obj, 服务器unix秒, …] ×68 |
| 0.0–67.6s | PAIR | C2S ping(0x18) → S2C 0x38 回显 ×60 |
| 1.2s | C2S | m54 格挡状态(值12649500227) |
| 1.2s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 1.2s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 1.2s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 1.2s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 1.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 1.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 1.2s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 1.3s | C2S | m54 格挡状态(值12425460252) |
| 1.4s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 1.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 1.4s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 1.4s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 1.4s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 1.4s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 1.4s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 1.4s | C2S | m54 格挡状态(值12425460252) ×3 |
| 1.6s | S2C | 属性更新0x10[O#0x150] State= |
| 1.7s | C2S | m15 移动上报 自动 state=4 pos=(932.5,43.9,1037.1) |
| 1.8s | C2S | m54 格挡状态(值12649500227) |
| 1.8s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 1.8s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 1.8s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 1.8s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 1.8s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 1.8s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 1.8s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 1.9s | S2C | op2C len=132 2c47830000000a010000000000e006004b010007 |
| 1.9s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 1.9s | C2S | m54 格挡状态(值12425460252) |
| 2.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 2.0s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 2.0s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 2.0s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 2.0s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 2.0s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 2.0s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 2.0s | C2S | m54 格挡状态(值12425460252) ×2 |
| 2.2s | C2S | m15 移动上报 自动 state=4 pos=(932.5,43.9,1037.1) |
| 2.3–4.5s | S2C | 对象同步0x21[0x7034e9b] 码108 len=47 ×2 |
| 3.2s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 3.3s | C2S | m15 移动上报 自动 state=5 pos=(932.5,43.9,1037.1) ×2 |
| 3.7s | C2S | 对象请求 len=33 |
| 3.7s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 3.7s | C2S | m431 选择目标 [obj(0x700bbff,0x499e06)] |
| 3.7s | C2S | 对象请求 len=33 |
| 3.7s | C2S | m431 选择目标 [obj(0x700bbff,0x499e06)] |
| 3.8s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 4.0s | C2S | m54 格挡状态(值12649500227) |
| 4.0s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 4.0s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 4.0s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 4.0s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 4.0s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 4.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 4.0s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 4.1s | C2S | m54 格挡状态(值12425460252) |
| 4.2s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 4.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 4.2s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 4.2s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 4.2s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 4.2s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 4.2s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 4.2s | C2S | m54 格挡状态(值12425460252) ×2 |
| 4.3s | C2S | m15 移动上报 自动 state=4 pos=(932.5,43.9,1037.1) |
| 4.3s | C2S | m15 移动上报 手动 state=7 pos=(932.5,43.9,1037.1) |
| 4.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 4.8s | C2S | m54 格挡状态(值12649500227) |
| 4.9s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 4.9s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 4.9s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 4.9s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 4.9s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 4.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 4.9s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 4.9s | C2S | m15 移动上报 手动 state=7 pos=(933.5,43.9,1034.7) ×2 |
| 5.0s | C2S | m54 格挡状态(值12425460252) |
| 5.1s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 5.1s | C2S | m54 格挡状态(值12425460252) ×2 |
| 5.1–5.3s | S2C | 对象同步0x21[0x7034e9b] 码0 len=91 ×2 |
| 5.1s | C2S | m15 移动上报 手动 state=7 pos=(933.7,43.9,1034.4) |
| 5.1s | C2S | m54 格挡状态(值12425460252) |
| 5.1s | C2S | m15 移动上报 自动 state=4 pos=(933.7,43.9,1034.3) |
| 5.8s | C2S | m54 格挡状态(值12649500227) |
| 5.9s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 5.9s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 5.9s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 5.9s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 5.9s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 5.9s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 5.9s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 6.0s | C2S | m54 格挡状态(值12425460252) |
| 6.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 6.0s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 6.0s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 6.0s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 6.0s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 6.0s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 6.0s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 6.0s | C2S | m54 格挡状态(值12425460252) ×3 |
| 6.2s | C2S | m15 移动上报 自动 state=4 pos=(933.7,43.9,1034.3) |
| 6.4s | S2C | 对象同步0x21[0x7034e9b] 码240 len=47 |
| 6.6s | S2C | op06 len=1 06 |
| 6.7s | C2S | 对象请求 len=33 |
| 6.7s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 6.7s | C2S | m431 选择目标 [obj(0x7003c31,0x499e0b)] |
| 6.7s | C2S | 对象请求 len=33 |
| 6.7s | C2S | m431 选择目标 [obj(0x7003c31,0x499e0b)] |
| 6.8s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 6.9s | C2S | m15 移动上报 ?2000 state=7 pos=(933.7,43.9,1034.3) |
| 6.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 7.1s | S2C | 对象同步0x21[0x7034e9b] 码45 len=47 |
| 7.7s | C2S | m15 移动上报 自动 state=4 pos=(930.5,43.9,1035.9) |
| 8.0–8.4s | S2C | 对象同步0x21[0x7034e9b] 码127 len=47 ×2 |
| 8.2s | C2S | m54 格挡状态(值12649500227) |
| 8.2s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 8.2s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 8.2s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 8.2s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 8.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 8.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 8.2s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 8.3s | S2C | 属性更新0x10[O#0x498afc] FightData=17141115015 |
| 8.3s | C2S | m15 移动上报 手动 state=4 pos=(930.5,43.9,1035.9) |
| 8.4s | C2S | m54 格挡状态(值3876302951) ×2 |
| 8.5s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 8.5s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 8.5s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 8.5s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 8.5s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 8.5s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 8.5s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 8.5s | C2S | m54 格挡状态(值3876302951) ×2 |
| 8.6s | C2S | m15 移动上报 手动 state=7 pos=(930.5,43.9,1035.9) |
| 8.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 8.8s | C2S | m15 移动上报 自动 state=4 pos=(929.9,43.9,1035.1) |
| 8.8s | S2C | 对象同步0x21[0x7034e9b] 码120 len=47 |
| 9.1s | S2C | 对象同步0x21[0x7034e9b] 码211 len=47 |
| 9.2s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 9.3s | C2S | m15 移动上报 自动 state=5 pos=(929.9,43.9,1035.1) ×2 |
| 9.4s | C2S | m15 移动上报 自动 state=7 pos=(929.9,43.9,1035.1) ×2 |
| 9.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 9.6s | C2S | m15 移动上报 自动 state=4 pos=(929.0,43.9,1034.8) |
| 9.6s | S2C | 对象同步0x21[0x7034e9b] 码9 len=91 |
| 9.7s | S2C | 对象同步0x21[0x7034e9b] 码12 len=47 |
| 9.9s | C2S | m15 移动上报 手动 state=7 pos=(929.0,43.9,1034.8) |
| 9.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 10.0s | C2S | m15 移动上报 自动 state=4 pos=(929.4,43.9,1035.4) |
| 10.1s | S2C | 对象同步0x21[0x7034e9b] 码18 len=91 |
| 10.3s | C2S | m15 移动上报 手动 state=7 pos=(929.4,43.9,1035.4) |
| 10.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 10.4s | S2C | 对象同步0x21[0x7034e9b] 码5 len=47 |
| 10.5s | C2S | m15 移动上报 手动 state=7 pos=(928.8,43.9,1034.5) |
| 10.6s | S2C | 对象同步0x21[0x7034e9b] 码193 len=47 |
| 10.6s | C2S | m15 移动上报 自动 state=4 pos=(928.5,43.9,1034.0) |
| 10.8s | S2C | 对象同步0x21[0x7034e9b] 码87 len=47 |
| 11.3s | S2C | 对象同步0x21[0x70295df] 码32 len=91 |
| 11.3s | C2S | m440 ? [] |
| 11.4s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 11.4s | C2S | m431 选择目标 [0] |
| 11.4s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 12.3s | S2C | S2C m1 系统提示 [0, "30359", "荆小龙(凌) ", "bossworld001_small"] |
| 12.3s | S2C | S2C m1 系统提示 [1, "sys_world_auction_bossworld001_small"] |
| 12.4s | S2C | S2C m603 ? [0, 0] |
| 12.6s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 12.6s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 12.6s | C2S | m82 ? [3] |
| 12.6s | S2C | S2C m250 ? [] ×2 |
| 13.3s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 13.4s | S2C | 对象同步0x21[0x702ade2] 码49 len=91 |
| 14.1s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand |
| 14.3s | S2C | 对象同步0x21[0x70295df] 码32 len=91 |
| 14.4s | C2S | 对象请求 len=33 |
| 14.5s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 14.5s | C2S | m431 选择目标 [obj(0x7003c31,0x499e0b)] |
| 14.5s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [928.458, 43.887, 1034.021, -2.52] |
| 14.5s | C2S | m15 移动上报 自动 state=7 pos=(928.5,43.9,1034.0) ×2 |
| 14.5s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 14.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 14.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_hs_kfkj02 |
| 14.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 14.5s | S2C | S2C m25 技能暂存(skill_pause) [30000, "skill_pause"] |
| 14.5s | S2C | 属性更新0x10[O#0x498afc] PauseSkill=CS_hs_kfkj02 |
| 14.5s | S2C | S2C m707 受理族变体 [15] |
| 14.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 14.5s | C2S | m220 冲锋接近上报 [928.14, 43.887, 1033.609, -2.484] ×2 |
| 14.6s | S2C | 对象同步0x21[0x7034e9b] 码201 len=91 |
| 14.7s | S2C | S2C m24 通用事件[0x498afc07034e9b] 冲锋动画 [925.398, 44.886, 1030.055, 1, "qg_fastrun_2h_01"] |
| 14.7s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 14.7s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 14.7s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 14.7s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 14.7s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 14.7s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 14.7s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 14.7s | S2C | 属性更新0x10[O#0x498afc] MPRatio=98 |
| 14.7s | S2C | 属性更新0x10[O#0x498afc] MP=10247 |
| 14.7s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=155 |
| 14.7s | S2C | 属性更新0x10[O#0x3bbc1c] State=walk |
| 14.7s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 14.7s | C2S | m15 移动上报 自动 state=4 pos=(927.7,43.9,1033.1) |
| 14.7s | S2C | S2C m707 受理族变体 [29] |
| 14.7s | C2S | m15 移动上报 自动 state=7 pos=(927.7,43.9,1033.1) ×2 |
| 14.8s | S2C | op2C len=204 2c47cb000000620100000000008008009b4e0307 |
| 14.9s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=165 |
| 14.9s | C2S | m212 冲锋/续招落地 [925.917, 43.886, 1030.729, -2.484] |
| 14.9s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 14.9s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 14.9s | S2C | 位置0x1F(短len=25) |
| 14.9s | S2C | 属性更新0x10[O#0x498afc] PauseSkill= |
| 14.9s | S2C | S2C m26 技能释放 [] |
| 14.9s | S2C | 属性更新0x10[O#0x499e0b] LogicState=1 |
| 14.9s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +0行 |
| 14.9s | S2C | 属性更新0x10[O#0x498afc] LogicState=1 |
| 14.9s | S2C | 位置0x1F(短len=25) |
| 14.9s | S2C | S2C m337 施法回显[0x498afc07034e9b] [925.917, 43.886, 1030.729, 3.799, 0] |
| 14.9s | S2C | S2C m24 通用事件[0x499e0b07003c31] 动作开始 [1, obj(0x7034e9b,0x498afc)] |
| 14.9s | S2C | S2C m19 锁定伤害段[0x498afc07034e9b] [obj(0x7003c31,0x499e0b), "CS_hs_kfkj02", 3000, 3000, 926.95, 43.886] |
| 14.9s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 14.9s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 14.9s | C2S | m15 移动上报 自动 state=4 pos=(925.9,43.9,1030.7) |
| 15.0s | S2C | 对象同步0x21[0x7003c31] 码201 len=91 |
| 15.0s | C2S | m15 移动上报 自动 state=7 pos=(925.9,43.9,1030.7) ×2 |
| 15.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 15.0s | C2S | m915 命中确认上报 [1, obj(0x7003c31,0x499e0b), 927.301, 43.896, 1032.522, 1.442] |
| 15.1s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 15.1s | S2C | 属性更新0x10[O#0x498afc] FP=2 |
| 15.1s | S2C | 属性更新0x10[O#0x498afc] SP=2 |
| 15.1s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 15.1s | S2C | S2C m1 系统提示 [2, "8066"] |
| 15.1s | S2C | S2C m24 通用事件[0x499e0b07003c31] 伤害飘字 [512, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 15.1s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x7003c31,0x499e0b)] |
| 15.2s | S2C | 对象同步0x21[0x7034e9b] 码180 len=179 |
| 15.2s | S2C | 多属性0x2D[0x9e0b0700](布局变体) len=23 2d0100313c00070b9e49000100a0019b |
| 15.2s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 15.2s | C2S | m15 移动上报 自动 state=5 pos=(924.8,43.9,1029.3) ×2 |
| 15.4s | S2C | 多属性0x2D[0x9e0b0700](布局变体) len=25 2d0100313c00070b9e49000200d7014d |
| 15.6s | S2C | 属性更新0x10[O#0x498afc] FP=4 |
| 15.6s | S2C | 属性更新0x10[O#0x498afc] SP=4 |
| 15.6s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 15.6s | S2C | S2C m24 通用事件[0x499e0b07003c31] 伤害飘字 [512, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 15.6s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [2] |
| 15.6s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x7003c31,0x499e0b)] |
| 15.8s | S2C | 属性更新0x10[O#0x498afc] FP=6 |
| 15.8s | S2C | 属性更新0x10[O#0x498afc] SP=6 |
| 15.8s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 15.8s | S2C | S2C m24 通用事件[0x499e0b07003c31] 伤害飘字 [512, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 15.8s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [3] |
| 15.8s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x7003c31,0x499e0b)] |
| 15.9s | S2C | 多属性0x2D[0x9e0b0700](布局变体) len=25 2d0100313c00070b9e49000200d70136 ×2 |
| 16.2s | S2C | 属性更新0x10[O#0x498afc] FP=8 |
| 16.2s | S2C | 属性更新0x10[O#0x498afc] SP=8 |
| 16.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 16.2s | S2C | S2C m24 通用事件[0x499e0b07003c31] 伤害飘字 [512, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 16.2s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [4] |
| 16.2s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x7003c31,0x499e0b)] |
| 16.4s | S2C | S2C m24 通用事件[0x498afc07034e9b] 动作开始 ["buff_ng_xyhxj01", obj(0x7034e9b,0x498afc), obj(0x7003c31,0x499e0b)] |
| 16.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC03E5AD,1 |
| 16.4s | S2C | S2C m169 buff添加 ["buf_ng_jz_006_01", obj(0x7034e9b,0x498afc)] |
| 16.4s | S2C | S2C m169 buff刷新? ["buf_ng_jz_006_01", obj(0x7034e9b,0x498afc)] |
| 16.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC03E5AD,5 |
| 16.5s | S2C | 多属性0x2D[0x9e0b0700](布局变体) len=25 2d0100313c00070b9e49000200d70108 |
| 16.5s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.5 |
| 16.5s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.5 |
| 16.5s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.5 |
| 16.5s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.5 |
| 16.5s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.5 |
| 16.5s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 16.5s | S2C | 属性更新0x10[O#0x498afc] FP=10 |
| 16.5s | S2C | 属性更新0x10[O#0x498afc] SP=10 |
| 16.5s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 16.5s | S2C | S2C m24 通用事件[0x499e0b07003c31] 伤害飘字 [184, 535, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 16.5s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC03E62B,4 |
| 16.5s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x7034e9b] |
| 16.5s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 16.5s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x7034e9b] ×2 |
| 16.5s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 35, 160, "ui_jh_001"] |
| 16.5s | S2C | S2C m24 通用事件[0x499e0b07003c31] 击退 [obj(0x7034e9b,0x498afc), "monm031301", "徐傠囃辭睼 "] |
| 16.5s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [5] |
| 16.5s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x7003c31,0x499e0b)] |
| 16.6s | S2C | op0D len=251 0d16d70007e5b63b009e676444b88b2f42c3b275 |
| 16.7s | S2C | 对象同步0x21[0x7003c31] 码49 len=47 |
| 16.7s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.4 |
| 16.7s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.4 |
| 16.7s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.4 |
| 16.7s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.4 |
| 16.7s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.4 |
| 16.7s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 16.9s | S2C | 多属性0x2D[0x9e0b0700](布局变体) len=59 2d0100313c00070b9e49000600d70100 |
| 17.3s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 17.3s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 17.3s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 17.6s | S2C | op31(sub1) 0x700d716 |
| 17.9s | C2S | m990 ? [obj(0x7003c31,0x499e0b), 926.157, 43.896, 1031.039] |
| 17.9s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 17.9s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 17.9s | C2S | m914 招式结束异步位置 [926.721, 43.886, 1031.771, -2.484, 12, 4] |
| 17.9s | S2C | 位置0x1F(短len=25) |
| 18.0s | S2C | 对象同步0x21[0x7034e9b] 码45 len=91 |
| 18.1s | C2S | m54 格挡状态(值4100342926) |
| 18.2s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 18.2s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 18.2s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 18.2s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 18.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5=11CE,7034E9B,498AFC,1,0,0 |
| 18.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 18.2s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 18.4s | C2S | m54 格挡状态(值3876302951) |
| 18.4s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 18.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5= |
| 18.4s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 18.4s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 18.4s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 18.4s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 18.4s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 18.4s | C2S | m54 格挡状态(值3876302951) ×3 |
| 18.5s | S2C | 属性更新0x10[O#0x499e0b] LogicState=0 |
| 18.8s | C2S | m15 移动上报 ?1000 state=7 pos=(926.7,43.9,1031.8) |
| 18.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 18.9s | S2C | 对象同步0x21[0x7034e9b] 码83 len=47 |
| 19.1s | C2S | m15 移动上报 ?1000 state=7 pos=(927.6,43.9,1032.3) |
| 19.1s | S2C | 对象同步0x21[0x7034e9b] 码208 len=47 |
| 19.3s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 19.3s | S2C | 属性更新0x10[O#0x498afc] MP=10347 |
| 19.5s | S2C | 属性更新0x10[O#0x498afc] LogicState=2 |
| 20.4s | C2S | m15 移动上报 自动 state=4 pos=(932.9,43.9,1035.3) |
| 20.6s | C2S | m15 移动上报 手动 state=7 pos=(932.9,43.9,1035.3) |
| 20.6s | S2C | 对象同步0x21[0x7034e9b] 码229 len=47 |
| 20.7s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 20.7s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 20.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 20.8s | S2C | 对象同步0x21[0x7034e9b] 码0 len=47 |
| 20.9s | S2C | 属性更新0x10[O#0x498afc] FP=0 |
| 21.1s | S2C | 属性更新0x10[O#0x498afc] SP=0 |
| 21.2s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 21.2s | C2S | m15 移动上报 手动 state=5 pos=(934.3,43.9,1032.9) ×2 |
| 21.5s | S2C | op0E len=9 0e313c00070b9e4900 |
| 21.5s | C2S | m15 移动上报 自动 state=4 pos=(935.0,43.9,1031.8) |
| 21.6s | S2C | op31(sub1) 0x70095b5 |
| 21.7s | S2C | 对象同步0x21[0x7034e9b] 码251 len=47 |
| 21.9s | S2C | op06 len=1 06 |
| 22.1s | C2S | 对象请求 len=33 |
| 22.1s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 22.1s | C2S | m431 选择目标 [obj(0x700bbff,0x499e06)] |
| 22.1s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [934.968, 43.887, 1031.767, 2.618] |
| 22.1s | C2S | m15 移动上报 自动 state=7 pos=(935.0,43.9,1031.8) ×2 |
| 22.2s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 22.2s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 22.2s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_hs_kfkj02 |
| 22.2s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 22.2s | S2C | S2C m284 空值标记帧 [] |
| 22.2s | S2C | 属性更新0x10[O#0x499e06] LogicState=1 |
| 22.2s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +0行 |
| 22.2s | S2C | 属性更新0x10[O#0x498afc] LogicState=1 |
| 22.2s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 22.2s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 22.2s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 22.2s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 22.2s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 22.2s | S2C | 属性更新0x10[O#0x498afc] MPRatio=98 |
| 22.2s | S2C | 属性更新0x10[O#0x498afc] MP=10206 |
| 22.2s | S2C | 位置0x1F(短len=25) |
| 22.2s | S2C | S2C m337 施法回显[0x498afc07034e9b] [934.968, 43.886, 1031.767, 2.84, 0] |
| 22.2s | S2C | S2C m24 通用事件[0x499e060700bbff] 动作开始 [1, obj(0x7034e9b,0x498afc)] |
| 22.2s | S2C | S2C m19 锁定伤害段[0x498afc07034e9b] [obj(0x700bbff,0x499e06), "CS_hs_kfkj02", 3000, 3000, 934.678, 43.886] |
| 22.2s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 22.2s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 22.2s | C2S | m990 ? ["117455921-4824587", 926.157, 43.896, 1031.039] |
| 22.2s | C2S | m15 移动上报 自动 state=7 pos=(935.0,43.9,1031.8) ×2 |
| 22.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 22.3s | C2S | m915 命中确认上报 [1, obj(0x700bbff,0x499e06), 934.507, 43.896, 1033.25, 1.442] |
| 22.3s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 22.3s | S2C | 属性更新0x10[O#0x498afc] FP=2 |
| 22.3s | S2C | 属性更新0x10[O#0x498afc] SP=2 |
| 22.3s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 22.3s | S2C | S2C m24 通用事件[0x499e060700bbff] 伤害飘字 [681, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 22.3s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC03E958,3 |
| 22.3s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x700bbff,0x499e06)] |
| 22.4s | S2C | 对象同步0x21[0x700bbff] 码49 len=223 |
| 22.5s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.3 |
| 22.5s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.3 |
| 22.5s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.3 |
| 22.5s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.3 |
| 22.5s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.3 |
| 22.5s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 22.6s | S2C | 多属性0x2D[0x9e060700](布局变体) len=35 2d0100ffbb0007069e49000300a0019b |
| 22.8s | S2C | 属性更新0x10[O#0x498afc] FP=4 |
| 22.8s | S2C | 属性更新0x10[O#0x498afc] SP=4 |
| 22.8s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 22.8s | S2C | S2C m24 通用事件[0x499e060700bbff] 伤害飘字 [639, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 22.8s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC03E762,2 |
| 22.8s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [2] |
| 22.8s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x700bbff,0x499e06)] |
| 23.0s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.2 |
| 23.0s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.2 |
| 23.0s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.2 |
| 23.0s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.2 |
| 23.0s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.2 |
| 23.0s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 23.1s | S2C | 属性更新0x10[O#0x498afc] FP=6 |
| 23.1s | S2C | 属性更新0x10[O#0x498afc] SP=6 |
| 23.1s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 23.1s | S2C | S2C m24 通用事件[0x499e060700bbff] 伤害飘字 [597, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 23.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC03E85C,1 |
| 23.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [3] |
| 23.1s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x700bbff,0x499e06)] |
| 23.2s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.1 |
| 23.2s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.1 |
| 23.2s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.1 |
| 23.2s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.1 |
| 23.2s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.1 |
| 23.2s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 23.3s | S2C | 多属性0x2D[0x9e060700](布局变体) len=25 2d0100ffbb0007069e49000200d70128 |
| 23.5s | S2C | 对象同步0x21[0x700bbff] 码51 len=47 |
| 23.5s | S2C | 属性更新0x10[O#0x498afc] FP=8 |
| 23.5s | S2C | 属性更新0x10[O#0x498afc] SP=8 |
| 23.5s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 23.5s | S2C | S2C m24 通用事件[0x499e060700bbff] 伤害飘字 [315, 240, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 23.5s | S2C | S2C m169 buff移除 ["buf_ng_jz_006_01", obj(0x7034e9b,0x498afc)] |
| 23.5s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 23.5s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x7034e9b] |
| 23.5s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 23.5s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x7034e9b] ×2 |
| 23.5s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 42, 160, "ui_jh_001"] |
| 23.5s | S2C | S2C m24 通用事件[0x499e060700bbff] 击退 [obj(0x7034e9b,0x498afc), "monm031301", "徐傠囃辭睼 "] |
| 23.5s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [4] |
| 23.5s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x700bbff,0x499e06)] |
| 23.5s | S2C | 多属性0x2D[0x9e060700](布局变体) len=25 2d0100ffbb0007069e49000200d7010e |
| 23.6s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.0 |
| 23.6s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.0 |
| 23.6s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.0 |
| 23.6s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.0 |
| 23.6s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.0 |
| 23.6s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 23.7s | S2C | 多属性0x2D[0x9e060700](布局变体) len=59 2d0100ffbb0007069e49000600d70100 |
| 23.8s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 24.0s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 24.0s | S2C | 属性更新0x10[O#0x498afc] MP=10306 |
| 24.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 24.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 25.1s | C2S | m990 ? [obj(0x700bbff,0x499e06), 935.578, 43.896, 1029.807] |
| 25.1s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 25.1s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 25.1s | C2S | m914 招式结束异步位置 [934.789, 43.886, 1032.343, 2.84, 12, 5] |
| 25.2s | S2C | 位置0x1F(短len=25) |
| 25.2s | C2S | m54 格挡状态(值3876302951) |
| 25.2s | S2C | 对象同步0x21[0x7034e9b] 码132 len=91 |
| 25.2s | C2S | m15 移动上报 手动 state=7 pos=(934.8,43.9,1032.3) |
| 25.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 25.4s | S2C | 对象同步0x21[0x7034e9b] 码102 len=47 |
| 25.5s | S2C | 属性更新0x10[O#0x499e06] LogicState=0 |
| 25.6–26.6s | S2C | op31(sub1) 0x700014b ×2 |
| 26.9–28.5s | S2C | 对象同步0x21[0x70295df] 码149 len=91 ×2 |
| 26.9s | S2C | 属性更新0x10[O#0x150] State=stand ×2 |
| 27.1s | S2C | 对象同步0x21[0x700014b] 码96 len=223 |
| 27.1s | S2C | 多属性0x2D[0x1500700](布局变体) len=23 2d01004b010007500100000100540004 |
| 27.2s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 27.3s | C2S | m15 移动上报 手动 state=5 pos=(943.6,43.9,1030.4) ×2 |
| 27.4s | S2C | S2C m1 系统提示 [3, "15917", "徐席梦云 ", "ini\scene\clone047", 2250] |
| 27.5s | C2S | m15 移动上报 手动 state=7 pos=(944.6,43.9,1030.2) |
| 27.5s | S2C | 属性更新0x10[O#0x498afc] LogicState=2 |
| 27.6s | S2C | 对象同步0x21[0x7034e9b] 码51 len=47 |
| 27.7s | S2C | op28(子码47) len=255 |
| 27.7s | S2C | op0E len=9 0effbb0007069e4900 |
| 27.7s | C2S | m440 ? [] |
| 27.8s | C2S | m431 选择目标 [0] |
| 27.8s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 27.8s | S2C | 属性更新0x10[O#0x3bbc21] State=stand |
| 27.9s | C2S | m15 移动上报 自动 state=4 pos=(946.6,43.9,1029.8) |
| 27.9s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 28.0s | S2C | 对象同步0x21[0x702ade2] 码97 len=135 |
| 28.0s | C2S | m54 格挡状态(值4100342926) |
| 28.1s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 28.1s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 28.1s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 28.1s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 28.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 28.1s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 28.1s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 28.2s | C2S | m15 移动上报 手动 state=4 pos=(946.6,43.9,1029.8) |
| 28.2s | S2C | 属性更新0x10[O#0x3bbc1c] State=stand |
| 28.3s | C2S | m54 格挡状态(值3876302951) ×4 |
| 28.4s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 28.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 28.4s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 28.4s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 28.4s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 28.4s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 28.4s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 28.4s | S2C | 属性更新0x10[O#0x498afc] SP=0 |
| 28.4s | C2S | m54 格挡状态(值3876302951) ×2 |
| 28.5s | C2S | m15 移动上报 手动 state=7 pos=(946.6,43.9,1029.8) |
| 28.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 28.6s | C2S | m15 移动上报 自动 state=4 pos=(947.0,43.9,1029.6) |
| 28.6s | S2C | 属性更新0x10[O#0x3bbc1c] State=walk |
| 28.6s | S2C | 属性更新0x10[O#0x3bbc21] State=walk |
| 28.6s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 28.6s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 28.6s | S2C | op28(子码47) len=234 |
| 28.7s | S2C | op2C len=201 2c47c8000000620100000000008008009b4e0307 |
| 29.4s | C2S | 对象请求 len=33 |
| 29.4s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 29.4s | C2S | m431 选择目标 [obj(0x700e7c0,0x499e12)] |
| 29.4s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [946.963, 43.887, 1029.569, 2.136] |
| 29.5s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 29.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 29.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_hs_kfkj02 |
| 29.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 29.5s | S2C | S2C m284 空值标记帧 [] |
| 29.5s | S2C | 属性更新0x10[O#0x499e12] LogicState=1 |
| 29.5s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +0行 |
| 29.5s | S2C | 属性更新0x10[O#0x498afc] LogicState=1 |
| 29.5s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 29.5s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 29.5s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 29.5s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 29.5s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 29.5s | S2C | 属性更新0x10[O#0x498afc] MPRatio=98 |
| 29.5s | S2C | 属性更新0x10[O#0x498afc] MP=10247 |
| 29.5s | S2C | 位置0x1F(短len=25) |
| 29.5s | S2C | S2C m337 施法回显[0x498afc07034e9b] [946.963, 43.886, 1029.569, 2.414, 0] |
| 29.5s | S2C | S2C m24 通用事件[0x499e120700e7c0] 动作开始 [1, obj(0x7034e9b,0x498afc)] |
| 29.5s | S2C | S2C m19 锁定伤害段[0x498afc07034e9b] [obj(0x700e7c0,0x499e12), "CS_hs_kfkj02", 3000, 3000, 945.144, 43.886] |
| 29.5s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 29.5s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 29.5s | C2S | m15 移动上报 自动 state=7 pos=(947.0,43.9,1029.6) |
| 29.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 29.5s | C2S | m15 移动上报 自动 state=7 pos=(946.8,43.9,1029.7) |
| 29.5s | C2S | m915 命中确认上报 [1, obj(0x700e7c0,0x499e12), 944.761, 43.896, 1032.041, 1.442] |
| 29.6s | S2C | 对象同步0x21[0x700e7c0] 码47 len=135 |
| 29.6s | S2C | 多属性0x2D[0x9e120700](布局变体) len=23 2d0100c0e70007129e49000100a0019b |
| 29.7s | S2C | S2C m586 状态推送(簇首) [5] |
| 29.7s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 29.7s | S2C | 属性更新0x10[O#0x498afc] FP=10 |
| 29.7s | S2C | 属性更新0x10[O#0x498afc] SP=2 |
| 29.7s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 29.7s | S2C | S2C m24 通用事件[0x499e120700e7c0] 伤害飘字 [513, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 29.7s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x700e7c0,0x499e12)] |
| 29.8–32.6s | S2C | 对象同步0x21[0x7034e9b] 码23 len=91 ×2 |
| 29.8s | S2C | 多属性0x2D[0x9e120700](布局变体) len=25 2d0100c0e70007129e49000200d7014d |
| 30.2s | S2C | 属性更新0x10[O#0x498afc] FP=12 |
| 30.2s | S2C | 属性更新0x10[O#0x498afc] SP=4 |
| 30.2s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 30.2s | S2C | S2C m24 通用事件[0x499e120700e7c0] 伤害飘字 [929, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 30.2s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [2] |
| 30.2s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x700e7c0,0x499e12)] |
| 30.2s | S2C | 多属性0x2D[0x9e120700](布局变体) len=25 2d0100c0e70007129e49000200d70123 |
| 30.4s | S2C | 对象同步0x21[0x700e7c0] 码226 len=47 |
| 30.4s | S2C | 属性更新0x10[O#0x498afc] FP=14 |
| 30.4s | S2C | 属性更新0x10[O#0x498afc] SP=6 |
| 30.4s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 30.4s | S2C | S2C m24 通用事件[0x499e120700e7c0] 伤害飘字 [790, 139, 0, "CS_hs_kfkj02", "SkillLock", "monm031301"] |
| 30.4s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x7034e9b] |
| 30.4s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 30.4s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x7034e9b] ×2 |
| 30.4s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 49, 160, "ui_jh_001"] |
| 30.4s | S2C | S2C m24 通用事件[0x499e120700e7c0] 击退 [obj(0x7034e9b,0x498afc), "monm031301", "徐傠囃辭睼 "] |
| 30.4s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [3] |
| 30.4s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x700e7c0,0x499e12)] |
| 30.4s | S2C | 多属性0x2D[0x9e120700](布局变体) len=59 2d0100c0e70007129e49000600d70100 |
| 30.7s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 31.7s | S2C | op31(sub1) 0x7008316 |
| 31.8s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 31.8s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 32.4s | S2C | 属性更新0x10[O#0x499e12] LogicState=0 |
| 32.4s | C2S | m990 ? [obj(0x700e7c0,0x499e12), 947.158, 43.896, 1029.35] |
| 32.5s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 32.5s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 32.5s | C2S | m914 招式结束异步位置 [945.392, 43.886, 1031.333, 2.414, 12, 6] |
| 32.5s | S2C | 位置0x1F(短len=25) |
| 32.5s | C2S | m54 格挡状态(值3876302951) |
| 32.6s | C2S | m82 ? [3] |
| 33.1s | C2S | m54 格挡状态(值4100342926) |
| 33.1s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 33.1s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 33.1s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 33.1s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 33.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 33.1s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 33.1s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 33.2s | C2S | m54 格挡状态(值3876302951) |
| 33.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 33.3s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 33.3s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 33.3s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 33.3s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 33.3s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 33.3s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 33.3s | C2S | m54 格挡状态(值3876302951) |
| 33.3s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 33.3s | C2S | m15 移动上报 自动 state=5 pos=(945.4,43.9,1031.3) ×2 |
| 33.3s | C2S | m54 格挡状态(值3876302951) |
| 33.9s | C2S | m15 移动上报 ?2000 state=7 pos=(945.4,43.9,1031.3) |
| 34.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 34.0s | C2S | m15 移动上报 手动 state=7 pos=(945.9,43.9,1031.6) |
| 34.1s | S2C | 对象同步0x21[0x7034e9b] 码88 len=47 |
| 34.2s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 34.2s | S2C | 属性更新0x10[O#0x498afc] MP=10347 |
| 34.3s | S2C | 对象同步0x21[0x7034e9b] 码91 len=47 |
| 34.4s | C2S | m54 格挡状态(值4100342926) |
| 34.4s | S2C | 属性更新0x10[O#0x498afc] LogicState=2 |
| 34.4s | C2S | m15 移动上报 手动 state=7 pos=(946.4,43.9,1033.1) |
| 34.4s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 34.4s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 34.4s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 34.4s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 34.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 34.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 34.4s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 34.5s | C2S | m54 格挡状态(值3876302951) |
| 34.5s | C2S | m15 移动上报 ?4000 state=7 pos=(946.5,43.9,1033.5) |
| 34.5s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 34.5s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 34.5s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 34.5s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 34.5s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 34.5s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 34.5s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 34.5s | C2S | m54 格挡状态(值3876302951) ×3 |
| 34.6s | C2S | m15 移动上报 ?4000 state=7 pos=(946.5,43.9,1033.6) ×2 |
| 34.7s | S2C | 对象同步0x21[0x7034e9b] 码203 len=135 |
| 35.0s | S2C | op0E len=9 0ec0e70007129e4900 |
| 35.0s | C2S | m440 ? [] |
| 35.0s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 35.0s | C2S | m431 选择目标 [0] |
| 35.1s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 35.6s | C2S | m3 ? [0] |
| 35.6s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.6s | C2S | m3 ? [0] |
| 35.6s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.6s | C2S | m3 ? [0] |
| 35.6s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.6s | C2S | m3 ? [0] |
| 35.6s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.6s | C2S | m3 ? [0] |
| 35.6s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.6s | C2S | m3 ? [0] |
| 35.6s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.7s | S2C | 属性更新0x10[O#0x498afc] SP=0 |
| 35.7s | C2S | m3 ? [0] |
| 35.7s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.7s | C2S | m3 ? [0] |
| 35.7s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.7s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=155 |
| 35.7s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=7.5 |
| 35.7s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=3.6 |
| 35.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=D9,7034E9B,498AFC,1,0,0 |
| 35.7s | S2C | S2C m169 buff添加 ["buf_qg_fast", obj(0x7034e9b,0x498afc)] |
| 35.7s | S2C | op31(sub1) 0x700d965 |
| 35.7s | C2S | m3 ? [0] |
| 35.7s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.7s | C2S | m3 ? [0] |
| 35.7s | C2S | m216 使用轻功 ["qinggong_8", 1] |
| 35.7s | C2S | m15 移动上报 手动 state=7 pos=(943.9,43.9,1038.0) |
| 35.8s | C2S | m216 使用轻功 ["qinggong_8", 0] |
| 35.8s | S2C | 对象同步0x21[0x7034e9b] 码58 len=47 |
| 35.9s | S2C | 属性更新0x10[O#0x498afc] FP=0 |
| 35.9s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=165 |
| 35.9s | S2C | S2C m169 buff移除 ["buf_qg_fast", obj(0x7034e9b,0x498afc)] |
| 35.9s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 35.9s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 35.9s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 35.9s | C2S | m15 移动上报 手动 state=7 pos=(943.0,43.9,1039.4) |
| 36.0s | S2C | 对象同步0x21[0x7034e9b] 码2 len=47 |
| 36.0s | C2S | m15 移动上报 自动 state=4 pos=(942.7,43.9,1039.8) |
| 36.2s | S2C | 对象同步0x21[0x7034e9b] 码232 len=47 |
| 36.4s | C2S | m54 格挡状态(值4100342926) |
| 36.5s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 36.5s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 36.5s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 36.5s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 36.5s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 36.5s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 36.5s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 36.5s | C2S | m15 移动上报 手动 state=7 pos=(942.7,43.9,1039.8) |
| 36.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 36.6s | C2S | m54 格挡状态(值3876302951) |
| 36.6s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 36.6s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 36.6s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 36.6s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 36.6s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 36.6s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 36.6s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 36.6s | C2S | m54 格挡状态(值3876302951) ×2 |
| 36.7–38.6s | S2C | 对象同步0x21[0x7034e9b] 码247 len=47 ×2 |
| 36.7s | C2S | m15 移动上报 手动 state=7 pos=(942.9,43.9,1039.6) |
| 36.9s | S2C | 对象同步0x21[0x7034e9b] 码249 len=47 |
| 37.1s | S2C | op06 len=1 06 |
| 37.1s | C2S | m54 格挡状态(值4100342926) |
| 37.2s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 37.2s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 37.2s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 37.2s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 37.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 37.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 37.2s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 37.2s | C2S | m15 移动上报 手动 state=7 pos=(944.5,43.9,1038.2) ×2 |
| 37.2s | C2S | m54 格挡状态(值3876302951) |
| 37.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 37.3s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 37.3s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 37.3s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 37.3s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 37.3s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 37.3s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 37.3s | C2S | m54 格挡状态(值3876302951) |
| 37.3s | C2S | m15 移动上报 自动 state=4 pos=(944.8,43.9,1038.0) |
| 37.3s | C2S | m54 格挡状态(值3876302951) ×2 |
| 37.4s | S2C | 对象同步0x21[0x7034e9b] 码196 len=91 |
| 37.4s | C2S | m15 移动上报 手动 state=7 pos=(944.8,43.9,1038.0) |
| 37.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 37.6s | S2C | 对象同步0x21[0x7034e9b] 码235 len=47 |
| 37.6–37.8s | C2S | m15 移动上报 手动 state=7 pos=(943.9,43.9,1038.5) ×2 |
| 37.8s | S2C | 对象同步0x21[0x7034e9b] 码118 len=47 |
| 37.8s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=152 |
| 38.0s | S2C | 对象同步0x21[0x7034e9b] 码69 len=47 |
| 38.3s | C2S | 对象请求 len=33 |
| 38.3s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 38.3s | C2S | m431 选择目标 [obj(0x70295df,0x3bbc1c)] |
| 38.3s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [937.917, 43.896, 1041.854, -1.061] |
| 38.4s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 38.4s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 38.4s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_hs_kfkj02 |
| 38.4s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 38.4s | S2C | S2C m284 空值标记帧 [] |
| 38.4s | S2C | 属性更新0x10[O#0x3bbc1c] LogicState=1 |
| 38.4s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +0行 |
| 38.4s | S2C | 属性更新0x10[O#0x498afc] LogicState=1 |
| 38.4s | S2C | 属性更新0x10[O#0x3bbc21] LogicState=1 |
| 38.4s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +1行 |
| 38.4s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 38.4s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 38.4s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 38.4s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 38.4s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 38.4s | S2C | 属性更新0x10[O#0x498afc] MPRatio=98 |
| 38.4s | S2C | 属性更新0x10[O#0x498afc] MP=10206 |
| 38.4s | S2C | 位置0x1F(短len=25) |
| 38.4s | S2C | S2C m337 施法回显[0x498afc07034e9b] [937.785, 43.886, 1041.927, -0.946, 0] |
| 38.4s | S2C | 属性更新0x10[O#0x498afc] LastWGAttackNpc=monm031302 |
| 38.4s | S2C | S2C m24 通用事件[0x3bbc1c070295df] 动作开始 [1, obj(0x7034e9b,0x498afc)] |
| 38.4s | S2C | S2C m19 锁定伤害段[0x498afc07034e9b] [obj(0x70295df,0x3bbc1c), "CS_hs_kfkj02", 3000, 3000, 939.01, 43.886] |
| 38.4s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 38.4s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 38.4s | C2S | m15 移动上报 手动 state=4 pos=(937.8,43.9,1041.9) |
| 38.4s | C2S | m15 移动上报 手动 state=7 pos=(937.8,43.9,1041.9) ×3 |
| 38.4s | S2C | 对象同步0x21[0x70295df] 码245 len=223 |
| 38.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 38.5s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 38.5s | S2C | 属性更新0x10[O#0x498afc] FP=2 |
| 38.5s | S2C | 属性更新0x10[O#0x498afc] SP=2 |
| 38.5s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 38.5s | S2C | S2C m24 通用事件[0x3bbc1c070295df] 伤害飘字 [508, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031302"] |
| 38.5s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x70295df,0x3bbc1c)] |
| 38.6s | S2C | 多属性0x2D(非标子码02) len=43 2d0200df9502071cbc3b000100a0019b |
| 38.7s | S2C | op31(sub1) 0x700fed1 |
| 38.8s | S2C | 多属性0x2D[0xbc1c0702](布局变体) len=25 2d0100df9502071cbc3b000200d70119 |
| 38.9s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=162 |
| 38.9s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 38.9s | S2C | 属性更新0x10[O#0x498afc] MP=10306 |
| 39.1s | S2C | 属性更新0x10[O#0x498afc] FP=4 |
| 39.1s | S2C | 属性更新0x10[O#0x498afc] SP=4 |
| 39.1s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 39.1s | S2C | S2C m24 通用事件[0x3bbc1c070295df] 伤害飘字 [170, 338, 0, "CS_hs_kfkj02", "SkillLock", "monm031302"] |
| 39.1s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x7034e9b] |
| 39.1s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 39.1s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x7034e9b] ×2 |
| 39.1s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 54, 160, "ui_jh_001"] |
| 39.1s | S2C | S2C m1 系统提示 [2, "300144"] |
| 39.1s | S2C | S2C m24 通用事件[0x3bbc1c070295df] 击退 [obj(0x7034e9b,0x498afc), "monm031302", "徐傠囃辭睼 "] |
| 39.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [2] |
| 39.1s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x70295df,0x3bbc1c)] |
| 39.2s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 39.2s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 39.2s | C2S | m15 移动上报 手动 state=5 pos=(934.9,43.9,1044.2) ×2 |
| 39.3s | S2C | 多属性0x2D[0xbc1c0702](布局变体) len=59 2d0100df9502071cbc3b000600d70100 |
| 39.7s | S2C | op28(子码47) len=236 ×3 |
| 40.6s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 40.6s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 40.7s | S2C | op28(子码47) len=225 |
| 41.0s | S2C | 属性更新0x10[O#0x3bbc1c] LogicState=0 |
| 41.2s | S2C | 多属性0x2D[0x9e730700](布局变体) len=20 2d010081560007739e49000100770201 |
| 41.3s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 41.3s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 41.3s | C2S | m990 ? [obj(0x70295df,0x3bbc1c), 936.657, 43.896, 1042.817] |
| 41.4s | S2C | 位置0x1F(短len=25) |
| 41.4s | C2S | m914 招式结束异步位置 [938.726, 43.886, 1041.263, -0.946, 12, 7] |
| 41.5s | S2C | 对象同步0x21[0x7034e9b] 码118 len=91 |
| 41.7s | S2C | op28(子码47) len=222 |
| 41.9s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=165 |
| 42.0s | C2S | m15 移动上报 手动 state=7 pos=(938.7,43.9,1041.3) |
| 42.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 42.2s | S2C | 对象同步0x21[0x7034e9b] 码175 len=47 |
| 42.2–42.7s | C2S | m15 移动上报 手动 state=7 pos=(939.6,43.9,1040.8) ×2 |
| 42.4s | S2C | 对象同步0x21[0x7034e9b] 码30 len=47 |
| 42.7s | S2C | op31(sub2) 0x70095b5 →0x701677b |
| 42.8s | S2C | 对象同步0x21[0x702ade2] 码189 len=91 |
| 42.8s | C2S | m15 移动上报 手动 state=7 pos=(941.4,43.9,1039.4) |
| 43.0–45.4s | S2C | 对象同步0x21[0x7034e9b] 码151 len=47 ×2 |
| 43.6s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 43.6s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 43.6s | C2S | m15 移动上报 自动 state=4 pos=(939.6,43.9,1036.2) |
| 43.7s | S2C | 对象同步0x21[0x7034e9b] 码131 len=47 |
| 44.6s | S2C | 属性更新0x10[O#0x3bbc21] State= |
| 44.8s | S2C | 对象同步0x21[0x702ade2] 码197 len=91 |
| 44.8s | S2C | op0D len=251 0d65d900073d043b002b776944e8fb4242a15277 |
| 45.0s | C2S | m15 移动上报 自动 state=7 pos=(939.6,43.9,1036.2) |
| 45.0s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 45.0s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 45.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 45.0s | C2S | m15 移动上报 自动 state=7 pos=(939.6,43.9,1036.2) |
| 45.2s | S2C | 对象同步0x21[0x7034e9b] 码139 len=91 |
| 45.2s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 45.3s | C2S | m15 移动上报 自动 state=5 pos=(939.5,43.9,1037.4) ×2 |
| 45.3–45.6s | C2S | m15 移动上报 ?1000 state=7 pos=(939.4,43.9,1037.6) ×3 |
| 45.7s | S2C | 对象同步0x21[0x7034e9b] 码121 len=47 |
| 45.7s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID=default_normal_skill |
| 45.7s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 45.7s | S2C | 位置0x1F(短len=25) |
| 45.7s | S2C | S2C m337 施法回显[0x3bbc210702ade2] [939.854, 43.886, 1037.043, 3.619, 0] |
| 45.7s | S2C | S2C m20 范围施放标记[0x3bbc210702ade2] ["atkhit3", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 45.7s | C2S | m15 移动上报 自动 state=4 pos=(939.4,43.9,1036.4) |
| 45.8s | S2C | op2C len=153 2c47980000000a010000000000800600e2ad0207 |
| 45.9s | C2S | 对象请求 len=33 |
| 45.9s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 45.9s | C2S | m431 选择目标 [obj(0x7005681,0x499e73)] |
| 45.9s | C2S | 对象请求 len=33 |
| 45.9s | C2S | m431 选择目标 [obj(0x7005681,0x499e73)] |
| 46.0s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 46.0s | S2C | S2C m4 对话标记? [] |
| 46.0s | S2C | S2C m5 NPC菜单 ["ui_npcclonetl_menu_04"] |
| 46.0s | S2C | S2C m6 NPC对话文本 [20000, "ui_npcclonetl_talk_04"] ×3 |
| 46.0s | S2C | S2C m7 NPC对话对象 [obj(0x7005681,0x499e73)] |
| 46.0s | C2S | m15 移动上报 自动 state=4 pos=(939.4,43.9,1036.4) |
| 46.3s | S2C | 对象同步0x21[0x7034e9b] 码54 len=47 |
| 46.6s | S2C | 属性更新0x10[O#0x498afc] LastAttackerNpc=monm031302 |
| 46.6s | S2C | 属性更新0x10[O#0x498afc] LastDodge=32.07 |
| 46.6s | S2C | 属性更新0x10[O#0x498afc] SP=5 |
| 46.6s | S2C | 属性更新0x10[O#0x498afc] HPRatio=99 |
| 46.6s | S2C | 属性更新0x10[O#0x498afc] HP=28260 |
| 46.6s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 46.6s | S2C | S2C m24 通用事件[0x498afc07034e9b] 伤害飘字 [25, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 46.6s | S2C | 属性更新0x10[O#0x498afc] HitHPRatio=99 |
| 46.6s | S2C | 属性更新0x10[O#0x498afc] HitHP=28273 |
| 46.6s | S2C | S2C m205 伤害目标列表[0x3bbc210702ade2] ["default_normal_skill", obj(0x7034e9b,0x498afc)] |
| 46.6s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID= |
| 46.6s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 46.6s | S2C | S2C m189 对象通知 [obj(0x702ade2,0x3bbc21)] |
| 46.9s | S2C | 对象同步0x21[0x702ade2] 码167 len=91 |
| 47.1s | S2C | 对象同步0x21[0x702ade2] 码147 len=135 |
| 47.9s | C2S | 对象请求 len=33 |
| 48.0s | S2C | S2C m8 对话结束 [] |
| 48.1s | C2S | m197 ? [obj(0x7005681,0x499e73)] |
| 48.6s | C2S | m54 格挡状态(值4100342926) |
| 48.6s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 48.6s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 48.6s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 48.6s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 48.6s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 48.6s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 48.6s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 48.9s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID=default_normal_skill |
| 48.9s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 48.9s | S2C | S2C m20 范围施放标记[0x3bbc210702ade2] ["atkhit1", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 49.1s | C2S | m15 移动上报 手动 state=4 pos=(939.4,43.9,1036.4) |
| 49.3s | S2C | 对象同步0x21[0x7034e9b] 码54 len=47 |
| 49.9s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=8 [1, obj(0x702ade2,0x3bbc21), "default_normal_skill", "徐傠囃辭睼 ", "monm031302"] |
| 49.9s | S2C | 属性更新0x10[O#0x498afc] LastParryTime=1794143798 |
| 49.9s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 49.9s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=59.0 |
| 49.9s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5=2B1E,7034E9B,498AFC,2,C38EC044593,0 |
| 49.9s | S2C | S2C m169 buff添加 ["eqp_wrist_01a", obj(0x7034e9b,0x498afc)] |
| 49.9s | S2C | 属性更新0x10[O#0x498afc] SP=9 |
| 49.9s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=99 |
| 49.9s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19827 |
| 49.9s | S2C | 属性更新0x10[O#0x498afc] FP=6 |
| 49.9s | S2C | 属性更新0x10[O#0x498afc] HP=28259 |
| 49.9s | S2C | 属性更新0x10[O#0x498afc] SP=19 |
| 49.9s | S2C | S2C m24 通用事件[0x498afc07034e9b] 伤害飘字 [1, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 49.9s | S2C | S2C m205 伤害目标列表[0x3bbc210702ade2] ["default_normal_skill", obj(0x7034e9b,0x498afc)] |
| 49.9s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID= |
| 49.9s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 49.9s | S2C | S2C m189 对象通知 [obj(0x702ade2,0x3bbc21)] |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] Sat=22 |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=35.0 |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=69.0 |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=21.0 |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=40.0 |
| 50.2s | S2C | S2C m24 通用事件[0x498afc07034e9b] 动作开始 ["buff_ng_xyhxj03", obj(0x7034e9b,0x498afc)] |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo6=4BFB,7034E9B,498AFC,5,C38EC0469B1,1 |
| 50.2s | S2C | S2C m169 buff添加 ["buf_ng_jz_006_02", obj(0x7034e9b,0x498afc)] |
| 50.2s | S2C | S2C m169 buff刷新? ["buf_ng_jz_006_02", obj(0x7034e9b,0x498afc)] |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo6=4BFB,7034E9B,498AFC,5,C38EC0469B1,5 |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=75.0 |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=109.0 |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=61.0 |
| 50.2s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=80.0 |
| 50.3s | C2S | m54 格挡状态(值3876302951) |
| 50.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 50.3s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 50.3s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 50.3s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 50.3s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 50.3s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 50.3s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 50.3s | C2S | m54 格挡状态(值3876302951) ×2 |
| 50.3s | S2C | 属性更新0x10[V#0x2] LeftTime=575072 |
| 50.3s | C2S | m54 格挡状态(值3876302951) |
| 50.3s | C2S | m15 移动上报 手动 state=7 pos=(939.4,43.9,1036.4) |
| 50.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 50.4s | C2S | m15 移动上报 自动 state=4 pos=(939.7,43.9,1036.8) |
| 50.6s | S2C | 对象同步0x21[0x7034e9b] 码167 len=91 |
| 50.8s | C2S | m15 移动上报 手动 state=7 pos=(939.7,43.9,1036.8) |
| 50.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 51.0s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19858 |
| 51.0s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=100 |
| 51.0s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19858 |
| 51.0s | S2C | 对象同步0x21[0x7034e9b] 码169 len=47 |
| 51.1s | C2S | m15 移动上报 手动 state=7 pos=(939.0,43.9,1035.9) |
| 51.1s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID=default_normal_skill |
| 51.1s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 51.1s | S2C | 位置0x1F(短len=25) |
| 51.1s | S2C | S2C m337 施法回显[0x3bbc210702ade2] [940.212, 43.886, 1037.545, 3.773, 0] |
| 51.1s | S2C | S2C m20 范围施放标记[0x3bbc210702ade2] ["atkhit2", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 51.1s | C2S | m15 移动上报 自动 state=4 pos=(938.9,43.9,1035.8) |
| 51.2s | C2S | m15 移动上报 手动 state=7 pos=(938.9,43.9,1035.8) |
| 51.2s | S2C | 对象同步0x21[0x7034e9b] 码37 len=91 |
| 51.3s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 51.3s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 51.3s | C2S | m15 移动上报 手动 state=5 pos=(939.1,43.9,1036.0) ×2 |
| 51.4s | C2S | m54 格挡状态(值4100342926) |
| 51.4s | C2S | m15 移动上报 自动 state=4 pos=(939.4,43.9,1036.3) |
| 51.4s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 51.4s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 51.4s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 51.4s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 51.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 51.4s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 51.4s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 51.5s | S2C | 对象同步0x21[0x7034e9b] 码23 len=91 |
| 51.6s | C2S | m15 移动上报 手动 state=4 pos=(939.4,43.9,1036.3) |
| 51.7s | S2C | 属性更新0x10[O#0x498afc] HP=28273 |
| 51.8s | S2C | 对象同步0x21[0x7034e9b] 码121 len=47 |
| 51.9s | S2C | op28(子码47) len=235 |
| 52.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=8 [1, obj(0x702ade2,0x3bbc21), "default_normal_skill", "徐傠囃辭睼 ", "monm031302"] |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] LastParryTime=1794143800 |
| 52.1s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 52.1s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo7=386B,7034E9B,498AFC,2,C38EC044DDB,0 |
| 52.1s | S2C | S2C m169 buff添加 ["eqp_nclo_03a", obj(0x7034e9b,0x498afc)] |
| 52.1s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 52.1s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] SP=23 |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=99 |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19827 |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] FP=8 |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] HP=28272 |
| 52.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] 伤害飘字 [1, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo6=4BFB,7034E9B,498AFC,5,C38EC046D1C,4 |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=65.0 |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=99.0 |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=51.0 |
| 52.1s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=70.0 |
| 52.1s | S2C | S2C m205 伤害目标列表[0x3bbc210702ade2] ["default_normal_skill", obj(0x7034e9b,0x498afc)] |
| 52.1s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID= |
| 52.1s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 52.1s | S2C | S2C m189 对象通知 [obj(0x702ade2,0x3bbc21)] |
| 52.2s | S2C | 属性更新0x10[O#0x150] State= |
| 52.3s | S2C | op06 len=1 06 |
| 52.4s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 52.4s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 52.6s | C2S | m82 ? [3] |
| 53.1s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19838 |
| 53.2s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID=default_normal_skill |
| 53.2s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 53.2s | S2C | 位置0x1F(短len=25) |
| 53.2s | S2C | S2C m337 施法回显[0x3bbc210702ade2] [940.212, 43.886, 1037.545, 3.764, 0] |
| 53.2s | S2C | S2C m20 范围施放标记[0x3bbc210702ade2] ["atkhit2", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 53.7s | C2S | m211 使用技能(出招) CS_hs_kfkj01 10值 [939.351, 43.887, 1036.344, -0.009] |
| 53.7s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 53.7s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_hs_kfkj01 |
| 53.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 53.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 53.7s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 53.7s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 53.7s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 53.7s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 53.7s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 53.7s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 53.7s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 53.7s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 53.7s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 53.7s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 53.7s | S2C | 属性更新0x10[O#0x498afc] MP=10347 |
| 53.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=489C,7034E9B,498AFC,1,C38EC04B209,1 |
| 53.7s | S2C | S2C m169 buff添加 ["buf_CS_hs_kfkj01", obj(0x7034e9b,0x498afc)] |
| 53.7s | S2C | S2C m20 范围施放标记[0x498afc07034e9b] ["CS_hs_kfkj01", 0, 0, 0.0] |
| 54.1s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19858 |
| 54.1s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=100 |
| 54.1s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19858 |
| 54.2s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 54.3s | S2C | 属性更新0x10[O#0x498afc] SP=24 |
| 54.3s | S2C | 属性更新0x10[O#0x498afc] HP=28251 |
| 54.3s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 54.3s | S2C | S2C m24 通用事件[0x498afc07034e9b] 伤害飘字 [21, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 54.3s | S2C | 属性更新0x10[O#0x498afc] HitHP=28263 |
| 54.3s | S2C | S2C m205 伤害目标列表[0x3bbc210702ade2] ["default_normal_skill", obj(0x7034e9b,0x498afc)] |
| 54.3s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID= |
| 54.3s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 54.3s | S2C | S2C m189 对象通知 [obj(0x702ade2,0x3bbc21)] |
| 54.7s | C2S | m54 格挡状态(值4100342926) |
| 54.7s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 54.7s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 54.7s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 54.7s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 54.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo8=11CE,7034E9B,498AFC,1,0,0 |
| 54.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 55.3s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID=default_break_parry_skill |
| 55.3s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 55.3s | S2C | S2C m20 范围施放标记[0x3bbc210702ade2] ["b_atk2_2", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 55.5s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=8 [0, obj(0x702ade2,0x3bbc21), "default_break_parry_skill", "徐傠囃辭睼 ", "monm031302"] |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] LastParryTime=1794143804 |
| 55.5s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] SP=28 |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=99 |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19827 |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] FP=10 |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] HP=28250 |
| 55.5s | S2C | S2C m24 通用事件[0x498afc07034e9b] 伤害飘字 [1, 0, 0, "default_break_parry_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] BufferInfo9=11D0,702ADE2,3BBC21,1,C38EC044F69,0 |
| 55.5s | S2C | S2C m169 buff添加 ["buf_CantParry", obj(0x7034e9b,0x498afc)] |
| 55.5s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] BufferInfo8= |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 55.5s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 55.5s | S2C | S2C m205 伤害目标列表[0x3bbc210702ade2] ["default_break_parry_skill", obj(0x7034e9b,0x498afc)] |
| 55.5s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillID= |
| 55.5s | S2C | 属性更新0x10[O#0x3bbc21] CurSkillTarget=None |
| 55.5s | S2C | S2C m189 对象通知 [obj(0x702ade2,0x3bbc21)] |
| 55.7s | C2S | 对象请求 len=33 |
| 55.7s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 55.7s | C2S | m431 选择目标 [obj(0x702ade2,0x3bbc21)] |
| 55.7s | C2S | m211 使用技能(出招) CS_hs_kfkj02 14值 [939.351, 43.887, 1036.344, 0.682] |
| 55.7s | C2S | m15 移动上报 手动 state=4 pos=(939.4,43.9,1036.3) |
| 55.8s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 55.8s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] |
| 55.8s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_hs_kfkj02 |
| 55.8s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 55.8s | S2C | S2C m284 空值标记帧 [] |
| 55.8s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 55.8s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 55.8s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 55.8s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =16802048 0x7034e9b] ×2 |
| 55.8s | S2C | 属性更新0x10[O#0x498afc] MPRatio=98 |
| 55.8s | S2C | 属性更新0x10[O#0x498afc] MP=10206 |
| 55.8s | S2C | 位置0x1F(短len=25) |
| 55.8s | S2C | S2C m337 施法回显[0x498afc07034e9b] [939.351, 43.886, 1036.344, 0.622, 0] |
| 55.8s | S2C | S2C m24 通用事件[0x3bbc210702ade2] 动作开始 [1, obj(0x7034e9b,0x498afc)] |
| 55.8s | S2C | S2C m19 锁定伤害段[0x498afc07034e9b] [obj(0x702ade2,0x3bbc21), "CS_hs_kfkj02", 3000, 3000, 937.858, 43.886] |
| 55.8s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 55.8s | S2C | 属性更新0x10[O#0x498afc] CantParry=2 |
| 55.8s | C2S | m15 移动上报 自动 state=7 pos=(939.4,43.9,1036.3) |
| 55.8s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 55.8s | C2S | m15 移动上报 自动 state=7 pos=(939.3,43.9,1036.3) |
| 55.8s | C2S | m915 命中确认上报 [1, obj(0x702ade2,0x3bbc21), 937.535, 43.896, 1033.811, 1.436] |
| 56.0s | S2C | S2C m169 buff移除 ["eqp_wrist_01a", obj(0x7034e9b,0x498afc)] |
| 56.0s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5= |
| 56.0s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=49.0 |
| 56.0s | S2C | 属性更新0x10[V#0x28] CurTargetCount=1 |
| 56.0s | S2C | 属性更新0x10[O#0x498afc] FP=12 |
| 56.0s | S2C | 属性更新0x10[O#0x498afc] SP=30 |
| 56.0s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 56.0s | S2C | S2C m24 通用事件[0x3bbc210702ade2] 伤害飘字 [511, 0, 0, "CS_hs_kfkj02", "SkillLock", "monm031302"] |
| 56.0s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x702ade2,0x3bbc21)] |
| 56.1s | S2C | 对象同步0x21[0x7034e9b] 码38 len=135 |
| 56.1s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=100 |
| 56.1s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19858 |
| 56.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=25 [31, 746, 0, "eqp_nclo_03a", "Buffer", "徐傠囃辭睼 "] |
| 56.2s | S2C | 多属性0x2D[0xbc210702](布局变体) len=25 2d0100e2ad020721bc3b000200d70118 |
| 56.5s | S2C | 对象同步0x21[0x702ade2] 码252 len=47 |
| 56.5s | S2C | 属性更新0x10[O#0x498afc] FP=14 |
| 56.5s | S2C | 属性更新0x10[O#0x498afc] SP=32 |
| 56.5s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 56.5s | S2C | S2C m24 通用事件[0x3bbc210702ade2] 伤害飘字 [167, 344, 0, "CS_hs_kfkj02", "SkillLock", "monm031302"] |
| 56.5s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x7034e9b] |
| 56.5s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 56.5s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x7034e9b] ×2 |
| 56.5s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 59, 160, "ui_jh_001"] |
| 56.5s | S2C | S2C m24 通用事件[0x3bbc210702ade2] 击退 [obj(0x7034e9b,0x498afc), "monm031302", "徐傠囃辭睼 "] |
| 56.5s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [2] |
| 56.5s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_hs_kfkj02", obj(0x702ade2,0x3bbc21)] |
| 56.7s | S2C | 属性更新0x10[O#0x498afc] HP=28263 |
| 56.7s | S2C | 多属性0x2D[0xbc210702](布局变体) len=59 2d0100e2ad020721bc3b000600d70100 |
| 56.8s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 57.0s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 57.3s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 57.3s | C2S | m15 移动上报 自动 state=5 pos=(940.5,43.9,1038.0) ×2 |
| 57.9s | S2C | op28(子码47) len=235 |
| 58.1s | S2C | S2C m169 buff移除 ["eqp_nclo_03a", obj(0x7034e9b,0x498afc)] |
| 58.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo7= |
| 58.2s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 58.2s | S2C | 属性更新0x10[O#0x498afc] CurSkillTarget=None |
| 58.3s | S2C | 多属性0x2D[0x9ea00700](布局变体) len=221 2d0100b7470007a09e49001000b50004 |
| 58.4s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 58.4s | S2C | 属性更新0x10[O#0x498afc] MP=10306 |
| 58.5s | S2C | S2C m169 buff移除 ["buf_CantParry", obj(0x7034e9b,0x498afc)] |
| 58.5s | S2C | 属性更新0x10[O#0x498afc] BufferInfo9= |
| 58.5s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 58.5s | S2C | 属性更新0x10[O#0x3bbc21] LogicState=0 |
| 58.8s | C2S | m990 ? [obj(0x702ade2,0x3bbc21), 939.621, 43.896, 1036.72] |
| 58.8s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 58.8s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 58.8s | C2S | m914 招式结束异步位置 [938.076, 43.886, 1034.565, 0.622, 12, 8] |
| 58.8s | S2C | 位置0x1F(短len=25) |
| 58.8s | C2S | m54 格挡状态(值3876302951) |
| 58.9s | S2C | 对象同步0x21[0x7034e9b] 码218 len=91 |
| 59.0s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 60.5s | S2C | 属性更新0x10[O#0x498afc] LogicState=2 |
| 61.1s | C2S | 对象请求 len=33 |
| 61.2s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 61.2s | C2S | m431 选择目标 [obj(0x7005681,0x499e73)] |
| 61.2s | C2S | 对象请求 len=33 |
| 61.2s | C2S | m431 选择目标 [obj(0x7005681,0x499e73)] |
| 61.2s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 61.2s | S2C | S2C m4 对话标记? [] |
| 61.2s | S2C | S2C m5 NPC菜单 ["ui_npcclonetl_menu_04"] |
| 61.2s | S2C | S2C m6 NPC对话文本 [20000, "ui_npcclonetl_talk_04"] ×3 |
| 61.2s | S2C | S2C m7 NPC对话对象 [obj(0x7005681,0x499e73)] |
| 61.7s | S2C | 属性更新0x10[O#0x498afc] HitHPRatio=100 |
| 61.7s | S2C | 属性更新0x10[O#0x498afc] HitHP=28285 |
| 61.7s | S2C | 属性更新0x10[O#0x498afc] HPRatio=100 |
| 61.7s | S2C | 属性更新0x10[O#0x498afc] HP=28285 |
| 62.5s | S2C | 属性更新0x10[O#0x498afc] SP=2 |
| 62.9s | S2C | 属性更新0x10[O#0x498afc] FP=0 |
| 63.1s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 63.1s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 63.3s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 63.3s | C2S | m15 移动上报 自动 state=5 pos=(938.1,43.9,1034.6) ×2 |
| 65.2s | C2S | 对象请求 len=33 |
| 65.2s | S2C | S2C m8 对话结束 [] |
| 65.2s | C2S | m197 ? [obj(0x7005681,0x499e73)] |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] BufferInfo6=4BFB,7034E9B,498AFC,5,C38EC0469F9,1 |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=35.0 |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=19.0 |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=21.0 |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=40.0 |
| 65.3s | S2C | S2C m169 buff移除 ["buf_ng_jz_006_02", obj(0x7034e9b,0x498afc)] |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] BufferInfo6= |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=25.0 |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=9.0 |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=11.0 |
| 65.3s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=30.0 |
| 65.6s | S2C | 属性更新0x10[O#0x498afc] SP=0 |
| 65.9s | S2C | op28(子码47) len=235 |
| 66.1s | C2S | m440 ? [] |
| 66.1s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 66.1s | C2S | m431 选择目标 [0] |
| 66.2s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 67.6s | S2C | op06 len=1 06 |
| 1794143756.5s | C2S | 6001 大厅心跳 04 00 00 00 10 EE 11 11 → 01 FF 00 00 ×7 |

## i.pcapng — 103.143.19.173:7072（g 的续段，中途回抓）

场景：hyjf07 十二次连发 + 冲刺/伤害/怒气全链。事件 2328 条 → 合并为 1222 段。抓包起点 2026-11-08 21:18:55（本地墙钟）

| 时间 | 方向 | 交互 |
|---|---|---|
| 0.0–214.7s | PAIR | C2S m924 MSG_DELAY(client ms) → S2C m294 回显[client ms, obj, 服务器unix秒, …] ×214 |
| 0.0s | S2C | op28(子码47) len=255 |
| 0.1–214.9s | PAIR | C2S ping(0x18) → S2C 0x38 回显 ×191 |
| 2.8s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 2.9s | S2C | op06 len=1 06 |
| 2.9s | C2S | m15 移动上报 自动 state=5 pos=(946.1,43.9,1040.4) ×2 |
| 4.7s | C2S | m15 移动上报 自动 state=8 pos=(946.1,43.9,1040.4) |
| 4.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 4.8s | S2C | 对象同步0x21[0x7034e9b] 码245 len=135 |
| 5.0s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [946.109, 45.193, 1040.392, 0.22] |
| 5.1s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 5.1s | S2C | 位置0x1F(短len=25) |
| 5.1s | S2C | S2C m337 施法回显[0x498afc07034e9b] [946.109, 45.336, 1040.392, 0.645] |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_jh_hyjf07 |
| 5.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] 冲锋动画 [954.764, 44.886, 1051.908, 2, "qg_fastruning_s_2h_01"] |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 5.1s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 5.1s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 5.1s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 5.1s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 5.1s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 5.1s | S2C | 属性更新0x10[O#0x498afc] MP=10298 |
| 5.1s | S2C | S2C m707 受理族变体 [15] |
| 5.1s | C2S | m15 移动上报 自动 state=6 pos=(946.1,45.3,1040.4) |
| 5.3s | S2C | 对象同步0x21[0x7034e9b] 码222 len=47 |
| 5.5s | S2C | 对象同步0x21[0x70295df] 码52 len=91 |
| 5.5s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 5.9s | C2S | m212 冲锋/续招落地 [954.65, 44.893, 1051.757, 0.645] |
| 5.9s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 5.9s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 5.9s | S2C | 位置0x1F(短len=25) |
| 5.9s | S2C | S2C m26 技能释放 [] |
| 5.9s | S2C | 位置0x1F(短len=25) |
| 5.9s | S2C | S2C m337 施法回显[0x498afc07034e9b] [954.65, 44.893, 1051.757, 0.645] |
| 5.9s | S2C | 位置0x1F(短len=25) |
| 5.9s | S2C | S2C m337 施法回显[0x498afc07034e9b] [954.65, 44.893, 1051.757, 0.22, 0] |
| 5.9s | S2C | S2C m20 范围施放标记[0x498afc07034e9b] ["CS_jh_hyjf07", 0, 0, 0.0, 955.665, 43.886] |
| 5.9s | C2S | m15 移动上报 手动 state=8 pos=(954.8,44.9,1051.9) |
| 6.0s | S2C | 对象同步0x21[0x701677b] 码217 len=91 |
| 6.0s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 6.0s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 6.0s | C2S | m15 移动上报 自动 state=4 pos=(954.6,44.9,1051.8) |
| 6.0s | C2S | m15 移动上报 自动 state=7 pos=(954.6,44.9,1051.8) |
| 6.0s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 6.1s | S2C | S2C m586 状态推送(簇首) [5] |
| 6.1s | S2C | op31(sub7) 0x700014b →0x7022212 |
| 6.1s | C2S | m82 ? [3] |
| 6.2s | S2C | op2C len=220 2c47db0000008e010000000000e009007b670107 |
| 6.3s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 7.1s | S2C | op28(子码47) len=233 |
| 7.1s | C2S | m15 移动上报 自动 state=8 pos=(954.6,44.4,1051.8) |
| 7.1s | C2S | m54 格挡状态(值35577620223) |
| 7.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 7.1s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 7.1s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 7.1s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 7.1s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 7.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 7.1s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 7.1s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 7.2s | C2S | m15 移动上报 自动 state=6 pos=(954.6,44.4,1051.8) |
| 7.2s | S2C | 对象同步0x21[0x7034e9b] 码149 len=91 |
| 7.2s | S2C | 多属性0x2D[0x9fc00703](布局变体) len=210 2d010045600307c09f49001000b50004 |
| 7.4–7.6s | C2S | m15 移动上报 自动 state=4 pos=(954.6,43.9,1051.8) ×2 |
| 7.7–8.3s | S2C | 对象同步0x21[0x7034e9b] 码148 len=47 ×3 |
| 8.0s | C2S | m54 格挡状态(值35353580248) |
| 8.1s | S2C | op28(子码47) len=225 |
| 8.1s | C2S | m54 格挡状态(值35353580248) |
| 8.1s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 8.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 8.1s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 8.1s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 8.1s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 8.1s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 8.1s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 8.1s | C2S | m54 格挡状态(值35353580248) ×2 |
| 8.2s | C2S | m15 移动上报 自动 state=4 pos=(954.6,43.9,1051.8) |
| 8.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 8.9s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 9.1s | S2C | op28(子码47) len=225 ×3 |
| 9.7s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 9.7s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 10.1s | S2C | op28(子码47) len=255 |
| 14.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 14.9s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 16.7s | S2C | 属性更新0x10[O#0x150] State= |
| 16.9s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 16.9s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 18.1s | S2C | op06 len=1 06 |
| 18.3s | S2C | 对象同步0x21[0x701677b] 码32 len=91 |
| 18.3s | S2C | 记录写格0x13[Count_Limit_Form_Rec#363 行5 列1 =33555712 0x0] |
| 18.3s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 18.3s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 18.3s | S2C | S2C m252 ? [] |
| 18.3s | S2C | S2C m250 ? [] ×3 |
| 20.5s | S2C | 对象同步0x21[0x70295df] 码49 len=91 |
| 20.5s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 20.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 21.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 21.4s | S2C | 对象同步0x21[0x701677b] 码32 len=91 |
| 21.4–22.0s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 21.8s | S2C | 对象同步0x21[0x7000141] 码105 len=223 |
| 22.0s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 22.2s | S2C | op2C len=148 2c47930000000a010000000000e006007b670107 |
| 26.1s | C2S | m82 ? [3] |
| 26.2s | S2C | op31(sub1) 0x700014b |
| 26.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 27.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 28.7s | S2C | 属性更新0x10[O#0x498afc] FightData=16036568468 |
| 32.6s | S2C | op0E len=9 0e81560007739e4900 |
| 32.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 33.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 33.3s | S2C | op06 len=1 06 |
| 34.1–35.6s | S2C | 对象同步0x21[0x701677b] 码149 len=91 ×2 |
| 35.2s | S2C | 对象同步0x21[0x70295df] 码97 len=91 |
| 35.2s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 35.6–36.1s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 36.1s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 36.3s | S2C | op2C len=136 2c47870000000a010000000000e006007b670107 |
| 38.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 39.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 40.9s | S2C | 属性更新0x10[O#0x146] State=stand ×2 |
| 41.1s | S2C | 对象同步0x21[0x7000141] 码179 len=223 |
| 41.1s | S2C | 多属性0x2D[0x1460700](布局变体) len=23 2d010041010007460100000100540004 |
| 43.7s | S2C | 属性更新0x10[O#0x498afc] Sat=21 |
| 43.8s | S2C | 属性更新0x10[V#0x2] LeftTime=574892 |
| 44.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 45.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 46.1s | C2S | m82 ? [3] |
| 46.8s | S2C | S2C m1 系统提示 [0, "83308", "沈京寒 ", 22000] |
| 48.6s | S2C | op06 len=1 06 |
| 50.4s | S2C | S2C m566 令牌应答串 ["RF/MXnsAwDNGUiU59ELAHZOwgjGVj5OewKVmIM3R"] |
| 50.5s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 50.5s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x7034e9b] |
| 50.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 51.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 54.3s | S2C | 对象同步0x21[0x701677b] 码177 len=91 |
| 56.5s | S2C | 对象同步0x21[0x70295df] 码81 len=91 |
| 56.5s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 56.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 57.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 57.5s | S2C | op31(sub1) 0x7000141 |
| 57.6s | S2C | 对象同步0x21[0x701677b] 码177 len=91 |
| 57.6–58.2s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 58.2s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 58.2s | S2C | op2C len=132 2c47830000000a010000000000e006007b670107 |
| 60.4s | S2C | op28(子码47) len=255 |
| 62.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 63.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 63.8s | S2C | op06 len=1 06 |
| 66.1s | C2S | m82 ? [3] |
| 66.7s | S2C | 记录写格0x13[WeatherRec#361 行0 列1 =50331648 0x0] ×2 |
| 67.1s | S2C | 属性更新0x10[O#0x150] State= |
| 67.4s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 67.4s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 68.9s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 69.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 74.1–75.2s | S2C | 对象同步0x21[0x701677b] 码217 len=91 ×2 |
| 74.8s | S2C | 对象同步0x21[0x70295df] 码52 len=91 |
| 74.8s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 75.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 75.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 75.2–75.3s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 75.3s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 75.4s | S2C | op2C len=151 2c47960000000a010000000000e006007b670107 |
| 76.5s | S2C | op31(sub1) 0x700014b |
| 78.2s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 78.2s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 78.3s | S2C | S2C m250 ? [] ×2 |
| 79.0s | S2C | op06 len=1 06 |
| 81.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 81.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 81.6s | S2C | op28(子码47) len=257 |
| 86.1s | C2S | m82 ? [3] |
| 87.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 87.1s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 87.5s | S2C | 对象同步0x21[0x701677b] 码32 len=91 |
| 88.2s | S2C | S2C m1 系统提示 [3, "15916", "徐乄凌乄枫 ", "ini\scene\clone062", 1250] |
| 89.7s | S2C | 对象同步0x21[0x70295df] 码49 len=91 |
| 89.7s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 90.4s | S2C | 属性更新0x10[O#0x498afc] LiveGroove_4=59402 |
| 90.4s | S2C | 属性更新0x10[O#0x498afc] LiveGroove_5=6936 |
| 90.4s | S2C | S2C m1 系统提示 [1, "8052", 223] |
| 90.4s | S2C | S2C m349 ? [223] |
| 90.4s | S2C | 属性更新0x10[O#0x498afc] Faculty=114338031 |
| 90.4s | S2C | 属性更新0x10[O#0x498afc] HoldFaculty=1525306 |
| 90.5s | S2C | 对象同步0x21[0x701677b] 码32 len=91 |
| 90.5–91.4s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 91.4s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 91.4s | S2C | op2C len=148 2c47930000000a010000000000e006007b670107 |
| 93.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 93.0s | S2C | S2C m1 系统提示 [2, "555504", "钟卿鹿梦远 ", "tableitem_CF_01_CD"] |
| 93.0s | C2S | m15 移动上报 自动 state=5 pos=(954.6,43.9,1051.8) ×2 |
| 94.3s | S2C | op06 len=1 06 |
| 96.5s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [954.65, 43.887, 1051.757, -2.854] |
| 96.5s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 96.5s | S2C | S2C m1 系统提示 [2, "9041"] |
| 97.3s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [954.65, 43.887, 1051.757, -2.854] |
| 97.3s | S2C | S2C m1 系统提示 [2, "9041"] |
| 97.7s | S2C | 属性更新0x10[O#0x498afc] FightData=25355913060 |
| 98.5s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [954.65, 43.887, 1051.757, -2.854] |
| 98.5s | S2C | 位置0x1F(短len=25) |
| 98.5s | S2C | S2C m337 施法回显[0x498afc07034e9b] [954.65, 43.886, 1051.757, 3.519] |
| 98.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_jh_hyjf07 |
| 98.5s | S2C | S2C m24 通用事件[0x498afc07034e9b] 冲锋动画 [949.396, 44.886, 1038.492, 1, "qg_fastrun_2h_01"] |
| 98.5s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 98.5s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 98.5s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 98.5s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 98.5s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 98.5s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 98.5s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 98.5s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 98.5s | S2C | 属性更新0x10[O#0x498afc] MP=10298 |
| 98.5s | S2C | S2C m707 受理族变体 [15] |
| 98.6s | S2C | 对象同步0x21[0x7034e9b] 码94 len=47 |
| 98.6s | S2C | 属性更新0x10[O#0x146] State= |
| 98.7s | S2C | op31(sub2) 0x7036045 →0x7007992 |
| 98.8s | S2C | 对象同步0x21[0x7000141] 码187 len=223 |
| 98.8s | S2C | 多属性0x2D[0x1460700](布局变体) len=20 2d010041010007460100000100540001 |
| 99.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 99.1s | C2S | m15 移动上报 手动 state=5 pos=(949.7,43.9,1039.4) ×2 |
| 99.1s | C2S | m212 冲锋/续招落地 [949.533, 43.886, 1038.837, -2.765] |
| 99.1s | C2S | m15 移动上报 自动 state=4 pos=(949.4,43.9,1038.5) |
| 99.1s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 99.1s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 99.1s | S2C | 位置0x1F(短len=25) |
| 99.1s | S2C | S2C m26 技能释放 [] |
| 99.1s | S2C | 位置0x1F(短len=25) |
| 99.1s | S2C | S2C m337 施法回显[0x498afc07034e9b] [949.533, 43.886, 1038.837, 3.519] |
| 99.1s | S2C | S2C m20 范围施放标记[0x498afc07034e9b] ["CS_jh_hyjf07", 0, 0, 0.0, 948.844, 43.886] |
| 99.2s | S2C | 对象同步0x21[0x7034e9b] 码92 len=47 |
| 99.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 99.7s | S2C | op31(sub3) 0x7000141 →0x700ee8b |
| 99.7s | S2C | op28(子码47) len=222 ×5 |
| 100.3s | C2S | m54 格挡状态(值4674065458) |
| 100.7s | S2C | op28(子码47) len=255 ×2 |
| 100.7s | C2S | m54 格挡状态(值4898105433) |
| 100.8s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 100.8s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 100.8s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 100.8s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 100.8s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 100.8s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 100.8s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 100.9s | C2S | m54 格挡状态(值4674065458) |
| 101.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 101.0s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 101.0s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 101.0s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 101.0s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 101.0s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 101.0s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 101.0s | C2S | m54 格挡状态(值4674065458) ×3 |
| 101.1s | C2S | m15 移动上报 自动 state=4 pos=(949.5,43.9,1038.8) |
| 101.2–102.9s | S2C | 对象同步0x21[0x7034e9b] 码28 len=47 ×2 |
| 102.1s | C2S | m54 格挡状态(值4898105433) |
| 102.1s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 102.1s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 102.1s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 102.1s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 102.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 102.1s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 102.1s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 102.2s | C2S | m54 格挡状态(值4674065458) |
| 102.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 102.3s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 102.3s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 102.3s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 102.3s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 102.3s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 102.3s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 102.3s | C2S | m54 格挡状态(值4674065458) ×2 |
| 102.4s | S2C | op02 len=369 0200000000000000000000000000000000000000 |
| 102.5s | C2S | op01 len=53 010000000000000000000000000000000000000000000000 |
| 102.7s | C2S | m15 移动上报 自动 state=4 pos=(949.5,43.9,1038.8) |
| 102.8s | S2C | op28(子码47) len=233 |
| 103.2s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 103.2s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 103.3–104.8s | S2C | 对象同步0x21[0x701677b] 码149 len=91 ×2 |
| 103.8s | S2C | S2C m1 系统提示 [0, "30359", "云海映星辰 ", "bossworld001_small"] |
| 103.8s | S2C | S2C m1 系统提示 [1, "sys_world_auction_bossworld001_small"] |
| 103.9s | S2C | S2C m603 ? [0, 0] |
| 103.9s | S2C | 属性更新0x10[V#0x2] LeftTime=574832 |
| 104.4s | S2C | 对象同步0x21[0x70295df] 码97 len=91 |
| 104.4s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 104.8s | S2C | 属性更新0x10[O#0x499fa8] State=stand |
| 105.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 105.1s | C2S | m15 移动上报 自动 state=5 pos=(949.5,43.9,1038.8) ×2 |
| 105.4s | S2C | 属性更新0x10[O#0x499fa8] State=walk |
| 105.4s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 105.4s | C2S | m15 移动上报 自动 state=8 pos=(949.5,43.9,1038.8) |
| 105.5s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 105.5s | S2C | op2C len=171 2c47aa00000036010000000000e007007b670107 |
| 106.1s | C2S | m82 ? [3] |
| 106.2s | C2S | m15 移动上报 自动 state=8 pos=(949.5,45.2,1038.8) |
| 106.2s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=158 |
| 106.2s | S2C | 属性更新0x10[O#0x498afc] JumpTimes=1 |
| 106.4–110.1s | S2C | 对象同步0x21[0x7034e9b] 码28 len=47 ×11 |
| 106.4–106.7s | C2S | m15 移动上报 自动 state=6 pos=(949.5,47.3,1038.8) ×2 |
| 107.1s | C2S | m15 移动上报 自动 state=8 pos=(949.5,51.0,1038.8) |
| 107.2s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=152 |
| 107.4s | C2S | m15 移动上报 自动 state=6 pos=(949.5,53.1,1038.8) |
| 108.0s | C2S | m216 使用轻功 ["qinggong_9", 1] |
| 108.1s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=142 |
| 108.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=DA,7034E9B,498AFC,1,0,0 |
| 108.1s | S2C | S2C m169 buff添加 ["buf_qg_light", obj(0x7034e9b,0x498afc)] |
| 108.1s | C2S | m216 使用轻功 ["qinggong_9", 1] ×3 |
| 108.1–108.5s | C2S | m15 移动上报 自动 state=7 pos=(949.5,55.1,1038.8) ×2 |
| 108.5s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=152 |
| 108.5s | C2S | m216 使用轻功 ["qinggong_9", 0] |
| 108.6s | S2C | S2C m169 buff移除 ["buf_qg_light", obj(0x7034e9b,0x498afc)] |
| 108.6s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 108.6s | C2S | m216 使用轻功 ["qinggong_9", 0] |
| 108.7–108.9s | C2S | m15 移动上报 自动 state=8 pos=(949.5,52.5,1038.8) ×2 |
| 109.5s | C2S | m15 移动上报 自动 state=6 pos=(949.5,50.3,1038.8) |
| 109.5s | S2C | op06 len=1 06 |
| 109.8–110.0s | C2S | m15 移动上报 自动 state=6 pos=(949.5,48.3,1038.8) ×2 |
| 110.1s | S2C | 属性更新0x10[O#0x498afc] JumpTimes=0 |
| 110.1s | C2S | m15 移动上报 自动 state=6 pos=(949.5,44.1,1038.8) |
| 110.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 110.2s | C2S | m15 移动上报 自动 state=4 pos=(949.5,43.9,1038.8) |
| 110.3–112.0s | S2C | 对象同步0x21[0x7034e9b] 码27 len=47 ×7 |
| 110.4s | C2S | m15 移动上报 自动 state=8 pos=(949.5,43.9,1038.8) |
| 110.5s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 110.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 110.6s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x7034e9b] |
| 111.0s | C2S | m15 移动上报 自动 state=8 pos=(949.5,45.4,1038.8) |
| 111.0s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=145 |
| 111.0s | S2C | 属性更新0x10[O#0x498afc] JumpTimes=1 |
| 111.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 111.0s | C2S | m15 移动上报 自动 state=5 pos=(949.5,46.1,1038.8) ×2 |
| 111.2–111.5s | C2S | m15 移动上报 自动 state=6 pos=(949.5,47.6,1038.8) ×2 |
| 111.5s | C2S | m15 移动上报 自动 state=8 pos=(949.5,49.7,1038.8) |
| 111.5s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=155 ×2 |
| 111.8s | C2S | m15 移动上报 自动 state=6 pos=(949.5,51.8,1038.8) |
| 112.1s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [949.533, 53.46, 1038.837, 0.885] |
| 112.2s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 112.2s | S2C | 位置0x1F(短len=25) |
| 112.2s | S2C | S2C m337 施法回显[0x498afc07034e9b] [949.533, 52.704, 1038.837, 0.474] |
| 112.2s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_jh_hyjf07 |
| 112.2s | S2C | S2C m24 通用事件[0x498afc07034e9b] 冲锋动画 [955.044, 44.886, 1049.586, 2, "qg_fastruning_s_2h_01"] |
| 112.2s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 112.2s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 112.2s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 112.2s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 112.2s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 112.2s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 112.2s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 112.2s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 112.2s | S2C | 属性更新0x10[O#0x498afc] MP=10298 |
| 112.2s | S2C | S2C m707 受理族变体 [15] |
| 112.2s | C2S | m15 移动上报 自动 state=6 pos=(949.5,52.7,1038.8) |
| 112.2s | S2C | 对象同步0x21[0x7034e9b] 码211 len=47 |
| 112.7s | S2C | 属性更新0x10[O#0x498afc] JumpTimes=0 |
| 112.8s | S2C | op31(sub5) 0x700fed1 →0x700eb44 |
| 112.9s | C2S | m212 冲锋/续招落地 [955.024, 44.919, 1049.547, 0.474] |
| 112.9s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 112.9s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 112.9s | S2C | 位置0x1F(短len=25) |
| 112.9s | S2C | S2C m26 技能释放 [] |
| 112.9s | S2C | 位置0x1F(短len=25) |
| 112.9s | S2C | S2C m337 施法回显[0x498afc07034e9b] [955.024, 44.919, 1049.547, 0.474] |
| 112.9s | S2C | S2C m20 范围施放标记[0x498afc07034e9b] ["CS_jh_hyjf07", 0, 0, 0.0, 955.729, 43.886] |
| 112.9s | C2S | m15 移动上报 手动 state=8 pos=(955.0,44.9,1049.6) |
| 112.9s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 112.9s | C2S | m15 移动上报 自动 state=4 pos=(955.0,44.9,1049.5) |
| 112.9s | C2S | m15 移动上报 自动 state=7 pos=(955.0,44.9,1049.5) |
| 113.0s | S2C | S2C m586 状态推送(簇首) [5] |
| 113.1s | S2C | 对象同步0x21[0x7034e9b] 码212 len=135 |
| 113.2s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 113.8s | S2C | op28(子码47) len=225 ×4 |
| 114.1s | C2S | m15 移动上报 自动 state=8 pos=(955.0,44.4,1049.5) |
| 114.1s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 114.1s | C2S | m54 格挡状态(值4898105433) |
| 114.2s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 114.2s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 114.2s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 114.2s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 114.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 114.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 114.2s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 114.2–114.4s | S2C | 对象同步0x21[0x7034e9b] 码138 len=47 ×2 |
| 114.2s | C2S | m15 移动上报 自动 state=6 pos=(955.0,44.3,1049.5) |
| 114.2s | S2C | S2C m1 系统提示 [2, "85201", "徐一萧影一 ", "qy_box_tableitem_TJ_02", "tableitem_TJ_02_JL"] |
| 114.4s | C2S | m15 移动上报 自动 state=4 pos=(955.0,43.9,1049.5) |
| 114.5s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=159 |
| 114.6–118.3s | S2C | 对象同步0x21[0x7034e9b] 码137 len=47 ×9 |
| 114.7s | C2S | m54 格挡状态(值4674065458) |
| 114.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 114.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 114.7s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 114.7s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 114.7s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 114.7s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 114.7s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 114.7s | C2S | m54 格挡状态(值4674065458) ×3 |
| 114.9s | C2S | m15 移动上报 自动 state=4 pos=(955.0,43.9,1049.5) |
| 115.4s | C2S | m15 移动上报 自动 state=8 pos=(955.0,43.9,1049.5) |
| 115.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 115.7s | S2C | 对象同步0x21[0x7000141] 码105 len=223 |
| 116.0s | C2S | m15 移动上报 自动 state=8 pos=(955.0,45.4,1049.5) |
| 116.1s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=152 |
| 116.1s | S2C | 属性更新0x10[O#0x498afc] JumpTimes=1 |
| 116.3–116.5s | C2S | m15 移动上报 自动 state=6 pos=(955.0,47.5,1049.5) ×2 |
| 116.8s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 116.8s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 116.9s | S2C | op31(sub1) 0x7036045 |
| 117.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 117.1s | C2S | m15 移动上报 自动 state=5 pos=(955.0,51.3,1049.5) ×2 |
| 117.4s | C2S | m15 移动上报 自动 state=6 pos=(955.0,51.1,1049.5) |
| 117.5s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=162 |
| 117.7s | S2C | 属性更新0x10[O#0x150] State= |
| 117.8s | C2S | m15 移动上报 自动 state=6 pos=(955.0,49.1,1049.5) |
| 117.9s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 117.9s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 118.1–118.3s | C2S | m15 移动上报 自动 state=6 pos=(955.0,47.1,1049.5) ×2 |
| 118.3s | S2C | 属性更新0x10[O#0x498afc] JumpTimes=0 |
| 118.4s | C2S | m15 移动上报 自动 state=4 pos=(955.0,43.9,1049.5) |
| 118.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 118.5–123.7s | S2C | 对象同步0x21[0x7034e9b] 码136 len=47 ×8 |
| 119.0s | C2S | m54 格挡状态(值4898105433) |
| 119.0s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 119.0s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 119.0s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 119.0s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 119.0s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 119.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 119.0s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 119.5s | C2S | m15 移动上报 手动 state=4 pos=(955.0,43.9,1049.5) |
| 119.9s | S2C | op28(子码47) len=225 |
| 120.0–120.5s | C2S | m15 移动上报 手动 state=4 pos=(955.0,43.9,1049.5) ×2 |
| 120.5s | S2C | 属性更新0x10[O#0x498afc] QingGongPoint=165 |
| 120.6s | C2S | m54 格挡状态(值4674065458) |
| 120.6s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 120.6s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 120.6s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 120.6s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 120.6s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 120.6s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 120.6s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 120.6s | C2S | m54 格挡状态(值4674065458) ×3 |
| 120.9s | C2S | m54 格挡状态(值4898105433) |
| 121.0s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 121.0s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 121.0s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 121.0s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 121.0s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 121.0s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 121.0s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 121.0s | C2S | m15 移动上报 手动 state=4 pos=(955.0,43.9,1049.5) |
| 121.1s | S2C | S2C m1 系统提示 [3, "15917", "凤君乖乖 ", "ini\scene\clone047", 2000] |
| 121.5s | C2S | m15 移动上报 手动 state=4 pos=(955.0,43.9,1049.5) |
| 121.9s | C2S | m54 格挡状态(值4674065458) |
| 121.9s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 121.9s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 121.9s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 121.9s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 121.9s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 121.9s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 121.9s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 121.9s | C2S | m54 格挡状态(值4674065458) ×3 |
| 122.0s | C2S | m15 移动上报 自动 state=4 pos=(955.0,43.9,1049.5) |
| 123.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 123.0s | C2S | m15 移动上报 自动 state=5 pos=(955.0,43.9,1049.5) ×2 |
| 123.1s | C2S | m54 格挡状态(值4898105433) |
| 123.2s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 123.2s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 123.2s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 123.2s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 123.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 123.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 123.2s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 123.3s | C2S | m54 格挡状态(值4674065458) |
| 123.3s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 123.3s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 123.3s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 123.3s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 123.3s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 123.3s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 123.3s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 123.3s | C2S | m54 格挡状态(值4674065458) ×3 |
| 123.5s | C2S | m15 移动上报 自动 state=4 pos=(955.0,43.9,1049.5) |
| 123.5s | S2C | 对象同步0x21[0x701677b] 码177 len=91 |
| 124.7s | S2C | op06 len=1 06 |
| 125.7s | S2C | 对象同步0x21[0x70295df] 码81 len=91 |
| 125.7s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 126.1s | C2S | m82 ? [3] |
| 126.8s | S2C | 对象同步0x21[0x701677b] 码177 len=91 |
| 126.8–127.5s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 127.5s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 127.6s | S2C | op2C len=132 2c47830000000a010000000000e006007b670107 |
| 128.9s | S2C | op31(sub1) 0x700014b |
| 129.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 129.1s | C2S | m15 移动上报 自动 state=5 pos=(955.0,43.9,1049.5) ×2 |
| 132.2s | C2S | m54 格挡状态(值4898105433) |
| 132.3s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 132.3s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 132.3s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 132.3s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 132.3s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 132.3s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 132.3s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 132.7s | C2S | m15 移动上报 手动 state=4 pos=(955.0,43.9,1049.5) |
| 132.8s | C2S | m54 格挡状态(值4674065458) ×4 |
| 132.8–136.9s | S2C | 对象同步0x21[0x7034e9b] 码136 len=47 ×6 |
| 132.9s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 132.9s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 132.9s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 132.9s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 132.9s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 132.9s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 132.9s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 132.9s | C2S | m54 格挡状态(值4674065458) |
| 133.6s | C2S | m54 格挡状态(值4898105433) |
| 133.7s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 133.7s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 133.7s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 133.7s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 133.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 133.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 133.7s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 133.7–134.2s | C2S | m15 移动上报 手动 state=4 pos=(955.0,43.9,1049.5) ×2 |
| 134.8s | S2C | 属性更新0x10[O#0x146] State=stand ×2 |
| 135.0s | S2C | 对象同步0x21[0x7000141] 码179 len=223 |
| 135.0s | S2C | 多属性0x2D[0x1460700](布局变体) len=23 2d010041010007460100000100540004 |
| 135.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 135.1s | C2S | m15 移动上报 自动 state=5 pos=(955.0,43.9,1049.5) ×2 |
| 135.7–136.7s | C2S | m15 移动上报 手动 state=4 pos=(955.0,43.9,1049.5) ×3 |
| 138.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5=1EA6,7034E9B,498AFC,0,0,1 |
| 138.7s | S2C | S2C m169 buff添加 ["BuffInParry_consume", obj(0x7034e9b,0x498afc)] |
| 139.7s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=99 |
| 139.7s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19660 |
| 139.7s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=26 [198, 0, 0, "BuffInParry_consume", "Buffer", "徐傠囃辭睼 "] |
| 140.0s | S2C | op06 len=1 06 |
| 140.7s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19671 |
| 140.7s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=98 |
| 140.7s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19473 |
| 140.7s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=26 [198, 0, 0, "BuffInParry_consume", "Buffer", "徐傠囃辭睼 "] |
| 141.0s | C2S | m54 格挡状态(值4674065458) |
| 141.0s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] ×2 |
| 141.0s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5= |
| 141.0s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 141.0s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 141.0s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 141.0s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 141.0s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 141.0s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 141.0s | C2S | m54 格挡状态(值4674065458) |
| 141.0s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 141.0s | C2S | m15 移动上报 自动 state=5 pos=(955.0,43.9,1049.5) ×2 |
| 141.1s | C2S | m54 格挡状态(值4674065458) |
| 141.7s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19587 |
| 142.7s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=99 |
| 142.7–143.7s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19701 ×2 |
| 143.4–144.5s | S2C | 对象同步0x21[0x701677b] 码217 len=91 ×2 |
| 144.1s | S2C | 对象同步0x21[0x70295df] 码52 len=91 |
| 144.1s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 144.1s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [955.024, 43.887, 1049.546, 2.13] |
| 144.1s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 144.1s | S2C | 位置0x1F(短len=25) |
| 144.1s | S2C | S2C m337 施法回显[0x498afc07034e9b] [955.024, 43.886, 1049.546, 3.171] |
| 144.1s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_jh_hyjf07 |
| 144.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] 冲锋动画 [954.65, 44.886, 1036.923, 1, "qg_fastrun_2h_01"] |
| 144.1s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 144.1s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 144.1s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 144.1s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 144.1s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 144.1s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 144.1s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 144.1s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 144.1s | S2C | 属性更新0x10[O#0x498afc] MP=10298 |
| 144.1s | S2C | S2C m707 受理族变体 [15] |
| 144.3s | S2C | 对象同步0x21[0x7034e9b] 码158 len=47 |
| 144.4s | S2C | 记录写格0x13[Count_Limit_Form_Rec#363 行5 列1 =33555712 0x0] |
| 144.4s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 144.4s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 144.4s | S2C | S2C m252 ? [] |
| 144.4s | S2C | S2C m250 ? [] ×3 |
| 144.5s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 144.6s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 144.7s | C2S | m212 冲锋/续招落地 [954.661, 43.886, 1037.286, -3.112] |
| 144.7s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19858 |
| 144.7s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=100 |
| 144.7s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19858 |
| 144.7s | C2S | m15 移动上报 自动 state=4 pos=(954.6,43.9,1036.9) |
| 144.8s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 144.8s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 144.8s | S2C | 位置0x1F(短len=25) |
| 144.8s | S2C | S2C m26 技能释放 [] |
| 144.8s | S2C | 位置0x1F(短len=25) |
| 144.8s | S2C | S2C m337 施法回显[0x498afc07034e9b] [954.661, 43.886, 1037.286, 3.171] |
| 144.8s | S2C | S2C m20 范围施放标记[0x498afc07034e9b] ["CS_jh_hyjf07", 0, 0, 0.0, 954.606, 43.886] |
| 144.8s | S2C | op2C len=151 2c47960000000a010000000000e006007b670107 |
| 145.0s | S2C | 对象同步0x21[0x7034e9b] 码155 len=47 |
| 145.1s | S2C | op31(sub5) 0x7000141 →0x7036045 |
| 145.2s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 146.1s | S2C | op28(子码47) len=222 ×5 |
| 146.1s | C2S | m82 ? [3] |
| 146.6s | C2S | m54 格挡状态(值4898105433) |
| 146.7s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 146.7s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 146.7s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 146.7s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 146.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 146.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 146.7s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 146.7s | C2S | m15 移动上报 手动 state=4 pos=(954.7,43.9,1037.3) |
| 146.9s | C2S | m54 格挡状态(值4674065458) |
| 146.9s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 146.9s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 146.9s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 146.9s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 146.9s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 146.9s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 146.9s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 146.9s | C2S | m54 格挡状态(值4674065458) ×3 |
| 147.0–147.4s | S2C | 对象同步0x21[0x7034e9b] 码75 len=47 ×2 |
| 147.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 147.1s | C2S | m15 移动上报 自动 state=5 pos=(954.7,43.9,1037.3) ×2 |
| 147.3s | C2S | m15 移动上报 自动 state=4 pos=(954.7,43.9,1037.3) |
| 148.8s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 148.8s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 150.1s | S2C | op28(子码47) len=255 |
| 153.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 153.1s | C2S | m15 移动上报 自动 state=5 pos=(954.7,43.9,1037.3) ×2 |
| 155.2s | S2C | op06 len=1 06 |
| 155.8s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [954.661, 43.887, 1037.286, 1.661] |
| 155.8s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 155.8s | S2C | 位置0x1F(短len=25) |
| 155.8s | S2C | S2C m337 施法回显[0x498afc07034e9b] [954.661, 43.886, 1037.286, 2.266] |
| 155.8s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_jh_hyjf07 |
| 155.8s | S2C | S2C m24 通用事件[0x498afc07034e9b] 冲锋动画 [956.469, 44.886, 1035.776, 1, "qg_fastrun_2h_01"] |
| 155.8s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 155.8s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 155.8s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 155.8s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 155.8s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 155.8s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 155.8s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 155.8s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 155.8s | S2C | 属性更新0x10[O#0x498afc] MP=10298 |
| 155.8s | S2C | S2C m707 受理族变体 [15] |
| 156.0s | S2C | 对象同步0x21[0x7034e9b] 码12 len=47 |
| 156.1s | C2S | m212 冲锋/续招落地 [956.432, 43.886, 1035.808, 2.266] |
| 156.2s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 156.2s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 156.2s | S2C | 位置0x1F(短len=25) |
| 156.2s | S2C | S2C m26 技能释放 [] |
| 156.2s | S2C | 位置0x1F(短len=25) |
| 156.2s | S2C | S2C m337 施法回显[0x498afc07034e9b] [956.432, 43.886, 1035.808, 2.267] |
| 156.2s | S2C | S2C m20 范围施放标记[0x498afc07034e9b] ["CS_jh_hyjf07", 0, 0, 0.0, 957.621, 43.886] |
| 156.2s | C2S | m15 移动上报 自动 state=4 pos=(956.5,43.9,1035.8) |
| 156.3s | S2C | 对象同步0x21[0x7034e9b] 码14 len=47 |
| 156.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 156.9s | S2C | 对象同步0x21[0x701677b] 码32 len=91 |
| 157.4s | C2S | m54 格挡状态(值4674065458) |
| 159.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 159.1s | S2C | 对象同步0x21[0x70295df] 码49 len=91 |
| 159.1s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 159.1s | C2S | m15 移动上报 自动 state=5 pos=(956.4,43.9,1035.8) ×2 |
| 160.0s | S2C | 对象同步0x21[0x701677b] 码32 len=91 |
| 160.0s | S2C | 属性更新0x10[O#0x499fa8] State=stand |
| 160.5s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 160.5s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 160.7s | S2C | 属性更新0x10[O#0x499fa8] State=walk |
| 160.7s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 160.8s | S2C | op2C len=148 2c47930000000a010000000000e006007b670107 |
| 163.9s | S2C | 属性更新0x10[V#0x2] LeftTime=574772 |
| 165.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 165.2s | C2S | m15 移动上报 自动 state=5 pos=(956.4,43.9,1035.8) ×2 |
| 166.1s | C2S | m82 ? [3] |
| 166.7s | S2C | 属性更新0x10[O#0x498afc] FightData=22030534485 |
| 168.2s | S2C | 属性更新0x10[O#0x150] State= |
| 168.6s | S2C | 对象同步0x21[0x700014b] 码198 len=223 |
| 168.6s | S2C | 多属性0x2D[0x1500700](布局变体) len=20 2d01004b010007500100000100540001 |
| 170.4s | S2C | S2C m566 令牌应答串 ["aPpFHixwYZd+tLmjgk7grgTr6wIIjLww8MDwFgjY"] |
| 170.5s | S2C | op06 len=1 06 |
| 170.6s | S2C | S2C m229 小数值推送 [144, 0] ×3 |
| 170.7s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x7034e9b] |
| 171.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 171.2s | C2S | m15 移动上报 自动 state=5 pos=(956.4,43.9,1035.8) ×2 |
| 172.7–174.2s | S2C | 对象同步0x21[0x701677b] 码149 len=91 ×2 |
| 173.7s | S2C | 对象同步0x21[0x70295df] 码97 len=91 |
| 173.7s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 174.2–174.7s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 174.7s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 174.8s | S2C | op2C len=136 2c47870000000a010000000000e006007b670107 |
| 177.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 177.2s | C2S | m15 移动上报 自动 state=5 pos=(956.4,43.9,1035.8) ×2 |
| 179.6s | C2S | m15 移动上报 自动 state=7 pos=(956.4,43.9,1035.8) |
| 179.6s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 179.6s | C2S | m15 移动上报 自动 state=7 pos=(956.4,43.9,1035.8) |
| 179.7s | S2C | 对象同步0x21[0x7034e9b] 码188 len=47 |
| 179.9s | C2S | m15 移动上报 自动 state=4 pos=(957.5,43.9,1036.5) |
| 179.9s | S2C | 对象同步0x21[0x7034e9b] 码168 len=47 |
| 180.1s | S2C | 对象同步0x21[0x7034e9b] 码184 len=47 |
| 180.2s | S2C | op31(sub1) 0x700835d |
| 181.0s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [957.542, 43.887, 1036.504, 1.012] |
| 181.1s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 181.1s | S2C | 位置0x1F(短len=25) |
| 181.1s | S2C | S2C m337 施法回显[0x498afc07034e9b] [957.542, 43.886, 1036.504, 0.737] |
| 181.1s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_jh_hyjf07 |
| 181.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] 冲锋动画 [962.559, 44.881, 1042.036, 1, "qg_fastrun_2h_01"] |
| 181.1s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 181.1s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 181.1s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 181.1s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 181.1s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 181.1s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 181.1s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 181.1s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 181.1s | S2C | 属性更新0x10[O#0x498afc] MP=10298 |
| 181.1s | S2C | S2C m707 受理族变体 [15] |
| 181.2s | S2C | 对象同步0x21[0x7034e9b] 码197 len=47 |
| 181.3s | S2C | op31(sub1) 0x700f0e5 |
| 181.3s | S2C | op28(子码47) len=257 |
| 181.4s | C2S | m212 冲锋/续招落地 [962.182, 43.885, 1041.619, 0.737] |
| 181.5s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 181.5s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 181.5s | S2C | 位置0x1F(短len=25) |
| 181.5s | S2C | S2C m26 技能释放 [] |
| 181.5s | S2C | 位置0x1F(短len=25) |
| 181.5s | S2C | S2C m337 施法回显[0x498afc07034e9b] [962.182, 43.885, 1041.619, 0.736] |
| 181.5s | S2C | S2C m20 范围施放标记[0x498afc07034e9b] ["CS_jh_hyjf07", 0, 0, 0.0, 963.566, 43.881] |
| 181.5s | C2S | m15 移动上报 自动 state=4 pos=(962.6,43.9,1042.0) |
| 181.6s | S2C | 对象同步0x21[0x7034e9b] 码203 len=47 |
| 181.8s | S2C | opCE len=32 ceafccf192f092f190c1a6cda1c1a151fe61f102 |
| 181.8s | S2C | S2C m18 战斗伤害帧[0x3bcdd507036191] ["hurt_half", 0, obj(0x7034e9b,0x498afc)] |
| 181.8s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [3] |
| 181.8s | S2C | S2C m24 通用事件[0x3bcdd507036191] c1=56 [] |
| 181.8s | S2C | S2C m24 通用事件[0x3bcdd507036191] c1=4 [obj(0x7034e9b,0x498afc), 1, 2000, 3.923] |
| 181.8s | S2C | 位置0x1F(短len=25) |
| 181.8s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_jh_hyjf07", obj(0x700adff,0x3bcdd0), obj(0x702a4a5,0x3bcdda), obj(0x7036191,0x3bcdd5)] |
| 181.8s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 181.8s | S2C | 属性更新0x10[V#0x28] CurTargetCount=3 |
| 181.8s | S2C | 属性更新0x10[O#0x498afc] FP=2 |
| 181.8s | S2C | 属性更新0x10[O#0x498afc] LastNGAttackNpc=monm031301 |
| 181.8s | S2C | 属性更新0x10[O#0x498afc] SP=2 |
| 181.8s | S2C | 属性更新0x10[O#0x3bcdd0] LogicState=1 |
| 181.8s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +0行 |
| 181.8s | S2C | 属性更新0x10[O#0x498afc] LogicState=1 |
| 181.8s | S2C | 属性更新0x10[O#0x3bcdd5] LogicState=1 |
| 181.8s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +1行 |
| 181.8s | S2C | 属性更新0x10[O#0x3bcdda] LogicState=1 |
| 181.8s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +2行 |
| 181.8s | S2C | 属性更新0x10[O#0x3bcdd0] CurSkillID=default_normal_skill |
| 181.8s | S2C | 属性更新0x10[O#0x3bcdd0] CurSkillTarget=None |
| 181.8s | S2C | 位置0x1F(短len=25) |
| 181.8s | S2C | S2C m337 施法回显[0x3bcdd00700adff] [964.139, 43.879, 1043.258, 4.054, 0] |
| 181.8s | S2C | S2C m20 范围施放标记[0x3bcdd00700adff] ["atkhit3", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 181.8s | S2C | S2C m1 系统提示 [2, "8066"] |
| 181.8s | S2C | S2C m24 通用事件[0x3bcdd00700adff] 伤害飘字 [2232, 1437, 2, "CS_jh_hyjf07", "SkillNormal", "monm031301"] |
| 181.8s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x7034e9b] |
| 181.8s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 181.8s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x7034e9b] ×2 |
| 181.8s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 26, 160, "ui_jh_001"] |
| 181.8s | S2C | S2C m24 通用事件[0x3bcdd00700adff] 击退 [obj(0x7034e9b,0x498afc), "monm031301", "徐傠囃辭睼 "] |
| 181.8s | S2C | S2C m18 战斗伤害帧[0x3bcdd00700adff] ["hurt_half", 0, obj(0x7034e9b,0x498afc)] |
| 181.8s | S2C | S2C m24 通用事件[0x3bcdd00700adff] 命中? [] |
| 181.8s | S2C | S2C m169 buff添加 ["eqp_wp_06a", obj(0x702a4a5,0x3bcdda)] |
| 181.8s | S2C | 属性更新0x10[O#0x498afc] FP=4 |
| 181.8s | S2C | 属性更新0x10[O#0x498afc] SP=4 |
| 181.8s | S2C | S2C m24 通用事件[0x3bcdda0702a4a5] 伤害飘字 [1860, 0, 2, "CS_jh_hyjf07", "SkillNormal", "monm031301"] |
| 181.8s | S2C | S2C m18 战斗伤害帧[0x3bcdda0702a4a5] ["hurt_half", 0, obj(0x7034e9b,0x498afc)] |
| 181.8s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [2] |
| 181.8s | S2C | S2C m24 通用事件[0x3bcdda0702a4a5] c1=56 [] |
| 181.8s | S2C | S2C m24 通用事件[0x3bcdda0702a4a5] c1=4 [obj(0x7034e9b,0x498afc), 1, 2000, 3.318] |
| 181.8s | S2C | 位置0x1F(短len=25) |
| 181.8s | S2C | 属性更新0x10[O#0x498afc] FP=6 |
| 181.8s | S2C | 属性更新0x10[O#0x498afc] SP=6 |
| 181.9s | S2C | custom0x1E(无法解TVar) |
| 182.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] 动作开始 ["buff_ng_xyhxj01", obj(0x7034e9b,0x498afc)] |
| 182.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC0945A7,1 |
| 182.1s | S2C | S2C m169 buff添加 ["buf_ng_jz_006_01", obj(0x7034e9b,0x498afc)] |
| 182.1s | S2C | S2C m169 buff刷新? ["buf_ng_jz_006_01", obj(0x7034e9b,0x498afc)] |
| 182.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC0945A7,5 |
| 182.1s | S2C | 多属性0x2D(非标子码03) len=165 2d0300ffad0007d0cd3b000600d70100 |
| 182.2s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.5 |
| 182.2s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.5 |
| 182.2s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.5 |
| 182.2s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.5 |
| 182.2s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.5 |
| 182.2s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 182.3s | S2C | op28(子码47) len=230 ×2 |
| 182.7s | C2S | m54 格挡状态(值5078226820) |
| 182.7s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 182.7s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 182.7s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 182.7s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 182.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5=11CE,7034E9B,498AFC,1,0,0 |
| 182.7s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 182.7s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 182.8s | C2S | m54 格挡状态(值4854186845) |
| 182.8s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=8 [1, obj(0x700adff,0x3bcdd0), "default_normal_skill", "徐傠囃辭睼 ", "monm031301"] |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] LastParryTime=1794144118 |
| 182.8s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +2行 |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=59.0 |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] BufferInfo6=2B1E,7034E9B,498AFC,2,C38EC09256C,0 |
| 182.8s | S2C | S2C m169 buff添加 ["eqp_wrist_01a", obj(0x7034e9b,0x498afc)] |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] SP=10 |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=99 |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19819 |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] FP=8 |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] HPRatio=99 |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] HP=28284 |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC094894,4 |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 182.8s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 182.8s | S2C | S2C m24 通用事件[0x498afc07034e9b] 伤害飘字 [1, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 182.8s | S2C | S2C m205 伤害目标列表[0x3bcdd00700adff] ["default_normal_skill", obj(0x7034e9b,0x498afc)] |
| 182.8s | S2C | 属性更新0x10[O#0x3bcdd0] State= |
| 182.8s | S2C | 属性更新0x10[O#0x3bcdd0] CurSkillID= |
| 182.8s | S2C | 属性更新0x10[O#0x3bcdd0] CurSkillTarget=None |
| 182.8s | C2S | m54 格挡状态(值4854186845) |
| 182.9s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5= |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 182.9s | C2S | m54 格挡状态(值4854186845) ×3 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.4 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.4 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.4 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.4 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.4 |
| 182.9s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=35.0 |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=69.0 |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=21.0 |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=40.0 |
| 183.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] 动作开始 ["buff_ng_xyhxj03", obj(0x7034e9b,0x498afc)] |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5=4BFB,7034E9B,498AFC,5,C38EC09498F,1 |
| 183.1s | S2C | S2C m169 buff添加 ["buf_ng_jz_006_02", obj(0x7034e9b,0x498afc)] |
| 183.1s | S2C | S2C m169 buff刷新? ["buf_ng_jz_006_02", obj(0x7034e9b,0x498afc)] |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5=4BFB,7034E9B,498AFC,5,C38EC09498F,5 |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=75.0 |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=109.0 |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=61.0 |
| 183.1s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=80.0 |
| 183.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 183.2s | C2S | m15 移动上报 自动 state=5 pos=(962.2,43.9,1041.6) ×2 |
| 183.2s | C2S | m15 移动上报 自动 state=4 pos=(962.2,43.9,1041.6) |
| 183.4s | S2C | 对象同步0x21[0x7034e9b] 码160 len=47 |
| 183.7s | S2C | 对象同步0x21[0x702a4a5] 码139 len=91 |
| 183.8s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19858 |
| 183.8s | S2C | 属性更新0x10[O#0x498afc] ParryRatio=100 |
| 183.8s | S2C | 属性更新0x10[O#0x498afc] ParryValue=19858 |
| 183.8s | S2C | S2C m169 buff移除 ["eqp_wp_06a", obj(0x702a4a5,0x3bcdda)] |
| 184.2s | S2C | 多属性0x2D[0xcdda0702](布局变体) len=20 2d0100a5a40207dacd3b000100770201 |
| 184.2s | S2C | op31(sub1) 0x700014b |
| 184.8s | C2S | m913 ? [obj(0x7036191,0x3bcdd5)] ×2 |
| 185.7s | S2C | op06 len=1 06 |
| 185.8s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 185.8s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 186.2s | C2S | m82 ? [3] |
| 186.5s | S2C | op0E len=9 0effad0007d0cd3b00 |
| 186.5s | C2S | m440 ? [] |
| 186.6s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 186.6s | C2S | m431 选择目标 [0] |
| 186.6s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 186.9s | S2C | 对象同步0x21[0x7036191] 码59 len=47 |
| 187.1s | S2C | 属性更新0x10[O#0x3bcdda] CurSkillID=default_normal_skill |
| 187.1s | S2C | 属性更新0x10[O#0x3bcdda] CurSkillTarget=None |
| 187.1s | S2C | S2C m20 范围施放标记[0x3bcdda0702a4a5] ["atkhit3", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 187.3s | S2C | 属性更新0x10[O#0x3bcdd5] State= |
| 187.3s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillID=default_normal_skill |
| 187.3s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillTarget=None |
| 187.3s | S2C | 位置0x1F(短len=25) |
| 187.3s | S2C | S2C m337 施法回显[0x3bcdd507036191] [962.91, 43.881, 1042.363, 3.916, 0] |
| 187.3s | S2C | S2C m20 范围施放标记[0x3bcdd507036191] ["atkhit1", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 187.8s | S2C | 属性更新0x10[O#0x498afc] HPRatio=100 |
| 187.8s | S2C | 属性更新0x10[O#0x498afc] HP=28285 |
| 187.8s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 188.1s | S2C | 属性更新0x10[O#0x498afc] LastAttackerNpc=monm031301 |
| 188.1s | S2C | 属性更新0x10[O#0x498afc] LastDodge=32.29 |
| 188.1s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=7 [1, obj(0x702a4a5,0x3bcdda), "default_normal_skill", "徐傠囃辭睼 ", "monm031301", 100] |
| 188.1s | S2C | 记录写格0x13[fight_relation_rec#114 行1 列1 =16777472 0x7034e9b] |
| 188.1s | S2C | 属性更新0x10[O#0x3bcdda] State= |
| 188.1s | S2C | 属性更新0x10[O#0x3bcdda] CurSkillID= |
| 188.1s | S2C | 属性更新0x10[O#0x3bcdda] CurSkillTarget=None |
| 188.1s | S2C | S2C m189 对象通知 [obj(0x702a4a5,0x3bcdda)] |
| 188.3s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=7 [1, obj(0x7036191,0x3bcdd5), "default_normal_skill", "徐傠囃辭睼 ", "monm031301", 100] |
| 188.3s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 188.3s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillID= |
| 188.3s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillTarget=None |
| 188.3s | S2C | S2C m189 对象通知 [obj(0x7036191,0x3bcdd5)] |
| 188.8s | S2C | S2C m169 buff移除 ["eqp_wrist_01a", obj(0x7034e9b,0x498afc)] |
| 188.8s | S2C | 属性更新0x10[O#0x498afc] BufferInfo6= |
| 188.8s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=59.0 |
| 189.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 189.2s | C2S | m15 移动上报 自动 state=5 pos=(962.2,43.9,1041.6) ×2 |
| 189.2s | S2C | S2C m1 系统提示 [0, "30359", "徐乄甄嗨 ", "bossworld001_small"] |
| 189.2s | S2C | S2C m1 系统提示 [1, "sys_world_auction_bossworld001_small"] |
| 189.2s | S2C | S2C m603 ? [0, 0] |
| 189.3s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillID=default_normal_skill |
| 189.3s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillTarget=None |
| 189.3s | S2C | S2C m20 范围施放标记[0x3bcdd507036191] ["atkhit2", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 190.3s | S2C | 属性更新0x10[O#0x498afc] SP=11 |
| 190.3s | S2C | 属性更新0x10[O#0x498afc] HPRatio=99 |
| 190.3s | S2C | 属性更新0x10[O#0x498afc] HP=28258 |
| 190.3s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 190.3s | S2C | 属性更新0x10[O#0x498afc] SP=21 |
| 190.3s | S2C | 属性更新0x10[O#0x498afc] HP=28268 |
| 190.3s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 190.3s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 190.3s | S2C | S2C m24 通用事件[0x498afc07034e9b] 伤害飘字 [17, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 190.3s | S2C | 属性更新0x10[O#0x498afc] HitHPRatio=99 |
| 190.3s | S2C | 属性更新0x10[O#0x498afc] HitHP=28277 |
| 190.3s | S2C | S2C m205 伤害目标列表[0x3bcdd507036191] ["default_normal_skill", obj(0x7034e9b,0x498afc)] |
| 190.3s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillID= |
| 190.3s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillTarget=None |
| 190.3s | S2C | S2C m189 对象通知 [obj(0x7036191,0x3bcdd5)] |
| 190.5s | S2C | 属性更新0x10[O#0x498afc] LiveGroove_4=59295 |
| 190.5s | S2C | 属性更新0x10[O#0x498afc] LiveGroove_5=6722 |
| 190.5s | S2C | S2C m1 系统提示 [1, "8052", 223] |
| 190.5s | S2C | S2C m349 ? [223] |
| 190.5s | S2C | 属性更新0x10[O#0x498afc] Faculty=114338254 |
| 190.5s | S2C | 属性更新0x10[O#0x498afc] HoldFaculty=1525402 |
| 190.6s | S2C | 属性更新0x10[O#0x3bcdda] CurSkillID=default_normal_skill |
| 190.6s | S2C | 属性更新0x10[O#0x3bcdda] CurSkillTarget=None |
| 190.6s | S2C | S2C m20 范围施放标记[0x3bcdda0702a4a5] ["atkhit2", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 191.2s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [962.182, 43.886, 1041.619, 0.874] |
| 191.2s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 191.2s | S2C | S2C m1 系统提示 [2, "9086"] |
| 191.6s | S2C | 属性更新0x10[O#0x498afc] SP=22 |
| 191.6s | S2C | 属性更新0x10[O#0x498afc] HP=28244 |
| 191.6s | S2C | 记录写格0x13[fight_relation_rec#114 行1 列1 =16777472 0x7034e9b] |
| 191.6s | S2C | 属性更新0x10[O#0x498afc] HP=28253 |
| 191.6s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC094799,3 |
| 191.6s | S2C | S2C m24 通用事件[0x498afc07034e9b] 伤害飘字 [15, 0, 0, "default_normal_skill", "SkillNormal", "徐傠囃辭睼 "] |
| 191.6s | S2C | 属性更新0x10[O#0x498afc] HitHP=28270 |
| 191.6s | S2C | S2C m205 伤害目标列表[0x3bcdda0702a4a5] ["default_normal_skill", obj(0x7034e9b,0x498afc)] |
| 191.6s | S2C | 属性更新0x10[O#0x3bcdda] CurSkillID= |
| 191.6s | S2C | 属性更新0x10[O#0x3bcdda] CurSkillTarget=None |
| 191.6s | S2C | S2C m189 对象通知 [obj(0x702a4a5,0x3bcdda)] |
| 191.7s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.3 |
| 191.7s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.3 |
| 191.7s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.3 |
| 191.7s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.3 |
| 191.7s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.3 |
| 191.7s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 191.9s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillID=default_normal_skill |
| 191.9s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillTarget=None |
| 191.9s | S2C | S2C m20 范围施放标记[0x3bcdd507036191] ["atkhit3", obj(0x7034e9b,0x498afc), 0, 0.0] |
| 191.9s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [962.182, 43.886, 1041.619, 0.874] |
| 192.0s | S2C | 位置0x1F(短len=25) |
| 192.0s | S2C | S2C m337 施法回显[0x498afc07034e9b] [962.182, 43.885, 1041.619, 6.084] |
| 192.0s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_jh_hyjf07 |
| 192.0s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 192.0s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 192.0s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 192.0s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 192.0s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 192.0s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 192.0s | S2C | 属性更新0x10[O#0x498afc] MP=10298 |
| 192.0s | S2C | 位置0x1F(短len=25) |
| 192.0s | S2C | S2C m337 施法回显[0x498afc07034e9b] [962.182, 43.885, 1041.619, 7.058, 0] |
| 192.0s | S2C | S2C m20 范围施放标记[0x498afc07034e9b] ["CS_jh_hyjf07", 0, 0, 0.0, 961.807, 43.886] |
| 192.0s | C2S | m15 移动上报 自动 state=4 pos=(962.2,43.9,1041.6) |
| 192.1s | S2C | 对象同步0x21[0x7034e9b] 码160 len=47 |
| 192.3s | S2C | 属性更新0x10[O#0x146] State= |
| 192.4s | S2C | 属性更新0x10[V#0x28] CurTargetCount=2 |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] FP=10 |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] SP=24 |
| 192.4s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 192.4s | S2C | S2C m24 通用事件[0x3bcdd507036191] 伤害飘字 [372, 1908, 2, "CS_jh_hyjf07", "SkillNormal", "monm031301"] |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC0946A8,2 |
| 192.4s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x7034e9b] |
| 192.4s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 192.4s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =67108864 0x7034e9b] ×2 |
| 192.4s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 33, 160, "ui_jh_001"] |
| 192.4s | S2C | S2C m24 通用事件[0x3bcdd507036191] 击退 [obj(0x7034e9b,0x498afc), "monm031301", "徐傠囃辭睼 "] |
| 192.4s | S2C | S2C m18 战斗伤害帧[0x3bcdd507036191] ["hurt_half", 0, obj(0x7034e9b,0x498afc)] |
| 192.4s | S2C | S2C m24 通用事件[0x3bcdd507036191] 命中? [] |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] FP=12 |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] SP=26 |
| 192.4s | S2C | 属性更新0x10[O#0x3bcdda] InParry=1 |
| 192.4s | S2C | 记录写格0x13[fight_relation_rec#114 行0 列1 =16777216 0x7034e9b] |
| 192.4s | S2C | S2C m24 通用事件[0x3bcdda0702a4a5] 伤害飘字 [372, 1908, 2, "CS_jh_hyjf07", "SkillNormal", "monm031301"] |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=4BF9,7034E9B,498AFC,5,C38EC0946A9,1 |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5=4BFB,7034E9B,498AFC,5,C38EC094A91,4 |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=65.0 |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=49.0 |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=51.0 |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=70.0 |
| 192.4s | S2C | 记录写格0x13[DailyLPRec#39 行0 列1 =33554432 0x7034e9b] |
| 192.4s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 192.4s | S2C | 属性更新0x10[O#0x3bcdda] InParry=0 |
| 192.4s | S2C | 记录写格0x13[JhBoss_Accept_Rec#22 行0 列1 =16777216 0x7034e9b] |
| 192.4s | S2C | S2C m446 区域boss进度 [2, 146, "quyuboss_emei2", 40, 160, "ui_jh_001"] |
| 192.4s | S2C | S2C m24 通用事件[0x3bcdda0702a4a5] 击退 [obj(0x7034e9b,0x498afc), "monm031301", "徐傠囃辭睼 "] |
| 192.4s | S2C | S2C m18 战斗伤害帧[0x3bcdda0702a4a5] ["hurt_half", 0, obj(0x7034e9b,0x498afc)] |
| 192.4s | S2C | S2C m24 通用事件[0x498afc07034e9b] 怒气段 [2] |
| 192.4s | S2C | S2C m24 通用事件[0x3bcdda0702a4a5] 命中? [] |
| 192.4s | S2C | S2C m205 伤害目标列表[0x498afc07034e9b] ["CS_jh_hyjf07", obj(0x7036191,0x3bcdd5), obj(0x702a4a5,0x3bcdda)] |
| 192.4s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 192.5s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.1 |
| 192.5s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.1 |
| 192.5s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.1 |
| 192.5s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.1 |
| 192.5s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.1 |
| 192.5s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 192.6s | S2C | 对象同步0x21[0x7000141] 码187 len=223 |
| 192.6s | S2C | 多属性0x2D[0x1460700](布局变体) len=20 2d010041010007460100000100540001 |
| 192.8s | S2C | 多属性0x2D(非标子码02) len=122 2d020091610307d5cd3b000600d70100 |
| 192.9s | S2C | S2C m24 通用事件[0x498afc07034e9b] c1=7 [1, obj(0x7036191,0x3bcdd5), "default_normal_skill", "徐傠囃辭睼 ", "monm031301", 100] |
| 192.9s | S2C | 记录加行0x11[fight_relation_rec#114 0x7034e9b/0x498afc] +0行 |
| 192.9s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillID= |
| 192.9s | S2C | 属性更新0x10[O#0x3bcdd5] CurSkillTarget=None |
| 193.0s | S2C | 对象同步0x21[0x701677b] 码177 len=91 |
| 193.4s | S2C | S2C m1 系统提示 [3, "15917", "徐夕凡 ", "ini\scene\clone047", 2000] |
| 194.3s | S2C | 属性更新0x10[O#0x3bcdda] LogicState=0 |
| 195.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 195.1s | S2C | 对象同步0x21[0x70295df] 码81 len=91 |
| 195.1s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 195.1s | C2S | m15 移动上报 自动 state=5 pos=(962.2,43.9,1041.6) ×2 |
| 195.2s | C2S | m54 格挡状态(值5078226820) |
| 195.2s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 195.2s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 195.2s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 195.2s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 195.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo6=11CE,7034E9B,498AFC,1,0,0 |
| 195.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 195.2s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 195.3s | S2C | 属性更新0x10[O#0x498afc] HP=28270 |
| 195.6s | C2S | m54 格挡状态(值4854186845) |
| 195.6s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 195.6s | S2C | 属性更新0x10[O#0x498afc] BufferInfo6= |
| 195.6s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 195.6s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 195.6s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 195.6s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 195.6s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 195.6s | C2S | m54 格挡状态(值4854186845) ×3 |
| 196.1s | C2S | m15 移动上报 自动 state=7 pos=(962.2,43.9,1041.6) |
| 196.2s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 196.2s | C2S | m15 移动上报 自动 state=7 pos=(962.2,43.9,1041.6) |
| 196.2s | S2C | 对象同步0x21[0x7034e9b] 码153 len=135 |
| 196.2s | S2C | 属性更新0x10[O#0x499fa8] State=stand |
| 196.5s | S2C | 对象同步0x21[0x7034e9b] 码241 len=47 |
| 196.6s | S2C | op0E len=9 0e91610307d5cd3b00 ×2 |
| 196.6s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 196.6s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 196.7s | C2S | m440 ? [] |
| 196.7s | C2S | m431 选择目标 [0] |
| 196.8s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 196.8s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 196.8s | S2C | 属性更新0x10[O#0x499fa8] State=walk |
| 196.8s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 196.9s | S2C | op2C len=132 2c47830000000a010000000000e006007b670107 |
| 197.1s | S2C | S2C m169 buff移除 ["buf_ng_jz_006_01", obj(0x7034e9b,0x498afc)] |
| 197.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 197.2s | S2C | 属性更新0x10[O#0x498afc] PhyDamageMul=1.0 |
| 197.2s | S2C | 属性更新0x10[O#0x498afc] StiDamageMul=1.0 |
| 197.2s | S2C | 属性更新0x10[O#0x498afc] JujDamageMul=1.0 |
| 197.2s | S2C | 属性更新0x10[O#0x498afc] NegDamageMul=1.0 |
| 197.2s | S2C | 属性更新0x10[O#0x498afc] MasDamageMul=1.0 |
| 197.2s | S2C | 属性更新0x10[O#0x498afc] UnderWaterModifySpeed=1.0 ×2 |
| 197.3s | S2C | op31(sub1) 0x700ee8b |
| 197.3s | C2S | m15 移动上报 自动 state=4 pos=(957.5,43.9,1039.3) |
| 197.4s | C2S | m15 移动上报 自动 state=7 pos=(957.5,43.9,1039.3) ×2 |
| 197.4s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 197.6s | S2C | 对象同步0x21[0x7034e9b] 码245 len=135 |
| 197.8s | S2C | 记录删行0x12[fight_relation_rec#114 行114 0x7034e9b] |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5=4BFB,7034E9B,498AFC,5,C38EC0949AF,1 |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=35.0 |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=19.0 |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=21.0 |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=40.0 |
| 198.1s | S2C | S2C m169 buff移除 ["buf_ng_jz_006_02", obj(0x7034e9b,0x498afc)] |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] BufferInfo5= |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] MagicParryDamResAdd=25.0 |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] PhyParryDamResAdd=9.0 |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] AIPhyParryAdd=11.0 |
| 198.1s | S2C | 属性更新0x10[O#0x498afc] AIMagicParryAdd=30.0 |
| 198.3s | C2S | m15 移动上报 自动 state=4 pos=(954.3,43.9,1042.7) |
| 198.6s | S2C | 对象同步0x21[0x7034e9b] 码68 len=47 |
| 199.7s | C2S | m15 移动上报 自动 state=7 pos=(954.3,43.9,1042.7) ×2 |
| 199.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 199.9s | S2C | 属性更新0x10[O#0x498afc] LogicState=2 |
| 200.0s | S2C | 对象同步0x21[0x7034e9b] 码73 len=91 |
| 200.4s | S2C | 属性更新0x10[O#0x498afc] HitHPRatio=100 |
| 200.4s | S2C | 属性更新0x10[O#0x498afc] HitHP=28285 |
| 200.4s | S2C | 属性更新0x10[O#0x498afc] HPRatio=100 |
| 200.4s | S2C | 属性更新0x10[O#0x498afc] HP=28285 |
| 200.7s | C2S | m15 移动上报 自动 state=4 pos=(952.5,43.9,1038.6) |
| 200.7s | C2S | m15 移动上报 自动 state=7 pos=(952.5,43.9,1038.6) ×2 |
| 200.7s | S2C | 状态推送簇 m586/645/646/750: S2C m586 状态推送(簇首) [5] | S2C m645 状态推送 [1] | S2C m646 状态推送 [1] | S2C m750 状态推送 [20] |
| 200.8s | S2C | 对象同步0x21[0x7034e9b] 码27 len=135 |
| 201.0s | S2C | op06 len=1 06 |
| 201.1s | C2S | m15 移动上报 自动 state=4 pos=(952.4,43.9,1040.5) |
| 201.2s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 201.2s | C2S | m15 移动上报 自动 state=5 pos=(952.4,43.9,1040.5) ×2 |
| 201.2s | S2C | 对象同步0x21[0x7034e9b] 码25 len=47 |
| 201.3s | S2C | op28(子码47) len=234 |
| 201.8s | S2C | 属性更新0x10[O#0x498afc] FP=0 |
| 202.1s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [952.392, 43.887, 1040.515, -0.064] |
| 202.2s | S2C | 记录删行0x12[cooldown_rec#109 行109 0x7034e9b] ×2 |
| 202.2s | S2C | S2C m1 系统提示 [2, "9086"] |
| 202.3s | S2C | op28(子码47) len=255 |
| 202.9s | S2C | 属性更新0x10[O#0x498afc] SP=0 |
| 202.9s | C2S | 对象请求 len=33 |
| 202.9s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 202.9s | C2S | m431 选择目标 [obj(0x7034ed0,0x499ea5)] |
| 202.9s | C2S | 对象请求 len=33 |
| 202.9s | C2S | m431 选择目标 [obj(0x7034ed0,0x499ea5)] |
| 203.0s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 204.4s | S2C | 记录删行0x12[Time_Limit_Form_Rec#362 行362 0x0] |
| 204.4s | S2C | 记录加行0x11[Time_Limit_Form_Rec#362 0x0/0x0] +0行 |
| 204.4s | S2C | S2C m250 ? [] ×2 |
| 204.4s | C2S | m211 使用技能(出招) CS_jh_hyjf07 10值 [952.392, 43.887, 1040.515, -0.064] |
| 204.5s | S2C | 位置0x1F(短len=25) |
| 204.5s | S2C | S2C m337 施法回显[0x498afc07034e9b] [952.392, 43.886, 1040.515, 3.683] |
| 204.5s | S2C | 属性更新0x10[O#0x498afc] CurSkillID=CS_jh_hyjf07 |
| 204.5s | S2C | S2C m24 通用事件[0x498afc07034e9b] 冲锋动画 [949.205, 44.886, 1035.214, 1, "qg_fastrun_2h_01"] |
| 204.5s | S2C | 属性更新0x10[O#0x498afc] CantMove=1 |
| 204.5s | S2C | 属性更新0x10[O#0x498afc] CantParry=1 |
| 204.5s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +97行 |
| 204.5s | S2C | 记录写格0x13[cooldown_rec#109 行97 列1 =24832 0x7034e9b] ×4 |
| 204.5s | S2C | 记录写格0x13[cooldown_rec#109 行3 列1 =16777984 0x7034e9b] ×2 |
| 204.5s | S2C | 记录加行0x11[cooldown_rec#109 0x7034e9b/0x498afc] +98行 |
| 204.5s | S2C | 记录写格0x13[cooldown_rec#109 行98 列1 =25088 0x7034e9b] ×4 |
| 204.5s | S2C | 属性更新0x10[O#0x498afc] MPRatio=99 |
| 204.5s | S2C | 属性更新0x10[O#0x498afc] MP=10298 |
| 204.5s | S2C | S2C m707 受理族变体 [15] |
| 204.7s | S2C | 对象同步0x21[0x7034e9b] 码29 len=47 |
| 204.9s | C2S | m212 冲锋/续招落地 [949.397, 43.886, 1035.533, -2.6] |
| 204.9s | S2C | 属性更新0x10[O#0x498afc] CantMove=0 |
| 204.9s | S2C | 属性更新0x10[O#0x498afc] CantParry=0 |
| 204.9s | S2C | 位置0x1F(短len=25) |
| 204.9s | S2C | S2C m26 技能释放 [] |
| 204.9s | S2C | 位置0x1F(短len=25) |
| 204.9s | S2C | S2C m337 施法回显[0x498afc07034e9b] [949.397, 43.886, 1035.533, 3.683] |
| 204.9s | S2C | 位置0x1F(短len=25) |
| 204.9s | S2C | S2C m337 施法回显[0x498afc07034e9b] [949.397, 43.886, 1035.533, 3.364, 0] |
| 204.9s | S2C | S2C m20 范围施放标记[0x498afc07034e9b] ["CS_jh_hyjf07", 0, 0, 0.0, 948.432, 43.886] |
| 204.9s | C2S | m15 移动上报 自动 state=4 pos=(949.2,43.9,1035.2) ×2 |
| 205.1s | S2C | 对象同步0x21[0x7034e9b] 码27 len=91 |
| 205.3s | S2C | op31(sub1) 0x700d19a |
| 205.4s | S2C | 属性更新0x10[V#0x28] CurTargetCount=0 |
| 205.4s | S2C | 属性更新0x10[O#0x498afc] CurSkillID= |
| 206.2s | C2S | m82 ? [3] |
| 206.3s | S2C | op0D len=251 0d65d900073d043b002b776944e8fb4242a15277 |
| 206.3s | S2C | op28(子码47) len=230 ×2 |
| 206.6s | C2S | m54 格挡状态(值5078226820) |
| 206.6s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 206.6s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 206.6s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 206.6s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 206.6s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 206.6s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 206.6s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 207.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 207.2s | C2S | m15 移动上报 自动 state=5 pos=(949.4,43.9,1035.5) ×2 |
| 207.6s | C2S | m54 格挡状态(值4854186845) |
| 207.7s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 207.7s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 207.7s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 207.7s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 207.7s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 207.7s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 207.7s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 207.7s | C2S | m54 格挡状态(值4854186845) ×2 |
| 208.6s | C2S | m54 格挡状态(值5078226820) |
| 208.6s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 208.6s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 208.6s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 208.6s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 208.6s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 208.6s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 208.6s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 208.7s | C2S | m54 格挡状态(值4854186845) |
| 208.8s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 208.8s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 208.8s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 208.8s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 208.8s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 208.8s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 208.8s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 208.8s | C2S | m54 格挡状态(值4854186845) ×3 |
| 209.1s | S2C | 属性更新0x10[O#0x498afc] MPRatio=100 |
| 209.1s | S2C | 属性更新0x10[O#0x498afc] MP=10388 |
| 209.2s | C2S | m54 格挡状态(值5078226820) |
| 209.2s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=1.77 |
| 209.2s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=-2.13 |
| 209.2s | S2C | 属性更新0x10[O#0x498afc] InParry=1 |
| 209.2s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=-93 |
| 209.2s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4=11CE,7034E9B,498AFC,1,0,0 |
| 209.2s | S2C | S2C m169 buff添加 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 209.2s | S2C | 属性更新0x10[O#0x498afc] InManualParry=1 |
| 209.3s | C2S | m15 移动上报 手动 state=4 pos=(949.4,43.9,1035.5) |
| 209.3s | C2S | m54 格挡状态(值4854186845) |
| 209.4s | S2C | S2C m169 buff移除 ["BuffInParry", obj(0x7034e9b,0x498afc)] |
| 209.4s | S2C | 属性更新0x10[O#0x498afc] BufferInfo4= |
| 209.4s | S2C | 属性更新0x10[O#0x498afc] InParry=0 |
| 209.4s | S2C | 属性更新0x10[O#0x498afc] MoveSpeed=4.5 |
| 209.4s | S2C | 属性更新0x10[O#0x498afc] RunSpeedAdd=0.6 |
| 209.4s | S2C | 属性更新0x10[O#0x498afc] ParryUpSpeedAdd=10 |
| 209.4s | S2C | 属性更新0x10[O#0x498afc] InManualParry=0 |
| 209.4s | C2S | m54 格挡状态(值4854186845) ×3 |
| 209.5s | S2C | op2C len=147 2c47920000000a010000000000e0060041010007 |
| 209.8s | C2S | m15 移动上报 自动 state=4 pos=(949.4,43.9,1035.5) |
| 210.0s | S2C | 对象同步0x21[0x7034e9b] 码107 len=47 |
| 210.9s | C2S | m440 ? [] |
| 210.9s | S2C | 属性更新0x10[O#0x498afc] LastObject=None |
| 210.9s | C2S | m431 选择目标 [0] |
| 211.0s | S2C | 属性更新0x10[O#0x498afc] AutoSelectObject=None |
| 212.7–213.9s | S2C | 对象同步0x21[0x701677b] 码217 len=91 ×2 |
| 213.1s | C2S | m9 登录握手 ["630a&&po@&a$#%+b*^@.", ";s*yj>[lkc$4#f[df-!tla&^mj[zsds+ds@lk~]", "11005", "861672DDE4634DA7BDF977AF1870F7F9"] |
| 213.2s | C2S | m15 移动上报 自动 state=5 pos=(949.4,43.9,1035.5) ×2 |
| 213.4s | S2C | 对象同步0x21[0x70295df] 码52 len=91 |
| 213.4s | S2C | 属性更新0x10[O#0x499fad] State=stand |
| 213.9–214.9s | S2C | 属性更新0x10[O#0x499fa8] State=stand ×2 |
| 214.9s | S2C | 属性更新0x10[O#0x499fad] State=walk |
| 214.9s | S2C | op2C len=139 2c478a0000000a010000000000e006007b670107 |
| 1794143936.6s | C2S | 6001 大厅心跳 04 00 00 00 10 EE 11 11 → 01 FF 00 00 ×22 |
