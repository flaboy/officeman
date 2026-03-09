# officeman

Internal Go service for workspace-backed Office document operations.

## Current Capability (V1)

- `GET /healthz`
- `POST /v1/workbooks/create`
- `POST /v1/workbooks/meta`
- `POST /v1/workbooks/write-cells`
- `POST /v1/workbooks/append-rows`
- `POST /v1/workbooks/add-sheet`
- `POST /v1/workbooks/rename-sheet`
- `POST /v1/workbooks/delete-sheet`
- `POST /v1/documents/write`
- `POST /v1/documents/read`

## Local Run

```bash
go run ./cmd/officeman
```

Environment variables:

- `OFFICEMAN_PORT` (default `7012`)

## Docker

Build local image:

```bash
docker build -t officeman:dev .
```

Run container:

```bash
docker run --rm -p 7012:7012 officeman:dev
```

## Response Envelope

Same shape as `browserd`:

```json
{
  "data": {},
  "error": null
}
```

Error example:

```json
{
  "data": null,
  "error": {
    "code": "INVALID_REQUEST",
    "message": "sheetName is required"
  }
}
```

## Lifecycle Difference from browserd

`officeman` is request-scoped, not session-scoped.

- `browserd` keeps browser runtime state and profile lifecycle across session APIs
- `officeman` resolves VFS per request
- every workbook or document request must carry its own `vfs.mounts`, `vfs.s3_sets`, and `vfs.template_vars`

## API Example

### Create

```http
POST /v1/workbooks/create
Content-Type: application/json

{
  "vfs": {
    "mounts": {
      "/workdir/": {
        "permission": "read_write",
        "bucket": "private",
        "path": "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
        "ttl_ms": 30000
      }
    },
    "s3_sets": {
      "private": {
        "bucket": "private-bucket"
      }
    },
    "template_vars": {
      "tenant_id": "t1",
      "team_id": "team1",
      "case_id": "c1"
    }
  },
  "filePath": "/workdir/report.xlsx",
  "sheets": [
    { "name": "Sheet1" }
  ]
}
```

### Write Cells

```http
POST /v1/workbooks/write-cells
Content-Type: application/json

{
  "vfs": {
    "mounts": {
      "/workdir/": {
        "permission": "read_write",
        "bucket": "private",
        "path": "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
        "ttl_ms": 30000
      }
    },
    "s3_sets": {
      "private": {
        "bucket": "private-bucket"
      }
    },
    "template_vars": {
      "tenant_id": "t1",
      "team_id": "team1",
      "case_id": "c1"
    }
  },
  "filePath": "/workdir/report.xlsx",
  "sheetName": "Sheet1",
  "startCell": "A1",
  "values": [
    ["name", "score"],
    ["alice", 95]
  ]
}
```

### Write Word Document

```http
POST /v1/documents/write
Content-Type: application/json

{
  "vfs": {
    "mounts": {
      "/workdir/": {
        "permission": "read_write",
        "bucket": "private",
        "path": "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
        "ttl_ms": 30000
      }
    },
    "s3_sets": {
      "private": {
        "bucket": "private-bucket"
      }
    },
    "template_vars": {
      "tenant_id": "t1",
      "team_id": "team1",
      "case_id": "c1"
    }
  },
  "filePath": "/workdir/brief.docx",
  "blocks": [
    { "type": "title", "text": "Weekly Report" },
    { "type": "heading", "level": 1, "text": "Progress" },
    { "type": "paragraph", "text": "Done." },
    {
      "type": "table",
      "rows": [
        ["name", "score"],
        ["alice", 95]
      ]
    }
  ]
}
```

## VFS Path Mapping Example

When:

- `mountPath = /workdir/`
- `mount.path = tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/`
- `filePath = /workdir/report.xlsx`

And:

- `tenant_id = t1`
- `team_id = team1`
- `case_id = c1`

Resolved S3 key becomes:

```text
tenants/t1/teams/team1/cases/c1/workspace/report.xlsx
```

## Current Non-Goals

- `.xls`
- `.doc`
- macros
- charts
- pivot tables
- formula recalculation engine
- style-preserving rich editing
