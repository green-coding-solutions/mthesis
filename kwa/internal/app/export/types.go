package export

import "mthesis/kwa/internal/constant"

// Request contains the normalized inputs for one export execution.
type Request struct {
	Mode      constant.ExportMode
	BatchSize int
	RunID     string
	OutPath   string
}
