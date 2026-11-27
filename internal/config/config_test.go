package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	dsnVar    = "CH_DSN"
	dsnRef    = "clickhouse:\n  dsn: ${CH_DSN}\n"
	remoteDSN = "clickhouse://remote:9000"
	userVar   = "CH_USER"
	yamlNull  = "null"
	dsnOrDef  = "clickhouse:\n  dsn: ${CH_DSN:-clickhouse://localhost:9000}\n"
)

func TestLoadSubstitutesEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		content string
		env     map[string]string
		want    string
	}{
		{
			name:    "simple variable",
			content: dsnRef,
			env:     map[string]string{dsnVar: remoteDSN},
			want:    remoteDSN,
		},
		{
			name:    "default used when variable unset",
			content: dsnOrDef,
			want:    "clickhouse://localhost:9000",
		},
		{
			name:    "default used when variable empty",
			content: dsnOrDef,
			env:     map[string]string{dsnVar: ""},
			want:    "clickhouse://localhost:9000",
		},
		{
			name:    "default ignored when variable set",
			content: dsnOrDef,
			env:     map[string]string{dsnVar: remoteDSN},
			want:    remoteDSN,
		},
		{
			name:    "empty default allowed",
			content: "clickhouse:\n  dsn: \"${CH_DSN:-}\"\n",
			want:    "",
		},
		{
			name:    "multiple variables in one value",
			content: "clickhouse:\n  dsn: clickhouse://${CH_USER}:${CH_PASS}@host:9000\n",
			env:     map[string]string{userVar: "reader", "CH_PASS": "secret"},
			want:    "clickhouse://reader:secret@host:9000",
		},
		{
			name:    "dollar without braces is left unchanged",
			content: "clickhouse:\n  dsn: clickhouse://${CH_USER}:pa$word@host:9000\n",
			env:     map[string]string{userVar: "admin", "word": "replaced"},
			want:    "clickhouse://admin:pa$word@host:9000",
		},
		{
			name:    "comments are not expanded",
			content: "clickhouse:\n  # dsn: ${CH_UNSET}\n  dsn: ${CH_DSN} # ${CH_UNSET}\n",
			env:     map[string]string{dsnVar: remoteDSN},
			want:    remoteDSN,
		},
		{
			name:    "keys are not expanded",
			content: "clickhouse:\n  dsn: ${CH_DSN}\ntelemetry:\n  headers:\n    ${CH_UNSET}: value\n",
			env:     map[string]string{dsnVar: remoteDSN},
			want:    remoteDSN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			cfg, err := loadFromString(t, tt.content)
			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg.ClickHouse.DSN)
		})
	}
}

func TestLoadSubstitutedValuesAreVerbatim(t *testing.T) {
	values := map[string]string{
		"tab":       "clickhouse://u:a\\tb\t@host:9000",
		"comment":   "clickhouse://u:p #x@host:9000",
		"newline":   "a\"\n  insecure_skip_verify: true\n  x: \"y",
		"quote":     "clickhouse://u:pa\"ss'@host:9000",
		"pem":       "-----BEGIN KEY-----\nabc\n# not a comment\n-----END KEY-----\n",
		"yaml-like": "[1, 2] &x {a: *x}",
		"null":      yamlNull,
		"tilde":     "~",
	}
	placeholders := map[string]string{
		"plain":         "${V}",
		"double-quoted": "\"${V}\"",
		"single-quoted": "'${V}'",
	}

	for valueName, value := range values {
		for placeholderName, placeholder := range placeholders {
			t.Run(valueName+"/"+placeholderName, func(t *testing.T) {
				t.Setenv("V", value)

				cfg, err := loadFromString(t, "clickhouse:\n  dsn: "+placeholder+"\n  database: mainnet\n")
				require.NoError(t, err)
				assert.Equal(t, value, cfg.ClickHouse.DSN)
				assert.Equal(t, "mainnet", cfg.ClickHouse.Database)
				assert.False(t, cfg.ClickHouse.InsecureSkipVerify)
			})
		}
	}
}

func TestLoadSubstitutesNonStringFields(t *testing.T) {
	t.Setenv("PORT", "8181")
	t.Setenv("FINAL", "true")

	cfg, err := loadFromString(t, "server:\n  port: ${PORT}\nclickhouse:\n  use_final: ${FINAL}\n")
	require.NoError(t, err)
	assert.Equal(t, 8181, cfg.Server.Port)
	assert.True(t, cfg.ClickHouse.UseFinal)
}

func TestSubstituteEnvVarsTyping(t *testing.T) {
	t.Setenv("PORT", "8181")
	t.Setenv("FINAL", "true")
	t.Setenv("HEX", "0x1F")
	t.Setenv("V", "a: b")
	t.Setenv("NULL", yamlNull)
	t.Setenv("TILDE", "~")

	out, err := substituteEnvVars([]byte(`
plain_int: ${PORT}
plain_bool: ${FINAL}
plain_hex: ${HEX}
plain_mixed: port-${PORT}
plain_mapping_like: ${V}
plain_empty_default: ${UNSET:-}
plain_null: ${NULL}
plain_tilde: ${TILDE}
double_quoted: "${PORT}"
single_quoted: '${FINAL}'
literal_block: |
  ${PORT}
untouched: "8080"
`))
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, yaml.Unmarshal(out, &got))

	assert.Equal(t, map[string]any{
		"plain_int":           8181,
		"plain_bool":          true,
		"plain_hex":           31,
		"plain_mixed":         "port-8181",
		"plain_mapping_like":  "a: b",
		"plain_empty_default": "",
		"plain_null":          yamlNull,
		"plain_tilde":         "~",
		"double_quoted":       "8181",
		"single_quoted":       "true",
		"literal_block":       "8181\n",
		"untouched":           "8080",
	}, got)
}

func TestLoadFailsOnMissingEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		content string
		env     map[string]string
		missing []string
	}{
		{
			name:    "unset variable",
			content: dsnRef,
			missing: []string{dsnVar},
		},
		{
			name:    "empty variable",
			content: dsnRef,
			env:     map[string]string{dsnVar: ""},
			missing: []string{dsnVar},
		},
		{
			name:    "every missing variable is named",
			content: "clickhouse:\n  dsn: clickhouse://${CH_USER}:${CH_PASS}@${CH_HOST:-localhost}:9000\n  database: ${CH_DB}${CH_PASS}\n",
			env:     map[string]string{userVar: "reader"},
			missing: []string{"CH_PASS", "CH_DB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			_, err := loadFromString(t, tt.content)
			require.Error(t, err)

			for _, name := range tt.missing {
				assert.Equal(t, 1, strings.Count(err.Error(), name), name)
			}

			assert.NotContains(t, err.Error(), userVar)
			assert.NotContains(t, err.Error(), "CH_HOST")
		})
	}
}

func TestLoadWithoutEnvVarsIsUnchanged(t *testing.T) {
	t.Setenv("word", "replaced")

	cfg, err := loadFromString(t, `
clickhouse:
  dsn: "clickhouse://u:pa$word@host:9000"
  database: "mainnet"
`)
	require.NoError(t, err)

	assert.Equal(t, "clickhouse://u:pa$word@host:9000", cfg.ClickHouse.DSN)
	assert.Equal(t, "mainnet", cfg.ClickHouse.Database)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoadMissingFile(t *testing.T) {
	viper.Reset()

	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
}

func loadFromString(t *testing.T, content string) (*Config, error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	viper.Reset()
	t.Cleanup(viper.Reset)

	return Load(path)
}
