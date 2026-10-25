// Package korea is a finder for the Korean peninsula and nearby seas only.
//
// It embeds 20 KB of tzdata 2026e geometry (Asia/Seoul and its neighbors, clipped to [Bounds])
// instead of the 6.7 MB world data, and builds in well under a millisecond.
// Inside Bounds it answers like the world finder; outside it returns "".
package korea

import (
	_ "embed"
	"fmt"

	"github.com/Alfex4936/tzf"
	"github.com/Alfex4936/tzf/pb"
	"google.golang.org/protobuf/proto"
)

//go:generate sh -c "cd ../cmd/regiontzpb && go run . -bbox 123.8,32.4,132.7,39.2 -o ../../korea/korea.pb"

//go:embed korea.pb
var data []byte

const (
	minLng, minLat, maxLng, maxLat = 123.8, 32.4, 132.7, 39.2 // keep in sync with go:generate
	cell                           = 0.02                     // grid cell size in degrees
	gridW                          = int((maxLng-minLng)/cell) + 1
	gridH                          = int((maxLat-minLat)/cell) + 1
)

// Bounds is the clip box of the embedded data (Asia/Seoul's extent plus 0.5°): min lng, min lat, max lng, max lat.
var Bounds = [4]float64{minLng, minLat, maxLng, maxLat}

type finder struct {
	tzf.F
	names []string
	grid  []uint8 // per cell: 1+index into names when no border crosses the cell, else 0 (polygon test)
}

// NewFinder builds a finder from the embedded data.
func NewFinder() (tzf.F, error) {
	var tzs pb.Timezones
	if err := proto.Unmarshal(data, &tzs); err != nil {
		return nil, fmt.Errorf("korea: %w", err)
	}
	f, err := tzf.NewFinderFromPB(&tzs, tzf.SetDropPBTZ)
	if err != nil {
		return nil, err
	}
	return finder{F: f, names: f.TimezoneNames(), grid: buildGrid(f, &tzs)}, nil
}

func cellOf(lng, lat float64) (int, int) {
	return min(int((lng-minLng)/cell), gridW-1), min(int((lat-minLat)/cell), gridH-1)
}

// buildGrid marks every cell an edge's bounding box touches as a border cell, then flood-fills the
// rest: a connected run of cells no edge enters lies inside one zone, so one polygon test names it.
func buildGrid(f tzf.F, tzs *pb.Timezones) []uint8 {
	border := make([]bool, gridW*gridH)
	markEdges := func(pts []*pb.Point) {
		for i := range pts {
			a, b := pts[i], pts[(i+1)%len(pts)]
			x0, y0 := cellOf(min(float64(a.Lng), float64(b.Lng))-1e-9, min(float64(a.Lat), float64(b.Lat))-1e-9)
			x1, y1 := cellOf(max(float64(a.Lng), float64(b.Lng))+1e-9, max(float64(a.Lat), float64(b.Lat))+1e-9)
			for y := max(y0, 0); y <= y1; y++ {
				for x := max(x0, 0); x <= x1; x++ {
					border[y*gridW+x] = true
				}
			}
		}
	}
	for _, tz := range tzs.Timezones {
		for _, p := range tz.Polygons {
			markEdges(p.Points)
			for _, h := range p.Holes {
				markEdges(h.Points)
			}
		}
	}

	index := make(map[string]uint8, len(f.TimezoneNames()))
	for i, n := range f.TimezoneNames() {
		index[n] = uint8(i + 1)
	}
	grid := make([]uint8, gridW*gridH)
	seen := make([]bool, gridW*gridH)
	var queue []int
	for start := range grid {
		if border[start] || seen[start] {
			continue
		}
		x, y := start%gridW, start/gridW
		v := index[f.GetTimezoneName(minLng+(float64(x)+0.5)*cell, minLat+(float64(y)+0.5)*cell)]
		seen[start] = true
		queue = append(queue[:0], start)
		for len(queue) > 0 {
			c := queue[len(queue)-1]
			queue = queue[:len(queue)-1]
			grid[c] = v
			cx, cy := c%gridW, c/gridW
			for _, n := range [4][2]int{{cx - 1, cy}, {cx + 1, cy}, {cx, cy - 1}, {cx, cy + 1}} {
				if n[0] < 0 || n[1] < 0 || n[0] >= gridW || n[1] >= gridH {
					continue
				}
				if i := n[1]*gridW + n[0]; !border[i] && !seen[i] {
					seen[i] = true
					queue = append(queue, i)
				}
			}
		}
	}
	return grid
}

func inBounds(lng, lat float64) bool {
	return lng >= minLng && lat >= minLat && lng <= maxLng && lat <= maxLat
}

// GetTimezoneName returns "" outside Bounds, where the data has no geometry.
func (f finder) GetTimezoneName(lng, lat float64) string {
	if !inBounds(lng, lat) {
		return ""
	}
	x, y := cellOf(lng, lat)
	if v := f.grid[y*gridW+x]; v != 0 {
		return f.names[v-1]
	}
	return f.F.GetTimezoneName(lng, lat)
}

func (f finder) GetTimezoneNames(lng, lat float64) ([]string, error) {
	if !inBounds(lng, lat) {
		return nil, tzf.ErrNoTimezoneFound
	}
	return f.F.GetTimezoneNames(lng, lat)
}
