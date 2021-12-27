package util

import (
	"log"
	"testing"
)

func TestTruncateAt(t *testing.T) {
	{
		testcase := "regular string truncation"
		input := "aaaaabbbbbcccccddddd"
		truncLen := 15

		expected := "aaaaabbbbbcc..."
		result := TruncateAt(input, truncLen)

		if result != expected {
			log.Fatalf("Truncation test case '%s' failed:\n  expected: '%s'\n  got: '%s'",
				testcase, expected, result)
		}
	}
	{
		testcase := "no truncation needed"
		input := "aaaaabbbbbcccccddddd"
		truncLen := 40

		expected := input
		result := TruncateAt(input, truncLen)

		if result != expected {
			log.Fatalf("Truncation test case '%s' failed:\n  expected: '%s'\n  got: '%s'",
				testcase, expected, result)
		}
	}
	{
		testcase := "just barely no truncation needed"
		input := "aaaaabbbbbcccccddddd"
		truncLen := 20

		expected := input
		result := TruncateAt(input, truncLen)

		if result != expected {
			log.Fatalf("Truncation test case '%s' failed:\n  expected: '%s'\n  got: '%s'",
				testcase, expected, result)
		}
	}
	{
		testcase := "just barely truncation needed"
		input := "aaaaabbbbbcccccddddd"
		truncLen := 19

		expected := "aaaaabbbbbcccccd..."
		result := TruncateAt(input, truncLen)

		if result != expected {
			log.Fatalf("Truncation test case '%s' failed:\n  expected: '%s'\n  got: '%s'",
				testcase, expected, result)
		}
	}
	{
		testcase := "only truncation remaining"
		input := "aaaaabbbbbcccccddddd"
		truncLen := 3

		expected := "..."
		result := TruncateAt(input, truncLen)

		if result != expected {
			log.Fatalf("Truncation test case '%s' failed:\n  expected: '%s'\n  got: '%s'",
				testcase, expected, result)
		}
	}
	{
		testcase := "only truncation remaining (1)"
		input := "aaaaabbbbbcccccddddd"
		truncLen := 1

		expected := "."
		result := TruncateAt(input, truncLen)

		if result != expected {
			log.Fatalf("Truncation test case '%s' failed:\n  expected: '%s'\n  got: '%s'",
				testcase, expected, result)
		}
	}
	{
		testcase := "truncate to zero"
		input := "aaaaabbbbbcccccddddd"
		truncLen := 0

		expected := ""
		result := TruncateAt(input, truncLen)

		if result != expected {
			log.Fatalf("Truncation test case '%s' failed:\n  expected: '%s'\n  got: '%s'",
				testcase, expected, result)
		}
	}
	{
		testcase := "truncate to negative"
		input := "aaaaabbbbbcccccddddd"
		truncLen := -4

		expected := ""
		result := TruncateAt(input, truncLen)

		if result != expected {
			log.Fatalf("Truncation test case '%s' failed:\n  expected: '%s'\n  got: '%s'",
				testcase, expected, result)
		}
	}
	{
		testcase := "truncate non-ascii"
		input := "🙈🙈🙈🙈|🙉🙉🙉🙉|🙊🙊🙊🙊"
		truncLen := 10

		expected := "🙈🙈🙈🙈|🙉🙉..."
		result := TruncateAt(input, truncLen)

		if result != expected {
			log.Fatalf("Truncation test case '%s' failed:\n  expected: '%s'\n  got: '%s'",
				testcase, expected, result)
		}
	}
}
