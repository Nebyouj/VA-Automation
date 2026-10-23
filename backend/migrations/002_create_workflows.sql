CREATE TABLE IF NOT EXISTS workflows (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS workflow_jobs (
    workflow_id INTEGER NOT NULL
        REFERENCES workflows(id) ON DELETE CASCADE,
    job_id INTEGER NOT NULL
        REFERENCES jobs(id) ON DELETE CASCADE,
    position INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (workflow_id, job_id),
    UNIQUE (workflow_id, position)
);

CREATE TABLE IF NOT EXISTS workflow_dependencies (
    workflow_id INTEGER NOT NULL,
    job_id INTEGER NOT NULL,
    depends_on_job_id INTEGER NOT NULL,

    PRIMARY KEY (workflow_id, job_id, depends_on_job_id),

    FOREIGN KEY (workflow_id, job_id)
        REFERENCES workflow_jobs(workflow_id, job_id)
        ON DELETE CASCADE,

    FOREIGN KEY (workflow_id, depends_on_job_id)
        REFERENCES workflow_jobs(workflow_id, job_id)
        ON DELETE CASCADE,

    CHECK (job_id <> depends_on_job_id)
);

CREATE INDEX IF NOT EXISTS idx_workflows_status
    ON workflows(status);

CREATE INDEX IF NOT EXISTS idx_workflow_dependencies_prerequisite
    ON workflow_dependencies(workflow_id, depends_on_job_id);
