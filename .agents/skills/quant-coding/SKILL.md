---
name: quant-coding
description: Use when implementing an approved quant platform plan on the remote ECS repository and the work must stay inside the repo's real architecture and safety boundaries
---

# Quant Coding

## Overview

Implement the approved plan in the remote ECS repository only. Keep the run workspace as the execution log and evidence index; it is not the code workspace.

## Required Companion Skills

- `test-driven-development` before any behavior change
- `systematic-debugging` on any unexpected failure
- `verification-before-completion` before any completion claim

## Rules

- Read the plan first.
- Read at least three similar implementations before editing.
- Code edits happen in `/root/.config/superpowers/worktrees/quant-lab/autoresearch-20260328-all-plan`.
- If operating from a controller workspace, use remote-safe transport for edits; do not treat a local copy as authoritative.
- Keep changes minimal and directly traceable to the plan.
- Keep `execution-log.md` updated as the run proceeds.

## Commit Shape

Prefer milestone commits, not noise commits. Each commit must represent a stable, reviewable checkpoint.

## Stop Conditions

Stop and escalate if:

- the plan requires unsafe live-state mutation without approval
- a fix would cross execution trust boundaries
- the change cannot be verified safely in the shared environment
