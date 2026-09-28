package spotlight

import _ "embed"

//go:embed spotlight.js
var JSContent []byte

//go:embed spotlight.html
var HTMLContent []byte
