 run 

# Leave Management System

Role-based leave management for employees, managers, and administrators. The
production application runs as one Go server with the React frontend embedded.

## Prerequisites

- Go 1.21 or newer
- Node.js 22 or newer, only for frontend development and builds
- A Supabase project

The application has been deployed on cloud service and can be accessed via: https://leave-management-system-zmf0.onrender.com/

# Testing accounts and their passwords for accessing the dashboard

```admin
gmail=haneeff110@gmail.com
password=123456
```

```manager
gmail=hasan@gmail.com
password=hjw4y3ggd
```

```employee
gmail=jack@gmail.com
password=eywhr324cf
```

## Fresh project setup

> Never commit `.env`, database passwords, service-role keys, JWT secrets, or
> production credentials. The repository `.gitignore` excludes `.env` files.
> `VITE_SUPABASE_ANON_KEY` is intended for the browser, but it must not be
> confused with the private `SUPABASE_SERVICE_ROLE_KEY`.

### 1. Install and run

From the project root:

```powershell
npm.cmd run dev
```

The preflight script validates the tools and environment configuration and
automatically installs missing root, frontend, and Go dependencies.

This starts both:

- Go backend: `http://localhost:8080`
- Vite frontend: `http://localhost:5173`

Open the development application at:

```text
http://localhost:5173
```

### 2. Deploy on Render guide

The production React build is embedded into the Go server, so only one Render
Web Service is required.

Configure the Render service as follows:

```text
Service type: Web Service
Runtime: Go
Root directory: backend
Build command: go build -o app ./cmd
Start command: ./app
Health check path: /health/ready
```

Add all four environment variables listed above through Render's Environment
settings.

Before committing frontend changes for the embedded production server, rebuild
and copy the generated files:

```powershell
cd frontend
npm.cmd run build
cd ..

Remove-Item backend/cmd/web/* -Recurse -Force
Copy-Item frontend/dist/* backend/cmd/web/ -Recurse -Force
```

## other alternatives to run the program locally as a single production-style server:

build the react frontend

```powershell
cd frontend
npm.cmd run build
cd ..

Remove-Item backend/cmd/web/* -Recurse -Force
Copy-Item frontend/dist/* backend/cmd/web/ -Recurse -Force
```

After the React frontend has been built and copied into `backend/cmd/web`,
Node.js is not required to run the application:

```powershell
cd backend
go run ./cmd
```

Open:

```text
http://localhost:8080
```

## Tests commands if needed

Run all backend checks:

```powershell
cd backend
go test ./...
go vet ./...
```

Run all frontend checks:

```powershell
cd frontend
npm run check
npm audit
```

Run only API-related tests:

```powershell
cd backend
go test ./internal/handlers -v
go test ./internal/middleware -v
go test ./internal/services -v

cd ../frontend
npm test -- src/services/api.test.js
```

Verify the configured database schema:

```powershell
cd backend
$env:RUN_DB_INTEGRATION_TESTS="1"
go test ./internal/database -run TestFreshSchemaObjectsExist -v
Remove-Item Env:RUN_DB_INTEGRATION_TESTS
```

Check a running backend:

```powershell
Invoke-RestMethod http://localhost:8080/health/live
Invoke-RestMethod http://localhost:8080/health/ready
```

## Documentation

The full workflow, architecture, database design, UI decisions, and screenshots
are documented in:

```text
docs/Leave_Management_System_Assessment_Documentation.docx
```
