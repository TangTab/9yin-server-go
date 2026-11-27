package fssoloworld

// currency.go 货币三币的帧面（2433 线 currenciesUpdate 口径：每币一帧
// 0x10 单属性 Int64，按名寻址）：
//
//	碎银 CapitalType1（Money；商店购销/任务奖励，moneyFrame 同源）
//	官银 CapitalType2（OfficialSilver；任务奖励 prize_base[1]）
//	黄金 CapitalType0（Gold；235 潜能赎回的支付币）
//
// 登录整批在 entergen/playerdata.go 携带三币；本文件是运行时增量帧。

import (
	"github.com/lizhanyin/9yin-server-go/internal/entergen"
)

// goldFrame 黄金余额帧（CapitalType0）。
func goldFrame(pl *Player) ([]byte, error) {
	return entergen.ObjectPropertyFrame(1, 1, 0, []entergen.IndexedProperty{
		entergen.NamedProperty("CapitalType0", entergen.Int64Value(pl.roleState.Gold)),
	})
}

// officialSilverFrame 官银余额帧（CapitalType2）。
func officialSilverFrame(pl *Player) ([]byte, error) {
	return entergen.ObjectPropertyFrame(1, 1, 0, []entergen.IndexedProperty{
		entergen.NamedProperty("CapitalType2", entergen.Int64Value(pl.roleState.OfficialSilver)),
	})
}

// pushCurrencies 三币余额帧整推（GM 设置后热发；碎银 → 官银 → 黄金）。
func (s *Server) pushCurrencies(pl *Player) {
	for _, frame := range [][]byte{mustCurrencyFrame(moneyFrame(pl)),
		mustCurrencyFrame(officialSilverFrame(pl)), mustCurrencyFrame(goldFrame(pl))} {
		s.sendToClient(pl, frame)
	}
}

// mustCurrencyFrame 编码失败静默（货币静态名，正常不可达）。
func mustCurrencyFrame(frame []byte, err error) []byte {
	if err != nil {
		return nil
	}
	return frame
}
