// Package schemaversion реализует L3-evolution §1: cyrb53 + hashOntology +
// helpers для effect.context.schemaVersion.
//
// Нормативная спецификация: idf-spec/spec/schemas/hash-function.md +
// hash-function.vectors.json. Реализация byte-в-byte совместима с
// reference JS impl (@intent-driven/core@0.107.0) и должна оставаться
// стабильной для одной major-версии формата.
package schemaversion

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"unicode/utf16"
)

// UnknownSchemaVersion — sentinel для legacy эффектов без поля.
const UnknownSchemaVersion = "unknown"

// GetSchemaVersion извлекает effect.context.schemaVersion. Возвращает
// UnknownSchemaVersion если эффект legacy / поле отсутствует. Никогда не
// паникует.
func GetSchemaVersion(effect map[string]any) string {
	if effect == nil {
		return UnknownSchemaVersion
	}
	ctx, ok := effect["context"].(map[string]any)
	if !ok {
		return UnknownSchemaVersion
	}
	v, ok := ctx["schemaVersion"].(string)
	if !ok || v == "" {
		return UnknownSchemaVersion
	}
	return v
}

// TagWithSchemaVersion возвращает копию эффекта с проставленным
// schemaVersion в context. Pure — не мутирует input. Если version пустой —
// возвращает копию без модификации (поведение совместимо с JS).
func TagWithSchemaVersion(effect map[string]any, version string) map[string]any {
	if effect == nil {
		return effect
	}
	out := make(map[string]any, len(effect))
	for k, v := range effect {
		out[k] = v
	}
	if version == "" {
		return out
	}
	var ctx map[string]any
	if existing, ok := out["context"].(map[string]any); ok {
		ctx = make(map[string]any, len(existing)+1)
		for k, v := range existing {
			ctx[k] = v
		}
	} else {
		ctx = make(map[string]any, 1)
	}
	ctx["schemaVersion"] = version
	out["context"] = ctx
	return out
}

// HashOntology — стабильный fingerprint онтологии для
// effect.context.schemaVersion. Возвращает 14-символьную lowercase hex
// строку. Алгоритм: canonicalize → JSON.stringify → cyrb53 → hex pad14.
//
// Для ontology == nil возвращает "00000000000000" (sentinel).
func HashOntology(ontology any) string {
	if ontology == nil {
		return "00000000000000"
	}
	canonical := canonicalize(ontology)
	serialized, err := jsonStringify(canonical)
	if err != nil {
		// JSON-несовместимый input — defense только; in-spec это panic
		// на стороне автора. Возвращаем sentinel чтобы не падать.
		return "00000000000000"
	}
	h := Cyrb53(serialized, 0)
	return fmt.Sprintf("%014s", hexU64(h))
}

// Cyrb53 — нормативная 53-bit pure hash function (см.
// idf-spec/spec/schemas/hash-function.md §1).
//
// Принимает строку и обрабатывает её как UTF-16 code unit sequence (так
// же как `String.charCodeAt(i)` в JS). Возвращает u64 с занятыми младшими
// 53 битами; старшие 11 бит = 0.
func Cyrb53(s string, seed uint32) uint64 {
	h1 := uint32(0xdeadbeef) ^ seed
	h2 := uint32(0x41c6ce57) ^ seed

	// String → UTF-16 code units — соответствует JS charCodeAt.
	codeUnits := utf16.Encode([]rune(s))
	for _, ch := range codeUnits {
		h1 = imul32(h1^uint32(ch), 0x9e3779b1) // 2654435761
		h2 = imul32(h2^uint32(ch), 0x5f356495) // 1597334677
	}

	// JS reference обновляет h1 до использования в h2-формуле; повторяем тот
	// же порядок (sequential dependency, не parallel).
	h1 = imul32(h1^(h1>>16), 0x85ebca6b) ^ imul32(h2^(h2>>13), 0xc2b2ae35)
	h2 = imul32(h2^(h2>>16), 0x85ebca6b) ^ imul32(h1^(h1>>13), 0xc2b2ae35)

	return uint64(h2&0x1fffff)<<32 | uint64(h1)
}

// imul32 — 32-битное умножение с обёрткой (modulo 2^32). Эквивалент
// Math.imul в JS.
func imul32(a, b uint32) uint32 {
	return a * b // uint32 в Go обёртывается на overflow
}

// hexU64 — нижние 53 бита u64 в hex без leading zeros.
func hexU64(v uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	s := hex.EncodeToString(buf[:])
	// Убрать leading zero hex digit'ы до 14 символов; pad обеспечит format
	for len(s) > 14 && s[0] == '0' {
		s = s[1:]
	}
	return s
}

// canonicalize — рекурсивно сортирует object keys лексикографически,
// сохраняет порядок массивов. Принимает результат json.Unmarshal с
// обычными типами (map[string]any, []any, primitive).
func canonicalize(value any) any {
	switch v := value.(type) {
	case nil:
		return nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make(canonicalMap, 0, len(keys))
		for _, k := range keys {
			out = append(out, canonicalEntry{Key: k, Value: canonicalize(v[k])})
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, e := range v {
			out[i] = canonicalize(e)
		}
		return out
	default:
		return v
	}
}

// canonicalEntry / canonicalMap — упорядоченное представление map'а для
// jsonStringify, чтобы JSON-сериализация шла в строго отсортированном
// порядке (Go map итерируется случайно).
type canonicalEntry struct {
	Key   string
	Value any
}
type canonicalMap []canonicalEntry

// jsonStringify — RFC 8259 stringification без отступов, с уже
// отсортированными ключами. Совпадает с дефолтом JS JSON.stringify(value).
func jsonStringify(value any) (string, error) {
	switch v := value.(type) {
	case canonicalMap:
		var buf []byte
		buf = append(buf, '{')
		for i, e := range v {
			if i > 0 {
				buf = append(buf, ',')
			}
			kb, err := json.Marshal(e.Key)
			if err != nil {
				return "", err
			}
			buf = append(buf, kb...)
			buf = append(buf, ':')
			vb, err := jsonStringify(e.Value)
			if err != nil {
				return "", err
			}
			buf = append(buf, vb...)
		}
		buf = append(buf, '}')
		return string(buf), nil
	case []any:
		var buf []byte
		buf = append(buf, '[')
		for i, e := range v {
			if i > 0 {
				buf = append(buf, ',')
			}
			vb, err := jsonStringify(e)
			if err != nil {
				return "", err
			}
			buf = append(buf, vb...)
		}
		buf = append(buf, ']')
		return string(buf), nil
	default:
		// Стандартный json.Marshal даёт RFC 8259 без пробелов для
		// primitive'ов / null / уже-canonicalized.
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
