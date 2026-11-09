package desktop

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"
	"unicode"
)

func TestInterfaceFontCoversStaticText(t *testing.T) {
	font, err := interfaceFont()
	if err != nil {
		t.Fatal(err)
	}
	// Check the face directly: ResolveGlyph would silently fall back to system
	// fonts and reintroduce the scan this subset is meant to avoid at startup.
	check := func(source, value string) {
		t.Helper()
		for _, character := range value {
			if unicode.IsGraphic(character) && font.GlyphIndex(character) == 0 {
				t.Errorf("%s: UI font missing %q (U+%04X); run scripts/generate-font.py", source, character, character)
			}
		}
	}
	check("view.html", view)
	check("password mask and ellipsis", "•…")
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		if matched, _ := filepath.Match("*_test.go", path); matched {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.BasicLit)
			if ok && literal.Kind == token.STRING {
				value, err := strconv.Unquote(literal.Value)
				if err != nil {
					t.Fatal(err)
				}
				check(path, value)
			}
			return true
		})
	}
}
