FROM centos:6

RUN yum update -y
RUN yum install -y rpm-build rpmdevtools yum-utils
VOLUME /root/rpmbuild

WORKDIR /root/rpmbuild
CMD rpmbuild -ba /root/rpmbuild/SPECS/swiftfs.json.spec
