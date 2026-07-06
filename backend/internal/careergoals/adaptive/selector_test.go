package adaptive

import (
	"math/rand"
	"testing"
	"time"
)

func TestAllocateWeak(t *testing.T) {
	cases := []struct {
		total int
		ratio float64
		want  int
	}{
		{20, 0.6, 12},
		{40, 0.6, 24},
		{10, 0.6, 6},
		{0, 0.6, 0},
		{20, 0, 0},
		{20, 1.5, 20}, // clamped to 1.0
		{20, -0.5, 0}, // clamped to 0
		{7, 0.5, 3},   // int() truncation, not rounding
	}
	for _, c := range cases {
		if got := allocateWeak(c.total, c.ratio); got != c.want {
			t.Errorf("allocateWeak(%d, %v) = %d, want %d", c.total, c.ratio, got, c.want)
		}
	}
}

func TestPerTopicAllocation(t *testing.T) {
	// 12 across 5 topics: 3,3,2,2,2 (remainder to earliest).
	got := perTopicAllocation(12, 5)
	want := []int{3, 3, 2, 2, 2}
	if len(got) != len(want) {
		t.Fatalf("length = %d, want %d", len(got), len(want))
	}
	sum := 0
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d = %d, want %d", i, got[i], want[i])
		}
		sum += got[i]
	}
	if sum != 12 {
		t.Errorf("allocation sum = %d, want 12", sum)
	}

	if a := perTopicAllocation(0, 3); len(a) != 3 || a[0] != 0 {
		t.Errorf("zero weakCount should give zeros, got %v", a)
	}
	if a := perTopicAllocation(5, 0); len(a) != 0 {
		t.Errorf("zero topics should give empty slice, got %v", a)
	}
}

func TestParseIDsSkipsInvalid(t *testing.T) {
	got := parseIDs([]string{"1", "bad", "3", ""})
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("parseIDs = %v, want [1 3]", got)
	}
}

func TestSetHelpersRoundTrip(t *testing.T) {
	merged := mergeSets(toSet([]int64{1, 2}), toSet([]int64{2, 3}))
	if len(merged) != 3 {
		t.Fatalf("mergeSets len = %d, want 3", len(merged))
	}
	if len(setToSlice(merged)) != 3 {
		t.Fatalf("setToSlice len = %d, want 3", len(setToSlice(merged)))
	}
}

func TestShuffleKeepsAllElements(t *testing.T) {
	s := &QuestionSelector{rng: rand.New(rand.NewSource(1))}
	qs := []Question{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}, {ID: "5"}}
	s.shuffle(qs)
	seen := map[string]bool{}
	for _, q := range qs {
		seen[q.ID] = true
	}
	for _, id := range []string{"1", "2", "3", "4", "5"} {
		if !seen[id] {
			t.Errorf("shuffle dropped element %s", id)
		}
	}
}

func TestWeekBoundsIsSundayToSaturday(t *testing.T) {
	// 2025-01-08 is a Wednesday; its week is Sun 2025-01-05 .. Sat 2025-01-11.
	start, end := weekBounds(time.Date(2025, 1, 8, 15, 30, 0, 0, time.UTC))
	if start.Weekday() != time.Sunday {
		t.Errorf("week start weekday = %v, want Sunday", start.Weekday())
	}
	if got := start.Format("2006-01-02"); got != "2025-01-05" {
		t.Errorf("week start = %s, want 2025-01-05", got)
	}
	if got := end.Format("2006-01-02"); got != "2025-01-11" {
		t.Errorf("week end = %s, want 2025-01-11", got)
	}
}
