# REST API Contract — v1

Base URL: `/plugins/com.mrvnklm.clockwork/api/v1`

**Auth**: the Mattermost server injects the `Mattermost-User-ID` header for
authenticated sessions. The router middleware rejects any request without it
(`401`). Never trust a user id from the body/query — always use the header.
Admin endpoints additionally require `PermissionManageSystem`
(`client.User.HasPermissionTo(userID, model.PermissionManageSystem)`), else `403`.

All request/response bodies are JSON unless noted. Errors use:
`{ "error": "<message>" }` with an appropriate status code.

## TimeEntry JSON shape

Matches `server/store.TimeEntry` (UTC unix **milliseconds**; `0` ⇒ unset):

```json
{
  "id": "abc…",
  "user_id": "xyz…",
  "start_at": 1733050800000,
  "end_at": 0,
  "break_seconds": 0,
  "break_started_at": 0,
  "project": "",
  "description": "",
  "status": "open",
  "locked": false,
  "created_at": 1733050800000,
  "updated_at": 1733050800000
}
```

`status` is the approval-workflow state: `open` → `submitted` → `approved`
(`reject` returns `submitted`→`open`, `reopen` returns `approved`→`open`).
`locked` is **derived**: `locked == (status != "open")`. It is kept in the JSON
so the existing lock banner keeps working; the owner may edit/delete only `open`
entries (otherwise `403` locked).

## Endpoints

### Config (any logged-in user)
| Method | Path | Success |
|---|---|---|
| GET | `/config` | `200 {"approval_enabled": bool}` — reads `configuration.EnableApproval`; tells the webapp whether to show the approval-workflow UI |

### Live timer
| Method | Path | Body | Success | Errors |
|---|---|---|---|---|
| GET  | `/timer/current` | — | `200 {"entry": TimeEntry \| null}` | |
| POST | `/timer/start` | `{"project"?, "description"?}` | `200 {"entry": TimeEntry}` | `409` already running |
| POST | `/timer/stop` | — | `200 {"entry": TimeEntry}` | `409` not running |
| POST | `/timer/break/start` | — | `200 {"entry": TimeEntry}` | `409` not running / already on break |
| POST | `/timer/break/stop` | — | `200 {"entry": TimeEntry}` | `409` not running / not on break |

### Entries (current user)
| Method | Path | Body | Success | Errors |
|---|---|---|---|---|
| GET    | `/entries?from=<ms>&to=<ms>` | — | `200 {"entries": TimeEntry[]}` newest first | `400` bad range |
| POST   | `/entries` | `{"start_at", "end_at"?, "break_seconds"?, "project"?, "description"?}` | `201 {"entry": TimeEntry}` | `400` invalid |
| PUT    | `/entries/{id}` | partial: `{"start_at"?, "end_at"?, "break_seconds"?, "project"?, "description"?}` | `200 {"entry": TimeEntry}` | `400`, `403` locked, `404` |
| DELETE | `/entries/{id}` | — | `204` | `403` locked, `404` |

### Reports (current user)
| Method | Path | Success |
|---|---|---|
| GET | `/reports/summary?from=<ms>&to=<ms>` | `200` grouped-by-day summary (below) |
| GET | `/reports/export?from=<ms>&to=<ms>` | `200 text/csv` of own entries (attachment) |

`/reports/summary` response (days grouped in the **user's Mattermost timezone**):
```json
{
  "total_net_seconds": 144000,
  "days": [
    {
      "date": "2026-06-01",
      "net_seconds": 28800,
      "break_seconds": 1800,
      "entries": [ TimeEntry, … ]
    }
  ]
}
```

### Autocomplete (current user)
| Method | Path | Success |
|---|---|---|
| GET | `/suggestions` | `200 {"projects": string[], "notes": string[]}` — the user's distinct recent project/note values for input autocomplete |

### Admin (require PermissionManageSystem)
| Method | Path | Success | Errors |
|---|---|---|---|
| GET | `/admin/entries?from=<ms>&to=<ms>&user_id=<optional>` | `200 {"entries": TimeEntry[], "usernames": {"<user_id>": "<username>"}}` (username map resolved server-side, capped at 500 distinct users) | `403` |
| GET | `/admin/export?from=<ms>&to=<ms>&user_id=<optional>` | `200 text/csv` (attachment) | `403` |

The optional `user_id` query param scopes **both** `/admin/entries` and
`/admin/export` to a single user (server-side filter). When the admin narrows to
one user the frontend must pass that `user_id` to both so the on-screen view and
the CSV export agree (and so the browser never loads all users' entries). The
`usernames` map on `/admin/entries` resolves each distinct `user_id` to a
username server-side (capped at 500 distinct users; beyond the cap the raw id is
returned).

### Approval workflow (gated on `EnableApproval`)

All endpoints below return **`404`** when the approval workflow is disabled in
the plugin configuration (`EnableApproval = false`, the default), making them
indistinguishable from unrouted paths. They each transition entries within
`[from, to)` whose current status matches; running (open-end) entries are never
transitioned. The response is the affected count.

| Method | Path | Body | Transition | Success | Errors |
|---|---|---|---|---|---|
| POST | `/timesheet/submit` | `{"from":<ms>,"to":<ms>}` | own `open`→`submitted` | `200 {"updated":<n>}` | `400`, `404` (disabled) |
| POST | `/timesheet/withdraw` | `{"from":<ms>,"to":<ms>}` | own `submitted`→`open` | `200 {"updated":<n>}` | `400`, `404` (disabled) |
| POST | `/admin/approve` | `{"user_id":<id>,"from":<ms>,"to":<ms>}` | `submitted`→`approved` | `200 {"updated":<n>}` | `400`, `403`, `404` (disabled) |
| POST | `/admin/reject` | `{"user_id":<id>,"from":<ms>,"to":<ms>}` | `submitted`→`open` | `200 {"updated":<n>}` | `400`, `403`, `404` (disabled) |
| POST | `/admin/reopen` | `{"user_id":<id>,"from":<ms>,"to":<ms>}` | `approved`→`open` | `200 {"updated":<n>}` | `400`, `403`, `404` (disabled) |

The admin endpoints additionally require `PermissionManageSystem` (`403`
otherwise). The `404` (disabled) check precedes the admin check.

## CSV format

Self export columns:
`date,start,end,break_minutes,net_hours,project,description`

Admin export prepends `username`:
`username,date,start,end,break_minutes,net_hours,project,description`

`date` in user/admin-local day, `start`/`end` as ISO-8601 local time, `net_hours`
as decimal hours (2 dp). Header row included.
