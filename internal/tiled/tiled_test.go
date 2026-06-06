package tiled

import "testing"

func TestObjPropBool(t *testing.T) {
	t.Parallel()
	o := &Object{
		Properties: []Property{
			{Name: "a", Type: "bool", Value: true},
			{Name: "b", Type: "string", Value: "true"},
			{Name: "c", Type: "string", Value: "false"},
		},
	}
	if v, ok := ObjPropBool(o, "a"); !ok || !v {
		t.Fatalf("a: got %v %v", v, ok)
	}
	if v, ok := ObjPropBool(o, "b"); !ok || !v {
		t.Fatalf("b: got %v %v", v, ok)
	}
	if v, ok := ObjPropBool(o, "c"); !ok || v {
		t.Fatalf("c: got %v %v", v, ok)
	}
	if _, ok := ObjPropBool(o, "missing"); ok {
		t.Fatal("expected missing")
	}
}

func TestObjPropNumeric(t *testing.T) {
	t.Parallel()
	o := &Object{
		Properties: []Property{
			{Name: "hp", Type: "int", Value: float64(7)},
			{Name: "speed", Type: "float", Value: 0.55},
			{Name: "aggro", Type: "string", Value: "128"},
		},
	}
	if v, ok := ObjPropInt(o, "hp"); !ok || v != 7 {
		t.Fatalf("hp: got %d %v", v, ok)
	}
	if v, ok := ObjPropFloat(o, "speed"); !ok || v != 0.55 {
		t.Fatalf("speed: got %v %v", v, ok)
	}
	if v, ok := ObjPropFloat(o, "aggro"); !ok || v != 128 {
		t.Fatalf("aggro: got %v %v", v, ok)
	}
	if _, ok := ObjPropInt(o, "missing"); ok {
		t.Fatal("expected missing int")
	}
}
