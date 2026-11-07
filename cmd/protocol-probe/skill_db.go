package main

import (
	"context"
	"database/sql"

	"github.com/local/9yin-go-server/internal/role"
)

// 技能域只读 DB 访问，移植 Java RoleInventoryStore.skills（读路径）：
// role_skills JOIN game_skills。与物品仓库同口径：失败由调用方回退
// starter 集（不阻塞登录）。本服学技能（msg31 秘籍）链路未实现，
// 因此只读不写；role_skills 行由 Java 参考服运行时 INSERT。
type skillDBStore struct {
	db *sql.DB
}

func newSkillDBStore(db *sql.DB) *skillDBStore {
	return &skillDBStore{db: db}
}

// learnedSkills 读取角色已学技能（View 40 行来源）。SQL 与 Java
// RoleInventoryStore.skills 逐字对齐：wuxue/anqi 进技能容器；
// qinggong 走 View 46、neigong 走 role_faculty 快照，均不在本查询。
func (s *skillDBStore) learnedSkills(ctx context.Context, roleID role.RoleID) ([]learnedSkillView, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT s.skill_id, s.skill_type, g.static_data, g.item_type "+
			"FROM role_skills s JOIN game_skills g ON g.skill_id = s.skill_id "+
			"WHERE s.role_id = ? AND s.skill_type IN ('wuxue', 'anqi') ORDER BY s.skill_id",
		roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	skills := make([]learnedSkillView, 0, 16)
	for rows.Next() {
		var view learnedSkillView
		var skillType string
		if err := rows.Scan(&view.configID, &skillType, &view.staticData, &view.itemType); err != nil {
			return nil, err
		}
		skills = append(skills, dbSkillViewRow(view.configID, view.staticData, view.itemType))
	}
	return skills, rows.Err()
}

// dbSkillViewRow 把 DB 行补齐为 View 40 显示行。等级链路（Java 侧的
// 武学等级.json / role_skill_progress）本服尚未接入，因此 Level=1、
// Fill=Total=0、MaxLevel=Level（1/1 自洽显示）。刻意不从 varprop 区间
// 推导 MaxLevel：花神七式 01 的 varprop 行到 19 级而已安装客户端实证
// 上限是 11，varprop 最大值会高估；Java dbSkillViewAdd 对 DB 行同样
// 只发 MaxLevel=Level。PauseTime 由动作时间线推导（与 starter 目录
// 同源），缺目录回退 0。
func dbSkillViewRow(configID string, staticData, itemType int32) learnedSkillView {
	if itemType == 0 {
		itemType = 1000
	}
	view := learnedSkillView{configID: configID, staticData: staticData, itemType: itemType, level: 1, maxLevel: 1}
	if catalog := installedCombatSkillCatalog; catalog != nil {
		if definition, ok := catalog.definition(configID, 1); ok {
			view.pauseTime = float32(definition.totalActionDuration().Milliseconds())
		}
	}
	return view
}
