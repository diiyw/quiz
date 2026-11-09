package desktop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onix-engine/onix/scene"
	htmlui "github.com/onix-engine/onix/ui/html"
	"github.com/onix-engine/onix/window"
)

func TestSuccessfulBatchExit(t *testing.T) {
	for _, automatic := range []bool{false, true} {
		name := "manual"
		if automatic {
			name = "context-menu"
		}
		t.Run(name, func(t *testing.T) {
			c := passwordController(t)
			// Supply a window context without opening a native window so the
			// completion path can request closure in this display-free test.
			c.app.Context = &window.Context{}
			c.exitOnSuccess = automatic
			dir := t.TempDir()
			first := copyArchive(t, dir, "sample.7z", "first.7z")
			second := copyArchive(t, dir, "sample.7z", "second.7z")
			c.enqueue([]string{first, second})
			if c.closing {
				t.Fatal("closed before processing the batch")
			}
			pumpPasswordUI(t, c, func() bool { return !c.busy })
			if c.closing != automatic || c.batchFailed || c.errorVisible {
				t.Fatal("unexpected completion state")
			}
			for _, stem := range []string{"first", "second"} {
				if _, err := os.Stat(filepath.Join(dir, stem, "hello.txt")); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestFailedBatchShowsErrorBeforeContinuing(t *testing.T) {
	c := passwordController(t)
	c.exitOnSuccess = true
	dir := t.TempDir()
	source := copyArchive(t, dir, "sample.7z", "plain.7z")
	c.enqueue([]string{filepath.Join(dir, "missing.zip"), source})
	pumpPasswordUI(t, c, func() bool { return c.errorVisible })
	if !c.batchFailed || c.closing || c.busy || len(c.queue) != 1 {
		t.Fatal("failure did not pause the batch and keep the window open")
	}
	if len(c.errorMessages) != 1 || !strings.Contains(c.errorMessages[0], "missing.zip") {
		t.Fatal("error did not identify the failed archive")
	}
	if _, hidden := c.nodes["error-dialog"].Attribute("hidden"); hidden {
		t.Fatal("error dialog is hidden")
	}
	c.click("choose")
	if !c.errorVisible || len(c.queue) != 1 {
		t.Fatal("background action dismissed the error")
	}
	c.click("error-confirm")
	pumpPasswordUI(t, c, func() bool { return !c.busy })
	if c.errorVisible || c.closing || c.lastDestination != filepath.Join(dir, "plain") {
		t.Fatal("acknowledging the error did not finish the remaining archive")
	}
}

func TestLongErrorDialogFitsAndScrolls(t *testing.T) {
	c := passwordController(t)
	message := strings.Repeat("解压 Windows App Beta.zip: handling file 504: Windows App Beta.app/Contents/Frameworks/"+strings.Repeat("a", 120)+"\n", 15)
	c.busy = true
	c.finished <- outcome{err: errors.New(message)}
	pumpPasswordUI(t, c, func() bool { return c.errorVisible })
	if len(c.errorMessages) != 1 || c.errorMessages[0] != message {
		t.Fatal("long error was truncated")
	}
	for _, id := range []string{"error-dialog", "error-message", "error-confirm"} {
		box, ok := c.nodes[id].Bounds()
		if !ok || box.Width <= 0 || box.Height <= 0 || box.X < 0 || box.Y < 0 || box.X+box.Width > 600.5 || box.Y+box.Height > 244.5 {
			t.Fatalf("%s outside compact window: %+v", id, box)
		}
	}
	if _, ok := c.nodes["error-message"].ScrollContainer(); !ok {
		t.Fatal("long error has no scroll container")
	}
	focus, _ := c.app.World.GetResource[*scene.InputFocus]()
	if focus == nil {
		t.Fatal("error dialog has no keyboard focus")
	}
	control, _ := c.app.World.Get[htmlui.Control](focus.Keyboard)
	if control.ID != "error-confirm" {
		t.Fatal("confirmation button is not focused")
	}
}

func TestCancellationDoesNotShowErrorOrAutoExit(t *testing.T) {
	c := passwordController(t)
	c.exitOnSuccess = true
	c.busy = true
	c.finished <- outcome{err: context.Canceled}
	c.update()
	if c.errorVisible || c.closing || !c.batchFailed {
		t.Fatal("cancellation showed an error or closed the window")
	}
}

func TestUnsupportedFilesDoNotAutoExit(t *testing.T) {
	c := passwordController(t)
	c.exitOnSuccess = true
	c.enqueue([]string{"unsupported.txt", "another.png"})
	if !c.errorVisible || c.closing || len(c.errorMessages) != 2 {
		t.Fatal("unsupported files did not show errors")
	}
	c.click("error-confirm")
	if c.closing {
		t.Fatal("failed batch closed after dismissing error")
	}
}
