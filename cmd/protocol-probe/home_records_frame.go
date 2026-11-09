package main

import (
	_ "embed"
	"log"
	"os"
	"strings"
)

//go:embed self_home_rec_frame.bin
var officialHomeRecordsFrame []byte

// homeRecordsMirrorEnabled 控制 S2C 0x0A 府邸记录帧的镜像下发（默认开，
// NINEYIN_HOME_RECORDS_MIRROR=0 关闭）。
//
// 背景（2026-11-10 官服客户端排查）：官服每次会话下发 1 帧 S2C 0x0A
// （10429B，self_home_rec/neighbour_rec/HomeFurnitureRec 府邸记录批量），
// 客户端取其帧头 16B 作为后续所有 C2S 0x0A 活动消息的信封头（回显
// 标签——登陆/c.pcapng 两条官服线实证：信封头与该帧 [1:17] 逐字节
// 一致）。本服此前从不发 S2C 0x0A → 客户端信封头全零。若客户端引擎
// 按信封头路由活动消息，全零可能落入异常分支。本实现逐字节镜像官服
// 帧（在官服客户端上验证过的原始字节，零结构风险），下发时机=进场
// （sendEntryScene 之后）。
func homeRecordsMirrorEnabled() bool {
	return strings.TrimSpace(os.Getenv("NINEYIN_HOME_RECORDS_MIRROR")) != "0"
}

// writeOfficialHomeRecordsFrame 下发官服镜像的 S2C 0x0A 记录帧。
func writeOfficialHomeRecordsFrame(conn sceneMessageConnection, remote string) {
	if !homeRecordsMirrorEnabled() || len(officialHomeRecordsFrame) == 0 {
		return
	}
	if err := conn.WriteFrame(officialHomeRecordsFrame); err != nil {
		log.Printf("%s: home records mirror frame: %v", remote, err)
		return
	}
	log.Printf("%s: home records mirror frame sent (%d bytes)", remote, len(officialHomeRecordsFrame))
}
