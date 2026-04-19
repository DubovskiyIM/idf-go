// Package document реализует materializeAsDocument(artifact, viewerWorld, ontology)
// → DocumentGraph согласно spec/05-materializations/document.md (v0.2.0).
package document

import (
	"fmt"
	"sort"
	"strings"

	"idf-go/types"
)

// DocumentGraph — output materialization. Top-level фиксирован; sections
// вариативны per archetype.
type DocumentGraph struct {
	Kind         string         `json:"kind"` // всегда "document"
	ProjectionID string         `json:"projectionId"`
	Archetype    string         `json:"archetype"`
	Viewer       string         `json:"viewer"`
	Title        string         `json:"title"`
	Sections     []Section      `json:"sections"`
}

type Section struct {
	Kind    string `json:"kind"` // "header" | "table" | "fields" | "actions" | "steps"
	Content any    `json:"content"`
}

// MaterializeAsDocument строит DocumentGraph из artifact + viewerWorld +
// ontology согласно spec/05-materializations/document.md.
func MaterializeAsDocument(art types.Artifact, vw types.ViewerWorld, ont types.Ontology) (DocumentGraph, error) {
	title := documentTitle(art, ont)

	doc := DocumentGraph{
		Kind:         "document",
		ProjectionID: art.ProjectionID,
		Archetype:    art.Archetype,
		Viewer:       art.Viewer,
		Title:        title,
		Sections:     []Section{},
	}

	// Section 1: header
	doc.Sections = append(doc.Sections, Section{
		Kind:    "header",
		Content: map[string]any{"title": title},
	})

	// Section 2+: per archetype
	switch art.Archetype {
	case "catalog":
		doc.Sections = append(doc.Sections, catalogTable(art, ont)...)
		doc.Sections = append(doc.Sections, catalogActions(art)...)
	case "detail":
		doc.Sections = append(doc.Sections, detailFields(art, ont))
		doc.Sections = append(doc.Sections, detailActions(art)...)
	case "form":
		doc.Sections = append(doc.Sections, formFields(art))
		doc.Sections = append(doc.Sections, formSubmit(art)...)
	case "dashboard":
		doc.Sections = append(doc.Sections, dashboardActions(art)...)
	case "feed":
		doc.Sections = append(doc.Sections, catalogTable(art, ont)...) // те же rules с body.entries
	case "wizard":
		doc.Sections = append(doc.Sections, wizardSteps(art))
	case "canvas":
		doc.Sections = append(doc.Sections, canvasField(art))
	default:
		return doc, fmt.Errorf("document: unknown archetype %s", art.Archetype)
	}

	return doc, nil
}

func documentTitle(art types.Artifact, ont types.Ontology) string {
	headerTitle := ""
	if header, ok := art.Slots["header"].(map[string]any); ok {
		if t, ok := header["title"].(string); ok {
			headerTitle = t
		}
	}
	switch art.Archetype {
	case "catalog":
		if headerTitle != "" {
			return headerTitle + " catalog"
		}
		return art.ProjectionID
	case "feed":
		if headerTitle != "" {
			return headerTitle + " feed"
		}
		return art.ProjectionID
	default:
		if headerTitle != "" {
			return headerTitle
		}
		return art.ProjectionID
	}
}

// Из artifact slots извлекаем имя entity (через header.title для catalog/feed).
// catalog/feed используют entity name как header.title.
func slotEntityName(art types.Artifact) string {
	if header, ok := art.Slots["header"].(map[string]any); ok {
		if t, ok := header["title"].(string); ok {
			return t
		}
	}
	return ""
}

func catalogTable(art types.Artifact, ont types.Ontology) []Section {
	body, ok := art.Slots["body"].(map[string]any)
	if !ok {
		return nil
	}
	itemsKey := "items"
	if art.Archetype == "feed" {
		itemsKey = "entries"
	}
	itemsRaw, _ := body[itemsKey].([]any)

	primary, secondary := primaryAndSecondaryFromBody(body, art, ont)
	cols := []string{}
	if primary != "" {
		cols = append(cols, primary)
	}
	if secondary != "" {
		cols = append(cols, secondary)
	}
	if len(cols) == 0 {
		cols = []string{"id"}
	}

	rows := [][]any{}
	for _, item := range itemsRaw {
		rec, ok := item.(map[string]any)
		if !ok {
			continue
		}
		row := []any{}
		for _, c := range cols {
			if v, ok := rec[c]; ok {
				row = append(row, v)
			} else {
				row = append(row, "")
			}
		}
		rows = append(rows, row)
	}

	return []Section{{
		Kind: "table",
		Content: map[string]any{
			"columns": stringsToAny(cols),
			"rows":    rowsToAny(rows),
		},
	}}
}

// primaryAndSecondary из body.itemDisplay (нормировано для catalog/feed v0.2.0).
func primaryAndSecondaryFromBody(body map[string]any, art types.Artifact, ont types.Ontology) (string, string) {
	if disp, ok := body["itemDisplay"].(map[string]any); ok {
		p, _ := disp["primary"].(string)
		s, _ := disp["secondary"].(string)
		return p, s
	}
	return "", ""
}

func primaryFieldName(ent types.Entity) string {
	for _, name := range ent.FieldsOrder {
		if name == "id" {
			continue
		}
		fdef := ent.Fields[name]
		if fdef.References != "" {
			continue
		}
		return name
	}
	return ""
}

func secondaryFieldName(ent types.Entity) string {
	seen := false
	for _, name := range ent.FieldsOrder {
		if name == "id" {
			continue
		}
		fdef := ent.Fields[name]
		if fdef.References != "" {
			continue
		}
		if !seen {
			seen = true
			continue
		}
		return name
	}
	return ""
}

func catalogActions(art types.Artifact) []Section {
	out := []Section{}
	footer, _ := art.Slots["footer"].(map[string]any)
	if actions, ok := footer["actions"].([]any); ok && len(actions) > 0 {
		out = append(out, Section{Kind: "actions", Content: actionsContent(actions, false)})
	}
	toolbar, _ := art.Slots["toolbar"].(map[string]any)
	if create, ok := toolbar["create"].(map[string]any); ok {
		out = append(out, Section{Kind: "actions", Content: []map[string]any{actionEntry(create, true)}})
	}
	return out
}

func detailFields(art types.Artifact, ont types.Ontology) Section {
	body, _ := art.Slots["body"].(map[string]any)
	fields, _ := body["fields"].([]any)
	entName := detailEntityFromArt(art, ont)
	out := []map[string]any{}
	for _, f := range fields {
		fm, ok := f.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{
			"name":  fm["name"],
			"value": fm["value"],
		}
		// badge для enum-полей: если type=enum в ontology, value становится badge
		if entName != "" {
			if ent, ok := ont.Entities[entName]; ok {
				if name, ok := fm["name"].(string); ok {
					if fdef, ok := ent.Fields[name]; ok {
						if fdef.Type == "enum" {
							if v, ok := fm["value"].(string); ok {
								entry["badge"] = v
							}
						}
					}
				}
			}
		}
		out = append(out, entry)
	}
	return Section{Kind: "fields", Content: anySlice(out)}
}

// derive entity name для detail: scan ontology, find entity whose fields contain
// all body.fields names. More robust than first-field-only heuristic.
func detailEntityFromArt(art types.Artifact, ont types.Ontology) string {
	body, _ := art.Slots["body"].(map[string]any)
	fields, _ := body["fields"].([]any)
	if len(fields) == 0 {
		return ""
	}
	bodyNames := []string{}
	for _, f := range fields {
		fm, ok := f.(map[string]any)
		if !ok {
			continue
		}
		if n, ok := fm["name"].(string); ok {
			bodyNames = append(bodyNames, n)
		}
	}
	if len(bodyNames) == 0 {
		return ""
	}
	for entName, ent := range ont.Entities {
		all := true
		for _, n := range bodyNames {
			if _, has := ent.Fields[n]; !has {
				all = false
				break
			}
		}
		if all {
			return entName
		}
	}
	return ""
}

func detailActions(art types.Artifact) []Section {
	footer, _ := art.Slots["footer"].(map[string]any)
	actions, _ := footer["actions"].([]any)
	if len(actions) == 0 {
		return nil
	}
	return []Section{{Kind: "actions", Content: actionsContent(actions, false)}}
}

func formFields(art types.Artifact) Section {
	body, _ := art.Slots["body"].(map[string]any)
	out := []map[string]any{}
	if fieldsArr, ok := body["fields"].([]any); ok {
		for _, f := range fieldsArr {
			switch fv := f.(type) {
			case string:
				out = append(out, map[string]any{"name": fv})
			case map[string]any:
				entry := map[string]any{}
				if n, ok := fv["name"].(string); ok {
					entry["name"] = n
				}
				if t, ok := fv["type"].(string); ok {
					entry["type"] = t
				}
				out = append(out, entry)
			}
		}
	}
	return Section{Kind: "fields", Content: anySlice(out)}
}

func formSubmit(art types.Artifact) []Section {
	footer, _ := art.Slots["footer"].(map[string]any)
	submit, ok := footer["submit"].(map[string]any)
	if !ok {
		return nil
	}
	intentID, _ := submit["intentId"].(string)
	if intentID == "" {
		return nil
	}
	confirmation, _ := submit["confirmation"].(string)
	entry := map[string]any{
		"intentId":     intentID,
		"label":        "submit: " + intentID,
		"confirmation": confirmation,
	}
	if confirmation == "destructive" {
		entry["badge"] = "destructive"
	}
	return []Section{{Kind: "actions", Content: []map[string]any{entry}}}
}

func dashboardActions(art types.Artifact) []Section {
	toolbar, _ := art.Slots["toolbar"].(map[string]any)
	actions, _ := toolbar["actions"].([]any)
	if len(actions) == 0 {
		return nil
	}
	return []Section{{Kind: "actions", Content: actionsContent(actions, false)}}
}

func wizardSteps(art types.Artifact) Section {
	body, _ := art.Slots["body"].(map[string]any)
	steps, _ := body["steps"].([]any)
	out := []map[string]any{}
	for _, s := range steps {
		sm, ok := s.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]any{
			"intentId": sm["intentId"],
			"label":    sm["label"],
			"isCommit": sm["isCommit"],
		}
		if isC, ok := sm["isCommit"].(bool); ok && isC {
			entry["badge"] = "commit"
		}
		out = append(out, entry)
	}
	return Section{Kind: "steps", Content: anySlice(out)}
}

func canvasField(art types.Artifact) Section {
	body, _ := art.Slots["body"].(map[string]any)
	ref, _ := body["canvasRef"].(string)
	return Section{Kind: "fields", Content: []map[string]any{{"name": "canvasRef", "value": ref}}}
}

// Helpers
func actionsContent(actions []any, isCreatePrefix bool) []map[string]any {
	out := []map[string]any{}
	for _, a := range actions {
		am, ok := a.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, actionEntry(am, isCreatePrefix))
	}
	return out
}

func actionEntry(am map[string]any, isCreatePrefix bool) map[string]any {
	intentID, _ := am["intentId"].(string)
	label, _ := am["label"].(string)
	if isCreatePrefix {
		label = "+ " + intentID
	}
	confirmation, _ := am["confirmation"].(string)
	entry := map[string]any{
		"intentId":     intentID,
		"label":        label,
		"confirmation": confirmation,
	}
	if confirmation == "destructive" {
		entry["badge"] = "destructive"
	}
	return entry
}

func stringsToAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func rowsToAny(rows [][]any) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i] = r
	}
	return out
}

func anySlice(maps []map[string]any) []any {
	out := make([]any, len(maps))
	for i, m := range maps {
		out[i] = m
	}
	return out
}

// Sort helper для table columns (если потребуется)
func sortStrings(s []string) {
	sort.Strings(s)
	_ = strings.TrimSpace // silence unused import
}
