package crystallize

import (
	"idf-go/types"
)

// slotsForm заполняет slots для form-архетипа (фаза 3).
//
// header.title       ← intent.id если intents.length === 1 иначе projection.id
// body.fields        ← массив {name, type, required: true} из intent.requiredFields
// footer.submit      ← {intentId, label} для main intent
func slotsForm(
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) map[string]any {
	var mainIntent types.Intent
	if len(proj.Intents) > 0 {
		mainIntent = intentByID[proj.Intents[0]]
	}

	title := proj.ID
	if len(proj.Intents) == 1 && mainIntent.ID != "" {
		title = mainIntent.ID
	}

	bodyFields := []any{}
	for _, rf := range mainIntent.RequiredFields {
		bodyFields = append(bodyFields, map[string]any{
			"name":     rf.Name,
			"type":     rf.Type,
			"required": true,
		})
	}

	submit := map[string]any{}
	if mainIntent.ID != "" {
		submit = map[string]any{
			"intentId": mainIntent.ID,
			"label":    mainIntent.ID,
		}
	}

	return map[string]any{
		"header": map[string]any{"title": title},
		"body":   map[string]any{"fields": bodyFields},
		"footer": map[string]any{"submit": submit},
	}
}
