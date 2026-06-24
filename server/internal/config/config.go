package config

import (
	"os"
	"strings"

	"github.com/kai-zer-ru/buhgalter/internal/locale"
)

type Config struct {
	Addr          string
	DBPath        string
	DataDir       string
	LogDir        string
	CORSOrigins   []string
	Version       string
	InstallMethod string
	BuildCommit   string
	BuildTime     string
	StaticEmbed   bool
	LocalesDir    string
}

func Load(version, installMethod, buildCommit, buildTime string) Config {
	cfg := Config{
		Addr:          envOr("BUHGALTER_ADDR", ":8765"),
		DBPath:        envOr("BUHGALTER_DB_PATH", "./data/buhgalter.db"),
		DataDir:       envOr("BUHGALTER_DATA_DIR", "./data"),
		LogDir:        envOr("BUHGALTER_LOG_DIR", "./logs"),
		Version:       version,
		InstallMethod: installMethod,
		BuildCommit:   buildCommit,
		BuildTime:     buildTime,
		StaticEmbed:   envOr("BUHGALTER_STATIC_EMBED", "true") != "false",
		LocalesDir:    locale.ResolveDir(envOr("BUHGALTER_LOCALES_DIR", "")),
	}

	// "*" reflects request Origin (required with session cookies; literal "*" is forbidden with credentials).
	origins := envOr("BUHGALTER_CORS_ORIGINS", "*")
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, o)
		}
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
