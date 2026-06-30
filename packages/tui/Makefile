.PHONY: build install run run-standalone clean

build:
	go build -o bin/codev-tui .

install:
	go install .

run:
	@if [ -n "$$ZELLIJ" ]; then go run . --no-zellij; else go run .; fi

run-standalone:
	go run . --no-zellij

clean:
	rm -rf bin/
