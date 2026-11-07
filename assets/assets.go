// Package assets contains the application icon and its original vector source.
package assets

import _ "embed"

//go:embed icon.png
var IconPNG []byte

//go:embed icon.svg
var IconSVG string
