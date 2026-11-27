package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dsnVar         = "CH_DSN"
	dsnWithDefault = "dsn: ${CH_DSN:-clickhouse://localhost:9000}"
)

func TestSubstituteEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		content string
		env     map[string]string
		want    string
	}{
		{
			name:    "simple variable",
			content: "dsn: ${CH_DSN}",
			env:     map[string]string{dsnVar: "clickhouse://u:p@host:9000"},
			want:    "dsn: clickhouse://u:p@host:9000",
		},
		{
			name:    "missing variable becomes empty",
			content: "dsn: ${CH_DSN}",
			want:    "dsn: ",
		},
		{
			name:    "default used when variable unset",
			content: dsnWithDefault,
			want:    "dsn: clickhouse://localhost:9000",
		},
		{
			name:    "default used when variable empty",
			content: dsnWithDefault,
			env:     map[string]string{dsnVar: ""},
			want:    "dsn: clickhouse://localhost:9000",
		},
		{
			name:    "default ignored when variable set",
			content: dsnWithDefault,
			env:     map[string]string{dsnVar: "clickhouse://remote:9000"},
			want:    "dsn: clickhouse://remote:9000",
		},
		{
			name:    "comment lines are skipped",
			content: "  # dsn: ${CH_DSN}",
			env:     map[string]string{dsnVar: "clickhouse://remote:9000"},
			want:    "  # dsn: ${CH_DSN}",
		},
		{
			name:    "multiple variables on one line",
			content: "dsn: clickhouse://${CH_USER}:${CH_PASS}@host:9000",
			env:     map[string]string{"CH_USER": "reader", "CH_PASS": "secret"},
			want:    "dsn: clickhouse://reader:secret@host:9000",
		},
		{
			name:    "dollar without braces is left unchanged",
			content: "dsn: clickhouse://u:pa$word@host:9000",
			env:     map[string]string{"word": "replaced"},
			want:    "dsn: clickhouse://u:pa$word@host:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			assert.Equal(t, tt.want, substituteEnvVars(tt.content))
		})
	}
}

func TestLoadSubstitutesEnvVars(t *testing.T) {
	t.Setenv(dsnVar, "clickhouse://reader:secret@host:9000")

	cfg := loadFromString(t, `
clickhouse:
  dsn: "${CH_DSN}"
  database: "${CH_DATABASE:-mainnet}"
`)

	assert.Equal(t, "clickhouse://reader:secret@host:9000", cfg.ClickHouse.DSN)
	assert.Equal(t, "mainnet", cfg.ClickHouse.Database)
}

func TestLoadWithoutEnvVarsIsUnchanged(t *testing.T) {
	t.Setenv("word", "replaced")

	cfg := loadFromString(t, `
clickhouse:
  dsn: "clickhouse://u:pa$word@host:9000"
  database: "mainnet"
`)

	assert.Equal(t, "clickhouse://u:pa$word@host:9000", cfg.ClickHouse.DSN)
	assert.Equal(t, "mainnet", cfg.ClickHouse.Database)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoadMissingFile(t *testing.T) {
	viper.Reset()

	_, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	require.Error(t, err)
}

func loadFromString(t *testing.T, content string) *Config {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	viper.Reset()
	t.Cleanup(viper.Reset)

	cfg, err := Load(path)
	require.NoError(t, err)

	return cfg
}
