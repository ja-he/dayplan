package styling

import (
	"log"
	"testing"

	"github.com/lucasb-eyer/go-colorful"
)

func Foo(t *testing.T) {
	l := getLuminance(colorfulColorFromHexString("#000000"))
	if l != 0.0 {
		t.Fatalf("lum %f != 0", l)
	}
}

func TestLighten(t *testing.T) {
	{
		testcase := "0% -> no change"

		input := colorful.Color{
			R: float64(0x12) / 255.0,
			G: float64(0x34) / 255.0,
			B: float64(0x56) / 255.0,
		}

		expected := input
		result := lightenColorfulColor(input, 0)

		if !result.AlmostEqualRgb(expected) {
			log.Fatalf("colors testcase '%s' failed: %s instead of %s", testcase, result.Hex(), expected.Hex())
		}
	}
	{
		testcase := "100% -> white"

		input := colorful.Color{
			R: float64(0x12) / 255.0,
			G: float64(0x34) / 255.0,
			B: float64(0x56) / 255.0,
		}

		expected := colorful.Color{
			R: 1.0,
			G: 1.0,
			B: 1.0,
		}
		result := lightenColorfulColor(input, 100)

		if !result.AlmostEqualRgb(expected) {
			log.Fatalf("colors testcase '%s' failed: %s instead of %s", testcase, result.Hex(), expected.Hex())
		}
	}
	{
		testcase := "50% -> 50% lighter"

		input := colorful.Color{
			R: float64(0x80) / 255.0,
			G: float64(0x80) / 255.0,
			B: float64(0x80) / 255.0,
		}

		expected := colorful.Color{
			R: float64(0xc0) / 255.0,
			G: float64(0xc0) / 255.0,
			B: float64(0xc0) / 255.0,
		}
		result := lightenColorfulColor(input, 50)

		if !result.AlmostEqualRgb(expected) {
			log.Fatalf("colors testcase '%s' failed: %s instead of %s", testcase, result.Hex(), expected.Hex())
		}
	}
	{
		testcase := "75% lighter <=> 50% lighter then 50% ligher again"

		input := colorful.Color{
			R: float64(0x80) / 255.0,
			G: float64(0x80) / 255.0,
			B: float64(0x80) / 255.0,
		}

		a := lightenColorfulColor(input, 75)
		b := lightenColorfulColor(lightenColorfulColor(input, 50), 50)

		if !a.AlmostEqualRgb(b) {
			log.Fatalf("colors testcase '%s' failed: 0x%06x != 0x%06x (dist: %f)", testcase, a.Hex(), b.Hex(), a.DistanceRgb(b))
		}
	}
}
