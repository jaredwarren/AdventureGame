


go run ./cmd/gentmj -o assets/maps/maze.tmj -w 12 -h 10 -seed 12345 -style prim
go run ./cmd/gentmj -o /tmp/x.tmj -w 8 -h 8 -seed 1 -style blend -blend 0.3 -extra 0.08

make gentmj                           # defaults: OUT=assets/maps/proc_maze.tmj MW=12 MH=10 SEED=0
make gentmj OUT=assets/maps/foo.tmj SEED=99 MW=15 MH=12

go run ./cmd/gentmj -o assets/maps/maze2.tmj -w 12 -h 10 -seed 1 \
  -floor-grass 0.5 -floor-water 0.2 -floor-floor2 0.3 -tree-deadend 0.35
# No trees:
go run ./cmd/gentmj -o /tmp/m.tmj -w 8 -h 8 -tree-deadend 0

