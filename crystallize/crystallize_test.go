package crystallize

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"idf-go/filter"
	"idf-go/fold"
	"idf-go/internal/jsonutil"
	"idf-go/parser"
	"idf-go/types"
)

const fixturesRoot = "../../idf-spec/spec/fixtures/library"
const specRoot = "../../idf-spec/spec"

func TestCrystallize(t *testing.T) {
	schemas := parser.DefaultSchemaSet(specRoot)

	ontData, _ := parser.ReadFile(filepath.Join(fixturesRoot, "ontology.json"))
	ont, err := parser.ParseOntology(ontData, schemas)
	if err != nil {
		t.Fatalf("parse ontology: %v", err)
	}

	intentsData, _ := parser.ReadFile(filepath.Join(fixturesRoot, "intents.json"))
	intents, err := parser.ParseIntents(intentsData, schemas)
	if err != nil {
		t.Fatalf("parse intents: %v", err)
	}

	projsData, _ := parser.ReadFile(filepath.Join(fixturesRoot, "projections.json"))
	projs, err := parser.ParseProjections(projsData, schemas)
	if err != nil {
		t.Fatalf("parse projections: %v", err)
	}
	projByID := map[string]types.Projection{}
	for _, p := range projs {
		projByID[p.ID] = p
	}

	cases := []struct {
		scenario   string
		projection string
		viewer     types.Viewer
		file       string
	}{
		{"bootstrap", "book-catalog", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "bootstrap-book-catalog-as-librarian-u-lib-1"},
		{"bootstrap", "book-catalog", types.Viewer{Role: "reader", ID: "u-r1"}, "bootstrap-book-catalog-as-reader-u-r1"},
		{"bootstrap", "book-detail", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "bootstrap-book-detail-as-librarian-u-lib-1"},
		{"bootstrap", "book-detail", types.Viewer{Role: "reader", ID: "u-r1"}, "bootstrap-book-detail-as-reader-u-r1"},
		{"bootstrap", "borrow-form", types.Viewer{Role: "reader", ID: "u-r1"}, "bootstrap-borrow-form-as-reader-u-r1"},
		{"borrow-cycle", "my-loans", types.Viewer{Role: "reader", ID: "u-r1"}, "borrow-cycle-my-loans-as-reader-u-r1"},
		{"borrow-cycle", "my-loans", types.Viewer{Role: "reader", ID: "u-r2"}, "borrow-cycle-my-loans-as-reader-u-r2"},
		{"borrow-cycle", "librarian-dashboard", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "borrow-cycle-librarian-dashboard-as-librarian-u-lib-1"},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			phiData, _ := parser.ReadFile(filepath.Join(fixturesRoot, "phi", tc.scenario+".json"))
			phi, _ := parser.ParsePhi(phiData, schemas)
			world, err := fold.Fold(phi, ont)
			if err != nil {
				t.Fatalf("fold: %v", err)
			}
			vw := filter.FilterWorldForRole(world, tc.viewer, ont)
			proj := projByID[tc.projection]

			art, err := Crystallize(intents, ont, proj, tc.viewer, vw)
			if err != nil {
				t.Fatalf("crystallize: %v", err)
			}

			expData, err := parser.ReadFile(filepath.Join(fixturesRoot, "expected/artifact", tc.file+".json"))
			if err != nil {
				t.Fatalf("read expected: %v", err)
			}
			var exp map[string]any
			_ = json.Unmarshal(expData, &exp)

			gotJSON, _ := json.Marshal(art)
			var gotMap map[string]any
			_ = json.Unmarshal(gotJSON, &gotMap)

			if !jsonutil.SemanticEqual(gotMap, exp) {
				gotPretty, _ := json.MarshalIndent(gotMap, "", "  ")
				expPretty, _ := json.MarshalIndent(exp, "", "  ")
				t.Errorf("artifact mismatch for %s\ngot:\n%s\nwant:\n%s", tc.file, gotPretty, expPretty)
			}
		})
	}
}
