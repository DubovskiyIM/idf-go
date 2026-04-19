package fold

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"idf-go/internal/jsonutil"
	"idf-go/parser"
)

const fixturesRoot = "../../idf-spec/spec/fixtures/library"
const specRoot = "../../idf-spec/spec"

func TestFold(t *testing.T) {
	schemas := parser.DefaultSchemaSet(specRoot)

	ontData, err := parser.ReadFile(filepath.Join(fixturesRoot, "ontology.json"))
	if err != nil {
		t.Fatalf("read ontology: %v", err)
	}
	ont, err := parser.ParseOntology(ontData, schemas)
	if err != nil {
		t.Fatalf("parse ontology: %v", err)
	}

	scenarios := []string{
		"empty",
		"bootstrap",
		"register-readers",
		"borrow-cycle",
		"borrow-and-return",
		"cancel-loan",
		"update-book",
	}
	for _, sc := range scenarios {
		t.Run(sc, func(t *testing.T) {
			phiData, err := parser.ReadFile(filepath.Join(fixturesRoot, "phi", sc+".json"))
			if err != nil {
				t.Fatalf("read phi: %v", err)
			}
			phi, err := parser.ParsePhi(phiData, schemas)
			if err != nil {
				t.Fatalf("parse phi: %v", err)
			}
			got, err := Fold(phi, ont)
			if err != nil {
				t.Fatalf("Fold: %v", err)
			}

			expData, err := parser.ReadFile(filepath.Join(fixturesRoot, "expected/world", sc+".json"))
			if err != nil {
				t.Fatalf("read expected: %v", err)
			}
			var exp map[string]any
			if err := json.Unmarshal(expData, &exp); err != nil {
				t.Fatalf("unmarshal expected: %v", err)
			}

			expWorld := exp["world"]
			gotJSON, _ := json.Marshal(got)
			var gotMap any
			_ = json.Unmarshal(gotJSON, &gotMap)

			if !jsonutil.SemanticEqual(gotMap, expWorld) {
				gotPretty, _ := json.MarshalIndent(gotMap, "", "  ")
				expPretty, _ := json.MarshalIndent(expWorld, "", "  ")
				t.Errorf("scenario %s: world mismatch\ngot:\n%s\nwant:\n%s", sc, gotPretty, expPretty)
			}
		})
	}
}
