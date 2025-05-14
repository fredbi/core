package fixtures

import (
	"embed"
)

// embedded test files

//go:embed  specs/* schemas/*
var EmbeddedFixtures embed.FS
