import pandas as pd
import re
from pathlib import Path

RESULTS_DIR = Path(__file__).parent.parent / "kwa" / "results"
OUTPUT_DIR = Path(__file__).parent.parent / "results"
OUTPUT_DIR.mkdir(exist_ok=True)
OUTPUT_FILE = OUTPUT_DIR / "results_linux.csv"

LANG_TO_COL_ID = {"nodejs": "node"}

LANG_COL_PATTERNS = [
    "disk_total_read_cgroup_container",
    "disk_total_write_cgroup_container",
    "network_total_cgroup_container",
]


def get_col_id(language: str) -> str:
    return LANG_TO_COL_ID.get(language, language)


def normalize(df: pd.DataFrame) -> pd.DataFrame:
    lang = df["language"].iloc[0]
    col_id = get_col_id(lang)

    rename_map = {}
    drop_cols = []

    for col in df.columns:
        for prefix in LANG_COL_PATTERNS:
            m = re.match(rf"^({re.escape(prefix)})-([^-]+)_container-(.+)$", col)
            if m:
                if m.group(2) == col_id:
                    rename_map[col] = f"{m.group(1)}-container-{m.group(3)}"
                else:
                    drop_cols.append(col)
                break

    return df.drop(columns=drop_cols).rename(columns=rename_map)


frames = []
for csv_file in sorted(RESULTS_DIR.glob("measurements_*.csv")):
    df = pd.read_csv(csv_file)
    frames.append(normalize(df))

merged = pd.concat(frames, ignore_index=True)
merged.to_csv(OUTPUT_FILE, index=False)
print(f"Written {len(merged)} rows × {len(merged.columns)} columns → {OUTPUT_FILE}")
