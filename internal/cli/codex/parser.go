package codex

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/ilcm96/codex-usage/internal/core/codexlog"
)

type FileParseResult struct {
	Size  int64
	Mtime int64
	// Daily[dateKey][model]usage
	Daily map[string]map[string]codexlog.Usage
}

func ParseSessionFile(path string, size int64, mtime int64, loc *time.Location) (FileParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return FileParseResult{}, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	res := FileParseResult{
		Size:  size,
		Mtime: mtime,
		Daily: map[string]map[string]codexlog.Usage{},
	}

	reader := bufio.NewScanner(f)
	reader.Buffer(make([]byte, 64*1024), 8*1024*1024)

	normalizer := codexlog.NewUsageNormalizer(codexlog.DefaultFallbackModel)

	for reader.Scan() {
		line := bytes.TrimSpace(reader.Bytes())
		if len(line) == 0 {
			continue
		}

		if normalizer.ObserveModel(line) {
			continue
		}
		if !codexlog.IsTokenCountEvent(line) {
			continue
		}

		day, ok := codexlog.DateKey(codexlog.Timestamp(line), loc)
		if !ok {
			continue
		}

		event, ok := normalizer.NormalizeUsageLine(line)
		if !ok {
			continue
		}

		dayMap := res.Daily[day]
		if dayMap == nil {
			dayMap = map[string]codexlog.Usage{}
			res.Daily[day] = dayMap
		}
		u := dayMap[event.Model]
		u.Add(event.Usage)
		dayMap[event.Model] = u
	}

	if err := reader.Err(); err != nil {
		return FileParseResult{}, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return res, nil
}
