-- Complete fresh-database setup for the Leave Management System.
-- Run this file once in the Supabase SQL editor after creating/resetting a project.
BEGIN;

-- Enable required PostgreSQL extensions.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- Enum Types
DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('employee', 'manager', 'admin');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE leave_status AS ENUM ('pending', 'approved', 'rejected');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Departments Table
CREATE TABLE IF NOT EXISTS departments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT
);

-- Users Table (Extends Supabase Auth)
CREATE TABLE IF NOT EXISTS users (
    -- In Supabase, this ID should match auth.users.id
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role user_role NOT NULL DEFAULT 'employee',
    manager_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_cannot_manage_self CHECK (manager_id IS NULL OR manager_id <> id)
);

-- Employees Table
CREATE TABLE IF NOT EXISTS employees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    employee_id VARCHAR(50) UNIQUE NOT NULL, -- e.g., EMP001
    designation VARCHAR(255) NOT NULL,
    joining_date DATE NOT NULL,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    UNIQUE(user_id)
);

-- Provision the public application profile from each verified Supabase Auth user.
-- Set app_metadata.role to manager/admin only through a trusted admin workflow.
CREATE OR REPLACE FUNCTION public.handle_new_auth_user()
RETURNS TRIGGER
SECURITY DEFINER SET search_path = public
AS $$
DECLARE
    requested_role user_role;
BEGIN
    requested_role := CASE
        WHEN NEW.raw_app_meta_data->>'role' IN ('employee', 'manager', 'admin')
        THEN (NEW.raw_app_meta_data->>'role')::user_role
        ELSE 'employee'::user_role
    END;

    INSERT INTO public.users (id, email, full_name, role)
    VALUES (
        NEW.id,
        NEW.email,
        COALESCE(NULLIF(NEW.raw_user_meta_data->>'full_name', ''), split_part(NEW.email, '@', 1)),
        requested_role
    );

    IF requested_role = 'employee' THEN
        INSERT INTO public.employees (user_id, employee_id, designation, joining_date)
        VALUES (
            NEW.id,
            COALESCE(NULLIF(NEW.raw_app_meta_data->>'employee_id', ''), 'EMP-' || upper(substr(replace(NEW.id::text, '-', ''), 1, 8))),
            COALESCE(NULLIF(NEW.raw_user_meta_data->>'designation', ''), 'Employee'),
            CURRENT_DATE
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS on_auth_user_created ON auth.users;
CREATE TRIGGER on_auth_user_created
    AFTER INSERT ON auth.users
    FOR EACH ROW EXECUTE FUNCTION public.handle_new_auth_user();

-- Backfill profiles for Auth users that existed before this trigger.
INSERT INTO public.users (id, email, full_name, role)
SELECT
    au.id,
    au.email,
    COALESCE(NULLIF(au.raw_user_meta_data->>'full_name', ''), split_part(au.email, '@', 1)),
    CASE
        WHEN au.raw_app_meta_data->>'role' IN ('employee', 'manager', 'admin')
        THEN (au.raw_app_meta_data->>'role')::user_role
        ELSE 'employee'::user_role
    END
FROM auth.users au
ON CONFLICT (id) DO NOTHING;

INSERT INTO public.employees (user_id, employee_id, designation, joining_date)
SELECT
    u.id,
    COALESCE(NULLIF(au.raw_app_meta_data->>'employee_id', ''), 'EMP-' || upper(substr(replace(u.id::text, '-', ''), 1, 8))),
    COALESCE(NULLIF(au.raw_user_meta_data->>'designation', ''), 'Employee'),
    CURRENT_DATE
FROM public.users u
JOIN auth.users au ON au.id=u.id
WHERE u.role='employee'
ON CONFLICT (user_id) DO NOTHING;

-- Leave Types Table
CREATE TABLE IF NOT EXISTS leave_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE NOT NULL,
    max_days_per_year INTEGER NOT NULL CHECK (max_days_per_year > 0),
    description TEXT
);

-- Leaves Table
CREATE TABLE IF NOT EXISTS leaves (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    leave_type_id UUID NOT NULL REFERENCES leave_types(id) ON DELETE CASCADE,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    reason TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    status leave_status NOT NULL DEFAULT 'pending',
    approved_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    approval_date TIMESTAMP WITH TIME ZONE,
    attachment_path TEXT,
    attachment_name TEXT,
    attachment_type VARCHAR(100),
    attachment_size INTEGER CHECK (
        attachment_size IS NULL OR attachment_size BETWEEN 1 AND 5242880
    ),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_leave_dates CHECK (end_date >= start_date),
    CONSTRAINT leave_within_one_year CHECK (
        EXTRACT(YEAR FROM start_date) = EXTRACT(YEAR FROM end_date)
    ),
    CONSTRAINT valid_leave_decision CHECK (
        (status = 'pending' AND approved_by_id IS NULL AND approval_date IS NULL)
        OR
        (status IN ('approved', 'rejected') AND approved_by_id IS NOT NULL AND approval_date IS NOT NULL)
    )
);

-- Indexes for Leaves Table
CREATE INDEX IF NOT EXISTS idx_leaves_employee_id ON leaves(employee_id);
CREATE INDEX IF NOT EXISTS idx_leaves_status ON leaves(status);
CREATE INDEX IF NOT EXISTS idx_leaves_start_date ON leaves(start_date);
CREATE INDEX IF NOT EXISTS idx_leaves_composite ON leaves(employee_id, status, start_date);
CREATE INDEX IF NOT EXISTS idx_users_manager_id ON users(manager_id);

-- Pending and approved leave may not overlap for the same employee.
DO $$ BEGIN
    ALTER TABLE leaves ADD CONSTRAINT no_overlapping_active_leave
    EXCLUDE USING gist (
        employee_id WITH =,
        daterange(start_date, end_date, '[]') WITH &&
    )
    WHERE (status IN ('pending', 'approved'));
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Leave Balances Table
CREATE TABLE IF NOT EXISTS leave_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    leave_type_id UUID NOT NULL REFERENCES leave_types(id) ON DELETE CASCADE,
    year INTEGER NOT NULL CHECK (year BETWEEN 2000 AND 2100),
    total_allocated INTEGER NOT NULL CHECK (total_allocated >= 0),
    used INTEGER NOT NULL DEFAULT 0 CHECK (used >= 0),
    remaining INTEGER NOT NULL CHECK (remaining >= 0),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(employee_id, leave_type_id, year),
    CONSTRAINT valid_balance CHECK (used <= total_allocated AND remaining = total_allocated - used)
);

-- Triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_leaves_modtime ON leaves;
CREATE TRIGGER update_leaves_modtime
    BEFORE UPDATE ON leaves
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

DROP TRIGGER IF EXISTS update_leave_balances_modtime ON leave_balances;
CREATE TRIGGER update_leave_balances_modtime
    BEFORE UPDATE ON leave_balances
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_column();

-- Only employee-role users may own an employee profile. Managers and admins
-- are management-only accounts in this assessment scope.
CREATE OR REPLACE FUNCTION enforce_employee_profile_role()
RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM users WHERE id = NEW.user_id AND role = 'employee'
    ) THEN
        RAISE EXCEPTION 'employee profiles require an employee-role user';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER SET search_path = public;

DROP TRIGGER IF EXISTS employees_require_employee_role ON employees;
CREATE TRIGGER employees_require_employee_role
    BEFORE INSERT OR UPDATE OF user_id ON employees
    FOR EACH ROW EXECUTE FUNCTION enforce_employee_profile_role();

CREATE OR REPLACE FUNCTION prevent_role_change_with_employee_profile()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.role <> 'employee' AND EXISTS (
        SELECT 1 FROM employees WHERE user_id = NEW.id
    ) THEN
        RAISE EXCEPTION 'remove the employee profile before changing this role';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER SET search_path = public;

DROP TRIGGER IF EXISTS users_preserve_employee_role ON users;
CREATE TRIGGER users_preserve_employee_role
    BEFORE UPDATE OF role ON users
    FOR EACH ROW EXECUTE FUNCTION prevent_role_change_with_employee_profile();

-- The Go API is the only data access layer. Block direct anon/authenticated
-- access through Supabase REST so roles and approval records cannot be changed
-- outside the server-side authorization checks.
ALTER TABLE departments ENABLE ROW LEVEL SECURITY;
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE employees ENABLE ROW LEVEL SECURITY;
ALTER TABLE leave_types ENABLE ROW LEVEL SECURITY;
ALTER TABLE leaves ENABLE ROW LEVEL SECURITY;
ALTER TABLE leave_balances ENABLE ROW LEVEL SECURITY;

-- Required reference data. Stable IDs keep API behavior and test fixtures
-- deterministic across fresh database installations.
INSERT INTO departments (id, name, description) VALUES
    ('11111111-1111-1111-1111-111111111111', 'Engineering', 'Software engineering team'),
    ('22222222-2222-2222-2222-222222222222', 'HR', 'Human resources and operations'),
    ('33333333-3333-3333-3333-333333333333', 'Finance', 'Finance and operations')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description;

INSERT INTO leave_types (id, name, max_days_per_year, description) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Casual Leave', 12, 'Short notice leave for personal reasons'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Annual Leave', 20, 'Planned annual vacation'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Sick Leave', 10, 'Medical leave')
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    max_days_per_year = EXCLUDED.max_days_per_year,
    description = EXCLUDED.description;

-- Private supporting-document storage. Files are accessed only through the Go
-- API with the server-side service-role key.
INSERT INTO storage.buckets (id, name, public, file_size_limit, allowed_mime_types)
VALUES (
    'leave-attachments',
    'leave-attachments',
    false,
    5242880,
    ARRAY['application/pdf', 'image/jpeg', 'image/png']
)
ON CONFLICT (id) DO UPDATE
SET public = EXCLUDED.public,
    file_size_limit = EXCLUDED.file_size_limit,
    allowed_mime_types = EXCLUDED.allowed_mime_types;

-- Least-privilege role for the Go API. Assign a password separately in a
-- protected SQL session, then use this role in SUPABASE_DB_URL.
DO $$ BEGIN
    CREATE ROLE leave_app NOLOGIN BYPASSRLS;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;
ALTER ROLE leave_app BYPASSRLS;

GRANT USAGE ON SCHEMA public TO leave_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON
    departments, users, employees, leave_types, leaves, leave_balances
TO leave_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO leave_app;

COMMIT;
