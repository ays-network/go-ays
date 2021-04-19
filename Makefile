# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: ays android ios ays-cross evm all test clean
.PHONY: ays-linux ays-linux-386 ays-linux-amd64 ays-linux-mips64 ays-linux-mips64le
.PHONY: ays-linux-arm ays-linux-arm-5 ays-linux-arm-6 ays-linux-arm-7 ays-linux-arm64
.PHONY: ays-darwin ays-darwin-386 ays-darwin-amd64
.PHONY: ays-windows ays-windows-386 ays-windows-amd64

GOBIN = ./build/bin
GO ?= latest
GORUN = env GO111MODULE=on go run

ays:
	$(GORUN) build/ci.go install ./cmd/ays
	@echo "Done building."
	@echo "Run \"$(GOBIN)/ays\" to launch ays."

all:
	$(GORUN) build/ci.go install

android:
	$(GORUN) build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/ays.aar\" to use the library."
	@echo "Import \"$(GOBIN)/ays-sources.jar\" to add javadocs"
	@echo "For more info see https://stackoverflow.com/questions/20994336/android-studio-how-to-attach-javadoc"
	
ios:
	$(GORUN) build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Geth.framework\" to use the library."

test: all
	$(GORUN) build/ci.go test

lint: ## Run linters.
	$(GORUN) build/ci.go lint

clean:
	env GO111MODULE=on go clean -cache
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/kevinburke/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go get -u github.com/golang/protobuf/protoc-gen-go
	env GOBIN= go install ./cmd/abigen
	@type "npm" 2> /dev/null || echo 'Please install node.js and npm'
	@type "solc" 2> /dev/null || echo 'Please install solc'
	@type "protoc" 2> /dev/null || echo 'Please install protoc'

# Cross Compilation Targets (xgo)

ays-cross: ays-linux ays-darwin ays-windows ays-android ays-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/ays-*

ays-linux: ays-linux-386 ays-linux-amd64 ays-linux-arm ays-linux-mips64 ays-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-*

ays-linux-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/ays
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep 386

ays-linux-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/ays
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep amd64

ays-linux-arm: ays-linux-arm-5 ays-linux-arm-6 ays-linux-arm-7 ays-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep arm

ays-linux-arm-5:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/ays
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep arm-5

ays-linux-arm-6:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/ays
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep arm-6

ays-linux-arm-7:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/ays
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep arm-7

ays-linux-arm64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/ays
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep arm64

ays-linux-mips:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/ays
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep mips

ays-linux-mipsle:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/ays
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep mipsle

ays-linux-mips64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/ays
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep mips64

ays-linux-mips64le:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/ays
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/ays-linux-* | grep mips64le

ays-darwin: ays-darwin-386 ays-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/ays-darwin-*

ays-darwin-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/ays
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/ays-darwin-* | grep 386

ays-darwin-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/ays
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ays-darwin-* | grep amd64

ays-windows: ays-windows-386 ays-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/ays-windows-*

ays-windows-386:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/ays
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/ays-windows-* | grep 386

ays-windows-amd64:
	$(GORUN) build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/ays
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/ays-windows-* | grep amd64
