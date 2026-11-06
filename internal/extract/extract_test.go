package extract

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mholt/archives"
)

func zipData(t *testing.T, names ...string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for _, name := range names {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = io.WriteString(f, "hello 世界"); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func tarData(t *testing.T, headers ...*tar.Header) []byte {
	t.Helper()
	var b bytes.Buffer
	w := tar.NewWriter(&b)
	for _, h := range headers {
		if err := w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Typeflag == tar.TypeReg {
			if _, err := io.WriteString(w, strings.Repeat("x", int(h.Size))); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func put(t *testing.T, directory, name string, data []byte) string {
	t.Helper()
	filename := filepath.Join(directory, name)
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
	return filename
}
func contents(t *testing.T, filename string) string {
	t.Helper()
	b, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestStem(t *testing.T) {
	for in, want := range map[string]string{"backup.tar.gz": "backup", "A.TAR.BZ2": "A", "v1.2.tgz": "v1.2", ".zip": "archive", "中文 name.7z": "中文 name", "note.txt.gz": "note.txt", "zip.zip.zip": "zip.zip"} {
		got, err := Stem(in)
		if err != nil || got != want {
			t.Errorf("Stem(%q)=%q,%v", in, got, err)
		}
	}
	if Supported("document.pdf") {
		t.Fatal("accepted pdf")
	}
}

func TestZIPNestedUnicodeAndCollision(t *testing.T) {
	dir := t.TempDir()
	source := put(t, dir, "旅行照片.zip", zipData(t, "nested/中文.txt"))
	first, err := Extract(context.Background(), source, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Files != 1 || first.Destination != filepath.Join(dir, "旅行照片") {
		t.Fatalf("%+v", first)
	}
	if got := contents(t, filepath.Join(first.Destination, "nested", "中文.txt")); got != "hello 世界" {
		t.Fatal(got)
	}
	second, err := Extract(context.Background(), source, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Destination != first.Destination+" (2)" {
		t.Fatal(second)
	}
}

func TestCompressionFormats(t *testing.T) {
	payload := tarData(t, &tar.Header{Name: "folder/hello.txt", Mode: 0755, Size: 5, Typeflag: tar.TypeReg})
	formats := []struct {
		format archives.Compressor
		suffix string
	}{
		{suffix: "gz", format: archives.Gz{}}, {suffix: "bz2", format: archives.Bz2{}}, {suffix: "xz", format: archives.Xz{}}, {suffix: "zst", format: archives.Zstd{}}, {suffix: "lz4", format: archives.Lz4{}}, {suffix: "lz", format: archives.Lzip{}},
	}
	for _, format := range formats {
		t.Run(format.suffix, func(t *testing.T) {
			for _, archive := range []bool{false, true} {
				var data bytes.Buffer
				writer, err := format.format.OpenWriter(&data)
				if err != nil {
					t.Fatal(err)
				}
				suffix, content := "."+format.suffix, []byte("single stream")
				if archive {
					suffix, content = ".tar"+suffix, payload
				}
				if _, err = writer.Write(content); err != nil {
					t.Fatal(err)
				}
				if err = writer.Close(); err != nil {
					t.Fatal(err)
				}
				source := put(t, t.TempDir(), "sample"+suffix, data.Bytes())
				result, err := Extract(context.Background(), source, Options{})
				if err != nil {
					t.Fatal(err)
				}
				name, want := "sample", "single stream"
				if archive {
					name, want = "folder/hello.txt", "xxxxx"
				}
				if got := contents(t, filepath.Join(result.Destination, name)); got != want {
					t.Errorf("got %q", got)
				}
			}
		})
	}
}

func TestSevenZip(t *testing.T) {
	for _, name := range []string{"sample.7z", "encrypted.7z"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", name))
			if err != nil {
				t.Fatal(err)
			}
			source := put(t, t.TempDir(), name, data)
			opts := Options{}
			if name == "encrypted.7z" {
				if _, err := Extract(context.Background(), source, opts); err == nil {
					t.Fatal("accepted missing password")
				}
				opts.Password = "quiz-test"
			}
			result, err := Extract(context.Background(), source, opts)
			if err != nil {
				t.Fatal(err)
			}
			if got := contents(t, filepath.Join(result.Destination, "hello.txt")); got != "Hello from Quiz!\n" {
				t.Fatal(got)
			}
		})
	}
}

// Build an independent RAR4 stored-file fixture (no external executable required).
func TestRAR(t *testing.T) {
	payload := []byte("RAR content")
	name := []byte("hello.txt")
	var archive bytes.Buffer
	archive.WriteString("Rar!\x1a\x07\x00")
	header := func(kind byte, flags uint16, rest []byte) {
		b := []byte{kind, byte(flags), byte(flags >> 8), 0, 0}
		binary.LittleEndian.PutUint16(b[3:], uint16(7+len(rest)))
		b = append(b, rest...)
		crc := crc32.ChecksumIEEE(b)
		archive.Write([]byte{byte(crc), byte(crc >> 8)})
		archive.Write(b)
	}
	header(0x73, 0, make([]byte, 6))
	file := make([]byte, 25)
	binary.LittleEndian.PutUint32(file, uint32(len(payload)))
	binary.LittleEndian.PutUint32(file[4:], uint32(len(payload)))
	file[8] = 3
	binary.LittleEndian.PutUint32(file[9:], crc32.ChecksumIEEE(payload))
	file[17], file[18] = 20, 0x30
	binary.LittleEndian.PutUint16(file[19:], uint16(len(name)))
	binary.LittleEndian.PutUint32(file[21:], 0100644)
	header(0x74, 0x8000, append(file, name...))
	archive.Write(payload)
	header(0x7b, 0, nil)
	source := put(t, t.TempDir(), "stored.rar", archive.Bytes())
	result, err := Extract(context.Background(), source, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := contents(t, filepath.Join(result.Destination, "hello.txt")); got != string(payload) {
		t.Fatal(got)
	}
}

func TestUnsafePathsCleanup(t *testing.T) {
	names := []string{"../escape.txt", "nested/../../escape", "/absolute", `C:\escape`, `..\escape`, `\\server\share`, "folder/CON.txt", "file:stream", "name. ", "a\x00b"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			source := put(t, dir, "unsafe.zip", zipData(t, "first.txt", name))
			if _, err := Extract(context.Background(), source, Options{}); err == nil {
				t.Fatal("accepted unsafe entry")
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 1 {
				t.Fatalf("partial output left: %v", entries)
			}
		})
	}
}

func TestLinksRejected(t *testing.T) {
	for _, kind := range []byte{tar.TypeSymlink, tar.TypeLink, tar.TypeFifo} {
		h := &tar.Header{Name: "link", Linkname: "../outside", Mode: 0777, Typeflag: kind}
		source := put(t, t.TempDir(), "links.tar", tarData(t, h))
		if _, err := Extract(context.Background(), source, Options{}); err == nil {
			t.Fatalf("accepted type %v", kind)
		}
	}
}

func TestFailureCleanup(t *testing.T) {
	for _, name := range []string{"limit", "entries", "duplicate", "cancel", "corrupt"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			data := zipData(t, "a", "b")
			opts := Options{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch name {
			case "limit":
				opts.MaxBytes = 3
			case "entries":
				opts.MaxEntries = 1
			case "duplicate":
				data = zipData(t, "a", "a")
			case "cancel":
				opts.OnProgress = func(p Progress) {
					if p.Bytes > 0 {
						cancel()
					}
				}
			case "corrupt":
				data = []byte("PK invalid archive")
			}
			source := put(t, dir, "bad.zip", data)
			if _, err := Extract(ctx, source, opts); err == nil {
				t.Fatal("expected error")
			}
			entries, _ := os.ReadDir(dir)
			if len(entries) != 1 {
				t.Fatalf("partial output left: %v", entries)
			}
		})
	}
}

func TestPreCancelledAndConcurrent(t *testing.T) {
	source := put(t, t.TempDir(), "files.zip", zipData(t, "a"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Extract(ctx, source, Options{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan string, 4)
	for range 4 {
		wg.Go(func() {
			result, err := Extract(context.Background(), source, Options{})
			if err != nil {
				t.Error(err)
				return
			}
			results <- result.Destination
		})
	}
	wg.Wait()
	close(results)
	seen := map[string]bool{}
	for result := range results {
		if seen[result] {
			t.Fatal("shared destination")
		}
		seen[result] = true
	}
	if len(seen) != 4 {
		t.Fatal(seen)
	}
}

func TestExistingSymlinkIsNeverFollowed(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "files")); err != nil {
		t.Skip(err)
	}
	source := put(t, dir, "files.zip", zipData(t, "a"))
	result, err := Extract(context.Background(), source, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Destination != filepath.Join(dir, "files (2)") {
		t.Fatal(result)
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Fatal("wrote outside extraction directory")
	}
}
