all:
	docker run \
		-v `pwd`:/go/src/github.com/hironobu-s/swiftfs \
		-w /go/src/github.com/hironobu-s/swiftfs \
		-ti \
		--rm \
		--name swiftfs-onbuild \
		golang:1.5.2 \
		make
	docker copy swiftfs-onbuild:/go/src/github.com/hironobu-s/swiftfs/bin bin
	docker rm swiftfs-onbuild
