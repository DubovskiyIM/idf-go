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
//  1. Инициализировать world с пустыми namespace для каждой entity.
//  2. Stable-sort phi по context.at ASC (tie-breaker — исходный индекс).
//  3. Применить каждый effect согласно kind.
func Fold(phi []types.Effect, ont types.Ontology) (types.World, error) {
	world := make(types.World, len(ont.Entities))
	for name := range ont.Entities {
		world[name] = make(map[string]map[string]any)
	}

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

	id, hasID := getID(eff.Fields)
	if !hasID && eff.Kind != "commit" {
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
