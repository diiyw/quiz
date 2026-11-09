// Package cli handles arguments for desktop and headless builds of Quiz.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/diiyw/quiz/assets"
	"github.com/diiyw/quiz/internal/extract"
	"github.com/diiyw/quiz/internal/integration"
)

type Request struct {
	Files   []string
	Options extract.Options
	Extract bool
}
type Desktop func(Request) error

// Version is set by the release build.
var Version = "dev"

// ThirdPartyNotices is populated with dependency license texts by the release build.
var ThirdPartyNotices = "See THIRD_PARTY_NOTICES.md and the license files in the Go module dependencies.\n\n" + assets.UIFontLicense

func Run(args []string, desktop Desktop, out, stderr io.Writer) int {
	flags := flag.NewFlagSet("quiz", flag.ContinueOnError)
	flags.SetOutput(stderr)
	headless := flags.Bool("headless", desktop == nil, "不打开窗口，在终端解压")
	extractNow := flags.Bool("extract", false, "立即解压指定文件，成功后退出（右键菜单使用）")
	install := flags.Bool("install-menu", false, "安装当前用户的右键菜单")
	uninstall := flags.Bool("uninstall-menu", false, "移除当前用户的右键菜单")
	version := flags.Bool("version", false, "显示版本")
	licenses := flags.Bool("licenses", false, "显示第三方许可")
	passwordEnv := flags.String("password-env", "", "从指定环境变量读取 7Z / RAR 密码")
	maxBytes := flags.Int64("max-bytes", 100<<30, "每个压缩包的最大解压字节数")
	maxEntries := flags.Int("max-entries", 1_000_000, "每个压缩包的最大条目数")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Quiz · 只做解压\n\n用法：quiz [选项] [--] 文件.zip 文件.7z ...\n目标：压缩包旁的同名文件夹；已存在则自动编号。\n支持：ZIP / 7Z / RAR / TAR / GZ / BZ2 / XZ / ZSTD / LZ4 / LZIP 及 TAR 组合。")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *install && *uninstall {
		fmt.Fprintln(stderr, "安装和移除菜单不能同时指定")
		return 2
	}
	if *version {
		fmt.Fprintln(out, "Quiz", Version)
		return 0
	}
	if *licenses {
		fmt.Fprintln(out, ThirdPartyNotices)
		return 0
	}
	if *install || *uninstall {
		if len(flags.Args()) != 0 {
			fmt.Fprintln(stderr, "菜单操作不能同时指定压缩文件")
			return 2
		}
		var message string
		var err error
		if *install {
			message, err = integration.Install()
		} else {
			message, err = integration.Uninstall()
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(out, message)
		return 0
	}
	if *maxBytes <= 0 || *maxEntries <= 0 {
		fmt.Fprintln(stderr, "解压大小和条目上限必须大于 0")
		return 2
	}
	request := Request{Files: flags.Args(), Options: extract.Options{MaxBytes: *maxBytes, MaxEntries: *maxEntries}, Extract: *extractNow}
	if *passwordEnv != "" {
		var exists bool
		request.Options.Password, exists = os.LookupEnv(*passwordEnv)
		if !exists {
			fmt.Fprintln(stderr, "指定的密码环境变量未设置")
			return 2
		}
	}
	if desktop != nil && !*headless {
		if err := desktop(request); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if len(request.Files) == 0 {
		flags.Usage()
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	code := 0
	for _, file := range request.Files {
		if ctx.Err() != nil {
			return 130
		}
		result, err := extract.Extract(ctx, file, request.Options)
		if err != nil {
			fmt.Fprintln(stderr, err)
			code = 1
			continue
		}
		fmt.Fprintf(out, "%s → %s（%d 个文件，%d 字节）\n", result.Source, result.Destination, result.Files, result.Bytes)
	}
	if ctx.Err() != nil {
		return 130
	}
	return code
}
