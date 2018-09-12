DOCKER_REPOSITORY := quay.io/amis
DOCKER_IMAGE := $(DOCKER_REPOSITORY)/indexer_geth
ifeq ($(DOCKER_IMAGE_TAG),)
DOCKER_IMAGE_TAG := $(shell git rev-parse --short HEAD 2> /dev/null)
endif
