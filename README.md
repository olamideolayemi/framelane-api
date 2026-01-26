# FrameLane API

Backend API for FrameLane. Provides authentication, frame catalogs, orders, payments, uploads, and admin management.

Base URL (local): `http://localhost:8080`

## Quickstart

1) Configure environment variables (see Configuration).
2) Run the server:

```bash
go run .
```

## Configuration

Environment variables are loaded from `.env` (or your shell). See `.env.example` for a template.

Required:

| Variable | Description |
| --- | --- |
| `DATABASE_URL` | PostgreSQL connection string |
| `JWT_SECRET` | JWT signing secret |

Common:

| Variable | Description |
| --- | --- |
| `JWT_EXPIRES_HOURS` | Token expiration in hours |
| `S3_ENDPOINT` | S3/MinIO endpoint |
| `S3_USE_SSL` | Use SSL for S3 (`true`/`false`) |
| `S3_ACCESS_KEY` | S3 access key |
| `S3_SECRET_KEY` | S3 secret key |
| `S3_BUCKET` | S3 bucket name |
| `S3_REGION` | S3 region |
| `SMTP_HOST` | SMTP host |
| `SMTP_PORT` | SMTP port |
| `SMTP_USER` | SMTP user |
| `SMTP_PASS` | SMTP password |
| `SMTP_FROM_EMAIL` | Sender email |
| `STRIPE_SECRET` | Stripe secret key |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret |
| `CURRENCY` | Currency code (default: `ngn`) |

## Authentication

JWT authentication is used for protected routes. Use:

```
Authorization: Bearer <token>
```

Admin routes require a token where the user is marked as admin.

## Response Envelope

All endpoints return a standard envelope:

Success:

```json
{
  "success": true,
  "data": { },
  "meta": { }
}
```

Error:

```json
{
  "success": false,
  "error": {
    "message": "invalid request body",
    "code": "bad_request",
    "details": "..."
  }
}
```

Error codes map to HTTP status:

- `bad_request` (400)
- `unauthorized` (401)
- `forbidden` (403)
- `not_found` (404)
- `conflict` (409)
- `unprocessable_entity` (422)
- `rate_limited` (429)
- `service_unavailable` (503)
- `internal_error` (500 and default)

## Pagination

List endpoints include pagination metadata:

```json
"meta": {
  "page": 1,
  "limit": 10,
  "total": 100,
  "total_pages": 10
}
```

Frame list endpoints use `page` and `per_page`. Order and user list endpoints use `page` and `limit`.

## Data Models (API-facing)

User (safe response):
- `id`, `email`, `name`, `phone`, `address`, `is_admin`, `is_active`, `created_at`, `updated_at`

FrameSize:
- `id`, `name`, `price`, `status`, `created_at`, `updated_at`

Frame (type):
- `id`, `name`, `description`, `status`, `image_url`, `created_at`, `updated_at`

Order:
- `id`, `orderId`, `user`, `frame`, `size`, `price`, `imageUrl`, `status`, `notes`, `createdAt`, `updatedAt`

## Status Values

Frame size/type status:
- `available`
- `out_of_stock`

Order status (canonical values stored):
- `Pending`
- `In Progress`
- `Processing`
- `Shipped`
- `Transit`
- `Delivered`
- `Cancelled`

The API accepts common lower-case and hyphen/underscore variants and normalizes to the canonical values above.

## Endpoints

### Health

- `GET /v1/health`

### Auth

- `POST /v1/auth/register`
- `POST /v1/auth/login`

### Frames (public)

- `GET /v1/frames/size`
  - Query: `page`, `per_page`, `status`, `sort_by`, `sort_dir`, `search`
- `GET /v1/frames`
  - Query: `page`, `per_page`, `status`, `sort_by`, `sort_dir`, `search`

### Orders (user)

- `GET /v1/orders` (auth)
- `POST /v1/orders` (auth)
- `GET /v1/track/:orderId` (public)

### Users (user)

- `PUT /v1/user/profile` (auth)

### Admin

Orders:
- `GET /v1/admin/orders`
- `PATCH /v1/admin/orders/:id/status`
- `DELETE /v1/admin/orders/:id`

Users:
- `GET /v1/admin/users`
- `GET /v1/admin/users/:id`
- `PATCH /v1/admin/users/:id/suspend`
- `DELETE /v1/admin/users/:id`

Frame sizes:
- `POST /v1/admin/frames/size`
- `PUT /v1/admin/frames/size/:id`
- `DELETE /v1/admin/frames/size/:id`

Frame types:
- `POST /v1/admin/frames`
- `PUT /v1/admin/frames/:id`
- `DELETE /v1/admin/frames/:id`

### Uploads

- `GET /v1/upload-url` (auth)
  - Query: `filename` (required), `content_type` (optional)

### Payments

- `POST /v1/payments/intent`
- `POST /v1/payments/webhook` (Stripe)

### WebSocket

- `GET /ws`

The WebSocket is a simple broadcast hub with no authentication and permissive origin checks. It is currently not integrated into order events.

## Examples

Register:

```bash
curl -X POST http://localhost:8080/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "Jane Doe",
    "phone": "555-0100",
    "address": "12 Main St"
  }'
```

Login:

```bash
curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

List frame sizes:

```bash
curl "http://localhost:8080/v1/frames/size?page=1&per_page=20&status=available&sort_by=price&sort_dir=asc"
```

Create order:

```bash
curl -X POST http://localhost:8080/v1/orders \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "address": "12 Main St",
    "frameId": "FRAME_UUID",
    "sizeId": "SIZE_UUID",
    "notes": "Leave at door",
    "imageUrl": "https://cdn.example.com/uploads/image.jpg"
  }'
```

Update order status (admin):

```bash
curl -X PATCH http://localhost:8080/v1/admin/orders/FL-ABC123/status \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{ "status": "processing" }'
```

Create payment intent:

```bash
curl -X POST http://localhost:8080/v1/payments/intent \
  -H "Content-Type: application/json" \
  -d '{ "order_id": "FL-ABC123" }'
```

Create frame type (admin, multipart):

```bash
curl -X POST http://localhost:8080/v1/admin/frames \
  -H "Authorization: Bearer <admin-token>" \
  -F "name=Wood Frame" \
  -F "description=Natural wood frame" \
  -F "status=available" \
  -F "image=@/path/to/image.jpg"
```

Get presigned upload URL:

```bash
curl "http://localhost:8080/v1/upload-url?filename=orders/abc.jpg&content_type=image/jpeg" \
  -H "Authorization: Bearer <token>"
```

## Frame Type Images (Local)

When creating or updating a frame type with an `image` file, the API stores it under:

```
./uploads/frames/<uuid>_<original_filename>
```

The response includes `image_url` like `/uploads/frames/<filename>`.

## Payments and Webhooks

- Payment intent amount is derived from the order's frame size price.
- Stripe metadata includes `orderId`, and the webhook uses this to update the order.
- On `payment_intent.succeeded`, order status moves to `Processing` and a status update email is sent.

## Rate Limiting and CORS

- Rate limit is enabled (10 requests per second, per IP).
- CORS allows `http://framelane-framer-app-v1.2.vercel.app` and `http://localhost:3000`.

## Deployment

- Render Postgres setup: `RENDER_POSTGRES_SETUP.md`
