package uilocales

import "embed"

// Files contains Android/web UI string catalogs for remote i18n sync.
//
//go:embed ru.json en.json
var Files embed.FS
