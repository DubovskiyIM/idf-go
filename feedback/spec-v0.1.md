# Backlog ambiguities `spec-v0.1` (выявлены при написании idf-go)

Этот файл — **не часть idf-go**. Это feedback автору спеки: места, где `spec-v0.1` потребовала угадывания или принятия implementer's choice из-за неоднозначности или отсутствия нормативного правила.

Источник: реализация `idf-go` (Go 1.26.2 + xeipuuv/gojsonschema), L1+L2 conformant против библиотечного fixture-домена. Все ambiguities обнаружены без чтения исходников первой реализации.

---

## Категория A: Implementer choices, требующие normative resolution в v0.2

### A-1: ✅ RESOLVED в spec-v0.1.1 (idf-go v0.1.1)

**Резолюция:** spec v0.1.1 нормировала admin-pattern через `role.base = "admin"` (spec-extension manifest §8.2 — пятая база сверх `owner|viewer|agent|observer`). Filter теперь использует explicit `role.Base == "admin"` вместо derived эвристики «все visibleFields == "*"». library/ontology.json получил `librarian.base = "admin"`. Manifest v2.1 должен sync таксономию (см. `idf-spec/feedback/manifesto-v2.md` Q-25).

История проблемы (для архива):

**Спека v0.1 (filter-world.md):** «`visibleFields` определяет column-filter (какие поля)» — то есть row-filter определяется отдельно (priority reference > ownerField > none).

**Спека (ontology.md, Q-3):** «entity MUST быть упомянута в `role.visibleFields` — иначе невидима даже при `kind: "reference"`. visibleFields контролирует cross-section (какие entity вообще видны); kind/ownerField — row-filter».

**Fixtures (expected/viewer-world/`*-as-librarian-*`):** для librarian с `visibleFields = {User: "*", Book: "*", Loan: "*"}` все три entity показывают полный набор записей (включая User'ов, у которых `id != viewer.id`).

**Конфликт:** проза спеки и fixtures расходятся. Прямое следование прозе даёт пустой `User` namespace для librarian (потому что `User.ownerField = "id"`, `viewer.id = "u-lib-1"`, ни один User не имеет такого id). Fixtures требуют, чтобы librarian видел всех `u-r1`, `u-r2`.

**Implementer choice:** реализована derived-эвристика «admin-роль = ВСЕ её `visibleFields[E]` равны `"*"`». Для library: librarian admin (3 entity = `"*"`), reader не admin (`User: ["id", "name"]`). Если admin — row-filter не применяется.

**Это самая существенная находка backlog'а.** Эвристика хрупкая: зависит от формы конфигурации, не от семантики. Если автор reader'а добавит четвёртое entity с `visibleFields[E] = "*"` (но оставит `User: ["id", "name"]`) — reader всё ещё не admin; но если изменит User на `"*"` — внезапно станет admin. Это unprincipled.

**Предложение для v0.2:** ввести normative admin-marker. Возможные пути:
1. `role.base = "agent"` (Reserved L4) с явной семантикой «admin row-override для всех entity, помеченных в visibleFields как "*"»
2. Per-entity флаг `role.adminFor: ["User", "Loan"]` явно
3. Расширить row-filter prose: «если visibleFields[E] = "*" — row-filter не применяется для E» (но это меняет fixtures для reader — Loan все увидит, что неправильно)

Predпочтительно (1): связать с base-таксономией — она ближе к семантической модели формата.

### A-2: Сохранение порядка ключей в `entities[E].fields` для primary/secondary heuristic

**Спека (Q-12, Q-24):** primary field — «первое поле в `ontology.entities[entity].fields` (по key order JSON-объекта)».

**Проблема:** Go `map[string]Field` теряет порядок ключей при стандартном `json.Unmarshal`. Implementer вынужден читать JSON через `json.Decoder.Token()` чтобы извлечь порядок имён.

**Implementer choice:** добавлено поле `Entity.FieldsOrder []string`, заполняемое отдельным проходом парсера (`extractFieldsOrder` в `parser/parser.go`).

**Предложение для v0.2:** либо явно зафиксировать в спеке, что implementer MUST сохранять JSON object key order для `entities.fields` (предложить альтернативу — explicit `fieldsOrder: []string` в JSON Schema), либо изменить heuristic на explicit declaration `entity.primary: "fieldName"`, `entity.secondary: "fieldName"`. Второй вариант более robust.

### A-3: Парсинг `_meta` в core-объектах (ontology, artifact)

**Проблема:** core JSON Schemas (effect, intent, projection) изначально не упоминают `_meta` поле. Wrapper-схемы (phi, intents-collection, projections-collection) — упоминают. Ontology и artifact схемы — также нужны (fixtures имеют `_meta`).

**Спека:** упомянула `_meta` как top-level в ontology.schema.json и artifact.schema.json. Это нормально (был исправлено в Task 9 design idf-spec). OK на v0.1.

**Предложение:** просто документировать как стандартную конвенцию: «все fixture-файлы могут иметь top-level `_meta` для metadata, парсер MUST принимать без эффекта».

### A-4: Порядок сортировки `slots.body.items` в catalog

**Спека (crystallize.md фаза 3):** «упорядочен по `id` ASC».

**Implementer choice:** sort.Strings (lexicographic) на keys map'а ID'ов. Работает для library (b1, b2, b3). Для произвольных доменов с числовыми id (например, "1", "10", "2") даст лекс-порядок (1, 10, 2), не numeric (1, 2, 10).

**Предложение для v0.2:** уточнить «lexicographic ordering» явно или нормировать numeric-aware ordering.

### A-5: detail recordId — первая запись по id ASC

**Спека (crystallize.md Q-23):** «для упрощения fixture'ов — первая запись по id ASC из viewerWorld[entity]».

**Implementer choice:** реализовано буквально. В реальности detail-проекции должны принимать конкретный recordId как параметр crystallize. Спека сама фиксирует это как Q-23 (Reserved v0.2+).

**OK для v0.1.**

### A-6: `slots.body.fields` для form архетипа после mergeProjections

**Спека:** describes phase 2 как «deep-merge с приоритетом authored»; для library borrow-form: derived `body.fields = [{name: "bookId", ...}]`, authored `body.fields = ["bookId"]` (массив имён). После merge — массив replace, итог `body.fields = ["bookId"]`.

**Implementer choice:** реализовано array-replace семантика согласно спеке. Это нормировано после спеки v0.1 commit'а.

**OK для v0.1.**

### A-7: Имя `α` поля в proto-effects

**Спека:** в effect.schema.json — `"kind"` (нормировано). В intent.schema.json `intent.effects[]` — также `"kind"` (тот же enum). Согласовано.

**OK для v0.1.**

### A-8: parseVWName / parseArtName в CLI: split по role-name

**Implementer choice:** CLI парсит имена fixture-файлов через heuristic «scan role names из ontology, найти prefix». Если role-name содержит `-as-` или `-` в неудачном месте — split может быть неоднозначен. Для library (reader, librarian) не проблема, но fragile.

**Предложение для v0.2:** добавить opt-in metadata в expected fixture-файлы: `_meta.scenario`, `_meta.projection`, `_meta.viewer.role`, `_meta.viewer.id`. Это сделает name-parsing избыточным.

### A-9: VisibleFieldsValue UnmarshalJSON

**Implementer choice:** type `VisibleFieldsValue` с custom UnmarshalJSON для разбора либо строки `"*"` либо массива строк. Не упомянуто в спеке; вытекает из необходимости parse'ить разные shapes JSON.

**OK для v0.1** — это implementation detail Go-стека. Other languages (Python, JS) могут не нуждаться в специальных типах.

---

## Категория B: Реальные баги спеки

Не обнаружено. fixtures валидируются, prose согласована (с одним исключением — A-1, который скорее ambiguity, чем bug).

---

## Категория C: Documentation gaps

### C-1: Отсутствие нормативного списка fields для каждой entity в expected/world

**Наблюдение:** spec/04-algebra/fold.md описывает алгоритм, но не нормирует, что `world[E][id]` MUST содержать ВСЕ fields из effect.fields (даже если ontology.entities[E].fields их не описывает). Например, если effect создаёт Book с `extraField: "x"`, попадёт ли `extraField` в world?

**Implementer choice:** да, попадёт. Реализация просто `DeepCopyMap(eff.Fields)` — не фильтрует по ontology.

**Не критично для v0.1** — fixtures не тестируют этот edge case.

### C-2: Отсутствие нормативного описания, что делать с `effect.context.IRR` на L1

**Спека:** `__irr` принимается как opaque (нормировано). 

**OK для v0.1.**

---

## Метрики честности эксперимента

- **Время реализации:** ~1 час (от `go mod init` до `L1+L2 CONFORMANT`)
- **Iterations при failing tests:** 1 (filter test failed на 4 librarian-сценариях; A-1 ambiguity → re-implementation; всё другое passed first time)
- **Open questions, потребовавшие угадывания:** 1 серьёзная (A-1), 8 минорных или OK-as-is
- **Места, где соблазн открыть исходники первой реализации был сильным:** A-1 ambiguity (хотел проверить, как первая реализация обрабатывает librarian). Подавлен — реализована derived эвристика на основе формы конфигурации.
- **Опасные знания из памяти, которые были подавлены:**
  - Помнил из CLAUDE.md проекта `idf`, что есть base-таксономия (`owner|viewer|agent|observer`) и что librarian, возможно, имеет специальный base. Не использовал — спека не нормирует base в L1+L2.
  - Помнил, что в первой реализации есть pattern bank, R8 hub-absorption, scheduler — все Reserved L4, не использовал.
  - Помнил конкретные имена файлов и пакетов первой реализации — не упоминал.

---

## Итог

**Формат IDF в скоупе spec-v0.1 (L1+L2) decoupled от React-specifics реализации.** Go-реализация прошла ВСЕ fixture-векторы:
- 7/7 phi-сценариев → world (fold)
- 18/18 (scenario × viewer) → viewerWorld (filter)
- 8/8 (scenario × projection × viewer) → artifact (crystallize)

**Спека v0.1 в основном достаточна** для independent implementer'а. Ключевая ambiguity (A-1) требует normative resolution в v0.2 — без неё derived-эвристика admin-роли работает для library, но хрупкая для произвольных доменов.

Эксперимент — **success.** Получаемая Go-реализация ~1500 LOC + ~500 LOC тестов; единственная external зависимость (gojsonschema). Без чтения первой реализации.
