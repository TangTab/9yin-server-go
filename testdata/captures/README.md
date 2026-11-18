# captures/

原服抓包落盘位（约定见 ../README.md「captures/」节）。

## 已录入

- `svrlist-request.bin` / `svrlist-response.bin` — svrlist 区服列表方言
  golden 对（源自旧分支 testdata/lister，loopback-lister 对官服系方言的
  实测转录）。**注意**：入库时经过 git 文本规范化，结尾 CR 被洗为 LF；
  实证可用形态（PS1，真实客户端接受）应答结尾为 CRLF——测试按
  「去结尾换行逐字节一致」对拍，见 internal/login/lister_test.go。
