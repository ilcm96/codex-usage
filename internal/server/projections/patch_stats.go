package projections

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ilcm96/codex-usage/internal/server/sessionparse"
)

type patchStats struct {
	AddedLines       int64
	LanguageLines    map[string]int64
	DominantLanguage string
}

type patchApplyEvent struct {
	Changes map[string]patchChange `json:"changes"`
}

type patchChange struct {
	Type        string `json:"type"`
	Content     string `json:"content"`
	UnifiedDiff string `json:"unified_diff"`
	MovePath    string `json:"move_path"`
}

func collectPatchStats(tools []sessionparse.ToolEvent) patchStats {
	if stats, ok := collectAppliedPatchStats(tools); ok {
		return stats
	}

	stats := patchStats{LanguageLines: map[string]int64{}}
	for _, tool := range tools {
		if !isPatchTool(tool) {
			continue
		}
		toolStats := parsePatchStats(tool.InputText)
		stats.AddedLines += toolStats.AddedLines
		for language, lines := range toolStats.LanguageLines {
			stats.LanguageLines[language] += lines
		}
	}
	stats.DominantLanguage = dominantPatchLanguage(stats.LanguageLines)
	return stats
}

func collectAppliedPatchStats(tools []sessionparse.ToolEvent) (patchStats, bool) {
	stats := patchStats{LanguageLines: map[string]int64{}}
	found := false
	for _, tool := range tools {
		if !strings.Contains(tool.Kind, "patch_apply") {
			continue
		}
		toolStats, ok := parseAppliedPatchStats(tool.PayloadJSON)
		if !ok {
			continue
		}
		found = true
		mergePatchStats(&stats, toolStats)
	}
	if !found {
		return patchStats{}, false
	}
	stats.DominantLanguage = dominantPatchLanguage(stats.LanguageLines)
	return stats, true
}

func parseAppliedPatchStats(payload []byte) (patchStats, bool) {
	var event patchApplyEvent
	if err := json.Unmarshal(payload, &event); err != nil || event.Changes == nil {
		return patchStats{}, false
	}

	stats := patchStats{LanguageLines: map[string]int64{}}
	for path, change := range event.Changes {
		languagePath := path
		if change.MovePath != "" {
			languagePath = change.MovePath
		}
		language := languageForPath(languagePath)
		var addedLines int64
		switch change.Type {
		case "add":
			addedLines = countContentLines(change.Content)
		case "update":
			addedLines = countAddedLinesInPatch(change.UnifiedDiff)
		}
		stats.AddedLines += addedLines
		stats.LanguageLines[language] += addedLines
	}
	stats.DominantLanguage = dominantPatchLanguage(stats.LanguageLines)
	return stats, true
}

func countContentLines(content string) int64 {
	var total int64
	for range strings.Lines(content) {
		total++
	}
	return total
}

func mergePatchStats(target *patchStats, source patchStats) {
	target.AddedLines += source.AddedLines
	for language, lines := range source.LanguageLines {
		target.LanguageLines[language] += lines
	}
}

func countPatchAddedLines(tools []sessionparse.ToolEvent) int64 {
	return collectPatchStats(tools).AddedLines
}

func isPatchTool(tool sessionparse.ToolEvent) bool {
	return tool.ToolName == "apply_patch" || strings.Contains(tool.Kind, "patch_apply")
}

func countAddedLinesInPatch(input string) int64 {
	var total int64
	for line := range strings.Lines(input) {
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			total++
		}
	}
	return total
}

func parsePatchStats(input string) patchStats {
	stats := patchStats{LanguageLines: map[string]int64{}}
	currentLanguage := ""
	for line := range strings.Lines(input) {
		line = strings.TrimSuffix(line, "\n")
		line = strings.TrimSuffix(line, "\r")
		if path := patchHeaderPath(line); path != "" {
			currentLanguage = languageForPath(path)
			continue
		}
		if !strings.HasPrefix(line, "+") || strings.HasPrefix(line, "+++") {
			continue
		}
		stats.AddedLines++
		language := currentLanguage
		if language == "" {
			language = "Unknown"
		}
		stats.LanguageLines[language]++
	}
	stats.DominantLanguage = dominantPatchLanguage(stats.LanguageLines)
	return stats
}

func patchHeaderPath(line string) string {
	for _, prefix := range []string{"*** Add File:", "*** Update File:", "*** Delete File:"} {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	if strings.HasPrefix(line, "+++ ") {
		path := strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
		if path == "/dev/null" {
			return ""
		}
		return strings.TrimPrefix(path, "b/")
	}
	return ""
}

func dominantPatchLanguage(languageLines map[string]int64) string {
	if len(languageLines) == 0 {
		return ""
	}
	languages := make([]string, 0, len(languageLines))
	for language := range languageLines {
		languages = append(languages, language)
	}
	sort.Strings(languages)
	dominant := ""
	var dominantLines int64
	for _, language := range languages {
		lines := languageLines[language]
		if lines > dominantLines {
			dominant = language
			dominantLines = lines
		}
	}
	return dominant
}

func languageForPath(path string) string {
	extension := strings.ToLower(filepath.Ext(path))
	switch extension {
	case ".go":
		return "Go"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "JavaScript"
	case ".json":
		return "JSON"
	case ".html", ".htm":
		return "HTML"
	case ".css":
		return "CSS"
	case ".scss", ".sass":
		return "Sass"
	case ".md", ".markdown":
		return "Markdown"
	case ".sql":
		return "SQL"
	case ".py":
		return "Python"
	case ".rs":
		return "Rust"
	case ".java":
		return "Java"
	case ".kt", ".kts":
		return "Kotlin"
	case ".swift":
		return "Swift"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".c", ".h":
		return "C"
	case ".cc", ".cpp", ".cxx", ".hpp", ".hh":
		return "C++"
	case ".cs":
		return "C#"
	case ".sh", ".bash", ".zsh":
		return "Shell"
	case ".yml", ".yaml":
		return "YAML"
	case ".xml":
		return "XML"
	case ".toml":
		return "TOML"
	case ".ini":
		return "INI"
	}

	switch strings.ToLower(filepath.Base(path)) {
	case "dockerfile":
		return "Dockerfile"
	case "makefile":
		return "Makefile"
	}
	return "Other"
}
