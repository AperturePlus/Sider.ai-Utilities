package ui

import _ "embed"

//go:embed chat.html
var ChatHTML []byte

//go:embed styles.css
var StylesCSS []byte

//go:embed app.js
var AppJS []byte
