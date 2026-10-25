// Command regiontzpb writes the timezones inside a bounding box as a small [pb.Timezones] file.
//
// Geometry comes from upstream github.com/ringsaturn/tzf/v2 (topology-aware simplified data, no
// gaps between neighbors), clipped to the box. Inside the box a finder built from the output
// answers like the world data; outside it knows nothing.
//
//	go run . -bbox 123.8,32.4,132.7,39.2 -o ../../korea/korea.pb
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/kr-map/tzf/pb"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/clip"
	"github.com/paulmach/orb/geojson"
	tzfv2 "github.com/ringsaturn/tzf/v2"
	"google.golang.org/protobuf/proto"
)

func main() {
	bboxFlag := flag.String("bbox", "", "minLng,minLat,maxLng,maxLat")
	out := flag.String("o", "", "output .pb path")
	flag.Parse()
	box, err := parseBBox(*bboxFlag)
	if err != nil || *out == "" {
		log.Fatalf("usage: regiontzpb -bbox minLng,minLat,maxLng,maxLat -o out.pb (%v)", err)
	}

	world, err := tzfv2.NewDefaultFinder()
	if err != nil {
		log.Fatal(err)
	}
	res := &pb.Timezones{Version: world.DataVersion(), Reduced: true}
	for _, name := range world.TimezoneNames() { // upstream order, which first-match ties follow
		raw, err := world.(tzfv2.GeoJSONer).GetTZGeoJSON(name)
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		fc, err := geojson.UnmarshalFeatureCollection(raw)
		if err != nil {
			log.Fatalf("%s: %v", name, err)
		}
		tz := &pb.Timezone{Name: name}
		for _, ft := range fc.Features {
			if !ft.Geometry.Bound().Intersects(box) {
				continue
			}
			for _, poly := range clip.MultiPolygon(box, toMultiPolygon(ft.Geometry)) {
				if p := toPB(poly); p != nil {
					tz.Polygons = append(tz.Polygons, p)
				}
			}
		}
		if len(tz.Polygons) > 0 {
			res.Timezones = append(res.Timezones, tz)
		}
	}

	data, err := proto.Marshal(res)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		log.Fatal(err)
	}
	points := 0
	for _, tz := range res.Timezones {
		for _, p := range tz.Polygons {
			points += len(p.Points)
			for _, h := range p.Holes {
				points += len(h.Points)
			}
		}
		fmt.Printf("%-16s %d polygons\n", tz.Name, len(tz.Polygons))
	}
	fmt.Printf("data %s: %d zones, %d points, %d bytes -> %s\n", res.Version, len(res.Timezones), points, len(data), *out)
}

func parseBBox(s string) (orb.Bound, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return orb.Bound{}, fmt.Errorf("bbox needs 4 numbers")
	}
	var v [4]float64
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return orb.Bound{}, err
		}
		v[i] = f
	}
	return orb.Bound{Min: orb.Point{v[0], v[1]}, Max: orb.Point{v[2], v[3]}}, nil
}

func toMultiPolygon(g orb.Geometry) orb.MultiPolygon {
	switch g := g.(type) {
	case orb.Polygon:
		return orb.MultiPolygon{g}
	case orb.MultiPolygon:
		return g
	}
	log.Fatalf("unexpected geometry %s", g.GeoJSONType())
	return nil
}

// toPB drops rings the clip collapsed to fewer than 3 distinct points.
func toPB(poly orb.Polygon) *pb.Polygon {
	if len(poly) == 0 || len(poly[0]) < 4 {
		return nil
	}
	p := &pb.Polygon{Points: ringPoints(poly[0])}
	for _, hole := range poly[1:] {
		if len(hole) >= 4 {
			p.Holes = append(p.Holes, &pb.Polygon{Points: ringPoints(hole)})
		}
	}
	return p
}

func ringPoints(r orb.Ring) []*pb.Point {
	pts := make([]*pb.Point, len(r))
	for i, c := range r {
		pts[i] = &pb.Point{Lng: float32(c[0]), Lat: float32(c[1])}
	}
	return pts
}
