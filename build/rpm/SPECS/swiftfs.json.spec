Name:           swiftfs
Version:        0.2.1
Release:        1%{?dist}
Summary:        SwiftFS is the file system to mount Swift OpenStack Object Storage via FUSE.
License:        MIT
URL:            https://github.com/hironobu-s/swiftfs
Source0:        https://github.com/hironobu-s/swiftfs/releases/download/current/swiftfs.amd64.gz

%description
SwiftFS is the file system to mount Swift OpenStack Object Storage via FUSE.

%prep
curl -s -L -o /root/rpmbuild/SOURCES/swiftfs.amd64.gz https://github.com/hironobu-s/swiftfs/releases/download/current/swiftfs.amd64.gz

%build
gunzip -c $RPM_SOURCE_DIR/swiftfs.amd64.gz > swiftfs
chmod +x swiftfs

%install
mkdir -p $RPM_BUILD_ROOT/usr/bin
cp $RPM_BUILD_DIR/swiftfs $RPM_BUILD_ROOT/usr/bin/swiftfs

%clean
rm -rf $RPM_BUILD_ROOT

%files
/usr/bin/swiftfs
