# go-configlib

Struct-tag-driven configuration binding for Go, built on [spf13/viper](https://github.com/spf13/viper) and [spf13/pflag](https://github.com/spf13/pflag).

Define a config struct once — get pflags, environment variables, and defaults wired up automatically.

## Install

```bash
go get github.com/barnowlsnest/go-configlib
```

## Usage

```go
package main

import (
    "fmt"

    "github.com/barnowlsnest/go-configlib/pkg/configs"
    "github.com/spf13/pflag"
    "github.com/spf13/viper"
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
    v := viper.New()
    fs := pflag.CommandLine
    cfg := &AppConfig{}

    // Register pflags, defaults, and env bindings.
    if err := configs.Register(v, fs, cfg); err != nil {
        panic(err)
    }

    pflag.Parse()
    _ = v.BindPFlags(fs)

    // Load resolved values into the struct.
    if err := configs.Load(v, cfg); err != nil {
        panic(err)
    }

    fmt.Printf("%+v\n", cfg)
}
```

## Struct Tags

| Tag       | Purpose                                  | Example            |
|-----------|------------------------------------------|--------------------|
| `name`    | Key segment (required; `"-"` skips)      | `name:"host"`      |
| `default` | Default value, parsed per field type     | `default:"8080"`   |
| `usage`   | pflag usage/help string                  | `usage:"server port"` |
| `flag`    | Set to `"-"` to skip pflag registration  | `flag:"-"`         |

## Supported Field Types

`string`, `int` (all widths: 8/16/32/64), `bool`.

Pointer variants (`*string`, `*int`, `*bool`) are supported — nil pointers are allocated on `Load`.

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

Both `Register` and `Load` accept an optional prefix to namespace all keys:

```go
configs.Register(v, fs, cfg, "myapp")
configs.Load(v, cfg, "myapp")
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
