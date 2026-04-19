// Package filter реализует filterWorldForRole(world, viewer, ontology)
// → viewerWorld согласно spec/04-algebra/filter-world.md.
//
// 3-приоритетный row-filter:
//  1. Если entity не упомянута в role.visibleFields — namespace отсутствует.
//  2. Если entity.kind == "reference" — все записи видны.
//  3. Если entity.ownerField задан — записи где record[ownerField] == viewer.id.
//  4. Иначе — privacy by default (пустой namespace).
//
// role.scope (priority 0 в манифесте) — Reserved L4, не используется.
//
// IMPLEMENTER CHOICE A-1 (см. feedback/spec-v0.1.md):
// Спека описывает visibleFields как column-filter («какие поля»), но
// fixtures expected/viewer-world/ для librarian ожидают полную row-видимость.
// Принятая интерпретация: роль считается «admin» (row-filter не применяется),
// если ВСЕ её visibleFields[E] равны "*" (full column visibility for every
// mentioned entity). Иначе — обычный row-filter согласно прозе спеки.
//
// Для library: librarian admin (все 3 entity = "*"), reader не admin
// (User = [id, name]). Это derived behavior from configuration shape;
// нужна нормативная резолюция в spec v0.2 (например, через base-таксономию).
package filter

import (
	"idf-go/internal/jsonutil"
	"idf-go/types"
)

// FilterWorldForRole применяет row-filter + column-filter согласно
// ontology и viewer.
func FilterWorldForRole(world types.World, viewer types.Viewer, ont types.Ontology) types.ViewerWorld {
	role, ok := ont.Roles[viewer.Role]
	if !ok {
		return types.ViewerWorld{}
	}

	isAdmin := isAdminRole(role)

	out := make(types.ViewerWorld, len(role.VisibleFields))
	for entityName := range ont.Entities {
		visible, mentioned := role.VisibleFields[entityName]
		if !mentioned {
			continue // priority 1: gate
		}
		entity := ont.Entities[entityName]
		filtered := make(map[string]map[string]any)
		switch {
		case isAdmin:
			// IMPLEMENTER CHOICE A-1: admin — row-filter skipped.
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

// isAdminRole возвращает true, если роль имеет visibleFields[E] == "*"
// для ВСЕХ упомянутых entities. Это derived "admin" status (см. A-1).
func isAdminRole(role types.Role) bool {
	if len(role.VisibleFields) == 0 {
		return false
	}
	for _, v := range role.VisibleFields {
		if !v.All {
			return false
		}
	}
	return true
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
