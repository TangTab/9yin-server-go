package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/local/9yin-go-server/internal/itemuse"
	"github.com/local/9yin-go-server/internal/luascript"
	"github.com/local/9yin-go-server/internal/neigong"
	"github.com/local/9yin-go-server/internal/skillscript"
)

// scriptHosts 聚合服务端的 Lua 脚本宿主：事件型（game.on，npc_service
// 等低频交互）与模块式域宿主（scripts/module/ 下自参考服拷贝的
// neigong/item/skill，契约见 scripts/README.md）。各宿主独立 VM、
// 错误互不连坐；模块宿主的公共机制在 internal/modulevm。combat 公式
// 引擎走 internal/combatcalc（combat_formula.go），不在此聚合。
type scriptHosts struct {
	events  *luascript.Host      // scripts/host
	neigong *neigong.Effects     // scripts/module/neigong
	items   *itemuse.Catalog     // scripts/module/item
	skills  *skillscript.Scripts // scripts/module/skill
}

// hostRegistry 是进程级脚本宿主登记册（main 装配后写入；连接会话的
// GM lua_reload 据此热重载全量宿主）。
var hostRegistry *scriptHosts

// moduleHost 是四类模块域宿主的公共面（装载/热重载用）。
type moduleHost interface {
	Enabled() bool
	Load() (int, error)
	Reload() (int, error)
}

// newScriptHosts 按各 flag 构建宿主；空路径即禁用该类。
func newScriptHosts(eventsDir, neigongDir, itemDir, skillDir string) *scriptHosts {
	return &scriptHosts{
		events:  luascript.New(eventsDir, log.Printf),
		neigong: neigong.New(neigongDir, log.Printf),
		items:   itemuse.New(itemDir, log.Printf),
		skills:  skillscript.New(skillDir, log.Printf),
	}
}

// load 启动时装载全部已启用宿主并输出就绪日志；失败仅记录（各宿主的
// 软语义保证坏文件不中断其余），服务照常启动。
func (h *scriptHosts) load() {
	if h.events.Enabled() {
		loaded, err := h.events.LoadDir()
		if err != nil {
			log.Printf("Lua scripts partially loaded count=%d: %v", loaded, err)
		} else {
			log.Printf("Lua scripts ready count=%d", loaded)
		}
	}
	for _, m := range []struct {
		name string
		dir  string
		host moduleHost
	}{
		{"内功特效脚本", "scripts/module/neigong", h.neigong},
		{"使用物脚本", "scripts/module/item", h.items},
		{"技能脚本", "scripts/module/skill", h.skills},
	} {
		if !m.host.Enabled() {
			continue
		}
		loaded, err := m.host.Load()
		if err != nil {
			log.Printf("%s partially loaded count=%d dir=%s: %v", m.name, loaded, m.dir, err)
		} else {
			log.Printf("%s ready count=%d dir=%s", m.name, loaded, m.dir)
		}
	}
}

// reloadAll GM 热重载全部已启用宿主，回报各计数；任一失败返回首个错误
// （其余仍已完成重载）。
func (h *scriptHosts) reloadAll() (string, error) {
	var parts []string
	if h.events.Enabled() {
		loaded, err := h.events.Reload()
		parts = append(parts, fmt.Sprintf("事件=%d", loaded))
		if err != nil {
			return strings.Join(parts, " "), err
		}
	}
	for _, m := range []struct {
		label string
		host  moduleHost
	}{
		{"内功", h.neigong},
		{"使用物", h.items},
		{"技能", h.skills},
	} {
		if !m.host.Enabled() {
			continue
		}
		loaded, err := m.host.Reload()
		parts = append(parts, fmt.Sprintf("%s=%d", m.label, loaded))
		if err != nil {
			return strings.Join(parts, " "), err
		}
	}
	return strings.Join(parts, " "), nil
}
