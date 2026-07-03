package skillrepairprompt

import "testing"

func TestBuild(t *testing.T) {
	got, err := Build("/tmp/demo/SKILL.md")
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	want := "参考[https://agentskills.io/specification]修复/tmp/demo/SKILL.md的错误。"
	if got != want {
		t.Fatalf("Build = %q, want %q", got, want)
	}
}

func TestNormalizeSkillPathRejectsInvalidInput(t *testing.T) {
	cases := []string{
		"",
		"relative/SKILL.md",
		"/tmp/demo/README.md",
	}
	for _, tc := range cases {
		if _, err := NormalizeSkillPath(tc); err == nil {
			t.Fatalf("NormalizeSkillPath(%q) expected error", tc)
		}
	}
}
