package fswatch

import (
	"runtime"
	"sync"
	"time"
)

// SysInfo 官方 sys_info 响应载体（ReceiveSysInfo 0x7bb30：4 个 AddDouble，
// 顺序 cpu/memload/total_phys/avail_phys）。
type SysInfo struct {
	CPURate  float64 // 官方 CCpuRate：PDH \Processor(_total)\% Processor Time（整机）
	MemLoad  float64 // 官方 dwMemoryLoad / 100.0（常量 0x100f92f0 = 100.0）
	TotalPhy float64 // 官方 dwTotalPhys
	AvailPhy float64 // 官方 dwAvailPhys
}

// Sampler 官方 CCpuRate（PDH 计数器）+ Port_GetMemoryInfo
// （GlobalMemoryStatus）的 Go 等价：后台周期采样，Get 即时读。
//
// 差异声明：官方 CPU 为 PDH 整机指标、内存为 GlobalMemoryStatus 物理内存；
// Go 标准库无跨平台等价口，默认以运行时视角给出（runtime.MemStats），
// CPU 缺省 0（未注入采集器时），可经 SetCPUCollector 接入真实指标
// （如 gopsutil cpu.Percent / mem.VirtualMemory）。
type Sampler struct {
	mu        sync.Mutex
	cpu       float64
	mem       SysInfo
	collector func() float64
	stop      chan struct{}
	wg        sync.WaitGroup
}

// NewSampler 启动采样循环（官方 DoWait 每秒 tick 采集）。
func NewSampler() *Sampler {
	s := &Sampler{stop: make(chan struct{})}
	s.collect()
	s.wg.Add(1)
	go s.loop()
	return s
}

// SetCPUCollector 注入真实 CPU 百分比采集器（0-100）。
func (s *Sampler) SetCPUCollector(fn func() float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.collector = fn
}

func (s *Sampler) loop() {
	defer s.wg.Done()
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.collect()
		}
	}
}

func (s *Sampler) collect() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.collector != nil {
		s.cpu = s.collector()
	}
	s.mem = readMemInfo()
}

// Info 官方 GetCpuRate/Port_GetMemoryInfo 读面。
func (s *Sampler) Info() SysInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return SysInfo{CPURate: s.cpu, MemLoad: s.mem.MemLoad, TotalPhy: s.mem.TotalPhy, AvailPhy: s.mem.AvailPhy}
}

// Stop 结束采样线程。
func (s *Sampler) Stop() {
	close(s.stop)
	s.wg.Wait()
}

// readMemInfo 官方 Port_GetMemoryInfo（0x7bd30）的运行时等价：
// out[0]=load/100、out[8]=total、out[0x10]=avail。
func readMemInfo() SysInfo {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	total := float64(ms.Sys)
	avail := float64(ms.Sys - ms.HeapInuse)
	load := 0.0
	if total > 0 {
		load = (total - avail) / total
	}
	return SysInfo{MemLoad: load, TotalPhy: total, AvailPhy: avail}
}
