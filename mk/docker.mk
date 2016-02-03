all: binary

binary:
	docker run \
		-v `pwd`:/go/src/github.com/hironobu-s/swiftfs \
		-w /go/src/github.com/hironobu-s/swiftfs \
		-ti \
		--name swiftfs-onbuild \
		-e NO_DOCKER=1\
		golang:1.5.2 \
		make
	docker cp swiftfs-onbuild:/go/src/github.com/hironobu-s/swiftfs/bin bin
	docker rm swiftfs-onbuild
