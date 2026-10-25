package tzf

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/kr-map/tzf/pb"
	tzfrellite "github.com/ringsaturn/tzf-rel-lite"
	"google.golang.org/protobuf/proto"
)

// DefaultFinder is a finder impl combine both [FuzzyFinder] and [Finder].
//
// It's designed for performance first and allow some not so correct return at some area.
type DefaultFinder struct {
	fuzzyFinder F
	finder      F
}

func NewDefaultFinder() (F, error) {
	var (
		fuzzyFinder, finder F
		fuzzyErr            error
		wg                  sync.WaitGroup
	)
	wg.Add(1)
	go func() { // the two decodes are independent; run them side by side
		defer wg.Done()
		fuzzyFinder, fuzzyErr = initFuzzyFinder(tzfrellite.PreindexData)
	}()
	finder, err := initCompressedFinder(tzfrellite.LiteCompressData)
	wg.Wait()
	if err := errors.Join(fuzzyErr, err); err != nil {
		return nil, fmt.Errorf("tzf: init finders: %w", err)
	}

	if finder.DataVersion() != fuzzyFinder.DataVersion() {
		return nil, fmt.Errorf(
			"tzf: DefaultFinder only support same data version for Finder(version=%v) and FuzzyFinder(version=%v)",
			finder.DataVersion(),
			fuzzyFinder.DataVersion(),
		)
	}

	// Force free mem by probuf, about 50~80MB
	runtime.GC()

	return &DefaultFinder{fuzzyFinder: fuzzyFinder, finder: finder}, nil
}

// initFuzzyFinder initializes a fuzzy finder from preindex data
func initFuzzyFinder(data []byte) (F, error) {
	input := &pb.PreindexTimezones{}
	if err := proto.Unmarshal(data, input); err != nil {
		return nil, err
	}
	return NewFuzzyFinderFromPB(input)
}

// initCompressedFinder initializes a compressed finder from compressed data
func initCompressedFinder(data []byte) (F, error) {
	input := &pb.CompressedTimezones{}
	if err := proto.Unmarshal(data, input); err != nil {
		return nil, err
	}
	return NewFinderFromCompressed(input, SetDropPBTZ)
}

// probeOffsets are tried in order when no zone contains the point itself (gaps left by simplified borders).
var probeOffsets = [...][2]float64{{-0.02, -0.02}, {-0.02, 0.02}, {0.02, -0.02}, {0.02, 0.02}}

func (f *DefaultFinder) GetTimezoneName(lng float64, lat float64) string {
	if name := f.nameAt(lng, lat); name != "" {
		return name
	}
	for _, d := range probeOffsets {
		if name := f.nameAt(lng+d[0], lat+d[1]); name != "" {
			return name
		}
	}
	return ""
}

func (f *DefaultFinder) nameAt(lng, lat float64) string {
	if name := f.fuzzyFinder.GetTimezoneName(lng, lat); name != "" {
		return name
	}
	return f.finder.GetTimezoneName(lng, lat)
}

func (f *DefaultFinder) GetTimezoneNames(lng float64, lat float64) ([]string, error) {
	return f.finder.GetTimezoneNames(lng, lat)
}

func (f *DefaultFinder) TimezoneNames() []string {
	return f.finder.TimezoneNames()
}

func (f *DefaultFinder) DataVersion() string {
	return f.finder.DataVersion()
}
