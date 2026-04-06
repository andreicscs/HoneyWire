package web

import _ "embed"

//go:embed templates/index.html
var IndexHTML []byte

//go:embed static/login.html
var LoginHTML []byte