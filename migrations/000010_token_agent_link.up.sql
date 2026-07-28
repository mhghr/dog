ALTER TABLE probe_agent_enrollment_tokens ADD COLUMN IF NOT EXISTS token_label VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE probe_agents ADD COLUMN IF NOT EXISTS enrollment_token_id UUID REFERENCES probe_agent_enrollment_tokens(id);
