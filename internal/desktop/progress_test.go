package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/onix-engine/onix/ui"
	htmlui "github.com/onix-engine/onix/ui/html"
	"github.com/onix-engine/onix/window"
)

func extractionController(t *testing.T) *controller {
	t.Helper()
	c := passwordController(t)
	c.app.Context = &window.Context{}
	c.exitOnSuccess = true
	c.configureView()
	for entity := range c.app.World.Query[htmlui.Surface]().All {
		surface := c.app.World.GetMut[htmlui.Surface](entity)
		surface.Width, surface.Height = 460, 220
	}
	return c
}

func assertExtractionBounds(t *testing.T, c *controller, ids ...string) {
	t.Helper()
	if err := ui.SyncHTML(c.app.World); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		box, ok := c.nodes[id].Bounds()
		if !ok || box.Width <= 0 || box.Height <= 0 || box.X < 0 || box.Y < 0 || box.X+box.Width > 460.5 || box.Y+box.Height > 220.5 {
			t.Fatalf("%s outside extraction window: %+v", id, box)
		}
	}
}

func TestExtractionWindowLayout(t *testing.T) {
	c := extractionController(t)
	assertExtractionBounds(t, c, "extraction-title", "extraction-progress", "extraction-cancel")
	for _, id := range []string{"choose", "install", "uninstall", "open"} {
		if box, ok := c.nodes[id].Bounds(); ok && box.Width > 0 && box.Height > 0 {
			t.Fatalf("main window control %s visible during extraction", id)
		}
	}
	c.showError("解压失败")
	assertExtractionBounds(t, c, "error-dialog", "error-message", "error-confirm")
}

func TestExtractionWindowCancelCleansUpAndCloses(t *testing.T) {
	for _, passwordPrompt := range []bool{false, true} {
		name := "progress-button"
		if passwordPrompt {
			name = "password-dialog"
		}
		t.Run(name, func(t *testing.T) {
			c := extractionController(t)
			dir := t.TempDir()
			first := copyArchive(t, dir, "encrypted.7z", "first.7z")
			second := copyArchive(t, dir, "sample.7z", "second.7z")
			c.enqueue([]string{first, second})
			if passwordPrompt {
				pumpPasswordUI(t, c, func() bool { return c.pendingPassword != nil })
				assertExtractionBounds(t, c, "password-dialog", "password", "password-confirm", "password-cancel")
				c.click("password-cancel")
			} else {
				c.click("extraction-cancel")
			}
			pumpPasswordUI(t, c, func() bool { return !c.busy })
			if !c.closing || len(c.queue) != 0 || c.pendingPassword != nil || c.errorVisible {
				t.Fatal("cancel did not stop the batch and close without an error")
			}
			for _, name := range []string{"first", "second"} {
				if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("cancel left output %s: %v", name, err)
				}
			}
		})
	}
}
