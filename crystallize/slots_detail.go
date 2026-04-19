package crystallize

import (
	"sort"

	"idf-go/internal/jsonutil"
	"idf-go/types"
)

// slotsDetail заполняет slots для detail-архетипа (фаза 3).
//
// Запись для detail: первая по id ASC из viewerWorld[entity] (Q-23).
//
// header.title       ← record[primaryField]
// body.fields        ← массив {name, value} по entity.FieldsOrder
// footer.actions     ← все доступные intents, sorted by intent.id ASC
func slotsDetail(
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) map[string]any {
	role := ont.Roles[viewer.Role]
	entity := ont.Entities[proj.Entity]

	ns := vw[proj.Entity]
	ids := make([]string, 0, len(ns))
	for id := range ns {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var record map[string]any
	if len(ids) > 0 {
		record = ns[ids[0]]
	}

	primary := primaryField(entity)
	header := map[string]any{}
	if record != nil && primary != "" {
		if v, ok := record[primary]; ok {
			header["title"] = v
		}
	}

	bodyFields := []any{}
	if record != nil {
		for _, name := range entity.FieldsOrder {
			if v, ok := record[name]; ok {
				bodyFields = append(bodyFields, map[string]any{
					"name":  name,
					"value": jsonutil.DeepCopy(v),
				})
			}
		}
	}

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
		"header": header,
		"body":   map[string]any{"fields": bodyFields},
		"footer": map[string]any{"actions": actions},
	}
}
