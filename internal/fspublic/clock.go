package fspublic

import "time"

// nanoNow 单调钟（官方 timeGetTime 语义）。
func nanoNow() int64 { return time.Now().UnixNano() }
