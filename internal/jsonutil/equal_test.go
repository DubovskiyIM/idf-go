package jsonutil

import "testing"

func TestSemanticEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b any
		want bool
	}{
		{"nil-nil", nil, nil, true},
		{"scalar-eq", "hello", "hello", true},
		{"scalar-neq", "hello", "world", false},
		{"int-as-float", float64(42), float64(42), true},
		{"empty-map", map[string]any{}, map[string]any{}, true},
		{"map-key-order",
			map[string]any{"a": 1.0, "b": 2.0},
			map[string]any{"b": 2.0, "a": 1.0},
			true,
		},
		{"map-extra-key",
			map[string]any{"a": 1.0},
			map[string]any{"a": 1.0, "b": 2.0},
			false,
		},
		{"array-order-matters",
			[]any{1.0, 2.0},
			[]any{2.0, 1.0},
			false,
		},
		{"array-equal", []any{1.0, 2.0}, []any{1.0, 2.0}, true},
		{"meta-ignored",
			map[string]any{"_meta": "x", "v": 1.0},
			map[string]any{"_meta": "y", "v": 1.0},
			true,
		},
		{"meta-nested-ignored",
			map[string]any{"outer": map[string]any{"_meta": "x", "v": 1.0}},
			map[string]any{"outer": map[string]any{"_meta": "y", "v": 1.0}},
			true,
		},
		{"nested",
			map[string]any{"x": map[string]any{"y": []any{1.0, 2.0}}},
			map[string]any{"x": map[string]any{"y": []any{1.0, 2.0}}},
			true,
		},
		{"type-mismatch",
			map[string]any{"a": 1.0},
			[]any{1.0},
			false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SemanticEqual(c.a, c.b); got != c.want {
				t.Errorf("SemanticEqual(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}
