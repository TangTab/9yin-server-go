// Package prop 实现属性运行时：Record 表格与行操作日志，
// 对应原版引擎的 Record 体系（RolePropRec / DirectPropertyRec 等）。
//
// 原版语义（从明文 lua 脚本与 struct XML 还原）：
//   - Record 是对象上的二维表，行级增删改
//   - 行操作要同步给客户端（增行/改行协议），不能整表重发
//   - nx_add_record_row_values / DirectPropertyRec 是 buff 属性包的载体
//
// 本包职责：Record 存储结构、行操作产生 RowOp 日志（供 sync 转协议）、
// 属性包（buff 加成等 Add 系字段的聚合计算）。
package prop
