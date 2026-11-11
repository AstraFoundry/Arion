package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/egoist/mygo/internal/automationcli"
)

// TestSkills checks that skills prints SKILL.md as it is, and that the
// guide tells of every command.
func TestSkills(t *testing.T) {
	var out, errs bytes.Buffer
	if code := automationcli.Run([]string{"skills"}, nil, &out, &errs, automationcli.Options{Skill: skill}); code != 0 || errs.Len() > 0 {
		t.Fatalf("skills: exit %d: %s", code, errs.String())
	}
	file, err := os.ReadFile("SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), file) {
		t.Error("skills does not print SKILL.md as it is")
	}
	if !bytes.HasPrefix(file, []byte("---\nname: mygo-automation\ndescription: ")) {
		t.Error("SKILL.md lacks its front matter")
	}
	for _, name := range automationcli.Commands() {
		if !strings.Contains(string(file), "`"+name+" ") && !strings.Contains(string(file), "`"+name+"`") {
			t.Errorf("SKILL.md does not tell of %s", name)
		}
	}
}
