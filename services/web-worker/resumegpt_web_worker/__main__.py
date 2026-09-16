from __future__ import annotations

import argparse

from .server import serve


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--serve", action="store_true")
    parser.add_argument("--address", default="0.0.0.0:8091")
    arguments = parser.parse_args()
    if not arguments.serve:
        parser.error("--serve is required")
    serve(arguments.address)


if __name__ == "__main__":
    main()
