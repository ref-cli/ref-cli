package example

import (
	"testing"
)

func TestParseBasic(t *testing.T) {
	content := "# To extract an archive:\ntar -xvf foo.tar\n\n# To create an archive:\ntar -czf foo.tar.gz foo/\n"
	ex := Parse("tar", content)
	if len(ex.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(ex.Entries))
	}
	if ex.Entries[0].Comment != "To extract an archive:" {
		t.Errorf("unexpected comment: %q", ex.Entries[0].Comment)
	}
	if ex.Entries[0].Command != "tar -xvf foo.tar" {
		t.Errorf("unexpected command: %q", ex.Entries[0].Command)
	}
}

func TestParseFrontmatter(t *testing.T) {
	content := "---\ntags: [ compression ]\nvariants: [ bsd, gnu ]\n---\n# To extract:\ntar -xvf foo.tar\n"
	ex := Parse("tar", content)
	if len(ex.Frontmatter.Tags) != 1 || ex.Frontmatter.Tags[0] != "compression" {
		t.Errorf("unexpected tags: %v", ex.Frontmatter.Tags)
	}
	if len(ex.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(ex.Entries))
	}
}

func TestParseInlineTags(t *testing.T) {
	content := "# [GNU] To use Perl regex:\ngrep -P '\\d+' file\n"
	ex := Parse("grep", content)
	if len(ex.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(ex.Entries))
	}
	e := ex.Entries[0]
	if len(e.Tags) != 1 || e.Tags[0] != "GNU" {
		t.Errorf("unexpected tags: %v", e.Tags)
	}
	if e.Comment != "To use Perl regex:" {
		t.Errorf("unexpected comment: %q", e.Comment)
	}
}

func TestParseEmpty(t *testing.T) {
	ex := Parse("cmd", "")
	if len(ex.Entries) != 0 {
		t.Errorf("expected 0 entries for empty content, got %d", len(ex.Entries))
	}
}

func TestParseCommentWithoutCommand(t *testing.T) {
	ex := Parse("cmd", "# Orphan comment\n")
	if len(ex.Entries) != 0 {
		t.Errorf("expected 0 entries for bare comment, got %d", len(ex.Entries))
	}
}

func TestParseCommandWithoutComment(t *testing.T) {
	ex := Parse("cmd", "some-command\n")
	if len(ex.Entries) != 0 {
		t.Errorf("expected 0 entries for command without comment, got %d", len(ex.Entries))
	}
}

func TestParseMultipleSequentialComments(t *testing.T) {
	// Only the last comment before a command is kept.
	content := "# First comment\n# Second comment\nsome-command\n"
	ex := Parse("cmd", content)
	if len(ex.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(ex.Entries))
	}
	if ex.Entries[0].Comment != "Second comment" {
		t.Errorf("expected 'Second comment', got %q", ex.Entries[0].Comment)
	}
}

func TestParseFrontmatterEmptyBody(t *testing.T) {
	content := "---\ntags: [ foo ]\n---\n"
	ex := Parse("cmd", content)
	if len(ex.Entries) != 0 {
		t.Errorf("expected 0 entries for empty body, got %d", len(ex.Entries))
	}
	if len(ex.Frontmatter.Tags) != 1 || ex.Frontmatter.Tags[0] != "foo" {
		t.Errorf("frontmatter not parsed: %v", ex.Frontmatter.Tags)
	}
}
