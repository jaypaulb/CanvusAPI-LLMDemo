# Performance Benchmarking Guide

This guide explains how to run and interpret performance benchmarks for CanvusLocalLLM image generation.

## Prerequisites

1. **CUDA-enabled GPU**: Benchmarks require a CUDA-capable GPU (RTX 3060 recommended for baseline)
2. **SD Model**: Set `SD_MODEL_PATH` environment variable to your Stable Diffusion model file
3. **Go 1.21+**: Required for running benchmarks

## Running Benchmarks

### Basic Usage

Run all benchmarks:
```bash
go test -bench=. ./tests/ -timeout=600s
```

### Individual Benchmarks

**512x512 Generation (Target: <30s on RTX 3060)**
```bash
go test -bench=BenchmarkImageGeneration512 -benchtime=1x -timeout=60s ./tests/
```

**768x768 Generation (Target: <60s on RTX 3060)**
```bash
go test -bench=BenchmarkImageGeneration768 -benchtime=1x -timeout=120s ./tests/
```

**CUDA Utilization Analysis (3 iterations)**
```bash
go test -bench=BenchmarkCUDAUtilization -benchtime=3x -timeout=180s ./tests/
```

### Running Multiple Iterations

For more reliable timing measurements, run multiple iterations:
```bash
# Run 5 iterations of each benchmark
go test -bench=BenchmarkImageGeneration512 -benchtime=5x -timeout=300s ./tests/
```

## Understanding Results

### Benchmark Output Format

```
BenchmarkImageGeneration512-8    1    28.5s    102.4 KB/image
```

- `BenchmarkImageGeneration512-8`: Benchmark name and GOMAXPROCS value
- `1`: Number of iterations run
- `28.5s`: Average time per operation
- `102.4 KB/image`: Custom metric (image size in KB)

### Performance Targets

Benchmarks are calibrated for **RTX 3060 with 25 steps**:

| Resolution | Target Time | Typical Image Size |
|------------|-------------|-------------------|
| 512x512    | <30s        | ~100-150 KB       |
| 768x768    | <60s        | ~200-300 KB       |

**Note**: Times will vary based on:
- GPU model (RTX 3060, 3070, 3080, 3090, etc.)
- Number of steps (benchmark uses 25)
- Thermal state (initial run vs. warmed up)
- System load

### Interpreting Warnings

**"WARNING: Average time exceeds target"**
- Not necessarily a failure - target is RTX 3060 baseline
- Compare to your GPU's expected performance tier
- Check for thermal throttling (see CUDA Utilization below)

**"WARNING: High timing variance"**
- Indicates >20% variance between runs
- Possible causes:
  - Thermal throttling (GPU overheating)
  - CUDA contention (other processes using GPU)
  - Background system activity
- Solution: Let GPU cool down, close other GPU applications

### CUDA Utilization Benchmark

The `BenchmarkCUDAUtilization` runs multiple iterations to detect performance consistency issues:

```
CUDA Utilization Stats:
  Average: 25.3s
  Min: 24.8s
  Max: 26.1s
  Variance: 1.3s
  Iterations: 3
PASS: Consistent CUDA utilization (5.1% variance)
```

**Good Performance**: Variance <10%
- GPU is consistently utilized
- No thermal throttling
- Stable performance

**Poor Performance**: Variance >20%
- Check GPU temperature
- Close other GPU applications
- Verify CUDA drivers are up to date
- Check system cooling

## Advanced Options

### Memory Profiling

```bash
go test -bench=BenchmarkImageGeneration512 -benchmem ./tests/
```

### CPU Profiling

```bash
go test -bench=BenchmarkImageGeneration512 -cpuprofile=cpu.prof ./tests/
go tool pprof cpu.prof
```

### Comparing Results

Use `benchstat` to compare benchmark runs:

```bash
# Install benchstat
go install golang.org/x/perf/cmd/benchstat@latest

# Run benchmarks and save results
go test -bench=. ./tests/ > old.txt
# ... make changes ...
go test -bench=. ./tests/ > new.txt

# Compare
benchstat old.txt new.txt
```

## Troubleshooting

### "SD_MODEL_PATH not set"

Set the environment variable:
```bash
export SD_MODEL_PATH=/path/to/your/model.safetensors
go test -bench=. ./tests/
```

### "CUDA may be unavailable"

1. Verify CUDA installation: `nvidia-smi`
2. Check CUDA version matches sd.cpp requirements
3. Ensure GPU is not in use by other processes
4. Try running with `CUDA_VISIBLE_DEVICES=0`

### Benchmarks Skip on Non-GPU Systems

This is expected behavior. Benchmarks automatically skip if:
- No CUDA GPU available
- SD model not found
- CUDA initialization fails

To run on CPU (very slow, not recommended):
```bash
# Note: sd.cpp may not support CPU-only mode
# Benchmarks will likely skip or fail
```

## Best Practices

1. **Warm-up Run**: First generation is often slower due to model loading
   ```bash
   go test -bench=BenchmarkImageGeneration512 -benchtime=2x ./tests/
   ```

2. **Isolated Environment**: Close GPU-intensive applications before benchmarking

3. **Consistent Conditions**: Run benchmarks with consistent GPU temperature

4. **Multiple Iterations**: Use `-benchtime=5x` or higher for reliable results

5. **Document Your Hardware**: Record GPU model, VRAM, driver version when sharing results

## Expected Performance by GPU

Approximate 512x512 generation times (25 steps):

| GPU Model    | Expected Time | Notes |
|--------------|---------------|-------|
| RTX 3060     | 25-30s        | Baseline |
| RTX 3070     | 20-25s        | ~20% faster |
| RTX 3080     | 15-20s        | ~40% faster |
| RTX 3090     | 12-18s        | ~50% faster |
| RTX 4060     | 18-23s        | Better efficiency |
| RTX 4070     | 12-17s        | Significant improvement |
| RTX 4080     | 10-14s        | High performance |
| RTX 4090     | 8-12s         | Fastest consumer GPU |

*Times are approximate and may vary based on model, step count, and system configuration.*

## Contributing Benchmark Results

When sharing benchmark results, include:
- GPU model and VRAM
- CUDA version
- Driver version
- Operating system
- Model file used (size, variant)
- Any relevant system specifications

Example:
```
GPU: RTX 3060 12GB
CUDA: 12.2
Driver: 535.104.05
OS: Ubuntu 22.04
Model: v1-5-pruned-emaonly.safetensors (4GB)
Results:
  512x512: 27.3s
  768x768: 58.1s
  CUDA Utilization: 6.2% variance
```
