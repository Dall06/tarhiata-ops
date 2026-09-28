package alerts

import _ "embed"

//go:embed alerts.js
var JSContent []byte

//go:embed alerts.html
var HTMLContent []byte
