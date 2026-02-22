// Package configs provides struct-tag-driven configuration binding on top of
// spf13/viper and spf13/pflag. Define a config struct with name, default, usage,
// and flag tags, then call [Register] and [Load] to wire up pflags, environment
// variables, and defaults automatically.
package configs

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config = viper.Viper

// Register reflects over spec (pointer to struct), registers pflag flags with
// zero defaults, sets viper defaults from the "default" tag, and auto-binds env
// vars derived from strings.ToUpper(key). An optional prefix is prepended to all
// keys. When a field is a struct, its "name" tag becomes the prefix for nested
// fields.
func Register(v *Config, fs *pflag.FlagSet, spec any, prefix ...string) error {
	val, p, err := derefStructPtr(spec, prefix)
	if err != nil {
		return err
	}
	return registerFields(v, fs, val.Type(), p)
}

// Load reads viper values into spec using the "name" tag and optional prefix.
// Call after flag parse and v.BindPFlags(fs).
func Load(v *Config, spec any, prefix ...string) error {
	val, p, err := derefStructPtr(spec, prefix)
	if err != nil {
		return err
	}
	return loadFields(v, val, p)
}
