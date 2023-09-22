package static

import (
	"embed"
)

//go:embed templates/*
var FS embed.FS

//go:embed img/favicon.ico
var FaviconBytes []byte
