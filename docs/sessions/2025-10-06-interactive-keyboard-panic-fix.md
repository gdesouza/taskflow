# Session: Fix interactive mode keyboard panic
**Date**: 2025-10-06
**Duration**: ~0.5 hours
**Participants**: Assistant

## Objectives
- Reproduce and diagnose panic: "already waiting for key" when opening task details.
- Implement safe keyboard lifecycle handling to prevent concurrent reads.
- Preserve existing interactive features (navigation, filtering, sorting, editing).

## Key Decisions
- Centralize keyboard ownership in each UI phase (list view vs. details view) to avoid overlapping `keyboard.GetKey` goroutines.
- Explicitly close the keyboard before entering `showTaskDetails` and reopen afterward.
- Add lifecycle handling inside `showTaskDetails` (open at start, defer close) instead of assuming caller state.
- Temporarily close/reopen keyboard around promptui interactions to switch between raw mode and standard input mode cleanly.

## Implementation Summary
Adjusted `interactive.go` so only one active keyboard listener exists at any moment. Entering the details view now releases the list view listener, preventing the panic caused by concurrent `GetKey` calls in separate goroutines.

## Technical Details
### Modified Components
- **cmd/task/interactive.go**: Added keyboard close/open around details invocation; added open/defer close inside `showTaskDetails`; refined edit flow within details view.

## Files Modified/Created
- `cmd/task/interactive.go` - Manage keyboard lifecycle during Enter key handling and within task details view.
- `docs/sessions/2025-10-06-interactive-keyboard-panic-fix.md` - This session record.

## Tests Added
- None (interactive TUI behavior not currently covered by automated tests). Manual validation recommended.

## Documentation Updates
- Added session log documenting rationale and changes.

## Lessons Learned
- Libraries providing global state (like `keyboard`) require strict ownership boundaries in concurrent or multi-phase UI loops.
- Prompt-based flows (promptui) need raw mode suspension to avoid conflicting terminal modes.

## Known Issues/TODOs
- Consider refactoring to a finite-state UI controller to simplify lifecycle management.
- Potential future enhancement: abstract keyboard handling into a small manager struct.

## Next Steps
- Manually regression test other interactive commands (filter, sort, add) for edge cases.
- Optionally add a lightweight integration test harness using pseudo-terminal (pty) for critical flows.

## Related Commits
- (Pending commit for this change.)

---

## Notes
No changes to business logic of tasks; only input handling adjusted.