package parser

import (
	"path/filepath"
	"testing"
)

const fixturesRoot = "../../idf-spec/spec/fixtures/library"
const specRoot = "../../idf-spec/spec"

func TestParseOntology(t *testing.T) {
	schemas := DefaultSchemaSet(specRoot)
	data, err := ReadFile(filepath.Join(fixturesRoot, "ontology.json"))
	if err != nil {
		t.Fatalf("read ontology.json: %v", err)
	}
	ont, err := ParseOntology(data, schemas)
	if err != nil {
		t.Fatalf("parse ontology: %v", err)
	}
	if got := len(ont.Entities); got != 3 {
		t.Errorf("entities count: got %d, want 3", got)
	}
	if got := len(ont.Roles); got != 2 {
		t.Errorf("roles count: got %d, want 2", got)
	}
	for _, name := range []string{"User", "Book", "Loan"} {
		if _, ok := ont.Entities[name]; !ok {
			t.Errorf("entity %s missing", name)
		}
	}
	if ont.Entities["Book"].Kind != "reference" {
		t.Errorf("Book.kind: got %q, want reference", ont.Entities["Book"].Kind)
	}
	if ont.Entities["Loan"].OwnerField != "userId" {
		t.Errorf("Loan.ownerField: got %q, want userId", ont.Entities["Loan"].OwnerField)
	}
	if !ont.Roles["librarian"].VisibleFields["User"].All {
		t.Errorf("librarian.User: ожидается All=true (visibleFields '*')")
	}
	rUser := ont.Roles["reader"].VisibleFields["User"]
	if rUser.All || len(rUser.Fields) != 2 {
		t.Errorf("reader.User: ожидается [id, name], got All=%v Fields=%v", rUser.All, rUser.Fields)
	}

	// FieldsOrder — критично для primary/secondary heuristic в crystallize
	expectedOrder := map[string][]string{
		"User": {"id", "name"},
		"Book": {"id", "title", "author", "isbn"},
		"Loan": {"id", "userId", "bookId", "status", "borrowedAt", "returnedAt"},
	}
	for name, want := range expectedOrder {
		got := ont.Entities[name].FieldsOrder
		if len(got) != len(want) {
			t.Errorf("entity %s FieldsOrder len: got %d (%v), want %d (%v)", name, len(got), got, len(want), want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("entity %s FieldsOrder[%d]: got %s, want %s", name, i, got[i], want[i])
			}
		}
	}
}

func TestParseIntents(t *testing.T) {
	schemas := DefaultSchemaSet(specRoot)
	data, err := ReadFile(filepath.Join(fixturesRoot, "intents.json"))
	if err != nil {
		t.Fatalf("read intents.json: %v", err)
	}
	intents, err := ParseIntents(data, schemas)
	if err != nil {
		t.Fatalf("parse intents: %v", err)
	}
	if got := len(intents); got != 7 {
		t.Errorf("intents count: got %d, want 7", got)
	}
}

func TestParseProjections(t *testing.T) {
	schemas := DefaultSchemaSet(specRoot)
	data, err := ReadFile(filepath.Join(fixturesRoot, "projections.json"))
	if err != nil {
		t.Fatalf("read projections.json: %v", err)
	}
	projs, err := ParseProjections(data, schemas)
	if err != nil {
		t.Fatalf("parse projections: %v", err)
	}
	if got := len(projs); got != 5 {
		t.Errorf("projections count: got %d, want 5", got)
	}
}

func TestParsePhi(t *testing.T) {
	schemas := DefaultSchemaSet(specRoot)
	scenarios := []struct {
		file  string
		count int
	}{
		{"empty.json", 0},
		{"bootstrap.json", 3},
		{"register-readers.json", 5},
		{"borrow-cycle.json", 6},
		{"borrow-and-return.json", 7},
		{"cancel-loan.json", 7},
		{"update-book.json", 4},
	}
	for _, sc := range scenarios {
		t.Run(sc.file, func(t *testing.T) {
			data, err := ReadFile(filepath.Join(fixturesRoot, "phi", sc.file))
			if err != nil {
				t.Fatalf("read %s: %v", sc.file, err)
			}
			phi, err := ParsePhi(data, schemas)
			if err != nil {
				t.Fatalf("parse phi: %v", err)
			}
			if got := len(phi); got != sc.count {
				t.Errorf("effects count: got %d, want %d", got, sc.count)
			}
		})
	}
}
