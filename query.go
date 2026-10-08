package okx

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var timeType = reflect.TypeOf(Time{})

// encodeQuery turns a request struct into query parameters using its json
// tags. Zero values are omitted, slices are comma-joined.
func encodeQuery(params any) (url.Values, error) {
	q := url.Values{}
	switch p := params.(type) {
	case nil:
		return q, nil
	case url.Values:
		return p, nil
	case map[string]string:
		for k, v := range p {
			if v != "" {
				q.Set(k, v)
			}
		}
		return q, nil
	}
	v := reflect.ValueOf(params)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return q, nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("okx: query params must be a struct, got %T", params)
	}
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" || !f.IsExported() {
			continue
		}
		if s := queryValue(v.Field(i)); s != "" {
			q.Set(name, s)
		}
	}
	return q, nil
}

func queryValue(v reflect.Value) string {
	if v.Type() == timeType {
		t := v.Interface().(Time)
		if t.IsZero() {
			return ""
		}
		return strconv.FormatInt(t.UnixMilli(), 10)
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.Int() == 0 {
			return ""
		}
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Bool:
		if !v.Bool() {
			return ""
		}
		return "true"
	case reflect.Slice:
		parts := make([]string, 0, v.Len())
		for i := range v.Len() {
			if s := queryValue(v.Index(i)); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ",")
	case reflect.Struct:
		if t, ok := v.Interface().(time.Time); ok && !t.IsZero() {
			return strconv.FormatInt(t.UnixMilli(), 10)
		}
	}
	return ""
}
