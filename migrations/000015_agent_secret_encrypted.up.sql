ALTER TABLE monitoring_agents
    ADD COLUMN IF NOT EXISTS secret_encrypted TEXT;
