package tzf_test

import (
	"runtime"
	"testing"

	"github.com/kr-map/tzf"
	"github.com/kr-map/tzf/korea"
)

// Points the k-pullup API asks about: Korean land, islands, coastal sea, and neighbors.
var koreaBenchPoints = []struct {
	name     string
	lng, lat float64
}{
	{"seoul", 126.9780, 37.5665},
	{"busan", 129.0756, 35.1796},
	{"jeju", 126.5312, 33.4996},
	{"dokdo", 131.8664, 37.2412},
	{"ulleung", 130.9057, 37.4844},
	{"busan-sea", 129.30, 35.05},
	{"pyongyang", 125.7625, 39.0392},
	{"fukuoka", 130.4017, 33.5904},
	{"tokyo", 139.6917, 35.6895},
	{"beijing", 116.4074, 39.9042},
	{"new-york", -74.0060, 40.7128},
	{"pacific", -150.0, 0.0},
}

var finders = []struct {
	name string
	new  func() (tzf.F, error)
}{{"default", tzf.NewDefaultFinder}, {"korea", korea.NewFinder}}

func BenchmarkInit(b *testing.B) {
	for _, f := range finders {
		b.Run(f.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := f.new(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestRetainedHeap logs the heap each finder keeps alive after init.
func TestRetainedHeap(t *testing.T) {
	for _, f := range finders {
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		finder, err := f.new()
		if err != nil {
			t.Fatal(err)
		}
		runtime.GC()
		runtime.ReadMemStats(&after)
		t.Logf("%-8s retained heap %.2f MB, allocated during init %.1f MB", f.name, float64(after.HeapAlloc-before.HeapAlloc)/1e6, float64(after.TotalAlloc-before.TotalAlloc)/1e6)
		runtime.KeepAlive(finder)
	}
}

func BenchmarkKoreaPoints(b *testing.B) {
	for _, f := range finders {
		finder, err := f.new()
		if err != nil {
			b.Fatal(err)
		}
		for _, p := range koreaBenchPoints {
			b.Run(f.name+"/"+p.name, func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					finder.GetTimezoneName(p.lng, p.lat)
				}
			})
		}
	}
}

func TestKoreaPointNames(t *testing.T) {
	for _, f := range finders {
		finder, err := f.new()
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range koreaBenchPoints {
			t.Logf("%-8s %-10s %q", f.name, p.name, finder.GetTimezoneName(p.lng, p.lat))
		}
	}
}
