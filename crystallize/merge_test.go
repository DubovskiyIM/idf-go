package crystallize

import (
	"testing"

	"idf-go/internal/jsonutil"
)

func TestMergeSlots(t *testing.T) {
	cases := []struct {
		name          string
		derived, auth map[string]any
		want          map[string]any
	}{
		{
			name:    "empty-authored",
			derived: map[string]any{"header": map[string]any{"title": "X"}},
			auth:    map[string]any{},
			want:    map[string]any{"header": map[string]any{"title": "X"}},
		},
		{
			name:    "scalar-replace",
			derived: map[string]any{"header": map[string]any{"title": "X"}},
			auth:    map[string]any{"header": map[string]any{"title": "Y"}},
			want:    map[string]any{"header": map[string]any{"title": "Y"}},
		},
		{
			name: "array-replace-not-merge",
			derived: map[string]any{
				"body": map[string]any{
					"fields": []any{
						map[string]any{"name": "x", "type": "string"},
					},
				},
			},
			auth: map[string]any{
				"body": map[string]any{
					"_authored": true,
					"fields":    []any{"x"},
				},
			},
			want: map[string]any{
				"body": map[string]any{
					"_authored": true,
					"fields":    []any{"x"},
				},
			},
		},
		{
			name: "deep-merge-objects",
			derived: map[string]any{
				"body": map[string]any{
					"a": "old",
					"b": "keep",
				},
			},
			auth: map[string]any{
				"body": map[string]any{
					"a": "new",
				},
			},
			want: map[string]any{
				"body": map[string]any{
					"a": "new",
					"b": "keep",
				},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := mergeSlots(c.derived, c.auth)
			if !jsonutil.SemanticEqualStrict(got, c.want) {
				t.Errorf("mergeSlots: got %v, want %v", got, c.want)
			}
		})
	}
}
