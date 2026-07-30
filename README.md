# Leave Management System

Role-based leave management for employees, managers, and administrators. The
production application runs as one Go server with the React frontend embedded.

## Prerequisites

- Go 1.21 or newer
- Node.js 22 or newer, only for frontend development and builds
- A Supabase project

# Three alternatives to run the program:

## 1. Run in development

From the project root:

```powershell
npm.cmd run dev
```

Before starting, this automatically checks the Node.js and Go installations,
required `.env` variables, and Supabase URL formats. Missing project dependencies
are installed automatically. Use `npm.cmd` in PowerShell if script execution
policy blocks `npm.ps1`.

This starts both:

- Go backend: `http://localhost:8080`
- Vite frontend: `http://localhost:5173`

Open the development application at:

```text
http://localhost:5173
```

## 2. Run only the Go backend

From the project root:

```powershell
go run ./backend/cmd
```


## 3. Run as a single production-style server

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

# Testing accounts and their passwords for accessing the dashboard

```admin
gmail=haneeff110@gmail.com
password=123456
```

```manager account
gmail=hasan@gmail.com
password=12345678
```

```employee account
gmail=jack@gmail.com
password=12345678
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
