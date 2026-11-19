// Package fspublic 复刻官方公共数据服务进程（FsPublic.dll + FsPubLogic.dll，
// 宿主 game_public/FsLoader GamePublic.ini）。
//
// 架构（docs/official/FsPublic全解与Go复刻.md 为权威）：
//   - GamePublic 服务实体：三消息队列（world/member/store）+ 五态状态机
//     （Unknown→LoadSpaces→Opened→Closing→Closed）+ 周期广播（kind=4）
//   - CPubKernel/CPubSpace/CPubData：公共数据空间的权威副本
//     （属性 CPropData + 记录表 CRecData），member 侧持有只读镜像
//   - FsPubLogic（Callee 逻辑模块）：Domain_* 玩法域挂载于 pub 空间
//
// 依赖边界：fscomm（Adapter/EchoSock/BroadSock/AppManager/LogicLoader），
// 不依赖 ecs/engine（公共数据服务无场景）。
package fspublic
