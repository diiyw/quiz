//go:build !windows

package integration

import "fmt"

func installWindows(string) error { return fmt.Errorf("需要在 Windows 上注册菜单") }
func uninstallWindows() error     { return fmt.Errorf("需要在 Windows 上移除菜单") }
