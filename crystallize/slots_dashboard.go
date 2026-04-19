package crystallize

import (
	"sort"

	"idf-go/types"
)

// slotsDashboard заполняет slots для dashboard-архетипа (фаза 3).
//
// header.title          ← projection.id
// body.sections         ← [] (composition Reserved L4)
// toolbar.actions       ← массив {intentId, label} для intent'ов проекции,
//                         доступных viewer'у, sorted by intent.id ASC
func slotsDashboard(
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) map[string]any {
	role := ont.Roles[viewer.Role]

	intentIDs := make([]string, len(proj.Intents))
	copy(intentIDs, proj.Intents)
	sort.Strings(intentIDs)

	actions := []any{}
	for _, id := range intentIDs {
		intent, ok := intentByID[id]
		if !ok || !roleCanExecute(role, id) {
			continue
		}
		actions = append(actions, map[string]any{
			"intentId": intent.ID,
			"label":    intent.ID,
		})
	}

	return map[string]any{
		"header":  map[string]any{"title": proj.ID},
		"body":    map[string]any{"sections": []any{}},
		"toolbar": map[string]any{"actions": actions},
	}
}
