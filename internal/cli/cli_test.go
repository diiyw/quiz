package cli

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestArgumentsAndDesktopDispatch(t *testing.T) {
	var output bytes.Buffer
	called := false
	desktop := func(r Request) error {
		called = true
		if len(r.Files) != 1 || r.Files[0] != "-file.zip" || !r.Extract {
			t.Fatal(r)
		}
		return nil
	}
	if code := Run([]string{"--extract", "--", "-file.zip"}, desktop, &output, &output); code != 0 || !called {
		t.Fatal(code, output.String())
	}
	if code := Run([]string{"file.zip"}, func(r Request) error {
		if r.Extract {
			t.Fatal("ordinary desktop launch enabled automatic exit")
		}
		return nil
	}, &output, &output); code != 0 {
		t.Fatal(code, output.String())
	}
	for _, args := range [][]string{{"--help"}, {"--max-bytes", "0"}, {"--install-menu", "--uninstall-menu"}, {"--password-env", "QUIZ_TEST_UNSET"}} {
		called = false
		code := Run(args, desktop, &output, &output)
		if called {
			t.Fatal("unexpected GUI")
		}
		if args[0] != "--help" && code == 0 {
			t.Fatal(args)
		}
	}
}
func TestHeadlessContinuesAfterFailure(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "sample.zip")
	f, err := os.Create(source)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	entry, err := w.Create("hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("hello"))
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	var out, stderr bytes.Buffer
	if code := Run([]string{"--headless", filepath.Join(dir, "missing.zip"), source}, nil, &out, &stderr); code != 1 {
		t.Fatalf("code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "sample", "hello.txt")); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() == 0 || out.Len() == 0 {
		t.Fatal("missing result/error output")
	}
}
