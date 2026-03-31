---
name: quant-plan
description: Use when turning a quant platform requirement into an executable plan that must respect remote ECS execution, real-verification constraints, and durable-doc completion
---

# Quant Plan

## Overview

Write the implementation plan for this quant platform only after reading the real repository, the current durable docs, and the current remote runtime constraints.

## Inputs

- requirement
- current run workspace
- current repo state
- current durable docs

## Required Reading

- `workspace/core.md` from controller root if present, else repo-local mirror
- controller `AGENTS.md` if present
- repo `AGENTS.md`
- current relevant repo code and docs

## Plan Requirements

Write `plan.md` inside the current run workspace. The plan must include:

- exact goal
- exact affected files and responsibilities
- proof-test matrix
- real-verification matrix
- doc-impact matrix for the five durable docs
- rollback / safety notes for any global-state change
- ordered tasks granular enough to execute independently

## Hard Rules

- Code changes happen in the remote ECS repo only.
- Plans must use the current project's real paths and commands.
- Any test plan that can disturb live positions or global runtime state must say exactly how it is contained or why it must stop for user approval.
- Do not hand-wave doc updates; name which durable docs must change and why.
