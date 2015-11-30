NAME=swiftfs
BINDIR=bin
GOARCH=amd64

all: clean  linux

darwin:
	GOOS=$@ GOARCH=$(GOARCH) go build $(GOFLAGS) -o $(BINDIR)/$@/$(NAME)
	cd bin/$@; gzip -c $(NAME) > $(NAME)-osx.$(GOARCH).gz

linux:
	GOOS=$@ GOARCH=$(GOARCH) go build $(GOFLAGS) -o $(BINDIR)/$@/$(NAME)
	cd bin/$@; gzip -c $(NAME) > $(NAME)-linux.$(GOARCH).gz

clean:
	rm -rf $(BINDIR)

test:
	go test -v *.go
	go test -v command/*.go
