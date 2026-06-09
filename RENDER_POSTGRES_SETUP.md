# Render Postgres Setup (Step-by-Step)

This guide shows how to create a PostgreSQL database on Render and connect it to this API.

## 1) Create the database

1. Log in to Render.
2. Click **New +** in the top-right.
3. Select **PostgreSQL**.
4. Choose:
   - **Name**: e.g., `framelane`
   - **Region**: same region as your web service
   - **Plan**: start with Free/Starter as needed
5. Click **Create Database**.

## 2) Get the connection string

1. Open the new database in Render.
2. Find **Internal Database URL** (preferred for services on Render).
3. Copy the URL.

Tip:
- Use **Internal Database URL** for Render services.
- Use **External Database URL** for local development.

## 3) Add environment variables to the web service

1. Open your web service in Render.
2. Go to **Environment**.
3. Set:
   - `DATABASE_URL` = `<Internal Database URL>`
   - `JWT_SECRET` = `<your secret>`
   - Any other required vars from `.env.example`
4. Click **Save Changes**.

## 4) Redeploy

1. Go to **Deploys** in your web service.
2. Click **Deploy latest commit** or **Clear build cache & deploy**.

## 5) Verify

1. Check logs for successful DB connection.
2. Visit `/v1/health` to confirm the API is running.

If you see connection errors, verify:
- `DATABASE_URL` is set to the Render Internal URL.
- The database and web service are in the same Render region.


test+1769397428@example.com
password123