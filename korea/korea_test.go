package korea

import "testing"

func TestKnownPoints(t *testing.T) {
	f, err := NewFinder()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		lng, lat float64
		want     string
	}{
		{126.9780, 37.5665, "Asia/Seoul"},     // Seoul
		{126.5312, 33.4996, "Asia/Seoul"},     // Jeju
		{131.8664, 37.2412, "Asia/Seoul"},     // Dokdo
		{125.7625, 39.0392, "Asia/Pyongyang"}, // Pyongyang
		{130.4017, 33.5904, "Asia/Tokyo"},     // Fukuoka
		{125.85938, 37.40502, "Etc/GMT-8"},    // sea tile the 2024a preindex wrongly gave to Seoul
		{139.6917, 35.6895, ""},               // Tokyo, outside Bounds
	} {
		if got := f.GetTimezoneName(c.lng, c.lat); got != c.want {
			t.Errorf("%v,%v = %q, want %q", c.lng, c.lat, got, c.want)
		}
	}
}

// Every grid cell that answers without a polygon test must agree with the polygon test inside it.
func TestGridMatchesPolygons(t *testing.T) {
	f, err := NewFinder()
	if err != nil {
		t.Fatal(err)
	}
	kf := f.(finder)
	filled := 0
	for i, v := range kf.grid {
		if v == 0 {
			continue
		}
		filled++
		x, y := float64(i%gridW), float64(i/gridW)
		for _, o := range [][2]float64{{0.01, 0.01}, {0.5, 0.5}, {0.99, 0.01}, {0.01, 0.99}, {0.99, 0.99}} {
			lng, lat := minLng+(x+o[0])*cell, minLat+(y+o[1])*cell
			if !inBounds(lng, lat) {
				continue
			}
			if want := kf.F.GetTimezoneName(lng, lat); want != kf.names[v-1] {
				t.Fatalf("cell %d at %v,%v: grid %q, polygons %q", i, lng, lat, kf.names[v-1], want)
			}
		}
	}
	t.Logf("%d of %d cells answer from the grid", filled, len(kf.grid))
}
