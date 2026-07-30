DROP TRIGGER IF EXISTS trg_mon_agents_updated_at ON monitoring_agents;
DROP TABLE IF EXISTS monitoring_agent_updates;
DROP TABLE IF EXISTS monitoring_agent_configs;
DROP TABLE IF EXISTS monitoring_agent_heartbeats_default;
DROP TABLE IF EXISTS monitoring_agent_heartbeats;
DROP TABLE IF EXISTS monitoring_agents;
DROP TABLE IF EXISTS monitoring_bootstrap_tokens;
DROP FUNCTION IF EXISTS update_mon_agents_updated_at();
