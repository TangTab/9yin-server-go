package entergen

// 自定义配置域 ClientReady 推送簇（fork 客户端本地不落盘——键位/快捷
// 栏/个性配置全靠服务端回推，缺发即「重登全丢」）。规则移植参考端
// cmd/protocol-probe custom_config.go/shortcut_records.go：
//
//	705 个性配置（0x1E 旧壳，on_parser_customizing 灌回配置对象）——
//	    连接级，须赶在客户端登录期上传 107 默认转储之前（本簇随
//	    ClientReady 立即下行即满足）；
//	键位域 0x10 单属性帧：198 ShortcutKey/199 ShortcutKey1/200
//	    ShortcutKey2（type7，串 = 存档值 + " " 尾空格——两形态不可与
//	    0x0D 档内嵌混用）、223 OperateMode（仅显式设置过）、201
//	    NeiGongBind（内功栏主窗体打开时快照）；
//	快捷栏 shortcut_rec 0x11 逐行（schema [3,7,7]：栏位索引/kind/
//	    config ID）；无编辑基线补 starter 打坐条目（EffectiveShortcuts）。
//
// 无档零帧（客户端保持本地默认并自行上传基线）。键位域挂玩家对象
// 属性视图、快捷栏行挂玩家对象记录——0x0C 切图清玩家对象后由
// PlayerObjectConfigFrames 重推（reentryReady，参考端 stable re-entry
// restore shortcuts/keybind 两步同口径）；705 连接级不随切图重置，无需重发。

import (
	"fmt"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// serverCustomPushCustomizing S2C 705：on_parser_customizing(configStr)
// 个性配置下行通道（C2S 107 上传的回推对应物）。
const serverCustomPushCustomizing int32 = 705

// CustomConfigFrames 自定义配置域推送簇（Ready 段，视图簇之后）=
// 705 连接级个性配置 + PlayerObjectConfigFrames 玩家对象域。
func CustomConfigFrames(ctx Context) ([][]byte, error) {
	frames := [][]byte{}
	state := ctx.State

	// 705 个性配置（空档零帧）。
	if state.Customizing != "" {
		frame, err := CustomMessageFrame(serverCustomPushCustomizing, CustomString(string(state.Customizing)))
		if err != nil {
			return nil, fmt.Errorf("encode customizing frame: %w", err)
		}
		frames = append(frames, frame)
	}

	playerObject, err := PlayerObjectConfigFrames(ctx)
	if err != nil {
		return nil, err
	}
	return append(frames, playerObject...), nil
}

// PlayerObjectConfigFrames 键位域 + 快捷栏行（挂玩家对象——0x0C 切图
// 清对象后重入重推用；705 连接级不在内）。
func PlayerObjectConfigFrames(ctx Context) ([][]byte, error) {
	frames := [][]byte{}
	state := ctx.State

	// 键位域：198 当前启用 / 199 方案一 / 200 方案二（保存面板
	// check_saved 读后两者判定「已保存」）。线上串 = 存档值 + " "。
	schemes := []struct {
		name  string
		value string
	}{
		{"ShortcutKey", state.Keybind},
		{"ShortcutKey1", state.Keybind1},
		{"ShortcutKey2", state.Keybind2},
	}
	for _, scheme := range schemes {
		if scheme.value == "" {
			continue
		}
		frame, err := ObjectPropertyFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, 0,
			[]IndexedProperty{NamedProperty(scheme.name, StringValue(scheme.value+" "))})
		if err != nil {
			return nil, fmt.Errorf("encode keybind %s: %w", scheme.name, err)
		}
		frames = append(frames, frame)
	}

	// 223 操作模式（从未设置不下发——客户端保持本地默认）。
	if state.OperateModeSet {
		frame, err := ObjectPropertyFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, 0,
			[]IndexedProperty{NamedProperty("OperateMode", Int32Value(state.OperateMode))})
		if err != nil {
			return nil, fmt.Errorf("encode operate mode: %w", err)
		}
		frames = append(frames, frame)
	}

	// 201 内功栏绑定串（get_svr_ng_bind 以 QueryProp 读它初始化绑定
	// 框；不回推则两端永不收敛）。
	if state.NeiGongBind != "" {
		frame, err := ObjectPropertyFrame(ctx.Identity.ObjectID, ctx.Identity.OwnerID, 0,
			[]IndexedProperty{NamedProperty("NeiGongBind", StringValue(state.NeiGongBind+" "))})
		if err != nil {
			return nil, fmt.Errorf("encode neigong bind: %w", err)
		}
		frames = append(frames, frame)
	}

	// 快捷栏行（0x11 逐行重放；schema 走 0x0A 全表自足，无需挂载帧）。
	shortcutFrames, err := ShortcutRowFrames(ctx.Identity.ObjectID, ctx.Identity.OwnerID, state.EffectiveShortcuts())
	if err != nil {
		return nil, err
	}
	frames = append(frames, shortcutFrames...)
	return frames, nil
}

// ShortcutRowFrames 快捷栏行集 → 0x11 帧序（编辑应答的重建半边与
// ClientReady 重放共用）。
func ShortcutRowFrames(objectID, ownerID uint32, rows []rolestate.ShortcutRow) ([][]byte, error) {
	if len(rows) == 0 {
		return nil, nil
	}
	record := clientdata.MustRecordIndexByName("shortcut_rec")
	frames := make([][]byte, 0, len(rows))
	for _, row := range rows {
		if row.Index < 0 {
			return nil, fmt.Errorf("shortcut index %d is negative", row.Index)
		}
		frame, err := RecordAddCellsFrame(objectID, ownerID, record, []RecordCell{
			RecordInt(uint32(row.Index)),
			RecordString(row.Kind),
			RecordString(row.ID),
		})
		if err != nil {
			return nil, fmt.Errorf("encode shortcut index %d: %w", row.Index, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// ShortcutRebuildFrames 快捷栏编辑应答的整表重建方言帧序：从尾行到 0
// 倒序 0x12 清空 oldRows 行，再按内存序 0x11 整表重放（倒序删行使
// 行号重编号永不参与——客户端引擎侧无论怎样维护行号都不会错位）。
func ShortcutRebuildFrames(objectID, ownerID uint32, oldRows int, rows []rolestate.ShortcutRow) ([][]byte, error) {
	record := clientdata.MustRecordIndexByName("shortcut_rec")
	frames := make([][]byte, 0, oldRows+len(rows))
	for row := oldRows - 1; row >= 0; row-- {
		frames = append(frames, RecordDelRowFrame(objectID, ownerID, record, uint16(row)))
	}
	add, err := ShortcutRowFrames(objectID, ownerID, rows)
	if err != nil {
		return nil, err
	}
	return append(frames, add...), nil
}
