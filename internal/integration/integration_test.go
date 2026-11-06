package integration

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGeneratedFiles(t *testing.T) {
	for _, platform := range []string{"linux", "darwin"} {
		t.Run(platform, func(t *testing.T) {
			dir := t.TempDir()
			files, err := Files(Config{Platform: platform, Home: dir, DataDir: dir, Executable: filepath.Join(dir, "Quiz space", "quiz")})
			if err != nil {
				t.Fatal(err)
			}
			if len(files) == 0 {
				t.Fatal("no integration files")
			}
			for _, file := range files {
				if err := os.MkdirAll(filepath.Dir(file.Path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(file.Path, []byte(file.Content), file.Mode); err != nil {
					t.Fatal(err)
				}
				if strings.HasSuffix(file.Path, ".plist") || strings.HasSuffix(file.Path, ".wflow") {
					decoder := xml.NewDecoder(strings.NewReader(file.Content))
					for {
						_, err := decoder.Token()
						if err == io.EOF {
							break
						}
						if err != nil {
							t.Fatal(err)
						}
					}
				}
				if strings.HasSuffix(file.Path, ".desktop") {
					if validator, err := exec.LookPath("desktop-file-validate"); err == nil {
						// Type=Service is a KDE extension rather than a freedesktop application.
						if strings.Contains(file.Content, "Type=Application") {
							if out, err := exec.Command(validator, file.Path).CombinedOutput(); err != nil {
								t.Fatalf("%s: %v", out, err)
							}
						}
					}
				}
				if strings.HasSuffix(file.Path, ".py") {
					if python, err := exec.LookPath("python3"); err == nil {
						if out, err := exec.Command(python, "-m", "py_compile", file.Path).CombinedOutput(); err != nil {
							t.Fatalf("%s: %v", out, err)
						}
					}
				}
			}
		})
	}
}

func TestNautilusScriptPreservesArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX shell test")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable")
	}
	dir := t.TempDir()
	executable := filepath.Join(dir, "quiz ' $`percent%")
	capture := filepath.Join(dir, "arguments.json")
	script := "#!" + python + "\nimport json,sys\nwith open(" + strconvPython(capture) + ",'w') as f: json.dump(sys.argv[1:],f)\n"
	if err := os.WriteFile(executable, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	files, _ := Files(Config{Platform: "linux", Home: dir, DataDir: dir, Executable: executable})
	for _, file := range files {
		if strings.HasPrefix(file.Content, "#!/bin/sh") {
			launcher := filepath.Join(dir, "launch.sh")
			if err := os.WriteFile(launcher, []byte(file.Content), 0755); err != nil {
				t.Fatal(err)
			}
			args := []string{"--extract", "--", filepath.Join(dir, "space ' and $().zip"), filepath.Join(dir, "line\nbreak.7z"), "-file.rar"}
			if out, err := exec.Command("sh", append([]string{launcher}, args[2:]...)...).CombinedOutput(); err != nil {
				t.Fatalf("%s: %v", out, err)
			}
			data, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			want, _ := json.Marshal(args)
			var got []string
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			normalized, _ := json.Marshal(got)
			if !bytes.Equal(normalized, want) {
				t.Fatalf("got %s want %s", normalized, want)
			}
		}
	}
}
func strconvPython(s string) string { b, _ := json.Marshal(s); return string(b) }

func TestCopyExecutableAndSelfInstall(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	destination := filepath.Join(dir, "new", "quiz")
	if err := os.WriteFile(source, []byte("binary contents"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := copyExecutable(source, destination); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "binary contents" {
		t.Fatalf("%s %v", data, err)
	}
	if err := copyExecutable(destination, destination); err != nil {
		t.Fatal(err)
	}
}
