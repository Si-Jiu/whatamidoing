// Package web embeds the viewer SPA (index.html, m3.css, app.js).
package web

import "embed"

// Files is the embedded viewer SPA, served from the server's web root.
//
//go:embed *
var Files embed.FS
