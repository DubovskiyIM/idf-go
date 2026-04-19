# CLAUDE.md — idf-go

## Цель проекта

Написать вторую референсную реализацию формата IDF на Go — conformant с **L1 + L2** по `spec-v0.1` (`~/WebstormProjects/idf-spec/spec/`). Цель эксперимента — стресс-тест формата на decoupling от React-specifics первой реализации.

## Язык

Все файлы, документация, коммит-сообщения, комментарии в коде — **на русском**. Имена типов, функций, переменных в Go-коде — английские (`Ontology`, `Fold`, `viewerWorld`), как принято в Go.

## Git-коммиты

Не добавлять Claude (или другого бота) в соавторы. Коммиты — от имени автора, без `Co-Authored-By` / `🤖 Generated with` трейлеров.

## Принцип честного эксперимента

Эта реализация развивается **в изоляции** от первой. Если код пишется, заглядывая в `idf/src/` или `idf-sdk/packages/*/src/`, — эксперимент теряет смысл: получается порт, а не независимая реализация.

### Allowed reads (white-list)

При работе над `idf-go` можно читать:

- `~/WebstormProjects/idf-spec/spec/**` — нормативная спека (МАИН SOURCE)
- `~/WebstormProjects/idf-spec/feedback/manifesto-v2.md` — backlog для манифеста (informative)
- `~/WebstormProjects/idf-spec/source/manifesto-v2.snapshot.md` — frozen snapshot манифеста
- `~/WebstormProjects/idf-spec/README.md`, `~/WebstormProjects/idf-spec/CLAUDE.md` — общие правила
- Содержимое `idf-go/` (свой репо)
- Документация Go и зависимостей (`pkg.go.dev`, `gojsonschema` README)

### Forbidden reads (black-list)

Категорически **нельзя** читать:

- `~/WebstormProjects/idf/src/**` — исходники React-прототипа
- `~/WebstormProjects/idf/server/**` — серверный код первой реализации
- `~/WebstormProjects/idf/scripts/**`
- `~/WebstormProjects/idf-sdk/packages/*/src/**` — исходники SDK
- `~/WebstormProjects/idf/**/*.test.{js,jsx,ts,tsx,cjs}` — тесты прототипа
- `~/WebstormProjects/idf-sdk/**/*.test.{js,jsx,ts,tsx}` — тесты SDK
- `~/WebstormProjects/idf/docs/implementation-status.md` — снимок реализации, не формата
- `~/WebstormProjects/idf/docs/archive/**` — архивные манифесты v1.x
- `~/WebstormProjects/idf-spec/design/**` — authorial process спеки (informative для авторов спеки, не для implementer'а)
- `~/WebstormProjects/idf-spec/plans/**` — то же
- Любые `package.json`, `vite.config.js`, `vitest.config.js` соседних репо

### Правило сомнения

Если не ясно, относится ли файл к white или black-list — **не читать**. Зафиксировать вопрос в `feedback/spec-v0.1.md` и спросить автора.

### Что делать, если знание уже есть в контексте

Если из предыдущих сессий или CLAUDE.md проекта `idf` помнятся имплементационные детали (имена файлов, версии пакетов, конкретные функции, числа тестов) — **активно их не использовать**. Спека описывает *формат*; реализация воспроизводит спеку, не копирует первую реализацию.

Жёсткое разделение:
- Спека говорит **X** (нормативно) → реализация делает X
- Память подсказывает **Y** (как первая реализация поступила) → игнорировать

## Стек

- **Go 1.22+** (any modern Go)
- `github.com/xeipuuv/gojsonschema` v1.2 — JSON Schema draft-07 validator
- Std lib only для прочего (`encoding/json`, `testing`, `os`, `path/filepath`)

## Стиль кода

- Один package per директория (стандарт Go)
- Файлы < 300 LOC; разрастание → разбить
- Table-driven тесты на std `testing` (без testify, без spec-frameworks)
- Errors через std `errors` + `fmt.Errorf` с `%w` wrapping
- Comments в коде — на русском, лаконично; doc-comments для exported identifiers — обязательно (Go convention)
- Никаких внешних HTTP / persistence / logging фреймворков

## Скоуп v1.0

**В скоупе:**
- L1: parsing 5 core-схем + Φ + fold + filterWorldForRole
- L2: crystallize 6 фаз (4-5 noop) + 7 архетипов (4 покрыты fixture-вектором)
- CLI `cmd/conformance` для human-readable отчёта
- Table-driven тесты, читающие fixtures из `../idf-spec/spec/fixtures/library/`

**Вне скоупа:**
- L3 материализации (pixel/voice/agent/document)
- L4 (Pattern Bank apply, scheduler, irreversibility integrity, invariants kinds)
- HTTP server / agent API
- Persistence (SQLite, files)
- Production-grade error wrapping, logging, observability — минимум, sufficient for tests

## Связь со спекой

`spec-v0.1` — sole normative input. Где спека неоднозначна (`Open question` секции, `Reserved L4` поля) — реализация либо:
1. Принимает позицию спеки (если зафиксирована) — например, `α: remove` на отсутствующей сущности — no-op (Q-15 в fold.md).
2. Делает minimal reasonable choice и фиксирует в `feedback/spec-v0.1.md` — будущий backlog для v0.2.

Реализация **не вносит новых норм**, не упомянутых в спеке.
