package types

// Artifact — output crystallize. Тип данных (не render).
// Спека: spec/03-objects/artifact.md.
type Artifact struct {
	Meta               map[string]any `json:"_meta,omitempty"`
	ProjectionID       string         `json:"projectionId"`
	Archetype          string         `json:"archetype"`
	Viewer             string         `json:"viewer"` // имя роли
	Slots              map[string]any `json:"slots"`
	Witnesses          []any          `json:"witnesses,omitempty"`          // Reserved L4
	Shape              string         `json:"shape,omitempty"`              // Reserved L4
	PatternAnnotations []any          `json:"patternAnnotations,omitempty"` // Reserved L4
}
