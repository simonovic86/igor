.PHONY: build clean

build:
	GOOS=wasip1 GOARCH=wasm go build -o agent.wasm .

clean:
	rm -f agent.wasm
