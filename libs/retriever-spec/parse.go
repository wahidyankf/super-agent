package spec

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"

	"go.yaml.in/yaml/v3"
)

// tagRule is the rule of a node whose YAML tag the spec does not allow.
const tagRule = "uses a YAML tag that the spec does not allow"

// parseDocument reads the bytes of one spec as one YAML document of at most MaxBytes. It refuses an alias, an anchor, a
// repeated field name, a tag other than the plain forms of text, number, boolean, null, map, and list, a number that is
// not finite, a field name that is not text, and a root that is not a mapping. It returns the value as JSON-shaped Go
// values: maps, lists, strings, booleans, nil, and json.Number for each number. The findings name paths, never values.
func parseDocument(data []byte) (any, []Finding) {
	if len(data) > MaxBytes {
		return nil, []Finding{finding("", "is larger than 256 KiB")}
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, []Finding{finding("", "is empty")}
		}
		return nil, []Finding{finding("", "is not readable YAML")}
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, []Finding{finding("", "holds more than one YAML document")}
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, []Finding{finding("", "is not a mapping of fields")}
	}
	w := &walker{}
	value := w.node(doc.Content[0], "")
	return value, w.findings
}

// walker reads a YAML tree into Go values and records a finding for each node the spec does not allow.
type walker struct {
	findings []Finding
}

// refuse records a finding at a path.
func (w *walker) refuse(path, rule string) {
	w.findings = append(w.findings, finding(path, rule))
}

// node reads one node at a path. An alias is not followed, so a value is never copied from another place.
func (w *walker) node(n *yaml.Node, path string) any {
	if n.Anchor != "" {
		w.refuse(path, "uses an anchor, which the spec does not allow")
	}
	if n.Kind == yaml.AliasNode {
		w.refuse(path, "uses an alias, which the spec does not allow")
		return nil
	}
	switch n.Kind {
	case yaml.MappingNode:
		return w.mapping(n, path)
	case yaml.SequenceNode:
		return w.sequence(n, path)
	case yaml.ScalarNode:
		return w.scalar(n, path)
	}
	w.refuse(path, "is not a plain value")
	return nil
}

// mapping reads a mapping. A field name must be plain text, and it may appear only once.
func (w *walker) mapping(n *yaml.Node, path string) any {
	if n.Tag != "!!map" {
		w.refuse(path, tagRule)
	}
	m := make(map[string]any, len(n.Content)/2)
	seen := make(map[string]bool, len(n.Content)/2)
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, val := n.Content[i], n.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Anchor != "" {
			w.refuse(path, "has a field name that is not plain text")
			continue
		}
		child := path + "/" + safeName(key.Value)
		if seen[key.Value] {
			w.refuse(child, "repeats a field name")
		}
		seen[key.Value] = true
		m[key.Value] = w.node(val, child)
	}
	return m
}

// sequence reads a list, and each item takes its index in the path.
func (w *walker) sequence(n *yaml.Node, path string) any {
	if n.Tag != "!!seq" {
		w.refuse(path, tagRule)
	}
	list := make([]any, 0, len(n.Content))
	for i, item := range n.Content {
		list = append(list, w.node(item, path+"/"+strconv.Itoa(i)))
	}
	return list
}

// scalar reads a plain scalar. A number must parse and be finite, and it is kept as the json.Number of its value.
func (w *walker) scalar(n *yaml.Node, path string) any {
	switch n.Tag {
	case "!!str":
		return n.Value
	case "!!null":
		return nil
	case "!!bool":
		b, err := strconv.ParseBool(n.Value)
		if err != nil {
			w.refuse(path, "is not a plain boolean")
			return nil
		}
		return b
	case "!!int":
		i, err := strconv.ParseInt(n.Value, 0, 64)
		if err != nil {
			w.refuse(path, "is not a plain integer")
			return nil
		}
		return json.Number(strconv.FormatInt(i, 10))
	case "!!float":
		f, err := strconv.ParseFloat(n.Value, 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			w.refuse(path, "is not a finite number")
			return nil
		}
		return json.Number(strconv.FormatFloat(f, 'g', -1, 64))
	}
	w.refuse(path, tagRule)
	return nil
}
