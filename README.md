# go-configlib

[![Build and Test](https://github.com/barnowlsnest/go-configlib/actions/workflows/build.yml/badge.svg?branch=main)](https://github.com/barnowlsnest/go-configlib/actions/workflows/build.yml)
[![Lint](https://github.com/barnowlsnest/go-configlib/actions/workflows/golangci-lint.yml/badge.svg?branch=main)](https://github.com/barnowlsnest/go-configlib/actions/workflows/golangci-lint.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/barnowlsnest/go-configlib/v2.svg)](https://pkg.go.dev/github.com/barnowlsnest/go-configlib/v2/pkg/configs)
[![Go Report Card](https://goreportcard.com/badge/github.com/barnowlsnest/go-configlib/v2)](https://goreportcard.com/report/github.com/barnowlsnest/go-configlib/v2)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Struct-tag-driven configuration binding for Go, built on [spf13/viper](https://github.com/spf13/viper) and [spf13/pflag](https://github.com/spf13/pflag).

Define a config struct once, and get CLI flags, environment variables, and defaults wired up automatically.

## Contents

- [Features](#features)
- [Requirements](#requirements)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Resolution Order](#resolution-order)
- [Advanced Usage](#advanced-usage)
- [Reference](#reference)
- [Error Handling](#error-handling)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Features

- **One struct, three sources**: CLI flags, environment variables, and defaults from a single struct definition.
- **One call to resolve**: `Resolve` registers, parses, binds, and loads in one step.
- **Nested and pointer structs**: nested structs become key prefixes (`db_host`), and nil pointers are allocated for you.
- **Optional namespacing**: a prefix (`myapp_host` / `MYAPP_HOST`) avoids collisions with other tools.
- **Escape hatches**: `Register` + `Load` let you add your own viper sources (config files, remote stores) between steps.
- **Viper underneath**: the returned `*configs.Config` is a `*viper.Viper`, so the full viper API stays available.

## Requirements

- Go **1.27** or later (see [`go.mod`](go.mod))

## Installation

```bash
go get github.com/barnowlsnest/go-configlib/v2
```

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/barnowlsnest/go-configlib/v2/pkg/configs"
)

type AppConfig struct {
    Host    string        `name:"host"    default:"localhost" usage:"server host"`
    Port    int           `name:"port"    default:"8080"      usage:"server port"`
    TLS     bool          `name:"tls"     default:"false"     usage:"enable TLS"`
    Timeout time.Duration `name:"timeout" default:"30s"       usage:"request timeout"`
    DB      struct {
        Host string `name:"host" default:"127.0.0.1" usage:"database host"`
        Port int    `name:"port" default:"5432"      usage:"database port"`
    } `name:"db"`
}

func main() {
    cfg := &AppConfig{}
    v, err := configs.Resolve(cfg)
    if err != nil {
        panic(err)
    }

    fmt.Printf("%+v\n", cfg)
    fmt.Println("host from viper:", v.GetString("host"))
}
```

Run it with any mix of sources:

```bash
DB_HOST=db.internal go run . --port=9090 --tls
```

`Resolve` creates a viper instance, registers flags, defaults, and env bindings on `pflag.CommandLine`, parses `os.Args`, and loads the resolved values into the struct.

> **Note:** `Resolve` registers flags on the global `pflag.CommandLine`, so call it only once per process. Use [`ResolveWithFlagSet`](#resolvewithflagset) when you need an isolated FlagSet (tests, multiple configs, subcommands).

## Resolution Order

Values are resolved in this order, highest priority first (viper's precedence):

1. **CLI flags** (`--key=value`)
2. **Environment variables** (`KEY`)
3. **Defaults** from the `default` struct tag

Each field's env var is bound explicitly during registration with `v.BindEnv(key, strings.ToUpper(key))`.

## Advanced Usage

### ResolveWithFlagSet

Use `ResolveWithFlagSet` when you need your own viper instance or FlagSet:

```go
v := viper.New()
fs := pflag.NewFlagSet("myapp", pflag.ContinueOnError)
cfg := &AppConfig{}

if err := configs.ResolveWithFlagSet(v, fs, cfg); err != nil {
    panic(err)
}
```

### Register + Load

For full control, for example to add a config file between registration and loading:

```go
v := viper.New()
fs := pflag.CommandLine
cfg := &AppConfig{}

if err := configs.Register(v, fs, cfg); err != nil {
    panic(err)
}

v.SetConfigFile("config.yaml")
_ = v.ReadInConfig() // optional extra source

pflag.Parse()
if err := v.BindPFlags(fs); err != nil {
    panic(err)
}

if err := configs.Load(v, cfg); err != nil {
    panic(err)
}
```

## Reference

### API

| Function | Description |
|----------|-------------|
| `Resolve(cfg, prefix...) (*Config, error)` | Creates a viper instance, uses `pflag.CommandLine`, then registers, parses, binds, and loads. |
| `ResolveWithFlagSet(v, fs, cfg, prefix...) error` | Same as `Resolve`, but with a viper instance and FlagSet you provide. |
| `Register(v, fs, cfg, prefix...) error` | Registers flags, viper defaults, and env bindings for every tagged field. |
| `Load(v, cfg, prefix...) error` | Reads resolved viper values back into the struct. Call after parsing flags and `v.BindPFlags(fs)`. |

`cfg` must be a pointer to a struct. `Config` is an alias for `viper.Viper`.

Full API docs: [pkg.go.dev](https://pkg.go.dev/github.com/barnowlsnest/go-configlib/v2/pkg/configs).

### Struct Tags

| Tag       | Purpose                                                  | Example               |
|-----------|----------------------------------------------------------|-----------------------|
| `name`    | Key segment. Required; fields without it or with `"-"` are skipped | `name:"host"` |
| `default` | Default value, parsed according to the field type        | `default:"8080"`      |
| `usage`   | pflag usage/help string                                  | `usage:"server port"` |
| `flag`    | Set to `"-"` to skip pflag registration                  | `flag:"-"`            |

### Supported Field Types

| Type | Pointer variant | Default / env format |
|------|-----------------|----------------------|
| `string` | `*string` | any string |
| `int`, `int8`, `int16`, `int32`, `int64` | `*int`, ... | base-10 integer |
| `float64` | `*float64` | decimal number |
| `bool` | `*bool` | `true` / `false` |
| `time.Duration` | `*time.Duration` | Go duration string (`2s`, `50m`, `1h30m`) |

Nil pointer fields are allocated on `Load`. Any other type returns an error wrapping `ErrConfig`.

### Keys, Flags, and Environment Variables

A field's **key** is its `name` tag, prefixed by any parent struct names and the optional global prefix, joined with `_`. The key is also the flag name; the env var is the key in upper case.

```go
type Config struct {
    Host string `name:"host"`
    DB   struct {
        Port int `name:"port"`
    } `name:"db"`
}

configs.Resolve(&Config{}, "myapp")
```

| Field     | Key            | Flag             | Env var        |
|-----------|----------------|------------------|----------------|
| `Host`    | `myapp_host`   | `--myapp_host`   | `MYAPP_HOST`   |
| `DB.Port` | `myapp_db_port`| `--myapp_db_port`| `MYAPP_DB_PORT`|

Nested pointer-to-struct fields (`*struct{...}`) work the same way.

### Env-Only Fields

Use `flag:"-"` to keep a value out of the CLI, for example secrets that should not appear in shell history or `ps` output. The field still gets a default and an env binding:

```go
Secret string `name:"secret" default:"changeme" flag:"-"`
// No --secret flag, but SECRET and the default still apply.
```

## Error Handling

Every error returned by the library wraps the sentinel `configs.ErrConfig`:

```go
if _, err := configs.Resolve(cfg); err != nil {
    if errors.Is(err, configs.ErrConfig) {
        // invalid struct, unsupported type, bad default, or flag parse failure
    }
}
```

## Development

This project uses [Task](https://taskfile.dev) (v3) and [golangci-lint](https://golangci-lint.run).

```bash
task go-build   # build all packages
task go-test    # run tests with coverage and benchmarks
task go-lint    # run golangci-lint (config: .golangci.yaml)
task sanity     # tidy, fmt, lint, build, vet, and test
```

Run a single test:

```bash
go test -run TestName ./pkg/configs/
```

## Contributing

Contributions are welcome.

1. Open an issue to discuss bugs or larger changes before you start.
2. Fork the repo and create a branch from `main`.
3. Add or update tests for your change. Tests are table-driven and use [testify](https://github.com/stretchr/testify).
4. Run `task sanity` and make sure it passes.
5. Open a pull request with a clear description of the change.

Adding a new field type means updating `checkSupportedKind`, `registerFlag`, `loadFields`, and `parseDefault` in `pkg/configs/utils.go`.

## License

Released under the [MIT License](LICENSE). Copyright (c) 2026 Barn Owls Nest.
