default:build

.PHONY:build

build:
	echo "building linux amd 64"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-w -s' -o bin/sv-linux-amd-64
	echo "build successfully linux amd 64"

	echo "building linux arm 64"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags '-w -s' -o bin/sv-linux-arm-64
	echo "build successfully linux arm 64"

	echo "building darwin 64"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags '-w -s' -o bin/sv-darwin-64
	echo "build successfully darwin 64"

	echo "building darwin arm-64 (m1)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags '-w -s' -o bin/sv-darwin-arm-64
	echo "build successfully darwin arm-64 (m1)"

	echo "building windows 64"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags '-w -s' -o bin/sv-windows-64.exe
	echo "build successfully windows 64"
