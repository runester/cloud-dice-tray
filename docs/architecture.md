# Architecture and Product Decisions

Status: agreed direction as of 2026-07-11. This document records decisions made during initial planning and implementation. Items under **Future work** are requirements, not necessarily implemented features.

## Product scope and delivery strategy

Cloud Dice Tray is a low-traffic shared dice rolling and chat application for tabletop role-playing games. Expected usage is fewer than 50 users, typically weekly or biweekly.

Development is incremental:

1. Dice expression parser and evaluator.
2. Local web workbench for private validation and rolling.
3. SQLite persistence and migrations.
4. Authentication and user provisioning.
5. Rooms, invitations, and persistent membership.
6. Persistent chat and dice rolls.
7. Real-time updates and presence.
8. Macros, administration, and room lifecycle maintenance.

The first two increments are implemented. See [roadmap.md](roadmap.md) for planned work.

## Technology choices

- Latest stable Go release. The module currently targets Go 1.25.
- Chi as the lightweight HTTP router on top of `net/http`.
- HTML/CSS and HTMX for the interface.
- HTMX is embedded in the executable rather than loaded from a CDN.
- YAML configuration using `go.yaml.in/yaml/v3` with unknown-field rejection.
- SQLite via `modernc.org/sqlite`, a pure-Go driver with no CGO requirement. This dependency is planned but not added yet.
- A single compiled Go executable with embedded web assets.
- Apache License 2.0, copyright 2026 runester@gmail.com.

## Configuration

Application settings are stored in YAML. The initial keys are:

- `server.listen_address`
- `server.base_url`
- `database.path`
- `security.session_secret_file`
- `security.admin_emails`

More settings can be added as they are needed. Production secrets may be stored in a separate plain-text file owned by the application user and set to mode `0600`. Local `config.yaml` is ignored by Git; `config.example.yaml` is the documented template.

The planned production database path is:

```text
/var/lib/cloud-dice-tray/cloud-dice-tray.db
```

## Dice expression language

### Parsing and validation

- The parser is handwritten using a lexer and recursive descent.
- The AST represents nested expressions and is evaluated from innermost nodes outward.
- Parsing and semantic validation do not consume randomness.
- Final evaluation revalidates the submitted expression and rolls exactly once.
- Server-side validation returns specific error messages, stable error codes, and source positions.
- Invalid expressions must never be added to future chat history.
- Errors that can be determined statically, such as using `3d6` directly in arithmetic, must be reported during validation rather than deferred until rolling.

### Values and lists

- Numeric values preserve whether they are integers or floating-point numbers.
- Dice expressions produce flat lists of numbers.
- Numeric literals produce scalars.
- Commas combine and flatten values into one list.
- A one-element list is accepted as a scalar in arithmetic. Thus `d20 + 5` is shorthand for `sum(d20) + 5`.
- A multi-element list cannot be used directly in arithmetic. For example, `3d6 + 2` is invalid; use `sum(3d6) + 2`.
- Errors are structured Go errors, not ordinary DSL values.

### Dice notation and limits

- `d`, `1d`, and `1d6` each mean one six-sided die.
- `3d` means three six-sided dice.
- `d20` means one twenty-sided die.
- `dF` means one Fudge die.
- Fudge dice return `-1`, `0`, or `1` with equal probability. They can be modeled as `ceil(1d6 / 2) - 2`.
- Dice operands must be literals. Computed forms such as `(1+2)d6` and `2d(3+3)` are not supported.
- Functions and Fudge notation are case-insensitive.
- Arbitrary whitespace is accepted.
- Maximum expression length: 256 characters.
- Maximum total dice rolled by a complete expression: 25.
- Per dice term, count must be from 1 through 25.
- Polyhedral dice must have from 2 through 120 sides.
- Maximum parenthesis/function nesting depth: 10.

### Operators

Conventional precedence is used, from highest to lowest:

1. Parentheses and function calls.
2. Dice notation.
3. Unary `+` and `-`.
4. Multiplication and division.
5. Addition and subtraction.
6. Comma/list construction.

Binary arithmetic operators are left-associative. Division always produces a floating-point value. Exact division by zero is an error; very small nonzero divisors are valid.

### Functions

Supported signatures are:

- `sum(values...)`
- `min(values...)`
- `max(values...)`
- `count(values...)`
- `maxk(k, values...)`
- `mink(k, values...)`
- `equals(target, values...)`
- `above(threshold, values...)`
- `below(threshold, values...)`
- `round(value)`
- `floor(value)`
- `ceil(value)`

Function behavior:

- Filter functions return an empty list when nothing matches.
- `count([])` is `0` and `sum([])` is `0`.
- `min([])` and `max([])` are errors.
- `k` must be an integer greater than or equal to 1.
- `maxk` and `mink` cap `k` at the number of supplied values.
- Rounding follows Go's mathematical behavior: halves round away from zero; floor rounds toward negative infinity; ceil rounds toward positive infinity.
- Decimal numeric literals are supported.
- Integral values are displayed without an unnecessary `.0` suffix.

### Randomness and result trace

- Production evaluation uses `crypto/rand` with unbiased rejection sampling.
- The random source is injectable for deterministic tests.
- Results retain the submitted expression, every raw die result grouped by dice term, and the final scalar or list result.
- Intermediate function results do not need to be displayed.
- Future chat entries will reveal the server-generated result to all room participants through the shared history.

## Authentication and identity

The original Google OAuth and emailed temporary-password ideas were dropped. Authentication will use exe.dev's authentication service.

Authenticated requests proxied by exe.dev include:

```text
X-ExeDev-UserID: stable unique user identifier
X-ExeDev-Email: authenticated user's email address
```

The application must only trust these headers when requests arrive through a trusted exe.dev proxy. A direct public listener must not allow clients to spoof identity headers. The precise production trust boundary and local-development authentication bypass must be designed before implementing authentication.

The stable exe.dev user ID should be retained as the external identity. Email is available for display, lookup, invitations, and administrator allowlisting. Authentication itself does not need to expire according to an application-specific inactivity policy.

Application sessions will use secure cookies. Although authentication cookies are browser-wide, each room page and its real-time connection are tab-specific. The intended user experience is one room per browser tab.

Administrators are selected through an email allowlist in YAML configuration.

exe.dev can send email through:

```text
POST http://169.254.169.254/gateway/email/send
Content-Type: application/json

{
  "to": "user@example.com",
  "subject": "Subject",
  "body": "Message body"
}
```

Because exe.dev email is limited to users authenticated through its service, it is suitable for invitations and notifications after user provisioning.

Deployment documentation:

- <https://exe.dev/docs/>
- <https://exe.dev/docs/login-with-exe>
- <https://exe.dev/docs/send-email>

## Persistent domain model

UUIDs are planned for primary domain identifiers. Exact SQLite UUID storage representation remains an implementation decision.

### User

- `id`: UUID, primary key
- `exe_dev_user_id`: stable unique external identifier
- `email`: unique
- `username`
- `created_at`

Room chat displays the user's application/identity name together with their room-specific nickname, which can represent a role-playing character.

### Room

- `id`: UUID, primary key and shareable room identifier
- `title`
- `created_at`
- `last_active_at`
- `gm_id`: user foreign key
- lifecycle state or timestamps needed for expiration

A room has exactly one designated GM: its creator.

### Room membership

- `id`: UUID, primary key
- `user_id`: user foreign key
- `room_id`: room foreign key
- `nickname`
- `macros`: initially proposed as JSON; a normalized table may be used if it improves validation and querying
- `joined_at`
- status: `invited`, `accepted`, `blocked`, or `left`
- a separate silenced flag or moderation state, because silencing and membership status are independent

Membership is persistent. Possession of a room URL or identifier is not sufficient to enter: users must be explicitly invited or approved.

### Chat message

- `id`: UUID, primary key
- `room_id`: room foreign key
- `user_id`: nullable for system messages if appropriate
- message kind: text, dice roll, or system/moderation event
- `content`
- structured dice expression/result data where applicable
- `sent_at`

Chat history is immutable to ordinary users and GMs. Only administrators may prune it.

### Recommended indexes

Users:

- Unique index on `email`.
- Unique index on `exe_dev_user_id`.
- Primary key index on `id`.

Rooms:

- Index on `gm_id`.
- Consider `(gm_id, id)` for rooms managed by a GM.
- Index lifecycle timestamps used by expiration and purge jobs.

Room memberships:

- Unique `(user_id, room_id)`; one membership record per user per room.
- `(room_id, status)` for listing/filtering room members.
- `(user_id, status)` for listing a user's rooms.
- Individual `room_id` and `user_id` indexes only where not made redundant by composite indexes and actual query plans.

Chat messages:

- `(room_id, sent_at DESC)` for chat pagination.
- `(room_id, user_id, sent_at DESC)` for a user's messages in a room.
- `(user_id, sent_at DESC)` for administrator queries across rooms.
- Additional individual indexes should be justified against SQLite query plans to avoid redundant write overhead.

Database foreign keys must be enabled. Migrations should be versioned and run safely at application startup or through a documented command.

## Rooms and permissions

### Room lifecycle

- A room becomes expired after 30 days of inactivity.
- It is purged after 60 days of inactivity.
- Expiration should make the room unavailable or read-only according to behavior finalized during implementation; it must not immediately destroy history.
- A cron job is acceptable for lifecycle processing. The executable should expose an idempotent maintenance command suitable for cron rather than requiring direct SQL.
- Activity that resets `last_active_at` should at minimum include messages and dice rolls; whether room visits count remains to be finalized.

### GM permissions

The room creator is its GM. The GM can:

- Invite or approve members.
- Cancel pending invitations.
- Silence and unsilence accepted members.
- Block and unblock members.
- Delete the room.

Silencing prevents the member from changing chat history: they cannot send text messages or roll dice. The member and GM must both see clearly that the member is silenced, and unsilencing must be easy. Silence and unsilence actions create system entries in chat history.

Blocking removes the member from the room and prevents future room access without affecting access to other rooms. Block and unblock actions create system entries in chat history. Blocked users are not displayed in current room presence, but the GM needs a management view for blocked members.

Authorization must be enforced server-side for HTTP, polling, and WebSocket paths; hiding controls is not sufficient.

## Chat and real-time communication

- Initial room load fetches the latest 50 chat messages.
- A **Load more** control paginates older messages.
- Text chats, dice rolls, and moderation/system events share the room history.
- Dice syntax is privately validated before submission.
- Final dice submission is revalidated, rolled once on the server, persisted, and broadcast.
- All connected room participants should receive the persisted roll at effectively the same time.

WebSockets are preferred for real-time delivery. HTMX polling is the fallback.

Connections use explicit room subscriptions:

- A room browser tab subscribes to one room.
- A user can open multiple rooms by using multiple tabs.
- The server verifies accepted membership on subscription and throughout the connection lifetime.
- Disconnecting removes that tab from room presence.
- Membership or moderation changes must invalidate or update active connections as appropriate.

Room presence displays users currently accessing the room, including GM and silenced indicators. Blocked users are removed rather than shown. Events will include, at minimum:

- New text message.
- New dice roll.
- Moderation/system message.
- Presence join/leave or a room-presence snapshot.
- Membership and silence state changes relevant to active clients.

A hub-and-spoke implementation keyed by room ID is suitable. Persist an event before broadcasting it so reconnection/polling can recover state from SQLite.

## Macros

Macros are per-user-per-room and persist whenever that user returns to that room. A macro includes:

- Label.
- Dice expression.
- Optional color.
- Optional icon.
- Option to request a parameter when clicked.

The runtime parameter syntax and safe substitution rules remain to be designed. Macro expressions must pass the same server-side validation and limits as manually entered expressions.

## Administration

Administrators can:

- Enter and inspect any room.
- Prune old chat history.
- Delete unused rooms.
- View registered users.
- See who is currently online.

Administrative actions should be authorized using the configured email allowlist and should create an audit record where practical.

## Deployment

Development occurs locally on Fedora. Production is an exe.dev VM running its exeuntu Linux distribution, likely behind exe.dev's proxy and authentication layer.

The server listens on a configurable nonstandard address/port. TLS and external routing are expected to be handled by the hosting proxy. Production setup should create an application user and protect:

- The YAML configuration.
- Session secret file (`0600`).
- SQLite database directory and file.
- Backups.

The application should support graceful shutdown before production deployment.
