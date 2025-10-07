-- Migration: Create default admin user
-- Description: Creates an admin user with username 'admin' and assigns admin role

-- Insert admin user (password: 'admin' - CHANGE THIS IN PRODUCTION!)
-- Password hash for 'admin' using bcrypt (cost=10)
-- You can generate new hash with: echo -n "yourpassword" | htpasswd -niBC 10 "" | cut -d: -f2
INSERT INTO users (username, email, full_name, password_hash, is_active)
VALUES (
  'admin',
  'admin@inflight.local',
  'System Administrator',
  '$2a$10$HhQ6aDONAFaQwNSSBNdlleHnhYByrEgMlrjz5xO5M8WNa7DaWSAai', -- bcrypt hash for 'admin'
  true
)
ON CONFLICT (username) DO UPDATE
SET
  email = EXCLUDED.email,
  full_name = EXCLUDED.full_name,
  is_active = EXCLUDED.is_active;

-- Assign admin role to the admin user
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u, roles r
WHERE u.username = 'admin' AND r.name = 'admin'
ON CONFLICT (user_id, role_id) DO NOTHING;
