package desktop

import (
	"sync"

	"github.com/diiyw/quiz/assets"
	"github.com/onix-engine/onix/text"
)

// Load only when opening the desktop UI. The subset covers static interface
// text; Onix still uses system fallback for other characters in user filenames.
var interfaceFont = sync.OnceValues(func() (*text.Font, error) {
	return text.LoadFont(assets.UIFont)
})
