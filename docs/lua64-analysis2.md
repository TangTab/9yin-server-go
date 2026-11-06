分析完成。产出两个文档 + 一个可重复运行的提取工具：


docs/lua64-analysis.md
 — 总分析报告

docs/lua64-protocol-dict.md
 — 完整协议字典（710 个消息 ID + 665 个下行 handler + 184 个引擎直发 ID 清单）
tools/lua64_extract.py — 源目录更新后可重新生成字典
这批资源里最值钱的东西
1. 协议架构定性（影响所有实现）

下行 custom 消息是“命令名寻址”：引擎按消息名直接调用 custom_handler.lua 同名函数，形参就是载荷布局。Go 侧需要的是“名字→编码器”注册表，不是数字分发
上行参数类型显式定型：shopid/configid/serialno 一律 string、金额一律 int64（单位=文）、槽位从 1 起
184 个 ID 客户端 Lua 层不发送（含 211 出招/218 格挡/914 落点/924 心跳等战斗主链）——这些字节格式仍靠抓包，但 lua64 给了全部语义
2. 印证已实现功能（都对上了，还补了细节）

CD 分段实锤：gui.lua:81-91 — 50000=GCD 基值，50001-51000=CoolDownTeam 组 CD，其中 50001-50100 无特效；CD 数据走引擎 cooldown_rec 不走 custom
背包扩展页：主视口 BaseCap 属性 + 小包条目（ItemType=31）BeginPos/EndPos 锚点区间，与已实现的 19-90 锚点完全吻合；小包有耐久（msg 38 修理）
踢人 error_code 用 20116/20122/51004 才能阻止客户端自动重连（否则自动重连 3 次）；重连续传 = 服务端发 custorm_player_oncontinue(scene) 等客户端回 979
3. 战斗二期关键情报

伤害数字/闪避/格挡字样不在 custom 通道（665 个 handler 里零命中）——短期用属性 HP + custom_trigger_skill_effect + custom_specific_sysinfo 飘字替代
buff 下行布局拿到：on_buffer_change(type,buffer_id,owner,sender,owner_name,sender_name,end_time,category)
伤害公式不在 lua64，在 ini.package 的 share\Skill\formula.ini + damage_calculate.ini —— 且客户端 tooltip 里有完整求值器（tips_func_skill.lua:309-487）可直接移植。解包工具 jiuyin-unpack.exe + 密钥就在同一目录，建议下一步就解 ini.package
4. 新解锁的低成本功能链（参数全在字典里） 交易（View 30/31 双视口+服务端锁定仲裁）、掉落箱（View 80，含任务掉落过滤）、商店回购（View 64，单参数）、邮寄（RecvLetterRec 15 列记录表）、摆摊全套（650-672）

一个小插曲：某个探查 agent 中途误报“源目录被移除”，我核实过 —— 目录完好，2296 个文件都在。