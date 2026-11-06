// Package integration installs user-scoped file-manager actions.
package integration

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type File struct {
	Path, Content string
	Mode          os.FileMode
}
type Config struct{ Platform, Home, DataDir, Executable string }

func currentConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	c := Config{Platform: runtime.GOOS, Home: home}
	switch runtime.GOOS {
	case "linux":
		c.DataDir = os.Getenv("XDG_DATA_HOME")
		if !filepath.IsAbs(c.DataDir) {
			c.DataDir = filepath.Join(home, ".local", "share")
		}
		c.Executable = filepath.Join(c.DataDir, "quiz", "bin", "quiz")
	case "darwin":
		c.Executable = filepath.Join(home, "Library", "Application Support", "Quiz", "quiz")
	case "windows":
		c.DataDir = os.Getenv("LOCALAPPDATA")
		if c.DataDir == "" {
			return c, fmt.Errorf("LOCALAPPDATA 未设置")
		}
		c.Executable = filepath.Join(c.DataDir, "Quiz", "quiz.exe")
	default:
		return c, fmt.Errorf("暂不支持此平台：%s", runtime.GOOS)
	}
	return c, nil
}

func Install() (string, error) {
	c, err := currentConfig()
	if err != nil {
		return "", err
	}
	source, err := os.Executable()
	if err != nil {
		return "", err
	}
	if err = copyExecutable(source, c.Executable); err != nil {
		return "", err
	}
	if c.Platform == "windows" {
		return "已注册资源管理器右键菜单；Windows 11 请查看“显示更多选项”。", installWindows(c.Executable)
	}
	files, err := Files(c)
	if err != nil {
		return "", err
	}
	for _, f := range files {
		if err = os.MkdirAll(filepath.Dir(f.Path), 0755); err != nil {
			return "", err
		}
		if err = os.WriteFile(f.Path, []byte(f.Content), f.Mode); err != nil {
			return "", err
		}
		if err = os.Chmod(f.Path, f.Mode); err != nil {
			return "", err
		}
	}
	if c.Platform == "darwin" {
		_ = exec.Command("/System/Library/CoreServices/pbs", "-flush").Run()
		_ = exec.Command("/System/Library/CoreServices/pbs", "-update").Run()
		return "已安装 Finder 快速操作；若未出现，请在系统设置的扩展中启用 Quiz。", nil
	}
	_ = exec.Command("update-desktop-database", filepath.Join(c.DataDir, "applications")).Run()
	return "已安装 Dolphin、Nemo 与 Nautilus 菜单。重新打开文件管理器后生效；Nautilus 可从“脚本”中使用。", nil
}

func Uninstall() (string, error) {
	c, err := currentConfig()
	if err != nil {
		return "", err
	}
	if c.Platform == "windows" {
		return "已移除 Quiz 右键菜单。", uninstallWindows()
	}
	files, err := Files(c)
	if err != nil {
		return "", err
	}
	var errs []error
	for _, f := range files {
		if err := os.Remove(f.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	if c.Platform == "darwin" {
		// Only remove now-empty directories belonging to our service.
		contents := filepath.Dir(files[0].Path)
		_ = os.Remove(contents)
		_ = os.Remove(filepath.Dir(contents))
		_ = exec.Command("/System/Library/CoreServices/pbs", "-update").Run()
	}
	return "已移除 Quiz 右键菜单，重启文件管理器后生效。", errors.Join(errs...)
}

func copyExecutable(source, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	if dstInfo, e := os.Stat(destination); e == nil && os.SameFile(info, dstInfo) {
		return nil
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(destination), ".quiz-install-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, copyErr := io.Copy(tmp, src)
	err = errors.Join(copyErr, tmp.Chmod(0755), tmp.Close())
	if err != nil {
		return err
	}
	return os.Rename(tmp.Name(), destination)
}

func Files(c Config) ([]File, error) {
	if c.Executable == "" || !filepath.IsAbs(c.Executable) {
		return nil, fmt.Errorf("菜单需要绝对可执行文件路径")
	}
	switch c.Platform {
	case "linux":
		return linuxFiles(c), nil
	case "darwin":
		return macFiles(c), nil
	default:
		return nil, fmt.Errorf("平台 %s 使用原生注册表或不受支持", c.Platform)
	}
}
