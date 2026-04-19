package crystallize

import (
	"idf-go/types"
)

// wrapByConfirmation реализует фазу 6 (см. spec/04-algebra/crystallize.md).
//
// Для каждого intent reference в slots-tree (объекты с полем intentId)
// добавляет поле confirmation:
//   - intent.effects[0].kind == "remove" → "destructive"
//   - иначе → "standard"
func wrapByConfirmation(slots map[string]any, intentByID map[string]types.Intent) {
	walkIntentRefs(slots, func(ref map[string]any) {
		id, _ := ref["intentId"].(string)
		intent, ok := intentByID[id]
		if !ok {
			return
		}
		ref["confirmation"] = confirmationLevel(intent)
	})
}

func confirmationLevel(intent types.Intent) string {
	if len(intent.Effects) > 0 && intent.Effects[0].Kind == "remove" {
		return "destructive"
	}
	return "standard"
}

// walkIntentRefs вызывает fn на каждом объекте, содержащем intentId,
// в slots-tree.
func walkIntentRefs(node any, fn func(map[string]any)) {
	switch v := node.(type) {
	case map[string]any:
		if _, isRef := v["intentId"]; isRef {
			fn(v)
			return
		}
		for _, child := range v {
			walkIntentRefs(child, fn)
		}
	case []any:
		for _, item := range v {
			walkIntentRefs(item, fn)
		}
	}
}
