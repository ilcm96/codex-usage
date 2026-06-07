package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/ilcm96/codex-usage/internal/cli/codex"
	"github.com/ilcm96/codex-usage/internal/cli/output"
	"github.com/ilcm96/codex-usage/internal/cli/report"
	"github.com/ilcm96/codex-usage/internal/cli/syncclient"
	"github.com/ilcm96/codex-usage/internal/core/pricing"
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

	if cmd == "sync" {
		return runSync(args)
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

func runSync(args []string) int {
	flags, err := parseSyncFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 2
	}
	if flags.help {
		printHelp()
		return 0
	}

	serverURL := firstNonEmpty(flags.serverURL, os.Getenv("CODEX_USAGE_SERVER_URL"))
	deviceToken := firstNonEmpty(flags.deviceToken, os.Getenv("CODEX_USAGE_DEVICE_TOKEN"))
	if serverURL == "" {
		fmt.Fprintln(os.Stderr, "Missing server URL. Use --server or CODEX_USAGE_SERVER_URL.")
		return 2
	}
	if deviceToken == "" {
		fmt.Fprintln(os.Stderr, "Missing device token. Use --token or CODEX_USAGE_DEVICE_TOKEN.")
		return 2
	}

	codexHome := codex.ResolveCodexHome()
	res, err := syncclient.Run(context.Background(), syncclient.Options{
		SessionsDir: filepath.Join(codexHome, "sessions"),
		CacheDir:    filepath.Join(codexHome, cacheRelDir),
		ServerURL:   serverURL,
		DeviceToken: deviceToken,
		Limit:       flags.limit,
		BatchSize:   flags.batchSize,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Sync failed:", err)
		return 1
	}
	fmt.Printf("Sync complete: scanned=%d uploaded=%d skipped=%d failed=%d\n", res.Scanned, res.Uploaded, res.Skipped, res.Failed)
	for _, errText := range res.Errors {
		fmt.Fprintln(os.Stderr, errText)
	}
	if res.Failed > 0 {
		return 1
	}
	return 0
}

type parsedFlags struct {
	help       bool
	version    bool
	forceColor bool
}

type syncFlags struct {
	help        bool
	serverURL   string
	deviceToken string
	limit       int
	batchSize   int
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

func parseSyncFlags(args []string) (syncFlags, error) {
	var out syncFlags
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-h", "--help":
			out.help = true
		case "--server":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("--server requires a value")
			}
			out.serverURL = args[i]
		case "--token":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("--token requires a value")
			}
			out.deviceToken = args[i]
		case "--limit":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("--limit requires a value")
			}
			var limit int
			if _, err := fmt.Sscanf(args[i], "%d", &limit); err != nil || limit < 0 {
				return out, fmt.Errorf("--limit must be a non-negative integer")
			}
			out.limit = limit
		case "--batch-size":
			i++
			if i >= len(args) {
				return out, fmt.Errorf("--batch-size requires a value")
			}
			var batchSize int
			if _, err := fmt.Sscanf(args[i], "%d", &batchSize); err != nil || batchSize < 0 {
				return out, fmt.Errorf("--batch-size must be a non-negative integer")
			}
			out.batchSize = batchSize
		default:
			return out, fmt.Errorf("Unknown option: %s", a)
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
	fmt.Println("  sync     Upload Codex sessions to a codex-usage API server")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  --color        Enable colored output (default: auto). FORCE_COLOR=1 has the same effect.")
	fmt.Println("  --server URL   API server URL for sync (or CODEX_USAGE_SERVER_URL)")
	fmt.Println("  --token TOKEN  Device token for sync (or CODEX_USAGE_DEVICE_TOKEN)")
	fmt.Println("  --limit N      Upload at most N changed sessions")
	fmt.Println("  --batch-size N Number of session files per upload batch (default: 50)")
	fmt.Println("  -h, --help     Display this help message")
	fmt.Println("  -v, --version  Display version")
	fmt.Println()
	fmt.Println("NOTES:")
	fmt.Println("  - Reads sessions from ~/.codex/sessions (or $CODEX_HOME/sessions).")
	fmt.Println("  - Caches are stored under ~/.codex/codex-usage/.")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
