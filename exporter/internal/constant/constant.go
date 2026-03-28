package constant

type Benchmark string

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

var languageByNormalized = map[string]ProgrammingLanguage{
	"c":         "c",
	"cpp":       "cpp",
	"cplusplus": "cpp",
	"csharp":    "csharp",
	"cs":        "csharp",
	"dart":      "dart",
	"erlang":    "erlang",
	"fortran":   "fortran",
	"fsharp":    "fsharp",
	"fs":        "fsharp",
	"go":        "go",
	"golang":    "go",
	"haskell":   "haskell",
	"java":      "java",
	"lua":       "lua",
	"nodejs":    "nodejs",
	"node":      "nodejs",
	"ocaml":     "ocaml",
	"perl":      "perl",
	"php":       "php",
	"python":    "python",
	"ruby":      "ruby",
	"rust":      "rust",
	"swift":     "swift",
}
