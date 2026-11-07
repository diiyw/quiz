package desktop

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/bodgit/sevenzip"
	"github.com/diiyw/quiz/internal/extract"
	"github.com/nwaples/rardecode/v2"
	"github.com/onix-engine/onix/ui"
	htmlui "github.com/onix-engine/onix/ui/html"
)

type passwordRequest struct {
	ctx    context.Context
	source string
	retry  bool
	reply  chan string
}

func needsPassword(err error, prompted bool) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var encrypted *sevenzip.ReadError
	if errors.As(err, &encrypted) && encrypted.Encrypted {
		return true
	}
	if errors.Is(err, rardecode.ErrArchiveEncrypted) || errors.Is(err, rardecode.ErrArchivedFileEncrypted) || errors.Is(err, rardecode.ErrBadPassword) {
		return true
	}
	// Older RAR encryption has no password verifier. Only interpret decoder
	// failures as a password hint after this archive requested a password.
	return prompted && (errors.Is(err, rardecode.ErrBadHeaderCRC) || errors.Is(err, rardecode.ErrBadFileChecksum) || errors.Is(err, rardecode.ErrCorruptBlockHeader) || errors.Is(err, rardecode.ErrCorruptFileHeader) || errors.Is(err, rardecode.ErrDecoderOutOfData))
}

func (c *controller) extractWithPassword(ctx context.Context, source string, opts extract.Options) (extract.Progress, error) {
	prompted := false
	for {
		result, err := extract.Extract(ctx, source, opts)
		if ctx.Err() != nil {
			return result, errors.Join(err, ctx.Err())
		}
		if !needsPassword(err, prompted) {
			return result, err
		}
		request := passwordRequest{ctx: ctx, source: source, retry: opts.Password != "", reply: make(chan string, 1)}
		select {
		case c.passwordRequests <- request:
			c.app.RequestRender()
		case <-ctx.Done():
			return result, ctx.Err()
		}
		select {
		case opts.Password = <-request.reply:
			prompted = true
		case <-ctx.Done():
			return result, ctx.Err()
		}
	}
}

func (c *controller) showPassword(request passwordRequest) {
	c.pendingPassword = &request
	c.password = ""
	c.nodes["password"].SetAttribute("value", "")
	c.nodes["password-file"].SetText(filepath.Base(request.source))
	message := "请输入压缩包密码"
	if request.retry {
		message = "密码无效，请重试"
	}
	c.nodes["password-message"].SetText(message)
	c.nodes["password-dialog"].SetHidden(false)
	c.nodes["state"].SetText("等待密码")
	c.nodes["destination"].SetText("")
	for _, id := range []string{"choose", "cancel", "open", "install", "uninstall", "password-confirm"} {
		c.disable(id, true)
	}
	if err := ui.SyncHTML(c.app.World); err != nil {
		if c.cancel != nil {
			c.cancel()
		}
		c.hidePassword()
		c.addHistory("无法显示密码窗口：" + err.Error())
		return
	}
	for entity, control := range c.app.World.Query[htmlui.Control]().All {
		if control.Ref().ID == "password" {
			ui.FocusControl(c.app.World, entity)
			break
		}
	}
	c.app.RequestRender()
}

func (c *controller) hidePassword() {
	c.pendingPassword = nil
	c.password = ""
	ui.FocusControl(c.app.World, 0)
	c.nodes["password"].SetAttribute("value", "")
	c.nodes["password-dialog"].SetHidden(true)
	c.disable("choose", false)
	c.disable("cancel", !c.busy)
	c.disable("open", c.lastDestination == "")
	c.disable("install", c.menuBusy)
	c.disable("uninstall", c.menuBusy)
}
