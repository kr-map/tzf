package preindex

import (
	"testing"

	"github.com/paulmach/orb/maptile"
	"github.com/tidwall/geojson/geometry"
)

func rect(minX, minY, maxX, maxY float64) *geometry.Poly {
	return geometry.NewPoly([]geometry.Point{{X: minX, Y: minY}, {X: maxX, Y: minY}, {X: maxX, Y: maxY}, {X: minX, Y: maxY}, {X: minX, Y: minY}}, nil, nil)
}

// A tile only half inside a zone must not become a fast-path tile (it used to pass unconditionally).
func TestEnsureInside(t *testing.T) {
	tile := maptile.New(2805, 6396, 13)
	b := tile.Bound()
	mid := (b.Min.Lon() + b.Max.Lon()) / 2
	if got := EnsureInside([]*geometry.Poly{rect(b.Min.Lon()-0.01, b.Min.Lat()-0.01, mid, b.Max.Lat()+0.01)}, []maptile.Tile{tile}); len(got) != 0 {
		t.Fatalf("cross-boundary tile kept: %v", got)
	}
	if got := EnsureInside([]*geometry.Poly{rect(b.Min.Lon()-0.01, b.Min.Lat()-0.01, b.Max.Lon()+0.01, b.Max.Lat()+0.01)}, []maptile.Tile{tile}); len(got) != 1 {
		t.Fatalf("covered tile dropped: %v", got)
	}
}
