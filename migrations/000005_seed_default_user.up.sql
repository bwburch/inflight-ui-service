-- Create default user for development/testing
INSERT INTO users (id, username, email, full_name, created_at)
VALUES (1, 'default', 'default@inflight.local', 'Default User', NOW())
ON CONFLICT (id) DO NOTHING;

-- Reset sequence to avoid conflicts
SELECT setval('users_id_seq', (SELECT MAX(id) FROM users));
