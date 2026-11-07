package ui

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// capture redirects output to a buffer and input to the given string, resetting
// separator state, and returns the buffer.
func capture(t *testing.T, input string) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	SetOutput(buf)
	SetInput(strings.NewReader(input))
	SetColorResolver(nil)
	t.Cleanup(func() { SetColorResolver(nil) })
	return buf
}

func TestReadSecret_UsesSubstitutedReaderWhenOverridden(t *testing.T) {
	capture(t, "sekret-token\n")

	got, err := ReadSecret("token: ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "sekret-token" {
		t.Errorf("got %q, want %q", got, "sekret-token")
	}
}

func TestInteractive_ForcedNonInteractiveOverridesTestInputOverride(t *testing.T) {
	capture(t, "") // SetInput makes interactiveOverride true
	SetNonInteractive(true)
	t.Cleanup(func() { SetNonInteractive(false) })

	if Interactive() {
		t.Error("expected SetNonInteractive(true) to force Interactive() false even with input redirected")
	}
}

func TestInteractive_NotForcedLeavesTestInputOverrideAlone(t *testing.T) {
	capture(t, "")
	SetNonInteractive(false)
	t.Cleanup(func() { SetNonInteractive(false) })

	if !Interactive() {
		t.Error("expected the test input override to still report interactive when non-interactive isn't forced")
	}
}

func TestBreak_NoLeadingBlank(t *testing.T) {
	buf := capture(t, "")
	Break() // nothing emitted yet
	Out("first")
	if got := buf.String(); got != "first\n" {
		t.Errorf("got %q, want %q", got, "first\n")
	}
}

func TestBreak_SingleBlankBetweenBlocks(t *testing.T) {
	buf := capture(t, "")
	Out("a")
	Break()
	Out("b")
	if got := buf.String(); got != "a\n\nb\n" {
		t.Errorf("got %q, want %q", got, "a\n\nb\n")
	}
}

func TestBreak_Idempotent(t *testing.T) {
	buf := capture(t, "")
	Out("a")
	Break()
	Break()
	Break()
	Out("b")
	if got := buf.String(); got != "a\n\nb\n" {
		t.Errorf("consecutive Breaks must collapse: got %q, want %q", got, "a\n\nb\n")
	}
}

func TestBreak_NoTrailingBlank(t *testing.T) {
	buf := capture(t, "")
	Out("a")
	Break() // nothing follows
	if got := buf.String(); got != "a\n" {
		t.Errorf("got %q, want %q (no trailing blank)", got, "a\n")
	}
}

// TestBreak_LoopWithEmptyIteration reproduces the trap that made the old eager
// sep() fragile: a Break at the top of every iteration, where one iteration
// prints nothing, must not yield a double blank or a stray leading blank.
func TestBreak_LoopWithEmptyIteration(t *testing.T) {
	buf := capture(t, "")
	items := []string{"a", "", "b"} // middle iteration prints nothing
	for _, it := range items {
		Break()
		if it != "" {
			Out("%s", it)
		}
	}
	if got := buf.String(); got != "a\n\nb\n" {
		t.Errorf("got %q, want %q", got, "a\n\nb\n")
	}
}

// TestResolveLoop_BlankAfterPrompt exercises the deferred-Break primitives
// directly: the next package's block is a step of its own, so its Break
// yields the blank after the answer. Within one step the output after a
// prompt runs tight (see TestPrompt_BlankBeforeOnly).
func TestResolveLoop_BlankAfterPrompt(t *testing.T) {
	buf := capture(t, "1\n")
	// package 1: header, then a menu + read
	Break()
	Out("pkg1: repo -> x")
	Break()
	Out("choose asset(s):")
	Out("  1) a")
	_ = ReadLine("enter number: ")
	// package 2: header
	Break()
	Out("pkg2: repo -> y")

	// The "\n" after "enter number: " is the deferred Break flushing. On a real
	// terminal the user's echoed Enter terminates the prompt line and this "\n"
	// renders as the blank line separating it from the next package block.
	want := "pkg1: repo -> x\n\nchoose asset(s):\n  1) a\nenter number: \npkg2: repo -> y\n"
	if got := buf.String(); got != want {
		t.Errorf("missing blank before next package block:\n got %q\nwant %q", got, want)
	}
}

// TestPrompt_BlankBeforeOnly: exactly one blank before a prompt, and the
// output it lets through on the line right below the answer.
func TestPrompt_BlankBeforeOnly(t *testing.T) {
	buf := capture(t, "1\n")
	Out("before")
	_, _ = Prompt(func() (int, error) {
		Menu("", "choose x:", []string{"a", "b"})
		return ReadSingle("enter number")
	})
	Out("after")
	want := "before\n\nchoose x:\n  1) a\n  2) b\nenter number [empty=1] (0=skip): after\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestPrompt_NoBlankAtEdges(t *testing.T) {
	buf := capture(t, "\n")
	// The leading Break is a no-op with nothing emitted yet, and nothing
	// follows the answer, so a prompt alone produces no stray blank.
	_, _ = Prompt(func() (string, error) { return ReadLine("p: "), nil })
	if got := buf.String(); got != "p: " {
		t.Errorf("got %q, want %q", got, "p: ")
	}
}

// TestPrompt_OneBlankBeforeTheNext: a second prompt right after the first
// gets the one blank every prompt does, and never two.
func TestPrompt_OneBlankBeforeTheNext(t *testing.T) {
	buf := capture(t, "\n\n")
	Out("a")
	_, _ = Prompt(func() (string, error) { return ReadLine("p1: "), nil })
	_, _ = Prompt(func() (string, error) { return ReadLine("p2: "), nil })
	Out("b")
	// On a terminal the answer's own newline ends each prompt line, so the
	// "\n" before p2 is its one blank, and b runs tight under p2's answer.
	want := "a\n\np1: \np2: b\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

// TestMenu_ShownAgainGetsItsOneBlank: a menu shown again inside one Prompt,
// as "show more" does, is a new prompt, so it gets the one blank every prompt
// does; the first menu shares Prompt's, so it is not doubled.
func TestMenu_ShownAgainGetsItsOneBlank(t *testing.T) {
	buf := capture(t, "3\n1\n")
	Out("before")
	_, _ = Prompt(func() (int, error) {
		Menu("uv", "choose bin(s)", []string{"uv", "uvx", "show more"})
		_, _ = ReadSingle("enter number")
		Menu("uv", "choose bin(s)", []string{"uv", "uvx", "uv-toolchain"})
		return ReadSingle("enter number")
	})
	Out("after")
	want := "before\n\nuv: choose bin(s)\n  1) uv\n  2) uvx\n  3) show more\nenter number [empty=1] (0=skip): " +
		"\nuv: choose bin(s)\n  1) uv\n  2) uvx\n  3) uv-toolchain\nenter number [empty=1] (0=skip): after\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestMenu_LabelAndNumbering(t *testing.T) {
	buf := capture(t, "")
	Menu("codex", "choose bin(s)", []string{"a", "b"})
	want := "codex: choose bin(s)\n  1) a\n  2) b\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestConfirm_FlushesPendingBreakBeforePrompt(t *testing.T) {
	buf := capture(t, "y\n")
	Out("a")
	if !Confirm("proceed") {
		t.Fatal("expected true for y")
	}
	if got := buf.String(); got != "a\n\nproceed [[y]/n]: " {
		t.Errorf("got %q", got)
	}
}

// TestConfirm_NoTrailingBlank asserts a confirm nests its follow-on output
// tight: a leading blank before the prompt, but none between the answer and the
// next line (the operation the user opted into).
func TestConfirm_NoTrailingBlank(t *testing.T) {
	buf := capture(t, "y\n")
	Out("a")
	Confirm("proceed")
	Out("running")
	if got := buf.String(); got != "a\n\nproceed [[y]/n]: running\n" {
		t.Errorf("got %q", got)
	}
}

func TestConfirm_Answers(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool
	}{{"\n", true}, {"y\n", true}, {"yes\n", true}, {"Y\n", true}, {"n\n", false}, {"no\n", false}} {
		capture(t, c.in)
		if got := Confirm("ok"); got != c.want {
			t.Errorf("Confirm(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestConfirm_NobodyToAnswer asserts a confirm with nobody there is no, whatever
// the read would have returned: an EOF reads as empty, and empty is yes.
func TestConfirm_NobodyToAnswer(t *testing.T) {
	buf := capture(t, "")
	SetNonInteractive(true)
	t.Cleanup(func() { SetNonInteractive(false) })
	if Confirm("remove 3 broken install(s)") {
		t.Fatal("Confirm with nobody to answer = true, want false")
	}
	if got := buf.String(); !strings.Contains(got, "no terminal to answer from") || strings.Contains(got, "[[y]/n]") {
		t.Errorf("got %q", got)
	}
}

func TestTable_SeparatedAsBlock(t *testing.T) {
	buf := capture(t, "")
	Out("before")
	Table([]string{"name", "n"}, [][]string{{"fzf", "1"}}, nil)
	want := "before\n\nname n\n---- -\nfzf  1\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
}

func TestTable_NoLeadingBlankWhenFirst(t *testing.T) {
	buf := capture(t, "")
	Table([]string{"name"}, [][]string{{"fzf"}}, nil)
	if got := buf.String(); strings.HasPrefix(got, "\n") {
		t.Errorf("table as first output must not start with a blank: got %q", got)
	}
}

func TestReadSingle_Valid(t *testing.T) {
	capture(t, "2\n")
	idx, err := ReadSingle("pick one")
	if err != nil || idx != 2 {
		t.Errorf("got %d, %v; want 2, nil", idx, err)
	}
}

func TestReadSingle_Zero_Skips(t *testing.T) {
	capture(t, "0\n")
	if _, err := ReadSingle("pick one"); !errors.Is(err, ErrSkip) {
		t.Errorf("expected ErrSkip, got %v", err)
	}
}

func TestReadSingle_Invalid(t *testing.T) {
	capture(t, "abc\n")
	if _, err := ReadSingle("pick one"); !errors.Is(err, ErrSkip) {
		t.Errorf("expected ErrSkip for invalid input, got %v", err)
	}
}

func TestReadSingle_MultipleNumbers_Skips(t *testing.T) {
	// "2 5" is multi-select syntax in a single-select prompt: reject the whole
	// line rather than silently salvaging the leading 2.
	capture(t, "2 5\n")
	if _, err := ReadSingle("pick one"); !errors.Is(err, ErrSkip) {
		t.Errorf("expected ErrSkip for multi-number input, got %v", err)
	}
}

func TestReadSingle_TrailingGarbage_Skips(t *testing.T) {
	capture(t, "2x\n")
	if _, err := ReadSingle("pick one"); !errors.Is(err, ErrSkip) {
		t.Errorf("expected ErrSkip for trailing garbage, got %v", err)
	}
}

func TestReadSingle_Empty_SelectsFirst(t *testing.T) {
	capture(t, "\n")
	idx, err := ReadSingle("pick one")
	if err != nil || idx != 1 {
		t.Errorf("got %d, %v; want 1, nil", idx, err)
	}
}

func TestDecorated_UsesColorResolver(t *testing.T) {
	buf := capture(t, "")
	SetColorResolver(func(role string) func(string) string {
		if role == "warn" {
			return func(s string) string { return "<" + s + ">" }
		}
		return nil
	})
	Warn("hello %s", "world")
	if got := buf.String(); got != "<‼ hello world>\n" {
		t.Errorf("got %q", got)
	}
}

func TestVerbose_OnlyUnderVerboseAndOnlyToErrOutput(t *testing.T) {
	buf := capture(t, "")
	errBuf := &bytes.Buffer{}
	SetErrOutput(errBuf)
	t.Cleanup(func() { SetErrOutput(os.Stderr); SetVerbose(false) })

	Verbose("hidden")
	SetVerbose(true)
	Verbose("shown %d", 1)

	if errBuf.String() != "shown 1\n" {
		t.Errorf("err output = %q, want only the line written under verbose", errBuf.String())
	}
	if buf.Len() != 0 {
		t.Errorf("output = %q, want nothing on stdout", buf.String())
	}
}
