package projections

import (
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

func collectPatchStats(tools []sessionparse.ToolEvent) patchStats {
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
