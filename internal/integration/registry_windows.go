//go:build windows

package integration

import (
	"errors"
	"strings"
	"syscall"

	"github.com/diiyw/quiz/internal/extract"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func windowsKey(extension string) string {
	return `Software\Classes\SystemFileAssociations\` + extension + `\shell\Quiz.Extract`
}

func installWindows(executable string) error {
	// Windows associations use the final extension (.gz also covers .tar.gz).
	seen := map[string]bool{}
	for _, ext := range extract.Extensions {
		ext = ext[strings.LastIndex(ext, "."):]
		if seen[ext] {
			continue
		}
		seen[ext] = true
		key, _, err := registry.CreateKey(registry.CURRENT_USER, windowsKey(ext), registry.SET_VALUE)
		if err != nil {
			return err
		}
		err = errors.Join(key.SetStringValue("", "用 Quiz 解压到同名文件夹"), key.SetStringValue("Icon", `"`+executable+`",0`), key.SetStringValue("MultiSelectModel", "Document"))
		key.Close()
		if err != nil {
			return err
		}
		command, _, err := registry.CreateKey(registry.CURRENT_USER, windowsKey(ext)+`\command`, registry.SET_VALUE)
		if err != nil {
			return err
		}
		err = command.SetStringValue("", syscall.EscapeArg(executable)+` --extract -- "%1"`)
		command.Close()
		if err != nil {
			return err
		}
	}
	notifyShell()
	return nil
}

func uninstallWindows() error {
	var errs []error
	for _, ext := range extract.Extensions {
		ext = ext[strings.LastIndex(ext, "."):]
		for _, key := range []string{windowsKey(ext) + `\command`, windowsKey(ext)} {
			if err := registry.DeleteKey(registry.CURRENT_USER, key); err != nil && !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
				errs = append(errs, err)
			}
		}
	}
	notifyShell()
	return errors.Join(errs...)
}
func notifyShell() {
	windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify").Call(0x08000000, 0, 0, 0)
}
