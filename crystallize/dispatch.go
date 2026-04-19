package crystallize

import (
	"fmt"

	"idf-go/types"
)

// assignToSlots — фаза 3 dispatcher per archetype (см. spec/04-algebra/crystallize.md).
func assignToSlots(
	archetype string,
	proj types.Projection,
	intentByID map[string]types.Intent,
	ont types.Ontology,
	viewer types.Viewer,
	vw types.ViewerWorld,
) (map[string]any, error) {
	switch archetype {
	case "catalog":
		return slotsCatalog(proj, intentByID, ont, viewer, vw), nil
	case "detail":
		return slotsDetail(proj, intentByID, ont, viewer, vw), nil
	case "form":
		return slotsForm(proj, intentByID, ont, viewer, vw), nil
	case "dashboard":
		return slotsDashboard(proj, intentByID, ont, viewer, vw), nil
	case "feed", "canvas", "wizard":
		return slotsLightly(archetype, proj, intentByID, ont, viewer, vw), nil
	default:
		return nil, fmt.Errorf("crystallize: unknown archetype %q", archetype)
	}
}

// roleCanExecute проверяет, доступен ли intent роли viewer'а.
// Поддерживает "*" (все intents).
func roleCanExecute(role types.Role, intentID string) bool {
	for _, id := range role.CanExecute {
		if id == "*" || id == intentID {
			return true
		}
	}
	return false
}

// primaryField возвращает имя «первичного» поля сущности — первое поле
// в Entity.FieldsOrder, не равное "id" и без references.
// См. spec/03-objects/artifact.md, Q-12/Q-24.
func primaryField(entity types.Entity) string {
	for _, name := range entity.FieldsOrder {
		if name == "id" {
			continue
		}
		if entity.Fields[name].References != "" {
			continue
		}
		return name
	}
	return ""
}

// secondaryField — следующее поле после primary, по тому же критерию.
func secondaryField(entity types.Entity) string {
	seen := false
	for _, name := range entity.FieldsOrder {
		if name == "id" {
			continue
		}
		if entity.Fields[name].References != "" {
			continue
		}
		if !seen {
			seen = true
			continue
		}
		return name
	}
	return ""
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func isCreateIntent(intent types.Intent) bool {
	return len(intent.Effects) > 0 && intent.Effects[0].Kind == "create"
}
