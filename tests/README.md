# Tests Directory

This directory contains all test files for CanvusLocalLLM.

## Test Files

### Integration Tests

- **`canvas_check_test.go`**: Canvus API connectivity tests
- **`llm_test.go`**: OpenAI/LLM integration tests
- **`testAPI_test.go`**: Comprehensive API endpoint tests
- **`config_integration_test.go`**: Configuration loading and validation tests
- **`image_generation_test.go`**: End-to-end image generation integration tests
- **`cicd_pipeline_integration_test.go`**: CI/CD pipeline integration tests
- **`webui_server_integration_test.go`**: Web UI server integration tests
- **`phase6_integration_test.go`**: Phase 6 production hardening tests
  - Logging pipeline: JSON formatting and database recording
  - Authentication flow: Complete login/logout cycle, rate limiting, concurrent sessions
  - Shutdown sequence: Ordered cleanup, request draining, error recovery

### Performance Benchmarks

- **`performance_test.go`**: Image generation performance benchmarks
  - `BenchmarkImageGeneration512`: 512x512 generation timing
  - `BenchmarkImageGeneration768`: 768x768 generation timing
  - `BenchmarkCUDAUtilization`: CUDA utilization and consistency

### Test Data

- **`test_data.go`**: Shared test fixtures and helper functions

## Running Tests

### All Tests

```bash
go test ./tests/
```

### Specific Test File

```bash
go test ./tests/canvas_check_test.go
```

### Specific Test Function

```bash
go test -run TestCanvasConnection ./tests/
```

### With Verbose Output

```bash
go test -v ./tests/
```

### With Coverage

```bash
go test -cover ./tests/
go test -coverprofile=coverage.out ./tests/
go tool cover -html=coverage.out
```

### With Race Detection

```bash
go test -race ./tests/
```

## Running Benchmarks

### All Benchmarks

```bash
go test -bench=. -run=^$ ./tests/
```

### Specific Benchmark

```bash
go test -bench=BenchmarkImageGeneration512 -run=^$ ./tests/
```

### With Memory Profiling

```bash
go test -bench=. -benchmem -run=^$ ./tests/
```

### Multiple Iterations

```bash
go test -bench=BenchmarkImageGeneration512 -benchtime=5x -run=^$ ./tests/
```

For detailed benchmarking instructions, see [docs/performance-benchmarking.md](/docs/performance-benchmarking.md).

## Environment Requirements

### Image Generation Tests

Image generation tests (`image_generation_test.go`, `performance_test.go`) require:

1. **CUDA-enabled GPU**: RTX 3060 or better recommended
2. **SD Model**: Set `SD_MODEL_PATH` environment variable
   ```bash
   export SD_MODEL_PATH=/path/to/model.safetensors
   ```
3. **CUDA Installation**: Verified with `nvidia-smi`

These tests will **automatically skip** if:
- `SD_MODEL_PATH` is not set
- Model file doesn't exist
- CUDA is not available
- Context pool creation fails

### API Tests

API tests require:
- `.env` file with Canvus API credentials
- Valid `CANVUS_SERVER`, `CANVUS_API_KEY`, `CANVAS_ID`

Tests will skip gracefully if credentials are not available.

## Test Organization

Tests follow Go testing conventions:
- Test functions: `func TestXxx(t *testing.T)`
- Benchmark functions: `func BenchmarkXxx(b *testing.B)`
- Helper functions: Use `t.Helper()` to mark helper functions

### Table-Driven Tests

Many tests use table-driven test patterns:

```go
tests := []struct {
    name     string
    input    Type
    expected Type
    wantErr  bool
}{
    {"valid case", input1, output1, false},
    {"error case", input2, nil, true},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test implementation
    })
}
```

## Debugging Tests

### Run Single Test with Logging

```bash
go test -v -run TestSpecificFunction ./tests/
```

### Check Test List

```bash
go test -list . ./tests/
```

### Skip Integration Tests

Integration tests check for required environment variables and skip if not present. To force skip:

```bash
unset SD_MODEL_PATH
unset CANVUS_API_KEY
go test ./tests/
```

## CI/CD Considerations

In CI/CD environments without GPU or API credentials:
- Most tests will skip automatically
- Use `-short` flag to skip long-running tests: `go test -short ./tests/`
- Benchmarks should not run in CI unless specifically configured

## Contributing Tests

When adding new tests:

1. Place in appropriate file or create new `*_test.go` file
2. Use descriptive test names: `TestFeature_Scenario`
3. Add table-driven tests for multiple scenarios
4. Use `t.Helper()` for helper functions
5. Handle missing dependencies gracefully with `t.Skip()`
6. Add documentation to this README if introducing new test categories

## Performance Expectations

See [docs/performance-benchmarking.md](/docs/performance-benchmarking.md) for:
- Benchmark targets by GPU model
- Performance troubleshooting
- CUDA utilization analysis
- Comparing benchmark results
