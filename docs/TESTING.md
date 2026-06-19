# Testing

## Unit tests

```sh
go test ./server/...
cd webapp && npm test
```

These need no database. The store's SQL methods are not exercised here.

## Integration tests (real Postgres + MySQL)

The store integration suite lives in `server/store/*_integration_test.go`, guarded
by the `integration` build tag, and runs against a **real** Postgres and a
**real** MySQL/MariaDB. Each driver is selected by a DSN env var; a driver whose
var is unset is skipped, so a plain `go test ./...` is unaffected.

| Driver   | Env var                       | Example DSN |
|----------|-------------------------------|-------------|
| Postgres | `CLOCKWORK_TEST_POSTGRES_DSN` | `postgres://clockwork:clockwork@localhost:5432/clockwork_test?sslmode=disable` |
| MySQL    | `CLOCKWORK_TEST_MYSQL_DSN`    | `root:root@tcp(localhost:3306)/clockwork_test?parseTime=true` |

The suite covers, on both drivers: `migrate()` idempotency, the additive
status-column `ADD` over a pre-existing table, the single-running-entry invariant
under concurrent `StartTimer`, and the full `SetStatusRange` transition matrix
(submit / approve / reopen / reject, plus running-entry exclusion and
affected-count correctness).

### Quickstart with Docker

```sh
docker run -d --name cw-pg \
  -e POSTGRES_USER=clockwork -e POSTGRES_PASSWORD=clockwork -e POSTGRES_DB=clockwork_test \
  -p 5432:5432 postgres:16-alpine

docker run -d --name cw-mysql \
  -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=clockwork_test \
  -p 3306:3306 mysql:8

export CLOCKWORK_TEST_POSTGRES_DSN="postgres://clockwork:clockwork@localhost:5432/clockwork_test?sslmode=disable"
export CLOCKWORK_TEST_MYSQL_DSN="root:root@tcp(localhost:3306)/clockwork_test?parseTime=true"

make test-integration   # or: go test -tags=integration -race ./server/store/...

docker rm -f cw-pg cw-mysql
```

CI runs this same suite on every push/PR via the `integration` job in
`.github/workflows/ci.yml`, using Postgres and MySQL service containers.
