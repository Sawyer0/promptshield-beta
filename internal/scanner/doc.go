/*
Package scanner implements a multi-tier content scanning engine for threat detection.

# Architecture

The scanner uses a progressive three-tier approach to balance performance and accuracy:

  - L1: Aho-Corasick multi-pattern matching (< 1ms)
  - L2: Optimized regex with Bloom filter pre-screening (< 10ms)
  - L3: Semantic analysis via LLM providers (< 100ms, opt-in)

# Streaming Design

The scanner uses a streaming architecture with bounded memory to handle arbitrarily
large inputs without loading them entirely into memory. This is achieved through:

  - Sliding window processing with configurable overlap
  - Line-by-line scanning with bufio.Scanner
  - Chunked evaluation for lines exceeding buffer size
  - Constant memory usage regardless of input size

# Performance

Typical performance characteristics:

  - Throughput: 10,000+ requests/second per instance
  - Latency P95: < 50ms for full pipeline
  - Memory: Constant (streaming architecture)
  - Scalability: Horizontal scaling with stateless design

# Example Usage

Basic scanning:

	sc := scanner.ScanEngineCstor(0)  // 0 = use default 16MB buffer
	sc.LoadRulePacks(packs)
	result, err := sc.ScanFile(ctx, "input.txt")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d violations\n", len(result.Violations))

Streaming from reader:

	sc := scanner.ScanEngineCstor(0)
	sc.LoadRulePacks(packs)
	result, err := sc.ScanReader(ctx, reader, "stream")

With semantic analysis:

	sc := scanner.ScanEngineCstor(0)
	sc.SetSemanticAnalyzer(openai.New(openai.Options{
		APIKey: os.Getenv("OPENAI_API_KEY"),
	}))
	sc.LoadRulePacks(packs)
	result, err := sc.ScanFile(ctx, "input.txt")

# Configuration

The scanner supports extensive configuration:

  - Buffer sizes and overlap for streaming
  - Timeouts and resource limits
  - Quarantine behavior on errors/timeouts
  - Composition strategies (first_match, priority_order)
  - Runtime context for conditional rules

# Thread Safety

Scanner instances are NOT thread-safe. Create separate instances for concurrent use,
or use sync.Pool for efficient reuse:

	pool := &sync.Pool{
		New: func() any {
			sc := scanner.ScanEngineCstor(0)
			sc.LoadRulePacks(packs)
			return sc
		},
	}

	// In handler:
	sc := pool.Get().(*scanner.Scanner)
	defer pool.Put(sc)
	result, err := sc.ScanReader(ctx, req.Body, "request")
*/
package scanner
