package integration

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/diiyw/quiz/assets"
	"github.com/diiyw/quiz/internal/extract"
)

const mimeTypes = "application/zip;application/x-7z-compressed;application/vnd.rar;application/x-rar;application/x-rar-compressed;application/x-tar;application/gzip;application/x-gzip;application/x-compressed-tar;application/x-bzip;application/x-bzip2;application/x-bzip-compressed-tar;application/x-xz;application/x-xz-compressed-tar;application/zstd;application/x-zstd-compressed-tar;application/x-lz4;application/x-lz4-compressed-tar;application/x-lzip;application/x-lzip-compressed-tar;"

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// Desktop entries have both Exec quoting and key-value escaping layers.
func desktopQuote(s string) string {
	s = strings.ReplaceAll(s, "%", "%%")
	s = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "`", "\\`", "$", `\$`).Replace(s)
	return `"` + strings.NewReplacer(`\`, `\\`, "\n", `\n`, "\r", `\r`).Replace(s) + `"`
}

func linuxFiles(c Config) []File {
	command := desktopQuote(c.Executable) + " --extract -- %F"
	service := "[Desktop Entry]\nType=Service\nMimeType=" + mimeTypes + "\nActions=QuizExtract;\nX-KDE-Priority=TopLevel\nServiceTypes=KonqPopupMenu/Plugin\n\n[Desktop Action QuizExtract]\nName=Extract to folder with Quiz\nName[zh_CN]=用 Quiz 解压到同名文件夹\nExec=" + command + "\n"
	extensions := make([]string, len(extract.Extensions))
	for i, ext := range extract.Extensions {
		extensions[i] = strings.TrimPrefix(ext, ".")
	}
	nemo := "[Nemo Action]\nName=Extract to folder with Quiz\nName[zh_CN]=用 Quiz 解压到同名文件夹\nComment=Extract beside the archive\nExec=" + command + "\nSelection=notnone\nExtensions=" + strings.Join(extensions, ";") + ";\nQuote=double\n"
	executableJSON, _ := json.Marshal(c.Executable)
	extensionsJSON, _ := json.Marshal(extract.Extensions)
	nautilus := `# Installed by Quiz. Requires the distribution's python3-nautilus package.
import gi
import subprocess
from gi.repository import GObject, Nautilus
EXECUTABLE = ` + string(executableJSON) + `
EXTENSIONS = tuple(` + string(extensionsJSON) + `)

class QuizMenu(GObject.GObject, Nautilus.MenuProvider):
    def get_file_items(self, *args):
        files = args[-1]
        paths = [f.get_location().get_path() for f in files]
        if not paths or any(not p or not p.lower().endswith(EXTENSIONS) for p in paths):
            return []
        item = Nautilus.MenuItem(name="Quiz::Extract", label="用 Quiz 解压到同名文件夹", tip="在压缩包旁创建同名文件夹")
        item.connect("activate", lambda *_: subprocess.Popen([EXECUTABLE, "--extract", "--", *paths]))
        return [item]
`
	// Nautilus passes locally selected paths as positional arguments. Keeping argv
	// intact supports spaces, quotes, newlines and leading hyphens.
	script := "#!/bin/sh\n[ \"$#\" -gt 0 ] || exit 0\nexec " + shellQuote(c.Executable) + " --extract -- \"$@\"\n"
	desktop := "[Desktop Entry]\nType=Application\nName=Quiz\nComment=Extract archives into their own folders\nIcon=io.github.diiyw.quiz\nExec=" + desktopQuote(c.Executable) + " --extract -- %F\nTerminal=false\nCategories=Utility;Archiving;\nMimeType=" + mimeTypes + "\n"
	return []File{
		{filepath.Join(c.DataDir, "icons", "hicolor", "scalable", "apps", "io.github.diiyw.quiz.svg"), assets.IconSVG, 0644},
		{filepath.Join(c.DataDir, "kio", "servicemenus", "quiz-extract.desktop"), service, 0755},
		{filepath.Join(c.DataDir, "kservices5", "ServiceMenus", "quiz-extract.desktop"), service, 0755},
		{filepath.Join(c.DataDir, "nemo", "actions", "quiz-extract.nemo_action"), nemo, 0644},
		{filepath.Join(c.DataDir, "nautilus-python", "extensions", "quiz_extract.py"), nautilus, 0644},
		{filepath.Join(c.DataDir, "nautilus", "scripts", "Quiz 解压到同名文件夹"), script, 0755},
		{filepath.Join(c.DataDir, "applications", "io.github.diiyw.quiz.desktop"), desktop, 0644},
	}
}
