include $(CURDIR)/main.mk

LOCAL_PATH := $(call my-dir)
LOCAL_TARGET := $(notdir $(LOCAL_PATH))

$(LOCAL_TARGET)-docker:
	@docker build -t $(DOCKER_IMAGE)-db-migration:$(REV) $(LOCAL_PATH)
	@docker tag $(DOCKER_IMAGE)-db-migration:$(REV) $(DOCKER_IMAGE)-db-migration:latest

$(LOCAL_TARGET)-docker.push:
	@docker push $(DOCKER_IMAGE)-db-migration:$(REV)
	@docker push $(DOCKER_IMAGE)-db-migration:latest

PHONY += $(LOCAL_TARGET)

help:
	@echo  '  migration-docker            - Build eth-indexer database migration docker image'
	@echo  '  migration-docker.push       - Push eth-indexer database migration docker image to quay.io'

.PHONY: FORCE
FORCE:
