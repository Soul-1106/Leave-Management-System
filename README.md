# Leave Management System

Role-based leave management for employees, managers, and administrators. The
production application runs as one Go server with the React frontend embedded.

## Prerequisites

- Go 1.21 or newer
- Node.js 22 or newer, only for frontend development and builds
- A Supabase project


## 1. Install and build

Install the root development runner and frontend dependencies:

```powershell
npm install
npm run setup
```

Create the production frontend bundle:

```powershell
cd frontend
npm run build
cd ..
```

Copy the built frontend into the Go server:

```powershell
Remove-Item backend/cmd/web/* -Recurse -Force
Copy-Item frontend/dist/* backend/cmd/web/ -Recurse -Force
```

## 4. Run in development

From the project root:

```powershell
npm run dev
```

This starts both:

- Go backend: `http://localhost:8080`
- Vite frontend: `http://localhost:5173`

Open the development application at:

```text
http://localhost:5173
```

## 5. Run as a single server

After building and copying the frontend into `backend/cmd/web`, Node.js is not
required to run the application:

```powershell
cd backend
go run ./cmd
```

Open:

```text
http://localhost:8080
```

## Tests

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
