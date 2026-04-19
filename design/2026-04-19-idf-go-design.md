# Дизайн `idf-go` — вторая референсная реализация формата IDF

**Дата:** 2026-04-19
**Статус:** дизайн-документ (брэйнсторм). Согласован с автором; следующий шаг — implementation plan через `writing-plans`.
**Source of truth:** `~/WebstormProjects/idf-spec/spec/` (нормативная спека v0.1, conformance L1+L2).

---

## 1. Цель и место в эксперименте

Манифест IDF v2 §26 фиксирует «вторая reference implementation» как структурный стресс-тест формата на decoupling от React-specifics первой реализации. После написания `spec-v0.1` (нормативной спеки уровня OpenAPI/JSON-LD) реализация на ином стеке имеет валидационный смысл: она проверяет, что формат описан достаточно полно и однозначно, чтобы independent implementer мог собрать конформную систему **без чтения исходников первой реализации**.

`idf-go` — эта вторая реализация. Стек: Go.

## 2. Critical-path сценарий

```bash
cd ~/WebstormProjects/idf-go
go test ./...                                                # все table-driven тесты
go run ./cmd/conformance ../idf-spec/spec/fixtures/library/  # human-readable отчёт
```

Pass всех тестов = **L1 + L2 conformant** против `spec-v0.1`.

## 3. Изоляционная политика

Полный текст — в [`CLAUDE.md`](../CLAUDE.md). Кратко:

**Allowed reads:**
- `~/WebstormProjects/idf-spec/spec/**` (нормативная спека — main source)
- `~/WebstormProjects/idf-spec/feedback/manifesto-v2.md` (informative)
- `~/WebstormProjects/idf-spec/source/manifesto-v2.snapshot.md` (frozen snapshot манифеста)

**Forbidden reads:**
- `~/WebstormProjects/idf/{src,server,scripts}/**` — исходники первой реализации
- `~/WebstormProjects/idf-sdk/packages/*/src/**` — SDK-исходники
- `*.test.{js,jsx,ts,tsx,cjs}` в обоих репо
- `~/WebstormProjects/idf/docs/{implementation-status.md,archive/}**`
- `~/WebstormProjects/idf-spec/{design,plans}/**` — authorial process спеки (это для авторов спеки, не для implementer'а)

**Честная оговорка:** автор реализации (LLM в этой сессии) не идеальный independent implementer — в контексте сессии уже присутствует `CLAUDE.md` проекта `idf` с описанием архитектуры. Полностью «забыть» это нельзя. Обязательства:

1. Не открывать source-файлы первой реализации (черёз tool-уровневую проверку — Read/Grep/Glob)
2. Не использовать имплементационные имена/числа из памяти (имена файлов, версии пакетов, конкретные функции, число доменов прототипа и т.п.)
3. Treat `spec-v0.1` как sole normative input

Это imperfect, но best-effort approximation независимого эксперимента.

## 4. Scope v1.0

### В скоупе

- **L1 нормативно полностью**:
  - Парсер 5 core-схем (`ontology`, `intent`, `effect`, `projection`, `artifact`) — JSON Schema validation + Go-typed decode
  - Парсер wrapper-схем для fixtures (`phi`, `intents-collection`, `projections-collection`)
  - Φ как append-only массив confirmed effects, упорядоченных по `context.at` ASC (stable tie-breaker — позиция в массиве)
  - `fold(Φ, ontology) → world` согласно [`spec/04-algebra/fold.md`](../../idf-spec/spec/04-algebra/fold.md)
  - `filterWorldForRole(world, viewer, ontology) → viewerWorld` — 3-приоритетный row-filter согласно [`spec/04-algebra/filter-world.md`](../../idf-spec/spec/04-algebra/filter-world.md)

- **L2 нормативно полностью**:
  - `crystallize(intents, ontology, projection, viewer, viewerWorld) → artifact` — 6 фаз pipeline согласно [`spec/04-algebra/crystallize.md`](../../idf-spec/spec/04-algebra/crystallize.md)
  - Фазы 4-5 (`matchPatterns`, `applyStructuralPatterns`) — noop (Pattern Bank Reserved L4)
  - 7 архетипов: `feed`, `catalog`, `detail`, `form`, `canvas`, `dashboard`, `wizard`
    - 4 покрыты fixture-вектором (catalog, detail, form, dashboard) — обязательно работают
    - 3 lightly-tested (feed, canvas, wizard) — структурно нормированы, без conformance-проверки
  - `mergeProjections` (фаза 2) с array-replace семантикой
  - `wrapByConfirmation` (фаза 6) с destructive/standard эвристикой

- **CLI `cmd/conformance`** — single-binary, читает `idf-spec/spec/fixtures/<domain>/`, прогоняет все 4 шага, печатает per-step pass/fail и общий итог.

- **Table-driven Go-тесты** (`conformance_test.go` в каждом пакете), читающие fixtures из `../idf-spec/spec/fixtures/library/` через relative path. Запускаются `go test ./...`.

### Вне скоупа v1.0

- **L3** материализации (pixel/voice/agent API/document) — Reserved в спеке
- **L4** (Pattern Bank apply, темпоральный scheduler, irreversibility integrity rule, 5 видов invariant'ов с handler'ами) — Reserved в спеке
- HTTP server, REST/gRPC API, WebSocket
- Persistence: SQLite, Postgres, append-only files — Φ только in-memory из JSON-input
- Production-grade observability: structured logging, metrics, tracing — минимум `fmt.Errorf`
- Authentication / authorization — viewer-scoping реализуется, но JWT/OAuth не нормированы
- Concurrency / multi-process — single-threaded достаточно для conformance

## 5. Архитектура

### Структура репо

```
idf-go/
├── go.mod                          # module idf-go
├── go.sum
├── README.md
├── CLAUDE.md                       # изоляционная политика
├── design/
│   └── 2026-04-19-idf-go-design.md # этот файл
├── plans/                          # implementation plan (после writing-plans)
├── feedback/
│   └── spec-v0.1.md                # backlog ambiguities спеки (заполняется по ходу)
│
├── types/                          # Go struct'ы для 5 core-объектов + helpers
│   ├── ontology.go                 # Ontology, Entity, Field, Role
│   ├── intent.go                   # Intent, ProtoEffect
│   ├── effect.go                   # Effect, Context (с at, initiator, intentId, __irr)
│   ├── projection.go               # Projection (slot-override map)
│   ├── artifact.go                 # Artifact (slots map[string]any)
│   ├── world.go                    # World, ViewerWorld type aliases
│   └── viewer.go                   # Viewer struct {Role, ID}
│
├── parser/
│   ├── parser.go                   # Parse* функции (Ontology/Intents/Projections/Phi)
│   └── schemas.go                  # embed JSON schemas (через //go:embed)
│
├── fold/
│   └── fold.go                     # Fold(phi []Effect, ontology Ontology) (World, error)
│
├── filter/
│   └── filter.go                   # FilterWorldForRole(world, viewer, ontology) ViewerWorld
│
├── crystallize/
│   ├── crystallize.go              # Crystallize(...) (Artifact, error) — главный entry
│   ├── derive.go                   # фаза 1: deriveProjections (auto archetype heuristic)
│   ├── merge.go                    # фаза 2: mergeProjections (deep-merge с array-replace)
│   ├── slots_catalog.go            # фаза 3: assignToSlots для catalog
│   ├── slots_detail.go             # фаза 3: detail
│   ├── slots_form.go               # фаза 3: form
│   ├── slots_dashboard.go          # фаза 3: dashboard
│   ├── slots_lightly.go            # фаза 3: feed/canvas/wizard минимальные
│   └── confirmation.go             # фаза 6: wrapByConfirmation
│
├── internal/
│   └── jsonutil/                   # semantic equality, deep-copy, JSON helpers
│       ├── equal.go                # SemanticEqual(a, b any) bool
│       └── deepcopy.go             # DeepCopy(v any) any
│
├── cmd/
│   └── conformance/
│       └── main.go                 # CLI: parse args (path к fixtures dir), run 4 шага, print
│
└── conformance_test.go             # top-level integration test, диспатчит на pakets
```

### Decoupling: каждый пакет — одна ответственность

| Пакет | Что делает | Зависит от |
|---|---|---|
| `types` | Struct-определения, без логики | std lib |
| `parser` | JSON Schema валидация + decode в `types` | `types`, `gojsonschema` |
| `fold` | Чистая функция `fold(phi, ontology) → world` | `types`, `internal/jsonutil` |
| `filter` | Чистая функция `filter(world, viewer, ontology) → viewerWorld` | `types`, `internal/jsonutil` |
| `crystallize` | Pipeline `(intents, ontology, projection, viewer, viewerWorld) → artifact` | `types`, `internal/jsonutil` |
| `cmd/conformance` | CLI orchestration | все пакеты выше |

Никакого внутреннего state, никаких глобальных переменных, никаких init() с сайд-эффектами.

### Тесты

Каждый пакет имеет `*_test.go` с table-driven тестами, читающими fixtures из `../idf-spec/spec/fixtures/library/`:

- `parser/parser_test.go` — каждый core fixture file должен парситься без ошибок
- `fold/fold_test.go` — для каждого `phi/<scenario>.json`: `fold(phi.effects, ontology)` deep-equal `expected/world/<scenario>.json`
- `filter/filter_test.go` — для каждой пары `(scenario, viewer)`: `filter(world, viewer, ontology)` deep-equal `expected/viewer-world/<scenario>-as-<role>-<id>.json`
- `crystallize/crystallize_test.go` — для каждой тройки `(scenario, projection, viewer)`: `crystallize(...)` deep-equal `expected/artifact/<scenario>-<projection>-as-<role>-<id>.json`

Семантическое сравнение через `internal/jsonutil.SemanticEqual` (deep-equal с игнорированием порядка ключей объектов; для массивов — порядок важен) согласно `spec/00-introduction.md` Конвенции.

`_meta` поле в expected-файлах игнорируется при сравнении.

## 6. Стек и зависимости

### Языковые требования

- **Go 1.22+** — для `//go:embed`, generics (если понадобятся), стабильные `slices`/`maps` пакеты в std lib

### External dependencies

- **`github.com/xeipuuv/gojsonschema` v1.2.x** — единственная external зависимость
  - JSON Schema draft-07 support
  - Стабильный, проверенный (используется в Kubernetes, Hashicorp tools)
  - Альтернатива: `github.com/santhosh-tekuri/jsonschema/v5` — поддерживает 2020-12, но overkill для draft-07
  - Альтернатива: написать свой mini-validator — YAGNI

### Std lib

- `encoding/json` — JSON parsing
- `testing` — table-driven tests, без testify
- `embed` — embed JSON schemas в binary
- `os`, `path/filepath`, `io/fs` — file IO
- `errors`, `fmt` — error handling
- `sort` — stable ordering для arrays
- `reflect` — для `SemanticEqual` (или recursive type-switch без reflect — выбор имплементатора)

### Ограничения

- Никаких UI / HTTP frameworks
- Никаких ORM / database drivers
- Никаких logging frameworks (`log` std lib для CLI output достаточно)
- Никаких mock-frameworks (table-driven tests вокруг чистых функций — без mocks)

## 7. Чтение fixtures из соседнего репо

Спека находится в `~/WebstormProjects/idf-spec/`, реализация — в `~/WebstormProjects/idf-go/`. Это две сестринские директории.

**Принятое решение:** относительный path `../idf-spec/spec/fixtures/library/` от корня `idf-go/` (где запускаются тесты). Hard-coded в `conformance_test.go` и в `cmd/conformance/main.go` default value.

**Альтернативы (отвергнуты):**
- Симлинк `idf-go/spec → ../idf-spec/spec/`: добавляет filesystem-state, ломается на Windows.
- Копия fixtures в idf-go: drift, нарушение DRY.
- Submodule: overkill для двух local repos.
- ENV var: усложняет local dev для нулевой выгоды.

**CLI `cmd/conformance`** принимает path как argument — `go run ./cmd/conformance <path-to-fixtures-dir>` — для возможности пройти conformance check на любом домене (не только library).

## 8. Решения по неоднозначностям спеки

Спека `v0.1` имеет 24 Open question (`spec-v0.1/feedback/manifesto-v2.md`). Реализация принимает позиции спеки и фиксирует доп. решения в `idf-go/feedback/spec-v0.1.md`. Ключевые:

- **`α: remove` на отсутствующей сущности** — no-op без ошибки (Q-15)
- **`α: replace` без существующей** — error (Q-15 симметрично)
- **Tie-breaker timestamps** — stable sort по позиции в массиве (Q-1, Q-14)
- **Self-id в visibleFields** — implementer SHOULD сохранять (Q-18) — реализация: всегда сохраняет id-поле
- **`α: commit`** — noop в fold (Lightly-tested, Reserved)
- **deriveProjections heuristic** — упрощённая (Q-9, Q-20)
- **detail recordId** — первая запись по id ASC из viewerWorld (Q-23)
- **primaryField** — первое не-id не-FK поле (Q-12, Q-24)
- **dashboard composition** — `body.sections = []` (Q-21, Reserved L4)
- **mergeProjections** — array-replace, scalar-replace, object-deep-merge

Любые decisions, не упомянутые в спеке и принятые ad-hoc реализацией, идут в `feedback/spec-v0.1.md` как «implementation choice — needs spec normalization in v0.2».

## 9. Открытые вопросы дизайна (микро-уровень)

Решения, которые лучше зафиксировать перед writing-plans:

- **D1: `effect.fields.id` обязателен на L1?** Спека (effect.schema.json + fold.md) требует `fields.id` для всех `kind`-эффектов. Реализация SHOULD выдавать explicit error при отсутствии id. **Принято: да.**
- **D2: World как `map[string]map[string]json.RawMessage` или typed structs?** Спека описывает world как `{EntityName: {entityId: entityRecord}}`. `entityRecord` — opaque (зависит от ontology). Использовать `map[string]map[string]map[string]any` (untyped JSON-like). Это упрощает rev-equal сравнение. **Принято.**
- **D3: Embed ли спека-схемы в binary через `go:embed`?** Спека находится в соседнем репо. Embed — копия, drift. Read at runtime — добавляет filesystem dependency. **Решение:** parser принимает `[]byte` schema content; `cmd/conformance` читает schemas из `../idf-spec/spec/schemas/` runtime; tests — то же. Без `embed`.
- **D4: Имя CLI binary?** `conformance` short и ясно. **Принято.**
- **D5: Output format CLI?** Plain text с pass/fail per step + summary. JSON output — YAGNI. **Принято: text only.**
- **D6: Версионирование idf-go**? `v0.1.0` для соответствия `spec-v0.1`. После — `v0.1.x` для bugfix, `v0.2.0` когда поднимет conformance до spec v0.2. **Принято.**

## 10. Критерий успеха

1. **Все table-driven тесты pass:**
   - `parser`: 4+ файла парсятся без ошибок (ontology + intents + projections + 7 phi)
   - `fold`: 7 phi-сценариев → 7 expected/world (deep-equal)
   - `filter`: 18 (scenario × viewer) → 18 expected/viewer-world (deep-equal)
   - `crystallize`: 8 (scenario × projection × viewer) → 8 expected/artifact (deep-equal)

2. **CLI печатает «L1+L2 conformant»** на library:
   ```
   $ go run ./cmd/conformance ../idf-spec/spec/fixtures/library/
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

3. **Self-audit изоляции:** prove что в течение работы не открывались запрещённые файлы. Способ: после реализации просмотреть transcript и убедиться, что Read/Grep/Glob по `idf/src`, `idf-sdk/packages/*/src`, `*.test.*`, `idf-spec/{design,plans}/` не вызывались.

4. **Backlog `feedback/spec-v0.1.md`** заполнен:
   - Места, где спека была неоднозначна
   - Implementer's choice + почему
   - Если бы были множественные равновозможные интерпретации — perhaps spec ambiguous
   - Если ошибся в первой попытке и переделал — value signal для уточнения спеки

## 11. План на следующий шаг

После одобрения этого дизайн-документа:

1. Self-review (placeholder/contradiction/scope/ambiguity)
2. Запросить ревизию у автора
3. Invoke `superpowers:writing-plans` для составления implementation plan'а:
   - Phase 1: Go module setup + types/ + parser/
   - Phase 2: fold + table-driven test
   - Phase 3: filter + table-driven test
   - Phase 4: crystallize фаза 1+2 (derive + merge)
   - Phase 5: crystallize фаза 3 (assignToSlots per archetype)
   - Phase 6: crystallize фаза 6 (wrapByConfirmation)
   - Phase 7: cmd/conformance CLI
   - Phase 8: feedback/spec-v0.1.md финализация + tag v0.1.0
