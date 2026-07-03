.PHONY: run test fmt vet check clean maps-check build edit edit-% gentmj

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

# Random maze → .tmj (Growing Tree + natural floor paint). Example: make gentmj OUT=assets/maps/maze.tmj SEED=42
# For -floor-* / -tree-deadend flags run: go run ./cmd/gentmj -h
OUT ?= assets/maps/proc_maze.tmj
MW ?= 12
MH ?= 10
SEED ?= 0
gentmj:
	go run ./cmd/gentmj -o $(OUT) -w $(MW) -h $(MH) -seed $(SEED)

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
