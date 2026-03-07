# go-configlib

Struct-tag-driven configuration binding for Go, built on [spf13/viper](https://github.com/spf13/viper) and [spf13/pflag](https://github.com/spf13/pflag).

Define a config struct once — get pflags, environment variables, and defaults wired up automatically.

## Install

```bash
go get github.com/barnowlsnest/go-configlib/v2
```

## Usage

```go
package main

import (
    "fmt"

    "github.com/barnowlsnest/go-configlib/v2/pkg/configs"
)

type AppConfig struct {
    Host string `name:"host" default:"localhost" usage:"server host"`
    Port int    `name:"port" default:"8080"      usage:"server port"`
    TLS  bool   `name:"tls"  default:"false"     usage:"enable TLS"`
    DB   struct {
        Host string `name:"host" default:"127.0.0.1"`
        Port int    `name:"port" default:"5432"`
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

`Resolve` creates a viper instance and FlagSet, registers flags + defaults + env bindings, parses CLI args, and loads resolved values into the struct — all in one call.

## Resolution Order

Values are resolved with the following priority (highest first, per viper's precedence):

1. **CLI flags** (`--key=value`)
2. **Environment variables** (`KEY`)
3. **Defaults** from the `default` struct tag

Environment variables are explicitly bound per field during registration via `v.BindEnv(key, UPPER_KEY)`.

## Advanced Usage

### ResolveWithFlagSet

Use `ResolveWithFlagSet` when you need a custom viper instance or FlagSet:

```go
v := viper.New()
fs := pflag.NewFlagSet("myapp", pflag.ExitOnError)
cfg := &AppConfig{}

if err := configs.ResolveWithFlagSet(v, fs, cfg); err != nil {
    panic(err)
}
```

### Register + Load

For full control (e.g. adding extra viper config sources between registration and loading):

```go
v := viper.New()
fs := pflag.CommandLine
cfg := &AppConfig{}

if err := configs.Register(v, fs, cfg); err != nil {
    panic(err)
}

pflag.Parse()
_ = v.BindPFlags(fs)

if err := configs.Load(v, cfg); err != nil {
    panic(err)
}
```

## Struct Tags

| Tag       | Purpose                                  | Example               |
|-----------|------------------------------------------|-----------------------|
| `name`    | Key segment (required; `"-"` skips)      | `name:"host"`         |
| `default` | Default value, parsed per field type     | `default:"8080"`      |
| `usage`   | pflag usage/help string                  | `usage:"server port"` |
| `flag`    | Set to `"-"` to skip pflag registration  | `flag:"-"`            |

## Supported Field Types

`string`, `int` (all widths: 8/16/32/64), `float64`, `bool`, `time.Duration`.

Pointer variants (`*string`, `*int`, `*float64`, `*bool`, `*time.Duration`) are supported — nil pointers are allocated on `Load`.

Duration fields accept Go duration strings (e.g. `"2s"`, `"50m"`, `"1h30m"`) in defaults, env vars, and flags.

## Nested Structs

Nested structs (including pointer-to-struct) use their `name` tag as a prefix, joined with `_`:

```go
type Config struct {
    DB struct {
        Host string `name:"host"`
    } `name:"db"`
}
// key: db_host  |  env: DB_HOST
```

## Prefix

All functions accept an optional prefix to namespace all keys:

```go
configs.Resolve(cfg, "myapp")
// key: myapp_host  |  env: MYAPP_HOST
```

## Environment Variables

Environment variables are auto-bound as `strings.ToUpper(key)`:

| Key         | Env Var     |
|-------------|-------------|
| `host`      | `HOST`      |
| `db_port`   | `DB_PORT`   |
| `myapp_tls` | `MYAPP_TLS` |

## Flag-Only and Env-Only Fields

Use `flag:"-"` to skip pflag registration. The field still gets a viper default and env binding:

```go
Secret string `name:"secret" default:"changeme" flag:"-"`
// No --secret flag, but SECRET env var and default still work.
```

## Error Handling

All errors wrap `configs.ErrConfig`, so you can check with `errors.Is(err, configs.ErrConfig)`.
