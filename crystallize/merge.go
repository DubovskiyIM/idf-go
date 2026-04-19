package crystallize

import (
	"idf-go/internal/jsonutil"
)

// mergeSlots реализует фазу 2 mergeProjections (см. spec/04-algebra/crystallize.md).
//
// Семантика merge:
//   - Объекты (map[string]any): рекурсивный deep-merge с приоритетом authored.
//   - Массивы ([]any): замена целиком (стандартная JSON merge семантика).
//   - Скалярные значения: замена на authored.
//   - Поле _authored: true сохраняется в результате (forward-compat marker).
func mergeSlots(derived, authored map[string]any) map[string]any {
	out := jsonutil.DeepCopyMap(derived)
	if out == nil {
		out = map[string]any{}
	}
	for k, v := range authored {
		if v == nil {
			out[k] = nil
			continue
		}
		dv, ok := out[k]
		if !ok {
			out[k] = jsonutil.DeepCopy(v)
			continue
		}
		dMap, dOk := dv.(map[string]any)
		aMap, aOk := v.(map[string]any)
		if dOk && aOk {
			out[k] = mergeSlots(dMap, aMap)
		} else {
			out[k] = jsonutil.DeepCopy(v)
		}
	}
	return out
}
