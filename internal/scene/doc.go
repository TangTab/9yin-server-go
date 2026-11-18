// Package scene 实现场景生命周期管理，对应原版 scenes.xml 体系。
//
// 原版语义（scenes.xml 实测字段）：
//   - ID / Name / Script（场景脚本类）/ Config（ini\scene\xxx 配置）
//   - Clonable="1" → 副本场景（CloneScene 脚本），每队/每人独立实例
//   - Resource / CollidePath（资源与碰撞根路径，指向 ..\Res\map）
//   - Born / ReliveLocation（出生点与复活点集合）
//   - Grouping / IsNight / Member
//
// 本包职责：场景注册表加载、主城场景单实例、副本场景克隆与回收、
// 出生/复活点选择。接口从第一天就预留 Clonable 语义（后续玩法依赖）。
package scene
