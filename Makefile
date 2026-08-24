BINARY := hospitality-scout

.PHONY: build run test vet fmt clean

build:
	go build -o $(BINARY) .

run: build
	./$(BINARY)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .

clean:
	rm -f $(BINARY)
