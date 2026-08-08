import type { AuthUser } from "@/shared/types/auth";

// Role-based UI rendering. Permissions gate *presentation* (buttons, menu
// items, sections); enforcement happens on the backend.
export type Role = "owner" | "admin" | "member" | "viewer";

export function can(role: Role | undefined, permission: string): boolean {
  if (!role) {
    return false;
  }
  switch (permission) {
    case "manage.workspace":
      return role === "owner" || role === "admin";
    case "manage.members":
      return role === "owner" || role === "admin";
    case "manage.monitors":
      return role !== "viewer";
    case "manage.resources":
      return role !== "viewer";
    case "manage.agents":
      return role === "owner" || role === "admin";
    case "view.observability":
      return true;
    default:
      return false;
  }
}

export function userRole(user: Pick<AuthUser, "roles"> | undefined): Role {
  const roles = user?.roles ?? [];
  if (roles.includes("owner")) return "owner";
  if (roles.includes("admin")) return "admin";
  if (roles.includes("viewer")) return "viewer";
  return "member";
}
