FROM golang:1.5.2-onbuild

RUN go get
WORKDIR /go/src/github.com/hironobu-s/swiftfs
ADD . /go/src/github.com/hironobu-s/swiftfs

RUN go build
RUN cp swiftfs /usr/bin/
