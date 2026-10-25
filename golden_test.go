package tzf_test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/kr-map/tzf"
	gocitiesjson "github.com/ringsaturn/go-cities.json"
)

// goldenPoints: every world city plus a 0.01° grid over Korea and its neighbors' coasts.
func goldenPoints() [][2]float64 {
	var pts [][2]float64
	for _, c := range gocitiesjson.Cities {
		pts = append(pts, [2]float64{c.Lng, c.Lat})
	}
	for lat := 32.5; lat <= 40.0; lat += 0.01 {
		for lng := 123.5; lng <= 132.5; lng += 0.01 {
			pts = append(pts, [2]float64{lng, lat})
		}
	}
	return pts
}

// TZF_GOLDEN=write <path> records answers; TZF_GOLDEN=<path> compares against them.
func TestGolden(t *testing.T) {
	mode, path, _ := strings.Cut(os.Getenv("TZF_GOLDEN"), " ")
	if mode == "" {
		t.Skip("set TZF_GOLDEN")
	}
	if mode != "write" {
		path = mode
	}
	f, err := tzf.NewDefaultFinder()
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, p := range goldenPoints() {
		names, _ := f.GetTimezoneNames(p[0], p[1])
		lines = append(lines, fmt.Sprintf("%.4f,%.4f %s %s", p[0], p[1], f.GetTimezoneName(p[0], p[1]), strings.Join(names, "|")))
	}
	if mode == "write" {
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d points", len(lines))
		return
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	sc := bufio.NewScanner(file)
	diff := 0
	for i := 0; sc.Scan(); i++ {
		if sc.Text() != lines[i] {
			if diff++; diff <= 20 {
				t.Errorf("want %s\n got %s", sc.Text(), lines[i])
			}
		}
	}
	t.Logf("%d points, %d differ", len(lines), diff)
}
