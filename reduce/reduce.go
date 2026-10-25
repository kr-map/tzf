// Package reduce could reduce Polygon size both polygon lines and float precise.
package reduce

import (
	"sync"

	"github.com/kr-map/tzf/pb"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/simplify"
)

func ReducePoints(points []*pb.Point) []*pb.Point {
	if len(points) == 0 {
		return points
	}
	// Pre-allocate to avoid multiple memory allocations
	original := make(orb.LineString, 0, len(points))
	for _, point := range points {
		original = append(original, orb.Point{float64(point.Lng), float64(point.Lat)})
	}
	// Perform the simplification in place to avoid extra allocations
	reduced := simplify.DouglasPeucker(0.001).Simplify(original).(orb.LineString)
	res := make([]*pb.Point, 0, len(reduced))
	for _, orbPoint := range reduced {
		res = append(res, &pb.Point{
			Lng: float32(orbPoint.Lon()),
			Lat: float32(orbPoint.Lat()),
		})
	}
	return res
}

func Do(input *pb.Timezones, skip int, precise float64, minist float64) *pb.Timezones {
	output := &pb.Timezones{
		Version:   input.Version,
		Timezones: make([]*pb.Timezone, len(input.Timezones)), // Pre-allocate the slice for all timezones
	}

	var wg sync.WaitGroup

	for idx, timezone := range input.Timezones {
		wg.Add(1)
		go func(idx int, tz *pb.Timezone) {
			defer wg.Done()
			reducedTimezone := &pb.Timezone{
				Name:     tz.Name,
				Polygons: make([]*pb.Polygon, len(tz.Polygons)), // Pre-allocate slice for all polygons
			}
			for i, polygon := range tz.Polygons {
				newPoly := &pb.Polygon{
					Points: ReducePoints(polygon.Points),
					Holes:  make([]*pb.Polygon, len(polygon.Holes)), // Pre-allocate slice for all holes
				}
				for j, hole := range polygon.Holes {
					newPoly.Holes[j] = &pb.Polygon{
						Points: ReducePoints(hole.Points),
					}
				}
				reducedTimezone.Polygons[i] = newPoly
			}
			output.Timezones[idx] = reducedTimezone
		}(idx, timezone)
	}
	wg.Wait()

	return output
}
