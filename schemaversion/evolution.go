package schemaversion

import (
	"encoding/json"
	"fmt"
)

// EvolutionEntry — одна запись в ontology.evolution[] (см. spec §2).
type EvolutionEntry struct {
	Hash       string          `json:"hash"`
	ParentHash *string         `json:"parentHash"` // nil для root
	Timestamp  string          `json:"timestamp"`
	AuthorID   string          `json:"authorId"`
	Diff       json.RawMessage `json:"diff,omitempty"`
	Upcasters  []Upcaster      `json:"upcasters,omitempty"`
}

// Upcaster — шаг трансформации эффекта (см. spec §3).
type Upcaster struct {
	FromHash    string          `json:"fromHash"`
	ToHash      string          `json:"toHash"`
	Declarative json.RawMessage `json:"declarative,omitempty"`
}

// ParseEvolution извлекает ontology.evolution[] из raw ontology'и (как
// any) и десериализует в []EvolutionEntry. Возвращает (nil, nil) если
// поле отсутствует.
func ParseEvolution(ontology any) ([]EvolutionEntry, error) {
	m, ok := ontology.(map[string]any)
	if !ok {
		return nil, nil
	}
	raw, ok := m["evolution"]
	if !ok {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal evolution: %w", err)
	}
	var entries []EvolutionEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal evolution: %w", err)
	}
	return entries, nil
}

// ValidateEvolutionLog проверяет invariants спеки §2:
//
//  1. Hash уникален в пределах лога
//  2. Root entry (parentHash == nil) ровно один (или ноль для пустого log'а)
//  3. Все non-root entries имеют parentHash указывающий на существующий entry'й
//  4. Цепочка от каждой версии до root достижима без циклов
//  5. Pattern hash и parentHash валидируется (14-char lowercase hex)
//
// Re-hash check (entry.hash SHOULD совпадать с hashOntology(...)) —
// implementer'ом MAY, не делается здесь поскольку требует ontology
// snapshot at-that-time который log не хранит.
//
// Возвращает все накопленные ошибки (multi-error через errors.Join не
// используется чтобы остаться compatible с Go 1.20-).
func ValidateEvolutionLog(entries []EvolutionEntry) []error {
	var errs []error
	if len(entries) == 0 {
		return nil
	}

	// 1. Hash uniqueness + format
	seen := make(map[string]int, len(entries))
	for i, e := range entries {
		if !isValidHash(e.Hash) {
			errs = append(errs, fmt.Errorf("entry[%d]: hash %q does not match pattern ^[0-9a-f]{14}$", i, e.Hash))
		}
		if e.ParentHash != nil && !isValidHash(*e.ParentHash) {
			errs = append(errs, fmt.Errorf("entry[%d]: parentHash %q does not match pattern ^[0-9a-f]{14}$", i, *e.ParentHash))
		}
		if prev, exists := seen[e.Hash]; exists {
			errs = append(errs, fmt.Errorf("entry[%d]: hash %q duplicates entry[%d]", i, e.Hash, prev))
		} else {
			seen[e.Hash] = i
		}
	}

	// 2. Exactly one root
	rootCount := 0
	for _, e := range entries {
		if e.ParentHash == nil {
			rootCount++
		}
	}
	if rootCount == 0 {
		errs = append(errs, fmt.Errorf("evolution log: no root entry (parentHash=null)"))
	}
	if rootCount > 1 {
		errs = append(errs, fmt.Errorf("evolution log: %d root entries (only one allowed)", rootCount))
	}

	// 3. parentHash existence
	for i, e := range entries {
		if e.ParentHash == nil {
			continue
		}
		if _, ok := seen[*e.ParentHash]; !ok {
			errs = append(errs, fmt.Errorf("entry[%d]: parentHash %q references non-existent entry", i, *e.ParentHash))
		}
	}

	// 4. Cycle detection via DFS — для каждого entry следуем parentHash
	//    до root или до повторного посещения текущей цепочки.
	for i, e := range entries {
		if e.ParentHash == nil {
			continue
		}
		visited := map[string]bool{e.Hash: true}
		curr := *e.ParentHash
		hops := 0
		for {
			if hops > len(entries) {
				errs = append(errs, fmt.Errorf("entry[%d] (hash=%q): cycle detected in parentHash chain", i, e.Hash))
				break
			}
			if visited[curr] {
				errs = append(errs, fmt.Errorf("entry[%d] (hash=%q): cycle detected at %q", i, e.Hash, curr))
				break
			}
			visited[curr] = true
			idx, ok := seen[curr]
			if !ok {
				// Уже обработано в (3); skip
				break
			}
			parent := entries[idx]
			if parent.ParentHash == nil {
				break // root reached
			}
			curr = *parent.ParentHash
			hops++
		}
	}

	return errs
}

// RehashAndVerify — optional implementer-side validation для §2 SHOULD-clause:
// "entry.hash SHOULD совпадать с hashOntology(ontology_at_that_time)".
//
// Для root entry: ontology_at_that_time = ontology_сейчас, потому что
// hashOntology исключает evolution[] поле (см. spec hash-function.md
// §2 step 0). Поэтому root entry rehash == hashOntology(currentOntology).
//
// Для non-root entries требуется snapshot онтологии at-that-time, который
// log не хранит — implementer должен реконструировать через obratный
// applying upcasters; вне scope этого helper'а.
//
// Возвращает true если root.hash совпадает с hashOntology(currentOntology),
// false если есть mismatch (drift) или log пустой.
func RehashAndVerifyRoot(entries []EvolutionEntry, currentOntology any) (matches bool, expected, got string) {
	for _, e := range entries {
		if e.ParentHash == nil {
			expected = HashOntology(currentOntology)
			return e.Hash == expected, expected, e.Hash
		}
	}
	return false, "", ""
}

func isValidHash(s string) bool {
	if len(s) != 14 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
