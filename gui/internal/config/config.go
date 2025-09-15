package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"encoding/json"
	"reflect"

	"github.com/go-playground/validator/v10"
	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v3"
)

type TGAPIConfig struct {
	APIID   string `mapstructure:"API_ID"`
	APIHash string `mapstructure:"API_HASH"`
}

type AppConfig struct {
	VenvPath       string `mapstructure:"venv_path" validate:"required"`
	ScriptsPath    string `mapstructure:"scripts_path" validate:"required"`
	Session        string `mapstructure:"session_name" validate:"required,filepath"`
	AuthConfigName string `mapstructure:"auth_config_name" validate:"required"`
	CreatorLogPath string `mapstructure:"creator_log_path" validate:"required,filepath"`
	AppLogPath     string `mapstructure:"app_log_path" validate:"required,filepath"`
	CreatorURI     string `mapstructure:"creator_uri" validate:"required,uri"`
}

func LoadConfig[T any](name, ext string, paths ...string) (*T, error) {

	var zero *T

	v := viper.New()
	for _, p := range paths {
		v.AddConfigPath(p)
	}
	v.SetConfigName(name)
	// if ext is empty, LoadConfig will try to read 'name' file
	// full name will be written as 'name.', when error
	if ext != "" {
		v.SetConfigType(ext)
	}

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := v.ReadInConfig(); err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return zero, fmt.Errorf("config %s.%s not found in %v: %w", name, ext, paths, err)
		}
		return zero, fmt.Errorf("failed to read config %q.%s: %w", name, ext, err)
	}

	var cfg T
	if err := v.Unmarshal(&cfg,
		func(dc *mapstructure.DecoderConfig) {
			dc.ErrorUnused = true
		}); err != nil {
		return zero, fmt.Errorf("failed to unmarshal config %s.%s into %T: %w", name, ext, cfg, err)
	}

	if err := validator.New(validator.WithRequiredStructEnabled()).
		Struct(&cfg); err != nil {
		return zero, fmt.Errorf("config %s.%s validation failed: %w", name, ext, err)
	}

	return &cfg, nil
}

func SaveConfig[T any](cfg *T, name, ext string) error {
	if name == "" {
		return errors.New("name is required")
	}
	path := name
	if ext != "" {
		path = fmt.Sprintf("%s.%s", name, ext)
	}

	_ = os.Remove(path)

	var data []byte
	var err error
	switch strings.ToLower(ext) {
	case "env":
		data, err = marshalEnv(cfg)
	case "":
		data, err = yaml.Marshal(cfg)
	default:
		return fmt.Errorf(
			"SaveConfig support only env"+
				" or empty extension, got: %s", ext)
	}
	if err != nil {
		return fmt.Errorf("encode config failed: %w", err)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("error when writing temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("error when renaming temp file: %w", err)
	}
	return nil
}

func marshalEnv[T any](cfg T) ([]byte, error) {
	var buf bytes.Buffer
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil, errors.New("invalid config type")
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("only config struct type is supported, got: %s", v.Kind())
	}

	// struct marshalling

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		key := f.Tag.Get("mapstructure")
		if key == "" {
			key = f.Name
		}
		key = strings.ToUpper(strings.ReplaceAll(
			strings.ReplaceAll(key, ".", "_"),
			"-", "_"))
		val := formatEnvValue(v.Field(i))
		if key != "" {
			_, _ = fmt.Fprintf(&buf, "%s=%s\n", key, val)
		}
	}
	return buf.Bytes(), nil
}

func formatEnvValue(v reflect.Value) string {
	if v.IsZero() {
		return ""
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if needsQuotes(s) {
			s = strings.ReplaceAll(s, "\\", "\\\\")
			s = strings.ReplaceAll(s, "\"", "\\\"")
			return "\"" + s + "\""
		}
		return s
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return fmt.Sprint(v.Interface())
	default:
		// fallback to json marshal
		// avoid it
		b, err := json.Marshal(v.Interface())
		if err != nil {
			return fmt.Sprintf("%q", fmt.Sprint(v.Interface()))
		}
		return string(b)
	}
}

func needsQuotes(s string) bool {
	if s == "" {
		return false
	}
	return strings.ContainsAny(s, " \t\n\r#=:\"'$")
}
