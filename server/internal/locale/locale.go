package locale

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	embeddedlocales "github.com/kai-zer-ru/buhgalter/locales"
)

var (
	catalogs   = map[string]map[string]string{}
	localesDir string
	loadOnce   sync.Once
)

// Dir returns the directory loaded by Load (empty if not loaded).
func Dir() string {
	return localesDir
}

// ResolveDir picks locales directory: explicit env, then ./locales, then ./server/locales.
func ResolveDir(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return filepath.Clean(configured)
	}
	for _, candidate := range []string{"locales", filepath.Join("server", "locales")} {
		if hasLocaleFile(candidate) {
			return candidate
		}
	}
	return filepath.Join("server", "locales")
}

func hasLocaleFile(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "ru.json"))
	return err == nil
}

// Load reads ru.json and en.json from dir.
// If files are absent, it falls back to locales embedded in the binary.
func Load(dir string) error {
	dir = filepath.Clean(dir)
	localesDir = dir
	next := map[string]map[string]string{}
	loaded := 0
	for _, lang := range []string{"ru", "en"} {
		path := filepath.Join(dir, lang+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", path, err)
		}
		if err := decodeCatalog(lang, path, data, next); err != nil {
			return err
		}
		loaded++
	}
	if loaded == 0 {
		for _, lang := range []string{"ru", "en"} {
			path := lang + ".json"
			data, err := fs.ReadFile(embeddedlocales.Files, path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("read embedded %s: %w", path, err)
			}
			if err := decodeCatalog(lang, "embedded:"+path, data, next); err != nil {
				return err
			}
			loaded++
		}
	}
	if loaded == 0 {
		return fmt.Errorf("no locale files in %s and no embedded locales", dir)
	}
	catalogs = next
	return nil
}

func decodeCatalog(lang, source string, data []byte, dst map[string]map[string]string) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse %s: %w", source, err)
	}
	dst[lang] = m
	return nil
}

func Lang(r *http.Request) string {
	accept := strings.ToLower(r.Header.Get("Accept-Language"))
	if strings.HasPrefix(accept, "en") || strings.Contains(accept, ",en") {
		return "en"
	}
	return "ru"
}

func T(r *http.Request, key, fallback string) string {
	ensureLoaded()
	lang := Lang(r)
	if msg, ok := catalogs[lang][key]; ok && msg != "" {
		return msg
	}
	if msg, ok := catalogs["ru"][key]; ok && msg != "" {
		return msg
	}
	return fallback
}

func ensureLoaded() {
	loadOnce.Do(func() {
		_ = Load(ResolveDir(os.Getenv("BUHGALTER_LOCALES_DIR")))
	})
}
