-- Rollback: Remove admin user

-- Remove admin user
DELETE FROM users WHERE username = 'admin';
