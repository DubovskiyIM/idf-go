package jsonutil

import "testing"

func TestDeepCopy(t *testing.T) {
	orig := map[string]any{
		"a": 1.0,
		"b": []any{"x", "y"},
		"c": map[string]any{"nested": true},
	}
	cp := DeepCopy(orig).(map[string]any)
	cp["a"] = 999.0
	cp["b"].([]any)[0] = "changed"
	cp["c"].(map[string]any)["nested"] = false

	if orig["a"] != 1.0 {
		t.Errorf("orig.a mutated: %v", orig["a"])
	}
	if orig["b"].([]any)[0] != "x" {
		t.Errorf("orig.b[0] mutated: %v", orig["b"].([]any)[0])
	}
	if orig["c"].(map[string]any)["nested"] != true {
		t.Errorf("orig.c.nested mutated: %v", orig["c"].(map[string]any)["nested"])
	}
}

func TestDeepCopyMap(t *testing.T) {
	if got := DeepCopyMap(nil); got != nil {
		t.Errorf("DeepCopyMap(nil) = %v, want nil", got)
	}
	m := map[string]any{"a": 1.0}
	cp := DeepCopyMap(m)
	cp["a"] = 999.0
	if m["a"] != 1.0 {
		t.Errorf("orig.a mutated: %v", m["a"])
	}
}
