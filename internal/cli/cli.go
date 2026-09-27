// Package cli implements the codemd command-line interface: flag parsing, input
// expansion, and orchestration of reference resolution.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/srcfile"
)

// Version is the reported build version. Override at build time with
// -ldflags "-X github.com/seercle/codemd/internal/cli.Version=vX.Y.Z".
var Version = "dev"

// Options holds the command-line flags that control how Run reads, rewrites, or
// reports on its inputs.
type Options struct {
	Write  bool
	Output string
	Diff   bool
	Check  bool
	Config string
	Force  bool
}

// Run executes the codemd command line with the given arguments, reading stdin
// and writing results to stdout and diagnostics to stderr. It returns the
// process exit code: 0 on success, 1 for reference or I/O errors, and 2 for
// usage errors.
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("codemd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, "codemd resolves code references embedded in Markdown.\n\n")
		fmt.Fprint(stderr, "Usage:\n")
		fmt.Fprint(stderr, "  codemd [flags] file.md...\n")
		fmt.Fprint(stderr, "  codemd [flags]              (no files: read stdin, write stdout)\n\n")
		fmt.Fprint(stderr, "Reference: [codemd]:# (MODE RANGE PATH [LANG] [strip] [\"LINK-TEXT\"])\n\n")
		fmt.Fprint(stderr, "Flags:\n")
		fs.PrintDefaults()
	}
	var opt Options
	fs.BoolVar(&opt.Write, "w", false, "write result in place")
	fs.StringVar(&opt.Output, "o", "", "write result to a new file")
	fs.BoolVar(&opt.Diff, "d", false, "print a unified diff")
	fs.BoolVar(&opt.Check, "check", false, "exit non-zero if any file would change")
	fs.StringVar(&opt.Config, "config", "", "path to config file")
	fs.BoolVar(&opt.Force, "f", false, "write even if some references failed")
	fs.BoolVar(&opt.Force, "force", false, "write even if some references failed")
	languages := fs.Bool("languages", false, "list supported languages and exit")
	version := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *version {
		fmt.Fprintf(stdout, "codemd %s\n", Version)
		return 0
	}
	if *languages {
		table := lang.Builtins()
		if opt.Config != "" {
			cfg, err := lang.LoadConfig(opt.Config)
			if err != nil {
				fmt.Fprintf(stderr, "codemd: %v\n", err)
				return 1
			}
			table, err = lang.Merge(lang.Builtins(), cfg)
			if err != nil {
				fmt.Fprintf(stderr, "codemd: %v\n", err)
				return 1
			}
		}
		for _, line := range lang.Describe(table) {
			fmt.Fprintln(stdout, line)
		}
		return 0
	}
	exclusive := 0
	for _, b := range []bool{opt.Write, opt.Output != "", opt.Diff, opt.Check} {
		if b {
			exclusive++
		}
	}
	if exclusive > 1 {
		fmt.Fprintln(stderr, "codemd: -w, -o, -d and --check are mutually exclusive")
		return 2
	}
	files := fs.Args()
	if len(files) > 0 {
		expanded, err := expandInputs(files)
		if err != nil {
			fmt.Fprintf(stderr, "codemd: %v\n", err)
			return 2
		}
		files = expanded
	}
	if opt.Output != "" && len(files) != 1 {
		fmt.Fprintln(stderr, "codemd: -o requires exactly one input file")
		return 2
	}
	if len(files) == 0 && (opt.Write || opt.Output != "" || opt.Diff) {
		fmt.Fprintln(stderr, "codemd: in-place flags require an input file")
		return 2
	}

	// --config is global: load and merge it once, then reuse for every file.
	var configTable map[string]lang.Language
	if opt.Config != "" {
		cfg, err := lang.LoadConfig(opt.Config)
		if err != nil {
			fmt.Fprintf(stderr, "codemd: %v\n", err)
			return 1
		}
		configTable, err = lang.Merge(lang.Builtins(), cfg)
		if err != nil {
			fmt.Fprintf(stderr, "codemd: %v\n", err)
			return 1
		}
	}

	resolver := Resolver{Loader: srcfile.NewLoader(), Table: lang.Builtins()}

	if len(files) == 0 {
		table, err := resolveTable(".", configTable)
		if err != nil {
			fmt.Fprintf(stderr, "codemd: %v\n", err)
			return 1
		}
		resolver.Table = table
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "codemd: %v\n", err)
			return 1
		}
		out, errs := resolver.ResolveDocument(string(data), ".")
		errCount := reportErrors("<stdin>", errs, stderr)
		if opt.Check {
			// --check is not an in-place flag: it is valid with stdin for CI piping.
			if out != string(data) {
				fmt.Fprintln(stderr, "codemd: <stdin> is out of date")
				errCount++
			}
		} else {
			io.WriteString(stdout, out)
		}
		if errCount > 0 {
			fmt.Fprintf(stderr, "codemd: %d error(s)\n", errCount)
			return 1
		}
		return 0
	}

	exit := 0
	errCount := 0
	checked := 0
	updated := 0
	changedCount := 0
	for _, file := range files {
		baseDir := filepath.Dir(file)
		table, err := resolveTable(baseDir, configTable)
		if err != nil {
			fmt.Fprintf(stderr, "codemd: %v\n", err)
			errCount++
			exit = 1
			continue
		}
		resolver.Table = table

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(stderr, "codemd: %v\n", err)
			errCount++
			exit = 1
			continue
		}
		out, errs := resolver.ResolveDocument(string(data), baseDir)
		fileErrs := len(errs)
		if n := reportErrors(file, errs, stderr); n > 0 {
			errCount += n
			exit = 1
		}
		changed := out != string(data)
		checked++
		if changed {
			changedCount++
		}
		switch {
		case opt.Check:
			if changed {
				fmt.Fprintf(stderr, "codemd: %s is out of date\n", file)
				errCount++
				exit = 1
			}
		case opt.Diff:
			if changed {
				io.WriteString(stdout, unifiedDiff(file, string(data), out))
			}
		case opt.Write:
			if fileErrs > 0 && !opt.Force {
				fmt.Fprintf(stderr, "codemd: %s: not written due to errors (use --force to write anyway)\n", file)
			} else if changed {
				if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
					fmt.Fprintf(stderr, "codemd: %v\n", err)
					errCount++
					exit = 1
				} else {
					updated++
				}
			}
		case opt.Output != "":
			if fileErrs > 0 && !opt.Force {
				fmt.Fprintf(stderr, "codemd: %s: not written due to errors (use --force to write anyway)\n", file)
			} else if err := os.WriteFile(opt.Output, []byte(out), 0o644); err != nil {
				fmt.Fprintf(stderr, "codemd: %v\n", err)
				errCount++
				exit = 1
			}
		default:
			io.WriteString(stdout, out)
		}
	}
	if opt.Check {
		fmt.Fprintf(stderr, "codemd: %d file(s) checked, %d out of date\n", checked, changedCount)
	}
	if opt.Write {
		fmt.Fprintf(stderr, "codemd: %d file(s) checked, %d updated\n", checked, updated)
	}
	if errCount > 0 {
		fmt.Fprintf(stderr, "codemd: %d error(s)\n", errCount)
	}
	return exit
}

// resolveTable returns the language table for baseDir. When explicit is
// non-nil (an --config table) it is used as-is; otherwise the config is
// discovered by walking up from baseDir.
func resolveTable(baseDir string, explicit map[string]lang.Language) (map[string]lang.Language, error) {
	if explicit != nil {
		return explicit, nil
	}
	cfgPath, err := lang.DiscoverConfig(baseDir)
	if err != nil {
		return nil, err
	}
	if cfgPath == "" {
		return lang.Builtins(), nil
	}
	cfg, err := lang.LoadConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	return lang.Merge(lang.Builtins(), cfg)
}

func reportErrors(name string, errs []RefError, stderr io.Writer) int {
	for _, e := range errs {
		if e.Line > 0 {
			fmt.Fprintf(stderr, "codemd: %s: line %d: %v\n", name, e.Line, e.Err)
		} else {
			fmt.Fprintf(stderr, "codemd: %s: %v\n", name, e.Err)
		}
	}
	if len(errs) > 0 {
		return len(errs)
	}
	return 0
}
