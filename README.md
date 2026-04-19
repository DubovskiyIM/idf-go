# idf-go

Вторая референсная реализация формата **Intent-Driven Frontend** (IDF) на Go. Conformance: **L1 + L2** против [`spec-v0.1`](../idf-spec/spec/).

## Цель

Структурный стресс-тест формата IDF: построить независимую реализацию **исключительно по нормативной спецификации** (`~/WebstormProjects/idf-spec/`), без чтения исходников первой реализации (`~/WebstormProjects/idf/`, `~/WebstormProjects/idf-sdk/`). Если реализация проходит conformance check на эталонном домене `library` — формат **decoupled от React-specifics**.

## Связанные репозитории (на одном уровне)

- `~/WebstormProjects/idf/` — первая реализация (React/Node прототип, 9 доменов) — **не читать**
- `~/WebstormProjects/idf-sdk/` — SDK monorepo первой реализации — **не читать**
- `~/WebstormProjects/idf-spec/` — нормативная спека v0.1 — **единственный input**
- `~/WebstormProjects/idf-go/` — этот репо

Подробно об изоляции: [`CLAUDE.md`](CLAUDE.md).

## Scope

- **L1**: parser (5 core-схем) + Φ append-only + `fold(Φ, ontology) → world` + `filterWorldForRole(world, viewer, ontology) → viewerWorld`
- **L2**: `crystallize(intents, ontology, projection, viewer, viewerWorld) → artifact` (6 фаз; фазы 4-5 noop в L2)

**Вне скоупа:** L3 материализации, L4 (Pattern Bank apply, scheduler, irreversibility integrity, invariants), HTTP server, persistence, agent API.

## Стек

- **Go 1.22+**
- `github.com/xeipuuv/gojsonschema` v1.2 — JSON Schema draft-07 validator
- Std lib only для всего остального (`encoding/json`, `testing`, `os`, `path/filepath`)

## Запуск

```bash
go test ./...                                              # все table-driven тесты на library fixtures
go run ./cmd/conformance ../idf-spec/spec/fixtures/library/  # human-readable отчёт
```

Pass всех тестов = L1 + L2 conformant.

## Backlog для спеки

`feedback/spec-v0.1.md` — места, где спека была неоднозначна и потребовала угадывания (читать после реализации).

## Лицензия

Apache 2.0.
