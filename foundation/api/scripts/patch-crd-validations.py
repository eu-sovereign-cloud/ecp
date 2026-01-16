#!/usr/bin/env python3
"""Patch generated Kubernetes CRDs with CEL validations.

Why this exists
---------------
CRDs and Go structs in this repo are generated via controller-gen. We keep the
source-of-truth in Go types, but some validations are easier to express as
x-kubernetes-validations (CEL). Since the CRDs are regenerated frequently, we
apply these validations as a post-generation patch step.

This script:
- reads one or more rules from a YAML file
- patches the specified CRD YAML files
- injects x-kubernetes-validations at the requested OpenAPI schema location

Rule use-case example
---------------------
Make BlockStorage.spec.sizeGB non-decreasable (immutability except allowing
increase) when updating an existing object.

CEL:  !oldSelf.hasValue() || self >= oldSelf.value()

Notes
-----
- Uses ruamel.yaml to preserve YAML formatting/comments reasonably well.
- Patches every version entry under spec.versions[*] that has a schema.

"""

from __future__ import annotations

import argparse
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Optional

from ruamel.yaml import YAML

yaml = YAML()
yaml.preserve_quotes = True
yaml.width = 4096


class PatchError(RuntimeError):
    pass


@dataclass(frozen=True)
class ValidationRule:
    name: str
    file: str
    spec_path: str
    validations: List[Dict[str, Any]]


def _as_list(x: Any) -> List[Any]:
    if x is None:
        return []
    if isinstance(x, list):
        return x
    return [x]


def _spec_path_to_schema_path(spec_path: str) -> str:
    """Convert a specPath like 'sizeGB' or 'storage.sizeGB' to full schema path."""
    parts = [p for p in spec_path.split(".") if p]
    return "properties.spec.properties." + ".properties.".join(parts)


def _load_rules(path: Path) -> List[ValidationRule]:
    doc = yaml.load(path.read_text())
    if not isinstance(doc, dict) or "rules" not in doc:
        raise PatchError(f"rules file {path} must be a YAML mapping with a 'rules' key")

    rules: List[ValidationRule] = []
    for raw in _as_list(doc.get("rules")):
        if not isinstance(raw, dict):
            raise PatchError("each rule must be a mapping")

        rules.append(
            ValidationRule(
                name=str(raw.get("name") or "unnamed"),
                file=str(raw.get("file") or ""),
                spec_path=str(raw.get("specPath") or ""),
                validations=list(raw.get("validations") or []),
            )
        )

    # basic validation
    for r in rules:
        if not r.file:
            raise PatchError(f"rule '{r.name}' missing 'file'")
        if not r.spec_path:
            raise PatchError(f"rule '{r.name}' missing 'specPath'")
        if not r.validations:
            raise PatchError(f"rule '{r.name}' missing 'validations'")

    return rules


def _get_versions(crd: Dict[str, Any]) -> List[Dict[str, Any]]:
    spec = crd.get("spec")
    if not isinstance(spec, dict):
        return []
    versions = spec.get("versions")
    if not isinstance(versions, list):
        return []
    return [v for v in versions if isinstance(v, dict)]


def _ensure_mapping(node: Any, key: str) -> Dict[str, Any]:
    if not isinstance(node, dict):
        raise PatchError(f"expected mapping while ensuring key {key!r}")
    if key not in node or node[key] is None:
        node[key] = {}
    if not isinstance(node[key], dict):
        raise PatchError(f"expected mapping at {key!r}")
    return node[key]


def _ensure_list(node: Any, key: str) -> List[Any]:
    if not isinstance(node, dict):
        raise PatchError(f"expected mapping while ensuring key {key!r}")
    if key not in node or node[key] is None:
        node[key] = []
    if not isinstance(node[key], list):
        raise PatchError(f"expected list at {key!r}")
    return node[key]


def _walk_schema_path(root_schema: Dict[str, Any], schema_path: str) -> Dict[str, Any]:
    """Walk a dot-separated schema path, creating intermediate mappings.

    Supported segments:
      - properties.<name>
      - items

    Example:
      properties.spec.properties.sizeGB
    """

    node: Any = root_schema
    for seg in [s for s in schema_path.split(".") if s]:
        if seg == "items":
            node = _ensure_mapping(node, "items")
            continue
        if seg == "properties":
            node = _ensure_mapping(node, "properties")
            continue

        # regular property name inside a properties mapping
        if not isinstance(node, dict):
            raise PatchError(f"cannot descend into segment {seg!r}; current node is not a mapping")
        if seg not in node or node[seg] is None:
            node[seg] = {}
        node = node[seg]

    if not isinstance(node, dict):
        raise PatchError(f"schemaPath '{schema_path}' did not resolve to a mapping")
    return node


def _merge_validations(existing: List[Any], wanted: List[Dict[str, Any]]) -> bool:
    """Append wanted validations if an identical rule isn't already present."""
    changed = False
    for v in wanted:
        if v in existing:
            continue
        existing.append(v)
        changed = True
    return changed


def patch_file(path: Path, rule: ValidationRule) -> bool:
    """Patch a single CRD file with the given rule. Returns True if file was modified."""
    if not path.exists():
        print(f"[WARN] rule '{rule.name}': file not found: {path}")
        return False

    docs = list(yaml.load_all(path.read_text()))
    touched = False
    schema_path = _spec_path_to_schema_path(rule.spec_path)

    for doc in docs:
        if not isinstance(doc, dict):
            continue
        if doc.get("kind") != "CustomResourceDefinition":
            continue

        # patch each version schema
        for ver in _get_versions(doc):
            schema = ((ver.get("schema") or {}).get("openAPIV3Schema"))
            if not isinstance(schema, dict):
                continue

            target = _walk_schema_path(schema, schema_path)
            validations_list = _ensure_list(target, "x-kubernetes-validations")
            if _merge_validations(validations_list, rule.validations):
                touched = True

    if touched:
        with path.open("w") as f:
            yaml.dump_all(docs, f)
        return True
    return False


def main(argv: Optional[List[str]] = None) -> int:
    ap = argparse.ArgumentParser(description="Patch generated CRDs with CEL validations")
    ap.add_argument(
        "--rules",
        required=True,
        type=Path,
        help="Path to YAML file describing validations to apply",
    )
    ap.add_argument(
        "--root",
        required=True,
        type=Path,
        help="Root directory containing generated CRD YAMLs (e.g. foundation/api/generated/crds)",
    )
    ap.add_argument(
        "--dry-run",
        action="store_true",
        help="Don't write files, only report what would change",
    )

    args = ap.parse_args(argv)

    rules = _load_rules(args.rules)

    total_patched = 0

    for rule in rules:
        file_path = args.root / rule.file

        if args.dry_run:
            if not file_path.exists():
                print(f"[DRY-RUN] rule '{rule.name}': file not found: {file_path}")
                continue
            print(f"[DRY-RUN] would patch {file_path}")
            continue

        if patch_file(file_path, rule):
            print(f"[PATCHED] {file_path}")
            total_patched += 1

    print(f"patched_files={total_patched} rules={len(rules)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

