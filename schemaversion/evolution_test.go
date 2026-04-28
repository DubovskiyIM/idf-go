package schemaversion

import (
	"strings"
	"testing"
)

func ptr(s string) *string { return &s }

func TestParseEvolution_Missing(t *testing.T) {
	got, err := ParseEvolution(map[string]any{"entities": map[string]any{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for missing evolution, got %v", got)
	}
}

func TestParseEvolution_NotMap(t *testing.T) {
	got, err := ParseEvolution(nil)
	if err != nil || got != nil {
		t.Fatalf("expected (nil, nil) for non-map, got (%v, %v)", got, err)
	}
}

func TestParseEvolution_Empty(t *testing.T) {
	got, err := ParseEvolution(map[string]any{"evolution": []any{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

func TestParseEvolution_TwoEntries(t *testing.T) {
	got, err := ParseEvolution(map[string]any{
		"evolution": []any{
			map[string]any{
				"hash":       "1a23f3f820e80b",
				"parentHash": nil,
				"timestamp":  "2026-04-28T10:00:00Z",
				"authorId":   "alice",
			},
			map[string]any{
				"hash":       "abc12300000000",
				"parentHash": "1a23f3f820e80b",
				"timestamp":  "2026-04-29T11:00:00Z",
				"authorId":   "bob",
				"upcasters": []any{
					map[string]any{
						"fromHash": "1a23f3f820e80b",
						"toHash":   "abc12300000000",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].ParentHash != nil {
		t.Fatalf("entry[0] parentHash should be nil, got %v", got[0].ParentHash)
	}
	if got[1].ParentHash == nil || *got[1].ParentHash != "1a23f3f820e80b" {
		t.Fatalf("entry[1] parentHash mismatch")
	}
	if len(got[1].Upcasters) != 1 || got[1].Upcasters[0].FromHash != "1a23f3f820e80b" {
		t.Fatalf("entry[1] upcaster mismatch")
	}
}

func TestValidateEvolutionLog_Empty(t *testing.T) {
	if errs := ValidateEvolutionLog(nil); len(errs) != 0 {
		t.Fatalf("empty log should pass: %v", errs)
	}
}

func TestValidateEvolutionLog_Valid_RootOnly(t *testing.T) {
	log := []EvolutionEntry{
		{Hash: "1a23f3f820e80b", ParentHash: nil, Timestamp: "t", AuthorID: "a"},
	}
	if errs := ValidateEvolutionLog(log); len(errs) != 0 {
		t.Fatalf("valid root should pass: %v", errs)
	}
}

func TestValidateEvolutionLog_Valid_TwoEntries(t *testing.T) {
	log := []EvolutionEntry{
		{Hash: "1a23f3f820e80b", ParentHash: nil, Timestamp: "t", AuthorID: "a"},
		{Hash: "abc12300000000", ParentHash: ptr("1a23f3f820e80b"), Timestamp: "t", AuthorID: "b"},
	}
	if errs := ValidateEvolutionLog(log); len(errs) != 0 {
		t.Fatalf("valid 2-entry log should pass: %v", errs)
	}
}

func TestValidateEvolutionLog_BadHash(t *testing.T) {
	log := []EvolutionEntry{
		{Hash: "not-hex!", ParentHash: nil, Timestamp: "t", AuthorID: "a"},
	}
	errs := ValidateEvolutionLog(log)
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "pattern") {
		t.Fatalf("expected pattern error, got %v", errs)
	}
}

func TestValidateEvolutionLog_DuplicateHash(t *testing.T) {
	log := []EvolutionEntry{
		{Hash: "1a23f3f820e80b", ParentHash: nil, Timestamp: "t", AuthorID: "a"},
		{Hash: "1a23f3f820e80b", ParentHash: ptr("1a23f3f820e80b"), Timestamp: "t", AuthorID: "b"},
	}
	errs := ValidateEvolutionLog(log)
	dup := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "duplicates") {
			dup = true
		}
	}
	if !dup {
		t.Fatalf("expected duplicates error, got %v", errs)
	}
}

func TestValidateEvolutionLog_NoRoot(t *testing.T) {
	log := []EvolutionEntry{
		{Hash: "1a23f3f820e80b", ParentHash: ptr("abc12300000000"), Timestamp: "t", AuthorID: "a"},
	}
	errs := ValidateEvolutionLog(log)
	noRoot := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "no root") {
			noRoot = true
		}
	}
	if !noRoot {
		t.Fatalf("expected no-root error, got %v", errs)
	}
}

func TestValidateEvolutionLog_TwoRoots(t *testing.T) {
	log := []EvolutionEntry{
		{Hash: "1a23f3f820e80b", ParentHash: nil, Timestamp: "t", AuthorID: "a"},
		{Hash: "abc12300000000", ParentHash: nil, Timestamp: "t", AuthorID: "b"},
	}
	errs := ValidateEvolutionLog(log)
	twoRoots := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "2 root") {
			twoRoots = true
		}
	}
	if !twoRoots {
		t.Fatalf("expected 2-roots error, got %v", errs)
	}
}

func TestValidateEvolutionLog_DanglingParent(t *testing.T) {
	log := []EvolutionEntry{
		{Hash: "1a23f3f820e80b", ParentHash: nil, Timestamp: "t", AuthorID: "a"},
		{Hash: "abc12300000000", ParentHash: ptr("dead00000000ff"), Timestamp: "t", AuthorID: "b"},
	}
	errs := ValidateEvolutionLog(log)
	dangling := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "non-existent") {
			dangling = true
		}
	}
	if !dangling {
		t.Fatalf("expected non-existent parent error, got %v", errs)
	}
}

func TestValidateEvolutionLog_Cycle(t *testing.T) {
	// A → B → A (cycle, no root)
	log := []EvolutionEntry{
		{Hash: "aaaaaaaaaaaaaa", ParentHash: ptr("bbbbbbbbbbbbbb"), Timestamp: "t", AuthorID: "a"},
		{Hash: "bbbbbbbbbbbbbb", ParentHash: ptr("aaaaaaaaaaaaaa"), Timestamp: "t", AuthorID: "b"},
	}
	errs := ValidateEvolutionLog(log)
	// Сначала будет 'no root' (это валидный цикл sigh), плюс cycle detection
	cycleOrNoRoot := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "cycle") || strings.Contains(e.Error(), "no root") {
			cycleOrNoRoot = true
		}
	}
	if !cycleOrNoRoot {
		t.Fatalf("expected cycle or no-root error, got %v", errs)
	}
}

func TestRehashAndVerifyRoot_Match(t *testing.T) {
	ontology := map[string]any{"a": 1.0, "b": 2.0, "c": 3.0}
	hash := HashOntology(ontology)
	log := []EvolutionEntry{
		{Hash: hash, ParentHash: nil, Timestamp: "t", AuthorID: "a"},
	}
	matches, expected, got := RehashAndVerifyRoot(log, ontology)
	if !matches {
		t.Fatalf("expected match: expected=%s got=%s", expected, got)
	}
}

func TestRehashAndVerifyRoot_Drift(t *testing.T) {
	log := []EvolutionEntry{
		{Hash: "1111111111111e", ParentHash: nil, Timestamp: "t", AuthorID: "a"},
	}
	matches, _, got := RehashAndVerifyRoot(log, map[string]any{"a": 1.0, "b": 2.0, "c": 3.0})
	if matches {
		t.Fatalf("expected drift, got match")
	}
	if got != "1111111111111e" {
		t.Fatalf("got mismatch: %s", got)
	}
}

func TestRehashAndVerifyRoot_NoRoot(t *testing.T) {
	matches, _, _ := RehashAndVerifyRoot(nil, map[string]any{})
	if matches {
		t.Fatalf("empty log should return false")
	}
}
