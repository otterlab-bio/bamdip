package sampler

import (
	"math/rand"

	bamnative "github.com/otterlab-bio/bamdriver/pkg/bamnative"
)

// SamplingMode 采样模式
type SamplingMode int

const (
	ModeCount SamplingMode = iota // 按记录数采样
	ModeRatio                     // 按比例采样
)

// SortOrder 排序方式常量
const (
	SortByName  = "name"
	SortByCoord = "coord"
	SortNone    = "none"
)

// SamplingConfig 采样配置
type SamplingConfig struct {
	Mode        SamplingMode // ModeCount 或 ModeRatio
	Count       int64        // 采样条数 (当 Mode == ModeCount 时生效)
	Ratio       float64      // 采样比例 (0.0-1.0，当 Mode == ModeRatio 时生效)
	Random      bool         // 是否随机采样 (在 Count 模式下：false 为默认前 N 条，true 为随机采样 N 条)
	Seed        int64        // 随机种子 (可选)
	SortOrder   string       // SortByName, SortByCoord, 或 SortNone
	CreateIndex bool         // 创建 BAI 索引 (要求 SortOrder == SortByCoord)
}

// SamplingStats 采样统计信息
type SamplingStats struct {
	TotalInputRecords  int64 // 输入总记录数
	TotalOutputRecords int64 // 输出总记录数
}

// Reservoir 蓄水池结构
type Reservoir struct {
	capacity int64
	size     int64
	records  []*bamnative.Record
	rand     *rand.Rand
}
