# REST API Contract — v1

Base URL: `/plugins/com.vsjwl.mm-time-tracking/api/v1`

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
  "locked": false,
  "created_at": 1733050800000,
  "updated_at": 1733050800000
}
```

## Endpoints

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

### Admin (require PermissionManageSystem)
| Method | Path | Success | Errors |
|---|---|---|---|
| GET | `/admin/entries?from=<ms>&to=<ms>&user_id=<optional>` | `200 {"entries": TimeEntry[]}` | `403` |
| GET | `/admin/export?from=<ms>&to=<ms>&user_id=<optional>` | `200 text/csv` (attachment) | `403` |

## CSV format

Self export columns:
`date,start,end,break_minutes,net_hours,project,description`

Admin export prepends `username`:
`username,date,start,end,break_minutes,net_hours,project,description`

`date` in user/admin-local day, `start`/`end` as ISO-8601 local time, `net_hours`
as decimal hours (2 dp). Header row included.
