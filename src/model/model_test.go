package model

import (
	"log"
	"testing"
)

func TestStartsDuring(t *testing.T) {
	{
		testcase := "starts during"

		// +-----+
		// | a +---+
		// |   | b |
		// +---|   |
		//     +---+

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("06:00|07:30|work|Get Started")

		expected := true
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts after"

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("06:40|07:30|work|Get Started")

		expected := false
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts at the same time"

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("05:50|07:30|work|Get Started")

		expected := true
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts flush"

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("06:30|07:30|work|Get Started")

		expected := false
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts during (contained, should not matter)"

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("05:55|06:20|work|Get Started")

		expected := true
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts before"

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("04:50|07:30|work|Get Started")

		expected := false
		result := b.StartsDuring(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
}

func TestIsContainedIn(t *testing.T) {
	{
		testcase := "is contained"

		// +-----+
		// | a +---+
		// |   | b |
		// |   |   |
		// |   +---+
		// +-----+

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("05:55|06:20|work|Get Started")

		expected := true
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts after"

		// +-----+
		// | a   |
		// |     |
		// |     |
		// +-----+
		//
		// +-----+
		// | b   |
		// |     |
		// +-----+

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("06:40|07:30|work|Get Started")

		expected := false
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "is fully flush"

		// +-----+
		// | a   |
		// |     |
		// |     |
		// +-----+
		// | b   |
		// |     |
		// +-----+

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("06:30|07:30|work|Get Started")

		expected := false
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "only starts during but ends after"

		// +-----+
		// | a +---+
		// |   | b |
		// +---|   |
		//     +---+

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("05:55|06:40|work|Get Started")

		expected := false
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "starts before ends during"

		//     +---+
		// +---| b |
		// | a |   |
		// |   +---+
		// +----+

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("05:55|06:40|work|Get Started")

		expected := false
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
	{
		testcase := "perfectly flush"

		// +---+---+
		// | a | b |
		// |   |   |
		// +---+---+

		a := NewEvent("05:50|06:30|eating|Breakfast")
		b := NewEvent("05:50|06:30|work|Get Started")

		expected := true
		result := b.IsContainedIn(a)

		if result != expected {
			log.Fatalf("test case '%s' failed.", testcase)
		}
	}
}
