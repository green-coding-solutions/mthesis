package constant

type Benchmark string

// benchmarkByNormalized maps parser-normalized tokens to canonical benchmark names.
var benchmarkByNormalized = map[string]Benchmark{
	"binarytrees":   "binary-trees",
	"fannkuchredux": "fannkuch-redux",
	"fasta":         "fasta",
	"fibonacci":     "fibonacci",
	"knucleotide":   "k-nucleotide",
	"mandelbrot":    "mandelbrot",
	"nbody":         "n-body",
	"regexredux":    "regex-redux",
	"spectralnorm":  "spectral-norm",
}

type ProgrammingLanguage string

// languageByNormalized maps parser-normalized tokens to canonical language names.
var languageByNormalized = map[string]ProgrammingLanguage{
	"c":       "c",
	"cpp":     "cpp",
	"csharp":  "csharp",
	"dart":    "dart",
	"erlang":  "erlang",
	"fortran": "fortran",
	"fsharp":  "fsharp",
	"fs":      "fsharp",
	"go":      "go",
	"haskell": "haskell",
	"java":    "java",
	"lua":     "lua",
	"nodejs":  "nodejs",
	"node":    "nodejs",
	"ocaml":   "ocaml",
	"perl":    "perl",
	"php":     "php",
	"python":  "python",
	"ruby":    "ruby",
	"rust":    "rust",
	"swift":   "swift",
}
