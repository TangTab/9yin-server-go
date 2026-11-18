// Package config 实现配置加载，对应原版的资源装载体系：
// Object/Property XML（ini 数值表）、struct XML（对象属性定义）、
// scenes.xml / logic_class.xml / modules.xml（注册表）。
//
// 原版格式特征（实测样本见 testdata/）：
//   - XML 声明 encoding=gb2312，中文字段需 GBK→UTF-8 转码
//   - 顶层 <Object>，行级 <Property ID=".." Name=".." .../>
//   - 引用链：Skill_Config.xml 按 Type/Path 指向 Skill_Main/Skill_Display/
//     Skill_Effect/Buff_* 分层文件
//   - 原版支持热重载（Reload=1）；本实现预留 watcher，不在首期做
package config
