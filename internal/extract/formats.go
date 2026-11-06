// Package extract expands archives into a new directory beside the source.
package extract

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Extensions is ordered longest first so compound suffixes are removed together.
var Extensions = []string{
	".tar.bz2", ".tar.zstd", ".tar.lz4", ".tar.gz", ".tar.xz", ".tar.zst", ".tar.lz",
	".tbz2", ".tzstd", ".tlz4", ".tgz", ".tbz", ".txz", ".tzst", ".tlz",
	".zip", ".7z", ".rar", ".tar", ".bz2", ".zstd", ".lz4", ".gz", ".xz", ".zst", ".lz",
}

func Stem(filename string) (string, error) {
	name := filepath.Base(filename)
	lower := strings.ToLower(name)
	for _, ext := range Extensions {
		if strings.HasSuffix(lower, ext) {
			stem := strings.TrimSpace(name[:len(name)-len(ext)])
			if stem == "" || stem == "." || stem == ".." {
				stem = "archive"
			}
			return stem, nil
		}
	}
	return "", fmt.Errorf("不支持的压缩格式：%s", name)
}

func Supported(filename string) bool { _, err := Stem(filename); return err == nil }
