package crystallize

import (
	"sort"

	"idf-go/internal/jsonutil"
	"idf-go/types"
)

// slotsCatalog заполняет slots для catalog-архетипа (фаза 3).
//
// header.title          ← projection.entity (или fallback projection.id)
// body.items            ← Object.values(viewerWorld[entity]) sorted by id ASC
// body.itemDisplay      ← {primary, secondary} heuristic (если entity задан)
// footer.actions        ← non-create intents проекции, accessible viewer'у, sorted by intent.id ASC
// toolbar.create        ← create-intent (если есть в проекции и accessible) или toolbar = {}
func slotsCatalog(
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) map[string]any {
	role := ont.Roles[viewer.Role]

	header := map[string]any{
		"title": coalesce(proj.Entity, proj.ID),
	}

	body := map[string]any{
		"items": catalogItems(proj.Entity, vw),
	}
	if proj.Entity != "" {
		entity := ont.Entities[proj.Entity]
		body["itemDisplay"] = map[string]any{
			"primary":   primaryField(entity),
			"secondary": secondaryField(entity),
		}
	}

	footerActions := []any{}
	toolbar := map[string]any{}

	intentIDs := make([]string, len(proj.Intents))
	copy(intentIDs, proj.Intents)
	sort.Strings(intentIDs)

	for _, id := range intentIDs {
		intent, ok := intentByID[id]
		if !ok {
			continue
		}
		if !roleCanExecute(role, id) {
			continue
		}
		if isCreateIntent(intent) {
			toolbar["create"] = map[string]any{
				"intentId": intent.ID,
				"label":    intent.ID,
			}
		} else {
			footerActions = append(footerActions, map[string]any{
				"intentId": intent.ID,
				"label":    intent.ID,
			})
		}
	}

	return map[string]any{
		"header":  header,
		"body":    body,
		"footer":  map[string]any{"actions": footerActions},
		"toolbar": toolbar,
	}
}

// catalogItems возвращает массив записей entity из viewerWorld,
// упорядоченных по id ASC.
func catalogItems(entityName string, vw types.ViewerWorld) []any {
	if entityName == "" {
		return []any{}
	}
	ns := vw[entityName]
	if ns == nil {
		return []any{}
	}
	ids := make([]string, 0, len(ns))
	for id := range ns {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, jsonutil.DeepCopyMap(ns[id]))
	}
	return items
}
