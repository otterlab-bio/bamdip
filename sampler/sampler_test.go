package sampler

import (
	"fmt"
	"testing"

	bamnative "github.com/otterlab-bio/bamdriver/pkg/bamnative"
)

func testRecord(name string) *bamnative.Record {
	return &bamnative.Record{
		Name:      name,
		Flags:     0,
		RefID:     0,
		Pos:       100,
		MapQ:      60,
		Cigar:     []bamnative.CigarOp{{Op: 'M', Len: 4}},
		MateRefID: -1,
		MatePos:   0,
		TLen:      0,
		Seq:       "ACGT",
		Qual:      []byte{30, 30, 30, 30},
		Aux:       []*bamnative.AuxField{},
	}
}

// TestReservoirSamplesAll verifies that when total items <= capacity, every item is kept.
func TestReservoirSamplesAll(t *testing.T) {
	r := NewReservoir(10, 42)
	for i := 0; i < 5; i++ {
		r.Add(testRecord(fmt.Sprintf("r%d", i)))
	}
	got := r.GetRecords()
	if len(got) != 5 {
		t.Errorf("expected 5 records, got %d", len(got))
	}
}

// TestReservoirExactCapacity verifies that when items == capacity, all are kept.
func TestReservoirExactCapacity(t *testing.T) {
	r := NewReservoir(5, 42)
	for i := 0; i < 5; i++ {
		r.Add(testRecord(fmt.Sprintf("r%d", i)))
	}
	got := r.GetRecords()
	if len(got) != 5 {
		t.Errorf("expected 5 records, got %d", len(got))
	}
}

// TestReservoirRespectsCapacity verifies that the reservoir never exceeds its capacity.
func TestReservoirRespectsCapacity(t *testing.T) {
	const cap int64 = 10
	r := NewReservoir(cap, 42)
	for i := 0; i < 1000; i++ {
		r.Add(testRecord(fmt.Sprintf("r%d", i)))
	}
	got := r.GetRecords()
	if int64(len(got)) != cap {
		t.Errorf("expected %d records, got %d", cap, len(got))
	}
}

// TestReservoirSizeTracks verifies that Size() reflects all items ever added.
func TestReservoirSizeTracks(t *testing.T) {
	r := NewReservoir(5, 42)
	for i := 0; i < 100; i++ {
		r.Add(testRecord(fmt.Sprintf("r%d", i)))
	}
	if r.Size() != 100 {
		t.Errorf("expected Size()=100, got %d", r.Size())
	}
}

// TestReservoirDeterministic verifies that the same seed produces the same sample.
func TestReservoirDeterministic(t *testing.T) {
	const n = 1000
	const cap int64 = 50

	run := func() []*bamnative.Record {
		r := NewReservoir(cap, 12345)
		for i := 0; i < n; i++ {
			r.Add(testRecord(fmt.Sprintf("r%d", i)))
		}
		return r.GetRecords()
	}

	a := run()
	b := run()

	if len(a) != len(b) {
		t.Fatalf("length mismatch: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Name != b[i].Name {
			t.Errorf("index %d: %v != %v", i, a[i].Name, b[i].Name)
		}
	}
}

// TestReservoirCapacityAccessor verifies Capacity() returns the configured value.
func TestReservoirCapacityAccessor(t *testing.T) {
	r := NewReservoir(42, 1)
	if r.Capacity() != 42 {
		t.Errorf("expected capacity 42, got %d", r.Capacity())
	}
}
