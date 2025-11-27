-- Script để tạo test user trong database
-- Chạy script này sau khi docker-compose đã start

-- Connect to PostgreSQL database
-- docker exec -it iaas-metadata-postgres psql -U iaas_user -d iaas_metadata

-- Tạo user admin với password đã hash (admin123)
INSERT INTO users (id, username, hash, email, created_at, updated_at)
VALUES (
    'test-user-123',
    'admin',
    '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', -- hash của 'admin123'
    'admin@iaas.local',
    NOW(),
    NOW()
) ON CONFLICT (username) DO NOTHING;

-- Tạo scope admin
INSERT INTO user_scopes (id, name, created_at, updated_at)
VALUES (
    'admin-scope',
    'admin',
    NOW(),
    NOW()
) ON CONFLICT (name) DO NOTHING;

-- Gán scope cho user
INSERT INTO user_scope_mapping (user_id, user_scope_id)
VALUES ('test-user-123', 'admin-scope')
ON CONFLICT DO NOTHING;

-- Verify
SELECT u.id, u.username, u.email, us.name as scope
FROM users u
LEFT JOIN user_scope_mapping usm ON u.id = usm.user_id
LEFT JOIN user_scopes us ON usm.user_scope_id = us.id
WHERE u.username = 'admin';

