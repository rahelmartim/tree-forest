SITE := site
PORT ?= 8080

.PHONY: test bench web serve gif forest clean

## test: vet and run the Go tests with the race detector
test:
	go vet ./...
	go test -race ./...

## bench: the Go benchmarks of the trees
bench:
	go test -run x -bench . ./trees

## web: build the page into site/
web:
	rm -rf $(SITE) && mkdir -p $(SITE)
	cp web/index.html web/style.css web/app.js web/draw.js $(SITE)/
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(SITE)/
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o $(SITE)/forest.wasm ./cmd/forest-wasm

## serve: build the page and serve it on http://localhost:$(PORT)
serve: web
	python3 -m http.server $(PORT) --directory $(SITE)

## gif: make docs/forest.gif again
gif:
	go run ./cmd/forestgif -out docs/forest.gif

## forest: a tree in the terminal
forest:
	go run ./cmd/forest

clean:
	rm -rf $(SITE)
