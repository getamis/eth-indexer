CURDIR := $(shell pwd)
GOBIN = $(shell pwd)/build/bin
DOCKER_REPOSITORY := quay.io/amis
DOCKER_IMAGE := $(DOCKER_REPOSITORY)/eth-indexer
ifeq ($(DOCKER_IMAGE_TAG),)
DOCKER_IMAGE_TAG := $(shell git rev-parse --short HEAD 2> /dev/null)
endif

define my-dir
$(patsubst %/,%,$(dir $(firstword $(MAKEFILE_LIST))))
endef
