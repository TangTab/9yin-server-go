// Package script 是 Lua 脚本层（gopher-lua），对应原版引擎的 lua 绑定，
// 承载行为逻辑（技能/buff/规则脚本）与伤害公式引擎。
//
// 脚本语料 = 官方服务端 Lua（数据根 lua/，testdata/lua 同构）：技能脚本
// 钩子 skill_prepare / skill_after_prepare / skill_before_hit / skill_after_hit
// / skill_finish / skill_break / get_damage（5 系伤害）/ adjust_*，buff 钩子
// buffer_active / buffer_deactive / buffer_timer / buffer_break，规则脚本
// Rule/skill_new.lua（相性比较）。契约见官方 SKILL常用函数.txt。
//
// 句柄语义（与原版的关键差异点）：
//   - 原版脚本持有活对象引用，nx_exists(obj) 直接判活
//   - 本实现一律传 Handle 句柄，判活 = 经 World 查宿主
//   - nx_query_property → Object.Query；战斗动作（nx_damage / nx_add_buffer…）
//     → World，由宿主（internal/fssoloworld 的战斗适配层）落帧
//
// 函数环境：每份钩子脚本（经 Function / RunFileSource 装载的技能、buff、
// 规则脚本）有自己的一张环境表，__index 指向全局表——读得到 nx_* 与内置
// 引擎，写只落在自己环境里；环境里的 require / dofile / loadfile 把模块
// 装进本环境。对应官方宿主「每份脚本一个独立 Lua 上下文」：脚本都把
// get_damage / skill_before_hit 定义在文件作用域，同名模块（AddBuff）各
// 有各的实现，共用全局会互相覆盖。全局环境只放 nx_*、内置引擎与经
// RunFile 装的模块。
//
// 错误隔离：运行期出错只禁用出错的那份脚本（引擎出错禁用引擎），调用方
// 按回退面结算——服务器永不因脚本层停摆。
package script
