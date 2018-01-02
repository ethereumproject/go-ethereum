.DEFAULT_GOAL := build

BUILD_VERSION?=snapshot
SOURCE_FILES?=$$(go list ./... | grep -v /vendor/)
TEST_PATTERN?=.
TEST_OPTIONS?=-race

BINARY=bin
BUILD_TIME=`date +%FT%T%z`
COMMIT=`git log --pretty=format:'%h' -n 1`

LDFLAGS=-ldflags "-X main.date=${BUILD_TIME} -X main.commit=${COMMIT} -X main.version=${BUILD_VERSION}"

setup: ## Install all the build and lint dependencies
	go get -u github.com/alecthomas/gometalinter
	go get -u github.com/golang/dep/...
	go get -u github.com/pierrre/gotestcover
	go get -u golang.org/x/tools/cmd/cover
	dep ensure
	gometalinter --install

fmt: ## gofmt and goimports all go files
	find . -name '*.go' -not -wholename './vendor/*' | while read -r file; do gofmt -w -s "$$file"; goimports -w "$$file"; done

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

build: ## Build a local snapshot binary version
	go build ${LDFLAGS} -o ${BINARY}/nem ./cmd/nem/...

clean: ## Remove a local snapshot binary version
	if [ -d ${BINARY} ] ; then rm -rf ${BINARY} ; fi

install: ## Install to $GOPATH/src
	go install ${LDFLAGS} ./cmd/nem/...

# Absolutely awesome: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


.PHONY: setup test cover fmt lint ci build clean install help static
