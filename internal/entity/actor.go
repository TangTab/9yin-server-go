// Package entity contains the authoritative runtime state for world objects.
package entity

import (
	"math"
	"sync"
	"time"
)

const (
	LogicStateNormal   uint8 = 0
	LogicStateFighting uint8 = 1
	// LogicStateSitcross is the modern native LS_SITCROSS state.  The client
	// owns the animation, while the server owns when this state may be entered.
	LogicStateSitcross uint8 = 102
	LogicStateDied     uint8 = 120
)

// Actor is server authority for a character's combat state.  The client may
// render it from ServerObjectProperty (0x10), but it never owns the values.
type Actor struct {
	mu sync.RWMutex

	objectID           uint32
	ownerID            uint32
	name               string
	hp                 int32
	baseMaxHP          int32
	maxHP              int32
	hitHP              int32
	mp                 int32
	baseMaxMP          int32
	maxMP              int32
	qingGong           int32
	maxQingGong        int32
	maxQingGongAdd     int32
	qingGongConsumeMul float32
	qingGongUpdatedAt  time.Time
	speedRatio         int32
	logic              uint8
}

type ActorState struct {
	ObjectID, OwnerID    uint32
	Name                 string
	HP, BaseMaxHP, MaxHP int32
	HitHP                int32
	MP, BaseMaxMP, MaxMP int32
	QingGongPoint        int32
	MaxQingGongPoint     int32
	MaxQingGongPointAdd  int32
	QingGongConsumeMul   float32
	SpeedRatio           int32
	LogicState           uint8
}

func NewActor(objectID, ownerID uint32, name string, maxHP, maxMP int32) *Actor {
	if maxHP < 1 {
		maxHP = 1
	}
	if maxMP < 1 {
		maxMP = 1
	}
	// Current retail gameplay documentation describes the normal full bar as
	// 150. Equipment/buff additions are represented separately below.
	const defaultMaxQingGong = 150
	return &Actor{
		objectID: objectID, ownerID: ownerID, name: name,
		hp: maxHP, baseMaxHP: maxHP, maxHP: maxHP, hitHP: maxHP,
		mp: maxMP, baseMaxMP: maxMP, maxMP: maxMP,
		qingGong: defaultMaxQingGong, maxQingGong: defaultMaxQingGong,
		qingGongUpdatedAt: time.Now(), speedRatio: 100,
	}
}

// NewPlayer constructs the default, living player state used by a new scene
// session.  Keeping these values here prevents the spawn and later 0x10
// updates from drifting into separate sources of truth.
func NewPlayer(objectID, ownerID uint32, name string) *Actor {
	return NewActor(objectID, ownerID, name, 1000, 500)
}

func (a *Actor) Snapshot() ActorState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return ActorState{
		ObjectID: a.objectID, OwnerID: a.ownerID, Name: a.name,
		HP: a.hp, BaseMaxHP: a.baseMaxHP, MaxHP: a.maxHP, HitHP: a.hitHP,
		MP: a.mp, BaseMaxMP: a.baseMaxMP, MaxMP: a.maxMP,
		QingGongPoint: a.qingGong, MaxQingGongPoint: a.maxQingGong,
		MaxQingGongPointAdd: a.maxQingGongAdd, QingGongConsumeMul: a.qingGongConsumeMul,
		SpeedRatio: a.speedRatio, LogicState: a.logic,
	}
}

// ApplyDamage returns true when the actor state changed.  Death is a state
// transition, not a client-side interpretation of HP == 0.
func (a *Actor) ApplyDamage(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.hp == 0 {
		return false
	}
	if a.hitHP < a.hp {
		a.hitHP = a.hp
	}
	a.hp -= amount
	if a.hp <= 0 {
		a.hp = 0
		a.hitHP = 0
		a.logic = LogicStateDied
	} else {
		a.logic = LogicStateFighting
	}
	return true
}

// ApplyHeal restores HP up to MaxHP.  Death is intentionally not cleared by
// healing: a future revive flow must make that state transition explicitly.
func (a *Actor) ApplyHeal(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic == LogicStateDied || a.hp >= a.maxHP {
		return false
	}
	before := a.hp
	a.hp += amount
	if a.hp > a.maxHP {
		a.hp = a.maxHP
	}
	if a.hitHP < a.hp {
		a.hitHP = a.hp
	}
	return a.hp != before
}

// RecoverBruisedHPWhileFighting settles only the bruised-blood layer during
// combat.  HitHP is the pre-hit/secondary bar, so reducing it toward HP makes
// the visible bruise (HitHP-HP) dissipate without granting any real HP.
//
// This deliberately does nothing in other LogicState values: sitting and
// explicit healing own real HP recovery, while a dead actor cannot recover.
func (a *Actor) RecoverBruisedHPWhileFighting(amount int32) bool {
	if amount <= 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic != LogicStateFighting || a.hp <= 0 || a.hitHP <= a.hp {
		return false
	}
	a.hitHP -= amount
	if a.hitHP < a.hp {
		a.hitHP = a.hp
	}
	return true
}

// RestoreWhileSit applies one authoritative sitting-recovery settlement. It
// never revives a dead actor and does nothing outside LS_SITCROSS; UI rate
// properties are therefore never mistaken for actual healing.
func (a *Actor) RestoreWhileSit(hpAmount, mpAmount int32) bool {
	if hpAmount < 0 || mpAmount < 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.logic != LogicStateSitcross || a.hp <= 0 {
		return false
	}
	changed := false
	if hpAmount > 0 && a.hp < a.maxHP {
		a.hp += hpAmount
		if a.hp > a.maxHP {
			a.hp = a.maxHP
		}
		if a.hitHP < a.hp {
			a.hitHP = a.hp
		}
		changed = true
	}
	if mpAmount > 0 && a.mp < a.maxMP {
		a.mp += mpAmount
		if a.mp > a.maxMP {
			a.mp = a.maxMP
		}
		changed = true
	}
	return changed
}

// SetDerivedMaxima installs the authoritative total resource caps after the
// role's equipped inner power has been evaluated. Base maxima never change;
// they remain available for the client protocol's MaxHP/MaxMP + Add display.
// A fully recovered actor stays full when a cap grows, while a partially
// depleted actor keeps its current absolute resource and is only clamped when
// a cap shrinks.
func (a *Actor) SetDerivedMaxima(maxHP, maxMP int32) bool {
	if maxHP < 1 {
		maxHP = 1
	}
	if maxMP < 1 {
		maxMP = 1
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.maxHP == maxHP && a.maxMP == maxMP {
		return false
	}
	hpWasFull := a.hp >= a.maxHP
	mpWasFull := a.mp >= a.maxMP
	a.maxHP = maxHP
	a.maxMP = maxMP
	if hpWasFull {
		a.hp = maxHP
	} else if a.hp > maxHP {
		a.hp = maxHP
	}
	if a.hitHP > maxHP {
		a.hitHP = maxHP
	}
	if mpWasFull {
		a.mp = maxMP
	} else if a.mp > maxMP {
		a.mp = maxMP
	}
	return true
}

// SpendMP is the authoritative resource gate for future skill handling.
func (a *Actor) SpendMP(cost int32) bool {
	if cost < 0 {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.mp < cost {
		return false
	}
	a.mp -= cost
	return true
}

// UseQingGong applies the current server-authoritative lightweight-resource
// cost. The multiplier is a fractional modifier: -0.20 means 80% cost and
// +0.20 means 120% cost. Costs cannot become negative.
func (a *Actor) UseQingGong(baseCost int32, now time.Time) (cost int32, ok bool) {
	if baseCost < 0 {
		return 0, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.restoreQingGongLocked(now)
	multiplier := float64(1 + a.qingGongConsumeMul)
	if multiplier < 0 {
		multiplier = 0
	}
	// qingGongConsumeMul is persisted as float32; remove representational dust
	// before rounding so -0.2 on a 25-point skill remains 20, not 21.
	cost = int32(math.Ceil(float64(baseCost)*multiplier - 1e-6))
	if a.qingGong < cost {
		return cost, false
	}
	a.qingGong -= cost
	// Local-test rule: do not let repeated movement verification strand the
	// player with an unusable lightness bar.  The refill is deliberately after
	// the authoritative cost/gate, so an action still has to be affordable to
	// start; only a successful action that leaves fewer than 50 points refills.
	if a.qingGong < 50 {
		a.qingGong = a.maxQingGong + a.maxQingGongAdd
		a.qingGongUpdatedAt = now
	}
	return cost, true
}

// RestoreQingGong advances the local-test resource timer. The rate is kept
// here (rather than client Lua) so later buffs can modify it authoritatively.
func (a *Actor) RestoreQingGong(now time.Time) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.restoreQingGongLocked(now)
}

func (a *Actor) restoreQingGongLocked(now time.Time) bool {
	if now.Before(a.qingGongUpdatedAt) {
		return false
	}
	// Local baseline: 10 points per second. Official config supplies the
	// per-skill cost; regeneration modifiers are future buff effects.
	points := int32(now.Sub(a.qingGongUpdatedAt) / (100 * time.Millisecond))
	if points <= 0 {
		return false
	}
	a.qingGongUpdatedAt = a.qingGongUpdatedAt.Add(time.Duration(points) * 100 * time.Millisecond)
	before := a.qingGong
	a.qingGong += points
	max := a.maxQingGong + a.maxQingGongAdd
	if a.qingGong > max {
		a.qingGong = max
	}
	return a.qingGong != before
}

func (a *Actor) SetQingGongConsumeMultiplier(multiplier float32) {
	a.mu.Lock()
	a.qingGongConsumeMul = multiplier
	a.mu.Unlock()
}

func (a *Actor) SetSpeedRatio(ratio int32) {
	if ratio < 0 {
		ratio = 0
	}
	a.mu.Lock()
	a.speedRatio = ratio
	a.mu.Unlock()
}

// SetLogicState changes only the current state; HP and MP remain owned by the
// same actor and are synchronised together by the player adapter.
func (a *Actor) SetLogicState(state uint8) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logic = state
}

// Revive is the minimal authoritative revival transition.
func (a *Actor) Revive() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.hp = a.maxHP
	a.hitHP = a.maxHP
	a.mp = a.maxMP
	a.qingGong = a.maxQingGong + a.maxQingGongAdd
	a.qingGongUpdatedAt = time.Now()
	a.speedRatio = 100
	a.logic = LogicStateNormal
}
