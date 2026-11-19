// adapt.go：fxcore.Core（FxCore.dll 复刻）→ SoloCore 适配面。
// fxcore.Core 已满足 fsloader.Core（Type/Version/Startup/Process/
// Shutdown）并具备 WantQuit（vtbl+0x40）——本适配补 MainEntity 槽
//（vtbl+0x78）的接口视图：*Entity → MainEntityView（帧面 vtbl+0xc），
// 并消解 typed-nil（nil *Entity 不得成为非 nil 接口值）。
package fssoloader

import (
	"github.com/lizhanyin/9yin-server-go/internal/fxcore"
)

// FxCoreAdapter 把 fxcore.Core 适配为 SoloCore（官方 FsSoloLoader
// 与 FxCore.dll 的装载契约）。
type FxCoreAdapter struct {
	*fxcore.Core
}

// FxCoreSoloCore 快捷构造。
func FxCoreSoloCore(c *fxcore.Core) SoloCore { return FxCoreAdapter{c} }

// MainEntity 实现 SoloCore：官方 vtbl[+0x78]() → core+0x314。
// 主实体未创建（nil）返回 nil 接口值——壳层 fatal "main entity error"。
func (a FxCoreAdapter) MainEntity() MainEntityView {
	if a.Core == nil || a.Core.Main == nil {
		return nil
	}
	return mainEntityView{ent: a.Core.Main}
}

// mainEntityView 主实体帧面（官方 IEntity vtbl+0xc，dt float 参）。
type mainEntityView struct {
	ent *fxcore.Entity
}

// Frame 实现 MainEntityView：装载器泵循环逐拍 dt=0（fldz 字面）。
func (v mainEntityView) Frame(dt float32) { v.ent.Frame(dt) }
