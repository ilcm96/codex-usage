package output

import (
	"testing"

	"github.com/ilcm96/codex-usage/internal/report"
)

func TestFormatKey_Monthly_YYYYMM(t *testing.T) {
	rep := report.AggregatedReport{GroupBy: "monthly"}
	if got := formatKey(rep, "2026-02"); got != "2026-02" {
		t.Fatalf("formatKey(monthly, 2026-02) = %q, want %q", got, "2026-02")
	}
}

func TestFormatKey_NotMonthly_Passthrough(t *testing.T) {
	rep := report.AggregatedReport{GroupBy: "daily"}
	if got := formatKey(rep, "2026-02-09"); got != "2026-02-09" {
		t.Fatalf("formatKey(daily, 2026-02-09) = %q, want %q", got, "2026-02-09")
	}
}

func TestFormatKey_Monthly_InvalidKey_Passthrough(t *testing.T) {
	rep := report.AggregatedReport{GroupBy: "monthly"}
	invalid := "2026-2"
	if got := formatKey(rep, invalid); got != invalid {
		t.Fatalf("formatKey(monthly, %q) = %q, want %q", invalid, got, invalid)
	}
}
