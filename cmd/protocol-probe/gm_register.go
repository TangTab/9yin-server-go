// GM 面板「注册账号」端点（fork /api/register 对齐，2026-11-11 移植）。
//
// 语义与 fork exe 实证一致：
//   - 账号/密码为空 → 400「账号不能为空」/「密码不能为空」
//   - 新账号 → {"account": 名, "account_id": N}
//   - 已有账号（含存量未设密账号）→ 覆写口令，{"account": 名, "updated": true}
//
// 账号键由 auth.AccountKeyFor 从明文计算（与登录帧密文哈希同域，注册即生效，
// 存量 acct:v1 键零迁移）；口令校验子存客户端密码槽密文形态（确定性加密，
// 登录时逐字节比对，全程不解密）。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/local/9yin-go-server/internal/auth"
	"github.com/local/9yin-go-server/internal/role"
)

// registerPasswordMax 限注册口令长度：派生链（AES→b64→校验头→b64）放大
// ~2.1 倍，120 字节口令的校验子 240 字节，恰在 accounts.password_hash
// VARBINARY(255) 之内。
const registerPasswordMax = 120

type registerBody struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

// gmRegisterHandler 服务 POST /api/register。accounts 为 nil（测试面板）
// 时返回 503。
func gmRegisterHandler(accounts role.AccountRepository) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if accounts == nil {
			writeGMError(writer, http.StatusServiceUnavailable, "role store unavailable")
			return
		}
		var body registerBody
		if err := decodeGMJSON(request, &body); err != nil {
			writeGMError(writer, http.StatusBadRequest, "invalid JSON body")
			return
		}
		account := strings.TrimSpace(body.Account)
		password := body.Password
		if account == "" {
			writeGMError(writer, http.StatusBadRequest, "账号不能为空")
			return
		}
		if password == "" {
			writeGMError(writer, http.StatusBadRequest, "密码不能为空")
			return
		}
		if len(password) > registerPasswordMax {
			writeGMError(writer, http.StatusBadRequest, "密码过长（上限 120 字节）")
			return
		}
		key, err := auth.AccountKeyFor(account)
		if err != nil {
			writeGMError(writer, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
		defer cancel()
		_, lookupErr := accounts.FindByKey(ctx, role.AccountKey(key))
		exists := lookupErr == nil
		if lookupErr != nil && !errors.Is(lookupErr, role.ErrNotFound) && !errors.Is(lookupErr, role.ErrAccountDisabled) {
			writeGMError(writer, http.StatusInternalServerError, lookupErr.Error())
			return
		}
		if _, err := accounts.RegisterAccount(ctx, role.AccountKey(key), auth.ComputePasswordVerifier(account, password)); err != nil {
			writeGMError(writer, http.StatusInternalServerError, err.Error())
			return
		}
		if exists {
			writeGMJSON(writer, http.StatusOK, map[string]any{"account": account, "updated": true})
			return
		}
		registered, err := accounts.FindByKey(ctx, role.AccountKey(key))
		if err != nil {
			writeGMJSON(writer, http.StatusOK, map[string]any{"account": account})
			return
		}
		writeGMJSON(writer, http.StatusOK, map[string]any{"account": account, "account_id": registered.ID})
	}
}

func decodeGMJSON(request *http.Request, value any) error {
	defer request.Body.Close()
	return json.NewDecoder(http.MaxBytesReader(nil, request.Body, gmCommandBodyLimit)).Decode(value)
}
