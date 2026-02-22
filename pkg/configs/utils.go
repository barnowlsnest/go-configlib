package configs

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

const (
	tagName    = "name"
	tagDefault = "default"
	tagUsage   = "usage"
	tagFlag    = "flag"
)

func checkSupportedKind(k reflect.Kind) error {
	switch k {
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Bool:
		return nil
	default:
		return fmt.Errorf("unsupported field type %s: %w", k, ErrConfig)
	}
}

func derefStructPtr(spec any, prefix []string) (val reflect.Value, p string, err error) {
	if len(prefix) > 0 {
		p = prefix[0]
	}
	val = reflect.ValueOf(spec)
	if val.Kind() != reflect.Pointer || val.Elem().Kind() != reflect.Struct {
		err = fmt.Errorf("spec must be pointer to struct, got %T: %w", spec, ErrConfig)
		return
	}
	val = val.Elem()
	return
}

func registerFields(v *Config, fs *pflag.FlagSet, typ reflect.Type, prefix string) error {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		name := field.Tag.Get(tagName)
		if name == "" || name == "-" {
			continue
		}

		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			if err := registerFields(v, fs, fieldType, joinKey(prefix, name)); err != nil {
				return err
			}
			continue
		}

		if err := checkSupportedKind(fieldType.Kind()); err != nil {
			return err
		}

		key := joinKey(prefix, name)
		usage := field.Tag.Get(tagUsage)
		defaultStr := field.Tag.Get(tagDefault)

		if defaultStr != "" {
			parsed, err := parseDefault(fieldType, defaultStr)
			if err != nil {
				return fmt.Errorf("field %s default %q: %w: %w", field.Name, defaultStr, err, ErrConfig)
			}
			v.SetDefault(key, parsed)
		}

		_ = v.BindEnv(key, strings.ToUpper(key))

		if field.Tag.Get(tagFlag) == "-" {
			continue
		}

		switch fieldType.Kind() {
		case reflect.String:
			fs.String(key, "", usage)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fs.Int(key, 0, usage)
		case reflect.Bool:
			fs.Bool(key, false, usage)
		default:
			return fmt.Errorf("unsupported field type %s: %w", fieldType.Kind(), ErrConfig)
		}
	}
	return nil
}

func loadFields(v *Config, val reflect.Value, prefix string) error {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		name := field.Tag.Get(tagName)
		if name == "" || name == "-" {
			continue
		}

		fieldVal := val.Field(i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(fieldType))
			}
			fieldVal = fieldVal.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			if err := loadFields(v, fieldVal, joinKey(prefix, name)); err != nil {
				return err
			}
			continue
		}

		if err := checkSupportedKind(fieldType.Kind()); err != nil {
			return err
		}

		key := joinKey(prefix, name)
		switch fieldType.Kind() {
		case reflect.String:
			fieldVal.SetString(v.GetString(key))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			fieldVal.SetInt(int64(v.GetInt(key)))
		case reflect.Bool:
			fieldVal.SetBool(v.GetBool(key))
		default:
			return fmt.Errorf("unsupported field type %s: %w", fieldType.Kind(), ErrConfig)
		}
	}
	return nil
}

func joinKey(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "_" + name
}

func parseDefault(typ reflect.Type, s string) (any, error) {
	switch typ.Kind() {
	case reflect.String:
		return s, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, err
		}
		return int(n), nil
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return nil, err
		}
		return b, nil
	default:
		return s, nil
	}
}
