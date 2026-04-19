package jsonutil

// DeepCopy возвращает глубокую копию JSON-подобного значения. Безопасно
// мутировать результат без воздействия на оригинал.
//
// Поддерживаемые типы: map[string]any, []any, string, float64, bool, nil.
// Прочие типы возвращаются как есть (shallow).
func DeepCopy(v any) any {
	switch vv := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(vv))
		for k, val := range vv {
			out[k] = DeepCopy(val)
		}
		return out
	case []any:
		out := make([]any, len(vv))
		for i, val := range vv {
			out[i] = DeepCopy(val)
		}
		return out
	default:
		return v
	}
}

// DeepCopyMap — type-safe shortcut для DeepCopy на map'ах.
func DeepCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	return DeepCopy(m).(map[string]any)
}
