default: build

.PHONY: \
	default \
	build \
	clean \
	help \
	version

build: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

build-linux-amd64:
	@mkdir -p svbin
	@echo "\033[33mBuilding linux/amd64 ▶️ \033[0m"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -o svbin/sv-linux-amd64
	@echo "\033[32mBuild successfully sv-linux-amd64 ✅ \033[0m\n"

build-linux-arm64:
	@mkdir -p svbin
	@echo "\033[33mBuilding linux/arm64 ▶️"
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags '-w -s' -o svbin/sv-linux-arm64
	@echo "\033[32mBuild successfully sv-linux-arm64 ✅ \033[0m\n"

build-darwin-amd64:
	@mkdir -p svbin
	@echo "\033[33mBuilding darwin/amd64 ▶️"
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags '-w -s' -o svbin/sv-darwin-amd64
	@echo "\033[32mBuild successfully sv-darwin-amd64 ✅ \033[0m\n"

build-darwin-arm64:
	@mkdir -p svbin
	@echo "\033[33mBuilding darwin/arm64 (Apple Silicon) ▶️"
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags '-w -s' -o svbin/sv-darwin-arm64
	@echo "\033[32mBuild successfully sv-darwin-arm64 ✅ \033[0m\n"

build-windows-amd64:
	@mkdir -p svbin
	@echo "\033[33mBuilding windows/amd64 ▶️"
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags '-w -s' -o svbin/sv-windows-amd64.exe
	@echo "\033[32mBuild successfully sv-windows-amd64.exe ✅ \033[0m\n"

clean:
	@rm -rf svbin

help:
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@echo '    help               Show this help screen'
	@echo '    build              Compile a program into an executable file'
	@echo '    clean              Clean all executable files'
	@echo '    version            Display Go version'
	@echo ''
	@echo 'Targets run by default is: build'
	@echo ''

version:
	@go version