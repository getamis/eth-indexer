# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

PHONY += all docker clean

include main.mk

TARGETS := $(sort $(notdir $(wildcard ./cmd/*)))
PHONY += $(TARGETS)

all: $(TARGETS)

.SECONDEXPANSION:
$(TARGETS): $(addprefix $(GOBIN)/,$$@)

$(GOBIN):
	@mkdir -p $@

$(GOBIN)/%: $(GOBIN) FORCE
	@go build -v -o $@ ./cmd/$(notdir $@)
	@echo "Done building."
	@echo "Run \"$(subst $(CURDIR),.,$@)\" to launch $(notdir $@)."

PROTOC_INCLUDES := \
		-I$(CURDIR)/vendor/github.com/gogo/protobuf/types \
		-I$(CURDIR)/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		-I$(GOPATH)/src

GRPC_PROTOS := \
	service/pb/*.proto

service-grpc: FORCE
	@protoc $(PROTOC_INCLUDES) \
		--gofast_out=plugins=grpc,\
Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/duration.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/struct.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types,\
Mgoogle/protobuf/wrappers.proto=github.com/gogo/protobuf/types:$(GOPATH)/src \
		$(addprefix $(CURDIR)/,$(GRPC_PROTOS))

	@protoc $(PROTOC_INCLUDES) \
		--grpc-gateway_out=logtostderr=true:$(GOPATH)/src $(addprefix $(CURDIR)/,$(GRPC_PROTOS))

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

contracts:
	$(shell solc eth/contracts/erc20.sol --abi --overwrite --output-dir eth/contracts)
	$(shell abigen --type ERC20 --abi eth/contracts/ERC20.abi --out eth/contracts/erc20.go --pkg contracts)

# dashboard-%:
# 	@$(MAKE) -f dashboard/Makefile $@

# %-docker:
# 	@docker build -f ./cmd/$(subst -docker,,$@)/Dockerfile -t $(DOCKER_IMAGE)-$(subst -docker,,$@):$(REV) .
# 	@docker tag $(DOCKER_IMAGE)-$(subst -docker,,$@):$(REV) $(DOCKER_IMAGE)-$(subst -docker,,$@):latest

# %-docker.push:
# 	@docker push $(DOCKER_IMAGE)-$(subst -docker.push,,$@):$(REV)
# 	@docker push $(DOCKER_IMAGE)-$(subst -docker.push,,$@):latest

clean:
	rm -fr $(GOBIN)/*

PHONY: help
help:
	@echo  'Generic targets:'
	@echo  '  indexer                       - Build indexer service'
	@echo  ''
	# @echo  'Code generation targets:'
	# @echo  '  server-grpc                 - Generate go files from proto for API'
	# @echo  ''
	# @echo  'Docker targets:'
	# @echo  '  server-docker               - Build regulatory API docker image'
	# @echo  '  server-docker.push          - Push regulatory API docker image to quay.io'
	# @$(MAKE) -f migration/Makefile $@
	# @$(MAKE) -f dashboard/Makefile $@
	# @echo  ''
	@echo  'Execute "make" or "make all" to build all targets marked with [*] '
	@echo  'For further info see the ./README.md file'

.PHONY: $(PHONY)

.PHONY: FORCE
FORCE:
