package types

// Projection — авторский контракт на view. Input для crystallize.
// Спека: spec/03-objects/projection.md.
type Projection struct {
	ID        string         `json:"id"`
	Archetype string         `json:"archetype,omitempty"` // "auto" | "feed" | "catalog" | "detail" | "form" | "canvas" | "dashboard" | "wizard"
	Intents   []string       `json:"intents"`
	Entity    string         `json:"entity,omitempty"`
	Slots     map[string]any `json:"slots,omitempty"`    // slot-override (фаза 2 mergeProjections)
	Patterns  map[string]any `json:"patterns,omitempty"` // Reserved L4
}
