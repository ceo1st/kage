package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/tamnd/kage/clone"
	"github.com/tamnd/kage/pack"
)

// packFlags holds the parsed flags for one invocation of kage pack.
type packFlags struct {
	format      string
	out         string
	base        string
	noCompress  bool
	title       string
	description string
	language    string
	date        string
}

func newPackCmd() *cobra.Command {
	f := &packFlags{}
	cmd := &cobra.Command{
		Use:   "pack <mirror-dir>",
		Short: "Pack a cloned mirror into a ZIM file or a self-contained viewer",
		Long: "pack turns a cloned folder into one distributable file. With --format zim\n" +
			"it writes an open ZIM archive (the format Kiwix uses) that kage open or any\n" +
			"ZIM reader can browse. With --format binary it appends that archive to a copy\n" +
			"of kage, producing a single executable that serves the site offline when run.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPack(args[0], f)
		},
	}
	fs := cmd.Flags()
	fs.StringVar(&f.format, "format", "zim", "output format: zim or binary")
	fs.StringVarP(&f.out, "out", "o", "", "output path (default per format)")
	fs.StringVar(&f.base, "base", "", "base kage binary for --format binary (default this kage)")
	fs.BoolVar(&f.noCompress, "no-compress", false, "store every cluster raw, no zstd")
	fs.StringVar(&f.title, "title", "", "archive title (default the main page's <title>)")
	fs.StringVar(&f.description, "description", "", "archive description")
	fs.StringVar(&f.language, "language", "eng", "archive language code")
	fs.StringVar(&f.date, "date", time.Now().UTC().Format("2006-01-02"), "archive date (YYYY-MM-DD)")
	return cmd
}

func runPack(mirrorArg string, f *packFlags) error {
	dir := resolveMirror(mirrorArg)
	zopts := pack.ZIMOptions{
		Out:         f.out,
		NoCompress:  f.noCompress,
		Title:       f.title,
		Description: f.description,
		Language:    f.language,
		Date:        f.date,
		Version:     Version,
	}

	switch f.format {
	case "zim":
		out, size, err := pack.BuildZIM(dir, zopts)
		if err != nil {
			return err
		}
		printPackResult(out, size)
		fmt.Fprintf(os.Stderr, "  open %s\n", styleAccent.Render("kage open "+out))
		return nil

	case "binary":
		zbytes, err := pack.BuildZIMBytes(dir, zopts)
		if err != nil {
			return err
		}
		target := resolveTargetOS(f.base)
		out := f.out
		if out == "" {
			out = defaultBinaryName(dir)
		}
		// A Windows viewer must end in .exe to run, whether the name came from
		// --out or the default, so make sure it does.
		if target == "windows" && !strings.HasSuffix(strings.ToLower(out), ".exe") {
			out += ".exe"
		}
		path, size, err := pack.BuildBinary(zbytes, pack.BinaryOptions{Out: out, Base: f.base})
		if err != nil {
			return err
		}
		printPackResult(path, size)
		printRunHint(path, target)
		return nil

	default:
		return fmt.Errorf("unknown --format %q (want zim or binary)", f.format)
	}
}

// resolveMirror accepts either a path to a mirror dir or a bare host. A bare
// host that is not a directory in the working dir is resolved against the
// default out dir, so "kage pack paulgraham.com" works right after a clone.
func resolveMirror(arg string) string {
	if info, err := os.Stat(arg); err == nil && info.IsDir() {
		return arg
	}
	candidate := filepath.Join(clone.DefaultOutDir(), arg)
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return arg
}

// defaultBinaryName derives a clean program name from the mirror's host by
// stripping a trailing dot-suffix (paulgraham.com -> paulgraham). The caller
// appends .exe for Windows targets.
func defaultBinaryName(dir string) string {
	host := filepath.Base(dir)
	if i := strings.IndexByte(host, '.'); i > 0 {
		return host[:i]
	}
	return host
}

// resolveTargetOS reports which OS the packed viewer will run on. With no
// --base it is this kage's OS; with one, we sniff the base's executable header
// so detection does not hinge on the file being named ".exe". If the header is
// unrecognised we fall back to that name heuristic.
func resolveTargetOS(base string) string {
	if base == "" {
		return runtime.GOOS
	}
	if os := pack.SniffOS(base); os != "" {
		return os
	}
	if strings.HasSuffix(strings.ToLower(base), ".exe") {
		return "windows"
	}
	return ""
}

func printPackResult(path string, size int64) {
	fmt.Fprintln(os.Stderr, styleOK.Render("packed")+" "+styleTitle.Render(path))
	fmt.Fprintf(os.Stderr, "  %s %s\n", styleAccent.Render("size"), humanBytes(size))
}

func printRunHint(path, target string) {
	rel := path
	if !strings.ContainsAny(path, "/\\") {
		rel = "./" + path
	}
	// A viewer built for another OS cannot run here, so say where it goes
	// instead of printing a run command that would not work.
	if target != "" && target != runtime.GOOS {
		fmt.Fprintf(os.Stderr, "  this is a %s viewer; copy %s to that machine to run it\n",
			osLabel(target), styleAccent.Render(filepath.Base(path)))
		return
	}
	fmt.Fprintf(os.Stderr, "  run %s to view the site offline\n", styleAccent.Render(rel))
	if target == "darwin" {
		fmt.Fprintln(os.Stderr, styleDim.Render("  (macOS may quarantine it: xattr -d com.apple.quarantine "+rel+")"))
	}
}

// osLabel turns a GOOS value into a friendly name for the run hint.
func osLabel(goos string) string {
	switch goos {
	case "windows":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return goos
	}
}

// humanBytes renders a byte count in B, KiB, MiB, or GiB.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
