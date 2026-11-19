# 本地无 make 时的等效命令见 README「编译」一节。
# 代码生成走 `go generate ./...`，Makefile 仅作包装，CI 与本地皆可直用 go 命令。

MODULE     := github.com/lizhanyin/9yin-server-go
BUILD_DIR  := build

# Windows 产物需显式 .exe 后缀，Linux/macOS 无后缀
ifeq ($(OS),Windows_NT)
  BIN_EXT := .exe
endif

CMD_PKGS := $(notdir $(wildcard cmd/*))

.PHONY: build test gen vet lint clean single cluster stop

# 编译全部进程入口到 build/，产物统一 game_<包名> 前缀（与 build.bat 一致）
build:
	for d in $(CMD_PKGS); do \
	  go build -o $(BUILD_DIR)/game_$$d$(BIN_EXT) ./cmd/$$d || exit 1; \
	done

# 全量测试（含 golden 对拍用例）
test:
	go test ./...

# 运行 structgen / protogen 代码生成
gen:
	go generate ./...

# 单进程合一形态（Windows 下经 server-cmd\start-single.bat）
single: build
	cmd /c "server-cmd\start-single.bat"

# 集群形态（官方启停序九进程；MEMBERS=12 拉满多场景服）
cluster: build
	cmd /c "server-cmd\start-cluster.bat"

# 集群停机（官方 kill.bat 逆序）
stop:
	cmd /c "server-cmd\stop-cluster.bat"

vet:
	go vet ./...

lint: vet

clean:
	rm -rf $(BUILD_DIR)
