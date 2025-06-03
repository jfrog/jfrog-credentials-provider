# Makefile for kubelet-plugin-aws CI

# Variables are expanded to string by Make
SHELL := /bin/bash
PROJECT_NAME := kubelet-plugin-aws
PLATFORMS:= linux/arm64 linux/amd64
BUILD_DIR := bin
BIN=jfrog-credential-provider-aws

# Make targets are file specific, thus PHONY 
.PHONY: all
all: build

# Build target
.PHONY: build-all
build-all:
	@echo "Building go binaries for all OSs..."
	VERSION := $(shell git describe --tags --always)
	PLATFORMS="$(PLATFORMS)" BUILD_DIR=$(BUILD_DIR) BIN=$(BIN) VERSION=$(VERSION) ./build-binary.sh
