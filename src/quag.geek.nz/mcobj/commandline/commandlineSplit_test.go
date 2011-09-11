package commandline

import (
	"testing"
)

func TestSingleArg(t *testing.T) {
	args := SplitCommandLine("abc")
	checkArgs(t, args, "abc")
}

func TestMultipleArgs(t *testing.T) {
	args := SplitCommandLine("abc def ghi jkl mno")
	checkArgs(t, args, "abc", "def", "ghi", "jkl", "mno")
}

func TestMultipleSpacesBetweenArgs(t *testing.T) {
	args := SplitCommandLine("  abc  def   ghi  ")
	checkArgs(t, args, "abc", "def", "ghi")
}

func TestQuotedSpaces(t *testing.T) {
	checkArgs(t, SplitCommandLine("\"a b c\""), "a b c")
	checkArgs(t, SplitCommandLine("'a b c'"), "a b c")
}

func TestQuoteWithinArg(t *testing.T) {
	checkArgs(t, SplitCommandLine("abc\"def\"hij"), "abcdefhij")
	checkArgs(t, SplitCommandLine("abc'def'hij"), "abcdefhij")
}

func TestEmptyQuoteArg(t *testing.T) {
	checkArgs(t, SplitCommandLine("\"\""), "")
	checkArgs(t, SplitCommandLine("''"), "")
}

func TestUnterminatedQuote(t *testing.T) {
	checkArgs(t, SplitCommandLine("\""), "")
	checkArgs(t, SplitCommandLine("'"), "")
}

func TestSlashSpace(t *testing.T) {
	checkArgs(t, SplitCommandLine("a\\ b"), "a b")
}

func TestSlashQuote(t *testing.T) {
	checkArgs(t, SplitCommandLine("a\\\"b"), "a\\\"b")
	checkArgs(t, SplitCommandLine("a\\\"b"), "a\\\"b")
}

func TestSlashQuoteInQuote(t *testing.T) {
	checkArgs(t, SplitCommandLine("\"a\\\"b\""), "a\\\"b")
	checkArgs(t, SplitCommandLine("'a\\\"b'"), "a\\\"b")
	checkArgs(t, SplitCommandLine("\"a\\'b\""), "a\\'b")
	checkArgs(t, SplitCommandLine("'a\\'b'"), "a\\'b")
}

func TestSlashSpaceInQuote(t *testing.T) {
	checkArgs(t, SplitCommandLine("\"a\\ b\""), "a b")
	checkArgs(t, SplitCommandLine("'a\\ b'"), "a b")
}

// TODO: correct for spaces in paths

func checkArgs(t *testing.T, args []string, expectedArgs ...string) {
	t.Logf("args: %q expected: %q", args, expectedArgs)
	checkArgCount(t, args, len(expectedArgs))
	for i, value := range expectedArgs {
		checkArg(t, args, i, value)
	}
}

func checkArgCount(t *testing.T, args []string, n int) {
	if len(args) != n {
		t.Errorf("Arg count %d instead of %d", len(args), n)
	}
}

func checkArg(t *testing.T, args []string, i int, value string) {
	if i >= len(args) {
		t.Errorf("args[%d] missing (should have been %q)", i, value)
	} else if args[i] != value {
		t.Errorf("args[%d] %q instead of %q", i, args[i], value)
	}
}
