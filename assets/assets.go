// Package assets contains the application icon and a subset of the UI font.
package assets

import _ "embed"

//go:embed icon.png
var IconPNG []byte

//go:embed icon.svg
var IconSVG string

//go:embed QuizUI.ttf
var UIFont []byte

//go:embed QuizUI-OFL.txt
var UIFontLicense string
