package audit

import _ "embed"

//go:embed audit.js
var JSContent []byte

//go:embed audit.html
var HTMLContent []byte
