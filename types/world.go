package types

// World — состояние мира после fold(Φ, ontology).
// Структура: {EntityName: {EntityID: EntityRecord}}.
// EntityRecord — map[string]any (зависит от ontology.entities[E].fields).
type World map[string]map[string]map[string]any

// ViewerWorld — output filterWorldForRole. Тот же тип что World;
// alias для type-safety и читабельности.
type ViewerWorld = World

// Viewer идентифицирует роль и id viewer'а для filterWorldForRole.
type Viewer struct {
	Role string
	ID   string
}
