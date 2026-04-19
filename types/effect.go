package types

// Effect — атом изменения мира. Confirmed effects складываются в Φ.
// Спека: spec/03-objects/effect.md.
type Effect struct {
	Kind    string         `json:"kind"` // "create" | "replace" | "remove" | "transition" | "commit"
	Entity  string         `json:"entity"`
	Fields  map[string]any `json:"fields,omitempty"`
	Context EffectContext  `json:"context"`
}

// EffectContext — метаданные эффекта. v0.1 нормирует только обязательное
// поле At (ISO 8601 timestamp). Остальные поля opaque.
type EffectContext struct {
	At        string         `json:"at"` // ISO 8601 date-time, обязательно
	Initiator string         `json:"initiator,omitempty"`
	IntentID  string         `json:"intentId,omitempty"`
	IRR       map[string]any `json:"__irr,omitempty"` // Reserved L4
}
