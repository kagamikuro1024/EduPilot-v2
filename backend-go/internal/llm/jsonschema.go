package llm

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// ValidateJSON kiểm `data` theo `schema` (JSON Schema, tập con đủ cho đầu ra cấu trúc của LLM: type, properties, required,
// additionalProperties, items, enum, const, minimum/maximum, minLength/maxLength, minItems/maxItems, pattern, anyOf).
// Không thêm thư viện (D52); kin-openapi chỉ dùng trong test của internal/contract.
func ValidateJSON(schema, data json.RawMessage) error {
	var s map[string]any
	if err := json.Unmarshal(schema, &s); err != nil {
		return fmt.Errorf("schema không phải JSON: %w", err)
	}
	var v any
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("không phải JSON hợp lệ: %w", err)
	}
	return check(s, v, "$")
}

func check(s map[string]any, v any, path string) error {
	if c, ok := s["const"]; ok && !jsonEqual(c, v) {
		return fmt.Errorf("%s: khác giá trị cố định", path)
	}
	if e, ok := s["enum"].([]any); ok {
		if !slices.ContainsFunc(e, func(x any) bool { return jsonEqual(x, v) }) {
			return fmt.Errorf("%s: không thuộc enum", path)
		}
	}
	if any0, ok := s["anyOf"].([]any); ok {
		for _, sub := range any0 {
			if m, ok := sub.(map[string]any); ok && check(m, v, path) == nil {
				goto typed
			}
		}
		return fmt.Errorf("%s: không khớp nhánh anyOf nào", path)
	}
typed:
	typ := s["type"]
	if ts, ok := typ.([]any); ok { // ["string","null"]
		var errs []string
		for _, t := range ts {
			ss, _ := t.(string)
			cp := maps0(s)
			cp["type"] = ss
			if err := check(cp, v, path); err == nil {
				return nil
			} else {
				errs = append(errs, err.Error())
			}
		}
		return fmt.Errorf("%s: không khớp kiểu nào (%s)", path, strings.Join(errs, "; "))
	}
	t, _ := typ.(string)
	switch t {
	case "object":
		return checkObject(s, v, path)
	case "array":
		return checkArray(s, v, path)
	case "string":
		str, ok := v.(string)
		if !ok {
			return fmt.Errorf("%s: cần string", path)
		}
		n := len([]rune(str))
		if m, ok := num(s["minLength"]); ok && float64(n) < m {
			return fmt.Errorf("%s: ngắn hơn minLength", path)
		}
		if m, ok := num(s["maxLength"]); ok && float64(n) > m {
			return fmt.Errorf("%s: dài hơn maxLength", path)
		}
		if p, ok := s["pattern"].(string); ok {
			if re, err := regexp.Compile(p); err == nil && !re.MatchString(str) {
				return fmt.Errorf("%s: không khớp pattern", path)
			}
		}
	case "integer", "number":
		n, ok := v.(json.Number)
		if !ok {
			return fmt.Errorf("%s: cần %s", path, t)
		}
		f, err := n.Float64()
		if err != nil || math.IsInf(f, 0) {
			return fmt.Errorf("%s: số không hợp lệ", path)
		}
		if t == "integer" && f != math.Trunc(f) {
			return fmt.Errorf("%s: cần số nguyên", path)
		}
		if m, ok := num(s["minimum"]); ok && f < m {
			return fmt.Errorf("%s: nhỏ hơn minimum", path)
		}
		if m, ok := num(s["maximum"]); ok && f > m {
			return fmt.Errorf("%s: lớn hơn maximum", path)
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("%s: cần boolean", path)
		}
	case "null":
		if v != nil {
			return fmt.Errorf("%s: cần null", path)
		}
	case "":
		if _, hasProps := s["properties"]; hasProps {
			return checkObject(s, v, path)
		}
	}
	return nil
}

func checkObject(s map[string]any, v any, path string) error {
	obj, ok := v.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: cần object", path)
	}
	props, _ := s["properties"].(map[string]any)
	if req, ok := s["required"].([]any); ok {
		for _, r := range req {
			if k, _ := r.(string); k != "" {
				if _, present := obj[k]; !present {
					return fmt.Errorf("%s: thiếu trường bắt buộc %q", path, k)
				}
			}
		}
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sub, known := props[k].(map[string]any)
		switch ap := s["additionalProperties"].(type) {
		case bool:
			if !known && !ap {
				return fmt.Errorf("%s: trường thừa %q (additionalProperties=false)", path, k)
			}
		case map[string]any:
			if !known {
				if err := check(ap, obj[k], path+"."+k); err != nil {
					return err
				}
			}
		}
		if known {
			if err := check(sub, obj[k], path+"."+k); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkArray(s map[string]any, v any, path string) error {
	arr, ok := v.([]any)
	if !ok {
		return fmt.Errorf("%s: cần array", path)
	}
	if m, ok := num(s["minItems"]); ok && float64(len(arr)) < m {
		return fmt.Errorf("%s: ít hơn minItems", path)
	}
	if m, ok := num(s["maxItems"]); ok && float64(len(arr)) > m {
		return fmt.Errorf("%s: nhiều hơn maxItems", path)
	}
	if items, ok := s["items"].(map[string]any); ok {
		for i, e := range arr {
			if err := check(items, e, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func num(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(normNum(a))
	bb, _ := json.Marshal(normNum(b))
	return string(ab) == string(bb)
}

// normNum đưa json.Number / float64 về cùng dạng để so sánh.
func normNum(v any) any {
	switch n := v.(type) {
	case json.Number:
		f, _ := n.Float64()
		return f
	case []any:
		out := make([]any, len(n))
		for i, e := range n {
			out[i] = normNum(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(n))
		for k, e := range n {
			out[k] = normNum(e)
		}
		return out
	}
	return v
}

func maps0(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
