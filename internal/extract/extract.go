package extract

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mholt/archives"
)

type Progress struct {
	Source, Destination, Current string
	Files                        int
	Bytes                        int64
}

type Options struct {
	OnProgress func(Progress)
	Password   string // Supported by the 7Z and RAR decoders.
	MaxBytes   int64
	MaxEntries int
}

// Extract never merges with an existing directory. A successfully reserved name
// is owned by this invocation; all writes are confined using os.Root.
func Extract(ctx context.Context, source string, opts Options) (result Progress, err error) {
	if err = ctx.Err(); err != nil {
		return
	}
	source, err = filepath.Abs(source)
	if err != nil {
		return
	}
	stem, err := Stem(source)
	if err != nil {
		return result, err
	}
	input, err := os.Open(source)
	if err != nil {
		return result, err
	}
	defer input.Close()
	stat, err := input.Stat()
	if err != nil {
		return result, err
	}
	if !stat.Mode().IsRegular() {
		return result, fmt.Errorf("请选择压缩文件：%s", source)
	}
	format, stream, err := archives.Identify(ctx, filepath.Base(source), input)
	if err != nil {
		return result, fmt.Errorf("无法识别压缩文件：%w", err)
	}
	switch f := format.(type) {
	case archives.SevenZip:
		f.Password = opts.Password
		format = f
	case archives.Rar:
		f.Password = opts.Password
		format = f
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 100 << 30
	}
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = 1_000_000
	}
	destination, err := reserveDirectory(filepath.Join(filepath.Dir(source), stem))
	if err != nil {
		return result, err
	}
	result = Progress{Source: source, Destination: destination}
	defer func() {
		if err != nil {
			if cleanupErr := os.RemoveAll(destination); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("清理未完成目录：%w", cleanupErr))
			}
		}
	}()
	root, err := os.OpenRoot(destination)
	if err != nil {
		return result, err
	}
	defer root.Close()
	worker := writer{ctx: ctx, root: root, opts: opts, progress: &result, buffer: make([]byte, 128*1024)}
	worker.report(true)
	if archive, ok := format.(archives.Extractor); ok {
		err = archive.Extract(ctx, stream, worker.entry)
	} else if compressor, ok := format.(archives.Decompressor); ok {
		var reader io.ReadCloser
		reader, err = compressor.OpenReader(stream)
		if err == nil {
			// A single compressed stream uses the source basename, never its untrusted header name.
			err = worker.file(stem, reader, 0644, stat.ModTime())
			err = errors.Join(err, reader.Close())
		}
	} else {
		err = fmt.Errorf("此格式不支持解压")
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = root.Chmod(".", 0755)
	}
	if err != nil {
		return result, fmt.Errorf("解压 %s：%w", filepath.Base(source), err)
	}
	result.Current = ""
	worker.report(true)
	return result, nil
}

func reserveDirectory(base string) (string, error) {
	for i := 1; i <= 10000; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s (%d)", base, i)
		}
		if err := os.Mkdir(candidate, 0700); err == nil {
			return candidate, nil
		} else if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("同名解压目录过多：%s", base)
}

type writer struct {
	lastReport time.Time
	ctx        context.Context
	root       *os.Root
	progress   *Progress
	buffer     []byte
	opts       Options
	entries    int
}

func (w *writer) entry(ctx context.Context, f archives.FileInfo) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	w.entries++
	if w.entries > w.opts.MaxEntries {
		return fmt.Errorf("超出文件数量上限 %d", w.opts.MaxEntries)
	}
	name, err := safeName(f.NameInArchive)
	if err != nil {
		return err
	}
	if f.LinkTarget != "" || f.Mode()&(fs.ModeSymlink|fs.ModeDevice|fs.ModeNamedPipe|fs.ModeSocket|fs.ModeIrregular) != 0 {
		return fmt.Errorf("不支持链接或特殊文件：%s", f.NameInArchive)
	}
	if f.IsDir() {
		return w.root.MkdirAll(name, 0755)
	}
	if !f.Mode().IsRegular() {
		return fmt.Errorf("不支持的文件类型：%s", name)
	}
	if name == "." {
		return fmt.Errorf("无效的文件名")
	}
	if f.Size() > w.opts.MaxBytes-w.progress.Bytes {
		return fmt.Errorf("超出解压大小上限 %d 字节", w.opts.MaxBytes)
	}
	reader, err := f.Open()
	if err != nil {
		return err
	}
	err = w.file(name, reader, f.Mode(), f.ModTime())
	return errors.Join(err, reader.Close())
}

func (w *writer) file(name string, reader io.Reader, mode fs.FileMode, modified time.Time) error {
	if err := w.root.MkdirAll(path.Dir(name), 0755); err != nil {
		return err
	}
	output, err := w.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	w.progress.Current = name
	w.report(true)
	_, copyErr := io.CopyBuffer(progressWriter{w: w, out: output}, contextReader{w.ctx, reader}, w.buffer)
	err = errors.Join(copyErr, output.Close())
	if err != nil {
		return err
	}
	// Keep executable bits, but never restore setuid, ownership or world-write bits.
	if err = w.root.Chmod(name, 0644|(mode.Perm()&0111)); err != nil {
		return err
	}
	if !modified.IsZero() {
		if err = w.root.Chtimes(name, modified, modified); err != nil {
			return err
		}
	}
	w.progress.Files++
	w.report(false)
	return nil
}

func (w *writer) report(force bool) {
	if w.opts.OnProgress != nil && (force || time.Since(w.lastReport) >= 100*time.Millisecond) {
		w.lastReport = time.Now()
		w.opts.OnProgress(*w.progress)
	}
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

type progressWriter struct {
	w   *writer
	out io.Writer
}

func (p progressWriter) Write(data []byte) (int, error) {
	if err := p.w.ctx.Err(); err != nil {
		return 0, err
	}
	if int64(len(data)) > p.w.opts.MaxBytes-p.w.progress.Bytes {
		return 0, fmt.Errorf("超出解压大小上限 %d 字节", p.w.opts.MaxBytes)
	}
	n, err := p.out.Write(data)
	p.w.progress.Bytes += int64(n)
	p.w.report(false)
	return n, err
}

// Reject Windows and POSIX escape paths on every host, so results are portable.
func safeName(name string) (string, error) {
	normalized := strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(normalized, "/") || strings.ContainsAny(normalized, ":\x00") {
		return "", fmt.Errorf("不安全的压缩路径：%q", name)
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return "", fmt.Errorf("不安全的压缩路径：%q", name)
		}
		if segment == "." || segment == "" {
			continue
		}
		if strings.TrimRight(segment, ". ") != segment || strings.ContainsAny(segment, "<>\"|?*\r\n") {
			return "", fmt.Errorf("不支持的文件名：%q", name)
		}
		base := strings.ToUpper(strings.SplitN(segment, ".", 2)[0])
		reserved := base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || base == "CONIN$" || base == "CONOUT$"
		if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9' {
			reserved = true
		}
		if reserved {
			return "", fmt.Errorf("不支持的设备文件名：%q", name)
		}
	}
	return path.Clean(normalized), nil
}
