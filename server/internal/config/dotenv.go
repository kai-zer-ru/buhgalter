package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const envFileEnvKey = "BUHGALTER_ENV_FILE"
const defaultEnvFileName = ".env"

// ResolveEnvFilePath returns the path to the project .env file.
func ResolveEnvFilePath() string {
	if p := strings.TrimSpace(os.Getenv(envFileEnvKey)); p != "" {
		return p
	}
	return defaultEnvFileName
}

// LoadDotEnv reads KEY=VALUE pairs from path into the process environment.
// Variables already set in the environment are not overwritten.
func LoadDotEnv(path string) error {
	values, err := readDotEnv(path)
	if err != nil {
		return err
	}
	for key, value := range values {
		if os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
	return nil
}

func readDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	out := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = unquoteEnvValue(strings.TrimSpace(value))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return out, nil
}

func unquoteEnvValue(raw string) string {
	if raw == "" {
		return ""
	}
	if (strings.HasPrefix(raw, `"`) && strings.HasSuffix(raw, `"`)) ||
		(strings.HasPrefix(raw, `'`) && strings.HasSuffix(raw, `'`)) {
		if v, err := strconv.Unquote(raw); err == nil {
			return v
		}
	}
	return raw
}
