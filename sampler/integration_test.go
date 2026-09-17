package sampler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	bamnative "github.com/otterlab-bio/bamdriver/pkg/bamnative"
)

func itMakeRecord(name string, flags uint16, pos int32) *bamnative.Record {
	return &bamnative.Record{
		Name:      name,
		Flags:     flags,
		RefID:     0,
		Pos:       pos,
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

func itWriteBAM(t *testing.T, records []*bamnative.Record) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.bam")
	header := &bamnative.Header{
		SortOrder:  "unknown",
		OtherLines: make(map[string]string),
		References: []*bamnative.Reference{
			{ID: 0, Name: "chr1", Len: 100000},
		},
	}
	w, err := bamnative.NewWriter(path, header)
	if err != nil {
		t.Fatalf("itWriteBAM: NewWriter: %v", err)
	}
	for _, rec := range records {
		if err := w.Write(rec); err != nil {
			_ = w.Close()
			t.Fatalf("itWriteBAM: Write: %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("itWriteBAM: Close: %v", err)
	}
	return path
}

func itReadBAM(t *testing.T, path string) []*bamnative.Record {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("itReadBAM: Open %s: %v", path, err)
	}
	defer f.Close()
	reader, err := bamnative.NewReader(f)
	if err != nil {
		t.Fatalf("itReadBAM: NewReader: %v", err)
	}
	var records []*bamnative.Record
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("itReadBAM: Read: %v", err)
		}
		records = append(records, rec)
	}
	return records
}

// TestSample_FirstN_Default verifies that -count N (with Random=false) selects the exact first N records
func TestSample_FirstN_Default(t *testing.T) {
	var inputRecs []*bamnative.Record
	for i := 0; i < 20; i++ {
		inputRecs = append(inputRecs, itMakeRecord(fmt.Sprintf("read_%02d", i), 0, int32(i*10)))
	}
	inPath := itWriteBAM(t, inputRecs)
	outPath := filepath.Join(t.TempDir(), "out.bam")

	cfg := &SamplingConfig{
		Mode:      ModeCount,
		Count:     5,
		Random:    false,
		SortOrder: SortNone,
	}
	s := NewBAMSampler(cfg)
	if err := s.Sample(inPath, outPath); err != nil {
		t.Fatalf("Sample failed: %v", err)
	}

	stats := s.GetStats()
	if stats.TotalInputRecords != 20 {
		t.Errorf("expected TotalInputRecords=20, got %d", stats.TotalInputRecords)
	}
	if stats.TotalOutputRecords != 5 {
		t.Errorf("expected TotalOutputRecords=5, got %d", stats.TotalOutputRecords)
	}

	outRecs := itReadBAM(t, outPath)
	if len(outRecs) != 5 {
		t.Fatalf("expected 5 output records, got %d", len(outRecs))
	}
	for i := 0; i < 5; i++ {
		expectedName := fmt.Sprintf("read_%02d", i)
		if outRecs[i].Name != expectedName {
			t.Errorf("record[%d]: expected Name=%s, got %s", i, expectedName, outRecs[i].Name)
		}
	}
}

// TestSample_FirstN_MultiMappers verifies RNA-seq scenario where 3-4 records have the same QNAME
func TestSample_FirstN_MultiMappers(t *testing.T) {
	var inputRecs []*bamnative.Record
	// 4 records with the exact same name
	inputRecs = append(inputRecs, itMakeRecord("multiread", 0, 100))
	inputRecs = append(inputRecs, itMakeRecord("multiread", 0, 200))
	inputRecs = append(inputRecs, itMakeRecord("multiread", 0, 300))
	inputRecs = append(inputRecs, itMakeRecord("multiread", 0, 400))
	inputRecs = append(inputRecs, itMakeRecord("other_read", 0, 500))

	inPath := itWriteBAM(t, inputRecs)
	outPath := filepath.Join(t.TempDir(), "out.bam")

	cfg := &SamplingConfig{
		Mode:      ModeCount,
		Count:     4,
		Random:    false,
		SortOrder: SortNone,
	}
	s := NewBAMSampler(cfg)
	if err := s.Sample(inPath, outPath); err != nil {
		t.Fatalf("Sample failed: %v", err)
	}

	outRecs := itReadBAM(t, outPath)
	if len(outRecs) != 4 {
		t.Fatalf("expected 4 records, got %d", len(outRecs))
	}
	// All 4 multiread records should be preserved
	for i := 0; i < 4; i++ {
		if outRecs[i].Name != "multiread" {
			t.Errorf("record[%d]: expected Name=multiread, got %s", i, outRecs[i].Name)
		}
		expectedPos := int32((i + 1) * 100)
		if outRecs[i].Pos != expectedPos {
			t.Errorf("record[%d]: expected Pos=%d, got %d", i, expectedPos, outRecs[i].Pos)
		}
	}
}

// TestSample_FirstN_ExceedsTotal verifies that requesting more records than input outputs all records
func TestSample_FirstN_ExceedsTotal(t *testing.T) {
	var inputRecs []*bamnative.Record
	for i := 0; i < 3; i++ {
		inputRecs = append(inputRecs, itMakeRecord(fmt.Sprintf("r%d", i), 0, int32(i*10)))
	}
	inPath := itWriteBAM(t, inputRecs)
	outPath := filepath.Join(t.TempDir(), "out.bam")

	cfg := &SamplingConfig{
		Mode:      ModeCount,
		Count:     100,
		Random:    false,
		SortOrder: SortNone,
	}
	s := NewBAMSampler(cfg)
	if err := s.Sample(inPath, outPath); err != nil {
		t.Fatalf("Sample failed: %v", err)
	}

	outRecs := itReadBAM(t, outPath)
	if len(outRecs) != 3 {
		t.Fatalf("expected 3 records, got %d", len(outRecs))
	}
}

// TestSample_FirstN_ZeroCount verifies that Count=0 outputs 0 records with valid header
func TestSample_FirstN_ZeroCount(t *testing.T) {
	var inputRecs []*bamnative.Record
	inputRecs = append(inputRecs, itMakeRecord("r0", 0, 10))
	inPath := itWriteBAM(t, inputRecs)
	outPath := filepath.Join(t.TempDir(), "out.bam")

	cfg := &SamplingConfig{
		Mode:      ModeCount,
		Count:     0,
		Random:    false,
		SortOrder: SortNone,
	}
	s := NewBAMSampler(cfg)
	if err := s.Sample(inPath, outPath); err != nil {
		t.Fatalf("Sample failed: %v", err)
	}

	outRecs := itReadBAM(t, outPath)
	if len(outRecs) != 0 {
		t.Fatalf("expected 0 records, got %d", len(outRecs))
	}
}

// TestSample_RandomN verifies random sampling of N records with reproducible seed
func TestSample_RandomN(t *testing.T) {
	var inputRecs []*bamnative.Record
	for i := 0; i < 100; i++ {
		inputRecs = append(inputRecs, itMakeRecord(fmt.Sprintf("read_%03d", i), 0, int32(i*10)))
	}
	inPath := itWriteBAM(t, inputRecs)
	outPath1 := filepath.Join(t.TempDir(), "out1.bam")
	outPath2 := filepath.Join(t.TempDir(), "out2.bam")

	cfg1 := &SamplingConfig{
		Mode:      ModeCount,
		Count:     15,
		Random:    true,
		Seed:      999,
		SortOrder: SortNone,
	}
	s1 := NewBAMSampler(cfg1)
	if err := s1.Sample(inPath, outPath1); err != nil {
		t.Fatalf("Sample run 1 failed: %v", err)
	}

	cfg2 := &SamplingConfig{
		Mode:      ModeCount,
		Count:     15,
		Random:    true,
		Seed:      999,
		SortOrder: SortNone,
	}
	s2 := NewBAMSampler(cfg2)
	if err := s2.Sample(inPath, outPath2); err != nil {
		t.Fatalf("Sample run 2 failed: %v", err)
	}

	recs1 := itReadBAM(t, outPath1)
	recs2 := itReadBAM(t, outPath2)

	if len(recs1) != 15 || len(recs2) != 15 {
		t.Fatalf("expected 15 records in both outputs, got %d and %d", len(recs1), len(recs2))
	}

	for i := 0; i < 15; i++ {
		if recs1[i].Name != recs2[i].Name || recs1[i].Pos != recs2[i].Pos {
			t.Errorf("seed reproducibility mismatch at [%d]: %s vs %s", i, recs1[i].Name, recs2[i].Name)
		}
	}
}

// TestSample_Ratio verifies ratio sampling mode
func TestSample_Ratio(t *testing.T) {
	var inputRecs []*bamnative.Record
	for i := 0; i < 80; i++ {
		inputRecs = append(inputRecs, itMakeRecord(fmt.Sprintf("read_%03d", i), 0, int32(i*10)))
	}
	inPath := itWriteBAM(t, inputRecs)
	outPath := filepath.Join(t.TempDir(), "out.bam")

	cfg := &SamplingConfig{
		Mode:      ModeRatio,
		Ratio:     0.25,
		Seed:      777,
		SortOrder: SortNone,
	}
	s := NewBAMSampler(cfg)
	if err := s.Sample(inPath, outPath); err != nil {
		t.Fatalf("Sample failed: %v", err)
	}

	stats := s.GetStats()
	if stats.TotalInputRecords != 80 {
		t.Errorf("expected TotalInputRecords=80, got %d", stats.TotalInputRecords)
	}
	if stats.TotalOutputRecords != 20 {
		t.Errorf("expected TotalOutputRecords=20, got %d", stats.TotalOutputRecords)
	}

	outRecs := itReadBAM(t, outPath)
	if len(outRecs) != 20 {
		t.Fatalf("expected 20 records, got %d", len(outRecs))
	}
}

// TestSample_SortCoord verifies that -sort coord sorts records by coordinate and sets header
func TestSample_SortCoord(t *testing.T) {
	var inputRecs []*bamnative.Record
	inputRecs = append(inputRecs, itMakeRecord("r3", 0, 500))
	inputRecs = append(inputRecs, itMakeRecord("r1", 0, 100))
	inputRecs = append(inputRecs, itMakeRecord("r2", 0, 300))

	inPath := itWriteBAM(t, inputRecs)
	outPath := filepath.Join(t.TempDir(), "out.bam")

	cfg := &SamplingConfig{
		Mode:      ModeCount,
		Count:     3,
		Random:    false,
		SortOrder: SortByCoord,
	}
	s := NewBAMSampler(cfg)
	if err := s.Sample(inPath, outPath); err != nil {
		t.Fatalf("Sample failed: %v", err)
	}

	outRecs := itReadBAM(t, outPath)
	if len(outRecs) != 3 {
		t.Fatalf("expected 3 records, got %d", len(outRecs))
	}
	if outRecs[0].Pos != 100 || outRecs[1].Pos != 300 || outRecs[2].Pos != 500 {
		t.Errorf("records not in coordinate order: %d, %d, %d", outRecs[0].Pos, outRecs[1].Pos, outRecs[2].Pos)
	}

	// Verify header
	f, _ := os.Open(outPath)
	defer f.Close()
	r, _ := bamnative.NewReader(f)
	if r.Header().SortOrder != "coordinate" {
		t.Errorf("expected header SortOrder=coordinate, got %s", r.Header().SortOrder)
	}
}

// TestSample_SortName verifies that -sort name sorts records alphabetically by QNAME
func TestSample_SortName(t *testing.T) {
	var inputRecs []*bamnative.Record
	inputRecs = append(inputRecs, itMakeRecord("zebra", 0, 100))
	inputRecs = append(inputRecs, itMakeRecord("apple", 0, 200))
	inputRecs = append(inputRecs, itMakeRecord("banana", 0, 300))

	inPath := itWriteBAM(t, inputRecs)
	outPath := filepath.Join(t.TempDir(), "out.bam")

	cfg := &SamplingConfig{
		Mode:      ModeCount,
		Count:     3,
		Random:    false,
		SortOrder: SortByName,
	}
	s := NewBAMSampler(cfg)
	if err := s.Sample(inPath, outPath); err != nil {
		t.Fatalf("Sample failed: %v", err)
	}

	outRecs := itReadBAM(t, outPath)
	if len(outRecs) != 3 {
		t.Fatalf("expected 3 records, got %d", len(outRecs))
	}
	if outRecs[0].Name != "apple" || outRecs[1].Name != "banana" || outRecs[2].Name != "zebra" {
		t.Errorf("records not in name order: %s, %s, %s", outRecs[0].Name, outRecs[1].Name, outRecs[2].Name)
	}

	f, _ := os.Open(outPath)
	defer f.Close()
	r, _ := bamnative.NewReader(f)
	if r.Header().SortOrder != "queryname" {
		t.Errorf("expected header SortOrder=queryname, got %s", r.Header().SortOrder)
	}
}

// TestSample_IndexRequiresCoordSort verifies -index without -sort coord returns an error
func TestSample_IndexRequiresCoordSort(t *testing.T) {
	inPath := itWriteBAM(t, []*bamnative.Record{itMakeRecord("r1", 0, 100)})
	outPath := filepath.Join(t.TempDir(), "out.bam")

	cfg := &SamplingConfig{
		Mode:        ModeCount,
		Count:       1,
		SortOrder:   SortNone,
		CreateIndex: true,
	}
	s := NewBAMSampler(cfg)
	err := s.Sample(inPath, outPath)
	if err == nil {
		t.Fatalf("expected error when -index without -sort coord, got nil")
	}
}

// TestSample_IndexBuildsBAI verifies -index creates a BAI file
func TestSample_IndexBuildsBAI(t *testing.T) {
	inPath := itWriteBAM(t, []*bamnative.Record{itMakeRecord("r1", 0, 100)})
	outPath := filepath.Join(t.TempDir(), "out.bam")

	cfg := &SamplingConfig{
		Mode:        ModeCount,
		Count:       1,
		SortOrder:   SortByCoord,
		CreateIndex: true,
	}
	s := NewBAMSampler(cfg)
	if err := s.Sample(inPath, outPath); err != nil {
		t.Fatalf("Sample failed: %v", err)
	}

	baiPath := outPath + ".bai"
	fi, err := os.Stat(baiPath)
	if err != nil {
		t.Fatalf("expected BAI index at %s, got error: %v", baiPath, err)
	}
	if fi.Size() == 0 {
		t.Errorf("BAI index file %s is empty", baiPath)
	}
}

// TestSample_RealBAMDriverData verifies sampling on the real bamdriver fixture
func TestSample_RealBAMDriverData(t *testing.T) {
	fixturePath := filepath.Join("..", "testdata", "Test_hg19_NRAS.bam")
	if _, err := os.Stat(fixturePath); err != nil {
		t.Skipf("skipping test: fixture not found at %s", fixturePath)
	}

	// 1. Test First 100
	outPath100 := filepath.Join(t.TempDir(), "first_100.bam")
	cfg100 := &SamplingConfig{
		Mode:      ModeCount,
		Count:     100,
		Random:    false,
		SortOrder: SortNone,
	}
	s100 := NewBAMSampler(cfg100)
	if err := s100.Sample(fixturePath, outPath100); err != nil {
		t.Fatalf("Sample first 100 failed: %v", err)
	}

	stats100 := s100.GetStats()
	if stats100.TotalInputRecords != 1285 {
		t.Errorf("expected 1285 total input records, got %d", stats100.TotalInputRecords)
	}
	if stats100.TotalOutputRecords != 100 {
		t.Errorf("expected 100 output records, got %d", stats100.TotalOutputRecords)
	}

	// Compare with original first 100
	origRecs := itReadBAM(t, fixturePath)
	sampledRecs := itReadBAM(t, outPath100)
	if len(sampledRecs) != 100 {
		t.Fatalf("expected 100 sampled records, got %d", len(sampledRecs))
	}
	for i := 0; i < 100; i++ {
		orig := origRecs[i]
		sampled := sampledRecs[i]
		if orig.Name != sampled.Name {
			t.Errorf("record[%d]: Name mismatch: %s vs %s", i, orig.Name, sampled.Name)
		}
		if orig.Flags != sampled.Flags {
			t.Errorf("record[%d]: Flags mismatch: %d vs %d", i, orig.Flags, sampled.Flags)
		}
		if orig.Pos != sampled.Pos {
			t.Errorf("record[%d]: Pos mismatch: %d vs %d", i, orig.Pos, sampled.Pos)
		}
		if orig.Seq != sampled.Seq {
			t.Errorf("record[%d]: Seq mismatch", i)
		}
	}

	// 2. Test Ratio 20%
	outPathRatio := filepath.Join(t.TempDir(), "ratio_20.bam")
	cfgRatio := &SamplingConfig{
		Mode:      ModeRatio,
		Ratio:     0.20,
		Seed:      42,
		SortOrder: SortNone,
	}
	sRatio := NewBAMSampler(cfgRatio)
	if err := sRatio.Sample(fixturePath, outPathRatio); err != nil {
		t.Fatalf("Sample ratio failed: %v", err)
	}
	// 1285 * 0.20 = 257
	if sRatio.GetStats().TotalOutputRecords != 257 {
		t.Errorf("expected 257 output records, got %d", sRatio.GetStats().TotalOutputRecords)
	}

	// 3. Test Coord Sort and Index
	outPathCoord := filepath.Join(t.TempDir(), "coord_indexed.bam")
	cfgCoord := &SamplingConfig{
		Mode:        ModeCount,
		Count:       200,
		Random:      false,
		SortOrder:   SortByCoord,
		CreateIndex: true,
	}
	sCoord := NewBAMSampler(cfgCoord)
	if err := sCoord.Sample(fixturePath, outPathCoord); err != nil {
		t.Fatalf("Sample coord with index failed: %v", err)
	}
	if _, err := os.Stat(outPathCoord + ".bai"); err != nil {
		t.Errorf("expected BAI index file to exist: %v", err)
	}
}



