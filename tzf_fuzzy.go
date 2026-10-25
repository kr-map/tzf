package tzf

import (
	"github.com/kr-map/tzf/pb"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/maptile"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// FuzzyFinder use a tile map to store timezone name. Data are made by
// [github.com/kr-map/tzf/cmd/preindextzpb] which powerd by
// [github.com/kr-map/tzf/preindex.PreIndexTimezones].
type FuzzyFinder struct {
	idxZoom int
	aggZoom int
	m       map[maptile.Tile][]string // timezones may have common area
	version string
	names   []string
}

func NewFuzzyFinderFromPB(input *pb.PreindexTimezones) (F, error) {
	f := &FuzzyFinder{
		m:       make(map[maptile.Tile][]string),
		idxZoom: int(input.IdxZoom),
		aggZoom: int(input.AggZoom),
		version: input.Version,
	}
	namesMap := map[string]bool{}
	for _, item := range input.Keys {
		tile := maptile.New(uint32(item.X), uint32(item.Y), maptile.Zoom(item.Z))
		if _, ok := f.m[tile]; !ok {
			f.m[tile] = make([]string, 0)
		}
		f.m[tile] = append(f.m[tile], item.Name)
		namesMap[item.Name] = true
	}
	f.names = maps.Keys(namesMap)
	slices.Sort(f.names)
	return f, nil
}

func (f *FuzzyFinder) GetTimezoneName(lng float64, lat float64) string {
	names, err := f.GetTimezoneNames(lng, lat)
	if err != nil || len(names) == 0 {
		return ""
	}
	return names[0]
}

func (f *FuzzyFinder) GetTimezoneNames(lng float64, lat float64) ([]string, error) {
	p := orb.Point{lng, lat}
	for z := f.aggZoom; z <= f.idxZoom; z++ {
		key := maptile.At(p, maptile.Zoom(z))
		if v, ok := f.m[key]; ok {
			return v, nil
		}
	}
	return nil, ErrNoTimezoneFound
}

func (f *FuzzyFinder) TimezoneNames() []string {
	return f.names
}

func (f *FuzzyFinder) DataVersion() string {
	return f.version
}
