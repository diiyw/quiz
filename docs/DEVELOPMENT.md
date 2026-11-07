# 开发指南

本文说明源码构建、图标维护和发布流程。使用方法请看 [README](../README.md)。

## 本地构建

需要 Go 1.27.3 和 C/C++ 工具链。Onix 使用远程 `0.1.1` 标签对应的固定版本，由 Go 自动下载，无需本地引擎目录。模块路径为 `github.com/onix-engine/onix`；无 `v` 前缀的标签由 Go 记录为对应提交的伪版本。

```sh
go mod download
go build -o dist/quiz .
go build -o dist/quiz-cli ./cmd/quiz-cli
go test -race ./internal/desktop ./internal/extract ./internal/integration ./internal/cli
```

Linux 开发库：`libx11-dev libxext-dev libxrender-dev libxi-dev libwayland-dev libxkbcommon-dev libgl1-mesa-dev pkg-config`。Windows 使用 MinGW-w64，macOS 使用 Xcode Command Line Tools。只构建 CLI 时可设置 `CGO_ENABLED=0`。

生成本机发布包：

```sh
python3 scripts/build.py --version v0.1.3 --os linux --arch amd64
python3 scripts/check-package.py --os linux
```

产物写入 `dist/`：Linux 为 `.tar.gz`、macOS 为 `.dmg`、Windows 为 `.msi`，各附带 `.sha256`。`check-package.py` 要求 `dist/` 中只有一份当前平台的包。

`--os` / `--arch` 可选择其他目标，但交叉构建桌面程序需要相应的 CGO 工具链；DMG 和 MSI 使用各平台原生工具，完整打包请在对应系统执行。

- macOS：使用系统自带的 `hdiutil` 生成 DMG、`codesign` 签名，并在 App Bundle 中配置 ICNS 图标。
- Windows：需要 MinGW-w64 的 `windres` 和 [.NET SDK](https://dotnet.microsoft.com/download)，执行 `dotnet tool install --global wix --version 6.0.2` 安装 WiX。若只有较新的 .NET Runtime，设置 `DOTNET_ROLL_FORWARD=Major`。MSI 包含开始菜单快捷方式，支持升级和卸载。

## 应用图标

原始矢量图标位于 [`assets/icon.svg`](../assets/icon.svg)，同时提供 PNG、Windows ICO 和 macOS ICNS。桌面窗口、Windows 可执行文件与安装器、macOS App Bundle，以及 Linux 菜单安装的桌面入口使用同一图标。

修改 SVG 后可重新生成平台资源（正常构建直接使用已提交的资源，无需安装图标工具）：

```sh
# 需要 Node.js/npm；脚本会临时下载固定版本的 resvg 来渲染 SVG
python3 -m pip install Pillow
python3 scripts/generate-icons.py
```

## GitHub Actions 发布

`.github/workflows/test.yml` 在 push / PR 时运行核心测试。`build.yml` 支持手动构建，并在推送 `v*` tag 时自动构建五种平台包、执行测试、生成 SHA256 校验文件，最后使用 `gh release` 发布。工作流还会解开 Linux tar.gz、挂载 DMG，以及安装和卸载 MSI，验证包内程序的解压行为。

```sh
git tag -a v0.1.3 -m "Quiz v0.1.3"
git push origin v0.1.3
```

工作流通过 Go 模块下载 `go.mod` 锁定的远程 Onix 版本。Release 仅在所有构建成功后发布，使用 GitHub 自动提供的 `GITHUB_TOKEN`，无需另设 PAT。手动构建保留 Actions artifacts，不创建 Release。
