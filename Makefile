NAME=swiftfs
BINDIR=bin
GOARCH=amd64
NO_DOCKER ?=

ifeq ($(NO_DOCKER),1)
  include mk/local.mk
else
  include mk/docker.mk
endif
