NAME=swiftfs
BINDIR=bin
GOARCH=amd64
USE_CONTAINER ?=

ifeq ($(USE_DOCKER),1)
  include mk/docker.mk
else
  include mk/local.mk
endif
