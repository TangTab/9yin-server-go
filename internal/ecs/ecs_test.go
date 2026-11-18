package ecs

import "testing"

type pos struct{ X, Y float32 }
type vel struct{ DX, DY float32 }
type dead struct{}

func TestEntityLifecycle(t *testing.T) {
	w := New()
	e1 := w.Create()
	if !w.Alive(e1) {
		t.Fatal("新建实体应存活")
	}
	if e1.Index() != 0 || e1.Gen() != 1 {
		t.Fatalf("首个实体应为 index=0 gen=1，得到 %v", e1)
	}
	if w.Alive(NullEntity) {
		t.Fatal("NullEntity 不应存活")
	}

	w.Destroy(e1)
	if w.Alive(e1) {
		t.Fatal("销毁后旧句柄应失效")
	}

	// 槽位复用：新实体同 index、gen+1
	e2 := w.Create()
	if e2.Index() != e1.Index() || e2.Gen() != e1.Gen()+1 {
		t.Fatalf("槽位复用应 gen+1：%v vs 旧 %v", e2, e1)
	}
	if w.Alive(e1) || !w.Alive(e2) {
		t.Fatal("仅新句柄存活")
	}
}

func TestStorageCRUD(t *testing.T) {
	w := New()
	e := w.Create()

	if w.Has[pos](e) {
		t.Fatal("未添加不应 Has")
	}
	w.Add(e, pos{1, 2})
	if !w.Has[pos](e) {
		t.Fatal("添加后应 Has")
	}
	got, ok := w.Get[pos](e)
	if !ok || got != (pos{1, 2}) {
		t.Fatalf("Get 结果错误：%v %v", got, ok)
	}

	v0 := w.Version[pos](e)
	if !w.Set(e, pos{1, 2}) {
		t.Fatal("Set 应成功")
	}
	if w.Version[pos](e) <= v0 {
		t.Fatal("Set 后版本应自增")
	}
	ref := w.GetRef[pos](e)
	ref.X = 9
	if got, _ := w.Get[pos](e); got.X != 9 {
		t.Fatal("GetRef 写入应生效")
	}
	w.Add(e, pos{0, 0}) // 覆写路径也应自增版本
	if w.Version[pos](e) <= v0 {
		t.Fatal("覆写后版本应自增")
	}

	w.Remove[pos](e)
	if w.Has[pos](e) || w.GetRef[pos](e) != nil {
		t.Fatal("Remove 后不应可取")
	}
	if w.Version[pos](e) != 0 {
		t.Fatal("移除后版本应为 0")
	}
}

func TestDestroyCleansAllStorages(t *testing.T) {
	w := New()
	e := w.Create()
	w.Add(e, pos{})
	w.Add(e, vel{})
	w.Destroy(e)
	if w.Count[pos]() != 0 || w.Count[vel]() != 0 {
		t.Fatal("销毁应清空全部组件")
	}
}

func TestEachIntersection(t *testing.T) {
	w := New()
	for i := 0; i < 10; i++ {
		e := w.Create()
		w.Add(e, pos{X: float32(i)})
		if i%2 == 0 {
			w.Add(e, vel{})
		}
	}
	var n1, n2 int
	w.Each1(func(e EntityID, p *pos) { n1++ })
	w.Each2(func(e EntityID, p *pos, v *vel) { n2++ })
	if n1 != 10 || n2 != 5 {
		t.Fatalf("Each1=%d Each2=%d，应为 10/5", n1, n2)
	}
}

func TestEachStructuralSafety(t *testing.T) {
	w := New()
	var es []EntityID
	for i := 0; i < 6; i++ {
		e := w.Create()
		es = append(es, e)
		w.Add(e, pos{X: float32(i)})
	}

	// 回调内销毁当前实体 + 摘除后续实体的组件：遍历应安全且数量正确
	visited := 0
	w.Each1(func(e EntityID, p *pos) {
		visited++
		if e == es[0] {
			w.Destroy(e) // 当前实体
		}
		if e == es[1] {
			w.Remove[pos](es[2]) // 后续实体
		}
	})
	if visited != 6 {
		t.Fatalf("快照遍历应访问 6 个，实际 %d", visited)
	}
	if w.Alive(es[0]) {
		t.Fatal("es[0] 应已销毁")
	}
	if w.Has[pos](es[2]) {
		t.Fatal("es[2] 组件应已被回调内移除")
	}
}

func TestVersionTravelsOnSwap(t *testing.T) {
	w := New()
	e1, e2 := w.Create(), w.Create()
	w.Add(e1, pos{1, 1})
	w.Add(e2, pos{2, 2})
	w.Set(e2, pos{3, 3})
	w.Set(e2, pos{4, 4})
	v2 := w.Version[pos](e2)

	w.Remove[pos](e1) // e2 会被 swap 到 e1 的 dense 槽
	if w.Version[pos](e2) != v2 {
		t.Fatalf("swap 后版本应随实体走：%d != %d", w.Version[pos](e2), v2)
	}
	if got, _ := w.Get[pos](e2); got != (pos{4, 4}) {
		t.Fatal("swap 后取值应不变")
	}
}

func TestSchedulerOrderAndTick(t *testing.T) {
	w := New()
	s := NewScheduler()
	var order []string
	s.AddFunc("a", func(w *World, dt float64) {
		if w.CurrentTick() == 0 {
			t.Fatal("System 执行时帧号应已推进")
		}
		order = append(order, "a")
	})
	s.AddFunc("b", func(*World, float64) { order = append(order, "b") })
	s.Tick(w, 1.0/60)
	s.Tick(w, 1.0/60)
	if len(order) != 4 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("执行顺序错误：%v", order)
	}
	if w.CurrentTick() != 2 {
		t.Fatalf("帧号应为 2，实际 %d", w.CurrentTick())
	}
}

func BenchmarkEach2(b *testing.B) {
	w := New()
	for i := 0; i < 10000; i++ {
		e := w.Create()
		w.Add(e, pos{X: float32(i)})
		if i%2 == 0 {
			w.Add(e, vel{DX: 1})
		}
	}
	s := NewScheduler()
	s.AddFunc("move", func(w *World, dt float64) {
		w.Each2(func(e EntityID, p *pos, v *vel) {
			p.X += v.DX * float32(dt)
		})
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Tick(w, 1.0/60)
	}
}
