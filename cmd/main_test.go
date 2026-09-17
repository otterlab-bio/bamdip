package main

import "testing"

func TestValidateArgsSamplingModeMutualExclusion(t *testing.T) {
	tests := []struct {
		name    string
		args    *CLIArgs
		wantErr bool
	}{
		{
			name: "ratio only",
			args: &CLIArgs{
				Ratio:      0.1,
				SortOrder:  "none",
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: false,
		},
		{
			name: "count only (default first N)",
			args: &CLIArgs{
				Count:      100,
				SortOrder:  "none",
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: false,
		},
		{
			name: "count with random flag",
			args: &CLIArgs{
				Count:      100,
				Random:     true,
				SortOrder:  "none",
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: false,
		},
		{
			name: "explicit zero count",
			args: &CLIArgs{
				HasCount:   true,
				Count:      0,
				SortOrder:  "none",
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: false,
		},
		{
			name: "missing ratio and count",
			args: &CLIArgs{
				SortOrder:  "none",
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: true,
		},
		{
			name: "ratio and count both set",
			args: &CLIArgs{
				Ratio:      0.1,
				Count:      100,
				SortOrder:  "none",
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: true,
		},
		{
			name: "negative count",
			args: &CLIArgs{
				Count:      -5,
				SortOrder:  "none",
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: true,
		},
		{
			name: "ratio out of bounds (> 1)",
			args: &CLIArgs{
				Ratio:      1.5,
				SortOrder:  "none",
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: true,
		},
		{
			name: "ratio out of bounds (< 0)",
			args: &CLIArgs{
				Ratio:      -0.1,
				SortOrder:  "none",
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		err := validateArgs(tc.args)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
	}
}

func TestValidatePositionalArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    *CLIArgs
		wantErr bool
	}{
		{
			name: "needs two args, given one",
			args: &CLIArgs{
				InputFiles: []string{"in.bam"},
			},
			wantErr: true,
		},
		{
			name: "needs two args, given two",
			args: &CLIArgs{
				InputFiles: []string{"in.bam", "out.bam"},
			},
			wantErr: false,
		},
		{
			name: "needs two args, given three",
			args: &CLIArgs{
				InputFiles: []string{"in.bam", "out.bam", "extra.bam"},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		err := validatePositionalArgs(tc.args)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
	}
}

func TestValidateArgsIndexRequiresCoordSort(t *testing.T) {
	tests := []struct {
		name    string
		args    *CLIArgs
		wantErr bool
	}{
		{
			name: "index without coord sort errors",
			args: &CLIArgs{
				Ratio:       0.1,
				SortOrder:   "none",
				CreateIndex: true,
				InputFiles:  []string{"in.bam", "out.bam"},
			},
			wantErr: true,
		},
		{
			name: "index with name sort errors",
			args: &CLIArgs{
				Ratio:       0.1,
				SortOrder:   "name",
				CreateIndex: true,
				InputFiles:  []string{"in.bam", "out.bam"},
			},
			wantErr: true,
		},
		{
			name: "index with coord sort passes",
			args: &CLIArgs{
				Ratio:       0.1,
				SortOrder:   "coord",
				CreateIndex: true,
				InputFiles:  []string{"in.bam", "out.bam"},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		err := validateArgs(tc.args)
		if tc.wantErr && err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
	}
}
