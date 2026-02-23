// Package configs provides struct-tag-driven configuration binding on top of
// spf13/viper and spf13/pflag. Define a config struct with name, default, usage,
// and flag tags, then call [Register] and [Load] to wire up pflags, environment
// variables, and defaults automatically.
package configs

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config = viper.Viper

// Register reflects over cfg (pointer to struct), registers pflag flags with
// zero defaults, sets viper defaults from the "default" tag, and auto-binds env
// vars derived from strings.ToUpper(key). An optional prefix is prepended to all
// keys. When a field is a struct, its "name" tag becomes the prefix for nested
// fields.
func Register(v *Config, fs *pflag.FlagSet, cfg any, prefix ...string) error {
	val, p, err := derefStructPtr(cfg, prefix)
	if err != nil {
		return err
	}
	return registerFields(v, fs, val.Type(), p)
}

// Load reads viper values into cfg using the "name" tag and optional prefix.
// Call after flag parse and v.BindPFlags(fs).
func Load(v *Config, cfg any, prefix ...string) error {
	val, p, err := derefStructPtr(cfg, prefix)
	if err != nil {
		return err
	}
	return loadFields(v, val, p)
}

// ResolveWithFlagSet registers flags, parses CLI args, binds flags to viper,
// and loads resolved values into cfg.
//
// Resolution order (highest priority first, per viper's precedence):
//  1. CLI flags (--key=value)
//  2. Environment variables (KEY)
//  3. Defaults from the "default" struct tag
func ResolveWithFlagSet(v *Config, fs *pflag.FlagSet, cfg any, prefix ...string) error {
	if err := Register(v, fs, cfg, prefix...); err != nil {
		return err
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		return fmt.Errorf("failed to parse flags: %w: %w", err, ErrConfig)
	}
	if err := v.BindPFlags(fs); err != nil {
		return fmt.Errorf("failed to bind flags: %w: %w", err, ErrConfig)
	}
	return Load(v, cfg, prefix...)
}

// Resolve creates a new viper instance and default FlagSet, then calls
// ResolveWithFlagSet. Returns the Config for further viper queries.
func Resolve(cfg any, prefix ...string) (*Config, error) {
	v := viper.New()
	fs := pflag.CommandLine
	if err := ResolveWithFlagSet(v, fs, cfg, prefix...); err != nil {
		return nil, err
	}
	return v, nil
}
