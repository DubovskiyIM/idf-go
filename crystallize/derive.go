package crystallize

import (
	"fmt"

	"idf-go/types"
)

// deriveArchetype реализует фазу 1 (deriveProjections) согласно
// spec/04-algebra/crystallize.md.
//
// Если archetype != "auto" и не пустой — использовать как есть.
// Иначе минимальная heuristic спеки v0.1:
//   - все intents имеют effects[0].kind == "create" и одну entity → "form"
//   - projection.entity задан → "detail"
//   - fallback → "catalog"
func deriveArchetype(proj types.Projection, intentByID map[string]types.Intent) (string, error) {
	if proj.Archetype != "" && proj.Archetype != "auto" {
		return proj.Archetype, nil
	}

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

	if proj.Entity != "" {
		return "detail", nil
	}

	return "catalog", nil
}
