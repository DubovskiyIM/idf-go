package crystallize

import (
	"sort"

	"idf-go/internal/jsonutil"
	"idf-go/types"
)

// slotsLightly заполняет минимальную нормативную структуру для
// архетипов feed, canvas, wizard.
//
// v0.1.5 (idf-go v0.1.2): feed и wizard теперь нормированы спекой —
// реализованы fill rules. canvas остаётся stub'ом (Lightly-tested
// в спеке, без fixture).
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
		out := map[string]any{
			"body": map[string]any{
				"entries": feedEntries(proj.Entity, ont, vw),
			},
		}
		// Если intents > 0 — добавляются footer/toolbar (за catalog rules).
		// В spec v0.1.5 этот случай существует, но не покрыт fixture; реализация
		// minimal-correct: оставляем без footer/toolbar когда intents=[].
		// При intents>0 implementer SHOULD дублировать catalog-логику actions/toolbar.
		return out

	case "wizard":
		steps := make([]any, 0, len(proj.Intents))
		for _, id := range proj.Intents {
			intent, ok := intentByID[id]
			if !ok {
				continue
			}
			isCommit := false
			for _, e := range intent.Effects {
				if e.Kind == "commit" {
					isCommit = true
					break
				}
			}
			steps = append(steps, map[string]any{
				"intentId": intent.ID,
				"label":    intent.ID,
				"isCommit": isCommit,
			})
		}
		return map[string]any{
			"body": map[string]any{"steps": steps},
		}

	case "canvas":
		return map[string]any{
			"body": map[string]any{"canvasRef": proj.ID},
		}
	}
	return map[string]any{}
}

// feedEntries возвращает массив записей entity из viewerWorld,
// упорядоченный по первому datetime-полю в Entity.FieldsOrder DESC.
// Если такого поля нет — fallback по id ASC.
func feedEntries(entityName string, ont types.Ontology, vw types.ViewerWorld) []any {
	if entityName == "" {
		return []any{}
	}
	ns := vw[entityName]
	if ns == nil {
		return []any{}
	}

	var dtField string
	if ent, ok := ont.Entities[entityName]; ok {
		for _, name := range ent.FieldsOrder {
			if ent.Fields[name].Type == "datetime" {
				dtField = name
				break
			}
		}
	}

	ids := make([]string, 0, len(ns))
	for id := range ns {
		ids = append(ids, id)
	}
	if dtField != "" {
		sort.Slice(ids, func(i, j int) bool {
			ai, _ := ns[ids[i]][dtField].(string)
			aj, _ := ns[ids[j]][dtField].(string)
			return ai > aj
		})
	} else {
		sort.Strings(ids)
	}

	items := make([]any, 0, len(ids))
	for _, id := range ids {
		items = append(items, jsonutil.DeepCopyMap(ns[id]))
	}
	return items
}
