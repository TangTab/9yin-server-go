# captures/

原服抓包落盘位（约定见 ../README.md「captures/」节）。

## 已录入

- `svrlist-request.bin` / `svrlist-response.bin` — svrlist 区服列表方言
  golden 对（源自旧分支 testdata/lister，loopback-lister 对官服系方言的
  实测转录）。**注意**：入库时经过 git 文本规范化，结尾 CR 被洗为 LF；
  实证可用形态（PS1，真实客户端接受）应答结尾为 CRLF——测试按
  「去结尾换行逐字节一致」对拍，见 internal/login/lister_test.go。
- `login-connect-001.bin` — 登录首飞线路帧（123B = XOR+EE 编码态的
  完整 wire 帧，源自旧分支 testdata/protocol/connect-first-flight.bin）。
  解码后 121B 明文 op02 四槽结构恰好收尾（末尾 `d5 83` 是 XOR 过的
  `EE EE` 终止符）。golden 消费见 internal/login/auth_test.go 与
  cmd/probe -login。
