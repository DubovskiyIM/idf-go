package types

import "encoding/json"

// Тонкие обёртки над encoding/json. Позволяют тестировать UnmarshalJSON
// без import cycle (если когда-нибудь понадобится подменить).
var (
	jsonUnmarshal = json.Unmarshal
	jsonMarshal   = json.Marshal
)

// TypeError — структурированная ошибка типизированного парсинга.
type TypeError struct {
	Field string
	Got   string
	Want  string
}

func (e *TypeError) Error() string {
	return "types: поле " + e.Field + " — got " + e.Got + ", want " + e.Want
}

func errInvalidVisibleFieldsString(s string) error {
	return &TypeError{Field: "visibleFields", Got: s, Want: `"*" or []string`}
}
