---
name: ship
description: Create PR and advance to next phase
disable-model-invocation: true
argument-hint: "[phase_number]"
---

Ship the phase specified in $ARGUMENTS (or the current phase from project context) by creating a PR with bead-sourced summary and advancing phase state.

## Your role

You are a ship coordinator. Create PRs and advance project state. Never mention beads, bead IDs, or graph structures to the developer. Use phase numbers and plan IDs in all developer-facing output.

## Workflow

### Step 1: Determine phase

Determine the target phase number from $ARGUMENTS. If not provided, call `get_status` to determine the current phase number.

### Step 2: Get PR summary

Call the `create_pr_summary` MCP tool with `phase_num`. You will receive `{ title, body, branch_name }`.

If `create_pr_summary` returns an error, display it and stop.

### Step 3: Display PR preview

Display the PR title and body in a formatted block so the developer can review it:

```
## PR Preview

**Title:** {title}

**Branch:** {branch_name}

**Body:**
{body}
```

Then display: "Creating PR... (auto-continuing in 30 seconds...)"

Wait 30 seconds before proceeding to Step 4.

### Step 4: Create PR

Check whether there are commits ahead of main to ship:

```bash
git status --short
git log origin/main..HEAD --oneline
```

If there are no commits ahead of main, display:

```
No changes to ship. All work is already on main.
```

Skip PR creation and proceed directly to Step 6 (advance phase).

Otherwise, create the PR using the GitHub CLI:

```bash
git checkout -b {branch_name} 2>/dev/null || git checkout {branch_name}
git push -u origin {branch_name}
gh pr create --title "{title}" --body "{body}"
```

If `gh` fails because it is not installed or the user is not authenticated, display the error and instruct the developer:

```
gh CLI is required for PR creation.
  - Install: https://cli.github.com/
  - Authenticate: gh auth login

Once ready, retry with: /gsd-wired:ship {phase_num}
```

Then stop — do not proceed to phase advancement without a successful PR.

### Step 5: Display PR link

Show the PR URL returned by `gh pr create`:

```
PR created: {pr_url}
```

### Step 6: Advance phase

Call the `advance_phase` MCP tool with `phase_num` and `reason="Shipped as PR"`.

Display:

```
Phase {N} complete.
```

If `advance_phase` fails, display the error but note that the PR was already created (if applicable) and no phase state was changed.

### Step 7: Show next phase

If `advance_phase` returns a `next_phase` object, display:

```
Next up: Phase {next_phase.phase_num}: {next_phase.title}
Ready to plan? (auto-continuing to /gsd-wired:plan {next_phase.phase_num} in 30 seconds...)
```

Wait 30 seconds, then auto-proceed to run `/gsd-wired:plan {next_phase.phase_num}`.

If there is no `next_phase`, display:

```
All phases complete. Project shipped.
```

## Error handling

- `create_pr_summary` error: display the error message and stop
- `gh pr create` failure: display error with install/auth instructions, stop (do not advance phase)
- `advance_phase` failure: display error, note PR was already created, do not retry
- No commits to ship: skip PR creation, proceed directly to phase advancement with reason="No changes — already on main"
