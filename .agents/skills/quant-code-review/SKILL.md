---
name: quant-code-review
description: Use when reviewing quant platform changes against the run workspace base commit and current repository rules before testing or completion
---

# Quant Code Review

## Overview

Review the actual diff, not the intention.

## Baseline

- Prefer `base_commit` from the current run workspace `status.md`.
- If missing, review against the branch tracking point and the current working tree diff.

## Review Priorities

1. correctness and regressions
2. boundary violations
3. testing gaps
4. doc-gate mismatches
5. maintainability

## Mandatory Checks

- Does any code bypass `execd` for exchange writes?
- Does any code blur notifier / OpenClaw Telegram ownership?
- Does any code confuse sqlite runtime truth with warehouse truth?
- Do tests prove the changed behavior?
- Do durable docs need changes the code diff has not yet reflected?

## Output

Write `code-review.md` in the current run workspace. Findings first, then summary.
