# idf-go Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Реализовать на Go конформную с `spec-v0.1` библиотеку и CLI для L1+L2 conformance check на эталонном домене `library`, без чтения исходников первой реализации.

**Architecture:** Шесть пакетов с одной ответственностью (`types`, `parser`, `fold`, `filter`, `crystallize`, `internal/jsonutil`) + CLI `cmd/conformance`. Пакеты — чистые функции без shared state. Тесты — table-driven, читают fixtures из `../idf-spec/spec/fixtures/library/` через relative path.

**Tech Stack:** Go 1.22+, `github.com/xeipuuv/gojsonschema` v1.2 (единственная external зависимость), std lib для остального.

**Source of truth:** `~/WebstormProjects/idf-spec/spec/` (нормативная спека v0.1). Запрещено читать `idf/{src,server}/`, `idf-sdk/packages/*/src/`, `*.test.*`, `idf-spec/{design,plans}/`. Подробно — `idf-go/CLAUDE.md`.

**Locked decisions из дизайна:**
- α-поле эффекта в JSON = `kind` (Q-D4 спеки)
- World как `map[string]map[string]map[string]any` (D2 дизайна)
- Без `go:embed` — schemas читаются runtime (D3 дизайна)
- Relative path `../idf-spec/spec/fixtures/library/` для tests
- Семантическое сравнение через `internal/jsonutil.SemanticEqual`
- TDD discipline: failing test first, minimal implementation, commit per task

**File structure (создаётся за время плана):**

```
idf-go/
├── go.mod                          # module idf-go
├── go.sum
├── README.md                       # уже есть
├── CLAUDE.md                       # уже есть
├── .gitignore                      # уже есть
├── design/                         # уже есть
├── plans/
│   └── 2026-04-19-idf-go-plan.md   # этот файл
├── feedback/
│   └── spec-v0.1.md                # backlog ambiguities (заполняется по ходу)
│
├── types/
│   ├── ontology.go                 # Ontology, Entity, Field, Role
│   ├── intent.go                   # Intent, ProtoEffect, RequiredField
│   ├── effect.go                   # Effect, EffectContext
│   ├── projection.go               # Projection
│   ├── artifact.go                 # Artifact
│   └── world.go                    # World, ViewerWorld, Viewer
│
├── parser/
│   ├── parser.go                   # ParseOntology, ParseIntents, ParseProjections, ParsePhi
│   └── parser_test.go              # table-driven, читает library fixtures
│
├── fold/
│   ├── fold.go                     # Fold(phi, ontology) → World
│   └── fold_test.go                # 7 phi-сценариев → 7 expected/world
│
├── filter/
│   ├── filter.go                   # FilterWorldForRole(world, viewer, ontology) → ViewerWorld
│   └── filter_test.go              # 18 (scenario, viewer)
│
├── crystallize/
│   ├── crystallize.go              # Crystallize(...) — entry, диспатч
│   ├── derive.go                   # фаза 1
│   ├── merge.go                    # фаза 2
│   ├── slots_catalog.go            # фаза 3 — catalog
│   ├── slots_detail.go             # фаза 3 — detail
│   ├── slots_form.go               # фаза 3 — form
│   ├── slots_dashboard.go          # фаза 3 — dashboard
│   ├── slots_lightly.go            # фаза 3 — feed/canvas/wizard
│   ├── confirmation.go             # фаза 6
│   └── crystallize_test.go         # integration: 8 expected/artifact
│
├── internal/jsonutil/
│   ├── equal.go                    # SemanticEqual(a, b any) bool
│   ├── equal_test.go
│   ├── deepcopy.go                 # DeepCopy(v any) any
│   └── deepcopy_test.go
│
└── cmd/conformance/
    └── main.go                     # CLI orchestration
```

**Phases:**
1. Module setup + types
2. Parser
3. Semantic equality (jsonutil)
4. fold
5. filter
6. crystallize фазы 1-2 (derive + merge)
7. crystallize фаза 3 (assignToSlots per archetype)
8. crystallize фаза 6 + integration
9. CLI conformance
10. Backlog + tag

**Commits:** каждая задача завершается одним commit'ом. Сообщения — на русском, без `Co-Authored-By` трейлеров.

---

## Phase 1: Module setup + types

### Task 1: `go mod init`

**Files:**
- Create: `go.mod`

- [ ] **Step 1: Init Go module**

```bash
cd ~/WebstormProjects/idf-go && go mod init idf-go
```

Expected: создан `go.mod` со строкой `module idf-go` и `go 1.22` (или текущая версия).

- [ ] **Step 2: Verify Go version**

Run: `go version`
Expected: Go 1.22 или новее. Если меньше — обновить через `brew upgrade go` (на macOS) или скачать с golang.org.

- [ ] **Step 3: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add go.mod && git commit -q -m "init: go module idf-go"
```

### Task 2: Типы для core-объектов

**Files:**
- Create: `types/ontology.go`, `types/intent.go`, `types/effect.go`, `types/projection.go`, `types/artifact.go`, `types/world.go`

- [ ] **Step 1: `types/ontology.go`**

```go
// Package types определяет Go-структуры для пяти core-объектов формата
// IDF (ontology, intent, effect, projection, artifact) согласно spec-v0.1.
//
// Структуры используют encoding/json теги для round-trip парсинга и
// сериализации. Поля, зарезервированные L4 (invariants, rules,
// witnesses, shape), приняты как opaque и сохраняются для
// forward-compatibility, но не участвуют в L1+L2 логике.
package types

// Ontology — тип данных домена: сущности, поля, роли.
// Спека: spec/03-objects/ontology.md.
type Ontology struct {
	Meta       map[string]any        `json:"_meta,omitempty"`
	Entities   map[string]Entity     `json:"entities"`
	Roles      map[string]Role       `json:"roles"`
	Invariants []any                 `json:"invariants,omitempty"` // Reserved L4
	Rules      []any                 `json:"rules,omitempty"`      // Reserved L4
}

// Entity — описание типа сущности.
type Entity struct {
	Kind       string           `json:"kind,omitempty"` // "internal" (default) | "reference" | "mirror" | "assignment"
	OwnerField string           `json:"ownerField,omitempty"`
	Fields     map[string]Field `json:"fields"`
}

// Field — описание поля сущности.
type Field struct {
	Type       string   `json:"type"` // "string" | "number" | "boolean" | "datetime" | "enum"
	Values     []string `json:"values,omitempty"`     // для type="enum"
	FieldRole  string   `json:"fieldRole,omitempty"`  // opaque hint
	References string   `json:"references,omitempty"` // FK на entity
	Required   bool     `json:"required,omitempty"`
}

// Role — viewer-тип.
type Role struct {
	VisibleFields map[string]VisibleFieldsValue `json:"visibleFields"`
	CanExecute    []string                       `json:"canExecute"`
	Scope         map[string]any                 `json:"scope,omitempty"` // Reserved L4
	Base          string                         `json:"base,omitempty"`  // Reserved L4
}

// VisibleFieldsValue представляет либо массив имён полей, либо строку "*"
// (все поля). См. spec/03-objects/ontology.md role.visibleFields.
type VisibleFieldsValue struct {
	All    bool     // true если "*"
	Fields []string // конкретные имена полей
}

// UnmarshalJSON разбирает значение visibleFields[Entity]: либо строка "*",
// либо массив строк.
func (v *VisibleFieldsValue) UnmarshalJSON(data []byte) error {
	// Попытка как строка "*"
	var s string
	if err := jsonUnmarshal(data, &s); err == nil {
		if s == "*" {
			v.All = true
			return nil
		}
		return errInvalidVisibleFieldsString(s)
	}
	// Попытка как массив строк
	var arr []string
	if err := jsonUnmarshal(data, &arr); err != nil {
		return err
	}
	v.Fields = arr
	return nil
}

// MarshalJSON сериализует обратно в строку "*" или массив имён.
func (v VisibleFieldsValue) MarshalJSON() ([]byte, error) {
	if v.All {
		return []byte(`"*"`), nil
	}
	return jsonMarshal(v.Fields)
}
```

- [ ] **Step 2: helper-обёртки `types/json_helpers.go`**

```go
package types

import "encoding/json"

// Тонкие обёртки над encoding/json, чтобы избежать import cycle при
// тестировании UnmarshalJSON.
var (
	jsonUnmarshal = json.Unmarshal
	jsonMarshal   = json.Marshal
)

func errInvalidVisibleFieldsString(s string) error {
	return &TypeError{Field: "visibleFields", Got: s, Want: `"*" or []string`}
}

// TypeError — структурированная ошибка типизированного парсинга.
type TypeError struct {
	Field string
	Got   string
	Want  string
}

func (e *TypeError) Error() string {
	return "types: поле " + e.Field + " — got " + e.Got + ", want " + e.Want
}
```

- [ ] **Step 3: `types/intent.go`**

```go
package types

// Intent — декларативная частица: возможность изменить мир.
// Спека: spec/03-objects/intent.md.
type Intent struct {
	ID             string          `json:"id"`
	OwnerRole      string          `json:"ownerRole"`
	RequiredFields []RequiredField `json:"requiredFields,omitempty"`
	Conditions     []any           `json:"conditions,omitempty"` // Reserved L4 — opaque
	Effects        []ProtoEffect   `json:"effects"`
}

// RequiredField — поле, обязательное для заполнения viewer'ом перед confirm.
type RequiredField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ProtoEffect — шаблон эффекта в intent.effects[]. Не готовый эффект:
// fields опциональны и могут быть templated в v0.2+ (Q-7 спеки).
type ProtoEffect struct {
	Kind   string                 `json:"kind"` // "create" | "replace" | "remove" | "transition" | "commit"
	Entity string                 `json:"entity"`
	Fields map[string]any         `json:"fields,omitempty"`
}
```

- [ ] **Step 4: `types/effect.go`**

```go
package types

// Effect — атом изменения мира. Confirmed effects складываются в Φ.
// Спека: spec/03-objects/effect.md.
type Effect struct {
	Kind    string         `json:"kind"` // "create" | "replace" | "remove" | "transition" | "commit"
	Entity  string         `json:"entity"`
	Fields  map[string]any `json:"fields,omitempty"`
	Context EffectContext  `json:"context"`
}

// EffectContext — метаданные эффекта. v0.1 нормирует только обязательное
// поле At (ISO 8601 timestamp). Остальные поля opaque.
type EffectContext struct {
	At        string         `json:"at"` // ISO 8601 date-time, обязательно
	Initiator string         `json:"initiator,omitempty"`
	IntentID  string         `json:"intentId,omitempty"`
	IRR       map[string]any `json:"__irr,omitempty"` // Reserved L4
	// Прочие поля попадают в Extra через UnmarshalJSON
	Extra map[string]any `json:"-"`
}
```

- [ ] **Step 5: `types/projection.go`**

```go
package types

// Projection — авторский контракт на view. Input для crystallize.
// Спека: spec/03-objects/projection.md.
type Projection struct {
	ID        string         `json:"id"`
	Archetype string         `json:"archetype,omitempty"` // "auto" | "feed" | "catalog" | "detail" | "form" | "canvas" | "dashboard" | "wizard"
	Intents   []string       `json:"intents"`
	Entity    string         `json:"entity,omitempty"`
	Slots     map[string]any `json:"slots,omitempty"`    // slot-override (фаза 2 mergeProjections)
	Patterns  map[string]any `json:"patterns,omitempty"` // Reserved L4
}
```

- [ ] **Step 6: `types/artifact.go`**

```go
package types

// Artifact — output crystallize. Тип данных (не render).
// Спека: spec/03-objects/artifact.md.
type Artifact struct {
	Meta              map[string]any `json:"_meta,omitempty"`
	ProjectionID      string         `json:"projectionId"`
	Archetype         string         `json:"archetype"`
	Viewer            string         `json:"viewer"` // имя роли
	Slots             map[string]any `json:"slots"`
	Witnesses         []any          `json:"witnesses,omitempty"`         // Reserved L4
	Shape             string         `json:"shape,omitempty"`             // Reserved L4
	PatternAnnotations []any         `json:"patternAnnotations,omitempty"` // Reserved L4
}
```

- [ ] **Step 7: `types/world.go`**

```go
package types

// World — состояние мира после fold(Φ, ontology).
// Структура: {EntityName: {EntityID: EntityRecord}}.
// EntityRecord — map[string]any (зависит от ontology.entities[E].fields).
type World map[string]map[string]map[string]any

// ViewerWorld — output filterWorldForRole. Тот же тип что World;
// alias для type-safety и читабельности.
type ViewerWorld = World

// Viewer идентифицирует роль и id viewer'а для filterWorldForRole.
type Viewer struct {
	Role string
	ID   string
}
```

- [ ] **Step 8: Verify build**

Run: `cd ~/WebstormProjects/idf-go && go build ./types/`
Expected: успех, без ошибок.

- [ ] **Step 9: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add types/ && git commit -q -m "types: 6 файлов struct-определений (ontology, intent, effect, projection, artifact, world)"
```

---

## Phase 2: Parser

### Task 3: parser + JSON Schema validation

**Files:**
- Create: `parser/parser.go`, `parser/parser_test.go`
- Modify: `go.mod` (add gojsonschema)

- [ ] **Step 1: Add `gojsonschema` dependency**

```bash
cd ~/WebstormProjects/idf-go && go get github.com/xeipuuv/gojsonschema@v1.2.0
```

Expected: `go.mod` обновлён, `go.sum` создан.

- [ ] **Step 2: `parser/parser.go`**

```go
// Package parser валидирует и декодирует JSON-файлы спеки IDF против
// JSON Schema draft-07 + Go struct-типов.
//
// Каждая Parse* функция принимает []byte и path к схема-файлу;
// возвращает typed-struct + error.
package parser

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"

	"idf-go/types"
)

// SchemaSet — набор путей к JSON Schema-файлам, нужных парсеру.
// По умолчанию указывает на ../idf-spec/spec/schemas/.
type SchemaSet struct {
	Ontology              string
	Intent                string
	IntentsCollection     string
	Projection            string
	ProjectionsCollection string
	Effect                string
	Phi                   string
	Artifact              string
}

// DefaultSchemaSet возвращает пути относительно cwd ../idf-spec/spec/schemas/.
func DefaultSchemaSet(specRoot string) SchemaSet {
	return SchemaSet{
		Ontology:              specRoot + "/schemas/ontology.schema.json",
		Intent:                specRoot + "/schemas/intent.schema.json",
		IntentsCollection:     specRoot + "/schemas/intents-collection.schema.json",
		Projection:            specRoot + "/schemas/projection.schema.json",
		ProjectionsCollection: specRoot + "/schemas/projections-collection.schema.json",
		Effect:                specRoot + "/schemas/effect.schema.json",
		Phi:                   specRoot + "/schemas/phi.schema.json",
		Artifact:              specRoot + "/schemas/artifact.schema.json",
	}
}

// validateAgainstSchema выполняет JSON Schema validation. Возвращает
// nil если data соответствует schema, иначе structured error.
func validateAgainstSchema(data []byte, schemaPath string, refSchemas []string) error {
	loader := gojsonschema.NewSchemaLoader()
	loader.Validate = false
	loader.Draft = gojsonschema.Draft7
	for _, ref := range refSchemas {
		if err := loader.AddSchemas(gojsonschema.NewReferenceLoader("file://" + ref)); err != nil {
			return fmt.Errorf("parser: load ref schema %s: %w", ref, err)
		}
	}
	schema, err := loader.Compile(gojsonschema.NewReferenceLoader("file://" + schemaPath))
	if err != nil {
		return fmt.Errorf("parser: compile schema %s: %w", schemaPath, err)
	}
	result, err := schema.Validate(gojsonschema.NewBytesLoader(data))
	if err != nil {
		return fmt.Errorf("parser: validate: %w", err)
	}
	if !result.Valid() {
		return fmt.Errorf("parser: schema validation failed for %s: %v", schemaPath, result.Errors())
	}
	return nil
}

// ParseOntology валидирует ontology.json против ontology.schema.json
// и декодирует в types.Ontology.
func ParseOntology(data []byte, schemas SchemaSet) (types.Ontology, error) {
	if err := validateAgainstSchema(data, schemas.Ontology, nil); err != nil {
		return types.Ontology{}, err
	}
	var ont types.Ontology
	if err := json.Unmarshal(data, &ont); err != nil {
		return types.Ontology{}, fmt.Errorf("parser: ontology decode: %w", err)
	}
	return ont, nil
}

// IntentsCollection — wrapper для fixture-файла intents.json.
type IntentsCollection struct {
	Meta    map[string]any  `json:"_meta,omitempty"`
	Intents []types.Intent  `json:"intents"`
}

// ParseIntents валидирует intents.json через intents-collection wrapper
// и декодирует.
func ParseIntents(data []byte, schemas SchemaSet) ([]types.Intent, error) {
	if err := validateAgainstSchema(data, schemas.IntentsCollection, []string{schemas.Intent}); err != nil {
		return nil, err
	}
	var coll IntentsCollection
	if err := json.Unmarshal(data, &coll); err != nil {
		return nil, fmt.Errorf("parser: intents decode: %w", err)
	}
	return coll.Intents, nil
}

// ProjectionsCollection — wrapper для fixture-файла projections.json.
type ProjectionsCollection struct {
	Meta        map[string]any      `json:"_meta,omitempty"`
	Projections []types.Projection  `json:"projections"`
}

// ParseProjections валидирует projections.json через wrapper и декодирует.
func ParseProjections(data []byte, schemas SchemaSet) ([]types.Projection, error) {
	if err := validateAgainstSchema(data, schemas.ProjectionsCollection, []string{schemas.Projection}); err != nil {
		return nil, err
	}
	var coll ProjectionsCollection
	if err := json.Unmarshal(data, &coll); err != nil {
		return nil, fmt.Errorf("parser: projections decode: %w", err)
	}
	return coll.Projections, nil
}

// PhiFile — wrapper для fixture-файла phi/<scenario>.json.
type PhiFile struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Effects []types.Effect `json:"effects"`
}

// ParsePhi валидирует phi/<scenario>.json через phi.schema.json и
// декодирует. Φ возвращается в исходном порядке (без сортировки —
// сортировка по at — задача fold).
func ParsePhi(data []byte, schemas SchemaSet) ([]types.Effect, error) {
	if err := validateAgainstSchema(data, schemas.Phi, []string{schemas.Effect}); err != nil {
		return nil, err
	}
	var phi PhiFile
	if err := json.Unmarshal(data, &phi); err != nil {
		return nil, fmt.Errorf("parser: phi decode: %w", err)
	}
	return phi.Effects, nil
}

// ReadFile — небольшая утилита для тестов и CLI.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
```

- [ ] **Step 3: `parser/parser_test.go` (table-driven)**

```go
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
	// Smoke: 3 entities, 2 roles
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
	// VisibleFields: librarian.User должно быть "*"
	if !ont.Roles["librarian"].VisibleFields["User"].All {
		t.Errorf("librarian.User: ожидается All=true (visibleFields '*')")
	}
	// VisibleFields: reader.User должно быть [id, name]
	rUser := ont.Roles["reader"].VisibleFields["User"]
	if rUser.All || len(rUser.Fields) != 2 {
		t.Errorf("reader.User: ожидается [id, name], got All=%v Fields=%v", rUser.All, rUser.Fields)
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
```

- [ ] **Step 4: Run tests**

Run: `cd ~/WebstormProjects/idf-go && go test ./parser/ -v`
Expected: все 4 TestParseXxx pass; TestParsePhi с 7 sub-tests pass.

Если падает на schema $ref resolution — отладить пути в `DefaultSchemaSet`.

- [ ] **Step 5: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add parser/ go.mod go.sum && git commit -q -m "parser: ParseOntology/Intents/Projections/Phi с JSON Schema валидацией (gojsonschema draft-07)"
```

---

## Phase 3: Semantic equality (jsonutil)

### Task 4: `internal/jsonutil/equal.go` + tests

**Files:**
- Create: `internal/jsonutil/equal.go`, `internal/jsonutil/equal_test.go`

- [ ] **Step 1: `equal.go`**

```go
// Package jsonutil содержит утилиты для семантического сравнения и
// глубокого копирования JSON-подобных структур (map[string]any,
// []any, scalars).
//
// Используется для сравнения fixture-output с computed output (deep-equal
// с игнорированием порядка ключей в map'ах; для массивов — порядок
// важен).
package jsonutil

// SemanticEqual сравнивает два JSON-подобных значения семантически:
//   - объекты (map[string]any): рекурсивно по ключам, порядок не важен
//   - массивы ([]any): рекурсивно по индексам, порядок важен
//   - scalars (string, float64, bool, nil): прямое сравнение
//
// Числа сравниваются как float64 (стандарт encoding/json).
//
// Поле "_meta" на любом уровне — игнорируется при сравнении (informative
// в expected fixtures).
func SemanticEqual(a, b any) bool {
	return equalIgnoring(a, b, map[string]bool{"_meta": true})
}

// SemanticEqualStrict — то же что SemanticEqual, но без игнорирования _meta.
func SemanticEqualStrict(a, b any) bool {
	return equalIgnoring(a, b, nil)
}

func equalIgnoring(a, b any, ignoreKeys map[string]bool) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return false
		}
		if !mapsEqual(av, bv, ignoreKeys) {
			return false
		}
		return true
	case []any:
		bv, ok := b.([]any)
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !equalIgnoring(av[i], bv[i], ignoreKeys) {
				return false
			}
		}
		return true
	case nil:
		return b == nil
	default:
		return a == b
	}
}

func mapsEqual(a, b map[string]any, ignoreKeys map[string]bool) bool {
	for k, av := range a {
		if ignoreKeys[k] {
			continue
		}
		bv, ok := b[k]
		if !ok {
			return false
		}
		if !equalIgnoring(av, bv, ignoreKeys) {
			return false
		}
	}
	for k := range b {
		if ignoreKeys[k] {
			continue
		}
		if _, ok := a[k]; !ok {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: `equal_test.go`**

```go
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
```

- [ ] **Step 3: Run tests**

Run: `cd ~/WebstormProjects/idf-go && go test ./internal/jsonutil/ -v`
Expected: все 11 sub-tests pass.

- [ ] **Step 4: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add internal/ && git commit -q -m "jsonutil: SemanticEqual с игнорированием _meta + tests (11 кейсов)"
```

### Task 5: `internal/jsonutil/deepcopy.go` + tests

**Files:**
- Create: `internal/jsonutil/deepcopy.go`, `internal/jsonutil/deepcopy_test.go`

- [ ] **Step 1: `deepcopy.go`**

```go
package jsonutil

// DeepCopy возвращает глубокую копию JSON-подобного значения. Безопасно
// мутировать результат без воздействия на оригинал.
//
// Поддерживаемые типы: map[string]any, []any, string, float64, bool, nil.
// Прочие типы — возвращаются как есть (shallow).
func DeepCopy(v any) any {
	switch vv := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(vv))
		for k, val := range vv {
			out[k] = DeepCopy(val)
		}
		return out
	case []any:
		out := make([]any, len(vv))
		for i, val := range vv {
			out[i] = DeepCopy(val)
		}
		return out
	default:
		return v
	}
}

// DeepCopyMap — type-safe shortcut для DeepCopy на map'ах.
func DeepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	return DeepCopy(m).(map[string]any)
}
```

- [ ] **Step 2: `deepcopy_test.go`**

```go
package jsonutil

import "testing"

func TestDeepCopy(t *testing.T) {
	orig := map[string]any{
		"a": 1.0,
		"b": []any{"x", "y"},
		"c": map[string]any{"nested": true},
	}
	cp := DeepCopy(orig).(map[string]any)
	// Мутация копии не должна затрагивать оригинал
	cp["a"] = 999.0
	cp["b"].([]any)[0] = "changed"
	cp["c"].(map[string]any)["nested"] = false

	if orig["a"] != 1.0 {
		t.Errorf("orig.a mutated: %v", orig["a"])
	}
	if orig["b"].([]any)[0] != "x" {
		t.Errorf("orig.b[0] mutated: %v", orig["b"].([]any)[0])
	}
	if orig["c"].(map[string]any)["nested"] != true {
		t.Errorf("orig.c.nested mutated: %v", orig["c"].(map[string]any)["nested"])
	}
}

func TestDeepCopyMap(t *testing.T) {
	if got := DeepCopyMap(nil); got != nil {
		t.Errorf("DeepCopyMap(nil) = %v, want nil", got)
	}
	m := map[string]any{"a": 1.0}
	cp := DeepCopyMap(m)
	cp["a"] = 999.0
	if m["a"] != 1.0 {
		t.Errorf("orig.a mutated: %v", m["a"])
	}
}
```

- [ ] **Step 3: Run + commit**

Run: `cd ~/WebstormProjects/idf-go && go test ./internal/jsonutil/ -v`
Expected: все pass.

```bash
cd ~/WebstormProjects/idf-go && git add internal/jsonutil/deepcopy.go internal/jsonutil/deepcopy_test.go && git commit -q -m "jsonutil: DeepCopy для JSON-подобных значений + tests"
```

---

## Phase 4: fold

### Task 6: `fold/fold.go` — write failing test first

**Files:**
- Create: `fold/fold.go`, `fold/fold_test.go`

- [ ] **Step 1: `fold/fold_test.go` — table-driven test**

```go
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

	// Читаем ontology один раз
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

			expWorld, ok := exp["world"].(map[string]any)
			if !ok {
				t.Fatalf("expected.world not a map")
			}

			// Преобразуем got (типизированный World) в map[string]any для сравнения
			gotJSON, _ := json.Marshal(got)
			var gotMap map[string]any
			_ = json.Unmarshal(gotJSON, &gotMap)

			if !jsonutil.SemanticEqual(gotMap, expWorld) {
				t.Errorf("scenario %s: world mismatch\ngot:  %v\nwant: %v", sc, gotMap, expWorld)
			}
		})
	}
}
```

- [ ] **Step 2: Run — should fail (no Fold function yet)**

Run: `cd ~/WebstormProjects/idf-go && go test ./fold/ -v`
Expected: FAIL — `undefined: Fold`.

- [ ] **Step 3: `fold/fold.go`**

```go
// Package fold реализует fold(Φ, ontology) → world согласно
// spec/04-algebra/fold.md.
//
// Φ — массив confirmed эффектов, упорядоченный по context.at ASC
// (stable sort; tie-breaker — позиция в исходном массиве).
//
// World — map[Entity]map[ID]map[Field]any. Каждая entity из
// ontology.entities присутствует как top-level ключ (даже пустая).
package fold

import (
	"fmt"
	"sort"

	"idf-go/internal/jsonutil"
	"idf-go/types"
)

// Fold применяет Φ к пустому миру согласно ontology.
//
// Алгоритм:
//   1. Инициализировать world с пустыми namespace для каждой entity.
//   2. Stable-sort phi по context.at ASC (tie-breaker — исходный индекс).
//   3. Применить каждый effect согласно kind.
func Fold(phi []types.Effect, ont types.Ontology) (types.World, error) {
	world := make(types.World, len(ont.Entities))
	for name := range ont.Entities {
		world[name] = make(map[string]map[string]any)
	}

	// Stable-sort по context.at; sort.SliceStable сохраняет порядок
	// эффектов с одинаковым at.
	sorted := make([]types.Effect, len(phi))
	copy(sorted, phi)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Context.At < sorted[j].Context.At
	})

	for _, eff := range sorted {
		if err := applyEffect(world, eff); err != nil {
			return nil, err
		}
	}
	return world, nil
}

func applyEffect(world types.World, eff types.Effect) error {
	ns, ok := world[eff.Entity]
	if !ok {
		return fmt.Errorf("fold: unknown entity %q in effect", eff.Entity)
	}

	id, ok := getID(eff.Fields)
	if !ok && eff.Kind != "commit" {
		return fmt.Errorf("fold: effect %s on %s missing fields.id", eff.Kind, eff.Entity)
	}

	switch eff.Kind {
	case "create":
		if _, exists := ns[id]; exists {
			return fmt.Errorf("fold: create-on-existing: %s/%s already exists", eff.Entity, id)
		}
		ns[id] = jsonutil.DeepCopyMap(eff.Fields)
	case "replace", "transition":
		existing, exists := ns[id]
		if !exists {
			return fmt.Errorf("fold: %s-on-missing: %s/%s does not exist", eff.Kind, eff.Entity, id)
		}
		// Shallow merge: новые поля перетирают старые.
		merged := jsonutil.DeepCopyMap(existing)
		for k, v := range eff.Fields {
			merged[k] = jsonutil.DeepCopy(v)
		}
		ns[id] = merged
	case "remove":
		// Idempotent: no-op если отсутствует (Q-15 спеки).
		delete(ns, id)
	case "commit":
		// L4 — no-op в L1+L2 (Lightly-tested).
	default:
		return fmt.Errorf("fold: unknown effect.kind %q", eff.Kind)
	}
	return nil
}

func getID(fields map[string]any) (string, bool) {
	if fields == nil {
		return "", false
	}
	v, ok := fields["id"]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
```

- [ ] **Step 4: Run — should pass**

Run: `cd ~/WebstormProjects/idf-go && go test ./fold/ -v`
Expected: PASS, 7 sub-tests pass (empty, bootstrap, register-readers, borrow-cycle, borrow-and-return, cancel-loan, update-book).

Если scenario fails — посмотреть diff и исправить либо expected (баг fixture) либо алгоритм fold (баг реализации). Спека первична.

- [ ] **Step 5: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add fold/ && git commit -q -m "fold: Fold(phi, ontology) → World с 5 видами α + 7 fixture-сценариев pass"
```

---

## Phase 5: filter

### Task 7: `filter/filter.go` — write failing test first

**Files:**
- Create: `filter/filter.go`, `filter/filter_test.go`

- [ ] **Step 1: `filter/filter_test.go`**

```go
package filter

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"idf-go/fold"
	"idf-go/internal/jsonutil"
	"idf-go/parser"
	"idf-go/types"
)

const fixturesRoot = "../../idf-spec/spec/fixtures/library"
const specRoot = "../../idf-spec/spec"

// pairsToTest строит таблицу (scenario, viewer) пар, существующих в
// expected/viewer-world/.
func pairsToTest() []struct {
	scenario string
	viewer   types.Viewer
	file     string // имя файла без .json в expected/viewer-world/
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
	ont, _ := parser.ParseOntology(ontData, schemas)

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
				t.Errorf("viewerWorld mismatch for %s\ngot:\n%s\nwant:\n%s",
					tc.file,
					strings.ReplaceAll(string(gotPretty), "\n", "\n  "),
					strings.ReplaceAll(string(expPretty), "\n", "\n  "))
			}
		})
	}
}
```

- [ ] **Step 2: Run — should fail**

Run: `cd ~/WebstormProjects/idf-go && go test ./filter/ -v`
Expected: FAIL — `undefined: FilterWorldForRole`.

- [ ] **Step 3: `filter/filter.go`**

```go
// Package filter реализует filterWorldForRole(world, viewer, ontology)
// → viewerWorld согласно spec/04-algebra/filter-world.md.
//
// 3-приоритетный row-filter:
//   1. Если entity не упомянута в role.visibleFields — namespace отсутствует.
//   2. Если entity.kind == "reference" — все записи видны.
//   3. Если entity.ownerField задан — записи где record[ownerField] == viewer.id.
//   4. Иначе — privacy by default (пустой namespace).
//
// role.scope (приоритет 0) — Reserved L4, не используется.
//
// Column-filter (после row-filter): если visibleFields[E] == "*" — все
// поля; иначе — только перечисленные.
package filter

import (
	"idf-go/internal/jsonutil"
	"idf-go/types"
)

// FilterWorldForRole применяет 3-приоритетный row-filter + column-filter
// согласно ontology и viewer.
func FilterWorldForRole(world types.World, viewer types.Viewer, ont types.Ontology) types.ViewerWorld {
	role, ok := ont.Roles[viewer.Role]
	if !ok {
		// Неизвестная роль — пустой viewerWorld
		return types.ViewerWorld{}
	}
	out := make(types.ViewerWorld, len(role.VisibleFields))
	for entityName := range ont.Entities {
		visible, mentioned := role.VisibleFields[entityName]
		if !mentioned {
			continue // priority 1: gate
		}
		entity := ont.Entities[entityName]
		filtered := make(map[string]map[string]any)
		switch {
		case entity.Kind == "reference":
			// priority 2: все записи видны
			for id, rec := range world[entityName] {
				filtered[id] = projectFields(rec, visible)
			}
		case entity.OwnerField != "":
			// priority 3: ownership filter
			for id, rec := range world[entityName] {
				if matchOwner(rec, entity.OwnerField, viewer.ID) {
					filtered[id] = projectFields(rec, visible)
				}
			}
			// priority 4 (none) — filtered остаётся пустым
		}
		out[entityName] = filtered
	}
	return out
}

func matchOwner(rec map[string]any, ownerField, viewerID string) bool {
	v, ok := rec[ownerField]
	if !ok {
		return false
	}
	s, ok := v.(string)
	return ok && s == viewerID
}

// projectFields возвращает копию rec, содержащую только поля из allowed
// (или все поля если allowed.All).
func projectFields(rec map[string]any, allowed types.VisibleFieldsValue) map[string]any {
	if allowed.All {
		return jsonutil.DeepCopyMap(rec)
	}
	out := make(map[string]any, len(allowed.Fields))
	for _, name := range allowed.Fields {
		if v, ok := rec[name]; ok {
			out[name] = jsonutil.DeepCopy(v)
		}
	}
	return out
}
```

- [ ] **Step 4: Run — should pass**

Run: `cd ~/WebstormProjects/idf-go && go test ./filter/ -v`
Expected: PASS, 18 sub-tests pass.

- [ ] **Step 5: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add filter/ && git commit -q -m "filter: FilterWorldForRole с 3-приоритетным row-filter + column-filter; 18 fixture-пар pass"
```

---

## Phase 6: crystallize фазы 1+2 (derive + merge)

### Task 8: `crystallize/crystallize.go` skeleton + `derive.go`

**Files:**
- Create: `crystallize/crystallize.go`, `crystallize/derive.go`

- [ ] **Step 1: `crystallize/crystallize.go` (entry, диспатчит фазы)**

```go
// Package crystallize реализует crystallize(intents, ontology, projection,
// viewer, viewerWorld) → artifact согласно spec/04-algebra/crystallize.md.
//
// 6 фаз pipeline:
//   1. deriveProjections — авто-вывод archetype
//   2. mergeProjections — slot-override через deep-merge
//   3. assignToSlots — per-archetype распределение intents/данных
//   4. matchPatterns — noop в L2 (Pattern Bank Reserved L4)
//   5. applyStructuralPatterns — noop в L2
//   6. wrapByConfirmation — destructive/standard на intent references
package crystallize

import (
	"fmt"

	"idf-go/types"
)

// Crystallize применяет 6-фазный pipeline.
func Crystallize(
	intents []types.Intent,
	ont types.Ontology,
	proj types.Projection,
	viewer types.Viewer,
	vw types.ViewerWorld,
) (types.Artifact, error) {
	// Lookup intent.id → Intent
	intentByID := make(map[string]types.Intent, len(intents))
	for _, it := range intents {
		intentByID[it.ID] = it
	}

	// Фаза 1: derive archetype если auto
	archetype, err := deriveArchetype(proj, intentByID)
	if err != nil {
		return types.Artifact{}, fmt.Errorf("crystallize phase 1: %w", err)
	}

	// Фаза 3: assignToSlots (фаза 2 mergeProjections применяется поверх результата)
	slots, err := assignToSlots(archetype, proj, intentByID, ont, viewer, vw)
	if err != nil {
		return types.Artifact{}, fmt.Errorf("crystallize phase 3: %w", err)
	}

	// Фаза 2: mergeProjections (применяется поверх derived slots согласно крист.md
	// «значения из projection.slots имеют приоритет над derived-значениями»)
	if proj.Slots != nil {
		slots = mergeSlots(slots, proj.Slots)
	}

	// Фаза 4-5: noop в L2

	// Фаза 6: wrapByConfirmation — добавляет confirmation-поле к intent references
	wrapByConfirmation(slots, intentByID)

	return types.Artifact{
		ProjectionID: proj.ID,
		Archetype:    archetype,
		Viewer:       viewer.Role,
		Slots:        slots,
	}, nil
}
```

- [ ] **Step 2: `crystallize/derive.go`**

```go
package crystallize

import (
	"fmt"

	"idf-go/types"
)

// deriveArchetype реализует фазу 1 (deriveProjections).
//
// Если archetype != "auto" — использовать как есть.
// Иначе — минимальная heuristic спеки v0.1:
//   - все intents имеют effects[0].kind == "create" и одну entity → "form"
//   - projection.entity задан → "detail"
//   - fallback → "catalog"
func deriveArchetype(proj types.Projection, intentByID map[string]types.Intent) (string, error) {
	if proj.Archetype != "" && proj.Archetype != "auto" {
		return proj.Archetype, nil
	}

	// Heuristic 1: все intents create + одна entity → form
	if len(proj.Intents) > 0 {
		allCreate := true
		var entity string
		for _, id := range proj.Intents {
			it, ok := intentByID[id]
			if !ok {
				return "", fmt.Errorf("derive: unknown intent %s", id)
			}
			if len(it.Effects) == 0 || it.Effects[0].Kind != "create" {
				allCreate = false
				break
			}
			if entity == "" {
				entity = it.Effects[0].Entity
			} else if entity != it.Effects[0].Entity {
				allCreate = false
				break
			}
		}
		if allCreate {
			return "form", nil
		}
	}

	// Heuristic 2: projection.entity задан → detail
	if proj.Entity != "" {
		return "detail", nil
	}

	// Fallback
	return "catalog", nil
}
```

- [ ] **Step 3: Verify build**

Run: `cd ~/WebstormProjects/idf-go && go build ./crystallize/`
Expected: compile errors про `assignToSlots`, `mergeSlots`, `wrapByConfirmation` (ещё не реализованы — будет в Tasks 9, 10, 12). Это OK на данный момент.

Если хочешь временно протестировать: добавить stub'ы:
```go
// Заглушки до реализации:
func assignToSlots(archetype string, proj types.Projection, intentByID map[string]types.Intent, ont types.Ontology, viewer types.Viewer, vw types.ViewerWorld) (map[string]any, error) {
	return map[string]any{}, nil
}
func mergeSlots(derived, authored map[string]any) map[string]any { return derived }
func wrapByConfirmation(slots map[string]any, intentByID map[string]types.Intent) {}
```

Положить в `crystallize/stubs.go` (с пометкой `// TODO`-комментарием), удалить когда будут реальные реализации.

- [ ] **Step 4: Commit (со stub'ами)**

```bash
cd ~/WebstormProjects/idf-go && git add crystallize/ && git commit -q -m "crystallize: skeleton Crystallize() + deriveArchetype (фаза 1) + stubs для остальных"
```

### Task 9: `crystallize/merge.go` — фаза 2 mergeProjections

**Files:**
- Create: `crystallize/merge.go`, `crystallize/merge_test.go`
- Modify: `crystallize/stubs.go` (удалить mergeSlots stub)

- [ ] **Step 1: `crystallize/merge_test.go`**

```go
package crystallize

import (
	"testing"

	"idf-go/internal/jsonutil"
)

func TestMergeSlots(t *testing.T) {
	cases := []struct {
		name           string
		derived, auth  map[string]any
		want           map[string]any
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
```

- [ ] **Step 2: Удалить `mergeSlots` из `stubs.go`**

```go
// Файл crystallize/stubs.go теперь:
package crystallize

import "idf-go/types"

// Заглушки до реализации:
func assignToSlots(archetype string, proj types.Projection, intentByID map[string]types.Intent, ont types.Ontology, viewer types.Viewer, vw types.ViewerWorld) (map[string]any, error) {
	return map[string]any{}, nil
}
func wrapByConfirmation(slots map[string]any, intentByID map[string]types.Intent) {}
```

- [ ] **Step 3: `crystallize/merge.go`**

```go
package crystallize

import (
	"idf-go/internal/jsonutil"
)

// mergeSlots реализует фазу 2 mergeProjections (см. spec/04-algebra/crystallize.md).
//
// Семантика merge:
//   - Объекты (map[string]any): рекурсивный deep-merge с приоритетом authored.
//   - Массивы ([]any): замена целиком (стандартная JSON merge семантика).
//   - Скалярные значения: замена на authored.
//   - Поле _authored: true сохраняется в результате (forward-compat marker).
func mergeSlots(derived, authored map[string]any) map[string]any {
	out := jsonutil.DeepCopyMap(derived)
	for k, v := range authored {
		if v == nil {
			out[k] = nil
			continue
		}
		dv, ok := out[k]
		if !ok {
			out[k] = jsonutil.DeepCopy(v)
			continue
		}
		// Если оба — map'ы, recursive merge; иначе — replace
		dMap, dOk := dv.(map[string]any)
		aMap, aOk := v.(map[string]any)
		if dOk && aOk {
			out[k] = mergeSlots(dMap, aMap)
		} else {
			out[k] = jsonutil.DeepCopy(v)
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests**

Run: `cd ~/WebstormProjects/idf-go && go test ./crystallize/ -v -run TestMergeSlots`
Expected: 4 sub-tests pass.

- [ ] **Step 5: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add crystallize/merge.go crystallize/merge_test.go crystallize/stubs.go && git commit -q -m "crystallize: фаза 2 mergeSlots (deep-merge с array-replace) + 4 unit tests"
```

---

## Phase 7: crystallize фаза 3 (assignToSlots per archetype)

### Task 10: `crystallize/slots_catalog.go`

**Files:**
- Create: `crystallize/slots_catalog.go`
- Modify: `crystallize/stubs.go` (постепенно удалять)

- [ ] **Step 1: Удалить `assignToSlots` stub и заменить на dispatcher**

В `crystallize/stubs.go` удалить `assignToSlots`. Заменить на dispatcher (создаётся в этом же step'е):

```go
// Файл crystallize/stubs.go:
package crystallize

import "idf-go/types"

// Stub только для wrapByConfirmation (будет реализован в Task 13)
func wrapByConfirmation(slots map[string]any, intentByID map[string]types.Intent) {}
```

Создать `crystallize/dispatch.go`:

```go
package crystallize

import (
	"fmt"

	"idf-go/types"
)

// assignToSlots — фаза 3 dispatcher per archetype (см. spec/04-algebra/crystallize.md).
func assignToSlots(
	archetype string,
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) (map[string]any, error) {
	switch archetype {
	case "catalog":
		return slotsCatalog(proj, intentByID, ont, viewer, vw), nil
	case "detail":
		return slotsDetail(proj, intentByID, ont, viewer, vw), nil
	case "form":
		return slotsForm(proj, intentByID, ont, viewer, vw), nil
	case "dashboard":
		return slotsDashboard(proj, intentByID, ont, viewer, vw), nil
	case "feed", "canvas", "wizard":
		// Lightly-tested: минимальная структура согласно spec/03-objects/artifact.md
		return slotsLightly(archetype, proj, intentByID, ont, viewer, vw), nil
	default:
		return nil, fmt.Errorf("crystallize: unknown archetype %q", archetype)
	}
}

// roleCanExecute проверяет, доступен ли intent роли viewer'а.
// Поддерживает "*" (все intents).
func roleCanExecute(role types.Role, intentID string) bool {
	for _, id := range role.CanExecute {
		if id == "*" || id == intentID {
			return true
		}
	}
	return false
}

// primaryField возвращает имя «первичного» поля сущности — первое поле
// (по key order onto.Entities[E].fields), не равное "id" и без
// references. См. spec/03-objects/artifact.md, Q-12/Q-24.
//
// ВАЖНО: Go maps не имеют гарантированного порядка итерации. Так как
// fields — map[string]Field, нужно использовать порядок, в котором они
// появлялись в JSON. encoding/json не сохраняет порядок ключей в map.
// Мы используем lexicographic порядок имён как stable proxy за
// отсутствием другой информации (это Open question — фиксировать как
// implementer choice в feedback/spec-v0.1.md).
func primaryField(entity types.Entity) string {
	names := sortedFieldNames(entity.Fields)
	for _, name := range names {
		if name == "id" {
			continue
		}
		if entity.Fields[name].References != "" {
			continue
		}
		return name
	}
	return ""
}

// secondaryField — следующее поле после primary, по тому же критерию.
func secondaryField(entity types.Entity) string {
	names := sortedFieldNames(entity.Fields)
	seen := false
	for _, name := range names {
		if name == "id" {
			continue
		}
		if entity.Fields[name].References != "" {
			continue
		}
		if !seen {
			seen = true
			continue
		}
		return name
	}
	return ""
}

func sortedFieldNames(fields map[string]types.Field) []string {
	out := make([]string, 0, len(fields))
	for name := range fields {
		out = append(out, name)
	}
	// stable lexicographic sort
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}
```

**ВНИМАНИЕ:** `primaryField` heuristic использует lexicographic order имён. Это даст для Book: fields = [author, id, isbn, title] → first non-id non-FK = "author", second = "isbn". Но fixture expected itemDisplay для book-catalog = `{primary: "title", secondary: "author"}`.

Это противоречит. Спека говорит «первое поле в `ontology.entities[entity].fields` (по key order JSON-объекта)». JSON порядок не сохраняется в Go map. Значит нужно сохранять порядок при парсинге.

**Решение:** добавить в `types.Entity` поле `FieldsOrder []string`, заполняемое в parser через `json.Decoder.Token()` (низкоуровневое чтение JSON). Это требует переработки parser.

**Альтернатива (проще):** использовать `github.com/iancoleman/orderedmap` или `github.com/wk8/go-ordered-map`. Но это новая dependency.

**Принятое решение для v1.0:** заполнять `Entity.FieldsOrder` при парсинге через альтернативный путь — `encoding/json.RawMessage` + manual order extraction. Это reasonable.

**Изменение в types/ontology.go:**

```go
type Entity struct {
	Kind        string           `json:"kind,omitempty"`
	OwnerField  string           `json:"ownerField,omitempty"`
	Fields      map[string]Field `json:"fields"`
	FieldsOrder []string         `json:"-"` // заполняется парсером
}
```

**Изменение в parser/parser.go:** в `ParseOntology` после `json.Unmarshal(data, &ont)` дополнительно прочитать порядок ключей в `entities[E].fields`:

```go
// Добавить функцию extractFieldsOrder в parser/parser.go:
func extractFieldsOrder(data []byte, ont *types.Ontology) error {
	var raw struct {
		Entities map[string]json.RawMessage `json:"entities"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for entityName, entRaw := range raw.Entities {
		var entObj struct {
			Fields json.RawMessage `json:"fields"`
		}
		if err := json.Unmarshal(entRaw, &entObj); err != nil {
			continue
		}
		order, err := jsonObjectKeysInOrder(entObj.Fields)
		if err != nil {
			continue
		}
		entity := ont.Entities[entityName]
		entity.FieldsOrder = order
		ont.Entities[entityName] = entity
	}
	return nil
}

// jsonObjectKeysInOrder возвращает имена ключей JSON-объекта в порядке
// их появления.
func jsonObjectKeysInOrder(data []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, fmt.Errorf("expected object, got %v", tok)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %v", tok)
		}
		keys = append(keys, key)
		// Skip value
		if err := skipValue(dec); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

func skipValue(dec *json.Decoder) error {
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if delim, ok := tok.(json.Delim); ok {
			switch delim {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
				if depth <= 0 {
					return nil
				}
			}
			continue
		}
		if depth == 0 {
			return nil
		}
	}
}
```

В `ParseOntology` после `json.Unmarshal`:
```go
if err := extractFieldsOrder(data, &ont); err != nil {
	return types.Ontology{}, fmt.Errorf("parser: extract fields order: %w", err)
}
```

И добавить `import "bytes"` в parser/parser.go.

Также обновить `primaryField`/`secondaryField` в crystallize/dispatch.go использовать `entity.FieldsOrder` вместо lexicographic:

```go
func primaryField(entity types.Entity) string {
	for _, name := range entity.FieldsOrder {
		if name == "id" {
			continue
		}
		if entity.Fields[name].References != "" {
			continue
		}
		return name
	}
	return ""
}

func secondaryField(entity types.Entity) string {
	seen := false
	for _, name := range entity.FieldsOrder {
		if name == "id" {
			continue
		}
		if entity.Fields[name].References != "" {
			continue
		}
		if !seen {
			seen = true
			continue
		}
		return name
	}
	return ""
}

// Удалить sortedFieldNames — больше не нужна.
```

- [ ] **Step 2: Применить изменения в types/ontology.go и parser/parser.go**

Изменения per Step 1.

- [ ] **Step 3: Re-run parser tests**

Run: `cd ~/WebstormProjects/idf-go && go test ./parser/ -v`
Expected: pass (не сломались).

Дополнительно убедиться, что FieldsOrder заполнен для library:
- Book: ["id", "title", "author", "isbn"]
- User: ["id", "name"]
- Loan: ["id", "userId", "bookId", "status", "borrowedAt", "returnedAt"]

Можно добавить assertion в `parser_test.go::TestParseOntology`:
```go
expectedOrder := map[string][]string{
	"User": {"id", "name"},
	"Book": {"id", "title", "author", "isbn"},
	"Loan": {"id", "userId", "bookId", "status", "borrowedAt", "returnedAt"},
}
for name, want := range expectedOrder {
	got := ont.Entities[name].FieldsOrder
	if len(got) != len(want) {
		t.Errorf("entity %s FieldsOrder len: got %d, want %d", name, len(got), len(want))
		continue
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("entity %s FieldsOrder[%d]: got %s, want %s", name, i, got[i], want[i])
		}
	}
}
```

- [ ] **Step 4: `crystallize/slots_catalog.go`**

```go
package crystallize

import (
	"sort"

	"idf-go/internal/jsonutil"
	"idf-go/types"
)

// slotsCatalog заполняет slots для catalog-архетипа согласно
// spec/04-algebra/crystallize.md фаза 3.
//
// header.title          ← projection.entity (или fallback projection.id)
// body.items            ← Object.values(viewerWorld[entity]) sorted by id ASC
// body.itemDisplay      ← {primary, secondary} heuristic'ом
// footer.actions        ← non-create intents проекции, accessible viewer'у, sorted by intent.id ASC
// toolbar.create        ← create-intent (если есть в проекции и accessible) или toolbar = {}
func slotsCatalog(
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) map[string]any {
	role := ont.Roles[viewer.Role]

	header := map[string]any{
		"title": coalesce(proj.Entity, proj.ID),
	}

	body := map[string]any{
		"items": catalogItems(proj.Entity, vw),
	}
	if proj.Entity != "" {
		entity := ont.Entities[proj.Entity]
		body["itemDisplay"] = map[string]any{
			"primary":   primaryField(entity),
			"secondary": secondaryField(entity),
		}
	}

	footerActions := []any{}
	toolbar := map[string]any{}

	// Сортируем intent.ids проекции
	intentIDs := make([]string, len(proj.Intents))
	copy(intentIDs, proj.Intents)
	sort.Strings(intentIDs)

	for _, id := range intentIDs {
		intent, ok := intentByID[id]
		if !ok {
			continue
		}
		if !roleCanExecute(role, id) {
			continue
		}
		if isCreateIntent(intent) {
			toolbar["create"] = map[string]any{
				"intentId": intent.ID,
				"label":    intent.ID,
			}
		} else {
			footerActions = append(footerActions, map[string]any{
				"intentId": intent.ID,
				"label":    intent.ID,
			})
		}
	}

	return map[string]any{
		"header":  header,
		"body":    body,
		"footer":  map[string]any{"actions": footerActions},
		"toolbar": toolbar,
	}
}

// catalogItems возвращает массив записей entity из viewerWorld,
// упорядоченных по id ASC.
func catalogItems(entityName string, vw types.ViewerWorld) []any {
	if entityName == "" {
		return []any{}
	}
	ns := vw[entityName]
	if ns == nil {
		return []any{}
	}
	ids := make([]string, 0, len(ns))
	for id := range ns {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, jsonutil.DeepCopyMap(ns[id]))
	}
	return items
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func isCreateIntent(intent types.Intent) bool {
	return len(intent.Effects) > 0 && intent.Effects[0].Kind == "create"
}
```

- [ ] **Step 5: Verify build**

Run: `cd ~/WebstormProjects/idf-go && go build ./crystallize/`
Expected: успех.

- [ ] **Step 6: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add types/ontology.go parser/ crystallize/dispatch.go crystallize/slots_catalog.go crystallize/stubs.go && git commit -q -m "crystallize: фаза 3 dispatcher + slotsCatalog; FieldsOrder в Entity для primary/secondary heuristic"
```

### Task 11: `crystallize/slots_detail.go` + `slots_form.go` + `slots_dashboard.go` + `slots_lightly.go`

**Files:**
- Create: 4 файла

- [ ] **Step 1: `slots_detail.go`**

```go
package crystallize

import (
	"sort"

	"idf-go/internal/jsonutil"
	"idf-go/types"
)

// slotsDetail заполняет slots для detail-архетипа (фаза 3).
//
// Запись для detail: первая по id ASC из viewerWorld[entity] (Q-23).
//
// header.title       ← record[primaryField]
// body.fields        ← массив {name, value} по entity.FieldsOrder
// footer.actions     ← все доступные intents, sorted by intent.id ASC
func slotsDetail(
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) map[string]any {
	role := ont.Roles[viewer.Role]
	entity := ont.Entities[proj.Entity]

	// Выбор записи: первая по id ASC
	ns := vw[proj.Entity]
	ids := make([]string, 0, len(ns))
	for id := range ns {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var record map[string]any
	if len(ids) > 0 {
		record = ns[ids[0]]
	}

	primary := primaryField(entity)
	header := map[string]any{}
	if record != nil && primary != "" {
		if v, ok := record[primary]; ok {
			header["title"] = v
		}
	}

	// body.fields: по entity.FieldsOrder, оставляя только присутствующие в record
	bodyFields := []any{}
	if record != nil {
		for _, name := range entity.FieldsOrder {
			if v, ok := record[name]; ok {
				bodyFields = append(bodyFields, map[string]any{
					"name":  name,
					"value": jsonutil.DeepCopy(v),
				})
			}
		}
	}

	// footer.actions: все доступные viewer'у intents проекции, sorted
	intentIDs := make([]string, len(proj.Intents))
	copy(intentIDs, proj.Intents)
	sort.Strings(intentIDs)

	actions := []any{}
	for _, id := range intentIDs {
		intent, ok := intentByID[id]
		if !ok || !roleCanExecute(role, id) {
			continue
		}
		actions = append(actions, map[string]any{
			"intentId": intent.ID,
			"label":    intent.ID,
		})
	}

	return map[string]any{
		"header": header,
		"body":   map[string]any{"fields": bodyFields},
		"footer": map[string]any{"actions": actions},
	}
}
```

- [ ] **Step 2: `slots_form.go`**

```go
package crystallize

import (
	"idf-go/types"
)

// slotsForm заполняет slots для form-архетипа (фаза 3).
//
// header.title       ← intent.id если intents.length === 1 иначе projection.id
// body.fields        ← массив {name, type, required: true} из intent.requiredFields
// footer.submit      ← {intentId, label} для main intent
func slotsForm(
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) map[string]any {
	var mainIntent types.Intent
	if len(proj.Intents) > 0 {
		mainIntent = intentByID[proj.Intents[0]]
	}

	title := proj.ID
	if len(proj.Intents) == 1 && mainIntent.ID != "" {
		title = mainIntent.ID
	}

	bodyFields := []any{}
	for _, rf := range mainIntent.RequiredFields {
		bodyFields = append(bodyFields, map[string]any{
			"name":     rf.Name,
			"type":     rf.Type,
			"required": true,
		})
	}

	submit := map[string]any{}
	if mainIntent.ID != "" {
		submit = map[string]any{
			"intentId": mainIntent.ID,
			"label":    mainIntent.ID,
		}
	}

	return map[string]any{
		"header": map[string]any{"title": title},
		"body":   map[string]any{"fields": bodyFields},
		"footer": map[string]any{"submit": submit},
	}
}
```

- [ ] **Step 3: `slots_dashboard.go`**

```go
package crystallize

import (
	"sort"

	"idf-go/types"
)

// slotsDashboard заполняет slots для dashboard-архетипа (фаза 3).
//
// header.title          ← projection.id
// body.sections         ← [] (composition Reserved L4)
// toolbar.actions       ← массив {intentId, label} для intent'ов проекции,
//                         sorted by intent.id ASC
func slotsDashboard(
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) map[string]any {
	role := ont.Roles[viewer.Role]

	intentIDs := make([]string, len(proj.Intents))
	copy(intentIDs, proj.Intents)
	sort.Strings(intentIDs)

	actions := []any{}
	for _, id := range intentIDs {
		intent, ok := intentByID[id]
		if !ok || !roleCanExecute(role, id) {
			continue
		}
		actions = append(actions, map[string]any{
			"intentId": intent.ID,
			"label":    intent.ID,
		})
	}

	return map[string]any{
		"header":  map[string]any{"title": proj.ID},
		"body":    map[string]any{"sections": []any{}},
		"toolbar": map[string]any{"actions": actions},
	}
}
```

- [ ] **Step 4: `slots_lightly.go`**

```go
package crystallize

import (
	"idf-go/types"
)

// slotsLightly заполняет минимальную нормативную структуру для
// Lightly-tested архетипов (feed, canvas, wizard).
// Conformance не проверяется fixture-вектором; реализация даёт минимум,
// упомянутый в spec/03-objects/artifact.md.
func slotsLightly(
	archetype string,
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) map[string]any {
	switch archetype {
	case "feed":
		return map[string]any{
			"body": map[string]any{"entries": []any{}},
		}
	case "canvas":
		return map[string]any{
			"body": map[string]any{"canvasRef": proj.ID},
		}
	case "wizard":
		return map[string]any{
			"body": map[string]any{"steps": []any{}},
		}
	}
	return map[string]any{}
}
```

- [ ] **Step 5: Verify build**

Run: `cd ~/WebstormProjects/idf-go && go build ./crystallize/`
Expected: успех.

- [ ] **Step 6: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add crystallize/slots_detail.go crystallize/slots_form.go crystallize/slots_dashboard.go crystallize/slots_lightly.go && git commit -q -m "crystallize: фаза 3 slots для detail/form/dashboard + минимум для feed/canvas/wizard"
```

### Task 12: `crystallize/confirmation.go` — фаза 6

**Files:**
- Create: `crystallize/confirmation.go`
- Modify: `crystallize/stubs.go` (удалить wrapByConfirmation stub)

- [ ] **Step 1: `crystallize/confirmation.go`**

```go
package crystallize

import (
	"idf-go/types"
)

// wrapByConfirmation реализует фазу 6 (см. spec/04-algebra/crystallize.md).
//
// Для каждого intent reference в slots.{footer.actions, toolbar.create,
// toolbar.actions, footer.submit} добавляет поле confirmation:
//   - intent.effects[0].kind == "remove" → "destructive"
//   - иначе → "standard"
func wrapByConfirmation(slots map[string]any, intentByID map[string]types.Intent) {
	walkIntentRefs(slots, func(ref map[string]any) {
		id, _ := ref["intentId"].(string)
		intent, ok := intentByID[id]
		if !ok {
			return
		}
		ref["confirmation"] = confirmationLevel(intent)
	})
}

func confirmationLevel(intent types.Intent) string {
	if len(intent.Effects) > 0 && intent.Effects[0].Kind == "remove" {
		return "destructive"
	}
	return "standard"
}

// walkIntentRefs вызывает fn на каждом объекте, содержащем intentId,
// в slots-tree.
func walkIntentRefs(node any, fn func(map[string]any)) {
	switch v := node.(type) {
	case map[string]any:
		if _, isRef := v["intentId"]; isRef {
			fn(v)
			return
		}
		for _, child := range v {
			walkIntentRefs(child, fn)
		}
	case []any:
		for _, item := range v {
			walkIntentRefs(item, fn)
		}
	}
}
```

- [ ] **Step 2: Удалить `wrapByConfirmation` stub**

```go
// Полностью удалить файл crystallize/stubs.go
```

Run: `rm ~/WebstormProjects/idf-go/crystallize/stubs.go`

- [ ] **Step 3: Verify build**

Run: `cd ~/WebstormProjects/idf-go && go build ./crystallize/`
Expected: успех.

- [ ] **Step 4: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add crystallize/confirmation.go && git rm crystallize/stubs.go && git commit -q -m "crystallize: фаза 6 wrapByConfirmation; удалены stubs"
```

---

## Phase 8: crystallize integration test

### Task 13: `crystallize/crystallize_test.go` — full pipeline на 8 expected/artifact

**Files:**
- Create: `crystallize/crystallize_test.go`

- [ ] **Step 1: `crystallize_test.go`**

```go
package crystallize

import (
	"encoding/json"
	"path/filepath"
	"strings"
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
				t.Errorf("artifact mismatch for %s\ngot:\n  %s\nwant:\n  %s",
					tc.file,
					strings.ReplaceAll(string(gotPretty), "\n", "\n  "),
					strings.ReplaceAll(string(expPretty), "\n", "\n  "))
			}
		})
	}
}
```

- [ ] **Step 2: Run — likely will fail in some scenarios on first try**

Run: `cd ~/WebstormProjects/idf-go && go test ./crystallize/ -v`
Expected: возможны failures из-за edge cases в slot-логике (например, пустой toolbar, sorting различия). Каждый failure — diff между got и want. Исправлять spot:
- Если actions/items в неверном порядке → fix sort
- Если toolbar={} vs toolbar отсутствует → fix slot output (всегда возвращать toolbar key)
- Если intentId/label/confirmation мисс одного поля — fix wrapByConfirmation walk

- [ ] **Step 3: Iterate to all-pass**

Каждый fix → re-run → goto next. Итерация до 8/8 sub-tests pass.

Distinguished known fixes (предсказуемые):

**Fix A: catalog без proj.Entity (librarian-dashboard сценарий — но это dashboard, не catalog)**: применимо к catalog, не dashboard. Если у catalog Entity="" — body.itemDisplay не должен быть. Проверить slotsCatalog: уже сделано (`if proj.Entity != ""`).

**Fix B: form-архетип с slot-override `_authored: true` — после mergeSlots должен сохранить this flag**: проверить mergeSlots.

**Fix C: dashboard может НЕ возвращать footer key вообще** — fixture показывает только header, body, toolbar для dashboard. Проверить slotsDashboard — он не возвращает footer, что match'ится.

- [ ] **Step 4: All-pass commit**

```bash
cd ~/WebstormProjects/idf-go && git add crystallize/ && git commit -q -m "crystallize: integration test 8/8 (scenario × projection × viewer) pass"
```

---

## Phase 9: CLI conformance

### Task 14: `cmd/conformance/main.go`

**Files:**
- Create: `cmd/conformance/main.go`

- [ ] **Step 1: `cmd/conformance/main.go`**

```go
// Command conformance прогоняет L1+L2 conformance check на указанной
// директории fixtures и печатает human-readable отчёт.
//
// Usage: conformance <path-to-fixtures-dir>
//
// Где path-to-fixtures-dir — каталог в формате:
//   <dir>/
//     ontology.json
//     intents.json
//     projections.json
//     phi/<scenario>.json
//     expected/world/<scenario>.json
//     expected/viewer-world/<scenario>-as-<role>-<id>.json
//     expected/artifact/<scenario>-<projection>-as-<role>-<id>.json
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

	// specRoot = fixturesDir/../..
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
	for sc := range scenarios {
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
		var gotMap map[string]any
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
		base := basename(f) // e.g. "borrow-cycle-as-reader-u-r1"
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
		base := basename(f) // e.g. "bootstrap-book-catalog-as-librarian-u-lib-1"
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

	// Overall
	fmt.Println()
	if allPass {
		fmt.Println("== OVERALL: L1+L2 CONFORMANT ==")
		os.Exit(0)
	}
	fmt.Println("== OVERALL: FAILURES ==")
	os.Exit(1)
}

func basename(f string) string {
	b := filepath.Base(f)
	return strings.TrimSuffix(b, ".json")
}

// parseVWName разбирает <scenario>-as-<role>-<id> в (scenario, viewer).
// Используем list of known role-names из ontology для определения, где
// scenario заканчивается и role начинается.
func parseVWName(name string, ont types.Ontology) (string, types.Viewer, bool) {
	// Шаблон: <scenario>-as-<role>-<id>
	const sep = "-as-"
	idx := strings.Index(name, sep)
	if idx < 0 {
		return "", types.Viewer{}, false
	}
	scenario := name[:idx]
	rest := name[idx+len(sep):]
	// rest = "<role>-<id>"
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
	// prefix = "<scenario>-<projection>"; нужно догадаться, где split.
	// Перебор: для каждого projection.id проверить, заканчивается ли prefix на "-<projection>".
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
```

- [ ] **Step 2: Build**

Run: `cd ~/WebstormProjects/idf-go && go build -o conformance ./cmd/conformance/`
Expected: создан binary `./conformance`.

- [ ] **Step 3: Run on library**

Run: `cd ~/WebstormProjects/idf-go && ./conformance ../idf-spec/spec/fixtures/library/`
Expected output (примерно):

```
== Step 1: Parser ==
  ontology.json: OK
  intents.json: OK
  projections.json: OK
  phi/*.json: 7/7 OK
== Step 2: fold ==
  7/7 scenarios: world matches expected
== Step 3: filterWorldForRole ==
  18/18 (scenario × viewer): viewerWorld matches expected
== Step 4: crystallize ==
  8/8 (scenario × projection × viewer): artifact matches expected

== OVERALL: L1+L2 CONFORMANT ==
```

Exit code 0.

- [ ] **Step 4: Cleanup binary + commit**

```bash
cd ~/WebstormProjects/idf-go && rm -f conformance && git add cmd/ && git commit -q -m "cmd/conformance: CLI orchestration; library fixtures L1+L2 conformant"
```

---

## Phase 10: Backlog + tag

### Task 15: `feedback/spec-v0.1.md` — backlog ambiguities

**Files:**
- Create: `feedback/spec-v0.1.md`

- [ ] **Step 1: `feedback/spec-v0.1.md`**

Outline (заполняется по ходу реализации; финализируется здесь):

```markdown
# Backlog ambiguities `spec-v0.1` (выявлены при написании idf-go)

Этот файл — **не часть idf-go**. Это feedback автору спеки: места, где
spec-v0.1 потребовала угадывания или принятия implementer's choice
из-за неоднозначности или отсутствия нормативного правила.

## Категория A: Implementer choices, требующие normative resolution в v0.2

### A-1: Сохранение порядка ключей в `entities[E].fields` для primary/secondary heuristic

**Спека (Q-12, Q-24):** primary field — «первое поле в `ontology.entities[entity].fields` (по key order JSON-объекта)».

**Проблема:** Go `map[string]Field` теряет порядок ключей при стандартном `json.Unmarshal`. Implementer вынужден читать JSON через `json.Decoder.Token()` чтобы извлечь порядок имён.

**Implementer choice:** добавлено поле `Entity.FieldsOrder []string`, заполняемое отдельным проходом парсера.

**Предложение для v0.2:** либо явно зафиксировать в спеке, что implementer MUST сохранять JSON object key order для `entities.fields` (дав alternative — explicit `fieldsOrder []string` в JSON Schema), либо изменить heuristic primary/secondary на explicit declaration `entity.primary: "fieldName"`, `entity.secondary: "fieldName"`.

### A-2: ...
(Заполнить по ходу: перечислить все edge cases, где спека была неоднозначна и потребовала ad-hoc решения.)

## Категория B: Реальные баги спеки (если найдены)

(Заполнить если в процессе обнаружены fixtures, которые не соответствуют нормативному алгоритму спеки. Ноль если всё валидно.)

## Категория C: Documentation gaps

(Заполнить, где спека nominally нормативна, но проза неполна и приходилось дочитывать в Open questions / cross-references.)

## Метрики честности эксперимента

- Время реализации: <Х часов>
- Количество iterations при failing tests: <Y>
- Open questions, потребовавшие угадывания: <Z>
- Места, где соблазн открыть исходники первой реализации был сильным: <W>
- Опасные знания из памяти, которые были подавлены: <список>

## Итог

(1-2 предложения: формат decoupled от React-specifics? Спека достаточна?)
```

- [ ] **Step 2: Заполнить категорию A полностью**

Перечислить ВСЕ implementer choices, сделанные в коде:
- A-1: FieldsOrder для primary/secondary
- A-2: Sort items в catalog по id ASC
- A-3: Detail recordId — первая запись по id ASC
- A-4: form header.title — intent.id если intents.length===1, иначе projection.id
- A-5: dashboard.body.sections = [] (composition Reserved)
- A-6: feed/canvas/wizard минимальная структура (сделано in slotsLightly)
- A-7: VisibleFieldsValue UnmarshalJSON: либо строка "*" либо массив
- A-8: parseVWName и parseArtName в CLI используют heuristic для split <scenario>-<viewer>; альтернатива — JSON metadata в каждом expected file (например `_meta.scenario`, `_meta.viewer`)
- A-9: any other discovered during impl

- [ ] **Step 3: Финализировать «Метрики честности эксперимента»**

Self-audit на основе git log + transcript:
- Сколько часов: оценка
- Iterations: подсчитать commits в Phase 8 (crystallize integration)
- Open questions потребовавшие угадывания: список из A-1..A-N
- Места соблазна открыть src: ad-hoc memory, описать честно
- Опасные знания подавлены: «помнил, что в первой реализации есть R8 hub-absorption, но spec-v0.1 не нормирует — намеренно проигнорировал»

- [ ] **Step 4: Commit**

```bash
cd ~/WebstormProjects/idf-go && git add feedback/ && git commit -q -m "feedback: spec-v0.1.md — backlog implementer choices + метрики честности"
```

### Task 16: Final smoke + tag v0.1.0

**Files:** verification only

- [ ] **Step 1: Полный test suite**

Run: `cd ~/WebstormProjects/idf-go && go test ./... -v 2>&1 | tail -40`
Expected: ВСЕ тесты pass.

- [ ] **Step 2: Полный CLI run**

Run: `cd ~/WebstormProjects/idf-go && go run ./cmd/conformance ../idf-spec/spec/fixtures/library/`
Expected: `OVERALL: L1+L2 CONFORMANT`, exit 0.

- [ ] **Step 3: Cross-references check (паранойя)**

Прежде чем тагать, убедиться что `idf-go/CLAUDE.md` черный список не нарушался во время работы. Просмотр transcript'а на наличие Read/Grep/Glob к запрещённым путям. Если что-то нашлось — описать в `feedback/spec-v0.1.md` секции «Метрики честности».

- [ ] **Step 4: git status clean + tag**

Run:
```bash
cd ~/WebstormProjects/idf-go && git status && git log --oneline | head -20
```

Expected: working tree clean, ~16 коммитов суммарно.

```bash
cd ~/WebstormProjects/idf-go && git tag -a v0.1.0 -m "idf-go v0.1.0 — L1+L2 conformant с spec-v0.1 на library

Вторая референсная реализация формата IDF на Go. Built исключительно
по spec-v0.1 без чтения исходников первой реализации (idf/, idf-sdk/).

Conformance:
- L1: parser (5 core схем + 3 wrapper) + Φ + fold + filterWorldForRole
- L2: crystallize 6 фаз (4-5 noop) + 4 архетипа (catalog/detail/form/dashboard) + mergeProjections

Все fixture-векторы pass:
- 7/7 phi-сценариев → world
- 18/18 (scenario × viewer) → viewerWorld
- 8/8 (scenario × projection × viewer) → artifact

Стек: Go 1.22+ + xeipuuv/gojsonschema (draft-07).

Backlog для spec-v0.2: feedback/spec-v0.1.md."
```

- [ ] **Step 5: Verify tag**

Run: `cd ~/WebstormProjects/idf-go && git tag --list && git log --oneline | wc -l`
Expected: tag `v0.1.0`, ~16-18 коммитов.

---

## Self-review плана

**1. Spec coverage:** каждое требование design doc'а покрыто:

- §4 Scope L1: parser (Task 3) + fold (Task 6) + filter (Task 7) ✓
- §4 Scope L2: crystallize 6 фаз (Tasks 8-13) ✓
- §5 Архитектура — все 6 пакетов созданы (types/, parser/, fold/, filter/, crystallize/, internal/jsonutil/) + cmd/conformance/ ✓
- §6 Стек: Go + gojsonschema; std lib only — соблюдается ✓
- §7 Чтение fixtures: relative path `../../idf-spec/spec/fixtures/library/` в тестах, `../..` в CLI — согласовано ✓
- §8 Решения по неоднозначностям: учтены в коде (no-op remove, primaryField heuristic, sort by id ASC, etc.) + зафиксированы в Task 15 backlog ✓
- §10 Критерий успеха: все 4 шага проверяются Tasks 6/7/13/14 + CLI ✓

**2. Placeholder scan:**
- Task 15 §A-2: ... (заполнить по ходу) — это **placeholder в плане**! Fix: подменю на конкретный список A-1..A-9 уже сейчас (выше так и сделал).
- Все остальные «выявить если найдены» — это нормальные конструкции «зависит от observed bugs», не placeholder'ы. Допустимо.

**3. Type consistency:**
- `types.World` = `map[string]map[string]map[string]any` — везде согласовано ✓
- `types.ViewerWorld = World` (alias) — согласовано ✓
- `types.Viewer{Role, ID}` — везде ✓
- `Crystallize(intents, ont, proj, viewer, vw)` — сигнатура одна по всем тестам ✓
- Method names: `Fold`, `FilterWorldForRole`, `Crystallize` — exported, согласованы ✓

**4. Известные риски при выполнении:**

- gojsonschema может ругаться на формат `date-time` так же как ajv ругался — нужно обработать gracefully (warning, not error). Если tests fall — добавить опцию игнорирования format
- `parseVWName`/`parseArtName` heuristic в CLI могут давать неправильный split, если role-name содержит `-as-` — для library не проблема (reader, librarian), но для будущих доменов — риск. Альтернатива: добавить `_meta.scenario`, `_meta.viewer.role`, `_meta.viewer.id` в expected fixtures. Это можно зафиксировать в backlog как A-8.
- Stub'ы между Task 8 и Task 12 — temporary; final state стейт чистый.
- Phase 8 (Task 13) integration test может потребовать многократных iterations — 8 fixtures × edge cases. План явно учитывает это (Step 3 «Iterate to all-pass»).

---

## Plan complete

Сохранён: `idf-go/plans/2026-04-19-idf-go-plan.md`.

## Execution choice

Auto mode active. Перехожу к **inline execution** через `superpowers:executing-plans`.
