# supress output, run `make XXX V=` to be verbose
V := @

GOBIN := $(shell go env GOPATH)/bin

MOCKGEN_NAME := mockgen
MOCKGEN_VERSION := v1.4.3

LINTER_NAME := golangci-lint
LINTER_VERSION := v1.29.0

default: build

.PHONY: build
build:
	@echo BUILDING $(OUT)
	$(V) go build -mod=readonly
	@echo DONE

.PHONY: test
test: GO_TEST_FLAGS := -race
test:
	$(V)go test -mod=readonly -v $(GO_TEST_FLAGS) $(GO_TEST_TAGS) ./...

.PHONY: lint
lint: check-linter-version
	$(V)$(LINTER_NAME) run --config .golangci.yml

.PHONY: generate
generate: check-all
	$(V)go generate -mod=readonly -x ./...

.PHONY: vendor
vendor:
	$(V)go mod tidy
	$(V)go mod vendor

.PHONY: install-mockgen
install-mockgen: MOCKGEN_INSTALL_PATH := $(GOBIN)/$(MOCKGEN_NAME)
install-mockgen:
	@echo INSTALLING $(MOCKGEN_INSTALL_PATH) $(MOCKGEN_NAME)
	# we need to change dir to use go modules without updating repo deps
	$(V)cd $(TMPDIR) && GO111MODULE=on go get github.com/golang/mock/mockgen@$(MOCKGEN_VERSION)
	@echo DONE


.PHONY: install-linter
install-linter: LINTER_INSTALL_PATH := $(GOBIN)/$(LINTER_NAME)
install-linter:
	@echo INSTALLING $(LINTER_INSTALL_PATH) $(LINTER_VERSION)
	$(V)curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | \
		sh -s -- -b $(GOBIN) $(LINTER_VERSION)
	@echo DONE

.PHONY: check-mockgen-version
check-mockgen-version: ACTUAL_MOCKGEN_VERSION := $(shell $(MOCKGEN_NAME) --version 2>/dev/null)
check-mockgen-version:
	$(V) [ -z $(ACTUAL_MOCKGEN_VERSION) ] && \
	 echo 'Mockgen is not installed, run `make install-mockgen`' && \
	 exit 1 || true

	$(V)if [ $(ACTUAL_MOCKGEN_VERSION) != $(MOCKGEN_VERSION) ] ; then \
		echo $(MOCKGEN_NAME) is version $(ACTUAL_MOCKGEN_VERSION), want $(MOCKGEN_VERSION) ; \
		echo 'Make sure $$GOBIN has precedence in $$PATH and' \
		'run `make mockgen-install` to install the correct version' ; \
        exit 1 ; \
	fi

.PHONY: check-linter-version
check-linter-version: ACTUAL_LINTER_VERSION := $(shell $(LINTER_NAME) --version 2>/dev/null | awk '{print $$4}')
check-linter-version:
	$(V) [ -z $(ACTUAL_LINTER_VERSION) ] && \
	 echo 'Linter is not installed, run `make install-linter`' && \
	 exit 1 || true

	$(V)if [ v$(ACTUAL_LINTER_VERSION) != $(LINTER_VERSION) ] ; then \
		echo $(LINTER_NAME) is version v$(ACTUAL_LINTER_VERSION), want $(LINTER_VERSION) ; \
		echo 'Make sure $$GOBIN has precedence in $$PATH and' \
		'run `make linter-install` to install the correct version' ; \
        exit 1 ; \
	fi

.PHONY: check-all
check-all: check-linter-version check-mockgen-version

.PHONY: install-all
install-all: install-linter install-mockgen
