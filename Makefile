# 本地无 make 时的等效命令见 README「编译」一节。
# 代码生成走 `go generate ./...`，Makefile 仅作包装，CI 与本地皆可直用 go 命令。

MODULE     := github.com/lizhanyin/9yin-server-go
BUILD_DIR  := build

.PHONY: build test gen vet lint clean

# 编译全部进程入口到 build/
build:
	go build -o $(BUILD_DIR)/ ./cmd/...

# 全量测试（含 golden 对拍用例）
test:
	go test ./...

# 运行 structgen / protogen 代码生成
gen:
	go generate ./...

vet:
	go vet ./...

lint: vet

clean:
	rm -rf $(BUILD_DIR)
