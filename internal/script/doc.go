// Package script 是 Lua 脚本层（gopher-lua），对应原版引擎的 lua 绑定，
// 承载行为逻辑（技能/buff/任务/副本脚本），复用原版 Res/lua 的脚本范式。
//
// 句柄语义（与原版的关键差异点）：
//   - 原版脚本持有活对象引用，nx_exists(obj) 直接判活
//   - 本实现一律传 EntityID（含 generation）句柄，判活 = 查询 World
//   - nx_query_property → GetComponent(e).Field 的类型化访问
//
// 实现顺序：本包在登录/场景/AOI/同步主干跑通之后才动工（最后做）。
// 目标是让 jn_normal（复活/中毒/AOE）级别的原版脚本能近乎直接移植。
package script
