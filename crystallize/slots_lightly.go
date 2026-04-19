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
