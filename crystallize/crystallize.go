// Package crystallize реализует crystallize(intents, ontology, projection,
// viewer, viewerWorld) → artifact согласно spec/04-algebra/crystallize.md.
//
// 6 фаз pipeline:
//  1. deriveProjections — авто-вывод archetype
//  2. mergeProjections — slot-override через deep-merge
//  3. assignToSlots — per-archetype распределение intents/данных
//  4. matchPatterns — noop в L2 (Pattern Bank Reserved L4)
//  5. applyStructuralPatterns — noop в L2
//  6. wrapByConfirmation — destructive/standard на intent references
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
	intentByID := make(map[string]types.Intent, len(intents))
	for _, it := range intents {
		intentByID[it.ID] = it
	}

	archetype, err := deriveArchetype(proj, intentByID)
	if err != nil {
		return types.Artifact{}, fmt.Errorf("crystallize phase 1: %w", err)
	}

	slots, err := assignToSlots(archetype, proj, intentByID, ont, viewer, vw)
	if err != nil {
		return types.Artifact{}, fmt.Errorf("crystallize phase 3: %w", err)
	}

	if proj.Slots != nil {
		slots = mergeSlots(slots, proj.Slots)
	}

	// Фазы 4-5 — noop в L2.

	wrapByConfirmation(slots, intentByID)

	return types.Artifact{
		ProjectionID: proj.ID,
		Archetype:    archetype,
		Viewer:       viewer.Role,
		Slots:        slots,
	}, nil
}
