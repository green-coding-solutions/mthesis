package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"mthesis/kwa/internal/entity"
)

// metricKeyExpression normalizes metric/detail/unit into a deterministic CSV-safe key.
const metricKeyExpression = `
concat_ws(
  '-',
  nullif(
    regexp_replace(
      regexp_replace(lower(metric), '[^a-z0-9]+', '_', 'g'),
      '(^_+|_+$)', '', 'g'
    ),
    ''
  ),
  nullif(
    regexp_replace(
      regexp_replace(lower(detail_name), '[^a-z0-9]+', '_', 'g'),
      '(^_+|_+$)', '', 'g'
    ),
    ''
  ),
  nullif(
    regexp_replace(
      regexp_replace(lower(unit), '[^a-z0-9]+', '_', 'g'),
      '(^_+|_+$)', '', 'g'
    ),
    ''
  )
)
`

// phaseFilterClause excludes setup/teardown phases so exports focus on benchmark execution phases.
const phaseFilterClause = `
type = 'TOTAL'
AND phase !~* '\[(baseline|installation|boot|idle|runtime|remove)\]'
`

// getPhaseMetricsQueryByID returns one aggregated row per phase for a specific run.
var getPhaseMetricsQueryByID = fmt.Sprintf(`
WITH filtered AS (
  SELECT
    run_id,
    phase,
    %s AS k,
    value
  FROM phase_stats
  WHERE run_id = $1
    AND %s
),
dedup AS (
  SELECT run_id, phase, k, MAX(value) AS value
  FROM filtered
  GROUP BY run_id, phase, k
)
SELECT
  run_id,
  phase,
  COALESCE(jsonb_object_agg(k, value ORDER BY k), '{}'::jsonb) AS metrics
FROM dedup
GROUP BY run_id, phase
ORDER BY run_id, phase;
`, metricKeyExpression, phaseFilterClause)

// getPhaseMetricsBatchQuery returns paginated aggregated phase rows across all runs.
var getPhaseMetricsBatchQuery = fmt.Sprintf(`
WITH filtered AS (
  SELECT
    run_id,
    phase,
    %s AS k,
    value
  FROM phase_stats
  WHERE %s
),
dedup AS (
  SELECT run_id, phase, k, MAX(value) AS value
  FROM filtered
  GROUP BY run_id, phase, k
)
SELECT
  run_id,
  phase,
  COALESCE(jsonb_object_agg(k, value ORDER BY k), '{}'::jsonb) AS metrics
FROM dedup
GROUP BY run_id, phase
ORDER BY run_id, phase
LIMIT $1 OFFSET $2;
`, metricKeyExpression, phaseFilterClause)

// getMetricKeysQuery returns the global normalized key universe used for stable CSV headers.
var getMetricKeysQuery = fmt.Sprintf(`
SELECT DISTINCT
  %s AS k
FROM phase_stats
WHERE %s
ORDER BY k;
`, metricKeyExpression, phaseFilterClause)

// GetPhaseMetricsByID fetches all aggregated phase rows for a single run.
func (s *service) GetPhaseMetricsByID(ctx context.Context, runID string) ([]entity.PhaseMetrics, error) {
	if strings.TrimSpace(runID) == "" {
		return nil, fmt.Errorf("runID must not be empty")
	}

	rows, err := s.db.QueryContext(ctx, getPhaseMetricsQueryByID, runID)
	if err != nil {
		return nil, fmt.Errorf("query phase metrics for run_id %q: %w", runID, err)
	}
	defer rows.Close()

	phaseMetrics, err := scanPhaseMetricsRows(rows)
	if err != nil {
		return nil, err
	}

	return phaseMetrics, nil
}

// GetPhaseMetricsBatch fetches aggregated phase rows across runs using LIMIT/OFFSET pagination.
func (s *service) GetPhaseMetricsBatch(ctx context.Context, limit, offset int) ([]entity.PhaseMetrics, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}
	if offset < 0 {
		return nil, fmt.Errorf("offset must be greater than or equal to zero")
	}

	rows, err := s.db.QueryContext(ctx, getPhaseMetricsBatchQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query phase metrics batch with limit %d offset %d: %w", limit, offset, err)
	}
	defer rows.Close()

	phaseMetrics, err := scanPhaseMetricsRows(rows)
	if err != nil {
		return nil, err
	}

	return phaseMetrics, nil
}

// GetMetricKeys fetches the full, ordered set of metric keys for CSV header generation.
func (s *service) GetMetricKeys(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, getMetricKeysQuery)
	if err != nil {
		return nil, fmt.Errorf("query metric keys: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan metric key row: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric key rows: %w", err)
	}

	return keys, nil
}

// scanPhaseMetricsRows decodes query rows shaped as (run_id, phase, metrics_jsonb).
func scanPhaseMetricsRows(rows *sql.Rows) ([]entity.PhaseMetrics, error) {
	phaseMetrics := make([]entity.PhaseMetrics, 0)
	for rows.Next() {
		var (
			row        entity.PhaseMetrics
			metricsRaw []byte
		)

		if err := rows.Scan(&row.RunID, &row.Phase, &metricsRaw); err != nil {
			return nil, fmt.Errorf("scan phase metrics row: %w", err)
		}

		if len(metricsRaw) == 0 {
			row.Metrics = make(map[string]int64)
		} else if err := json.Unmarshal(metricsRaw, &row.Metrics); err != nil {
			return nil, fmt.Errorf("decode metrics json for phase %q: %w", row.Phase, err)
		}

		if row.Metrics == nil {
			row.Metrics = make(map[string]int64)
		}

		phaseMetrics = append(phaseMetrics, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate phase metrics rows: %w", err)
	}

	return phaseMetrics, nil
}
