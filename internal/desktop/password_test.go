package desktop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bodgit/sevenzip"
	"github.com/diiyw/quiz/internal/extract"
	"github.com/nwaples/rardecode/v2"
	"github.com/onix-engine/onix"
	"github.com/onix-engine/onix/scene"
	"github.com/onix-engine/onix/ui"
	htmlui "github.com/onix-engine/onix/ui/html"
)

func passwordController(t *testing.T) *controller {
	t.Helper()
	app := onix.NewApp()
	app.SetRenderMode(onix.RenderOnDemand)
	app.SetIdleEnabled(true)
	if err := app.InstallSystems[onix.UIHTML](); err != nil {
		t.Fatal(err)
	}
	surface := htmlui.New(view)
	font, err := interfaceFont()
	if err != nil {
		t.Fatal(err)
	}
	surface.Width, surface.Height, surface.Font = 600, 244, font
	spawner := app.World.NewSpawner1[htmlui.Surface]()
	owner := spawner.Spawn(surface)
	c := &controller{app: app, nodes: make(map[string]htmlui.Node), finished: make(chan outcome, 1), menuFinished: make(chan menuOutcome, 1), passwordRequests: make(chan passwordRequest, 1), idle: true}
	for _, id := range []string{"main-view", "extraction-view", "extraction-title", "extraction-progress", "extraction-cancel", "choose", "password", "password-dialog", "password-file", "password-message", "password-confirm", "password-cancel", "error-dialog", "error-message", "error-confirm", "state", "filename", "detail", "destination", "cancel", "open", "history", "install", "uninstall", "menu-status"} {
		node, ok := htmlui.NodeByID(app.World, owner, id)
		if !ok {
			t.Fatalf("missing node %s", id)
		}
		c.nodes[id] = node
	}
	t.Cleanup(func() {
		if c.cancel != nil {
			c.cancel()
		}
		c.workers.Wait()
	})
	return c
}

func pumpPasswordUI(t *testing.T, c *controller, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		c.update()
		if err := ui.SyncSystem("").Run(c.app.World); err != nil {
			t.Fatal(err)
		}
		if ready() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for password flow")
		}
		time.Sleep(time.Millisecond)
	}
}

func copyArchive(t *testing.T, dir, fixture, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "extract", "testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func enterPassword(c *controller, value string) {
	c.app.World.WriteMessage(htmlui.Event{ID: "password", Kind: htmlui.EventInput, Value: value})
	c.update()
	c.app.World.WriteMessage(htmlui.Event{ID: "password", Kind: htmlui.EventSubmit})
	c.update()
}

func assertPasswordFocus(t *testing.T, c *controller) {
	t.Helper()
	focus, _ := c.app.World.GetResource[*scene.InputFocus]()
	if focus == nil {
		t.Fatal("password dialog has no keyboard focus")
	}
	control, _ := c.app.World.Get[htmlui.Control](focus.Keyboard)
	if control.ID != "password" || control.Type != "password" {
		t.Fatal("password field was not focused and masked")
	}
}

func TestPasswordDialogRetryCancelAndQueue(t *testing.T) {
	c := passwordController(t)
	dir := t.TempDir()
	first := copyArchive(t, dir, "encrypted.7z", "first.7z")
	second := copyArchive(t, dir, "encrypted.7z", "second.7z")
	plain := copyArchive(t, dir, "sample.7z", "plain.7z")
	pumpPasswordUI(t, c, func() bool { return true })
	if box, ok := c.nodes["password"].Bounds(); ok && box.Height > 0 {
		t.Fatal("password field visible on main screen")
	}
	c.enqueue([]string{first, second, plain})
	pumpPasswordUI(t, c, func() bool { return c.pendingPassword != nil })
	if c.pendingPassword.source != first || c.pendingPassword.retry || !c.busy || len(c.queue) != 2 {
		t.Fatal("password prompt did not pause the current archive")
	}
	for _, id := range []string{"choose", "cancel", "open", "install", "uninstall", "password-confirm"} {
		if _, disabled := c.nodes[id].Attribute("disabled"); !disabled {
			t.Fatalf("%s should be disabled while prompting", id)
		}
	}
	for _, id := range []string{"password-dialog", "password", "password-confirm", "password-cancel"} {
		box, ok := c.nodes[id].Bounds()
		if !ok || box.Width <= 0 || box.Height <= 0 || box.X < 0 || box.Y < 0 || box.X+box.Width > 600.5 || box.Y+box.Height > 244.5 {
			t.Fatalf("dialog control %s outside compact window: %+v", id, box)
		}
	}
	assertPasswordFocus(t, c)
	c.click("password-confirm")
	if c.pendingPassword == nil {
		t.Fatal("empty password submitted")
	}
	enterPassword(c, "wrong-password")
	pumpPasswordUI(t, c, func() bool { return c.pendingPassword != nil })
	if !c.pendingPassword.retry || c.pendingPassword.source != first || c.password != "" {
		t.Fatal("wrong password did not reopen a cleared dialog")
	}
	assertPasswordFocus(t, c)
	enterPassword(c, "quiz-test")
	pumpPasswordUI(t, c, func() bool { return c.pendingPassword != nil && c.pendingPassword.source == second })
	if c.pendingPassword.retry {
		t.Fatal("password was reused for a different archive")
	}
	assertPasswordFocus(t, c)
	data, err := os.ReadFile(filepath.Join(dir, "first", "hello.txt"))
	if err != nil || string(data) != "Hello from Quiz!\n" {
		t.Fatalf("password retry failed: %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "first (2)")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("password retry left an extra output directory")
	}
	c.click("password-cancel")
	pumpPasswordUI(t, c, func() bool { return !c.busy })
	if c.lastDestination != filepath.Join(dir, "plain") || c.pendingPassword != nil || c.password != "" {
		t.Fatal("cancel did not clear the password and continue the queue")
	}
	if _, err := os.Stat(filepath.Join(dir, "second")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancel left a partial output directory")
	}
	if value, _ := c.nodes["password"].Attribute("value"); value != "" {
		t.Fatal("password remains in the hidden field")
	}
	for range c.app.World.Query[htmlui.ScrollbarPart]().All {
		t.Fatal("compact window has a scrollbar")
	}
}

func TestPasswordWaitStopsOnContextCancellation(t *testing.T) {
	c := passwordController(t)
	source := copyArchive(t, t.TempDir(), "encrypted.7z", "secret.7z")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := c.extractWithPassword(ctx, source, extract.Options{})
		done <- err
	}()
	select {
	case <-c.passwordRequests:
	case <-time.After(5 * time.Second):
		t.Fatal("no password request")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("closing while awaiting password blocked the worker")
	}
}

func TestOrdinaryArchivesDoNotPrompt(t *testing.T) {
	for _, corrupt := range []bool{false, true} {
		t.Run(fmt.Sprint(corrupt), func(t *testing.T) {
			c := passwordController(t)
			source := copyArchive(t, t.TempDir(), "sample.7z", "archive.7z")
			if corrupt {
				if err := os.WriteFile(source, []byte("corrupt archive"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			c.enqueue([]string{source})
			pumpPasswordUI(t, c, func() bool { return !c.busy || c.pendingPassword != nil })
			if c.pendingPassword != nil {
				t.Fatal("ordinary archive opened a password dialog")
			}
			if corrupt == (c.lastDestination != "") {
				t.Fatal("unexpected extraction result")
			}
		})
	}
}

func TestPasswordErrorHints(t *testing.T) {
	for _, err := range []error{rardecode.ErrArchiveEncrypted, rardecode.ErrArchivedFileEncrypted, rardecode.ErrBadPassword, &sevenzip.ReadError{Encrypted: true, Err: errors.New("decode failed")}} {
		if !needsPassword(fmt.Errorf("archive: %w", err), false) {
			t.Errorf("encrypted decoder error did not request a password: %v", err)
		}
	}
	if needsPassword(rardecode.ErrBadFileChecksum, false) || !needsPassword(rardecode.ErrBadFileChecksum, true) {
		t.Fatal("RAR checksum failure must require prior encryption evidence")
	}
	for _, err := range []error{os.ErrPermission, context.Canceled, &sevenzip.ReadError{Err: errors.New("corrupt")}, errors.Join(context.Canceled, rardecode.ErrBadPassword), errors.New("password field is not supported")} {
		if needsPassword(err, true) {
			t.Errorf("unrelated error prompted for a password: %v", err)
		}
	}
}
