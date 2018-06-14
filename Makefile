# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

include main.mk

TARGETS := $(sort $(filter-out Dockerfile, $(filter-out flags, $(notdir $(wildcard ./cmd/*)))))
PHONY += $(TARGETS)

PHONY += all
all: $(TARGETS)

.SECONDEXPANSION:
$(TARGETS): $(addprefix $(GOBIN)/,$$@)

$(GOBIN):
	@mkdir -p $@

$(GOBIN)/%: $(GOBIN) FORCE
	@go build -v -o $@ ./cmd/$(notdir $@)
	@echo "Done building."
	@echo "Run \"$(subst $(CURDIR),.,$@)\" to launch $(notdir $@)."

migration-%:
	@$(MAKE) -f migration/Makefile $@

coverage.txt:
	@touch $@

test: coverage.txt FORCE
	@for d in `go list ./... | grep -v vendor | grep -v mock`; do		\
		go test -v -coverprofile=profile.out -covermode=atomic $$d;	\
		if [ $$? -eq 0 ]; then						\
			echo "\033[32mPASS\033[0m:\t$$d";			\
			if [ -f profile.out ]; then				\
				cat profile.out >> coverage.txt;		\
				rm profile.out;					\
			fi							\
		else								\
			echo "\033[31mFAIL\033[0m:\t$$d";			\
			exit -1;						\
		fi								\
	done;

contracts: FORCE
	$(shell solc contracts/erc20.sol --bin --abi --optimize --overwrite --output-dir contracts)
	$(shell abigen --type ERC20Token --abi contracts/ERC20Token.abi -bin contracts/ERC20Token.bin -out contracts/erc20_token.go --pkg contracts)
	$(shell abigen --type MithrilToken --abi contracts/MithrilToken.abi -bin contracts/MithrilToken.bin -out contracts/mithril_token.go --pkg contracts)

eth-indexer-docker:
	@docker build -f ./cmd/Dockerfile -t $(DOCKER_IMAGE):$(DOCKER_IMAGE_TAG) .

eth-indexer-docker.push:
	@docker push $(DOCKER_IMAGE):$(DOCKER_IMAGE_TAG)

PHONY += clean
clean:
	rm -fr $(GOBIN)/*

PHONY += help
help:
	@echo  'Generic targets:'
	@echo  '* indexer                     - Build eth-indexer'
	@echo  ''
	@echo  'Code generation targets:'
	@echo  '  contracts                   - Compile solidity contracts'
	@echo  ''
	@echo  'Docker targets:'
	@echo  '  eth-indexer-docker          - Build eth-indexer docker image'
	@echo  '  eth-indexer-docker.push     - Push eth-indexer docker image to quay.io'
	@$(MAKE) -f migration/Makefile $@
	@echo  ''
	@echo  'Test targets:'
	@echo  '  test                        - Run all unit tests'
	@echo  ''
	@echo  'Cleaning targets:'
	@echo  '  clean                       - Remove built executables'
	@echo  ''
	@echo  'Execute "make" or "make all" to build all targets marked with [*] '
	@echo  'For further info see the ./README.md file'

PHONY += FORCE
FORCE:

.PHONY: $(PHONY)
