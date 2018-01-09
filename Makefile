.DEFAULT_GOAL := build_geth

BUILD_VERSION?=snapshot
SOURCE_FILES?=$$(go list ./... | grep -v /vendor/)
TEST_PATTERN?=.
TEST_OPTIONS?=-race

BINARY=bin
BUILD_TIME=`date +%FT%T%z`
COMMIT=`git log --pretty=format:'%h' -n 1`

# Choose to install geth with or without SputnikVM.
WITH_SVM?=1

LDFLAGS=-ldflags "-X main.Version="`git describe --tags`

setup: ## Install all the build and lint dependencies
	go get -u github.com/alecthomas/gometalinter
	go get -u github.com/golang/dep/...
	go get -u github.com/pierrre/gotestcover
	go get -u golang.org/x/tools/cmd/cover
	dep ensure
	gometalinter --install

build_all: ## Build a local snapshot binary versions of all commands
	./scripts/build.sh ${BINARY}
	make build_geth

build_geth: ## Build a local snapshot binary version of geth. Use WITH_SVM=0 to disable building with SputnikVM (default: WITH_SVM=1)
	$(info Building ${BINARY}/geth)
	if [ ${WITH_SVM} == 1 ]; then ./scripts/build_sputnikvm.sh build ; else mkdir -p ./bin && go build ${LDFLAGS} -o ${BINARY}/geth ./cmd/geth ; fi

install_all: ## Install all packages to $GOPATH/bin
	go install ./cmd/{abigen,bootnode,disasm,ethtest,evm,gethrpctest,rlpdump}
	make install_geth

install_geth: ## Install geth to $GOPATH/bin. Use WITH_SVM=0 to disable building with SputnikVM (default: WITH_SVM=1)
	$(info Installing $$GOPATH/bin/geth)
	if [ ${WITH_SVM} == 1 ]; then ./scripts/build_sputnikvm.sh install ; else go install ${LDFLAGS} ./cmd/geth ; fi

fmt: ## gofmt and goimports all go files
	find . -name '*.go' -not -wholename './vendor/*' -not -wholename './_vendor*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done

ci: lint test ## Run all code checks and tests

lint: ## Run all the linters
	gometalinter \
		--tests \
		--vendor \
		--vendored-linters \
		--disable=interfacer \
		--disable=maligned \
		--enable=gosimple \
		--enable=staticcheck \
		--enable=gofmt \
		--enable=goimports \
		--enable=lll \
		--enable=misspell \
		--cyclo-over=15 \
		--dupl-threshold=100 \
		--line-length=120 \
		--deadline=360s \
		./...

test: ## Run all the tests
	gotestcover $(TEST_OPTIONS) -covermode=atomic -coverprofile=coverage.txt $(SOURCE_FILES) -run $(TEST_PATTERN) -timeout=30s

cover: test ## Run all the tests and opens the coverage report
	go tool cover -html=coverage.txt

clean: ## Remove local snapshot binary directory
	if [ -d ${BINARY} ] ; then rm -rf ${BINARY} ; fi

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


.PHONY: setup test cover fmt lint ci build_all build_geth install_all install_geth clean help static
