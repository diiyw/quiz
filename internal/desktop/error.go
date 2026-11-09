package desktop

import (
	"strings"

	"github.com/onix-engine/onix/ui"
	htmlui "github.com/onix-engine/onix/ui/html"
)

func (c *controller) showError(message string) {
	c.errorVisible = true
	if message != "" {
		c.errorMessages = append(c.errorMessages, message)
	}
	c.nodes["error-message"].SetText(strings.Join(c.errorMessages, "\n\n"))
	c.nodes["error-dialog"].SetHidden(false)
	for _, id := range []string{"choose", "cancel", "open", "install", "uninstall"} {
		c.disable(id, true)
	}
	if err := ui.SyncHTML(c.app.World); err == nil {
		for entity, control := range c.app.World.Query[htmlui.Control]().All {
			if control.Ref().ID == "error-confirm" {
				ui.FocusControl(c.app.World, entity)
				break
			}
		}
	}
	c.app.RequestRender()
}

func (c *controller) hideError() {
	c.errorVisible = false
	c.errorMessages = nil
	ui.FocusControl(c.app.World, 0)
	c.nodes["error-dialog"].SetHidden(true)
	c.nodes["error-message"].SetText("")
	c.disable("choose", false)
	c.disable("cancel", !c.busy)
	c.disable("open", c.lastDestination == "")
	c.disable("install", c.menuBusy)
	c.disable("uninstall", c.menuBusy)
}
