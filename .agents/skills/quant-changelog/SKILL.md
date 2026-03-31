---
name: quant-changelog
description: Use when summarizing quant platform changes from the current run against the recorded base commit and current repository state
---

# Quant Changelog

## Overview

Produce a factual Chinese changelog for the current run.

## Baseline

- Prefer `base_commit` from `status.md`
- Include both committed diff and working-tree diff when relevant

## Coverage

Summarize:

- behavior changes
- verification impact
- durable-doc impact
- repo doc impact

## Output

Write `changelog.md` in the current run workspace.

Do not write from commit messages alone. Read the diff and the touched code.
