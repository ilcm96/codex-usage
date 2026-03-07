package output

import (
	"fmt"
	"sort"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/ilcm96/codex-usage/internal/codex"
	"github.com/ilcm96/codex-usage/internal/report"
)

type TableRendererOptions struct {
	Color   bool
	Compact bool
	Width   int
}

type TableRenderer struct {
	opt TableRendererOptions
}

func NewTableRenderer(opt TableRendererOptions) *TableRenderer {
	return &TableRenderer{opt: opt}
}

func (r *TableRenderer) Render(rep report.AggregatedReport) {
	if r.opt.Color {
		// Allow forced color even when stdout isn't a TTY (e.g. piping to a file).
		text.ANSICodesSupported = true
		text.EnableColors()
	} else {
		text.DisableColors()
	}

	if len(rep.Rows) == 0 {
		fmt.Println("No Codex usage data found.")
		return
	}

	t := table.NewWriter()
	style := table.StyleLight
	style.Format.Header = text.FormatDefault
	style.Options.SeparateRows = true
	style.Options.DoNotColorBordersAndSeparators = true
	if r.opt.Color {
		style.Color.Header = text.Colors{text.FgCyan}
	}
	t.SetStyle(style)
	if r.opt.Color {
		t.SetRowPainter(func(row table.Row) text.Colors {
			// Color the totals row.
			if len(row) > 0 {
				if s, ok := row[0].(string); ok && s == "Total" {
					return text.Colors{text.FgYellow}
				}
			}
			return nil
		})
	}

	dateHeader := headerFor(rep.GroupBy)
	if r.opt.Compact {
		t.AppendHeader(table.Row{dateHeader, "Models", "Input", "Output", "Cost (USD)"})
		for _, row := range rep.Rows {
			t.AppendRow(table.Row{
				formatKey(rep, row.Key),
				r.formatModels(row.Models),
				formatNumber(row.Totals.InputTokens),
				formatNumber(row.Totals.OutputTokens),
				formatCurrency(row.CostUSD),
			})
		}
		t.AppendRow(table.Row{
			"Total", "",
			formatNumber(rep.Totals.InputTokens),
			formatNumber(rep.Totals.OutputTokens),
			formatCurrency(sumCost(rep.Rows)),
		})
		fmt.Println(t.Render())
		return
	}

	t.AppendHeader(table.Row{
		dateHeader,
		"Models",
		"Input",
		"Output",
		"Reasoning",
		"Cache Read",
		"Total Tokens",
		"Cost (USD)",
	})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft},
		{Number: 2, Align: text.AlignLeft},
		{Number: 3, Align: text.AlignRight},
		{Number: 4, Align: text.AlignRight},
		{Number: 5, Align: text.AlignRight},
		{Number: 6, Align: text.AlignRight},
		{Number: 7, Align: text.AlignRight},
		{Number: 8, Align: text.AlignRight},
	})

	for _, row := range rep.Rows {
		t.AppendRow(table.Row{
			formatKey(rep, row.Key),
			r.formatModels(row.Models),
			formatNumber(row.Totals.InputTokens),
			formatNumber(row.Totals.OutputTokens),
			formatNumber(row.Totals.ReasoningTokens),
			formatNumber(row.Totals.CacheReadTokens),
			formatNumber(row.Totals.TotalTokens),
			formatCurrency(row.CostUSD),
		})
	}

	t.AppendRow(table.Row{
		"Total",
		"",
		formatNumber(rep.Totals.InputTokens),
		formatNumber(rep.Totals.OutputTokens),
		formatNumber(rep.Totals.ReasoningTokens),
		formatNumber(rep.Totals.CacheReadTokens),
		formatNumber(rep.Totals.TotalTokens),
		formatCurrency(sumCost(rep.Rows)),
	})

	fmt.Println(t.Render())
}

func headerFor(groupBy string) string {
	switch groupBy {
	case "monthly":
		return "Month"
	default:
		return "Date"
	}
}

func formatKey(rep report.AggregatedReport, key string) string {
	if rep.GroupBy != "monthly" {
		return key
	}
	// key is "YYYY-MM"
	loc := time.Local
	if rep.Timezone != "" {
		if l, err := time.LoadLocation(rep.Timezone); err == nil {
			loc = l
		}
	}
	t, err := time.ParseInLocation("2006-01", key, loc)
	if err != nil {
		return key
	}
	// Keep the output stable/sort-friendly: YYYY-MM (e.g. 2026-02)
	return t.Format("2006-01")
}

func sumCost(rows []report.Row) float64 {
	var s float64
	for _, r := range rows {
		s += r.CostUSD
	}
	return s
}

func (r *TableRenderer) formatModels(models map[string]codex.Usage) string {
	if len(models) == 0 {
		return ""
	}
	names := make([]string, 0, len(models))
	for name := range models {
		names = append(names, name)
	}
	sort.Strings(names)

	lines := make([]string, 0, len(names))
	for _, name := range names {
		lines = append(lines, "- "+name)
	}
	return joinLines(lines)
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	out := lines[0]
	for i := 1; i < len(lines); i++ {
		out += "\n" + lines[i]
	}
	return out
}

func formatNumber(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	// insert commas
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		return "-" + s
	}
	return s
}

func formatCurrency(v float64) string {
	return fmt.Sprintf("$%.2f", v)
}
