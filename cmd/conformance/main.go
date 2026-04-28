// Command conformance прогоняет L1+L2 conformance check на указанной
// директории fixtures и печатает human-readable отчёт.
//
// Usage:
//
//	conformance <fixtures-dir>                # default mode: compare to expected/*
//	conformance <fixtures-dir> --emit <dir>   # emit JSON outputs (для cross-stack diff)
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
//	  expected/document/<scenario>-<projection>-as-<role>-<id>.json
//
// В --emit режиме CLI не сравнивает выход с expected/* — он повторяет ту же
// итерацию (scenario × viewer × projection из expected-директорий как
// каноничного списка триплетов) и пишет каждый артефакт под идентичным
// basename'ом в `<emit-dir>/{world,viewer-world,artifact,document}/<basename>.json`.
// Cross-stack harness потом байт-сравнивает (или semantic-сравнивает) эти
// директории попарно между stack'ами.
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
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: conformance <fixtures-dir> [--emit <out-dir>]")
		os.Exit(2)
	}
	fixturesDir := args[0]
	emitDir := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--emit":
			if i+1 >= len(args) {
				fail("--emit requires <out-dir>")
			}
			emitDir = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "--emit=") {
				emitDir = strings.TrimPrefix(args[i], "--emit=")
			} else {
				fail("unknown arg: %s", args[i])
			}
		}
	}
	emitting := emitDir != ""

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

	if emitting {
		if err := os.MkdirAll(emitDir, 0o755); err != nil {
			fail("mkdir emit-dir: %v", err)
		}
	}

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

		if emitting {
			if err := emitJSON(filepath.Join(emitDir, "world", sc+".json"), map[string]any{"world": world}); err != nil {
				fail("emit world: %v", err)
			}
			foldPass++
			continue
		}

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
	if emitting {
		fmt.Printf("  %d/%d scenarios: world emitted\n", foldPass, len(scenarios))
	} else {
		fmt.Printf("  %d/%d scenarios: world matches expected\n", foldPass, len(scenarios))
	}

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

		if emitting {
			if err := emitJSON(filepath.Join(emitDir, "viewer-world", base+".json"), map[string]any{"viewerWorld": got}); err != nil {
				fail("emit viewer-world: %v", err)
			}
			vwPass++
			continue
		}

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
	if emitting {
		fmt.Printf("  %d/%d (scenario × viewer): viewerWorld emitted\n", vwPass, len(vwFiles))
	} else {
		fmt.Printf("  %d/%d (scenario × viewer): viewerWorld matches expected\n", vwPass, len(vwFiles))
	}

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

		if emitting {
			if err := emitJSON(filepath.Join(emitDir, "artifact", base+".json"), art); err != nil {
				fail("emit artifact: %v", err)
			}
			artPass++
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
	if emitting {
		fmt.Printf("  %d/%d (scenario × projection × viewer): artifact emitted\n", artPass, len(artFiles))
	} else {
		fmt.Printf("  %d/%d (scenario × projection × viewer): artifact matches expected\n", artPass, len(artFiles))
	}

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

		if emitting {
			if err := emitJSON(filepath.Join(emitDir, "document", base+".json"), doc); err != nil {
				fail("emit document: %v", err)
			}
			docPass++
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
		if emitting {
			fmt.Printf("  %d/%d (scenario × projection × viewer): document emitted\n", docPass, len(docFiles))
		} else {
			fmt.Printf("  %d/%d (scenario × projection × viewer): document matches expected\n", docPass, len(docFiles))
		}
	}

	fmt.Println()
	if emitting {
		fmt.Printf("== EMITTED to %s ==\n", emitDir)
		os.Exit(0)
	}
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

func emitJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
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
