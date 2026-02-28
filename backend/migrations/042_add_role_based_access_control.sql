-- Migration 042: Add role-based access control for admin/founder features
-- Adds role column to users table for managing dashboard access

ALTER TABLE users 
ADD COLUMN IF NOT EXISTS role VARCHAR(32) DEFAULT 'user';

-- Ensure role is one of: 'user', 'admin', 'founder'
ALTER TABLE users 
ADD CONSTRAINT check_user_role CHECK (role IN ('user', 'admin', 'founder'));

-- Create index for efficient role lookups
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role) WHERE role IN ('admin', 'founder');

-- Role descriptions:
-- 'user': Regular user (default)
-- 'admin': Platform administrator (can moderate, view analytics)
-- 'founder': Founder/owner with full dashboard access

-- To set a user as founder, run:
-- UPDATE users SET role = 'founder' WHERE email = 'founder@example.com';
