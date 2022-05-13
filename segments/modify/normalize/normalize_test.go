package normalize

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
	"testing"

	"github.com/bwNetFlow/flowpipeline/pb"
	"github.com/bwNetFlow/flowpipeline/segments"
)

// Normalize Segment test, in-flow SampleingRate test
func TestSegment_Normalize_inFlowSamplingRate(t *testing.T) {
	result := segments.TestSegment("normalize", map[string]string{},
		&pb.EnrichedFlow{SamplingRate: 32, Bytes: 1})
	if result.Bytes != 32 {
		t.Error("Segment Normalize is not working with in-flow SamplingRate.")
	}
}

// Normalize Segment test, fallback SampleingRate test
func TestSegment_Normalize_fallbackSamplingRate(t *testing.T) {
	result := segments.TestSegment("normalize", map[string]string{"fallback": "42"},
		&pb.EnrichedFlow{SamplingRate: 0, Bytes: 1})
	if result.Bytes != 42 {
		t.Error("Segment Normalize is not working with fallback SamplingRate.")
	}
}

// Normalize Segment test, no fallback SampleingRate test
func TestSegment_Normalize_noFallbackSamplingRate(t *testing.T) {
	result := segments.TestSegment("normalize", map[string]string{},
		&pb.EnrichedFlow{SamplingRate: 0, Bytes: 1})
	if result.Bytes != 1 {
		t.Error("Segment Normalize is not working with fallback SamplingRate.")
	}
}

// Normalize Segment benchmark passthrough
func BenchmarkNormalize(b *testing.B) {
	log.SetOutput(ioutil.Discard)
	os.Stdout, _ = os.Open(os.DevNull)

	segment := Normalize{}.New(map[string]string{})

	in, out := make(chan *pb.EnrichedFlow), make(chan *pb.EnrichedFlow)
	segment.Rewire(in, out)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go segment.Run(wg)

	for n := 0; n < b.N; n++ {
		in <- &pb.EnrichedFlow{SamplingRate: 0, Bytes: 1}
		_ = <-out
	}
	close(in)
}
