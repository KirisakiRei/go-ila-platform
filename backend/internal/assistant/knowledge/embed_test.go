package knowledge

import (
	"strings"
	"testing"
)

func TestBaseSystemPromptLoadsAllFiles(t *testing.T) {
	prompt, err := BaseSystemPrompt()
	if err != nil {
		t.Fatalf("BaseSystemPrompt error: %v", err)
	}
	required := []string{
		"SOUL.md",
		"SKILLS.md",
		"MEMORY.md",
		"ila2/README.md",
		"ila2/key-level-irl.md",
		"ila2/key-level-erl.md",
		"ila2/c2-closure-irl.md",
		"ila2/c2-closure-erl.md",
		"ila2/cisd.md",
		"ila2/time-cycle.md",
		"ila2/alur-entry.md",
		"ila2/diagram.md",
	}
	for _, marker := range required {
		if !strings.Contains(prompt, "<!-- FILE: "+marker+" -->") {
			t.Fatalf("dokumen %s tidak termuat di BaseSystemPrompt", marker)
		}
	}
}
