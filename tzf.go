// Package tzf is a package convert (lng,lat) to timezone.
//
// Inspired by timezonefinder https://github.com/jannikmi/timezonefinder,
// fast python package for finding the timezone of any point on earth (coordinates) offline.
package tzf

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/kr-map/tzf/convert"
	"github.com/kr-map/tzf/pb"
	"github.com/kr-map/tzf/reduce"
	"github.com/tidwall/geojson/geometry"
	"github.com/tidwall/rtree"
	"github.com/twpayne/go-polyline"
	"golang.org/x/exp/slices"
)

var ErrNoTimezoneFound = errors.New("tzf: no timezone found")

type Option struct {
	DropPBTZ bool
}

type OptionFunc = func(opt *Option)

// SetDropPBTZ will make Finder not save [github.com/kr-map/tzf/pb.Timezone] in memory
func SetDropPBTZ(opt *Option) {
	opt.DropPBTZ = true
}

type tzitem struct {
	idx   int // position in the input; overlapping zones resolve to the lowest
	pbtz  *pb.Timezone
	name  string
	polys []*geometry.Poly
	min   [2]float64
	max   [2]float64
}

func newNotFoundErr(lng float64, lat float64) error {
	return fmt.Errorf("tzf: not found for %v,%v", lng, lat)
}

func (i *tzitem) ContainsPoint(p geometry.Point) bool {
	// Quick rejection if the point is outside the tzitem's bounding box
	if p.X < i.min[0] || p.X > i.max[0] || p.Y < i.min[1] || p.Y > i.max[1] {
		return false
	}

	for _, poly := range i.polys {
		if poly.ContainsPoint(p) {
			return true
		}
	}
	return false
}

func (i *tzitem) getMinMax() ([2]float64, [2]float64) {
	retmin := [2]float64{
		i.polys[0].Rect().Min.X,
		i.polys[0].Rect().Min.Y,
	}
	retmax := [2]float64{
		i.polys[0].Rect().Max.X,
		i.polys[0].Rect().Max.Y,
	}

	for _, poly := range i.polys {
		minx := poly.Rect().Min.X
		miny := poly.Rect().Min.Y
		if minx < retmin[0] {
			retmin[0] = minx
		}
		if miny < retmin[1] {
			retmin[1] = miny
		}

		maxx := poly.Rect().Max.X
		maxy := poly.Rect().Max.Y
		if maxx > retmax[0] {
			retmax[0] = maxx

		}
		if maxy > retmax[1] {
			retmax[1] = maxy
		}
	}
	return retmin, retmax
}

// Finder is based on point-in-polygon search algo.
//
// Memory will use about 100MB if lite data and 1G if full data.
// Performance is very stable and very accuate.
type Finder struct {
	items   []*tzitem
	names   []string
	reduced bool
	tr      *rtree.RTreeG[*tzitem]
	opt     *Option
	version string
}

func NewFinderFromRawJSON(input *convert.BoundaryFile, opts ...OptionFunc) (F, error) {
	timezones, err := convert.Do(input)
	if err != nil {
		return nil, err
	}
	return NewFinderFromPB(timezones, opts...)
}

func NewFinderFromPB(input *pb.Timezones, opts ...OptionFunc) (F, error) {
	opt := newOption(opts)
	return build(input.Version, input.Reduced, opt, len(input.Timezones), func(i int) (*tzitem, error) {
		tz := input.Timezones[i]
		item := &tzitem{name: tz.Name}
		if !opt.DropPBTZ {
			item.pbtz = tz
		}
		for _, polygon := range tz.Polygons {
			holes := make([][]geometry.Point, len(polygon.Holes))
			for h, hole := range polygon.Holes {
				holes[h] = pbPoints(hole.Points)
			}
			item.polys = append(item.polys, newPoly(pbPoints(polygon.Points), holes))
		}
		return item, nil
	})
}

// NewFinderFromCompressed decodes the polylines straight into polygon points; with
// [SetDropPBTZ] it never materializes the intermediate [pb.Timezones].
func NewFinderFromCompressed(input *pb.CompressedTimezones, opts ...OptionFunc) (F, error) {
	opt := newOption(opts)
	if !opt.DropPBTZ || input.Method != pb.CompressMethod_Polyline {
		tzs, err := reduce.Decompress(input)
		if err != nil {
			return nil, err
		}
		return NewFinderFromPB(tzs, opts...)
	}
	return build(input.Version, false, opt, len(input.Timezones), func(i int) (*tzitem, error) {
		tz := input.Timezones[i]
		item := &tzitem{name: tz.Name}
		for _, polygon := range tz.Data {
			exterior, err := polylinePoints(polygon.Points)
			if err != nil {
				return nil, fmt.Errorf("tzf: %s: %w", tz.Name, err)
			}
			holes := make([][]geometry.Point, len(polygon.Holes))
			for h, hole := range polygon.Holes {
				if holes[h], err = polylinePoints(hole.Points); err != nil {
					return nil, fmt.Errorf("tzf: %s: %w", tz.Name, err)
				}
			}
			item.polys = append(item.polys, newPoly(exterior, holes))
		}
		return item, nil
	})
}

func newOption(opts []OptionFunc) *Option {
	opt := &Option{}
	for _, optFunc := range opts {
		optFunc(opt)
	}
	return opt
}

func newPoly(exterior []geometry.Point, holes [][]geometry.Point) *geometry.Poly {
	return geometry.NewPoly(exterior, holes, &geometry.IndexOptions{Kind: geometry.RTree, MinPoints: 64})
}

func pbPoints(points []*pb.Point) []geometry.Point {
	out := make([]geometry.Point, len(points))
	for i, p := range points {
		out[i] = geometry.Point{X: float64(p.Lng), Y: float64(p.Lat)}
	}
	return out
}

// polylinePoints matches reduce.DecompressedPolylineBytesToPoints bit for bit: a float running sum
// (as polyline.DecodeCoords does) rounded through float32 (as pb.Point stores it).
func polylinePoints(buf []byte) ([]geometry.Point, error) {
	out := make([]geometry.Point, 0, len(buf)/4)
	var lng, lat float64
	for len(buf) > 0 {
		dx, rest, err := polyline.DecodeInt(buf)
		if err != nil {
			return nil, err
		}
		dy, rest, err := polyline.DecodeInt(rest)
		if err != nil {
			return nil, err
		}
		buf = rest
		lng += float64(dx) / 1e5
		lat += float64(dy) / 1e5
		out = append(out, geometry.Point{X: float64(float32(lng)), Y: float64(float32(lat))})
	}
	return out, nil
}

// build makes the zones in parallel (polygon indexing dominates load time), then indexes them in input order.
func build(version string, reduced bool, opt *Option, n int, zone func(i int) (*tzitem, error)) (F, error) {
	items := make([]*tzitem, n)
	errs := make([]error, n)
	next := atomic.Int64{}
	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), max(n, 1)) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := int(next.Add(1) - 1); i < n; i = int(next.Add(1) - 1) {
				items[i], errs[i] = zone(i)
			}
		}()
	}
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	f := &Finder{items: items, names: make([]string, n), reduced: reduced, tr: &rtree.RTreeG[*tzitem]{}, opt: opt, version: version}
	for i, item := range items {
		item.idx = i
		f.names[i] = item.name
		if len(item.polys) == 0 {
			continue // nothing to find; keep the name
		}
		item.min, item.max = item.getMinMax()
		f.tr.Insert(item.min, item.max, item)
	}
	return f, nil
}

// lookup returns the first zone in input order that contains the point, or nil.
func (f *Finder) lookup(p geometry.Point) *tzitem {
	var best *tzitem
	f.tr.Search([2]float64{p.X, p.Y}, [2]float64{p.X, p.Y}, func(_, _ [2]float64, item *tzitem) bool {
		if (best == nil || item.idx < best.idx) && item.ContainsPoint(p) {
			best = item
		}
		return true
	})
	return best
}

// GetTimezoneName returns the first matching zone in data order (alphabetical in the released data), or "".
func (f *Finder) GetTimezoneName(lng float64, lat float64) string {
	if item := f.lookup(geometry.Point{X: lng, Y: lat}); item != nil {
		return item.name
	}
	return ""
}

// IsKoreaTimezone reports whether the point is in Asia/Seoul.
func (f *Finder) IsKoreaTimezone(lng float64, lat float64) bool {
	return f.GetTimezoneName(lng, lat) == "Asia/Seoul"
}

// Deprecated: use GetTimezoneName.
func (f *Finder) GetTimezoneName2(lng, lat float64) string {
	return f.GetTimezoneName(lng, lat)
}

func (f *Finder) GetTimezoneNames(lng float64, lat float64) ([]string, error) {
	p := geometry.Point{X: lng, Y: lat}
	var ret []string
	f.tr.Search([2]float64{lng, lat}, [2]float64{lng, lat}, func(_, _ [2]float64, item *tzitem) bool {
		if item.ContainsPoint(p) {
			ret = append(ret, item.name)
		}
		return true
	})
	if len(ret) == 0 {
		return nil, newNotFoundErr(lng, lat)
	}
	slices.Sort(ret)
	return ret, nil
}

func (f *Finder) TimezoneNames() []string {
	return f.names
}

func (f *Finder) DataVersion() string {
	return f.version
}
