package model

import (
	"testing"
)

func TestOffset(t *testing.T) {
	{
		stamp := Timestamp{Hour: 10, Minute: 10}
		offset := TimeOffset{Timestamp{0, 0}, true}
		result := stamp.Offset(offset)
		if result != stamp {
			t.Fatalf("Timestamp 10:10 + 0:00 should be 10:10, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{Hour: 10, Minute: 10}
		offset := TimeOffset{Timestamp{0, 0}, false}
		result := stamp.Offset(offset)
		if result != stamp {
			t.Fatalf("Timestamp 10:10 - 0:00 should be 10:10, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{10, 10}
		offset := TimeOffset{Timestamp{1, 0}, true}
		result := stamp.Offset(offset)
		if (result != Timestamp{11, 10}) {
			t.Fatalf("Timestamp 10:10 + 1:00 should be 11:10, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{10, 10}
		offset := TimeOffset{Timestamp{1, 0}, false}
		result := stamp.Offset(offset)
		if (result != Timestamp{9, 10}) {
			t.Fatalf("Timestamp 10:10 - 1:00 should be 9:10, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{10, 10}
		offset := TimeOffset{Timestamp{0, 49}, true}
		result := stamp.Offset(offset)
		if (result != Timestamp{10, 59}) {
			t.Fatalf("Timestamp 10:10 + 0:49 should be 10:59, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{10, 10}
		offset := TimeOffset{Timestamp{0, 50}, true}
		result := stamp.Offset(offset)
		if (result != Timestamp{11, 00}) {
			t.Fatalf("Timestamp 10:10 + 0:50 should be 11:00, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{10, 10}
		offset := TimeOffset{Timestamp{0, 51}, true}
		result := stamp.Offset(offset)
		if (result != Timestamp{11, 01}) {
			t.Fatalf("Timestamp 10:10 + 0:51 should be 11:01, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{0, 10}
		offset := TimeOffset{Timestamp{1, 0}, false}
		result := stamp.Offset(offset)
		if (result != Timestamp{23, 10}) {
			t.Fatalf("Timestamp 0:10 - 1:00 should be 23:10, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{23, 10}
		offset := TimeOffset{Timestamp{1, 0}, true}
		result := stamp.Offset(offset)
		if (result != Timestamp{0, 10}) {
			t.Fatalf("Timestamp 23:10 + 1:00 should be 0:10, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{1, 30}
		offset := TimeOffset{Timestamp{2, 40}, false}
		result := stamp.Offset(offset)
		if (result != Timestamp{22, 50}) {
			t.Fatalf("Timestamp 1:30 - 2:40 should be 22:50, but is '%s'", result.ToString())
		}
	}

	{
		stamp := Timestamp{22, 30}
		offset := TimeOffset{Timestamp{2, 40}, true}
		result := stamp.Offset(offset)
		if (result != Timestamp{1, 10}) {
			t.Fatalf("Timestamp 22:30 + 2:40 should be 1:10, but is '%s'", result.ToString())
		}
	}
}
