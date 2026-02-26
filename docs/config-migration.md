# Config Migration Guide

This document captures migration rules discovered during the 2026-02-25 config/profile refactor.

Current status: compatibility aliases used during this migration were removed in [ADR 0005](./adr/0005-remove-config-compatibility-aliases.md).

## Rules

- Keep unknown-key validation strict during staged migrations.
- Remove legacy YAML tags from public config structs as soon as the new schema is introduced.
- If cross-package callers still depend on old in-memory fields, expose temporary aliases with `json:"-" yaml:"-"`.
- Treat aliases as an internal bridge only; do not re-expose legacy YAML keys.
- Remove aliases immediately after downstream migration completes; do not keep them as permanent fallback.
- Add loader validation that rejects legacy keys and invalid strategy/profile combinations.
- Regenerate `schemas/config.schema.json` and update config tests in the same change.

## Migration Checklist

1. Introduce new config keys and defaults.
2. Remove old YAML tags from external surface.
3. Add temporary alias fields only when needed for internal compatibility.
4. Add/adjust validation for bounds, required keys, and legacy-key rejection.
5. Update schema and tests.
6. Remove temporary aliases after downstream migration completes and confirm no runtime fallback path remains.
