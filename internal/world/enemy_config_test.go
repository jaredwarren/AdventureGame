package world

import (
	"testing"

	"github.com/jaredwarren/game-test/internal/progression"
	"github.com/jaredwarren/game-test/internal/tiled"
)

func TestBuildFromTiled_EnemyProps(t *testing.T) {
	t.Parallel()
	objs := `[{"id":1,"type":"spawn","x":16,"y":32,"width":0,"height":0},
	{"id":2,"type":"enemy","x":48,"y":48,"width":0,"height":0,"properties":[
		{"name":"hp","type":"int","value":7},
		{"name":"speed","type":"float","value":0.9},
		{"name":"aggro","type":"float","value":64},
		{"name":"damage","type":"int","value":4},
		{"name":"is_boss","type":"bool","value":true}
	]}]`
	m, err := tiled.ParseMap([]byte(tinyMapJSON(objs)))
	if err != nil {
		t.Fatal(err)
	}
	w, err := BuildFromTiled(m, "testmap", progression.DefaultStats(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Enemies) != 1 {
		t.Fatalf("expected 1 enemy, got %d", len(w.Enemies))
	}
	e := w.Enemies[0]
	if e.HP != 7 || e.MaxHP != 7 {
		t.Errorf("HP = %d/%d", e.HP, e.MaxHP)
	}
	if e.AI.Speed != 0.9 {
		t.Errorf("Speed = %v", e.AI.Speed)
	}
	if e.AI.AggroRadius != 64 {
		t.Errorf("AggroRadius = %v", e.AI.AggroRadius)
	}
	if e.ContactHurt.Damage != 4 {
		t.Errorf("Damage = %d", e.ContactHurt.Damage)
	}
	if !e.IsBoss {
		t.Error("IsBoss = false")
	}
}
