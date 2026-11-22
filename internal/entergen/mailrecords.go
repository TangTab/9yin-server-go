package entergen

// mailrecords.go 邮箱记录行（2433 线 cmd/protocol-probe/mail_records.go
// 移植，Java 线 MailFlowTest 真机验证方言）：
//
//   - 邮箱不是专用消息，是记录表 0x11 行：RecvLetterRec（官方 447 schema
//     内嵌表按名取下标，2433 实测 319），16 列——col0 发件人(宽)/1 senderUID
//     /2 类型/3 标题(宽)/4 正文(宽)/5 金(数字窄串)/6 银/7 附件XML/8 时间
//     unix 秒 i64/9 已读/10 序列号/11 交易金/12 勾选/13 剩余时间 i64/
//     14 交易完成/15 f64——列型由 RecordAddCellsFrame 按官方 schema 自动
//     编码（宽串=UTF-16LE）。
//   - 行变更方言 = 0x12 删行 + 0x11 尾插（勿用 0x13 单格帧——快捷栏整表
//     重建同款教训）；服务端列表与客户端行序 1:1（读信/领附件移尾保持）。
//   - 领附件帧序钉死：入包 0x19+0x18 → 0x12 删行 → 0x11 重推
//     （MailFlowTest:144-148 真机序）。
//   - 推送窗口：登录 ClientReady 后、切图 0x0C 重入簇内整表重放（0x0C
//     清玩家对象上挂的一切记录）、GM 在线直推单行。

import (
	"strconv"

	"github.com/lizhanyin/9yin-server-go/internal/clientdata"
	"github.com/lizhanyin/9yin-server-go/internal/rolestate"
)

// RecordRecvLetter RecvLetterRec 官方记录表下标（2433 实测 319；导出供
// fssoloworld 侧帧序断言同源）。
var RecordRecvLetter = clientdata.MustRecordIndexByName("RecvLetterRec")

// LetterSystemPost 系统邮件类型（form_mail.lua LETTER_SYSTEM_POST_USER=101；
// 101..199 客户端归系统页签，发件人空串显示「系统邮件」）。
const LetterSystemPost int32 = 101

// LetterRowFrame 单封信 → 0x11 行帧（16 列按官方 schema 写满）。
func LetterRowFrame(objectID, ownerID uint32, letter rolestate.Letter) ([]byte, error) {
	return RecordAddCellsFrame(objectID, ownerID, RecordRecvLetter, []RecordCell{
		RecordString(letter.Sender),    // 0 发件人（宽串列）
		RecordString(letter.SenderUID), // 1
		RecordInt(uint32(letter.Type)), // 2
		RecordString(letter.Title),     // 3 标题（宽串列）
		RecordString(letter.Content),   // 4 正文（宽串列）
		RecordString(goldCell(letter.Gold)),
		RecordString(goldCell(letter.Silver)),
		RecordString(letter.Appendix), // 7 附件 XML（窄串列）
		RecordInt64(letter.DateUnix),  // 8 unix 秒
		readCell(letter.Read),         // 9
		RecordString(letter.Serial),   // 10 序列号
		RecordString("0"),             // 11 交易金（官方恒 "0"）
		RecordInt(0),                  // 12 勾选
		RecordInt64(0),                // 13 剩余时间
		RecordInt(0),                  // 14 交易完成
		RecordDouble(0),               // 15
	})
}

// LetterRowFrames 整表重放（行序=下发序=客户端行序；空表零帧）。
func LetterRowFrames(objectID, ownerID uint32, letters []rolestate.Letter) ([][]byte, error) {
	if len(letters) == 0 {
		return nil, nil
	}
	frames := make([][]byte, 0, len(letters))
	for i := range letters {
		frame, err := LetterRowFrame(objectID, ownerID, letters[i])
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// LetterDelRowFrame 0x12 删行（row=记录内 0 基行号=Letters 切片下标）。
func LetterDelRowFrame(objectID, ownerID uint32, row int) []byte {
	return RecordDelRowFrame(objectID, ownerID, RecordRecvLetter, uint16(row))
}

// AppendixXML 单物件附件 XML（客户端 form_mail_read.lua ParseXmlData 口径：
// root 首子 Object@Config、其首子 Prop@Amount/ColorLevel；Java
// letterAppendixXml 同源）。
func AppendixXML(configID string, amount int) string {
	return `<Root><Object Config="` + configID + `"><Prop Amount="` +
		strconv.Itoa(amount) + `" ColorLevel="0"/></Object></Root>`
}

// goldCell 金/银数字窄串（官方方言；空值回落 "0"）。
func goldCell(v string) string {
	if v == "" {
		return "0"
	}
	return v
}

func readCell(read bool) RecordCell {
	v := uint32(0)
	if read {
		v = 1
	}
	return RecordCell{Int: &v}
}
