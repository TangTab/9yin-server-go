package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/local/9yin-go-server/internal/role"
)

// 邮件域（Java LetterStore/letterRowFrame 全链对齐）：role_letters.snapshot
// {"letters":[…]}（snake_case 字段）登录整表重放为 RecvLetterRec 319 的
// 16 列行（官方列型 [8,7,3,8,8,7,7,7,4,3,7,7,3,4,3,6]：col0/3/4 宽串）。
// 空邮箱零帧（Java lettersGranted 同口径）。重登/0x0C 重入按内存镜像
// 整表重放；已读/删信等 C2S 变更路径本服暂缺，重放只读。
const (
	recordRecvLetter uint16 = 319 // 官方 447 schema：RecvLetterRec

	letterDBQueryTimeout = 5 * time.Second
)

// letterEntry 单封信（RecvLetterRec 16 列中服务端关心的 11 列；其余恒
// 0/空串——Java Letter 同构）。gold/silver 官方线型是数字串（"0"）。
type letterEntry struct {
	Serial          string `json:"serial"`
	Type            int32  `json:"type"`
	Sender          string `json:"sender"`
	SenderUID       string `json:"sender_uid"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	Gold            string `json:"gold"`
	Silver          string `json:"silver"`
	AppendixXML     string `json:"appendix_xml"`
	DateUnixSeconds int64  `json:"date_unix_seconds"`
	ReadFlag        bool   `json:"read_flag"`
}

type letterDBStore struct {
	db *sql.DB
}

func newLetterDBStore(db *sql.DB) *letterDBStore {
	return &letterDBStore{db: db}
}

// Load 读 role_letters 快照（Java 写入形态：{"letters":[…]}）；无行/缺表
// /解码失败一律返回空（空邮箱零帧，不阻塞登录）。
func (store *letterDBStore) Load(roleID role.RoleID) []letterEntry {
	if store == nil || store.db == nil || roleID == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), letterDBQueryTimeout)
	defer cancel()
	var raw []byte
	err := store.db.QueryRowContext(ctx,
		`SELECT snapshot FROM role_letters WHERE role_id = ?`, roleID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		log.Printf("letter store: load role=%d: %v", roleID, err)
		return nil
	}
	var snapshot struct {
		Letters []letterEntry `json:"letters"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		log.Printf("letter store: decode role=%d snapshot: %v", roleID, err)
		return nil
	}
	return snapshot.Letters
}

// letterRowFrame RecvLetterRec 整行（Java letterRowFrame 同 16 列）。
func letterRowFrame(letter letterEntry) ([]byte, error) {
	readFlag := uint32(0)
	if letter.ReadFlag {
		readFlag = 1
	}
	return serverRecordAddCells(playerObjectID, playerOwnerID, recordRecvLetter, []recordCell{
		recordString(letter.Sender),         // 0 发件人（宽）
		recordString(letter.SenderUID),      // 1 发件人 uid
		recordInt(uint32(letter.Type)),      // 2 类型
		recordString(letter.Title),          // 3 标题（宽）
		recordString(letter.Content),        // 4 正文（宽）
		recordString(letter.Gold),           // 5 金（数字串）
		recordString(letter.Silver),         // 6 银（数字串）
		recordString(letter.AppendixXML),    // 7 附件 XML
		recordInt64(letter.DateUnixSeconds), // 8 时间（unix 秒）
		recordInt(readFlag),                 // 9 已读
		recordString(letter.Serial),         // 10 序列号
		recordString("0"),                   // 11 交易金（数字串）
		recordInt(0),                        // 12 勾选
		recordInt64(0),                      // 13 剩余时间
		recordInt(0),                        // 14 交易完成
		recordFloat64(0),                    // 15 保留 f64
	})
}

// pushLetterFrames 邮件整表重放（行序=下发序，SERVER_RECORD_ADD 追加）。
func pushLetterFrames(link sceneMessageConnection, letters []letterEntry) error {
	for index, letter := range letters {
		frame, err := letterRowFrame(letter)
		if err != nil {
			return fmt.Errorf("encode letter row %d (%s): %w", index, letter.Serial, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write letter row %d: %w", index, err)
		}
	}
	return nil
}
