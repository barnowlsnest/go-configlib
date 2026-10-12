# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go library that provides struct-tag-driven configuration binding on top of [spf13/viper](https://github.com/spf13/viper) and [spf13/pflag](https://github.com/spf13/pflag). Define a config struct with `name`, `default`, `usage`, and `flag` tags, then call `Resolve` (or `Register` + `Load` for advanced control) to wire up pflags, env vars, and defaults automatically.

Module path: `github.com/barnowlsnest/go-configlib/v2`

## Build & Test Commands

This project uses [Task](https://taskfile.dev) (v3) as its task runner.

```bash
task go-build              # build all packages
task go-test               # run tests with coverage + benchmarks
task go-vet                # static analysis
task go-fmt                # format all Go files
task go-lint               # run golangci-lint --fix (config: .golangci.yaml)
task go-mod                # tidy dependencies
task sanity                # run all checks sequentially (mod, fmt, lint, build, vet, test)
```

CI (`.github/workflows/build.yml`) calls `task go-build` and `task go-test`, so keep those task names stable.

To run a single test directly:
```bash
go test -run TestName ./pkg/configs/
```

## Architecture

All library code lives in a single package: `pkg/configs/`.

- **configs.go** — Public API. `Config` is a type alias for `*viper.Viper`. Entry points:
  - `Resolve(cfg, prefix...)` — one-call convenience: creates a viper + FlagSet, registers, parses, binds, and loads. Returns `*Config`.
  - `ResolveWithFlagSet(v, fs, cfg, prefix...)` — same as `Resolve` but with caller-provided viper and FlagSet.
  - `Register(v, fs, cfg, prefix...)` — reflects over a struct to register pflag flags (with zero-value defaults), set viper defaults from `default` tags, and auto-bind env vars (`strings.ToUpper(key)`).
  - `Load(v, cfg, prefix...)` — reads resolved viper values back into the struct fields. Call after `flag.Parse()` and `v.BindPFlags(fs)`.
- **utils.go** — Internal helpers:
  - `derefStructPtr` — shared validation for `Register`/`Load`: extracts prefix, validates cfg is pointer-to-struct, returns dereferenced `reflect.Value`.
  - `checkSupportedKind` — single source of truth for which `reflect.Kind` values are supported.
  - `registerFields` / `loadFields` — recursively walk struct fields. Pointer types are unwrapped (nil pointers allocated in `loadFields`).
  - `registerFlag` — registers a single pflag for a leaf field (flag name = full key, e.g. `db_host`).
  - `joinKey` — builds `prefix_name` keys.
  - `parseDefault` — converts `default` tag strings to typed values.
- **errors.go** — Sentinel error `ErrConfig`. All returned errors wrap it for `errors.Is` checking.
- **configs_test.go** — Table-driven tests using testify (`require`/`assert`).

### Struct tags

Tag name constants are defined in `utils.go` (`tagName`, `tagDefault`, `tagUsage`, `tagFlag`).

| Tag       | Purpose                                         |
|-----------|-------------------------------------------------|
| `name`    | Viper/pflag key segment (required; `"-"` skips) |
| `default` | Default value (parsed per field type)            |
| `usage`   | pflag usage string                               |
| `flag`    | Set to `"-"` to skip pflag registration (env/default only) |

Nested structs (and pointer-to-struct) use their `name` tag as the prefix for child keys, joined with `_`.

### Supported field types

`string`, `int` (all widths), `float64`, `bool`, `time.Duration`, and their pointer variants. `time.Duration` is detected via `durationType` before the kind switch (its kind is `Int64`). Adding a new type requires updating `checkSupportedKind`, `registerFlag` (pflag registration switch), `loadFields` (viper getter switch), and `parseDefault`.

### Linting

golangci-lint config (`.golangci.yaml`) enforces `funlen` (100 lines / 50 statements), `dupl` (threshold 100), `gocyclo` (max 15), and `lll` (140 chars). Test files are excluded from `dupl`, `funlen`, `goconst`, `gocyclo`, and `gosec`. Import ordering uses `goimports` with local prefix `github.com/barnowlsnest/go-configlib/v2`.
