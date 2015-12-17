all: clean  linux

linux:
	go get .
	GOOS=$@ GOARCH=$(GOARCH) go build $(GOFLAGS) -o $(BINDIR)/$@/$(NAME)
	cd bin/$@; gzip -c $(NAME) > $(NAME)-linux.$(GOARCH).gz

clean:
	rm -rf $(BINDIR)/*

test:
	go test github.com/hironobu-s/swiftfs/...
