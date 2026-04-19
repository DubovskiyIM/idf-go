// Package types определяет Go-структуры для пяти core-объектов формата
// IDF (ontology, intent, effect, projection, artifact) согласно spec-v0.1.
//
// Структуры используют encoding/json теги для round-trip парсинга и
// сериализации. Поля, зарезервированные L4 (invariants, rules,
// witnesses, shape), приняты как opaque и сохраняются для
// forward-compatibility, но не участвуют в L1+L2 логике.
package types

// Ontology — тип данных домена: сущности, поля, роли.
// Спека: spec/03-objects/ontology.md.
type Ontology struct {
	Meta       map[string]any    `json:"_meta,omitempty"`
	Entities   map[string]Entity `json:"entities"`
	Roles      map[string]Role   `json:"roles"`
	Invariants []any             `json:"invariants,omitempty"` // Reserved L4
	Rules      []any             `json:"rules,omitempty"`      // Reserved L4
}

// Entity — описание типа сущности.
type Entity struct {
	Kind        string           `json:"kind,omitempty"` // "internal" (default) | "reference" | "mirror" | "assignment"
	OwnerField  string           `json:"ownerField,omitempty"`
	Fields      map[string]Field `json:"fields"`
	FieldsOrder []string         `json:"-"` // заполняется парсером (см. parser.extractFieldsOrder)
}

// Field — описание поля сущности.
type Field struct {
	Type       string   `json:"type"`                 // "string" | "number" | "boolean" | "datetime" | "enum"
	Values     []string `json:"values,omitempty"`     // для type="enum"
	FieldRole  string   `json:"fieldRole,omitempty"`  // opaque hint
	References string   `json:"references,omitempty"` // FK на entity
	Required   bool     `json:"required,omitempty"`
}

// Role — viewer-тип.
type Role struct {
	VisibleFields map[string]VisibleFieldsValue `json:"visibleFields"`
	CanExecute    []string                      `json:"canExecute"`
	Scope         map[string]any                `json:"scope,omitempty"` // Reserved L4
	Base          string                        `json:"base,omitempty"`  // Reserved L4
}

// VisibleFieldsValue представляет значение role.visibleFields[Entity]:
// либо строка "*" (все поля), либо массив имён полей.
// См. spec/03-objects/ontology.md.
type VisibleFieldsValue struct {
	All    bool     // true если "*"
	Fields []string // конкретные имена полей
}

// UnmarshalJSON разбирает значение visibleFields[Entity]: либо строка "*",
// либо массив строк.
func (v *VisibleFieldsValue) UnmarshalJSON(data []byte) error {
	var s string
	if err := jsonUnmarshal(data, &s); err == nil {
		if s == "*" {
			v.All = true
			return nil
		}
		return errInvalidVisibleFieldsString(s)
	}
	var arr []string
	if err := jsonUnmarshal(data, &arr); err != nil {
		return err
	}
	v.Fields = arr
	return nil
}

// MarshalJSON сериализует обратно в строку "*" или массив имён.
func (v VisibleFieldsValue) MarshalJSON() ([]byte, error) {
	if v.All {
		return []byte(`"*"`), nil
	}
	return jsonMarshal(v.Fields)
}
