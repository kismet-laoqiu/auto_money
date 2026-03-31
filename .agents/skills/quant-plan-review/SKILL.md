---
name: quant-plan-review
description: Use when reviewing a quant platform plan for root-cause quality, safe real verification, and fit with the current ECS runtime and trust boundaries
---

# Quant Plan Review

## Overview

Review the plan against the real repository and the real runtime constraints. Reject plans that only look complete on paper.

## Review Bar

Verify:

- the plan matches current repo architecture
- proof tests are present
- real-verification paths are present
- impact safety is explicit
- durable-doc updates are explicit
- no step bypasses `execd`, notifier ownership, or current trust boundaries

## Required Findings

Call out:

- missing proof tests
- fake or insufficient real verification
- unsafe tests that can affect live state without rollback
- doc-gate omissions
- over-engineered structure that does not fit this repo

## Output

Write or update `plan-review.md` in the current run workspace with:

- summary
- findings
- revision requests
- final recommendation
