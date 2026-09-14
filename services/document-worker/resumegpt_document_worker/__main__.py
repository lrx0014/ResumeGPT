import argparse
import json
import sys
from pathlib import Path

from .extractor import ExtractionError, extract

from .server import serve


def main() -> int:
    parser = argparse.ArgumentParser(description="Extract immutable evidence segments from one quarantined document.")
    parser.add_argument("path", type=Path, nargs="?")
    parser.add_argument("--serve", action="store_true", help="Run the internal extraction HTTP service.")
    parser.add_argument("--address", default="0.0.0.0:8090", help="Internal HTTP listen address.")
    parser.add_argument("--skip-malware-scan", action="store_true", help=argparse.SUPPRESS)
    arguments = parser.parse_args()
    if arguments.serve:
        serve(arguments.address)
        return 0
    if arguments.path is None:
        parser.error("path is required unless --serve is used")
    try:
        print(extract(arguments.path, malware_scan=not arguments.skip_malware_scan).to_json())
        return 0
    except ExtractionError as error:
        print(json.dumps({"error": {"code": error.code, "message": str(error)}}), file=sys.stderr)
        return 1
    except Exception:
        print(json.dumps({"error": {"code": "internal_error", "message": "Document extraction failed unexpectedly."}}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
