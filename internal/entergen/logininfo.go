package entergen

// 登录信息帧簇（b-line official_login_align.go 移植）：
// 0.1s 位 m366/m623/m655 + 就绪位 m607/m52/m586 + m169 buff/m416/m294。
// 位置：0x0A 记录表后、进场前（m366 族）；ClientReady 后（m607 族）。

import (
	"fmt"
	"time"
)

// 官方消息号（lua64 client_custom_define.lua / 抓包实证）。
const (
	MsgServerTime     = 52  // [1, unix 秒]
	MsgClientIP       = 366 // [客户端 IP]
	MsgServerInfo     = 623 // [0,0,3]
	MsgAccountCard    = 607 // [账号, 角色名]
	MsgServerIdentity = 655 // [sid,sid,服务器名,sid,账号,账号 ID]
	MsgStatePushHead  = 586 // [5]
	MsgBuff           = 169 // [buff 名, obj]
	MsgJyzDomain      = 416 // ["DomainJiuYinZhi", step]
	MsgTimeSync       = 294 // 对钟应答
	MsgServerTimeNow  = 52
)

// fileTimeEpochOffsetMs 1601-01-01 相对 1970-01-01 的毫秒差。
const fileTimeEpochOffsetMs = int64(11644473600000)

// uptimeDays m294 尾值「世界天数」常量（外服样本全场恒定；客户端无活值
// 消费证据——按原样镜像）。
const uptimeDays = 46335.831

// buffGroupFlag101 公共 buff 组标志（BufferGroupPublicRec 同域，永久）。
const buffGroupFlag101 = "buff_group_flag_101"

// LoginInfoFrames 0.1s 位帧簇（表后、进场前）：m366/m623/m655。
func LoginInfoFrames(ctx Context) ([][]byte, error) {
	frames := make([][]byte, 0, 3)
	ipFrame, err := CustomMessageFrame(MsgClientIP, CustomString(ctx.Session.ClientIP))
	if err != nil {
		return nil, err
	}
	frames = append(frames, ipFrame)
	infoFrame, err := CustomMessageFrame(MsgServerInfo, CustomInt(0), CustomInt(0), CustomInt(3))
	if err != nil {
		return nil, err
	}
	frames = append(frames, infoFrame)
	idFrame, err := CustomMessageFrame(MsgServerIdentity,
		CustomInt(ctx.Session.ServerID), CustomInt(ctx.Session.ServerID),
		CustomWideString(ctx.Session.ServerName), CustomInt(ctx.Session.ServerID),
		CustomString(ctx.Session.Account), CustomInt64(ctx.Session.AccountID))
	if err != nil {
		return nil, err
	}
	frames = append(frames, idFrame)
	return frames, nil
}

// ReadyInfoFrames 就绪位帧簇（ClientReady 后首拍）：m607/m52/m586。
func ReadyInfoFrames(ctx Context, now time.Time) ([][]byte, error) {
	frames := make([][]byte, 0, 3)
	cardFrame, err := CustomMessageFrame(MsgAccountCard,
		CustomString(ctx.Session.Account), CustomWideString(ctx.RoleName))
	if err != nil {
		return nil, err
	}
	frames = append(frames, cardFrame)
	timeFrame, err := CustomMessageFrame(MsgServerTime, CustomInt(1), CustomInt64(now.Unix()))
	if err != nil {
		return nil, err
	}
	frames = append(frames, timeFrame)
	headFrame, err := CustomMessageFrame(MsgStatePushHead, CustomInt(5))
	if err != nil {
		return nil, err
	}
	frames = append(frames, headFrame)
	return frames, nil
}

// LoginBuffFrames 登录期 m169 buff：组标志 + 装备内功可见 buff。
func LoginBuffFrames(ctx Context) ([][]byte, error) {
	names := []string{buffGroupFlag101}
	for i := range ctx.State.NeiGong {
		book := &ctx.State.NeiGong[i]
		if book.ConfigID == ctx.State.CurNeiGong && book.BuffID != "" {
			names = append(names, book.BuffID)
			break
		}
	}
	frames := make([][]byte, 0, len(names))
	for _, name := range names {
		frame, err := CustomMessageFrame(MsgBuff,
			CustomString(name), CustomObject(ctx.Identity.ObjectID, ctx.Identity.OwnerID))
		if err != nil {
			return nil, fmt.Errorf("encode login buff %s: %w", name, err)
		}
		frames = append(frames, frame)
	}
	return frames, nil
}

// JyzDomainFrame m416 九阴志域声明。
func JyzDomainFrame(step int32) ([]byte, error) {
	return CustomMessageFrame(MsgJyzDomain, CustomString("DomainJiuYinZhi"), CustomInt(step))
}

// TimeSyncReplyFrame m294 对钟应答：[client_ms 回显, reqid=FILETIME ms,
// unix 秒, uptime 天数常量]。
func TimeSyncReplyFrame(clientMs int64, now time.Time) ([]byte, error) {
	return CustomMessageFrame(MsgTimeSync,
		CustomInt64(clientMs),
		CustomInt64(now.UnixMilli()+fileTimeEpochOffsetMs),
		CustomInt64(now.Unix()),
		CustomFloat64(uptimeDays),
	)
}
