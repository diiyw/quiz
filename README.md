# Quiz

使用 [Onix](https://github.com/onix-engine/onix) 构建的跨平台桌面解压工具。当前只提供解压功能，文件始终留在本机。

选择或拖入压缩包立即解压，也可安装系统右键菜单后直接操作。支持批量队列、进度、取消、打开结果目录；7Z / RAR 需要密码时自动弹窗输入。

```text
下载/照片.zip       → 下载/照片/
下载/backup.tar.gz  → 下载/backup/
已有“照片”文件夹   → 下载/照片 (2)/
```

## 下载与运行

从仓库 [Releases](https://github.com/diiyw/quiz/releases) 下载对应平台包并解开。

| 平台 | 产物 | 启动方式 |
| --- | --- | --- |
| Windows x64 | `quiz-v*-windows-amd64.zip` | 双击 `quiz.exe` |
| macOS Intel | `quiz-v*-darwin-amd64.tar.gz` | 打开 `Quiz.app` |
| macOS Apple Silicon | `quiz-v*-darwin-arm64.tar.gz` | 打开 `Quiz.app` |
| Linux x64 / ARM64 | `quiz-v*-linux-*.tar.gz` | 运行 `./quiz` |

所有包同时提供无图形依赖的 `quiz-cli`（Windows 为 `quiz-cli.exe`）以及依赖许可文件。
macOS 最低 13.0；应用采用 ad-hoc 签名，尚未 Developer ID 签名或公证，首次运行可能需要在“隐私与安全性”中允许打开。
Linux 桌面程序需要 X11 / XWayland、OpenGL 和系统字体；文件选择使用桌面的 XDG Portal。x64 包基于 Ubuntu 22.04，ARM64 包基于 Ubuntu 24.04。

## 右键菜单

打开 Quiz，点击“安装右键菜单”，或运行桌面可执行文件的 `--install-menu` 命令。安装会将程序复制到用户目录，因此安装后移动原下载文件不影响菜单。更新版本后再次安装即可。

| 平台 | 菜单位置 |
| --- | --- |
| Windows | 资源管理器 → 用 Quiz 解压到同名文件夹；Windows 11 通常在“显示更多选项” |
| macOS | Finder → 快速操作 / 服务 → 用 Quiz 解压到同名文件夹 |
| KDE Dolphin | 压缩文件的右键菜单 |
| Nemo | 压缩文件的右键菜单 |
| GNOME Nautilus | 安装 `python3-nautilus` 后有直接菜单；也提供“脚本”子菜单入口 |

安装只修改当前用户的菜单配置，不接管默认打开方式。文件管理器可能需要重新打开；macOS 可能需要在系统设置的扩展中启用该快速操作。Finder 服务可显示在普通文件上，但 Quiz 只接受支持的压缩扩展名。其他 Linux 文件管理器可使用“打开方式 → Quiz”，暂未提供专属菜单插件。

“移除菜单”或 `--uninstall-menu` 会删除本应用的注册项，保留复制到用户目录的程序，避免删除正在运行的文件。其位置为 Linux `$XDG_DATA_HOME/quiz/bin/quiz`（默认 `~/.local/share`）、macOS `~/Library/Application Support/Quiz/quiz`、Windows `%LOCALAPPDATA%\Quiz\quiz.exe`。

## 格式和边界

支持 ZIP、7Z、RAR、TAR、GZIP、BZIP2、XZ、Zstandard、LZ4、LZIP，以及对应的 TAR 组合和 `tgz` / `tbz2` / `txz` 等常见别名。解码器来自 [mholt/archives](https://github.com/mholt/archives)，无需另装 7-Zip。

- ZIP 加密、分卷压缩包和自解压程序暂不支持；7Z / RAR 密码支持取决于底层解码器。
- 默认每个压缩包最多解压 100 GiB、100 万条目；命令行可调整上限。
- 不覆盖、不合并已有目录。损坏、取消、越界路径或不支持的条目导致整个任务失败，并清理本次目录；已有内容保持不变。
- 符号链接、硬链接、设备文件及 Windows 不合法文件名会被拒绝；常规文件保留修改时间和可执行位，不恢复所有权或特权权限。
- 各桌面平台的菜单体验需在对应系统上验收；自动构建会验证二进制解压行为，不替代 Finder / Explorer 的交互验收。

## 命令行

```sh
./quiz-cli -- archive.zip archive.7z
./quiz --headless -- backup.tar.gz
./quiz --install-menu
./quiz --uninstall-menu
./quiz-cli --max-bytes 214748364800 --max-entries 2000000 -- backup.tar.zst
./quiz-cli --password-env QUIZ_PASSWORD -- private.7z
./quiz-cli --version
```

`--password-env` 从指定环境变量取密码，不在进程参数中传明文。桌面程序遇到加密包时弹窗输入密码，仅用于当前压缩包；密码无效时可重试，取消则跳过当前包并继续队列。命令行单个任务失败仍继续其他任务，但返回非零状态；Ctrl+C 取消并清理当前任务。

## 本地构建

需要 Go 1.27.3 和 C/C++ 工具链。本项目依赖相邻目录的 Onix，`go.mod` 使用 `replace github.com/diiyw/onix => ../onix`。

```sh
git clone https://github.com/onix-engine/onix.git ../onix
git -C ../onix checkout "$(cat .onix-revision)"
go mod download
go build -o dist/quiz .
go build -o dist/quiz-cli ./cmd/quiz-cli
go test -race ./internal/extract ./internal/integration ./internal/cli
```

Linux 开发库：`libx11-dev libxext-dev libxrender-dev libxi-dev libwayland-dev libxkbcommon-dev libgl1-mesa-dev pkg-config`。Windows 使用 MinGW-w64，macOS 使用 Xcode Command Line Tools。只构建 CLI 时可设置 `CGO_ENABLED=0`。

生成本机发布包：

```sh
python3 scripts/build.py --version v0.1.0 --os linux --arch amd64
python3 scripts/smoke.py build/quiz-v0.1.0-linux-amd64/quiz
```

`--os` / `--arch` 可选择其他目标，但交叉构建桌面程序需要相应的 CGO 工具链；发布工作流使用各平台原生 runner。

## GitHub Actions 发布

`.github/workflows/test.yml` 在 push / PR 时运行核心测试。`build.yml` 支持手动构建，并在推送 `v*` tag 时自动构建五种平台包、执行测试、生成 SHA256 校验文件，最后使用 `gh release` 发布。

```sh
git tag -a v0.1.0 -m "Quiz v0.1.0"
git push origin v0.1.0
```

工作流从公共仓库获取 `.onix-revision` 固定的 Onix 提交。Release 仅在所有构建成功后发布，使用 GitHub 自动提供的 `GITHUB_TOKEN`，无需另设 PAT。手动构建保留 Actions artifacts，不创建 Release。
