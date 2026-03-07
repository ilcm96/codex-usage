package report

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/ilcm96/codex-usage/internal/cache"
	"github.com/ilcm96/codex-usage/internal/codex"
	"github.com/ilcm96/codex-usage/internal/pricing"
)

type BuildOptions struct {
	SessionsDir string
	CacheDir    string
	Pricing     pricing.Pricing
	Location    *time.Location
}

type AggregatedReport struct {
	Title    string
	GroupBy  string
	Timezone string
	Rows     []Row
	Totals   RowTotals
}

type Row struct {
	Key     string
	Models  map[string]codex.Usage
	Totals  RowTotals
	CostUSD float64
}

type RowTotals struct {
	InputTokens     int64
	OutputTokens    int64
	ReasoningTokens int64
	CacheReadTokens int64
	TotalTokens     int64
}

type cacheEntry struct {
	Size  int64                             `json:"size"`
	Mtime int64                             `json:"mtime"`
	Daily map[string]map[string]codex.Usage `json:"daily"`
}

func metaString(meta map[string]any, key string) (string, bool) {
	v, ok := meta[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func metaInt64(meta map[string]any, key string) (int64, bool) {
	v, ok := meta[key]
	if !ok {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return int64(t), true
	case int64:
		return t, true
	case int:
		return int64(t), true
	default:
		return 0, false
	}
}

func expectedCacheMeta(opt BuildOptions, loc *time.Location) map[string]any {
	return map[string]any{
		"sessionsDir":   opt.SessionsDir,
		"timezone":      loc.String(),
		"parserVersion": codex.ParserVersion,
		"mtimeUnit":     "unix_nano",
	}
}

func isCacheMetaCompatible(meta map[string]any, expected map[string]any) bool {
	if meta == nil {
		return false
	}

	sessionsDir, ok := metaString(meta, "sessionsDir")
	if !ok || sessionsDir != expected["sessionsDir"] {
		return false
	}
	timezone, ok := metaString(meta, "timezone")
	if !ok || timezone != expected["timezone"] {
		return false
	}
	mtimeUnit, ok := metaString(meta, "mtimeUnit")
	if !ok || mtimeUnit != expected["mtimeUnit"] {
		return false
	}
	parserVersion, ok := metaInt64(meta, "parserVersion")
	if !ok || parserVersion != int64(expected["parserVersion"].(int)) {
		return false
	}
	return true
}

func BuildReport(cmd string, opt BuildOptions) (AggregatedReport, error) {
	// Ensure sessions dir exists.
	if st, err := os.Stat(opt.SessionsDir); err != nil || !st.IsDir() {
		return AggregatedReport{}, fmt.Errorf("Codex sessions directory not found: %s", opt.SessionsDir)
	}

	files, err := codex.DiscoverSessionFilesWithInfo(opt.SessionsDir)
	if err != nil {
		return AggregatedReport{}, err
	}

	loc := opt.Location
	if loc == nil {
		loc = time.Local
	}

	if len(files) == 0 {
		return AggregatedReport{
			Title:    titleFor(cmd),
			GroupBy:  cmd,
			Timezone: loc.String(),
			Rows:     []Row{},
		}, nil
	}

	cachePath := filepath.Join(opt.CacheDir, "cache-v2.json")
	c, err := cache.LoadCache[cacheEntry](cachePath)
	dirty := false
	if err != nil {
		// For a personal tool, treat cache issues as non-fatal.
		c = cache.CacheV1[cacheEntry]{Version: cache.CacheVersion, Files: map[string]cacheEntry{}}
		dirty = true
	}

	expMeta := expectedCacheMeta(opt, loc)
	if !isCacheMetaCompatible(c.Meta, expMeta) {
		// Invalidate on parsing or bucketing changes.
		c = cache.CacheV1[cacheEntry]{Version: cache.CacheVersion, Files: map[string]cacheEntry{}, Meta: expMeta}
		dirty = true
	} else {
		// Ensure we keep meta consistent even if older caches had it omitted.
		c.Meta = expMeta
	}

	// Purge stale cache entries for files that no longer exist on disk.
	known := make(map[string]struct{}, len(files))
	for _, f := range files {
		known[f.Path] = struct{}{}
	}
	for p := range c.Files {
		if _, ok := known[p]; ok {
			continue
		}
		if _, err := os.Stat(p); err != nil && os.IsNotExist(err) {
			delete(c.Files, p)
			dirty = true
		}
	}

	// Global daily aggregation.
	globalDaily := map[string]map[string]codex.Usage{}

	// First, merge cache hits and collect misses.
	var misses []codex.SessionFileInfo

	for _, f := range files {
		ce, ok := c.Files[f.Path]
		if ok && ce.Size == f.Size && ce.Mtime == f.Mtime && ce.Daily != nil {
			mergeDaily(globalDaily, ce.Daily)
			continue
		}
		misses = append(misses, f)
	}

	// Parse misses concurrently.
	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan codex.SessionFileInfo, workers*2)
	results := make(chan struct {
		path string
		ent  cacheEntry
		err  error
	}, workers*2)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for m := range jobs {
				parsed, err := codex.ParseSessionFile(m.Path, m.Size, m.Mtime, loc)
				if err != nil {
					results <- struct {
						path string
						ent  cacheEntry
						err  error
					}{path: m.Path, err: err}
					continue
				}
				ent := cacheEntry{
					Size:  parsed.Size,
					Mtime: parsed.Mtime,
					Daily: parsed.Daily,
				}
				results <- struct {
					path string
					ent  cacheEntry
					err  error
				}{path: m.Path, ent: ent, err: nil}
			}
		}()
	}

	go func() {
		for _, m := range misses {
			jobs <- m
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	for r := range results {
		if r.err != nil {
			// Keep going; skip unreadable files.
			continue
		}
		c.Files[r.path] = r.ent
		dirty = true
		mergeDaily(globalDaily, r.ent.Daily)
	}

	// Save updated cache (best-effort).
	if dirty {
		_ = cache.SaveCache(cachePath, c)
	}

	rows := buildRowsFromDaily(cmd, globalDaily, opt.Pricing)

	report := AggregatedReport{
		Title:    titleFor(cmd),
		GroupBy:  cmd,
		Timezone: loc.String(),
		Rows:     rows,
		Totals:   sumTotals(rows),
	}
	return report, nil
}

func titleFor(cmd string) string {
	switch cmd {
	case "monthly":
		return "Codex Token Usage Report - Monthly"
	default:
		return "Codex Token Usage Report - Daily"
	}
}

func mergeDaily(global map[string]map[string]codex.Usage, daily map[string]map[string]codex.Usage) {
	for day, models := range daily {
		gm := global[day]
		if gm == nil {
			gm = map[string]codex.Usage{}
			global[day] = gm
		}
		for model, usage := range models {
			cur := gm[model]
			cur.Add(usage)
			gm[model] = cur
		}
	}
}

func buildRowsFromDaily(cmd string, daily map[string]map[string]codex.Usage, pr pricing.Pricing) []Row {
	switch cmd {
	case "monthly":
		return buildMonthly(daily, pr)
	default:
		return buildDaily(daily, pr)
	}
}

func buildDaily(daily map[string]map[string]codex.Usage, pr pricing.Pricing) []Row {
	keys := make([]string, 0, len(daily))
	for k := range daily {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var rows []Row
	for _, day := range keys {
		models := daily[day]
		row := Row{
			Key:    day,
			Models: cloneModels(models),
		}
		finalizeRow(&row, pr)
		rows = append(rows, row)
	}
	return rows
}

func buildMonthly(daily map[string]map[string]codex.Usage, pr pricing.Pricing) []Row {
	byMonth := map[string]map[string]codex.Usage{}
	for day, models := range daily {
		if len(day) < 7 {
			continue
		}
		month := day[:7]
		mm := byMonth[month]
		if mm == nil {
			mm = map[string]codex.Usage{}
			byMonth[month] = mm
		}
		for model, usage := range models {
			cur := mm[model]
			cur.Add(usage)
			mm[model] = cur
		}
	}

	keys := make([]string, 0, len(byMonth))
	for k := range byMonth {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var rows []Row
	for _, month := range keys {
		row := Row{Key: month, Models: cloneModels(byMonth[month])}
		finalizeRow(&row, pr)
		rows = append(rows, row)
	}
	return rows
}

func cloneModels(in map[string]codex.Usage) map[string]codex.Usage {
	out := map[string]codex.Usage{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

func finalizeRow(row *Row, pr pricing.Pricing) {
	// Calculate display totals and cost (per-model pricing).
	var totals RowTotals
	var cost float64
	// Policy: if a model is unknown to the embedded pricing table, fall back to gpt-5 pricing.
	fallbackPricing, hasFallbackPricing := pr.GetModelPricing("gpt-5")

	// stable order not required here; table renderer can sort models.
	for model, u := range row.Models {
		cacheRead := u.CachedInputTokens
		if cacheRead > u.InputTokens {
			cacheRead = u.InputTokens
		}
		input := u.InputTokens - cacheRead
		if input < 0 {
			input = 0
		}
		reasoning := u.ReasoningOutputTokens
		if reasoning > u.OutputTokens {
			reasoning = u.OutputTokens
		}
		if reasoning < 0 {
			reasoning = 0
		}

		totals.InputTokens += input
		totals.OutputTokens += u.OutputTokens
		totals.ReasoningTokens += reasoning
		totals.CacheReadTokens += cacheRead
		totals.TotalTokens += u.TotalTokens

		if prc, ok := pr.GetModelPricing(model); ok {
			cost += pricing.CostUSD(struct {
				InputTokens     int64
				CacheReadTokens int64
				OutputTokens    int64
			}{
				InputTokens:     u.InputTokens,
				CacheReadTokens: cacheRead,
				OutputTokens:    u.OutputTokens,
			}, prc)
		} else if hasFallbackPricing {
			cost += pricing.CostUSD(struct {
				InputTokens     int64
				CacheReadTokens int64
				OutputTokens    int64
			}{
				InputTokens:     u.InputTokens,
				CacheReadTokens: cacheRead,
				OutputTokens:    u.OutputTokens,
			}, fallbackPricing)
		} else {
			_ = model // pricing missing; cost remains 0 for this model.
		}
	}

	row.Totals = totals
	row.CostUSD = cost
}

func sumTotals(rows []Row) RowTotals {
	var t RowTotals
	for _, r := range rows {
		t.InputTokens += r.Totals.InputTokens
		t.OutputTokens += r.Totals.OutputTokens
		t.ReasoningTokens += r.Totals.ReasoningTokens
		t.CacheReadTokens += r.Totals.CacheReadTokens
		t.TotalTokens += r.Totals.TotalTokens
	}
	return t
}
