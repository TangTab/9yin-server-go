package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/local/9yin-go-server/internal/clientdata"
	"github.com/local/9yin-go-server/internal/role"
)

// 货币域（msg70/71 商店购销引入）：碎银（CapitalType1@417 i64）落
// role_currency 快照；金币（416）/银卡（418）保持会话级（fork 快照仅
// {Silver}，Java applyCapital 同口径不落库）。帧序对齐 fork
// currenciesUpdate：三币各一帧 0x10。
var (
	propCapitalType0 = clientdata.MustPlayerProperty("CapitalType0").Index // CapitalType0（fork=416）
	propCapitalType1 = clientdata.MustPlayerProperty("CapitalType1").Index // CapitalType1（fork=417）
	propCapitalType2 = clientdata.MustPlayerProperty("CapitalType2").Index // CapitalType2（fork=418）
)

const currencyDBQueryTimeout = 5 * time.Second

// currencySnapshot 是 role_currency.snapshot 列的 JSON 形态。字段名与
// fork 二进制（9yin_server1 build 2026-09-14）的 {Silver json:"silver"}
// 序列化兼容——快照超集字段在 Unmarshal 时被忽略，多服务端共存不互蚀。
type currencySnapshot struct {
	Silver int32 `json:"Silver"`
}

// currencyDBStore 把碎银余额持久化到 role_currency（role_id PK /
// snapshot JSON / version / updated_at）。缺表/缺库时 Load 返回无存档、
// Save 记日志跳过（建表权威在 migrations/0014，运行时不做 DDL）。
type currencyDBStore struct {
	db *sql.DB
}

func newCurrencyDBStore(db *sql.DB) *currencyDBStore {
	return &currencyDBStore{db: db}
}

// Load 读取碎银存档；无行、缺表或坏快照一律按“无存档”处理（登录链走
// starter 初值）。
func (store *currencyDBStore) Load(roleID role.RoleID) (currencySnapshot, bool) {
	if store == nil || store.db == nil || roleID == 0 {
		return currencySnapshot{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), currencyDBQueryTimeout)
	defer cancel()
	var raw []byte
	err := store.db.QueryRowContext(ctx,
		`SELECT snapshot FROM role_currency WHERE role_id = ?`, roleID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return currencySnapshot{}, false
	}
	if err != nil {
		log.Printf("currency store: load role=%d: %v", roleID, err)
		return currencySnapshot{}, false
	}
	var snapshot currencySnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		log.Printf("currency store: decode role=%d snapshot: %v", roleID, err)
		return currencySnapshot{}, false
	}
	return snapshot, true
}

// Save upsert 快照（version 自增、updated_at 由库维护，role_faculty 同款）。
func (store *currencyDBStore) Save(roleID role.RoleID, value currencySnapshot) error {
	if store == nil {
		return nil
	}
	if store.db == nil {
		return errors.New("currency store: no database")
	}
	if roleID == 0 {
		return errors.New("currency store: zero role id")
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode currency snapshot: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), currencyDBQueryTimeout)
	defer cancel()
	if _, err := store.db.ExecContext(ctx, `INSERT INTO role_currency (role_id, snapshot, version, updated_at)
VALUES (?, ?, 1, UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE snapshot = VALUES(snapshot), version = version + 1, updated_at = UTC_TIMESTAMP(6)`,
		roleID, raw); err != nil {
		return fmt.Errorf("persist currency role=%d: %w", roleID, err)
	}
	return nil
}

// ---- playerActor 货币助手 ----

// trySpendSilver 余额足够才扣减并返回 true；不足时不动余额（买/卖事务
// 的“失败拒绝”半边，addSilver 只做饱和加法）。
func (p *playerActor) trySpendSilver(cost int32) bool {
	if cost < 0 {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.silver < cost {
		return false
	}
	p.silver -= cost
	return true
}

func (p *playerActor) silverCardBalance() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.silverCard
}

func (p *playerActor) silverBalance() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.silver
}

// setSilver 登录装载存档碎银（覆盖 starter 初值；setGold 同构）。
func (p *playerActor) setSilver(value int32) {
	p.mu.Lock()
	p.silver = value
	p.mu.Unlock()
}

func (p *playerActor) goldBalance() int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.gold
}

// addSilverCard 银卡饱和加法（addSilver 同构）。
func (p *playerActor) addSilverCard(amount int32) int32 {
	p.mu.Lock()
	defer p.mu.Unlock()
	if amount > 0 && p.silverCard > math.MaxInt32-amount {
		p.silverCard = math.MaxInt32
	} else {
		p.silverCard += amount
	}
	return p.silverCard
}

// trySpendSilverCard 银卡（CapitalType2，会话级）扣款，语义同 trySpendSilver。
func (p *playerActor) trySpendSilverCard(cost int32) bool {
	if cost < 0 {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.silverCard < cost {
		return false
	}
	p.silverCard -= cost
	return true
}

// trySpendGold 金币（CapitalType0，会话级）扣款，语义同 trySpendSilver。
func (p *playerActor) trySpendGold(cost int32) bool {
	if cost < 0 {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.gold < cost {
		return false
	}
	p.gold -= cost
	return true
}

func (p *playerActor) currencySnapshotOf() currencySnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return currencySnapshot{Silver: p.silver}
}

// currenciesUpdate 三币单属性帧组（fork handleShopBuy/SellCustom 成功
// 路径统一回帧；goldUpdate/silverUpdate 同构）。
func (p *playerActor) currenciesUpdate() ([][]byte, error) {
	p.mu.Lock()
	gold, silver, card := p.gold, p.silver, p.silverCard
	p.mu.Unlock()
	frames := make([][]byte, 0, 3)
	for _, prop := range []struct {
		index uint16
		name  string
		value int32
	}{
		{propCapitalType0, "CapitalType0", gold},
		{propCapitalType1, "CapitalType1", silver},
		{propCapitalType2, "CapitalType2", card},
	} {
		frame, err := sceneObjectProperties(playerObjectID, playerOwnerID, 0, []clientdata.IndexedProperty{
			{Index: prop.index, Name: prop.name, Value: clientdata.Int64Value(int64(prop.value))},
		})
		if err != nil {
			return nil, err
		}
		frames = append(frames, frame)
	}
	return frames, nil
}
