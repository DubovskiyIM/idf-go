package types

// Intent — декларативная частица: возможность изменить мир.
// Не функция и не handler — структура данных формата.
// Спека: spec/03-objects/intent.md.
type Intent struct {
	ID             string          `json:"id"`
	OwnerRole      string          `json:"ownerRole"`
	RequiredFields []RequiredField `json:"requiredFields,omitempty"`
	Conditions     []any           `json:"conditions,omitempty"` // Reserved L4 — opaque
	Effects        []ProtoEffect   `json:"effects"`
}

// RequiredField — поле, обязательное для заполнения viewer'ом перед confirm.
type RequiredField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ProtoEffect — шаблон эффекта в intent.effects[]. Не готовый эффект:
// fields опциональны и могут быть templated в v0.2+ (Q-7 спеки).
type ProtoEffect struct {
	Kind   string         `json:"kind"` // "create" | "replace" | "remove" | "transition" | "commit"
	Entity string         `json:"entity"`
	Fields map[string]any `json:"fields,omitempty"`
}
