// Package replaydata 内嵌 B 线 wholesale 回放的 bin 骨架（2026-11-15
// 起生产服不再依赖 G: 盘 tmp 路径——exe 自带数据，env 仅作换包实验
// 覆盖入口）。两文件 = c.pcapng 提取的 335 帧纯子序列透传（P1 13 帧
// 进场/P2 322 帧第二爆发），内容盘点与分派欠账清单见
// replay_native_replace.go 文件头。源文件 tmp/op28_analysis/*.bin
// 演进时必须同步重拷本目录（TestEmbeddedReplayBinsMatchSource 拦漂移）。
package replaydata

import _ "embed"

//go:embed replay_p1_slim.bin
var Phase1 []byte

//go:embed replay_p2_slim.bin
var Phase2 []byte
