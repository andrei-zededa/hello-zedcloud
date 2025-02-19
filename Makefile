REGISTRY_PATH := docker.io/andreizededa
APP_NAME := hello-zedcloud 
VERSION := $(shell misc/version_from_git_tags.bash)
VER_TO_CONT_TAG := $(shell echo "${VERSION}" | tr '+' '_')

build:
	@echo "${APP_NAME} version ${VERSION}"
	@echo "${VERSION}" > ./version
	go build -o ./hello-zedcloud

lint:
	golangci-lint run
	shellcheck misc/version_from_git_tags.bash

image: build
	docker build -t "${APP_NAME}:${VER_TO_CONT_TAG}" -f ./Dockerfile .

image-latest: image
	docker tag "${APP_NAME}:${VER_TO_CONT_TAG}" "${APP_NAME}:latest"

push: image
	docker push "${APP_NAME}:${VER_TO_CONT_TAG}" "${REGISTRY_PATH}/${APP_NAME}:${VER_TO_CONT_TAG}"

push-latest: image-latest push
	docker push "${APP_NAME}:${VER_TO_CONT_TAG}" "${REGISTRY_PATH}/${APP_NAME}:latest"
