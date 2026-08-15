package timegrid

import (
	"errors"
	"sort"
	"time"

	"gridflex/internal/model"
)

type Window struct {
	StartIndex int
	EndIndex   int
}

func (w Window) Length() int {
	if w.EndIndex <= w.StartIndex {
		return 0
	}
	return w.EndIndex - w.StartIndex
}

func (w Window) Contains(index int) bool {
	return index >= w.StartIndex && index < w.EndIndex
}

func (w Window) Overlaps(other Window) bool {
	return w.StartIndex < other.EndIndex && other.StartIndex < w.EndIndex
}

func (w Window) Intersection(other Window) (Window, bool) {
	start := w.StartIndex
	if other.StartIndex > start {
		start = other.StartIndex
	}
	end := w.EndIndex
	if other.EndIndex < end {
		end = other.EndIndex
	}
	if start >= end {
		return Window{}, false
	}
	return Window{StartIndex: start, EndIndex: end}, true
}

func (g Grid) ValidateWindow(window Window) error {
	if window.StartIndex < 0 {
		return errors.New("window start must be non-negative")
	}
	if window.EndIndex <= window.StartIndex {
		return errors.New("window end must exceed start")
	}
	if window.EndIndex > g.Count() {
		return errors.New("window end exceeds grid")
	}
	return nil
}

func (g Grid) WindowIntervals(window Window) ([]model.Interval, error) {
	if err := g.ValidateWindow(window); err != nil {
		return nil, err
	}
	result := make([]model.Interval, window.Length())
	copy(result, g.intervals[window.StartIndex:window.EndIndex])
	return result, nil
}

func (g Grid) WindowDuration(window Window) (time.Duration, error) {
	if err := g.ValidateWindow(window); err != nil {
		return 0, err
	}
	return time.Duration(window.Length()) * time.Duration(g.minutes) * time.Minute, nil
}

func MergeWindows(input []Window) []Window {
	if len(input) == 0 {
		return nil
	}
	ordered := append([]Window(nil), input...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].StartIndex != ordered[j].StartIndex {
			return ordered[i].StartIndex < ordered[j].StartIndex
		}
		return ordered[i].EndIndex < ordered[j].EndIndex
	})
	result := make([]Window, 0, len(ordered))
	for _, window := range ordered {
		if window.Length() == 0 {
			continue
		}
		if len(result) == 0 {
			result = append(result, window)
			continue
		}
		last := &result[len(result)-1]
		if window.StartIndex <= last.EndIndex {
			if window.EndIndex > last.EndIndex {
				last.EndIndex = window.EndIndex
			}
			continue
		}
		result = append(result, window)
	}
	return result
}

func ComplementWindows(count int, occupied []Window) ([]Window, error) {
	if count <= 0 {
		return nil, errors.New("count must be positive")
	}
	merged := MergeWindows(occupied)
	for _, window := range merged {
		if window.StartIndex < 0 || window.EndIndex > count {
			return nil, errors.New("occupied window outside grid")
		}
	}
	result := make([]Window, 0)
	cursor := 0
	for _, window := range merged {
		if cursor < window.StartIndex {
			result = append(result, Window{StartIndex: cursor, EndIndex: window.StartIndex})
		}
		cursor = window.EndIndex
	}
	if cursor < count {
		result = append(result, Window{StartIndex: cursor, EndIndex: count})
	}
	return result, nil
}
