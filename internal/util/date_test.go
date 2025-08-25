package util

import "testing"
import "time"

func TestBusinessDaysAgo(t *testing.T) {
	from := time.Date(2023, time.January, 16, 15, 0, 0, 0, time.UTC) // Monday
	want := time.Date(2023, time.January, 5, 0, 0, 0, 0, time.UTC)   // 7 business days earlier
	got := BusinessDaysAgo(from, 7)
	if !got.Equal(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
