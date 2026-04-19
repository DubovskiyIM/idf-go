package crystallize

import "idf-go/types"

// Заглушки до реализации (Task 10-12).
// Будут удалены в финальном коммите Phase 8.

func assignToSlots(
	archetype string,
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) (map[string]any, error) {
	return map[string]any{}, nil
}

func wrapByConfirmation(slots map[string]any, intentByID map[string]types.Intent) {}
