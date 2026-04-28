package commands

import (
	"fmt"
	"strings"

	"github.com/lukew-cogapp/colflow-cli/internal/format"
	"github.com/parquet-go/parquet-go"
)

// SchemaEntry is a flat row produced by walking the schema tree, used for JSON output.
type SchemaEntry struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Optional bool   `json:"optional"`
	Depth    int    `json:"depth"`
}

// FlattenSchema walks the schema, returning rows in display order. Parquet list/map
// wrapper groups (list.element, key_value.key/value) are collapsed into the parent.
func FlattenSchema(schema *parquet.Schema) []SchemaEntry {
	var out []SchemaEntry
	for _, f := range schema.Fields() {
		walkNode(&out, f, f.Name(), 0, false)
	}
	return out
}

func walkNode(out *[]SchemaEntry, n parquet.Node, name string, depth int, parentOptional bool) {
	optional := parentOptional || n.Optional()

	if n.Leaf() {
		*out = append(*out, SchemaEntry{
			Name:     name,
			Type:     nodeTypeName(n),
			Optional: optional,
			Depth:    depth,
		})
		return
	}

	// Detect Parquet LIST encoding: group with single child "list" (or "array"),
	// whose single child is "element" (or "item").
	if listElem, ok := unwrapList(n); ok {
		if listElem.Leaf() {
			*out = append(*out, SchemaEntry{
				Name:     name,
				Type:     "list[" + nodeTypeName(listElem) + "]",
				Optional: optional,
				Depth:    depth,
			})
			return
		}
		*out = append(*out, SchemaEntry{
			Name:     name,
			Type:     "list[group]",
			Optional: optional,
			Depth:    depth,
		})
		for _, f := range listElem.Fields() {
			walkNode(out, f, f.Name(), depth+1, false)
		}
		return
	}

	// Detect Parquet MAP encoding.
	if k, v, ok := unwrapMap(n); ok {
		*out = append(*out, SchemaEntry{
			Name:     name,
			Type:     "map[" + nodeTypeName(k) + " -> " + nodeTypeName(v) + "]",
			Optional: optional,
			Depth:    depth,
		})
		return
	}

	*out = append(*out, SchemaEntry{
		Name:     name,
		Type:     "group",
		Optional: optional,
		Depth:    depth,
	})
	for _, f := range n.Fields() {
		walkNode(out, f, f.Name(), depth+1, false)
	}
}

func unwrapList(n parquet.Node) (parquet.Node, bool) {
	fields := n.Fields()
	if len(fields) != 1 {
		return nil, false
	}
	inner := fields[0]
	innerName := strings.ToLower(inner.Name())
	if innerName != "list" && innerName != "array" {
		return nil, false
	}
	innerFields := inner.Fields()
	if len(innerFields) != 1 {
		return nil, false
	}
	elemName := strings.ToLower(innerFields[0].Name())
	if elemName != "element" && elemName != "item" {
		return nil, false
	}
	return innerFields[0], true
}

func unwrapMap(n parquet.Node) (key, value parquet.Node, ok bool) {
	fields := n.Fields()
	if len(fields) != 1 {
		return nil, nil, false
	}
	kv := fields[0]
	if strings.ToLower(kv.Name()) != "key_value" && strings.ToLower(kv.Name()) != "map" {
		return nil, nil, false
	}
	kvFields := kv.Fields()
	if len(kvFields) != 2 {
		return nil, nil, false
	}
	var k, v parquet.Node
	for _, f := range kvFields {
		switch strings.ToLower(f.Name()) {
		case "key":
			k = f
		case "value":
			v = f
		}
	}
	if k == nil || v == nil {
		return nil, nil, false
	}
	return k, v, true
}

// PrintSchemaTree emits a tree-formatted schema. typeWidth pads the type column.
func PrintSchemaTree(entries []SchemaEntry) {
	maxName := 0
	for _, e := range entries {
		w := e.Depth*2 + len(e.Name)
		if w > maxName {
			maxName = w
		}
	}
	for _, e := range entries {
		indent := strings.Repeat("  ", e.Depth)
		nameCol := indent + e.Name
		nullable := format.Yellow("required")
		if e.Optional {
			nullable = format.Gray("optional")
		}
		typeStr := format.Gray(e.Type)
		if strings.HasPrefix(e.Type, "list[") || strings.HasPrefix(e.Type, "map[") || e.Type == "group" {
			typeStr = format.Cyan(e.Type)
		}
		fmt.Printf("  %s  %s  %s\n",
			format.PadRight(nameCol, maxName),
			format.PadRight(typeStr, 32),
			nullable,
		)
	}
}
