package colors

import (
	"log"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestColorFromHexString(t *testing.T) {
	{
		testcase := "pure red"

		input := "ff0000"
		expected := tcell.NewHexColor(0xff0000)
		result := ColorFromHexString(input)
		if result != expected {
			log.Fatalf("colors testcase '%s' failed: 0x%06x instead of 0x%06x", testcase, result.Hex(), expected.Hex())
		}
	}
	{
		testcase := "pure green"

		input := "00ff00"
		expected := tcell.NewHexColor(0x00ff00)
		result := ColorFromHexString(input)
		if result != expected {
			log.Fatalf("colors testcase '%s' failed: 0x%06x instead of 0x%06x", testcase, result.Hex(), expected.Hex())
		}
	}
	{
		testcase := "pure blue"

		input := "0000ff"
		expected := tcell.NewHexColor(0x0000ff)
		result := ColorFromHexString(input)
		if result != expected {
			log.Fatalf("colors testcase '%s' failed: 0x%06x instead of 0x%06x", testcase, result.Hex(), expected.Hex())
		}
	}
	{
		testcase := "regular color"

		input := "123456"
		expected := tcell.NewHexColor(0x123456)
		result := ColorFromHexString(input)
		if result != expected {
			log.Fatalf("colors testcase '%s' failed: 0x%06x instead of 0x%06x", testcase, result.Hex(), expected.Hex())
		}
	}
}

func TestDarken(t *testing.T) {
	{
		testcase := "100% -> no change"

		input := tcell.NewHexColor(0x123456)

		expected := input
		result := Darken(input, 100)

		if result != expected {
			log.Fatalf("colors testcase '%s' failed: 0x%06x instead of 0x%06x", testcase, result.Hex(), expected.Hex())
		}
	}
	{
		testcase := "0% -> black"

		input := tcell.NewHexColor(0x123456)

		expected := tcell.NewHexColor(0x000000)
		result := Darken(input, 0)

		if result != expected {
			log.Fatalf("colors testcase '%s' failed: 0x%06x instead of 0x%06x", testcase, result.Hex(), expected.Hex())
		}
	}
	{
		testcase := "50% -> dimmed by half"

		input := tcell.NewHexColor(0x204060)

		expected := tcell.NewHexColor(0x102030)
		result := Darken(input, 50)

		if result != expected {
			log.Fatalf("colors testcase '%s' failed: 0x%06x instead of 0x%06x", testcase, result.Hex(), expected.Hex())
		}
	}
}
