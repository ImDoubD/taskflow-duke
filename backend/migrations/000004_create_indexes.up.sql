-- Tasks: filtered by project
CREATE INDEX idx_tasks_project_id  ON tasks(project_id);

-- Tasks: filtered by assignee
CREATE INDEX idx_tasks_assignee_id ON tasks(assignee_id);

-- Tasks: filtered by status
CREATE INDEX idx_tasks_status ON tasks(status);

-- Projects: filtered by owner
CREATE INDEX idx_projects_owner_id ON projects(owner_id);

-- Note: users.email already has a unique constraint index — no extra index needed.
