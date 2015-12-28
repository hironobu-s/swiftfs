all:
	docker build --rm -t swiftfs-rpm .
	docker run --rm -v `pwd`:/root/rpmbuild --name swiftfs-rpm-building swiftfs-rpm
	docker rmi swiftfs-rpm
