package cli

import (
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const consoleRoot = "../../testdata/console"

var (
	buildOnce sync.Once
	builtBin  string
	buildErr  error
)

// TestMain removes the shared binary built by codemdBinary once the package's
// tests finish.
func TestMain(m *testing.M) {
	code := m.Run()
	if builtBin != "" {
		_ = os.RemoveAll(filepath.Dir(builtBin))
	}
	os.Exit(code)
}

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
		exec := c
		// `echo $?` observes the previous command's status; rewrite only that
		// exact command so a literal `$?` elsewhere is left untouched.
		if strings.TrimSpace(c) == "echo $?" {
			exec = `echo "${__st}"`
		}
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
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}

var urlHostPattern = regexp.MustCompile(`(https?)://([^/\s)"']+)`)

// urlHost is an external (non-loopback) URL host and its scheme.
type urlHost struct {
	scheme string
	host   string
}

// externalURLHosts returns the non-loopback hosts referenced by transcript.
func externalURLHosts(transcript string) []urlHost {
	var hosts []urlHost
	for _, m := range urlHostPattern.FindAllStringSubmatch(transcript, -1) {
		host := m[2]
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			continue
		}
		hosts = append(hosts, urlHost{scheme: m[1], host: host})
	}
	return hosts
}

// networkReachable reports whether the external hosts referenced by a
// transcript can be dialed. A transcript with no external host is always
// reachable (it needs no network). Scenarios that do reference one are skipped
// when offline, mirroring TestIntegrationExternalLink.
func networkReachable(transcript string) bool {
	hosts := externalURLHosts(transcript)
	if len(hosts) == 0 {
		return true
	}
	for _, h := range hosts {
		port := "443"
		if h.scheme == "http" {
			port = "80"
		}
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(h.host, port), 10*time.Second)
		if err == nil {
			_ = conn.Close()
			return true
		}
	}
	return false
}

func TestExternalURLHosts(t *testing.T) {
	if got := externalURLHosts("no urls here"); len(got) != 0 {
		t.Fatalf("expected no hosts, got %v", got)
	}
	if got := externalURLHosts("$ codemd http://127.0.0.1:8137/x.go"); len(got) != 0 {
		t.Fatalf("localhost must not be external, got %v", got)
	}
	got := externalURLHosts("$ codemd https://example.com/a http://127.0.0.1/b")
	if len(got) != 1 || got[0].host != "example.com" || got[0].scheme != "https" {
		t.Fatalf("unexpected hosts: %v", got)
	}
	if !networkReachable("no urls here") {
		t.Fatal("a transcript with no URL needs no network and must be reachable")
	}
	if !networkReachable("$ codemd http://127.0.0.1:8137/x.go") {
		t.Fatal("a localhost URL must not require external network")
	}
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
	entries, err := os.ReadDir(consoleRoot)
	if err != nil {
		t.Fatalf("read objectives: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			src := filepath.Join(consoleRoot, e.Name())
			obj, err := os.ReadFile(filepath.Join(src, "transcript.console"))
			if err != nil {
				t.Skipf("no transcript.console: %v", err)
			}
			if !networkReachable(string(obj)) {
				t.Skipf("network unreachable; skipping %s", e.Name())
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
	err := verifyScenario(bin, dir, dir, false)
	if err == nil {
		t.Fatal("expected stale transcript to be detected as drift")
	}
	if !strings.Contains(err.Error(), "out of date") {
		t.Fatalf("expected out-of-date mismatch, got: %v", err)
	}
	if err := verifyScenario(bin, dir, dir, true); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if err := verifyScenario(bin, dir, dir, false); err != nil {
		t.Fatalf("round-trip after update failed: %v", err)
	}
}
