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
