package fingerprint

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

// Sum calculates a stable SHA-256 over JSON after rejecting values forbidden
// by sha256-mvp-v1. Struct slices must already be sorted by their stable IDs.
func Sum(value any) (string, error) {
	if err := validateValue(reflect.ValueOf(value), "$"); err != nil {
		return "", err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("canonical json: %w", err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, b); err != nil {
		return "", fmt.Errorf("compact canonical json: %w", err)
	}
	sum := sha256.Sum256(compact.Bytes())
	return hex.EncodeToString(sum[:]), nil
}

func SortedStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

func validateValue(v reflect.Value, path string) error {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		return validateValue(v.Elem(), path)
	}
	if v.Type() == reflect.TypeOf(json.Number("")) {
		if _, err := strconv.ParseInt(v.String(), 10, 64); err != nil {
			return fmt.Errorf("%s: number %q is not a base-10 integer", path, v.String())
		}
		return nil
	}
	switch v.Kind() {
	case reflect.Float32, reflect.Float64:
		return fmt.Errorf("%s: floating point values are forbidden by sha256-mvp-v1", path)
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("%s: canonical maps require string keys", path)
		}
		iter := v.MapRange()
		for iter.Next() {
			if err := validateValue(iter.Value(), path+"."+iter.Key().String()); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err := validateValue(v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case reflect.Struct:
		// time.Time has a stable JSON representation and no exported float fields.
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			if err := validateValue(v.Field(i), path+"."+field.Name); err != nil {
				return err
			}
		}
	}
	return nil
}
