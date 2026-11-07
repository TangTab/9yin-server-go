package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// 称号系统（fork title_system.go 复刻；docs/fork-p2-shop-title-jingmai.md）。
//
// handleSetTitleCustom（msg295）在 fork 里极简：len>=2 → values[1].Int32
// → playerActor.setTitleID（互斥锁直写，无所有权校验、不授予）→
// titleIDUpdateFrame（0x10 TitleID@360）→ 写帧（title_system.go:189-202
// 反汇编实证）。佩戴持久化 role_titles（Java PanelDataStore 同表形）；
// 拥有集合经 title_rec 记录行（官方 schema 5 列 [u32,u32,f64,u8,f64]，
// 数值列语义未校准保持零填充——FashionSuitRec 同例）进世界重放。
const (
	clientCustomSetTitle int32  = 295
	propTitleID          uint16 = 360
	titleIDMax           int32  = 500000
	titleDBQueryTimeout         = 5 * time.Second
)

// titleRow 是 role_titles 的行形态（Java PanelDataStore：role_id+title_id
// PK、title_type、is_worn、obtained_unix）。
type titleRow struct {
	titleID   int32
	titleType int32
	worn      bool
}

// titleDBStore 持久化称号拥有+佩戴位。缺表/缺库时 Load 返回空、佩戴位
// 只留会话（建表权威在 migrations/0014，运行时不做 DDL）。
type titleDBStore struct {
	db *sql.DB
}

func newTitleDBStore(db *sql.DB) *titleDBStore {
	return &titleDBStore{db: db}
}

// Load 读取角色称号集合；无行/缺表/坏数据按“无称号”处理。
func (store *titleDBStore) Load(roleID role.RoleID) []titleRow {
	if store == nil || store.db == nil || roleID == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), titleDBQueryTimeout)
	defer cancel()
	rows, err := store.db.QueryContext(ctx,
		`SELECT title_id, title_type, is_worn FROM role_titles WHERE role_id = ? ORDER BY title_id`, roleID)
	if err != nil {
		log.Printf("title store: load role=%d: %v", roleID, err)
		return nil
	}
	defer rows.Close()
	var titles []titleRow
	for rows.Next() {
		var titleID, titleType int32
		var worn int32
		if err := rows.Scan(&titleID, &titleType, &worn); err != nil {
			log.Printf("title store: scan role=%d: %v", roleID, err)
			return nil
		}
		titles = append(titles, titleRow{titleID: titleID, titleType: titleType, worn: worn != 0})
	}
	if err := rows.Err(); err != nil {
		log.Printf("title store: iterate role=%d: %v", roleID, err)
		return nil
	}
	return titles
}

// wear 单佩戴位写库：先清同角色全部 is_worn，再 upsert 目标行（未拥有
// 即补行——与 fork 无所有权校验的宽松语义配套）；titleID=0 仅清位。
func (store *titleDBStore) wear(roleID role.RoleID, titleID int32) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("title store: no database")
	}
	if roleID == 0 {
		return errors.New("title store: zero role id")
	}
	ctx, cancel := context.WithTimeout(context.Background(), titleDBQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx,
		`UPDATE role_titles SET is_worn = 0 WHERE role_id = ?`, roleID); err != nil {
		return fmt.Errorf("clear worn titles role=%d: %w", roleID, err)
	}
	if titleID == 0 {
		return nil
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO role_titles (role_id, title_id, title_type, is_worn, obtained_unix)
VALUES (?, ?, 0, 1, UNIX_TIMESTAMP())
ON DUPLICATE KEY UPDATE is_worn = 1, title_type = title_type`,
		roleID, titleID); err != nil {
		return fmt.Errorf("wear title role=%d id=%d: %w", roleID, titleID, err)
	}
	return nil
}

// ---- playerActor 称号域 ----

func (p *playerActor) setTitleIDValue(value int32) {
	p.mu.Lock()
	p.titleID = value
	p.mu.Unlock()
}

func (p *playerActor) titleIDValue() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.titleID
}

// restoreTitles 登录装载：拥有集合 + 佩戴位（is_worn 行唯一时生效）。
func (p *playerActor) restoreTitles(titles []titleRow) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ownedTitles = append([]int32(nil), ownedTitleIDs(titles)...)
	for _, row := range titles {
		if row.worn {
			p.titleID = row.titleID
			break
		}
	}
}

func ownedTitleIDs(titles []titleRow) []int32 {
	ids := make([]int32, 0, len(titles))
	for _, row := range titles {
		ids = append(ids, row.titleID)
	}
	return ids
}

func (p *playerActor) ownedTitleSnapshot() []int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]int32(nil), p.ownedTitles...)
}

// titleIDUpdateFrame 佩戴位单属性帧（0x10 TitleID@360 Int32——官方表
// 线型 3；fork MOVW u16 是其自协商表特例，本服走官方表）。
func (p *playerActor) titleIDUpdateFrame() ([]byte, error) {
	return sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
		{Index: propTitleID, Name: "TitleID", Value: clientdata.Int32Value(p.titleIDValue())},
	})
}

// ---- msg295 ----

// handleSetTitleCustom [295, i32 titleId]：0=取消佩戴；范围闸 0..500000；
// 无所有权校验（fork 同口径，客户端可信环境）；佩戴落库失败仅记日志。
func handleSetTitleCustom(link sceneMessageConnection, player *playerActor, custom clientCustomMessage, remote string, titles *titleDBStore, roleID role.RoleID) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomSetTitle {
		return false, nil
	}
	if player == nil {
		log.Printf("%s: reject set title before player spawn", remote)
		return true, nil
	}
	if len(custom.Values) < 2 {
		log.Printf("%s: reject malformed set title values=%v", remote, custom.Values)
		return true, nil
	}
	titleID, ok := custom.Values[1].exactInt32()
	if !ok {
		log.Printf("%s: reject malformed set title values=%v", remote, custom.Values)
		return true, nil
	}
	if titleID < 0 || titleID > titleIDMax {
		log.Printf("%s: reject set title %d: titleId must be within 0-%d", remote, titleID, titleIDMax)
		return true, nil
	}
	player.setTitleIDValue(titleID)
	if titles != nil && roleID != 0 {
		if err := titles.wear(roleID, titleID); err != nil {
			log.Printf("%s: persist title change: %v", remote, err)
		}
	}
	frame, err := player.titleIDUpdateFrame()
	if err != nil {
		return true, fmt.Errorf("encode title update: %w", err)
	}
	if err := link.WriteFrame(frame); err != nil {
		return true, fmt.Errorf("write title update: %w", err)
	}
	log.Printf("%s: set title id=%d", remote, titleID)
	return true, nil
}

// titleRecAddRow 编码单条 title_rec 记录行（官方 schema 3 列
// [u32,u8,i64]：titleId、佩戴标记、到期时间戳 ms——0=永久）。
// 此前自协商方言误声明 5 列 [u32,u32,f64,u8,f64]。
func titleRecAddRow(titleID int32, worn bool) ([]byte, error) {
	wornByte := uint32(0)
	if worn {
		wornByte = 1
	}
	return serverRecordAddCells(playerObjectID, playerOwnerID, recordTitle, []recordCell{
		recordInt(uint32(titleID)),
		recordByte(wornByte),
		recordInt64(0),
	})
}

// pushTitleFrames 进世界重放（ClientReady 窗口）：title_rec 拥有行 +
// 佩戴位 0x10（无佩戴零帧）。挂玩家对象记录视图，0x0C 重入随对象清空
// 由 stable re-entry 分支重推。
func pushTitleFrames(link sceneMessageConnection, player *playerActor) error {
	owned := player.ownedTitleSnapshot()
	worn := player.titleIDValue()
	wornSet := make(map[int32]struct{}, len(owned))
	if worn != 0 {
		wornSet[worn] = struct{}{}
	}
	for _, id := range owned {
		_, isWorn := wornSet[id]
		frame, err := titleRecAddRow(id, isWorn)
		if err != nil {
			return fmt.Errorf("encode title_rec id=%d: %w", id, err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write title_rec id=%d: %w", id, err)
		}
	}
	if worn != 0 {
		frame, err := player.titleIDUpdateFrame()
		if err != nil {
			return fmt.Errorf("encode title replay: %w", err)
		}
		if err := link.WriteFrame(frame); err != nil {
			return fmt.Errorf("write title replay: %w", err)
		}
	}
	if len(owned) > 0 {
		log.Printf("replayed %d title_rec rows (worn=%d)", len(owned), worn)
	}
	return nil
}
