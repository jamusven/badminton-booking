package static

import (
	"embed"
)

//go:embed templates/* css/*
var FS embed.FS

//go:embed img/favicon.ico
var FaviconBytes []byte
