.PHONY: run test fmt vet check clean maps-check build edit edit-% gentmj world-grid wasm wasm-clean wasm-rebuild wasm-serve wasm-serve-lan

# Map id for the in-engine editor (assets/maps/$(MAP).tmj).
MAP ?= field1

# Goals after `edit` are the map id (e.g. `make edit field2`). Consume them so Make does not
# look for a file named `field2`.
EDIT_MAP :=
ifeq ($(firstword $(MAKECMDGOALS)),edit)
  EDIT_MAP := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
endif
ifneq ($(EDIT_MAP),)
.PHONY: $(EDIT_MAP)
$(EDIT_MAP):
	@:
endif

# Default target
all: test

CMD := ./cmd/game

run:
	go run $(CMD)

# In-engine .tmj editor: `make edit`, `make edit field2`, `make edit MAP=field2`, or `make edit-field2`.
edit:
	go run $(CMD) -edit $(if $(EDIT_MAP),$(firstword $(EDIT_MAP)),$(MAP))

edit-%:
	go run $(CMD) -edit $(patsubst edit-%,%,$@)

sprite:
	go run ./scripts/pickup_atlas_editor

build:
	go build -o bin/game-test $(CMD)

# WASM build for browser testing with virtual touch controls (auto-enabled).
wasm:
	GOOS=js GOARCH=wasm go build -o web/game.wasm $(CMD)
	@WASM_JS="$$(go env GOROOT)/lib/wasm/wasm_exec.js"; \
	if [ ! -f "$$WASM_JS" ]; then WASM_JS="$$(go env GOROOT)/misc/wasm/wasm_exec.js"; fi; \
	cp "$$WASM_JS" web/wasm_exec.js
	@date +%s > web/wasm-build-id.txt
	@echo "WASM build id: $$(cat web/wasm-build-id.txt) (browsers fetch game.wasm?v=<id>)"

wasm-clean:
	rm -f web/game.wasm web/wasm_exec.js web/wasm-build-id.txt

wasm-rebuild: wasm-clean wasm

wasm-serve: wasm
	@echo "Local only: http://127.0.0.1:8080/"
	cd web && python3 -m http.server 8080 --bind 127.0.0.1

# Same as wasm-serve but reachable from other devices on your Wi-Fi (phone, tablet).
wasm-serve-lan: wasm
	@echo "On this Mac:  http://127.0.0.1:8080/"
	@echo "On your phone: http://$$(ipconfig getifaddr en0 2>/dev/null || hostname):8080/"
	@echo "(Phone must be on the same Wi-Fi. Allow incoming connections if macOS firewall prompts.)"
	cd web && python3 -m http.server 8080 --bind 0.0.0.0

# Random maze → .tmj (Growing Tree + natural floor paint). Example: make gentmj OUT=assets/maps/maze.tmj SEED=42
# For -floor-* / -tree-deadend flags run: go run ./cmd/gentmj -h
OUT ?= assets/maps/proc_maze.tmj
MW ?= 12
MH ?= 10
SEED ?= 0
gentmj:
	go run ./cmd/gentmj -o $(OUT) -w $(MW) -h $(MH) -seed $(SEED)

world-grid:
	go run ./cmd/genworldgrid -out assets/maps

check: fmt vet test

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

maps-check:
	@test -f assets/maps/field1.tmj && test -f assets/maps/field2.tmj

clean:
	rm -rf bin/
	rm -f game-test save.json coverage.out
