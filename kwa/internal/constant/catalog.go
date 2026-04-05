package constant

// Benchmark is the canonical benchmark identifier used in exported rows.
type Benchmark string

var allBenchmarks = []Benchmark{
	"binary-trees",
	"fannkuch-redux",
	"k-nucleotide",
	"n-body",
	"regex-redux",
	"spectral-norm",
	"fasta",
	"mandelbrot",
	"fibonacci",
}

// allMeasureBenchmarks is the benchmark subset supported by scripts/measure.sh.
var allMeasureBenchmarks = allBenchmarks[:len(allBenchmarks)-1]

// benchmarkByNormalized maps parser-normalized tokens to canonical benchmark names.
var benchmarkByNormalized = buildNormalizedBenchmarkLookup(allBenchmarks)

// ProgrammingLanguage is the canonical language identifier used in exported rows.
type ProgrammingLanguage string

var allProgrammingLanguages = []ProgrammingLanguage{
	"c",
	"cpp",
	"csharp",
	"dart",
	"erlang",
	"fsharp",
	"go",
	"haskell",
	"java",
	"lua",
	"nodejs",
	"ocaml",
	"perl",
	"php",
	"python",
	"ruby",
	"rust",
	"swift",
}

// languageByNormalized maps parser-normalized tokens to canonical language names.
var languageByNormalized = buildNormalizedProgrammingLanguageLookup(allProgrammingLanguages)

// buildNormalizedBenchmarkLookup creates a normalized parser lookup map for canonical benchmarks.
func buildNormalizedBenchmarkLookup(benchmarks []Benchmark) map[string]Benchmark {
	lookup := make(map[string]Benchmark, len(benchmarks))
	for _, benchmark := range benchmarks {
		lookup[normalizeEnumToken(string(benchmark))] = benchmark
	}

	return lookup
}

// buildNormalizedProgrammingLanguageLookup creates a normalized parser lookup map for canonical languages.
func buildNormalizedProgrammingLanguageLookup(languages []ProgrammingLanguage) map[string]ProgrammingLanguage {
	lookup := make(map[string]ProgrammingLanguage, len(languages))
	for _, language := range languages {
		lookup[normalizeEnumToken(string(language))] = language
	}

	return lookup
}
