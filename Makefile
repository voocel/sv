default:build

.PHONY: \
	build \
	clean \
	help \
	version

build: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

build-linux-amd64:
	echo "building linux amd 64"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -o svbin/sv-linux-amd-64
	echo "build successfully linux amd 64"

build-linux-arm64:
	echo "building linux arm 64"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags '-w -s' -o svbin/sv-linux-arm-64
	echo "build successfully linux arm 64"

build-darwin-amd64:
	echo "building darwin amd 64"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags '-w -s' -o svbin/sv-darwin-amd-64
	echo "build successfully darwin amd 64"

build-darwin-arm64:
	echo "building darwin arm 64 (m1)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags '-w -s' -o svbin/sv-darwin-arm-64
	echo "build successfully darwin arm 64 (m1)"

build-windows-amd64:
	echo "building windows amd 64"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags '-w -s' -o svbin/sv-windows-amd-64.exe
	echo "build successfully windows amd 64"

clean:
	rm -f svbin/*

help:
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@echo '    help               Show this help screen'
	@echo '    build           	  Compile a program into an executable file'
	@echo '    clean              Clean all executable files'
	@echo '    version            Display Go version'
	@echo ''
	@echo 'Targets run by default is: build'
	@echo ''

version:
	@go version