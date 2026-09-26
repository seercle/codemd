package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/seercle/codemd/internal/lang"
	"github.com/seercle/codemd/internal/srcfile"
)

type Options struct {
	Write  bool
	Output string
	Diff   bool
	Check  bool
	Config string
}

func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("codemd", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opt Options
	fs.BoolVar(&opt.Write, "w", false, "write result in place")
	fs.StringVar(&opt.Output, "o", "", "write result to a new file")
	fs.BoolVar(&opt.Diff, "d", false, "print a unified diff")
	fs.BoolVar(&opt.Check, "check", false, "exit non-zero if any file would change")
	fs.StringVar(&opt.Config, "config", "", "path to config file")
	if err := fs.Parse(args); err != nil {
		return 2
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
		data, err := io.ReadAll(stdin)
		if err != nil {
			fmt.Fprintf(stderr, "codemd: %v\n", err)
			return 1
		}
		if configTable != nil {
			resolver.Table = configTable
		}
		out, errs := resolver.ResolveDocument(string(data), ".")
		io.WriteString(stdout, out)
		return reportErrors(errs, stderr)
	}

	exit := 0
	for _, file := range files {
		// Derive a fresh table per file so config cannot leak across files.
		table := lang.Builtins()
		switch {
		case configTable != nil:
			table = configTable
		default:
			baseDir := filepath.Dir(file)
			if cfgPath, err := lang.DiscoverConfig(baseDir); err == nil && cfgPath != "" {
				if cfg, err := lang.LoadConfig(cfgPath); err == nil {
					if merged, err := lang.Merge(lang.Builtins(), cfg); err == nil {
						table = merged
					}
				}
			}
		}
		resolver.Table = table

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(stderr, "codemd: %v\n", err)
			exit = 1
			continue
		}
		baseDir := filepath.Dir(file)
		out, errs := resolver.ResolveDocument(string(data), baseDir)
		if reportErrors(errs, stderr) != 0 {
			exit = 1
		}
		changed := out != string(data)
		switch {
		case opt.Check:
			if changed {
				fmt.Fprintf(stderr, "codemd: %s is out of date\n", file)
				exit = 1
			}
		case opt.Diff:
			if changed {
				io.WriteString(stdout, unifiedDiff(file, string(data), out))
			}
		case opt.Write:
			if changed {
				if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
					fmt.Fprintf(stderr, "codemd: %v\n", err)
					exit = 1
				}
			}
		case opt.Output != "":
			if err := os.WriteFile(opt.Output, []byte(out), 0o644); err != nil {
				fmt.Fprintf(stderr, "codemd: %v\n", err)
				exit = 1
			}
		default:
			io.WriteString(stdout, out)
		}
	}
	return exit
}

func reportErrors(errs []RefError, stderr io.Writer) int {
	for _, e := range errs {
		if e.Line > 0 {
			fmt.Fprintf(stderr, "codemd: line %d: %v\n", e.Line, e.Err)
		} else {
			fmt.Fprintf(stderr, "codemd: %v\n", e.Err)
		}
	}
	if len(errs) > 0 {
		return 1
	}
	return 0
}

func unifiedDiff(name, old, new string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n+++ %s\n", name, name)
	oldLines := strings.Split(old, "\n")
	newLines := strings.Split(new, "\n")
	for _, l := range oldLines {
		fmt.Fprintf(&b, "-%s\n", l)
	}
	for _, l := range newLines {
		fmt.Fprintf(&b, "+%s\n", l)
	}
	return b.String()
}
