package sampler

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"sort"
	"time"

	bamnative "github.com/otterlab-bio/bamdriver/pkg/bamnative"
)

// ============================================================================
// 蓄水池算法实现
// ============================================================================

// NewReservoir 创建新的蓄水池
func NewReservoir(capacity int64, seed int64) *Reservoir {
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	return &Reservoir{
		capacity: capacity,
		records:  make([]*bamnative.Record, 0, capacity),
		rand:     rand.New(rand.NewSource(seed)),
	}
}

// Add 向蓄水池添加记录
func (r *Reservoir) Add(record *bamnative.Record) {
	r.size++
	if int64(len(r.records)) < r.capacity {
		r.records = append(r.records, record)
	} else if r.capacity > 0 {
		j := r.rand.Int63n(r.size)
		if j < r.capacity {
			r.records[j] = record
		}
	}
}

// GetRecords 获取蓄水池中的所有记录
func (r *Reservoir) GetRecords() []*bamnative.Record {
	result := make([]*bamnative.Record, len(r.records))
	copy(result, r.records)
	return result
}

// Size 获取处理过的总记录数
func (r *Reservoir) Size() int64 {
	return r.size
}

// Capacity 获取蓄水池容量
func (r *Reservoir) Capacity() int64 {
	return r.capacity
}

// ============================================================================
// BAM采样器主类型
// ============================================================================

// BAMSampler BAM记录采样器
type BAMSampler struct {
	config *SamplingConfig
	stats  *SamplingStats
}

// NewBAMSampler 创建新的采样器
func NewBAMSampler(config *SamplingConfig) *BAMSampler {
	return &BAMSampler{
		config: config,
		stats:  &SamplingStats{},
	}
}

// GetStats 获取采样统计信息
func (s *BAMSampler) GetStats() *SamplingStats {
	return s.stats
}

// Sample 执行BAM采样
func (s *BAMSampler) Sample(inputPath, outputPath string) error {
	if s.config.CreateIndex && s.config.SortOrder != SortByCoord {
		return errors.New("-index requires -sort coord")
	}

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input BAM: %w", err)
	}
	defer inputFile.Close()

	reader, err := bamnative.NewReader(inputFile)
	if err != nil {
		return fmt.Errorf("read input BAM: %w", err)
	}
	header := reader.Header()

	var outputRecords []*bamnative.Record

	if s.config.Mode == ModeCount && !s.config.Random {
		// First-N 模式：按输入顺序取前 N 条
		targetCount := s.config.Count
		if targetCount < 0 {
			targetCount = 0
		}
		for {
			rec, readErr := reader.Read()
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				return fmt.Errorf("read record at %d: %w", s.stats.TotalInputRecords, readErr)
			}
			s.stats.TotalInputRecords++
			if int64(len(outputRecords)) < targetCount {
				outputRecords = append(outputRecords, rec)
			}
		}
	} else if s.config.Mode == ModeCount && s.config.Random {
		// Random-N 模式：单遍蓄水池采样
		targetCount := s.config.Count
		if targetCount < 0 {
			targetCount = 0
		}
		reservoir := NewReservoir(targetCount, s.config.Seed)
		for {
			rec, readErr := reader.Read()
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				return fmt.Errorf("read record at %d: %w", s.stats.TotalInputRecords, readErr)
			}
			s.stats.TotalInputRecords++
			reservoir.Add(rec)
		}
		outputRecords = reservoir.GetRecords()
	} else {
		// ModeRatio 模式：先统计总数，再按比例采样
		for {
			_, readErr := reader.Read()
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				return fmt.Errorf("scan record at %d: %w", s.stats.TotalInputRecords, readErr)
			}
			s.stats.TotalInputRecords++
		}

		targetCount := int64(float64(s.stats.TotalInputRecords) * s.config.Ratio)
		if targetCount < 0 {
			targetCount = 0
		}
		if targetCount > s.stats.TotalInputRecords {
			targetCount = s.stats.TotalInputRecords
		}

		// 第二遍读取并蓄水池抽样
		if _, seekErr := inputFile.Seek(0, io.SeekStart); seekErr != nil {
			return fmt.Errorf("seek input BAM: %w", seekErr)
		}
		pass2Reader, err := bamnative.NewReader(inputFile)
		if err != nil {
			return fmt.Errorf("reopen input BAM: %w", err)
		}
		reservoir := NewReservoir(targetCount, s.config.Seed)
		for {
			rec, readErr := pass2Reader.Read()
			if errors.Is(readErr, io.EOF) {
				break
			}
			if readErr != nil {
				return fmt.Errorf("read pass2 record: %w", readErr)
			}
			reservoir.Add(rec)
		}
		outputRecords = reservoir.GetRecords()
	}

	s.stats.TotalOutputRecords = int64(len(outputRecords))

	// 排序与更新 Header
	switch s.config.SortOrder {
	case SortByCoord:
		sortRecordsByCoord(outputRecords)
		header.SortOrder = "coordinate"
	case SortByName:
		sortRecordsByName(outputRecords)
		header.SortOrder = "queryname"
	default:
		// 保持原样或 unknown
	}

	// 写入输出 BAM
	if err := s.writeOutput(outputRecords, outputPath, header); err != nil {
		return fmt.Errorf("write output BAM: %w", err)
	}

	// 索引
	if s.config.CreateIndex {
		if err := bamnative.BuildIndex(outputPath); err != nil {
			return fmt.Errorf("build BAI index: %w", err)
		}
	}

	return nil
}

// ============================================================================
// 排序辅助函数
// ============================================================================

func sortRecordsByCoord(records []*bamnative.Record) {
	sort.Slice(records, func(i, j int) bool {
		ri, rj := records[i], records[j]
		if ri.RefID != rj.RefID {
			return ri.RefID < rj.RefID
		}
		return ri.Pos < rj.Pos
	})
}

func sortRecordsByName(records []*bamnative.Record) {
	sort.Slice(records, func(i, j int) bool {
		return records[i].Name < records[j].Name
	})
}

// ============================================================================
// 写入辅助函数
// ============================================================================

func (s *BAMSampler) writeOutput(records []*bamnative.Record, outputPath string, header *bamnative.Header) error {
	writer, err := bamnative.NewWriter(outputPath, header)
	if err != nil {
		return fmt.Errorf("create BAM writer: %w", err)
	}
	defer writer.Close()

	for _, rec := range records {
		if err := writer.Write(rec); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}
	return nil
}
