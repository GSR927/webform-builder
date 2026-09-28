# Webform Builder & Submission Platform — Assessment Submission

A multi-tenant platform for designing, publishing, and collecting submissions on
dynamic web forms at scale. Built for MailerCloud's first-round technical assessment.

- **ARCHITECTURE.md** — full system design, data flow, requirement-by-requirement
  justification, data model, and an honest "Built vs. Designed" split.
- **TRADEOFFS.md** — three key decisions, alternatives rejected, and why.

## What's Actually Running (Working Slice)

- Go API (`backend/main.go`): create/update draft forms, publish immutable versions,
  serve a published form publicly, accept public submissions with schema-derived
  server-side validation.
- RabbitMQ-backed durability: submissions are published with publisher confirms before
  the API acks the end-user's request; a MySQL inbox fallback covers RabbitMQ
  unavailability.
- A consumer (`backend/consumer`) that batches queue messages into a durable store
  (MySQL by default in this dev setup; swap `USE_CLICKHOUSE=1` + a ClickHouse DSN for
  the production target described in ARCHITECTURE.md).
- A minimal Vue 3 renderer (`frontend/src/FormRenderer.vue`) that dynamically renders
  text/email/select fields, evaluates one conditional-visibility rule, and surfaces
  server-side validation errors per field.
- A k6 load generator (`loadgen/loadgen.js`) simulating a steady baseline followed by a
  25x sudden burst, to demonstrate the submission path staying responsive.
- Automated tests (`backend/tests`): server-side validation against a dynamic schema,
  and a test proving submission-integrity is preserved across a form-version change.

## Prerequisites

- Docker + Docker Compose (recommended — single command, no local Go/Node/MySQL setup)
- OR, for running natively: Go 1.22+, Node 18+, MySQL 8, RabbitMQ 3.13+

## Run It (single command)

```bash
docker compose up --build
```

This starts MySQL (with migrations auto-applied), RabbitMQ (management UI at
`http://localhost:15672`, guest/guest), the Go API on `:8080`, the consumer, and the
Vue dev server on `:5173`.

## Manual Walkthrough

1. **Create a draft form:**
   ```bash
   curl -X POST http://localhost:8080/api/forms \
     -H "Content-Type: application/json" \
     -d '{"name": "Newsletter Signup"}'
   ```
2. **Save a schema to the draft:**
   ```bash
   curl -X PUT http://localhost:8080/api/forms/1/draft \
     -H "Content-Type: application/json" \
     -d '{
       "fields": [
         {"key": "name", "type": "text", "label": "Name", "required": true},
         {"key": "email", "type": "email", "label": "Email", "required": true},
         {"key": "role", "type": "select", "label": "I am a...", "required": true,
          "validators": {"options": ["buyer", "seller"]}},
         {"key": "companyName", "type": "text", "label": "Company", "required": true,
          "showIf": {"field": "role", "equals": "seller"}}
       ]
     }'
   ```
3. **Publish it (creates an immutable version):**
   ```bash
   curl -X POST http://localhost:8080/api/forms/1/publish
   ```
4. **View the public form:** open `http://localhost:5173?formId=1`, or fetch the raw
   schema with `curl http://localhost:8080/f/1`.
5. **Submit to it:**
   ```bash
   curl -X POST http://localhost:8080/f/1/submit \
     -H "Content-Type: application/json" \
     -d '{"versionId": 1, "payload": {"name": "Asha", "email": "asha@example.com", "role": "buyer"}}'
   ```

## Running Tests

```bash
cd backend
go test ./... -v
```

## Running the Load Generator

```bash
# install k6 locally, then:
BASE_URL=http://localhost:8080 FORM_ID=1 VERSION_ID=1 k6 run loadgen/loadgen.js
```

## Repository Layout

```
backend/           Go API, models, validation, queue publisher, consumer, tests
  db/migrations/    MySQL schema (forms, form_versions, submissions_inbox, submissions)
  consumer/         RabbitMQ -> durable-store batch consumer
frontend/           Vue 3 public form renderer
clickhouse/         Production submissions schema (ClickHouse), see TRADEOFFS.md
loadgen/            k6 burst-load script
ARCHITECTURE.md     Primary design deliverable
TRADEOFFS.md        Three key decisions and rejected alternatives
```

## Honest Scope Note

This is a timeboxed vertical slice, not a production build. See ARCHITECTURE.md's
"Built vs. Designed" section for exactly what's implemented versus designed-but-not-
built (HAProxy/autoscaling, CDN, per-tenant rate tiers, CAPTCHA, DLQ replay tooling,
ClickHouse materialized views, full drag-drop builder UI).
