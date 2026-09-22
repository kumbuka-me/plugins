#!/usr/bin/env python3
"""Restrict temporary preview packages to permissions supported by the static CLI."""

import argparse
import os
import stat
import tempfile
from pathlib import Path
from zipfile import BadZipFile, ZipFile

import yaml

PERMISSIONS = {"browser:render", "pages:content", "pages:read"}


def rewrite_package(path):
    # Never extract archive paths. Replace the archive only after a successful rewrite.
    with ZipFile(path) as source:
        names = source.namelist()
        if len(names) != len(set(names)):
            raise ValueError(f"{path}: duplicate archive entries")
        if "plugin.yaml" not in names:
            raise ValueError(f"{path}: missing plugin.yaml")
        manifest = yaml.safe_load(source.read("plugin.yaml"))
        permissions = manifest.get("permissions") if isinstance(
            manifest, dict) else None
        if not isinstance(permissions, list) or any(not isinstance(p, str) for p in permissions):
            raise ValueError(f"{path}: permissions must be a list of strings")
        manifest["permissions"] = sorted(set(permissions) & PERMISSIONS)
        content = yaml.safe_dump(
            manifest, sort_keys=False, allow_unicode=True).encode()
        with tempfile.NamedTemporaryFile(dir=path.parent, prefix=".preview-package-", delete=False) as temporary:
            temporary_path = Path(temporary.name)
        try:
            with ZipFile(temporary_path, "w") as target:
                target.comment = source.comment
                for entry in source.infolist():
                    target.writestr(entry, content if entry.filename ==
                                    "plugin.yaml" else source.read(entry))
            temporary_path.chmod(stat.S_IMODE(path.stat().st_mode))
            os.replace(temporary_path, path)
        finally:
            temporary_path.unlink(missing_ok=True)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("packages", type=Path, nargs="+")
    args = parser.parse_args()
    try:
        for package in args.packages:
            rewrite_package(package.resolve())
    except (OSError, ValueError, BadZipFile, yaml.YAMLError) as error:
        parser.exit(1, f"preview package: {error}\n")
