package configs_test

import (
	"errors"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barnowlsnest/go-configlib/pkg/configs"
)

type basicSpec struct {
	Host string `name:"host" default:"localhost" usage:"server host"`
	Port int    `name:"port" default:"8080" usage:"server port"`
	TLS  bool   `name:"tls" default:"true" usage:"enable TLS"`
}

type nestedSpec struct {
	DB struct {
		Host string `name:"host" default:"127.0.0.1"`
		Port int    `name:"port" default:"5432"`
	} `name:"db"`
}

type pointerSpec struct {
	Name *string `name:"name" default:"test"`
	Port *int    `name:"port" default:"3000"`
	TLS  *bool   `name:"tls" default:"false"`
}

type pointerStructSpec struct {
	DB *struct {
		Host string `name:"host" default:"localhost"`
	} `name:"db"`
}

type skipFieldSpec struct {
	Name   string `name:"name" default:"app"`
	Secret string `name:"secret" default:"s3cret" flag:"-"`
}

type skippedNameSpec struct {
	Included string `name:"included"`
	Skipped  string `name:"-"`
	NoTag    string
}

type unsupportedSpec struct {
	Data float64 `name:"data"`
}

type invalidDefaultSpec struct {
	Port int `name:"port" default:"notanumber"`
}

func newViperAndFlags() (*configs.Config, *pflag.FlagSet) {
	return viper.New(), pflag.NewFlagSet("test", pflag.ContinueOnError)
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name      string
		cfg       any
		prefix    []string
		wantErr   bool
		errTarget error
		check     func(t *testing.T, v *configs.Config, fs *pflag.FlagSet)
	}{
		{
			name: "basic types with defaults",
			cfg:  &basicSpec{},
			check: func(t *testing.T, v *configs.Config, fs *pflag.FlagSet) {
				assert.Equal(t, "localhost", v.GetString("host"))
				assert.Equal(t, 8080, v.GetInt("port"))
				assert.Equal(t, true, v.GetBool("tls"))

				assert.NotNil(t, fs.Lookup("host"))
				assert.NotNil(t, fs.Lookup("port"))
				assert.NotNil(t, fs.Lookup("tls"))
			},
		},
		{
			name:   "with prefix",
			cfg:    &basicSpec{},
			prefix: []string{"app"},
			check: func(t *testing.T, v *configs.Config, fs *pflag.FlagSet) {
				assert.Equal(t, "localhost", v.GetString("app_host"))
				assert.Equal(t, 8080, v.GetInt("app_port"))
				assert.NotNil(t, fs.Lookup("app_host"))
			},
		},
		{
			name: "nested struct",
			cfg:  &nestedSpec{},
			check: func(t *testing.T, v *configs.Config, fs *pflag.FlagSet) {
				assert.Equal(t, "127.0.0.1", v.GetString("db_host"))
				assert.Equal(t, 5432, v.GetInt("db_port"))
				assert.NotNil(t, fs.Lookup("db_host"))
			},
		},
		{
			name: "pointer fields",
			cfg:  &pointerSpec{},
			check: func(t *testing.T, v *configs.Config, fs *pflag.FlagSet) {
				assert.Equal(t, "test", v.GetString("name"))
				assert.Equal(t, 3000, v.GetInt("port"))
				assert.Equal(t, false, v.GetBool("tls"))
				assert.NotNil(t, fs.Lookup("name"))
			},
		},
		{
			name: "pointer to struct",
			cfg:  &pointerStructSpec{},
			check: func(t *testing.T, v *configs.Config, fs *pflag.FlagSet) {
				assert.Equal(t, "localhost", v.GetString("db_host"))
				assert.NotNil(t, fs.Lookup("db_host"))
			},
		},
		{
			name: "flag skip registers default but no flag",
			cfg:  &skipFieldSpec{},
			check: func(t *testing.T, v *configs.Config, fs *pflag.FlagSet) {
				assert.Equal(t, "app", v.GetString("name"))
				assert.Equal(t, "s3cret", v.GetString("secret"))
				assert.NotNil(t, fs.Lookup("name"))
				assert.Nil(t, fs.Lookup("secret"))
			},
		},
		{
			name: "skipped and untagged fields ignored",
			cfg:  &skippedNameSpec{},
			check: func(t *testing.T, v *configs.Config, fs *pflag.FlagSet) {
				assert.NotNil(t, fs.Lookup("included"))
				assert.Nil(t, fs.Lookup("-"))
				assert.Nil(t, fs.Lookup("Skipped"))
				assert.Nil(t, fs.Lookup("NoTag"))
			},
		},
		{
			name:      "non-pointer cfg",
			cfg:       basicSpec{},
			wantErr:   true,
			errTarget: configs.ErrConfig,
		},
		{
			name:      "pointer to non-struct",
			cfg:       ptrTo("hello"),
			wantErr:   true,
			errTarget: configs.ErrConfig,
		},
		{
			name:      "unsupported field type",
			cfg:       &unsupportedSpec{},
			wantErr:   true,
			errTarget: configs.ErrConfig,
		},
		{
			name:      "invalid default value",
			cfg:       &invalidDefaultSpec{},
			wantErr:   true,
			errTarget: configs.ErrConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, fs := newViperAndFlags()
			err := configs.Register(v, fs, tt.cfg, tt.prefix...)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errTarget != nil {
					assert.True(t, errors.Is(err, tt.errTarget), "expected error wrapping %v, got: %v", tt.errTarget, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, v, fs)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		cfg       func() any
		prefix    []string
		setup     func(v *configs.Config)
		wantErr   bool
		errTarget error
		check     func(t *testing.T, cfg any)
	}{
		{
			name: "basic types",
			cfg:  func() any { return &basicSpec{} },
			setup: func(v *configs.Config) {
				v.Set("host", "example.com")
				v.Set("port", 9090)
				v.Set("tls", false)
			},
			check: func(t *testing.T, cfg any) {
				s := cfg.(*basicSpec)
				assert.Equal(t, "example.com", s.Host)
				assert.Equal(t, 9090, s.Port)
				assert.Equal(t, false, s.TLS)
			},
		},
		{
			name:   "with prefix",
			cfg:    func() any { return &basicSpec{} },
			prefix: []string{"app"},
			setup: func(v *configs.Config) {
				v.Set("app_host", "prefixed.com")
				v.Set("app_port", 7070)
				v.Set("app_tls", true)
			},
			check: func(t *testing.T, cfg any) {
				s := cfg.(*basicSpec)
				assert.Equal(t, "prefixed.com", s.Host)
				assert.Equal(t, 7070, s.Port)
				assert.Equal(t, true, s.TLS)
			},
		},
		{
			name: "nested struct",
			cfg:  func() any { return &nestedSpec{} },
			setup: func(v *configs.Config) {
				v.Set("db_host", "10.0.0.1")
				v.Set("db_port", 3306)
			},
			check: func(t *testing.T, cfg any) {
				s := cfg.(*nestedSpec)
				assert.Equal(t, "10.0.0.1", s.DB.Host)
				assert.Equal(t, 3306, s.DB.Port)
			},
		},
		{
			name: "pointer fields allocated from nil",
			cfg:  func() any { return &pointerSpec{} },
			setup: func(v *configs.Config) {
				v.Set("name", "loaded")
				v.Set("port", 4000)
				v.Set("tls", true)
			},
			check: func(t *testing.T, cfg any) {
				s := cfg.(*pointerSpec)
				require.NotNil(t, s.Name)
				require.NotNil(t, s.Port)
				require.NotNil(t, s.TLS)
				assert.Equal(t, "loaded", *s.Name)
				assert.Equal(t, 4000, *s.Port)
				assert.Equal(t, true, *s.TLS)
			},
		},
		{
			name: "pointer to struct allocated from nil",
			cfg:  func() any { return &pointerStructSpec{} },
			setup: func(v *configs.Config) {
				v.Set("db_host", "dbhost")
			},
			check: func(t *testing.T, cfg any) {
				s := cfg.(*pointerStructSpec)
				require.NotNil(t, s.DB)
				assert.Equal(t, "dbhost", s.DB.Host)
			},
		},
		{
			name: "skipped and untagged fields untouched",
			cfg:  func() any { return &skippedNameSpec{Skipped: "keep", NoTag: "keep"} },
			setup: func(v *configs.Config) {
				v.Set("included", "yes")
			},
			check: func(t *testing.T, cfg any) {
				s := cfg.(*skippedNameSpec)
				assert.Equal(t, "yes", s.Included)
				assert.Equal(t, "keep", s.Skipped)
				assert.Equal(t, "keep", s.NoTag)
			},
		},
		{
			name:      "non-pointer cfg",
			cfg:       func() any { return basicSpec{} },
			wantErr:   true,
			errTarget: configs.ErrConfig,
		},
		{
			name:      "pointer to non-struct",
			cfg:       func() any { return ptrTo("hello") },
			wantErr:   true,
			errTarget: configs.ErrConfig,
		},
		{
			name:      "unsupported field type",
			cfg:       func() any { return &unsupportedSpec{} },
			wantErr:   true,
			errTarget: configs.ErrConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			cfg := tt.cfg()

			if tt.setup != nil {
				tt.setup(v)
			}

			err := configs.Load(v, cfg, tt.prefix...)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errTarget != nil {
					assert.True(t, errors.Is(err, tt.errTarget), "expected error wrapping %v, got: %v", tt.errTarget, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestResolveWithFlagSet(t *testing.T) {
	tests := []struct {
		name      string
		cfg       func() any
		prefix    []string
		env       map[string]string
		wantErr   bool
		errTarget error
		check     func(t *testing.T, cfg any)
	}{
		{
			name: "defaults only",
			cfg:  func() any { return &basicSpec{} },
			check: func(t *testing.T, cfg any) {
				s := cfg.(*basicSpec)
				assert.Equal(t, "localhost", s.Host)
				assert.Equal(t, 8080, s.Port)
				assert.Equal(t, true, s.TLS)
			},
		},
		{
			name: "env var override",
			cfg:  func() any { return &basicSpec{} },
			env:  map[string]string{"HOST": "envhost", "PORT": "9999"},
			check: func(t *testing.T, cfg any) {
				s := cfg.(*basicSpec)
				assert.Equal(t, "envhost", s.Host)
				assert.Equal(t, 9999, s.Port)
				assert.Equal(t, true, s.TLS)
			},
		},
		{
			name:      "non-pointer cfg",
			cfg:       func() any { return basicSpec{} },
			wantErr:   true,
			errTarget: configs.ErrConfig,
		},
		{
			name:      "unsupported field type",
			cfg:       func() any { return &unsupportedSpec{} },
			wantErr:   true,
			errTarget: configs.ErrConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, val := range tt.env {
				t.Setenv(k, val)
			}

			v := viper.New()
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			cfg := tt.cfg()

			err := configs.ResolveWithFlagSet(v, fs, cfg, tt.prefix...)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errTarget != nil {
					assert.True(t, errors.Is(err, tt.errTarget),
						"expected error wrapping %v, got: %v", tt.errTarget, err)
				}
				return
			}

			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	cfg := &basicSpec{}
	v, err := configs.Resolve(cfg)
	require.NoError(t, err)
	require.NotNil(t, v)

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)
	assert.Equal(t, true, cfg.TLS)
	assert.Equal(t, "localhost", v.GetString("host"))
}

func ptrTo[T any](v T) *T {
	return &v
}
