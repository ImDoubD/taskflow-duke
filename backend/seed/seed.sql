-- Reference seed script.
-- The actual seeding is performed by the Go function in internal/database/seed.go,
-- which generates a fresh bcrypt hash at runtime and uses ON CONFLICT DO NOTHING.
--
-- This file documents what the seed creates so reviewers can verify the data model.

-- Test user  (password: password123)
INSERT INTO users (id, name, email, password)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'Test User',
    'test@example.com',
    '<bcrypt hash of "password123" — generated at runtime>'
) ON CONFLICT (email) DO NOTHING;

-- Demo project owned by the test user
INSERT INTO projects (id, name, description, owner_id)
VALUES (
    'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'Demo Project',
    'A sample project for testing',
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'
) ON CONFLICT (id) DO NOTHING;

-- Three tasks with different statuses
INSERT INTO tasks (id, title, description, status, priority, project_id, created_by, due_date)
VALUES
    ('c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a33',
     'Set up project structure',
     'Initialize the repository and tooling',
     'done', 'high',
     'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     '2026-04-10'),

    ('d0eebc99-9c0b-4ef8-bb6d-6bb9bd380a44',
     'Implement authentication',
     'Register and login endpoints with JWT',
     'in_progress', 'high',
     'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     '2026-04-20'),

    ('e0eebc99-9c0b-4ef8-bb6d-6bb9bd380a55',
     'Write API documentation',
     'Document all endpoints with examples',
     'todo', 'medium',
     'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
     '2026-04-30')

ON CONFLICT (id) DO NOTHING;
