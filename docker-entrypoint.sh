#!/bin/sh
set -e

if [ -n "$DEEZER_ARL" ]; then
    python3 - <<'PYEOF'
import os, sys
from pathlib import Path

import click
from tomlkit.api import dumps, parse

config_dir = Path(click.get_app_dir("streamrip"))
config_dir.mkdir(parents=True, exist_ok=True)
config_path = config_dir / "config.toml"

# If no config exists yet, generate the default one
if not config_path.exists():
    import subprocess
    subprocess.run(["rip", "config", "reset", "-y"], check=True)

with open(config_path) as f:
    cfg = parse(f.read())

cfg["deezer"]["arl"] = os.environ["DEEZER_ARL"]

with open(config_path, "w") as f:
    f.write(dumps(cfg))

print(f"[entrypoint] Deezer ARL written to {config_path}.", file=sys.stderr)
PYEOF
else
    echo "[entrypoint] DEEZER_ARL not set – streamrip config unchanged." >&2
fi

exec "$@"