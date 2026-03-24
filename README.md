# REST API (Go)

A Go REST API backed by SQLite for user registration, JWT authentication, event management (with optional or required image depending on request body format), user sign-ups for events, and static image file serving.

## Stack

- **Go** (module `rest.api`; see `go.mod`)
- **`net/http`** with a `ServeMux` that supports method and path patterns (Go 1.22+)
- **SQLite** via `modernc.org/sqlite` (local file `app.db`)
- **JWT** (`github.com/golang-jwt/jwt/v5`) for protected routes
- **bcrypt** (`golang.org/x/crypto`) for password hashing

## How to run

From the repository root:

```bash
export JWT_SECRET='your-secret-key'
go run ./cmd/api
```

The server listens on **`http://localhost:8080`**.

### Environment variables

| Variable     | Description |
|--------------|-------------|
| `JWT_SECRET` | Key used to sign and verify JWTs. If empty, login/signup do not return a token and routes that require JWT respond with **503** (authentication not configured). |
| `JWT_TTL`    | Token lifetime (e.g. `24h`, `1h`). Default: **24 hours**. |

On first run, migrations create/update tables in `app.db` and the **`images/`** directory is created to store files uploaded with events.

## Entities (data model)

- **User** — `id`, `name`, `email`, `password_hash`, `created_at`. `id` is exposed in responses as a string; the JWT carries it in the `user_id` claim.
- **Event** — `title`, `description`, `address`, `date` (stored as RFC3339), `user_id` (creator), optional `image_url` (path such as `/images/<file>.jpg`).
- **EventRegistration** — many-to-many between a user and an event (`event_id`, `user_id`, `registered_at`) with a composite primary key. Deleting an event removes related registrations (SQLite cascade with foreign keys enabled).

## Authentication

1. **Signup** (`POST /signup`) or **Login** (`POST /login`) with a JSON body; when `JWT_SECRET` is set, the response may include a **`token`** field (HS256 JWT) with `user_id` and `email` claims.
2. Protected routes require the header:
   ```http
   Authorization: Bearer <token>
   ```
3. Middleware validates the JWT and attaches the user to the request context; the token’s `user_id` is used to create events, register/unregister, and for **update/delete authorization**: only the **creator** of the event (`events.user_id`) may `PUT` or `DELETE` that event. Another authenticated user gets **403** with a message that the token is not valid for that action.

Common errors: **401** (missing / invalid / expired token), **503** on protected routes if `JWT_SECRET` is not set.

## File upload (images)

There is **no** standalone upload route. Images are sent **together with the event**:

- **`POST /events`** with `Content-Type: multipart/form-data`: text fields `title`, `description`, `address`, `date` (RFC3339) and a **required** file field **`image`** (JPEG, PNG, GIF, or WebP, up to ~10 MiB). The server stores the file under `images/` with a random name and sets `image_url` to something like `/images/<name>.jpg`.
- **`PUT /events/{id}`** with multipart: same fields; the **`image`** field is **optional** — if omitted, the previous `image_url` is kept.
- **`POST /events`** / **`PUT /events/{id}`** with JSON: `application/json` with the same text fields; optional **`image_url`** string (e.g. a path you already know) without sending a file.

Images are served at **`GET /images/<filename>`** (public, no JWT).

## API — Endpoints

Example base URL: `http://localhost:8080`.

### Users

| Method | Path      | Auth | Description |
|--------|-----------|------|-------------|
| `POST` | `/signup` | No   | Register a user (JSON: `name`, `email`, `password`). |
| `POST` | `/login`  | No   | Sign in (JSON: `email`, `password`). |
| `GET`  | `/users`  | No   | List users. |

### Events

| Method   | Path            | Auth        | Description |
|----------|-----------------|-------------|-------------|
| `GET`    | `/events`       | No          | List all events (ordered by date). |
| `GET`    | `/events/{id}`  | No          | Get one event. |
| `POST`   | `/events`       | Bearer JWT  | Create an event (JSON, or multipart with required `image`). |
| `PUT`    | `/events/{id}`  | Bearer JWT  | Update an event (creator only). |
| `DELETE` | `/events/{id}`  | Bearer JWT  | Delete an event (creator only). |

### Event registrations

| Method   | Path                         | Auth        | Description |
|----------|------------------------------|-------------|-------------|
| `POST`   | `/events/{id}/register`      | Bearer JWT  | Register the token’s user for the event. |
| `DELETE` | `/events/{id}/register`      | Bearer JWT  | Remove the registration. |

### Static images

| Method | Path          | Auth | Description |
|--------|---------------|------|-------------|
| `GET`  | `/images/...` | No   | Serve files from the `images/` folder. |

## Quick `curl` examples

**Login** (the JSON response includes `token` when `JWT_SECRET` is set):

```bash
curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"your-password"}'
```

Copy the `token` value and use `Authorization: Bearer ...` in the following calls (e.g. `export TOKEN='...'`).

**Create an event with an image (multipart):**

```bash
curl -s -X POST http://localhost:8080/events \
  -H "Authorization: Bearer $TOKEN" \
  -F "title=Workshop" \
  -F "description=API introduction" \
  -F "address=123 Main St" \
  -F "date=2026-06-15T14:00:00Z" \
  -F "image=@./photo.jpg"
```

**Create an event with JSON only (no file):**

```bash
curl -s -X POST http://localhost:8080/events \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title":"Meetup",
    "description":"Community meetup",
    "address":"Lisbon",
    "date":"2026-07-01T18:00:00Z",
    "image_url":"/images/example.png"
  }'
```

## Response format

- Success: JSON with objects such as `user`, `event`, `events`, `message`, etc.
- Errors: usually `{"error":"message"}` with an appropriate HTTP status (400, 401, 403, 404, 409, 413, 500, …).

## Project layout (summary)

- `cmd/api` — application entrypoint and route registration
- `auth/` — JWT, middleware, signup/login, user listing
- `internal/db` — SQLite and migrations
- `internal/events` — event and registration handlers and rules
- `internal/uploads` — multipart image validation and storage
- `internal/storage` — user persistence

## CORS note

This API does not set CORS headers. Browser calls from another origin may be blocked; for SPAs on another domain/port, add CORS middleware or use a proxy.
