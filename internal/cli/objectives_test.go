package cli

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const objectivesRoot = "../../testdata/objectives/console"

var (
	buildOnce sync.Once
	builtBin  string
	buildErr  error
)

// codemdBinary builds the real binary once and returns its path.
func codemdBinary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "codemd-bin-")
		if err != nil {
			buildErr = err
			return
		}
		builtBin = filepath.Join(dir, "codemd")
		cmd := exec.Command("go", "build", "-o", builtBin, "github.com/seercle/codemd/cmd/codemd")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			buildErr = fmt.Errorf("build codemd: %w", err)
		}
	})
	if buildErr != nil {
		t.Fatal(buildErr)
	}
	return builtBin
}

func scenarioEnv(dir, bin string) []string {
	return []string{
		"PATH=" + filepath.Dir(bin) + string(os.PathListSeparator) + os.Getenv("PATH"),
		"HOME=" + dir,
		"LC_ALL=C",
		"LANG=C",
		"TZ=UTC",
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// transcriptCommands returns the raw command text of every "$ " line.
func transcriptCommands(transcript string) []string {
	var cmds []string
	for _, line := range strings.Split(strings.TrimSuffix(transcript, "\n"), "\n") {
		if strings.HasPrefix(line, "$ ") {
			cmds = append(cmds, strings.TrimPrefix(line, "$ "))
		}
	}
	return cmds
}

// buildScript emits a bash script that prints each "$ cmd" prompt, runs the
// command with stderr merged, and records its status so a literal `echo $?`
// observes the preceding command's exit code.
func buildScript(transcript string) string {
	var b strings.Builder
	b.WriteString("set +e\n__st=0\n")
	for _, c := range transcriptCommands(transcript) {
		exec := strings.ReplaceAll(c, "$?", "${__st}")
		fmt.Fprintf(&b, "printf '%%s\\n' %s\n", shellQuote("$ "+c))
		fmt.Fprintf(&b, "{ %s\n} 2>&1\n", exec)
		b.WriteString("__st=$?\n")
	}
	return b.String()
}

// produceTranscript replays the transcript in dir with the binary at bin,
// sourcing setup.sh first when present, and returns the combined output.
func produceTranscript(bin, dir string) (string, error) {
	obj, err := os.ReadFile(filepath.Join(dir, "transcript.console"))
	if err != nil {
		return "", err
	}
	script := ""
	if _, err := os.Stat(filepath.Join(dir, "setup.sh")); err == nil {
		script += "source ./setup.sh 1>&2\n"
	}
	script += buildScript(string(obj))

	// Capture stdout to a real file rather than a bytes.Buffer: os/exec hands an
	// *os.File straight to the child with no copy goroutine, so Run returns even
	// when a background server started by setup.sh holds the descriptor open.
	// setup.sh should still redirect background-server output to avoid races.
	out, err := os.CreateTemp("", "codemd-objective-")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = out.Close()
		_ = os.Remove(out.Name())
	}()

	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = dir
	cmd.Env = scenarioEnv(dir, bin)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	cmd.WaitDelay = 10 * time.Second
	runErr := cmd.Run()
	// Reap any background server started by setup.sh.
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	data, readErr := os.ReadFile(out.Name())
	if runErr != nil {
		return "", fmt.Errorf("replay: %w\n%s", runErr, data)
	}
	if readErr != nil {
		return "", readErr
	}
	return string(data), nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o755)
	})
}

// verifyScenario replays runDir with bin and checks the result against the
// stored transcript in srcDir. In update mode it rewrites srcDir's
// transcript.console with the produced output; otherwise it byte-compares and
// reports a mismatch.
func verifyScenario(bin, srcDir, runDir string, update bool) error {
	got, err := produceTranscript(bin, runDir)
	if err != nil {
		return err
	}
	objPath := filepath.Join(srcDir, "transcript.console")
	if update {
		return os.WriteFile(objPath, []byte(got), 0o644)
	}
	want, err := os.ReadFile(objPath)
	if err != nil {
		return err
	}
	if got != string(want) {
		return fmt.Errorf("objective out of date; run scripts/update-objectives.sh\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	return nil
}

func TestConsoleObjectives(t *testing.T) {
	bin := codemdBinary(t)
	update := os.Getenv("OBJECTIVES_UPDATE") == "1"
	entries, err := os.ReadDir(objectivesRoot)
	if err != nil {
		t.Fatalf("read objectives: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			src := filepath.Join(objectivesRoot, e.Name())
			if _, err := os.Stat(filepath.Join(src, "transcript.console")); err != nil {
				t.Skipf("no transcript.console: %v", err)
			}
			dst := t.TempDir()
			if err := copyDir(src, dst); err != nil {
				t.Fatal(err)
			}
			if err := verifyScenario(bin, src, dst, update); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestObjectiveRunnerDetectsDrift(t *testing.T) {
	bin := codemdBinary(t)
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"transcript.console": "$ codemd --version\nstale output\n",
	})
	if err := verifyScenario(bin, dir, dir, false); err == nil {
		t.Fatal("expected stale transcript to be detected as drift")
	}
	if err := verifyScenario(bin, dir, dir, true); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if err := verifyScenario(bin, dir, dir, false); err != nil {
		t.Fatalf("round-trip after update failed: %v", err)
	}
}
