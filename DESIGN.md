# Context Design Report

## Purpose
This document records the current design conclusions for model-context injection in `src/main/java`.
It is intended for future comparison and re-analysis.
It only keeps conclusions, key points, and cautions.

## Current Assumptions
- `TaskRag` injects response schema into `System Prompt`, not `History`.
- `RequestContextBuilder` defaults internal injected context to `assistant`.
- `main@main` now loads `rag_goal` and then chains into `goal-guard`.
- `goal-guard` evaluates the current answer out of band through `goal@main`, not by mutating the user query in place.
- On goal failure, the system resets `query` back to `original` and stores exactly one latest goal hint in `rag_goal`.
- `RequestChecker` now enforces that the live request query still contains `original` or `initial`, depending on task type.
- `#router` and `#memory` stay empty or are not loaded for simple tasks.

## Overall Judgment
- The current design is healthier and more internally consistent than earlier versions.
- The main architecture is now reasonable.
- The biggest improvements are:
  - output schema moved out of `History` and into `System Prompt`
  - internal control instructions no longer default to `user`
  - goal correction no longer rewrites the user query with synthetic intent
  - simple tasks avoid most router and memory overhead
- The remaining issues are mostly conditional, not structural.

## Positive Designs

### Clearly Positive
- `TaskRag`
  - Response schema is now injected into `System Prompt`.
  - This is the best compromise when native `response_schema` cannot be used because of tool-calling constraints.
  - It strengthens output control without pretending to be user intent.
- `SafetyRag`
  - Short, stable, high-value constraints.
  - Good candidate for always-on prompt content.
- current `CliRag`
  - Effective environment injection is much more controlled than before.
  - `user` and `soul` are truncated.
  - `workspace` and `sys` are high-value environment facts.
- `RequestCapacity`
  - Only records prompt size and upgrades complexity.
  - Does not inject extra context text.
  - Good defensive design for long-context situations.
- `RequestChecker`
  - Pure validation, no extra context injection.
  - It now protects a key invariant: the live request query must still contain the original user intent.
- `RequestRewriter` history compression
  - Net positive.
  - Reduces context bloat and helps long conversations remain usable.
- `GoalGuardAssistant` + `GoalInsertRag`
  - This is the main new improvement replacing the old due-time style guard.
  - Goal evaluation is separated into a dedicated `goal@main` pass with structured output.
  - On failure, the system resets the task query back to `original` instead of overwriting it with synthetic guidance.
  - Only one latest goal hint is stored and replayed.
  - This preserves user intent while still giving the next round a bounded corrective nudge.

### Positive Under Current Runtime Assumptions
- `RouterRag`
  - Positive for multi-agent or coordination-heavy tasks.
  - Acceptable if empty or skipped on simple tasks.
- `MemoryRag`
  - Positive for long-running user continuity and preference recall.
  - Acceptable if empty or skipped on simple tasks.
- `PlanRag`
  - Positive for tasks that genuinely benefit from planning.
  - Planning instructions are injected as `assistant` context, not `user`.
  - It is healthier than directly rewriting the query with planning text.

## Conditional or Mixed Designs

### `SkillsSchemaRag`
- Main issue is no longer cache.
- Main issue is attention cost.
- Even with stable content, it keeps the model thinking in a "skill-first" mode.
- Acceptable when skills are central to the product.
- Risky if the skill list grows too large or becomes too detailed.

### `CliInsertRag`
- Real user-added content is correctly kept as `user`.
- The wrapper instruction is now effectively `assistant`, which is the right direction.
- Still risky because recalled insert content is pushed back into current query history.
- Strongly affects KV cache on the current round.
- Good for explicit user corrections and supplements.
- Risky if stale inserts keep being replayed.

### `GoalGuardAssistant`
- Defensive, not productivity-enhancing.
- Useful for catching answers that drift away from the user's actual goal.
- Healthier than rewriting the active query with `why_do_this`.
- Still adds an extra model round and retry path.
- Quality now depends mainly on whether the goal prompt remains conservative and specific.

### `GoalInsertRag`
- Better than replaying multiple historical goal hints.
- Current bounded design is acceptable because it stores only the latest hint.
- Still injects extra history into the next round, so wording quality matters.
- Risky only if the hint becomes too long, too abstract, or starts substituting for user intent.

### `RequestFunCallStore` and reason replay via `RequestRewriter`
- Helpful for traceability and workflow continuity.
- Risk: old internal rationale may leak back into the current task and slightly bias behavior.
- Acceptable if replay volume stays small and filtered.

## Negative or Watchlist Designs

### Main Watchlist
- `SkillsSchemaRag`
  - Not structurally wrong.
  - Still the biggest long-term attention-tax source if the skills payload grows.
- `CliInsertRag`
  - Still the clearest active cache-breaker.
  - Also the easiest source of stale guidance replay.
- `RouterRag`
  - If router content is ever re-enabled for simple tasks, it can quickly over-process the task.
  - Offline hint is still a real injection path and should be watched.
- `MemoryRag`
  - If memory starts loading on simple tasks, it can reintroduce goal drift quickly.
- `GoalGuardAssistant`
  - If the guard becomes too eager, it can create unnecessary retry loops.
  - If `why_do_this` becomes generic, it becomes attention tax instead of useful guidance.

### No Longer Major Problems
- Default internal context pretending to be `user`
  - Previously a major issue.
  - Now largely fixed by defaulting `RequestContextBuilder` to `assistant`.
- `TaskRag` in `History`
  - Previously a major issue.
  - Now fixed by moving schema to `System Prompt`.
- Goal correction by directly overwriting the active query
  - Previously risky because synthetic intent could replace user intent.
  - Now fixed by resetting to `original` and carrying correction through bounded assistant history.
- `RequestDueTime`
  - Removed from the request rewrite path.
- `RequestRemind`
  - Removed.
- `HistoryBlockAssistant`
  - Removed.

## Cache Conclusions

### Low Cache Risk
- `SafetyRag`
- `TaskRag`
- current `CliRag`
- `RequestCapacity`
- `RequestChecker`

### Acceptable Cache Risk Under Stable Prefix Assumption
- `SkillsSchemaRag`
- `RouterRag`
- `MemoryRag`
- `GoalInsertRag`
  - Acceptable because it injects at most one bounded assistant history on retry.

### Clear Cache Risk
- `CliInsertRag`
  - Recalled content is re-appended into current query histories.
  - This is the clearest place where KV cache is intentionally disrupted.

## Goal-Focus Conclusions

### Good for Focus
- `TaskRag`
- `SafetyRag`
- current `CliRag`
- `RequestChecker`
- `GoalGuardAssistant` + `GoalInsertRag`
  - Positive under the current design because they preserve original query ownership and only add one bounded corrective hint.

### Focus Depends on Task Type
- `PlanRag`
  - positive for complex tasks
  - unnecessary for trivial tasks
- `RouterRag`
  - positive for coordination-heavy tasks
  - distracting for simple direct tasks
- `MemoryRag`
  - positive for continuity-heavy tasks
  - distracting for simple or one-shot tasks

### Main Focus Risks
- `SkillsSchemaRag`
  - can widen the action space too early
- `CliInsertRag`
  - can replay stale supplements
- `GoalGuardAssistant`
  - can over-trigger retries if the pass/fail standard drifts too high
- `GoalInsertRag`
  - can become synthetic steering if the hint stops being short and concrete

## Design Principles to Preserve
- Output contract belongs in `System Prompt`, not `user` history.
- Real user supplements should stay `user`.
- Internal wrappers, plan instructions, and process contracts should be `assistant`.
- Goal correction should not replace the original user query.
- Goal hints should stay bounded, latest-only, and assistant-scoped.
- Simple tasks should avoid router and memory whenever possible.
- Stable prompt prefixes are acceptable if they are high-value and byte-identical across requests.
- Attention cost matters even when cache cost is low.

## Future Review Checklist
- Check whether `#skills` has grown too long.
- Confirm `#router` is still empty or skipped for simple tasks.
- Confirm `#memory` is still empty or skipped for simple tasks.
- Check whether `CliInsertRag` recall volume is still bounded and recent.
- Confirm `GoalGuardAssistant` still resets `query` back to `original` before retry.
- Confirm `GoalInsertRag` still stores only the latest goal hint.
- Check whether goal hints remain short, concrete, and action-guiding.
- Check whether `RequestChecker` still enforces the original or initial query containment invariant.
- Check whether fun-call reason replay is still small and filtered.
- Check whether any new internal context path defaults back to `user`.

## Final Snapshot
- Current design state: reasonable and usable.
- Main architecture: positive.
- Main risks: attention tax from skills, replay tax from inserts, conditional drift from router or memory if enabled too broadly, and over-eager goal retries if guard quality regresses.
- If future regressions happen, first inspect:
  - `SkillsSchemaRag`
  - `CliInsertRag`
  - `GoalGuardAssistant`
  - `GoalInsertRag`
  - `RouterRag`
  - `MemoryRag`
