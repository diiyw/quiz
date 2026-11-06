package integration

import (
	"bytes"
	"encoding/xml"
	"path/filepath"
)

func xmlText(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func macFiles(c Config) []File {
	script := "exec " + shellQuote(c.Executable) + " --extract -- \"$@\""
	base := filepath.Join(c.Home, "Library", "Services", "Quiz Extract.workflow", "Contents")
	info := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>io.github.diiyw.quiz.extract</string>
<key>CFBundleName</key><string>Quiz Extract</string>
<key>NSServices</key><array><dict>
<key>NSMenuItem</key><dict><key>default</key><string>用 Quiz 解压到同名文件夹</string></dict>
<key>NSMessage</key><string>runWorkflowAsService</string>
<key>NSRequiredContext</key><dict><key>NSApplicationIdentifier</key><string>com.apple.finder</string></dict>
<key>NSSendFileTypes</key><array><string>public.archive</string><string>org.gnu.gnu-zip-archive</string><string>public.data</string></array>
<key>NSSendTypes</key><array><string>NSFilenamesPboardType</string></array>
</dict></array></dict></plist>
`
	workflow := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>AMApplicationBuild</key><string>523</string><key>AMApplicationVersion</key><string>2.10</string>
<key>AMDocumentVersion</key><string>2</string>
<key>actions</key><array><dict><key>action</key><dict>
<key>AMAccepts</key><dict><key>Container</key><string>List</string><key>Optional</key><true/><key>Types</key><array><string>com.apple.cocoa.string</string></array></dict>
<key>AMActionVersion</key><string>2.0.3</string>
<key>AMParameterProperties</key><dict><key>COMMAND_STRING</key><dict/><key>CheckedForUserDefaultShell</key><dict/><key>inputMethod</key><dict/><key>shell</key><dict/></dict>
<key>AMProvides</key><dict><key>Container</key><string>List</string><key>Types</key><array><string>com.apple.cocoa.string</string></array></dict>
<key>ActionBundlePath</key><string>/System/Library/Automator/Run Shell Script.action</string>
<key>ActionName</key><string>Run Shell Script</string>
<key>ActionParameters</key><dict><key>COMMAND_STRING</key><string>` + xmlText(script) + `</string><key>CheckedForUserDefaultShell</key><true/><key>inputMethod</key><integer>1</integer><key>shell</key><string>/bin/sh</string></dict>
<key>BundleIdentifier</key><string>com.apple.RunShellScript</string>
<key>Class Name</key><string>RunShellScriptAction</string>
<key>InputUUID</key><string>74751E0C-7BAA-4CAA-8502-BF8C3B660601</string>
<key>OutputUUID</key><string>74751E0C-7BAA-4CAA-8502-BF8C3B660602</string>
<key>UUID</key><string>74751E0C-7BAA-4CAA-8502-BF8C3B660603</string>
<key>isViewVisible</key><integer>1</integer>
</dict><key>isViewVisible</key><integer>1</integer></dict></array>
<key>connectors</key><dict/>
<key>workflowMetaData</key><dict>
<key>applicationBundleID</key><string>com.apple.finder</string><key>applicationBundleIDsByPath</key><dict/>
<key>applicationPath</key><string>/System/Library/CoreServices/Finder.app</string>
<key>applicationPaths</key><array><string>/System/Library/CoreServices/Finder.app</string></array>
<key>inputTypeIdentifier</key><string>com.apple.Automator.fileSystemObject</string>
<key>outputTypeIdentifier</key><string>com.apple.Automator.nothing</string>
<key>presentationMode</key><integer>15</integer>
<key>serviceInputTypeIdentifier</key><string>com.apple.Automator.fileSystemObject</string>
<key>serviceOutputTypeIdentifier</key><string>com.apple.Automator.nothing</string>
<key>serviceProcessesInput</key><integer>0</integer>
<key>workflowTypeIdentifier</key><string>com.apple.Automator.servicesMenu</string>
</dict></dict></plist>
`
	return []File{{filepath.Join(base, "Info.plist"), info, 0644}, {filepath.Join(base, "document.wflow"), workflow, 0644}}
}
