package skillagent

import (
	"errors"
	"testing"
)

func TestSkillErrorWrapsCause(t *testing.T) {
	cause := errors.New("load failed")
	err := NewSkillError("review-skill", "activate", cause)

	if !errors.Is(err, cause) {
		t.Fatal("Expected SkillError to unwrap the original cause")
	}
	want := `skill "review-skill": activate: load failed`
	if err.Error() != want {
		t.Fatalf("Expected %q, got %q", want, err.Error())
	}
}

func TestSkillErrorWithoutSkillName(t *testing.T) {
	err := NewSkillError("", "discover", ErrNoSkillsDiscovered)
	want := "discover: no skills discovered"

	if err.Error() != want {
		t.Fatalf("Expected %q, got %q", want, err.Error())
	}
	if !errors.Is(err, ErrNoSkillsDiscovered) {
		t.Fatal("Expected sentinel error to be preserved")
	}
}

func TestParseErrorFormatsLineAndUnwrapsCause(t *testing.T) {
	cause := errors.New("yaml syntax")
	err := NewParseError("SKILL.md", 7, "invalid frontmatter", cause)
	want := "parse error at SKILL.md:7: invalid frontmatter"

	if err.Error() != want {
		t.Fatalf("Expected %q, got %q", want, err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("Expected ParseError to unwrap the original cause")
	}
}

func TestValidationErrorFormat(t *testing.T) {
	err := NewValidationError("name", "is required")
	want := `validation error: field "name": is required`

	if err.Error() != want {
		t.Fatalf("Expected %q, got %q", want, err.Error())
	}
}
