package rolestate

// repute.go 声望与历练的累加面（任务完成奖励；官服引导抓包 #1933/#2409：
// Repute_Record 行＝名称、数值、标志位，新声望加行、已有声望逐格写数值；
// #1938：历练写玩家属性 LiveGroove_<下标>）。

// AddRepute 声望累加：已有同名记录累加数值（added=false，返回行下标与新
// 数值）；没有则追加一条（added=true）。
func (s *RoleState) AddRepute(name string, count int32) (row int, total int32, added bool) {
	for i := range s.Reputes {
		if s.Reputes[i].Name == name {
			s.Reputes[i].Count += count
			return i, s.Reputes[i].Count, false
		}
	}
	s.Reputes = append(s.Reputes, Repute{Name: name, Count: count})
	return len(s.Reputes) - 1, count, true
}

// AddLiveGroove 历练累加（下标 1..5），返回新余额；下标越界不累加。
func (s *RoleState) AddLiveGroove(index int, count int32) (int32, bool) {
	if index < 1 || index >= len(s.LiveGrooves) || count == 0 {
		return 0, false
	}
	s.LiveGrooves[index] += count
	return s.LiveGrooves[index], true
}
