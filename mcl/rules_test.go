package mcl

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile creates a file with the given content, including its parent
// directories.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestRuleAllows(t *testing.T) {
	tests := []struct {
		name string
		rule Rule
		opts Options
		want bool
	}{
		{
			name: "allow without constraints",
			rule: Rule{Action: "allow"},
			want: true,
		},
		{
			name: "disallow without constraints",
			rule: Rule{Action: "disallow"},
			want: false,
		},
		{
			name: "allow for another os",
			rule: Rule{Action: "allow", OS: &OSRule{Name: otherOSName()}},
			want: false,
		},
		{
			name: "disallow for another os",
			rule: Rule{Action: "disallow", OS: &OSRule{Name: otherOSName()}},
			want: true,
		},
		{
			name: "unknown os names are ignored",
			rule: Rule{Action: "allow", OS: &OSRule{Name: "plan9"}},
			want: true,
		},
		{
			name: "allow for the current os",
			rule: Rule{Action: "allow", OS: &OSRule{Name: currentOSName()}},
			want: true,
		},
		{
			name: "feature is not enabled",
			rule: Rule{Action: "allow", Features: map[string]bool{"has_custom_resolution": true}},
			opts: Options{},
			want: false,
		},
		{
			name: "feature is enabled",
			rule: Rule{Action: "allow", Features: map[string]bool{"has_custom_resolution": true}},
			opts: Options{CustomResolution: true},
			want: true,
		},
		{
			name: "demo user feature",
			rule: Rule{Action: "allow", Features: map[string]bool{"is_demo_user": true}},
			opts: Options{Demo: true},
			want: true,
		},
		{
			name: "quick play feature",
			rule: Rule{Action: "allow", Features: map[string]bool{"is_quick_play_singleplayer": true}},
			opts: Options{QuickPlaySingleplayer: "world"},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ruleAllows(test.rule, test.opts); got != test.want {
				t.Errorf("ruleAllows() = %v, want %v", got, test.want)
			}
		})
	}
}

// otherOSName returns the name of an operating system that is not the one
// the test runs on.
func otherOSName() string {
	if currentOSName() == "linux" {
		return "windows"
	}
	return "linux"
}

func TestRuleListAllows(t *testing.T) {
	allow := Rule{Action: "allow"}
	deny := Rule{Action: "disallow", OS: &OSRule{Name: currentOSName()}}

	if !ruleListAllows([]Rule{allow}, Options{}) {
		t.Error("a single allowing rule should allow")
	}
	if ruleListAllows([]Rule{allow, deny}, Options{}) {
		t.Error("a single denying rule should deny")
	}
	if !ruleListAllows(nil, Options{}) {
		t.Error("an empty rule list should allow")
	}
}
