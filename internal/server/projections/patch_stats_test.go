package projections

import (
	"strings"
	"testing"

	"github.com/ilcm96/codex-usage/internal/server/sessionparse"
)

func TestCountPatchAddedLines(t *testing.T) {
	tools := []sessionparse.ToolEvent{
		{
			Kind:     "custom_tool_call",
			ToolName: "apply_patch",
			InputText: strings.Join([]string{
				"*** Begin Patch",
				"*** Add File: main.go",
				"+package main",
				"+",
				"+func main() {}",
				"*** Update File: README.md",
				"@@",
				"-old",
				"+new",
				"+++ b/not-counted",
				"*** End Patch",
			}, "\n"),
		},
		{
			Kind:      "event_msg_patch_apply",
			InputText: "+tracked by event_msg\n context\n",
		},
		{
			Kind:      "function_call",
			ToolName:  "exec_command",
			InputText: "+not a patch",
		},
	}

	if got := countPatchAddedLines(tools); got != 5 {
		t.Fatalf("countPatchAddedLines() = %d, want 5", got)
	}
}

func TestCollectPatchStatsGroupsAddedLinesByLanguage(t *testing.T) {
	tools := []sessionparse.ToolEvent{
		{
			Kind:     "custom_tool_call",
			ToolName: "apply_patch",
			InputText: strings.Join([]string{
				"*** Begin Patch",
				"*** Add File: internal/server/main.go",
				"+package main",
				"+func main() {}",
				"*** Update File: web/src/App.tsx",
				"@@",
				"+export function App() {",
				"+  return null;",
				"+}",
				"*** Update File: README.md",
				"+# Notes",
				"*** End Patch",
			}, "\n"),
		},
	}

	got := collectPatchStats(tools)
	if got.AddedLines != 6 {
		t.Fatalf("AddedLines = %d, want 6", got.AddedLines)
	}
	if got.LanguageLines["Go"] != 2 {
		t.Fatalf("Go lines = %d, want 2", got.LanguageLines["Go"])
	}
	if got.LanguageLines["TypeScript"] != 3 {
		t.Fatalf("TypeScript lines = %d, want 3", got.LanguageLines["TypeScript"])
	}
	if got.LanguageLines["Markdown"] != 1 {
		t.Fatalf("Markdown lines = %d, want 1", got.LanguageLines["Markdown"])
	}
	if got.DominantLanguage != "TypeScript" {
		t.Fatalf("DominantLanguage = %q, want TypeScript", got.DominantLanguage)
	}
}

func TestCollectPatchStatsPrefersAppliedChanges(t *testing.T) {
	tools := []sessionparse.ToolEvent{
		{
			Kind:     "custom_tool_call",
			ToolName: "apply_patch",
			InputText: strings.Join([]string{
				"*** Begin Patch",
				"*** Add File: ignored.go",
				"+this attempted patch must not be counted twice",
				"*** End Patch",
			}, "\n"),
		},
		{
			Kind: "patch_apply_end",
			PayloadJSON: []byte(`{
				"type":"patch_apply_end",
				"success":true,
				"changes":{
					"internal/server/main.go":{"type":"add","content":"package main\n\nfunc main() {}\n"},
					"web/src/App.tsx":{"type":"update","move_path":null,"unified_diff":"@@\n-old\n+new\n+next\n"},
					"README.md":{"type":"delete","content":"removed\n"}
				}
			}`),
		},
	}

	got := collectPatchStats(tools)
	if got.AddedLines != 5 {
		t.Fatalf("AddedLines = %d, want 5", got.AddedLines)
	}
	if got.LanguageLines["Go"] != 3 {
		t.Fatalf("Go lines = %d, want 3", got.LanguageLines["Go"])
	}
	if got.LanguageLines["TypeScript"] != 2 {
		t.Fatalf("TypeScript lines = %d, want 2", got.LanguageLines["TypeScript"])
	}
	if got.LanguageLines["Markdown"] != 0 {
		t.Fatalf("Markdown lines = %d, want 0", got.LanguageLines["Markdown"])
	}
	if got.DominantLanguage != "Go" {
		t.Fatalf("DominantLanguage = %q, want Go", got.DominantLanguage)
	}
}

func TestCollectPatchStatsUsesMoveDestinationLanguage(t *testing.T) {
	tools := []sessionparse.ToolEvent{
		{
			Kind: "patch_apply_end",
			PayloadJSON: []byte(`{
				"changes":{
					"script.js":{"type":"update","move_path":"script.ts","unified_diff":"@@\n+const value: string = 'moved';\n"}
				}
			}`),
		},
	}

	got := collectPatchStats(tools)
	if got.AddedLines != 1 || got.LanguageLines["TypeScript"] != 1 {
		t.Fatalf("unexpected moved patch stats: %+v", got)
	}
}

func TestCollectPatchStatsDoesNotCountFailedAppliedPatch(t *testing.T) {
	tools := []sessionparse.ToolEvent{
		{
			Kind:      "custom_tool_call",
			ToolName:  "apply_patch",
			InputText: "*** Begin Patch\n*** Add File: failed.go\n+not applied\n*** End Patch\n",
		},
		{
			Kind:        "patch_apply_end",
			PayloadJSON: []byte(`{"type":"patch_apply_end","success":false,"changes":{}}`),
		},
	}

	got := collectPatchStats(tools)
	if got.AddedLines != 0 || len(got.LanguageLines) != 0 {
		t.Fatalf("failed patch stats = %+v, want zero", got)
	}
}
