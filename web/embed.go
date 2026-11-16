// Package web embeds the built walkthrough UI.
package web

import "embed"

// Dist is the Vite build output. A placeholder index.html keeps the Go
// binary buildable before the frontend has been built.
//
//go:embed all:dist
var Dist embed.FS
