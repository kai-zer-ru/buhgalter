package httpserver_test

import "github.com/kai-zer-ru/buhgalter/internal/config"

func testConfig(dataDir string) config.Config {
	return config.Config{
		Version:      "test",
		StaticEmbed:  false,
		DataDir:      dataDir,
		AllowedHosts: []string{"127.0.0.1", "localhost", "::1"},
	}
}
