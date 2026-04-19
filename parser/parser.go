// Package parser валидирует и декодирует JSON-файлы спеки IDF против
// JSON Schema draft-07 + Go struct-типов.
//
// Каждая Parse* функция принимает []byte и SchemaSet с путями к
// схема-файлам; возвращает typed-struct + error.
package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/xeipuuv/gojsonschema"

	"idf-go/types"
)

// SchemaSet — набор путей к JSON Schema-файлам, нужных парсеру.
// По умолчанию указывает на <specRoot>/schemas/.
type SchemaSet struct {
	Ontology              string
	Intent                string
	IntentsCollection     string
	Projection            string
	ProjectionsCollection string
	Effect                string
	Phi                   string
	Artifact              string
}

// DefaultSchemaSet возвращает пути относительно <specRoot>/schemas/.
func DefaultSchemaSet(specRoot string) SchemaSet {
	return SchemaSet{
		Ontology:              specRoot + "/schemas/ontology.schema.json",
		Intent:                specRoot + "/schemas/intent.schema.json",
		IntentsCollection:     specRoot + "/schemas/intents-collection.schema.json",
		Projection:            specRoot + "/schemas/projection.schema.json",
		ProjectionsCollection: specRoot + "/schemas/projections-collection.schema.json",
		Effect:                specRoot + "/schemas/effect.schema.json",
		Phi:                   specRoot + "/schemas/phi.schema.json",
		Artifact:              specRoot + "/schemas/artifact.schema.json",
	}
}

// validateAgainstSchema выполняет JSON Schema validation. Возвращает
// nil если data соответствует schema, иначе structured error.
func validateAgainstSchema(data []byte, schemaPath string, refSchemas []string) error {
	loader := gojsonschema.NewSchemaLoader()
	loader.Draft = gojsonschema.Draft7
	loader.Validate = false

	for _, ref := range refSchemas {
		if err := loader.AddSchemas(gojsonschema.NewReferenceLoader("file://" + ref)); err != nil {
			return fmt.Errorf("parser: load ref schema %s: %w", ref, err)
		}
	}
	schema, err := loader.Compile(gojsonschema.NewReferenceLoader("file://" + schemaPath))
	if err != nil {
		return fmt.Errorf("parser: compile schema %s: %w", schemaPath, err)
	}
	result, err := schema.Validate(gojsonschema.NewBytesLoader(data))
	if err != nil {
		return fmt.Errorf("parser: validate: %w", err)
	}
	if !result.Valid() {
		return fmt.Errorf("parser: schema validation failed for %s: %v", schemaPath, result.Errors())
	}
	return nil
}

// ParseOntology валидирует ontology.json против ontology.schema.json
// и декодирует в types.Ontology. Дополнительно заполняет
// Entity.FieldsOrder через отдельный проход (см. extractFieldsOrder).
func ParseOntology(data []byte, schemas SchemaSet) (types.Ontology, error) {
	if err := validateAgainstSchema(data, schemas.Ontology, nil); err != nil {
		return types.Ontology{}, err
	}
	var ont types.Ontology
	if err := json.Unmarshal(data, &ont); err != nil {
		return types.Ontology{}, fmt.Errorf("parser: ontology decode: %w", err)
	}
	if err := extractFieldsOrder(data, &ont); err != nil {
		return types.Ontology{}, fmt.Errorf("parser: extract fields order: %w", err)
	}
	return ont, nil
}

// extractFieldsOrder читает порядок ключей entities[E].fields через
// json.Decoder.Token() и заполняет Entity.FieldsOrder. Это нужно для
// primary/secondary heuristic в crystallize (Q-12, Q-24 спеки).
func extractFieldsOrder(data []byte, ont *types.Ontology) error {
	var raw struct {
		Entities map[string]json.RawMessage `json:"entities"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for entityName, entRaw := range raw.Entities {
		var entObj struct {
			Fields json.RawMessage `json:"fields"`
		}
		if err := json.Unmarshal(entRaw, &entObj); err != nil {
			continue
		}
		order, err := jsonObjectKeysInOrder(entObj.Fields)
		if err != nil {
			continue
		}
		entity := ont.Entities[entityName]
		entity.FieldsOrder = order
		ont.Entities[entityName] = entity
	}
	return nil
}

// jsonObjectKeysInOrder возвращает имена ключей JSON-объекта в порядке
// их появления.
func jsonObjectKeysInOrder(data []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != '{' {
		return nil, fmt.Errorf("expected object, got %v", tok)
	}
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key, got %v", tok)
		}
		keys = append(keys, key)
		if err := skipValue(dec); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// skipValue читает и отбрасывает следующее значение в decoder'е.
func skipValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil // scalar
	}
	switch delim {
	case '{', '[':
		for dec.More() {
			if delim == '{' {
				// читаем key
				if _, err := dec.Token(); err != nil {
					return err
				}
			}
			if err := skipValue(dec); err != nil {
				return err
			}
		}
		// читаем закрывающую delim
		if _, err := dec.Token(); err != nil {
			return err
		}
	}
	return nil
}

// IntentsCollection — wrapper для fixture-файла intents.json.
type IntentsCollection struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Intents []types.Intent `json:"intents"`
}

// ParseIntents валидирует intents.json через intents-collection wrapper
// и декодирует.
func ParseIntents(data []byte, schemas SchemaSet) ([]types.Intent, error) {
	if err := validateAgainstSchema(data, schemas.IntentsCollection, []string{schemas.Intent}); err != nil {
		return nil, err
	}
	var coll IntentsCollection
	if err := json.Unmarshal(data, &coll); err != nil {
		return nil, fmt.Errorf("parser: intents decode: %w", err)
	}
	return coll.Intents, nil
}

// ProjectionsCollection — wrapper для fixture-файла projections.json.
type ProjectionsCollection struct {
	Meta        map[string]any     `json:"_meta,omitempty"`
	Projections []types.Projection `json:"projections"`
}

// ParseProjections валидирует projections.json через wrapper и декодирует.
func ParseProjections(data []byte, schemas SchemaSet) ([]types.Projection, error) {
	if err := validateAgainstSchema(data, schemas.ProjectionsCollection, []string{schemas.Projection}); err != nil {
		return nil, err
	}
	var coll ProjectionsCollection
	if err := json.Unmarshal(data, &coll); err != nil {
		return nil, fmt.Errorf("parser: projections decode: %w", err)
	}
	return coll.Projections, nil
}

// PhiFile — wrapper для fixture-файла phi/<scenario>.json.
type PhiFile struct {
	Meta    map[string]any `json:"_meta,omitempty"`
	Effects []types.Effect `json:"effects"`
}

// ParsePhi валидирует phi/<scenario>.json через phi.schema.json и
// декодирует. Φ возвращается в исходном порядке (без сортировки —
// сортировка по at — задача fold).
func ParsePhi(data []byte, schemas SchemaSet) ([]types.Effect, error) {
	if err := validateAgainstSchema(data, schemas.Phi, []string{schemas.Effect}); err != nil {
		return nil, err
	}
	var phi PhiFile
	if err := json.Unmarshal(data, &phi); err != nil {
		return nil, fmt.Errorf("parser: phi decode: %w", err)
	}
	return phi.Effects, nil
}

// ReadFile — небольшая утилита для тестов и CLI.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
