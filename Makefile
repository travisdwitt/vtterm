BINARY := vtterm

.PHONY: build install run fmt clean

build:
	go build -o $(BINARY) .

install:
	go install .

run:
	go run .

fmt:
	gofmt -w .

clean:
	rm -f $(BINARY)
