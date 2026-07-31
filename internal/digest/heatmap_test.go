package digest

import (
	"strings"
	"testing"
	"time"
)

func TestRenderHeatmap_Empty(t *testing.T) {
	heatmapStr := RenderHeatmap(nil, 4)

	if !strings.Contains(heatmapStr, "·") {
		t.Error("Expected empty heatmap to contain '·' characters")
	}
}

func TestRenderHeatmap_WithData(t *testing.T) {
	today := time.Now()
	data := []DayActivity{
		{Date: today.AddDate(0, 0, -4).Format("2006-01-02"), Count: 10},
		{Date: today.AddDate(0, 0, -3).Format("2006-01-02"), Count: 4},
		{Date: today.AddDate(0, 0, -2).Format("2006-01-02"), Count: 2},
		{Date: today.AddDate(0, 0, -1).Format("2006-01-02"), Count: 1},
		{Date: today.Format("2006-01-02"), Count: 0},
	}

	heatmapStr := RenderHeatmap(data, 4)

	// Ensure we have different activity characters
	if !strings.Contains(heatmapStr, "·") {
		t.Error("Expected heatmap to contain '·' (count 0)")
	}
	if !strings.Contains(heatmapStr, "░") {
		t.Error("Expected heatmap to contain '░' (count 1)")
	}
	if !strings.Contains(heatmapStr, "▒") {
		t.Error("Expected heatmap to contain '▒' (count 2-3)")
	}
	if !strings.Contains(heatmapStr, "▓") {
		t.Error("Expected heatmap to contain '▓' (count 4-5)")
	}
	if !strings.Contains(heatmapStr, "█") {
		t.Error("Expected heatmap to contain '█' (count > 5)")
	}
}

func TestRenderHeatmap_Legend(t *testing.T) {
	heatmapStr := RenderHeatmap(nil, 4)

	if !strings.Contains(heatmapStr, "Less") {
		t.Error("Expected heatmap to contain 'Less' legend text")
	}
	if !strings.Contains(heatmapStr, "More") {
		t.Error("Expected heatmap to contain 'More' legend text")
	}
}

// Every weekday gets its own row — a grid that only drew Mon/Wed/Fri silently
// hid four sevenths of the activity.
func TestRenderHeatmap_AllWeekdaysRendered(t *testing.T) {
	today := time.Now()
	var data []DayActivity
	for i := 6; i >= 0; i-- {
		data = append(data, DayActivity{Date: today.AddDate(0, 0, -i).Format("2006-01-02"), Count: 1})
	}

	heatmapStr := RenderHeatmap(data, 4)
	grid, _, _ := strings.Cut(heatmapStr, "\nLess")

	if got := strings.Count(grid, "░"); got != 7 {
		t.Errorf("Expected 7 active cells (one per weekday), got %d in:\n%s", got, heatmapStr)
	}
	if got := strings.Count(grid, "\n"); got != 7 {
		t.Errorf("Expected 7 rows, got %d", got)
	}
}

func TestRenderHeatmap_WeeksDefaultsToFour(t *testing.T) {
	if got, want := RenderHeatmap(nil, 0), RenderHeatmap(nil, 4); got != want {
		t.Errorf("Expected a non-positive week count to render four weeks, got:\n%s", got)
	}
}

func TestRenderHeatmap_LegendCoversEveryLevel(t *testing.T) {
	_, legend, found := strings.Cut(RenderHeatmap(nil, 4), "\nLess")
	if !found {
		t.Fatal("Expected a legend")
	}
	for _, level := range []string{"·", "░", "▒", "▓", "█"} {
		if !strings.Contains(legend, level) {
			t.Errorf("Legend is missing level %q: %q", level, legend)
		}
	}
}
