package agents

import (
	"os"
	"strings"
	"testing"
)

func TestMigrationSQLValid(t *testing.T) {
	up, err := os.ReadFile("../../../migrations/000006_probe_agents.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(up) == 0 {
		t.Fatal("migration file is empty")
	}

	required := []string{
		"CREATE TYPE probe_agent_status",
		"CREATE TABLE probe_agents",
		"CREATE TABLE probe_agent_enrollment_tokens",
		"CREATE TABLE probe_agent_audit_log",
	}

	content := string(up)
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("missing '%s' in migration", r)
		}
	}
}

func TestMigrationDownSQLExists(t *testing.T) {
	down, err := os.ReadFile("../../../migrations/000006_probe_agents.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(down) == 0 {
		t.Fatal("down migration is empty")
	}
}
