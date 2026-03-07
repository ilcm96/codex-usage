package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/ilcm96/codex-usage/internal/codex"
	"github.com/ilcm96/codex-usage/internal/output"
	"github.com/ilcm96/codex-usage/internal/pricing"
	"github.com/ilcm96/codex-usage/internal/report"
)

const (
	projectName = "codex-usage"
	cacheRelDir = "codex-usage"
)

func Run(args []string) int {
	cmd := "daily"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd = args[0]
		args = args[1:]
	}

	flags, err := parseFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}

	if flags.help {
		printHelp()
		return 0
	}
	if flags.version {
		// No git tags here; keep it simple.
		fmt.Println(projectName)
		return 0
	}

	if cmd != "daily" && cmd != "monthly" {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		return 2
	}

	// Default is "auto": enable color when stdout is a terminal.
	// NO_COLOR disables colors unless explicitly overridden by --color or FORCE_COLOR.
	colorEnabled := output.IsTerminal(os.Stdout.Fd())
	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
	}
	if flags.forceColor || os.Getenv("FORCE_COLOR") != "" {
		colorEnabled = true
	}

	width := 120
	if output.IsTerminal(os.Stdout.Fd()) {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			width = w
		}
	}
	compact := width < 100

	codexHome := codex.ResolveCodexHome()
	sessionsDir := filepath.Join(codexHome, "sessions")
	cacheDir := filepath.Join(codexHome, cacheRelDir)

	pr, err := pricing.LoadEmbeddedPricing()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load embedded pricing:", err)
		return 1
	}

	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		// Should not happen on typical systems; fall back to local.
		loc = time.Local
	}

	aggregated, err := report.BuildReport(cmd, report.BuildOptions{
		SessionsDir: sessionsDir,
		CacheDir:    cacheDir,
		Pricing:     pr,
		Location:    loc,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	out := output.NewTableRenderer(output.TableRendererOptions{
		Color:   colorEnabled,
		Compact: compact,
		Width:   width,
	})
	out.Render(aggregated)
	return 0
}

type parsedFlags struct {
	help       bool
	version    bool
	forceColor bool
}

func parseFlags(args []string) (parsedFlags, error) {
	out := parsedFlags{
		forceColor: false,
	}
	for _, a := range args {
		switch a {
		case "-h", "--help":
			out.help = true
		case "-v", "--version":
			out.version = true
		case "--color":
			out.forceColor = true
		default:
			if strings.HasPrefix(a, "-") {
				return out, fmt.Errorf("Unknown option: %s", a)
			}
			return out, fmt.Errorf("Unexpected argument: %s", a)
		}
	}
	return out, nil
}

func printHelp() {
	fmt.Println("USAGE:")
	fmt.Println("  codex-usage [daily|monthly] [OPTIONS]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  daily    Show Codex token usage grouped by day")
	fmt.Println("  monthly  Show Codex token usage grouped by month")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  --color        Enable colored output (default: auto). FORCE_COLOR=1 has the same effect.")
	fmt.Println("  -h, --help     Display this help message")
	fmt.Println("  -v, --version  Display version")
	fmt.Println()
	fmt.Println("NOTES:")
	fmt.Println("  - Reads sessions from ~/.codex/sessions (or $CODEX_HOME/sessions).")
	fmt.Println("  - Caches are stored under ~/.codex/codex-usage/.")
}
