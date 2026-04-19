// Package jsonutil содержит утилиты для семантического сравнения и
// глубокого копирования JSON-подобных структур (map[string]any,
// []any, scalars).
//
// Используется для сравнения fixture-output с computed output (deep-equal
// с игнорированием порядка ключей в map'ах; для массивов — порядок
// важен, согласно spec/00-introduction.md Конвенции).
package jsonutil

// SemanticEqual сравнивает два JSON-подобных значения семантически:
//   - объекты (map[string]any): рекурсивно по ключам, порядок не важен
//   - массивы ([]any): рекурсивно по индексам, порядок важен
//   - scalars (string, float64, bool, nil): прямое сравнение
//
// Числа сравниваются как float64 (стандарт encoding/json).
//
// Поле "_meta" на любом уровне игнорируется при сравнении (informative
// в expected fixtures).
func SemanticEqual(a, b any) bool {
	return equalIgnoring(a, b, map[string]bool{"_meta": true})
}

// SemanticEqualStrict — то же что SemanticEqual, но без игнорирования _meta.
func SemanticEqualStrict(a, b any) bool {
	return equalIgnoring(a, b, nil)
}

func equalIgnoring(a, b any, ignoreKeys map[string]bool) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return false
		}
		return mapsEqual(av, bv, ignoreKeys)
	case []any:
		bv, ok := b.([]any)
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !equalIgnoring(av[i], bv[i], ignoreKeys) {
				return false
			}
		}
		return true
	case nil:
		return b == nil
	default:
		return a == b
	}
}

func mapsEqual(a, b map[string]any, ignoreKeys map[string]bool) bool {
	for k, av := range a {
		if ignoreKeys[k] {
			continue
		}
		bv, ok := b[k]
		if !ok {
			return false
		}
		if !equalIgnoring(av, bv, ignoreKeys) {
			return false
		}
	}
	for k := range b {
		if ignoreKeys[k] {
			continue
		}
		if _, ok := a[k]; !ok {
			return false
		}
	}
	return true
}
