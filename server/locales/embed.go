package locales

import "embed"

// Files contains built-in locale catalogs packaged into the binary.
//
//go:embed ru.json en.json
var Files embed.FS

