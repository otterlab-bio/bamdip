package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/otterlab-bio/bamdip/sampler"
)

// CLIArgs 命令行参数结构体
type CLIArgs struct {
	Count       int64   // 采样条数 (-n 或 -count)
	HasCount    bool    // 是否显式指定了 count
	Ratio       float64 // 采样比例 (-ratio)
	HasRatio    bool    // 是否显式指定了 ratio
	Random      bool    // 是否随机采样 (-random)
	Seed        int64   // 随机种子 (-seed)
	SortOrder   string  // 排序方式 (-sort)
	CreateIndex bool    // 创建 BAI 索引 (-index)
	InputFiles  []string
}

func main() {
	args := parseArgs()

	if err := validateArgs(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		usage()
		os.Exit(1)
	}

	config := createConfig(args)
	s := sampler.NewBAMSampler(config)

	startTime := time.Now()
	err := s.Sample(args.InputFiles[0], args.InputFiles[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	stats := s.GetStats()
	elapsed := time.Since(startTime)

	fmt.Printf("=== bamdip Complete ===\n")
	fmt.Printf("Total input records:  %d\n", stats.TotalInputRecords)
	fmt.Printf("Total output records: %d\n", stats.TotalOutputRecords)
	fmt.Printf("Time elapsed:         %v\n", elapsed)
}

func parseArgs() *CLIArgs {
	args := &CLIArgs{}

	var countShort, countLong int64
	flag.Int64Var(&countShort, "n", -1, "Number of alignment records to sample (default: first N; with -random: random N)")
	flag.Int64Var(&countLong, "count", -1, "Alias for -n")
	flag.Float64Var(&args.Ratio, "ratio", -1.0, "Sampling ratio (0.0-1.0), e.g., 0.1 for 10%")
	flag.BoolVar(&args.Random, "random", false, "Perform random downsampling instead of default first-N")
	flag.Int64Var(&args.Seed, "seed", 0, "Random seed (optional, for reproducible sampling)")
	flag.StringVar(&args.SortOrder, "sort", "none", "Sort order for output: name, coord, or none (default: none)")
	flag.BoolVar(&args.CreateIndex, "index", false, "Create BAI index for output (requires -sort coord)")

	flag.Usage = usage
	flag.Parse()

	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "n":
			args.HasCount = true
			args.Count = countShort
		case "count":
			args.HasCount = true
			args.Count = countLong
		case "ratio":
			args.HasRatio = true
		}
	})

	args.InputFiles = flag.Args()
	return args
}

func validateArgs(args *CLIArgs) error {
	hasCount := args.HasCount || args.Count > 0
	hasRatio := args.HasRatio || args.Ratio > 0

	if !hasCount && !hasRatio {
		return fmt.Errorf("either -n/-count or -ratio must be specified")
	}
	if hasCount && hasRatio {
		return fmt.Errorf("only one of -n/-count or -ratio can be specified")
	}

	if hasRatio && (args.Ratio < 0 || args.Ratio > 1) {
		return fmt.Errorf("ratio must be between 0.0 and 1.0")
	}

	if hasCount && args.Count < 0 {
		return fmt.Errorf("count must be non-negative")
	}

	if err := validatePositionalArgs(args); err != nil {
		return err
	}

	if args.SortOrder != sampler.SortByName && args.SortOrder != sampler.SortByCoord && args.SortOrder != sampler.SortNone {
		return fmt.Errorf("sort order must be 'name', 'coord', or 'none'")
	}

	if args.CreateIndex && args.SortOrder != sampler.SortByCoord {
		return fmt.Errorf("-index requires -sort coord")
	}

	return nil
}

func validatePositionalArgs(args *CLIArgs) error {
	if len(args.InputFiles) != 2 {
		return fmt.Errorf("requires exactly 2 positional arguments: <input.bam> <output.bam>")
	}
	return nil
}

func createConfig(args *CLIArgs) *sampler.SamplingConfig {
	config := &sampler.SamplingConfig{
		Ratio:       args.Ratio,
		Count:       args.Count,
		Random:      args.Random,
		Seed:        args.Seed,
		SortOrder:   args.SortOrder,
		CreateIndex: args.CreateIndex,
	}

	if args.Ratio != 0 {
		config.Mode = sampler.ModeRatio
	} else {
		config.Mode = sampler.ModeCount
	}

	return config
}

func usage() {
	fmt.Fprintf(os.Stderr, `bamdip - Subsample BAM alignment records (first-N, random-N, or ratio)

Usage:
  bamdip [options] <input.bam> <output.bam>

Sampling options (choose one):
  -n INT, -count INT    Number of alignment records to sample (default: first N records)
  -ratio FLOAT          Sampling ratio (0.0-1.0), e.g., 0.1 for 10%%
  -random               Perform random downsampling instead of default first-N
  -seed INT             Random seed (optional, for reproducible results)

Output options:
  -sort string          Sort order for output: name, coord, or none (default: none)
  -index                Create BAI index for output (requires -sort coord)

Examples:
  # Extract the first 1000 alignments
  bamdip -n 1000 input.bam output.bam

  # Randomly downsample 1000 alignments with seed
  bamdip -n 1000 -random -seed 42 input.bam output.bam

  # Randomly sample 10%% of alignments
  bamdip -ratio 0.1 input.bam output.bam

  # Extract first 5000 alignments, sort by coordinate and generate BAI index
  bamdip -n 5000 -sort coord -index input.bam output.bam
`)
}
