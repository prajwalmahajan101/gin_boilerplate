---
name: Bug report
about: Report a defect so it can be reproduced and fixed
title: "bug: "
labels: bug
assignees: ""
---

## Describe the bug

A clear description of what went wrong.

## To reproduce

Steps / request that triggers it:

1. `curl ...`
2. ...

## Expected vs actual

- **Expected:**
- **Actual:** (include the response envelope and the `request_id`)

## Logs

```
paste relevant structured logs (match on request_id)
```

## Environment

- Version / commit:
- OS / arch:
- Go version (`go version`):
- Postgres / Valkey versions:

## Additional context

Anything else that helps — config, whether Valkey/DB were up, etc.
