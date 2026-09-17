# bamdip 发布准备与验证清单

## 项目变更说明

本项目已由原 `bamsampler`（强制双端 pair 配对）重构并更名为 **`bamdip`**：
- 底层驱动升级为 `github.com/otterlab-bio/bamdriver v1.0.0`。
- 采样原子单位更新为单条 BAM alignment record，天然支持 RNA-seq 等场景中一条 QNAME 对应 3~4 条比对记录的情况。
- 模式规范：
  - `-n / -count N`：默认截取流式前 N 条记录。
  - `-random`：配合 `-n / -count` 开启单遍蓄水池随机下采样（支持 `-seed`）。
  - `-ratio R`：按比例采样（两遍流式蓄水池采样，支持 `-seed`）。
  - `-sort coord|name` 与 `-index`：支持按坐标/名称排序并在坐标排序时生成 BAI。

## 验证覆盖矩阵

1. **单元与集成测试**：
   - 默认 First-N 采样。
   - RNA-seq 多重比对记录（3~4 条同 QNAME）完整保留。
   - 边界情况：请求数超过总数、N=0、空 BAM。
   - Random-N 随机种子可复现性验证。
   - Ratio 采样比例精确性与种子可复现性。
   - 坐标排序与名称排序校验、`-index` 必须配合 `-sort coord` 规则。
   - 真实 `bamdriver` 数据集 (`testdata/Test_hg19_NRAS.bam`) 端到端验证。
2. **Bash 对照与 Samtools 检查**：
   - `scripts/compare_bash.sh` 对比 `(samtools view -H ; samtools view | head -n N) | samtools view -b`。
   - 保证 record stream 逐行逐字段 100% 一致。
   - 产物通过 `samtools quickcheck -v` 检查。
3. **CI 自动化**：
   - GitHub Actions `ci.yml` 覆盖 Go 1.24、vet、单元测试、Bash 参考对照测试、多场景 samtools quickcheck 与 idxstats 校验。
