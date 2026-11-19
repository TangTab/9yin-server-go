package fsserver

// InterConnect 复刻 CInterConnect（§2.3）：ident 算术 + 连接注册表 + 发送族。
//
// Go 侧发送面以 SendFunc 注入（fscomm.Adapter / 测试桩皆可）；ident ↔ 连接
// 的注册由 Inner 受面（assemble.go）驱动 SetCommIdent/SetAdapterDisconnect。
type InterConnect struct {
	// Send 按 comm ident 外发一条内网帧 {src=world(0), dst, type}+payload。
	Send func(ident, dst, typ uint32, payload []byte) error

	// Connected 按 ident 查询连接在册（CheckConnect 族）。
	Connected func(ident uint32) bool
}

// MemberIdent …（0x18+idx，越界 panic 由调用侧断言避免）
func MemberIdent(idx uint32) uint32 { return MemberIdentBase + idx }

// EntryIdent …（8+idx）
func EntryIdent(idx uint32) uint32 { return EntryIdentBase + idx }

// StoreIdent …（4+idx）
func StoreIdent(idx uint32) uint32 { return StoreIdentBase + idx }

// SendToMember 定向 member（断言 idx < MAX_MEMBER_NUM）。
func (ic *InterConnect) SendToMember(idx, dst, typ uint32, payload []byte) error {
	if idx >= MaxMemberNum {
		return errIndex("size_t(index) < MAX_MEMBER_NUM")
	}
	return ic.Send(MemberIdent(idx), dst, typ, payload)
}

// SendToAllMember 全 member 广播（官方遍历 0..15）。
func (ic *InterConnect) SendToAllMember(dst, typ uint32, payload []byte) error {
	for i := uint32(0); i < MaxMemberNum; i++ {
		if ic.Connected == nil || ic.Connected(MemberIdent(i)) {
			if err := ic.Send(MemberIdent(i), dst, typ, payload); err != nil {
				return err
			}
		}
	}
	return nil
}

// SendToEntry 定向 entry。
func (ic *InterConnect) SendToEntry(idx, dst, typ uint32, payload []byte) error {
	if idx >= MaxEntryNum {
		return errIndex("size_t(index) < MAX_ENTRY_NUM")
	}
	return ic.Send(EntryIdent(idx), dst, typ, payload)
}

// SendToAllEntry …
func (ic *InterConnect) SendToAllEntry(dst, typ uint32, payload []byte) error {
	for i := uint32(0); i < MaxEntryNum; i++ {
		if ic.Connected == nil || ic.Connected(EntryIdent(i)) {
			if err := ic.Send(EntryIdent(i), dst, typ, payload); err != nil {
				return err
			}
		}
	}
	return nil
}

// SendToStore 定向 store。
func (ic *InterConnect) SendToStore(idx, dst, typ uint32, payload []byte) error {
	if idx >= MaxStoreNum {
		return errIndex("size_t(index) < MAX_STORE_NUM")
	}
	return ic.Send(StoreIdent(idx), dst, typ, payload)
}

// SendToAllStore …
func (ic *InterConnect) SendToAllStore(dst, typ uint32, payload []byte) error {
	for i := uint32(0); i < MaxStoreNum; i++ {
		if ic.Connected == nil || ic.Connected(StoreIdent(i)) {
			if err := ic.Send(StoreIdent(i), dst, typ, payload); err != nil {
				return err
			}
		}
	}
	return nil
}

// SendToPublic …
func (ic *InterConnect) SendToPublic(dst, typ uint32, payload []byte) error {
	return ic.Send(PublicIdent, dst, typ, payload)
}

type indexError string

func (e indexError) Error() string { return string(e) }

func errIndex(what string) error { return indexError(what) }
