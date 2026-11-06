package desktop

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/diiyw/onix"
	"github.com/diiyw/onix/ecs"
	htmlui "github.com/diiyw/onix/ui/html"
	"github.com/diiyw/onix/window"
	_ "github.com/diiyw/onix/window/filedialog"
	"github.com/diiyw/quiz/internal/cli"
	"github.com/diiyw/quiz/internal/extract"
	"github.com/diiyw/quiz/internal/integration"
)

//go:embed view.html
var view string

type outcome struct {
	err      error
	progress extract.Progress
}
type menuOutcome struct {
	err     error
	message string
}

type controller struct {
	app             *onix.App
	nodes           map[string]htmlui.Node
	cancel          context.CancelFunc
	finished        chan outcome
	menuFinished    chan menuOutcome
	password        string
	lastDestination string
	queue           []string
	history         []string
	progress        extract.Progress
	options         extract.Options
	workers         sync.WaitGroup
	events          ecs.MessageReader[htmlui.Event]
	progressMu      sync.Mutex
	progressChanged bool
	busy            bool
	menuBusy        bool
	closing         bool
}

func Run(request cli.Request) error {
	app := onix.NewApp()
	app.SetRenderMode(onix.RenderOnDemand)
	app.SetIdleEnabled(true)
	if err := app.InstallSystems[onix.UIHTML](); err != nil {
		return err
	}
	app.SetWindowSize(860, 730)
	app.SetWindowTitle("Quiz · 解压工具")
	surface := htmlui.New(view)
	spawner := app.World.NewSpawner1[htmlui.Surface]()
	owner := spawner.Spawn(surface)
	c := &controller{app: app, nodes: make(map[string]htmlui.Node), finished: make(chan outcome, 1), menuFinished: make(chan menuOutcome, 1), options: request.Options, password: request.Options.Password}
	defer func() {
		if c.cancel != nil {
			c.cancel()
		}
		c.workers.Wait()
	}()
	for _, id := range []string{"choose", "password", "state", "filename", "detail", "destination", "cancel", "open", "history", "install", "uninstall", "menu-status"} {
		node, ok := htmlui.NodeByID(app.World, owner, id)
		if !ok {
			app.Destroy()
			return fmt.Errorf("缺少界面节点：%s", id)
		}
		c.nodes[id] = node
	}
	app.OnFilesDropped(func(_, _ float64, files []string) {
		if !c.closing {
			c.enqueue(files)
		}
	})
	app.OnCloseRequested(func() bool {
		if !c.busy && !c.menuBusy {
			return true
		}
		c.closing = true
		c.queue = nil
		if c.cancel != nil {
			c.cancel()
		}
		c.nodes["state"].SetText("正在清理，请稍候")
		app.RequestRender()
		return false
	})
	if err := app.AddSystem(onix.StageUpdate, ecs.SystemConfig{Name: "quiz_events", System: func(*ecs.SystemContext) error { c.update(); return nil }}); err != nil {
		app.Destroy()
		return err
	}
	c.enqueue(request.Files)
	return app.Run()
}

func (c *controller) enqueue(files []string) {
	for _, file := range files {
		if !extract.Supported(file) {
			c.addHistory("不支持：" + filepath.Base(file))
			continue
		}
		c.queue = append(c.queue, file)
	}
	if !c.busy {
		c.next()
	}
	c.app.RequestRender()
}

func (c *controller) next() {
	if len(c.queue) == 0 {
		c.busy = false
		c.disable("cancel", true)
		return
	}
	source := c.queue[0]
	c.queue = c.queue[1:]
	c.busy = true
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.disable("cancel", false)
	c.nodes["filename"].SetText(filepath.Base(source))
	c.nodes["state"].SetAttribute("class", "state")
	c.nodes["state"].SetText(fmt.Sprintf("正在解压 · 等待 %d 项", len(c.queue)))
	c.nodes["detail"].SetText("正在读取压缩包…")
	c.nodes["destination"].SetText("")
	opts := c.options
	opts.Password = c.password
	opts.OnProgress = func(p extract.Progress) {
		c.progressMu.Lock()
		c.progress = p
		c.progressChanged = true
		c.progressMu.Unlock()
		c.app.RequestRender()
	}
	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		result, err := extract.Extract(ctx, source, opts)
		cancel()
		c.finished <- outcome{progress: result, err: err}
		c.app.RequestRender()
	}()
}

func (c *controller) update() {
	for event := range c.events.All(c.app.World) {
		if event.ID == "password" && event.Kind == htmlui.EventInput {
			c.password = event.Value
		}
		if event.Kind == htmlui.EventClick && !c.closing {
			c.click(event.ID)
		}
	}
	c.progressMu.Lock()
	p, changed := c.progress, c.progressChanged
	c.progressChanged = false
	c.progressMu.Unlock()
	if changed {
		c.nodes["detail"].SetText(fmt.Sprintf("%d 个文件 · %s  /  %s", p.Files, byteSize(p.Bytes), p.Current))
		c.nodes["destination"].SetText(p.Destination)
	}
	select {
	case result := <-c.finished:
		c.busy = false
		c.cancel = nil
		if result.err != nil {
			label := "解压失败"
			if errors.Is(result.err, context.Canceled) {
				label = "已取消"
			}
			c.nodes["state"].SetAttribute("class", "error")
			c.nodes["state"].SetText(label)
			c.nodes["detail"].SetText(result.err.Error())
			c.addHistory(label + " · " + result.err.Error())
		} else {
			c.lastDestination = result.progress.Destination
			c.disable("open", false)
			c.nodes["state"].SetText("解压完成")
			c.nodes["detail"].SetText(fmt.Sprintf("已解压 %d 个文件 · %s", result.progress.Files, byteSize(result.progress.Bytes)))
			c.nodes["destination"].SetText(c.lastDestination)
			c.addHistory("已完成 · " + c.lastDestination)
		}
		c.next()
		c.app.RequestRender()
	default:
	}
	select {
	case result := <-c.menuFinished:
		c.menuBusy = false
		c.disable("install", false)
		c.disable("uninstall", false)
		if result.err != nil {
			c.nodes["menu-status"].SetText("操作失败：" + result.err.Error())
		} else {
			c.nodes["menu-status"].SetText(result.message)
		}
		c.app.RequestRender()
	default:
	}
	if c.closing && !c.busy && !c.menuBusy {
		c.app.RequestClose()
	}
}

func (c *controller) click(id string) {
	switch id {
	case "choose":
		extensions := make([]string, len(extract.Extensions))
		for i, ext := range extract.Extensions {
			extensions[i] = strings.TrimPrefix(ext, ".")
		}
		files, err := c.app.OpenFiles(window.FileDialogOptions{Title: "选择压缩文件", Filters: []window.FileDialogFilter{{Name: "压缩文件", Extensions: extensions}}})
		if err != nil {
			c.addHistory("文件选择失败：" + err.Error())
			return
		}
		c.enqueue(files)
	case "cancel":
		c.queue = nil
		if c.cancel != nil {
			c.cancel()
			c.nodes["state"].SetText("正在取消并清理…")
		}
	case "open":
		if c.lastDestination != "" {
			if err := openFolder(c.lastDestination); err != nil {
				c.addHistory("打开目录失败：" + err.Error())
			}
		}
	case "install", "uninstall":
		if c.menuBusy {
			return
		}
		c.menuBusy = true
		c.disable("install", true)
		c.disable("uninstall", true)
		c.nodes["menu-status"].SetText("正在更新右键菜单…")
		c.workers.Add(1)
		go func() {
			defer c.workers.Done()
			var message string
			var err error
			if id == "install" {
				message, err = integration.Install()
			} else {
				message, err = integration.Uninstall()
			}
			c.menuFinished <- menuOutcome{message: message, err: err}
			c.app.RequestRender()
		}()
	}
	c.app.RequestRender()
}

func (c *controller) disable(id string, disabled bool) {
	if disabled {
		c.nodes[id].SetAttribute("disabled", "")
	} else {
		c.nodes[id].RemoveAttribute("disabled")
	}
}
func (c *controller) addHistory(line string) {
	c.history = append(c.history, line)
	if len(c.history) > 30 {
		c.history = c.history[len(c.history)-30:]
	}
	c.nodes["history"].SetText(strings.Join(c.history, "\n"))
}
func byteSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
}
func openFolder(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", path)
	case "darwin":
		cmd = exec.Command("open", "--", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
