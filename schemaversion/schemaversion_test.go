// Тесты против нормативных vectors из idf-spec.
//
// Локация vectors-файла резолвится относительно git repo root, чтобы тест
// работал и из main checkout, и из worktree. Если файл не найден —
// тест skip'ается (не fail'ится), потому что spec — внешний репо.
package schemaversion

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type vectorEntry struct {
	Kind     string `json:"kind"`
	Input    any    `json:"input"`
	Expected string `json:"expected"`
}

type vectorFile struct {
	Vectors []vectorEntry `json:"vectors"`
}

func loadVectors(t *testing.T) []vectorEntry {
	t.Helper()

	// Ищем vectors-файл в нескольких типичных расположениях относительно
	// idf-go checkout.
	candidates := []string{
		// Из main checkout idf-go/schemaversion: ../../ = sibling
		"../../idf-spec/spec/schemas/hash-function.vectors.json",
		// Из worktree idf-go/.worktrees/<name>/schemaversion: ../../../../ = sibling
		"../../../../idf-spec/spec/schemas/hash-function.vectors.json",
		// idf-spec ещё в worktree (до merge spec PR'а):
		"../../idf-spec/.worktrees/hash-function/spec/schemas/hash-function.vectors.json",
		"../../../../idf-spec/.worktrees/hash-function/spec/schemas/hash-function.vectors.json",
	}
	cwd, _ := os.Getwd()
	for _, rel := range candidates {
		p := filepath.Join(cwd, rel)
		if data, err := os.ReadFile(p); err == nil {
			var v vectorFile
			if err := json.Unmarshal(data, &v); err == nil {
				t.Logf("loaded %d vectors from %s", len(v.Vectors), p)
				return v.Vectors
			}
		}
	}
	t.Skipf("hash-function.vectors.json not found in any candidate path; skipping cross-stack vector test")
	return nil
}

func TestVectors_Cyrb53String(t *testing.T) {
	for _, v := range loadVectors(t) {
		if v.Kind != "cyrb53-string" {
			continue
		}
		input, ok := v.Input.(string)
		if !ok {
			t.Fatalf("expected string input for cyrb53-string vector, got %T", v.Input)
		}
		t.Run(fmt.Sprintf("cyrb53(%q)", input), func(t *testing.T) {
			h := Cyrb53(input, 0)
			got := fmt.Sprintf("%014x", h)
			if got != v.Expected {
				t.Fatalf("Cyrb53(%q): got %s, expected %s", input, got, v.Expected)
			}
		})
	}
}

func TestVectors_HashOntology(t *testing.T) {
	for _, v := range loadVectors(t) {
		if v.Kind != "hashOntology" {
			continue
		}
		t.Run(fmt.Sprintf("%v", v.Input), func(t *testing.T) {
			got := HashOntology(v.Input)
			if got != v.Expected {
				t.Fatalf("HashOntology(%v): got %s, expected %s", v.Input, got, v.Expected)
			}
		})
	}
}

// Sanity — internal invariants без зависимости от vectors-файла.

func TestHashOntology_Nil(t *testing.T) {
	if got := HashOntology(nil); got != "00000000000000" {
		t.Fatalf("HashOntology(nil): got %s", got)
	}
}

func TestHashOntology_OrderIndependent(t *testing.T) {
	a := map[string]any{"a": 1.0, "b": 2.0, "c": 3.0}
	b := map[string]any{"c": 3.0, "a": 1.0, "b": 2.0}
	if HashOntology(a) != HashOntology(b) {
		t.Fatalf("HashOntology should be order-independent for object keys")
	}
}

func TestHashOntology_ArrayOrderMatters(t *testing.T) {
	a := map[string]any{"roles": []any{"admin", "viewer"}}
	b := map[string]any{"roles": []any{"viewer", "admin"}}
	if HashOntology(a) == HashOntology(b) {
		t.Fatalf("HashOntology should distinguish array order")
	}
}

func TestGetSchemaVersion(t *testing.T) {
	cases := []struct {
		name   string
		effect map[string]any
		want   string
	}{
		{"nil", nil, UnknownSchemaVersion},
		{"empty", map[string]any{}, UnknownSchemaVersion},
		{"no-context", map[string]any{"id": "e1"}, UnknownSchemaVersion},
		{"empty-context", map[string]any{"context": map[string]any{}}, UnknownSchemaVersion},
		{"empty-string", map[string]any{"context": map[string]any{"schemaVersion": ""}}, UnknownSchemaVersion},
		{"valid", map[string]any{"context": map[string]any{"schemaVersion": "abc123"}}, "abc123"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GetSchemaVersion(c.effect); got != c.want {
				t.Fatalf("GetSchemaVersion: got %q, want %q", got, c.want)
			}
		})
	}
}

func TestTagWithSchemaVersion_Pure(t *testing.T) {
	original := map[string]any{
		"id":      "e1",
		"context": map[string]any{"id": "row-1"},
	}
	tagged := TagWithSchemaVersion(original, "v123")
	if v := GetSchemaVersion(tagged); v != "v123" {
		t.Fatalf("tagged should carry version, got %q", v)
	}
	if v := GetSchemaVersion(original); v != UnknownSchemaVersion {
		t.Fatalf("original should remain untouched, got %q", v)
	}
	// Existing context fields preserved
	if id, _ := tagged["context"].(map[string]any)["id"].(string); id != "row-1" {
		t.Fatalf("tagged context should preserve existing id field")
	}
}
