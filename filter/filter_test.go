package filter

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"idf-go/fold"
	"idf-go/internal/jsonutil"
	"idf-go/parser"
	"idf-go/types"
)

const fixturesRoot = "../../idf-spec/spec/fixtures/library"
const specRoot = "../../idf-spec/spec"

func pairsToTest() []struct {
	scenario string
	viewer   types.Viewer
	file     string
} {
	return []struct {
		scenario string
		viewer   types.Viewer
		file     string
	}{
		{"empty", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "empty-as-librarian-u-lib-1"},
		{"empty", types.Viewer{Role: "reader", ID: "u-r1"}, "empty-as-reader-u-r1"},
		{"bootstrap", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "bootstrap-as-librarian-u-lib-1"},
		{"bootstrap", types.Viewer{Role: "reader", ID: "u-r1"}, "bootstrap-as-reader-u-r1"},
		{"register-readers", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "register-readers-as-librarian-u-lib-1"},
		{"register-readers", types.Viewer{Role: "reader", ID: "u-r1"}, "register-readers-as-reader-u-r1"},
		{"register-readers", types.Viewer{Role: "reader", ID: "u-r2"}, "register-readers-as-reader-u-r2"},
		{"borrow-cycle", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "borrow-cycle-as-librarian-u-lib-1"},
		{"borrow-cycle", types.Viewer{Role: "reader", ID: "u-r1"}, "borrow-cycle-as-reader-u-r1"},
		{"borrow-cycle", types.Viewer{Role: "reader", ID: "u-r2"}, "borrow-cycle-as-reader-u-r2"},
		{"borrow-and-return", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "borrow-and-return-as-librarian-u-lib-1"},
		{"borrow-and-return", types.Viewer{Role: "reader", ID: "u-r1"}, "borrow-and-return-as-reader-u-r1"},
		{"borrow-and-return", types.Viewer{Role: "reader", ID: "u-r2"}, "borrow-and-return-as-reader-u-r2"},
		{"cancel-loan", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "cancel-loan-as-librarian-u-lib-1"},
		{"cancel-loan", types.Viewer{Role: "reader", ID: "u-r1"}, "cancel-loan-as-reader-u-r1"},
		{"cancel-loan", types.Viewer{Role: "reader", ID: "u-r2"}, "cancel-loan-as-reader-u-r2"},
		{"update-book", types.Viewer{Role: "librarian", ID: "u-lib-1"}, "update-book-as-librarian-u-lib-1"},
		{"update-book", types.Viewer{Role: "reader", ID: "u-r1"}, "update-book-as-reader-u-r1"},
	}
}

func TestFilterWorldForRole(t *testing.T) {
	schemas := parser.DefaultSchemaSet(specRoot)
	ontData, _ := parser.ReadFile(filepath.Join(fixturesRoot, "ontology.json"))
	ont, err := parser.ParseOntology(ontData, schemas)
	if err != nil {
		t.Fatalf("parse ontology: %v", err)
	}

	for _, tc := range pairsToTest() {
		t.Run(tc.file, func(t *testing.T) {
			phiData, err := parser.ReadFile(filepath.Join(fixturesRoot, "phi", tc.scenario+".json"))
			if err != nil {
				t.Fatalf("read phi: %v", err)
			}
			phi, _ := parser.ParsePhi(phiData, schemas)
			world, err := fold.Fold(phi, ont)
			if err != nil {
				t.Fatalf("fold: %v", err)
			}
			vw := FilterWorldForRole(world, tc.viewer, ont)

			expData, err := parser.ReadFile(filepath.Join(fixturesRoot, "expected/viewer-world", tc.file+".json"))
			if err != nil {
				t.Fatalf("read expected: %v", err)
			}
			var exp map[string]any
			_ = json.Unmarshal(expData, &exp)
			expVW := exp["viewerWorld"]

			gotJSON, _ := json.Marshal(vw)
			var gotMap any
			_ = json.Unmarshal(gotJSON, &gotMap)

			if !jsonutil.SemanticEqual(gotMap, expVW) {
				gotPretty, _ := json.MarshalIndent(gotMap, "", "  ")
				expPretty, _ := json.MarshalIndent(expVW, "", "  ")
				t.Errorf("viewerWorld mismatch for %s\ngot:\n%s\nwant:\n%s", tc.file, gotPretty, expPretty)
			}
		})
	}
}
