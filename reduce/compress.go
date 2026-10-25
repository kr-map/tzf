package reduce

import (
	"fmt"
	"sync"

	"github.com/kr-map/tzf/pb"
	"github.com/twpayne/go-polyline"
)

func CompressedPointsToPolylineBytes(points []*pb.Point) []byte {
	expect := [][]float64{}
	for _, point := range points {
		expect = append(expect, []float64{float64(point.Lng), float64(point.Lat)})
	}
	return polyline.EncodeCoords(expect)
}

func DecompressedPolylineBytesToPoints(input []byte) []*pb.Point {
	coords, _, _ := polyline.DecodeCoords(input)
	expect := make([]*pb.Point, len(coords))
	for i, coord := range coords {
		expect[i] = &pb.Point{
			Lng: float32(coord[0]), Lat: float32(coord[1]),
		}
	}
	return expect
}

func CompressWithPolyline(input *pb.Timezones) *pb.CompressedTimezones {
	output := &pb.CompressedTimezones{
		Method:    pb.CompressMethod_Polyline,
		Version:   input.Version,
		Timezones: make([]*pb.CompressedTimezone, len(input.Timezones)), // Allocate exact number of timezones
	}

	var wg sync.WaitGroup

	for idx, timezone := range input.Timezones {
		wg.Add(1)
		go func(idx int, tz *pb.Timezone) {
			defer wg.Done()
			reducedTimezone := &pb.CompressedTimezone{
				Name: tz.Name,
				Data: make([]*pb.CompressedPolygon, 0, len(tz.Polygons)),
			}
			for _, polygon := range tz.Polygons {
				newPoly := &pb.CompressedPolygon{
					Points: CompressedPointsToPolylineBytes(polygon.Points),
					Holes:  make([]*pb.CompressedPolygon, 0, len(polygon.Holes)),
				}
				for _, hole := range polygon.Holes {
					newPoly.Holes = append(newPoly.Holes, &pb.CompressedPolygon{
						Points: CompressedPointsToPolylineBytes(hole.Points),
					})
				}
				reducedTimezone.Data = append(reducedTimezone.Data, newPoly)
			}
			output.Timezones[idx] = reducedTimezone
		}(idx, timezone)
	}
	wg.Wait()

	return output
}

func Compress(input *pb.Timezones, method pb.CompressMethod) (*pb.CompressedTimezones, error) {
	switch method {
	case pb.CompressMethod_Polyline:
		return CompressWithPolyline(input), nil
	default:
		return nil, fmt.Errorf("tzf/reduce: unknown method %v", method)
	}
}

func DecompressWithPolyline(input *pb.CompressedTimezones) *pb.Timezones {
	output := &pb.Timezones{
		Version:   input.Version,
		Timezones: make([]*pb.Timezone, len(input.Timezones)),
	}

	var wg sync.WaitGroup
	for idx, timezone := range input.Timezones {
		wg.Add(1)
		go func(idx int, tz *pb.CompressedTimezone) {
			defer wg.Done()
			reducedTimezone := &pb.Timezone{
				Name:     tz.Name,
				Polygons: make([]*pb.Polygon, len(tz.Data)),
			}
			for i, polygon := range tz.Data {
				newPoly := &pb.Polygon{
					Points: DecompressedPolylineBytesToPoints(polygon.Points),
					Holes:  make([]*pb.Polygon, len(polygon.Holes)),
				}
				for j, hole := range polygon.Holes {
					newPoly.Holes[j] = &pb.Polygon{
						Points: DecompressedPolylineBytesToPoints(hole.Points),
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

func Decompress(input *pb.CompressedTimezones) (*pb.Timezones, error) {
	switch input.Method {
	case pb.CompressMethod_Polyline:
		return DecompressWithPolyline(input), nil
	default:
		return nil, fmt.Errorf("tzf/reduce: unknown method %v", input.Method)
	}
}
