package postgres

import (
	"strings"
	"testing"

	"monitoring-platform/packages/shared/domain"
)

func TestBuildResourceListFilter(t *testing.T) {
	t.Run("organization only", func(t *testing.T) {
		clause, args := buildResourceListFilter(domain.ResourceListFilter{OrganizationID: "org-1"})
		if !strings.Contains(clause, "r.organization_id = $1::uuid") {
			t.Errorf("expected org predicate in %q", clause)
		}
		if len(args) != 1 || args[0] != "org-1" {
			t.Errorf("unexpected args %v", args)
		}
	})

	t.Run("workspace and type", func(t *testing.T) {
		filter := domain.ResourceListFilter{
			OrganizationID: "org-1",
			WorkspaceID:    "ws-1",
			ResourceTypeID: "rt-1",
		}
		clause, args := buildResourceListFilter(filter)
		if !strings.Contains(clause, "r.workspace_id = $2::uuid") || !strings.Contains(clause, "r.resource_type_id = $3::uuid") {
			t.Errorf("unexpected clause %q", clause)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d", len(args))
		}
	})

	t.Run("status and search", func(t *testing.T) {
		filter := domain.ResourceListFilter{
			OrganizationID: "org-1",
			Status:         "active",
			Search:         "web",
		}
		clause, args := buildResourceListFilter(filter)
		if !strings.Contains(clause, "r.status = $2") {
			t.Errorf("expected status predicate in %q", clause)
		}
		if !strings.Contains(clause, "ILIKE $3") {
			t.Errorf("expected search predicate in %q", clause)
		}
		if len(args) != 3 {
			t.Errorf("expected 3 args, got %d", len(args))
		}
		if _, ok := args[2].(string); !ok || args[2] == "web" {
			t.Errorf("expected search arg to be LIKE-escaped, got %v", args[2])
		}
	})

	t.Run("tags", func(t *testing.T) {
		filter := domain.ResourceListFilter{
			OrganizationID: "org-1",
			Tags:           map[string]string{"env": "prod"},
		}
		clause, args := buildResourceListFilter(filter)
		if !strings.Contains(clause, "t.key = $2 AND t.value = $3") {
			t.Errorf("expected tag predicates in %q", clause)
		}
		if len(args) != 3 || args[1] != "env" || args[2] != "prod" {
			t.Errorf("unexpected args %v", args)
		}
	})
}
