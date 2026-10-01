package mcpcatalog

import "encoding/json"

// schema deliberately models only a decidable, small subset.
type schema struct {
	typ        string
	enum       []string
	props      map[string]*schema
	required   map[string]bool
	additional bool
}

func parseSchema(v any) (*schema, bool) {
	if v == nil {
		return nil, false
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(data, &m) != nil || m == nil {
		return nil, false
	}
	return parseObject(m, true)
}
func parseObject(m map[string]json.RawMessage, root bool) (*schema, bool) {
	s := &schema{additional: true, props: map[string]*schema{}, required: map[string]bool{}}
	if json.Unmarshal(m["type"], &s.typ) != nil {
		return nil, false
	}
	if root && s.typ != "object" {
		return nil, false
	}
	if !root && s.typ != "string" && s.typ != "boolean" && s.typ != "integer" && s.typ != "number" && s.typ != "null" {
		return nil, false
	}
	for key, v := range m {
		switch key {
		case "type":
		case "description", "title":
			var text *string
			if json.Unmarshal(v, &text) != nil || text == nil {
				return nil, false
			}
		case "enum":
			var valid bool
			s.enum, valid = stringArray(v)
			if s.typ != "string" || !valid || len(s.enum) == 0 {
				return nil, false
			}
		case "additionalProperties":
			if !root || json.Unmarshal(v, &s.additional) != nil || string(v) == "null" {
				return nil, false
			}
		case "required":
			names, valid := stringArray(v)
			if !root || !valid {
				return nil, false
			}
			for _, name := range names {
				if s.required[name] {
					return nil, false
				}
				s.required[name] = true
			}
		case "properties":
			var props map[string]map[string]json.RawMessage
			if !root || json.Unmarshal(v, &props) != nil || props == nil {
				return nil, false
			}
			for name, m := range props {
				p, ok := parseObject(m, false)
				if !ok {
					return nil, false
				}
				s.props[name] = p
			}
		default:
			return nil, false
		}
	}
	for name := range s.required {
		if s.props[name] == nil {
			return nil, false
		}
	}
	return s, true
}
func subset(a, b *schema) bool {
	if a.typ != b.typ && !(a.typ == "integer" && b.typ == "number") {
		return false
	}
	if a.typ != "object" {
		if len(b.enum) == 0 {
			return true
		}
		if len(a.enum) == 0 {
			return false
		}
		values := map[string]bool{}
		for _, v := range b.enum {
			values[v] = true
		}
		for _, v := range a.enum {
			if !values[v] {
				return false
			}
		}
		return true
	}
	if a.additional && !b.additional {
		return false
	}
	for name := range b.required {
		if !a.required[name] {
			return false
		}
	}
	for name, p := range a.props {
		q := b.props[name]
		if q == nil {
			if !b.additional {
				return false
			}
		} else if !subset(p, q) {
			return false
		}
	}
	// Unconstrained source properties could violate the target's typed property.
	for name := range b.props {
		if a.props[name] == nil && a.additional {
			return false
		}
	}
	return true
}

func stringArray(v json.RawMessage) ([]string, bool) {
	var raw []json.RawMessage
	if json.Unmarshal(v, &raw) != nil || raw == nil {
		return nil, false
	}
	values := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, entry := range raw {
		var value *string
		if json.Unmarshal(entry, &value) != nil || value == nil || seen[*value] {
			return nil, false
		}
		seen[*value] = true
		values = append(values, *value)
	}
	return values, true
}
