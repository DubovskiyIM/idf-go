// Package filter реализует filterWorldForRole(world, viewer, ontology)
// → viewerWorld согласно spec/04-algebra/filter-world.md (v0.1.1).
//
// 4-приоритетный row-filter:
//  0. gate visibleFields: если entity не упомянута — namespace отсутствует.
//  1. role.base == "admin" — admin-override (видеть все записи).
//  2. entity.kind == "reference" — все записи видны.
//  3. entity.ownerField задан — записи где record[ownerField] == viewer.id.
//  4. иначе — privacy by default (пустой namespace).
//
// role.scope (priority 0 в манифесте §14) — Reserved L4, не используется
// в v0.1; manifest'овский priority 0 в спеке отдан admin-override.
//
// Spec v0.1.1 (Q-25): role.base = "admin" — spec-extension manifest §8.2
// (5-я база сверх owner|viewer|agent|observer). Resolves A-1 ambiguity
// idf-go v0.1.0 (derived эвристика «все visibleFields = '*'» заменена
// на explicit role.base check).
package filter

import (
	"idf-go/internal/jsonutil"
	"idf-go/types"
)

// FilterWorldForRole применяет 4-приоритетный row-filter + column-filter
// согласно ontology и viewer.
func FilterWorldForRole(world types.World, viewer types.Viewer, ont types.Ontology) types.ViewerWorld {
	role, ok := ont.Roles[viewer.Role]
	if !ok {
		return types.ViewerWorld{}
	}

	isAdmin := role.Base == "admin"

	out := make(types.ViewerWorld, len(role.VisibleFields))
	for entityName := range ont.Entities {
		visible, mentioned := role.VisibleFields[entityName]
		if !mentioned {
			continue // priority 0: gate
		}
		entity := ont.Entities[entityName]
		filtered := make(map[string]map[string]any)
		switch {
		case isAdmin:
			// priority 1: admin-override — видеть все записи без owner-проверки.
			for id, rec := range world[entityName] {
				filtered[id] = projectFields(rec, visible)
			}
		case entity.Kind == "reference":
			// priority 2: все записи видны
			for id, rec := range world[entityName] {
				filtered[id] = projectFields(rec, visible)
			}
		case entity.OwnerField != "":
			// priority 3: ownership filter
			for id, rec := range world[entityName] {
				if matchOwner(rec, entity.OwnerField, viewer.ID) {
					filtered[id] = projectFields(rec, visible)
				}
			}
			// priority 4 (none) — filtered остаётся пустым
		}
		out[entityName] = filtered
	}
	return out
}

func matchOwner(rec map[string]any, ownerField, viewerID string) bool {
	v, ok := rec[ownerField]
	if !ok {
		return false
	}
	s, ok := v.(string)
	return ok && s == viewerID
}

// projectFields возвращает копию rec, содержащую только поля из allowed
// (или все поля если allowed.All).
func projectFields(rec map[string]any, allowed types.VisibleFieldsValue) map[string]any {
	if allowed.All {
		return jsonutil.DeepCopyMap(rec)
	}
	out := make(map[string]any, len(allowed.Fields))
	for _, name := range allowed.Fields {
		if v, ok := rec[name]; ok {
			out[name] = jsonutil.DeepCopy(v)
		}
	}
	return out
}
