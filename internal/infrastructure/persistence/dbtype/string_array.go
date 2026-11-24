package dbtype

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// StringArray maps PostgreSQL text[].
type StringArray []string

func (StringArray) GormDataType() string {
	return "text[]"
}

func (StringArray) GormDBDataType(dialect string) string {
	return "text[]"
}

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	if len(a) == 0 {
		return "{}", nil
	}
	parts := make([]string, len(a))
	for i, s := range a {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		parts[i] = `"` + escaped + `"`
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

func (a *StringArray) Scan(value any) error {
	if value == nil {
		*a = nil
		return nil
	}
	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return fmt.Errorf("unsupported StringArray scan type %T", value)
	}
	raw = strings.TrimSpace(raw)
	if raw == "{}" || raw == "" {
		*a = StringArray{}
		return nil
	}
	if len(raw) < 2 || raw[0] != '{' || raw[len(raw)-1] != '}' {
		return fmt.Errorf("invalid text[] literal: %q", raw)
	}
	inner := raw[1 : len(raw)-1]
	*a = parsePGTextArray(inner)
	return nil
}

func parsePGTextArray(inner string) StringArray {
	var out StringArray
	var cur strings.Builder
	inQuotes := false
	escape := false
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if escape {
			cur.WriteByte(c)
			escape = false
			continue
		}
		if c == '\\' && inQuotes {
			escape = true
			continue
		}
		if c == '"' {
			inQuotes = !inQuotes
			continue
		}
		if c == ',' && !inQuotes {
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	out = append(out, cur.String())
	return out
}
