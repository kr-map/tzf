package main

import (
	"testing"

	"github.com/kr-map/tzf/korea"
	tzfv2 "github.com/ringsaturn/tzf/v2"
)

// The clipped data must answer exactly like upstream's world finder everywhere inside the box.
func TestKoreaMatchesWorld(t *testing.T) {
	world, err := tzfv2.NewDefaultFinder()
	if err != nil {
		t.Fatal(err)
	}
	kr, err := korea.NewFinder()
	if err != nil {
		t.Fatal(err)
	}
	if kr.DataVersion() != world.DataVersion() {
		t.Fatalf("korea data %s, world %s: run go generate ./korea", kr.DataVersion(), world.DataVersion())
	}
	b := korea.Bounds
	const step = 0.005 // ~500 m; ~3.2M points
	n, seoul, diff := 0, 0, 0
	for lat := b[1] + step/2; lat < b[3]; lat += step {
		for lng := b[0] + step/2; lng < b[2]; lng += step {
			n++
			want, got := world.GetTimezoneName(lng, lat), kr.GetTimezoneName(lng, lat)
			if got == "Asia/Seoul" {
				seoul++
			}
			if want != got {
				diff++
				if !nearby(world, lng, lat, got) {
					t.Errorf("%.4f,%.4f: world %q, korea %q", lng, lat, want, got)
				}
			}
		}
	}
	t.Logf("%d points, %d Asia/Seoul, %d border ties", n, seoul, diff)
	if got := kr.GetTimezoneName(b[0]-0.01, b[1]); got != "" {
		t.Errorf("outside bounds: %q", got)
	}
}

// nearby reports whether world gives name within 2 m: the float32 copy rounds shared borders by under a meter.
func nearby(world tzfv2.F, lng, lat float64, name string) bool {
	const d = 2e-5
	for _, o := range [][2]float64{{-d, 0}, {d, 0}, {0, -d}, {0, d}} {
		if world.GetTimezoneName(lng+o[0], lat+o[1]) == name {
			return true
		}
	}
	return false
}
