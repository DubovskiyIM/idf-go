// Command conformance прогоняет L1+L2 conformance check на указанной
// директории fixtures и печатает human-readable отчёт.
//
// Usage: conformance <path-to-fixtures-dir>
//
// Где path-to-fixtures-dir — каталог в формате:
//
//	<dir>/
//	  ontology.json
//	  intents.json
//	  projections.json
//	  phi/<scenario>.json
//	  expected/world/<scenario>.json
//	  expected/viewer-world/<scenario>-as-<role>-<id>.json
//	  expected/artifact/<scenario>-<projection>-as-<role>-<id>.json
//
// Также требуется, чтобы schema-файлы были в <fixtures-dir>/../../schemas/.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"idf-go/crystallize"
	"idf-go/document"
	"idf-go/filter"
	"idf-go/fold"
	"idf-go/internal/jsonutil"
	"idf-go/parser"
	"idf-go/types"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: conformance <fixtures-dir>")
		os.Exit(2)
	}
	fixturesDir := os.Args[1]

	specRoot, err := filepath.Abs(filepath.Join(fixturesDir, "..", ".."))
	if err != nil {
		fail("compute specRoot: %v", err)
	}
	schemas := parser.DefaultSchemaSet(specRoot)

	allPass := true

	// Step 1: Parser
	fmt.Println("== Step 1: Parser ==")
	ontData, err := parser.ReadFile(filepath.Join(fixturesDir, "ontology.json"))
	if err != nil {
		fail("read ontology: %v", err)
	}
	ont, err := parser.ParseOntology(ontData, schemas)
	if err != nil {
		fmt.Printf("  ontology.json: FAIL — %v\n", err)
		allPass = false
	} else {
		fmt.Println("  ontology.json: OK")
	}

	intentsData, err := parser.ReadFile(filepath.Join(fixturesDir, "intents.json"))
	if err != nil {
		fail("read intents: %v", err)
	}
	intents, err := parser.ParseIntents(intentsData, schemas)
	if err != nil {
		fmt.Printf("  intents.json: FAIL — %v\n", err)
		allPass = false
	} else {
		fmt.Println("  intents.json: OK")
	}

	projsData, err := parser.ReadFile(filepath.Join(fixturesDir, "projections.json"))
	if err != nil {
		fail("read projections: %v", err)
	}
	projs, err := parser.ParseProjections(projsData, schemas)
	if err != nil {
		fmt.Printf("  projections.json: FAIL — %v\n", err)
		allPass = false
	} else {
		fmt.Println("  projections.json: OK")
	}
	projByID := make(map[string]types.Projection, len(projs))
	for _, p := range projs {
		projByID[p.ID] = p
	}

	phiFiles, _ := filepath.Glob(filepath.Join(fixturesDir, "phi", "*.json"))
	sort.Strings(phiFiles)
	scenarios := make(map[string][]types.Effect)
	phiOK := 0
	for _, f := range phiFiles {
		data, err := parser.ReadFile(f)
		if err != nil {
			fmt.Printf("  %s: FAIL read — %v\n", f, err)
			allPass = false
			continue
		}
		phi, err := parser.ParsePhi(data, schemas)
		if err != nil {
			fmt.Printf("  %s: FAIL — %v\n", filepath.Base(f), err)
			allPass = false
			continue
		}
		scenarios[basename(f)] = phi
		phiOK++
	}
	fmt.Printf("  phi/*.json: %d/%d OK\n", phiOK, len(phiFiles))

	// Step 2: fold
	fmt.Println("== Step 2: fold ==")
	worlds := make(map[string]types.World)
	foldPass := 0
	scenarioNames := sortedKeys(scenarios)
	for _, sc := range scenarioNames {
		world, err := fold.Fold(scenarios[sc], ont)
		if err != nil {
			fmt.Printf("  %s: FAIL fold — %v\n", sc, err)
			allPass = false
			continue
		}
		worlds[sc] = world

		expData, err := parser.ReadFile(filepath.Join(fixturesDir, "expected/world", sc+".json"))
		if err != nil {
			fmt.Printf("  %s: SKIP no expected — %v\n", sc, err)
			continue
		}
		var exp map[string]any
		_ = json.Unmarshal(expData, &exp)
		gotJSON, _ := json.Marshal(world)
		var gotMap any
		_ = json.Unmarshal(gotJSON, &gotMap)
		if !jsonutil.SemanticEqual(gotMap, exp["world"]) {
			fmt.Printf("  %s: FAIL world mismatch\n", sc)
			allPass = false
			continue
		}
		foldPass++
	}
	fmt.Printf("  %d/%d scenarios: world matches expected\n", foldPass, len(scenarios))

	// Step 3: filterWorldForRole
	fmt.Println("== Step 3: filterWorldForRole ==")
	vwFiles, _ := filepath.Glob(filepath.Join(fixturesDir, "expected/viewer-world", "*.json"))
	sort.Strings(vwFiles)
	vwPass := 0
	for _, f := range vwFiles {
		base := basename(f)
		scenario, viewer, ok := parseVWName(base, ont)
		if !ok {
			fmt.Printf("  %s: SKIP unparsable name\n", base)
			continue
		}
		world, ok := worlds[scenario]
		if !ok {
			fmt.Printf("  %s: SKIP no world for %s\n", base, scenario)
			continue
		}
		got := filter.FilterWorldForRole(world, viewer, ont)

		expData, _ := parser.ReadFile(f)
		var exp map[string]any
		_ = json.Unmarshal(expData, &exp)
		gotJSON, _ := json.Marshal(got)
		var gotMap any
		_ = json.Unmarshal(gotJSON, &gotMap)
		if !jsonutil.SemanticEqual(gotMap, exp["viewerWorld"]) {
			fmt.Printf("  %s: FAIL viewerWorld mismatch\n", base)
			allPass = false
			continue
		}
		vwPass++
	}
	fmt.Printf("  %d/%d (scenario × viewer): viewerWorld matches expected\n", vwPass, len(vwFiles))

	// Step 4: crystallize
	fmt.Println("== Step 4: crystallize ==")
	artFiles, _ := filepath.Glob(filepath.Join(fixturesDir, "expected/artifact", "*.json"))
	sort.Strings(artFiles)
	artPass := 0
	for _, f := range artFiles {
		base := basename(f)
		scenario, projID, viewer, ok := parseArtName(base, ont, projByID)
		if !ok {
			fmt.Printf("  %s: SKIP unparsable name\n", base)
			continue
		}
		world, ok := worlds[scenario]
		if !ok {
			fmt.Printf("  %s: SKIP no world for %s\n", base, scenario)
			continue
		}
		vw := filter.FilterWorldForRole(world, viewer, ont)
		proj := projByID[projID]
		art, err := crystallize.Crystallize(intents, ont, proj, viewer, vw)
		if err != nil {
			fmt.Printf("  %s: FAIL crystallize — %v\n", base, err)
			allPass = false
			continue
		}
		expData, _ := parser.ReadFile(f)
		var exp map[string]any
		_ = json.Unmarshal(expData, &exp)
		gotJSON, _ := json.Marshal(art)
		var gotMap map[string]any
		_ = json.Unmarshal(gotJSON, &gotMap)
		if !jsonutil.SemanticEqual(gotMap, exp) {
			fmt.Printf("  %s: FAIL artifact mismatch\n", base)
			allPass = false
			continue
		}
		artPass++
	}
	fmt.Printf("  %d/%d (scenario × projection × viewer): artifact matches expected\n", artPass, len(artFiles))

	// Step 5: document materialization (L3, с spec v0.2.0)
	fmt.Println("== Step 5: materializeAsDocument ==")
	docDir := filepath.Join(fixturesDir, "expected/document")
	docFiles, _ := filepath.Glob(filepath.Join(docDir, "*.json"))
	sort.Strings(docFiles)
	docPass := 0
	docLevel := "L3"
	if len(docFiles) == 0 {
		fmt.Println("  (no expected/document fixtures — L3 skipped)")
		docLevel = ""
	}
	for _, f := range docFiles {
		base := basename(f)
		scenario, projID, viewer, ok := parseArtName(base, ont, projByID)
		if !ok {
			fmt.Printf("  %s: SKIP unparsable name\n", base)
			continue
		}
		world, ok := worlds[scenario]
		if !ok {
			continue
		}
		vw := filter.FilterWorldForRole(world, viewer, ont)
		proj := projByID[projID]
		art, err := crystallize.Crystallize(intents, ont, proj, viewer, vw)
		if err != nil {
			fmt.Printf("  %s: FAIL crystallize — %v\n", base, err)
			allPass = false
			continue
		}
		doc, err := document.MaterializeAsDocument(art, vw, ont)
		if err != nil {
			fmt.Printf("  %s: FAIL document — %v\n", base, err)
			allPass = false
			continue
		}
		expData, _ := parser.ReadFile(f)
		var exp map[string]any
		_ = json.Unmarshal(expData, &exp)
		gotJSON, _ := json.Marshal(doc)
		var gotMap map[string]any
		_ = json.Unmarshal(gotJSON, &gotMap)
		if !jsonutil.SemanticEqual(gotMap, exp) {
			fmt.Printf("  %s: FAIL document mismatch\n", base)
			allPass = false
			continue
		}
		docPass++
	}
	if docLevel != "" {
		fmt.Printf("  %d/%d (scenario × projection × viewer): document matches expected\n", docPass, len(docFiles))
	}

	fmt.Println()
	if allPass {
		if docLevel == "L3" {
			fmt.Println("== OVERALL: L1+L2+L3(document) CONFORMANT ==")
		} else {
			fmt.Println("== OVERALL: L1+L2 CONFORMANT ==")
		}
		os.Exit(0)
	}
	fmt.Println("== OVERALL: FAILURES ==")
	os.Exit(1)
}

func basename(f string) string {
	b := filepath.Base(f)
	return strings.TrimSuffix(b, ".json")
}

func sortedKeys(m map[string][]types.Effect) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// parseVWName разбирает <scenario>-as-<role>-<id> в (scenario, viewer).
func parseVWName(name string, ont types.Ontology) (string, types.Viewer, bool) {
	const sep = "-as-"
	idx := strings.Index(name, sep)
	if idx < 0 {
		return "", types.Viewer{}, false
	}
	scenario := name[:idx]
	rest := name[idx+len(sep):]
	for roleName := range ont.Roles {
		prefix := roleName + "-"
		if strings.HasPrefix(rest, prefix) {
			return scenario, types.Viewer{
				Role: roleName,
				ID:   rest[len(prefix):],
			}, true
		}
	}
	return "", types.Viewer{}, false
}

// parseArtName разбирает <scenario>-<projection>-as-<role>-<id>.
func parseArtName(name string, ont types.Ontology, projByID map[string]types.Projection) (string, string, types.Viewer, bool) {
	const sep = "-as-"
	idx := strings.Index(name, sep)
	if idx < 0 {
		return "", "", types.Viewer{}, false
	}
	prefix := name[:idx]
	rest := name[idx+len(sep):]
	for projID := range projByID {
		suffix := "-" + projID
		if strings.HasSuffix(prefix, suffix) {
			scenario := prefix[:len(prefix)-len(suffix)]
			for roleName := range ont.Roles {
				rolePrefix := roleName + "-"
				if strings.HasPrefix(rest, rolePrefix) {
					return scenario, projID, types.Viewer{
						Role: roleName,
						ID:   rest[len(rolePrefix):],
					}, true
				}
			}
		}
	}
	return "", "", types.Viewer{}, false
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
