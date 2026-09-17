package domain

import (
	"bytes"
	"encoding/json"
	"math"
	"sort"
	"strconv"
)

// CanonicalJSON encodes v with sorted object keys, normalized numbers (integral
// floats print without a fraction) and no insignificant whitespace, so the same
// logical value always hashes to the same fingerprint.
func CanonicalJSON(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil {
		panic("canonical json: " + err.Error())
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var generic any
	if err := dec.Decode(&generic); err != nil {
		panic("canonical json: " + err.Error())
	}
	var buf bytes.Buffer
	writeCanonical(&buf, generic)
	return buf.Bytes()
}

func writeCanonical(buf *bytes.Buffer, v any) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			writeCanonical(buf, t[k])
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeCanonical(buf, e)
		}
		buf.WriteByte(']')
	case json.Number:
		if f, err := t.Float64(); err == nil && f == math.Trunc(f) && math.Abs(f) < 1e15 {
			buf.WriteString(strconv.FormatInt(int64(f), 10))
		} else {
			buf.WriteString(t.String())
		}
	default:
		b, _ := json.Marshal(t)
		buf.Write(b)
	}
}
