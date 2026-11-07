package main

import (
	"log"
)

// 新手选择（msg972，fork freshman_guide.go:31-47 反汇编还原）。
//
// fork 语义：`[972, i32 choice]`（exactInt32 宽容解析）——**纯日志无帧、
// 无状态写入**（引导流程由客户端本地驱动，FinishFreshmanRecStr@225 由
// 客户端自行管理）；choice==5 → 标签「老手(跳过引导)」(len 20)，其余
// 值 → 「新手(走引导)」(len 17)。参数不完整/非整数返回 (false, nil)
// 让其余 handler 跳过。
const clientCustomFreshManChoice int32 = 972

// freshmanChoiceLabel 是 fork 内嵌的两段标签（CMOV 选择，长度 17/20）。
func freshmanChoiceLabel(choice int32) string {
	if choice == 5 {
		return "老手(跳过引导)"
	}
	return "新手(走引导)"
}

func handleFreshManChoice(custom clientCustomMessage, remote string) (bool, error) {
	if len(custom.Values) == 0 || custom.Values[0].Type != 2 || custom.Values[0].Int32 != clientCustomFreshManChoice {
		return false, nil
	}
	if len(custom.Values) < 2 {
		return false, nil
	}
	choice, ok := custom.Values[1].exactInt32()
	if !ok {
		return false, nil
	}
	log.Printf("%s: fresh-man choice=%d (%s)", remote, choice, freshmanChoiceLabel(choice))
	return true, nil
}
