<p align="center">
  <img src="assets/icon.png" width="128" height="128" alt="Quiz 应用图标">
</p>

# Quiz

**把压缩包，变成同名文件夹。**

Quiz 是一款适用于 Windows、macOS 和 Linux 的桌面解压工具。拖入压缩包即可解压，无需另装 7-Zip；文件和密码都留在本机，不会上传。

[下载最新版](https://github.com/diiyw/quiz/releases/latest) · [反馈问题](https://github.com/diiyw/quiz/issues)

- **拖入即解压**：支持同时选择或拖入多个压缩包，依次处理。
- **自动整理**：在压缩包旁创建同名文件夹；已有同名目录时自动编号，避免覆盖。
- **密码解压**：打开加密的 7Z / RAR 时弹窗输入密码。
- **右键解压**：安装右键菜单后，可直接在文件管理器中操作。
- **随时取消**：查看解压进度、取消任务，完成后一键打开文件夹。

## 下载与安装

当前版本：**v0.1.4**。根据你的系统和处理器选择下载：

| 系统 | 下载 | 安装方式 |
| --- | --- | --- |
| Windows 64 位 | [MSI 安装包](https://github.com/diiyw/quiz/releases/download/v0.1.4/quiz-v0.1.4-windows-amd64.msi) | 双击安装，从开始菜单打开 Quiz |
| macOS Apple 芯片（M 系列） | [DMG 安装包](https://github.com/diiyw/quiz/releases/download/v0.1.4/quiz-v0.1.4-darwin-arm64.dmg) | 打开 DMG，将 `Quiz.app` 拖入 Applications（应用程序） |
| macOS Intel 芯片 | [DMG 安装包](https://github.com/diiyw/quiz/releases/download/v0.1.4/quiz-v0.1.4-darwin-amd64.dmg) | 打开 DMG，将 `Quiz.app` 拖入 Applications（应用程序） |
| Linux x64 | [tar.gz 压缩包](https://github.com/diiyw/quiz/releases/download/v0.1.4/quiz-v0.1.4-linux-amd64.tar.gz) | 解压后运行目录中的 `quiz` |
| Linux ARM64 | [tar.gz 压缩包](https://github.com/diiyw/quiz/releases/download/v0.1.4/quiz-v0.1.4-linux-arm64.tar.gz) | 解压后运行目录中的 `quiz` |

Windows 安装只对当前用户生效，无需管理员权限。

安装包只包含桌面应用，不附带独立 CLI、说明文档或校验文件。第三方许可已嵌入应用，可通过 `--licenses` 查看。

macOS 需要 **13.0 或更新版本**。可在“关于本机”中查看芯片类型。应用尚未经过 Apple 公证，首次打开如被系统拦截，请在“系统设置 → 隐私与安全性”中允许打开。

Linux 需要支持 X11 / XWayland 和 OpenGL 的图形桌面。x64 包基于 Ubuntu 22.04，ARM64 包基于 Ubuntu 24.04；其他发行版需具备兼容的系统库。

## 开始解压

1. 打开 Quiz，将压缩包拖入窗口，或点击“选择文件解压”。
2. Quiz 会在压缩包所在目录中创建同名文件夹，并开始解压。
3. 完成后点击“打开文件夹”，查看解压结果。

```text
下载/照片.zip       → 下载/照片/
下载/backup.tar.gz  → 下载/backup/
已有“照片”文件夹   → 下载/照片 (2)/
```

可以一次添加多个压缩包，Quiz 会按顺序处理。点击“取消全部”会停止当前任务并清空等待队列；当前任务已解压的内容会清理，原压缩包和已有文件保持不变。

解压加密的 7Z / RAR 时，输入密码后点击“继续解压”。密码错误可以重试；取消密码输入会跳过该压缩包，继续处理其他任务。

## 在右键菜单中使用

在 Quiz 窗口中点击“安装右键菜单”，然后重新打开文件管理器。选中压缩包后右键，选择“用 Quiz 解压到同名文件夹”。

右键启动时仅显示“解压中”窗口，包含标题、滚动进度条和取消按钮，全部任务成功后自动关闭。点击取消会停止全部任务，清理未完成的内容后关闭窗口。解压失败时会弹窗显示完整错误；批量任务在点击“确定”后继续处理剩余文件，并保留窗口。手动打开 Quiz 进行解压时，完成后窗口会保留。

| 文件管理器 | 菜单位置 |
| --- | --- |
| Windows 资源管理器 | 右键菜单；Windows 11 通常需先点击“显示更多选项” |
| macOS Finder | “快速操作”或“服务”；如未出现，请在系统设置的扩展中启用 Quiz |
| KDE Dolphin / Nemo | 压缩包的右键菜单 |
| GNOME Nautilus | “脚本”子菜单；安装 `python3-nautilus` 后也可使用直接右键菜单 |

右键菜单不会改变压缩文件的默认打开方式。更新 Quiz 后，请再次点击“安装右键菜单”，让菜单使用新版本。无需该功能时，在应用中点击“移除菜单”。

## 支持的格式

ZIP、7Z、RAR、TAR、GZIP、BZIP2、XZ、Zstandard、LZ4、LZIP，以及 `.tar.gz`、`.tar.xz`、`.tar.zst`、`.tgz`、`.tbz2`、`.txz` 等组合和别名。

使用时请留意：

- 暂不支持加密 ZIP、分卷压缩包和自解压程序；部分 7Z / RAR 加密方式也可能无法解压。
- 默认每个压缩包最多解压 **100 GiB、100 万条目**。
- 如果压缩包损坏、含有不安全路径、符号链接或其他不支持的内容，Quiz 会停止该任务并清理本次输出，避免留下不完整的结果。
- Quiz 目前只提供解压功能，不提供创建压缩包的功能。

## 常见问题

### 解压后的文件在哪里？

就在原压缩包旁边的同名文件夹中。解压完成后点击“打开文件夹”即可查看；原压缩包不会被删除。

### 为什么没有覆盖已有文件夹？

Quiz 会自动创建带编号的新文件夹，例如 `照片 (2)`，保护已有内容。

### 如何更新或卸载？

更新时下载并安装新版本即可。使用右键菜单的用户，更新后还需在应用中再次点击“安装右键菜单”。

卸载前先打开 Quiz，点击“移除菜单”。Windows 随后可从系统“已安装的应用”中卸载；macOS 将 Quiz 从“应用程序”移到废纸篓；Linux 删除解压出的程序目录。

菜单安装时会单独保存一份程序，移除菜单后仍会保留。若要彻底清理，可在退出 Quiz 后删除对应文件：

| 系统 | 菜单保存的程序 |
| --- | --- |
| Windows | `%LOCALAPPDATA%\Quiz\quiz.exe` |
| macOS | `~/Library/Application Support/Quiz/quiz` |
| Linux | `~/.local/share/quiz/bin/quiz`；设置过 `XDG_DATA_HOME` 时位于该目录下的 `quiz/bin/quiz` |

### 遇到问题如何反馈？

请在 [Issues](https://github.com/diiyw/quiz/issues) 中说明系统版本、Quiz 版本、压缩格式和错误提示。请勿公开压缩包密码或包含隐私的文件。

## 命令行使用（可选）

桌面应用支持通过 `--headless` 在终端解压。在 Linux 程序所在目录运行：

```sh
./quiz --headless -- archive.zip archive.7z
./quiz --headless --password-env QUIZ_PASSWORD -- private.7z
./quiz --help
```

macOS 中将 `./quiz` 换成 `/Applications/Quiz.app/Contents/MacOS/quiz`；Windows PowerShell 中进入安装目录，将其换成 `.\quiz.exe`。`--password-env` 从指定的环境变量读取密码，使用前需设置该变量。

---

[开发指南](docs/DEVELOPMENT.md) · [第三方许可](THIRD_PARTY_NOTICES.md)
