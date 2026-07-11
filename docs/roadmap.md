# Implementation Roadmap

This roadmap summarizes the agreed incremental plan. Checkboxes describe project state, not requirement priority.

## Increment 1: Dice expression engine

- [x] Handwritten lexer and recursive-descent parser.
- [x] AST and typed scalar/list values.
- [x] Polyhedral and Fudge dice.
- [x] Arithmetic, list, aggregation, filtering, selection, and rounding operations.
- [x] Structured errors with source positions.
- [x] Syntax and semantic validation without rolling.
- [x] Cryptographically secure unbiased rolls.
- [x] Raw roll trace and final result.
- [x] Unit tests.

## Increment 2: Local dice workbench

- [x] Chi-based HTTP server.
- [x] Embedded HTML, CSS, and HTMX.
- [x] Private server-side validation action.
- [x] Final roll action showing expression, raw dice, and result.
- [x] Visible, swappable validation and evaluation errors.
- [x] YAML configuration with strict field checking.
- [x] HTTP handler and configuration tests.

## Increment 3: SQLite persistence and migrations

Recommended next increment.

- [ ] Add `modernc.org/sqlite`.
- [ ] Choose and document UUID storage representation.
- [ ] Implement a versioned migration mechanism.
- [ ] Create users, rooms, room memberships, messages, and macros or macro-storage schema.
- [ ] Represent text, dice roll, and system messages without losing structured roll details.
- [ ] Add agreed unique constraints, foreign keys, and query indexes.
- [ ] Enable SQLite foreign keys and choose WAL/busy-timeout settings.
- [ ] Add repository/query packages rather than placing SQL in HTTP handlers.
- [ ] Add integration tests using temporary databases.
- [ ] Document backup and migration behavior.

Suggested acceptance criteria:

- A new database migrates from zero to the latest schema automatically or via a documented command.
- Re-running migrations is safe.
- Schema constraints prevent duplicate membership and external identity records.
- Messages can be paginated newest-first and displayed chronologically.
- Structured dice traces round-trip without loss.

## Increment 4: exe.dev authentication and users

- [ ] Confirm production proxy topology and trusted-header boundary.
- [ ] Consume verified `X-ExeDev-UserID` and `X-ExeDev-Email` headers.
- [ ] Never trust client-supplied identity headers on an exposed direct listener.
- [ ] Provision/update local user records.
- [ ] Add secure cookie-backed application sessions where needed.
- [ ] Design a clearly isolated local-development authentication mode.
- [ ] Implement administrator email allowlisting.
- [ ] Add authorization middleware and tests.

## Increment 5: Rooms and persistent membership

- [ ] Create rooms with the creator assigned as GM.
- [ ] Implement invitation and approval workflow.
- [ ] Require accepted membership for normal room access.
- [ ] Support room-specific nicknames alongside user identity names.
- [ ] Add GM member-management interface.
- [ ] Implement silence/unsilence with system messages.
- [ ] Implement block/unblock with system messages and access revocation.
- [ ] Implement room deletion.
- [ ] Test every authorization boundary independently of the UI.

## Increment 6: Persistent chat and dice rolls

- [ ] Persist text messages.
- [ ] Privately validate dice expressions before final submission.
- [ ] Revalidate, roll once, and atomically persist final dice results.
- [ ] Store submitted expression, raw rolls, and final value.
- [ ] Load the latest 50 messages.
- [ ] Add keyset-based **Load more** pagination.
- [ ] Keep history immutable for users and GMs.
- [ ] Prevent silenced users from text chat and dice rolls.

## Increment 7: Real-time room updates

- [ ] Add WebSocket support and per-room hub/subscriptions.
- [ ] Authenticate and authorize subscriptions.
- [ ] Broadcast only after database persistence succeeds.
- [ ] Display current room presence, GM status, and silence status.
- [ ] Remove blocked users and revoke active room connections.
- [ ] Add reconnect behavior that recovers missed events from persisted history.
- [ ] Add HTMX polling fallback.
- [ ] Test multiple tabs and multiple rooms.

## Increment 8: Per-room macros

- [ ] Define macro storage model.
- [ ] Add label, expression, optional color, and optional icon.
- [ ] Design runtime parameter syntax and safe substitution.
- [ ] Create, test, edit, reorder, and delete macros.
- [ ] Validate macro expressions server-side.
- [ ] Scope macros to one user in one room.

## Increment 9: Room lifecycle and administration

- [ ] Finalize which events update room activity.
- [ ] Define expired-room user experience.
- [ ] Mark rooms expired after 30 inactive days.
- [ ] Purge rooms after 60 inactive days.
- [ ] Expose an idempotent maintenance command suitable for cron.
- [ ] Add administrator room access, message pruning, and room deletion.
- [ ] Add registered-user and online-presence views.
- [ ] Add administrative audit records where practical.

## Increment 10: Production hardening and deployment

- [ ] Add graceful shutdown.
- [ ] Add request logging, panic recovery, and secure response headers.
- [ ] Add CSRF protection to state-changing HTTP actions.
- [ ] Add origin checks and limits for WebSockets.
- [ ] Add request/body and rate limits where appropriate.
- [ ] Add health/readiness endpoints.
- [ ] Document exe.dev deployment and Login with exe setup.
- [ ] Protect configuration and secret files with application-user ownership and mode `0600`.
- [ ] Document SQLite backups and restoration.
- [ ] Configure the scheduled room-maintenance job.
- [ ] Run full tests, `go vet`, and deployment smoke tests.

## Open decisions to resolve when relevant

- SQLite UUID representation: canonical text versus 16-byte blob.
- Whether normalized macro rows are preferable to JSON on membership records.
- Exact username source/editing rules because exe.dev only guarantees ID and email headers.
- Invitation behavior for an email address that has never authenticated with exe.dev.
- Whether room visits count as activity or only messages/rolls do.
- Whether a 30-day expired room is read-only, hidden, or explicitly restorable.
- Session details and local authentication workflow.
- Runtime macro parameter syntax and substitution rules.
- WebSocket library choice and polling interval.
- Chat retention/pruning policy for administrators.
