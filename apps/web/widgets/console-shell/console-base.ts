// consoleBasePath derives the navigation base prefix from the active pathname.
// Workspace-scoped consoles live under /console/w/[slug]/...; everything else
// falls back to the classic /app surface. Kept dependency-free so it can be
// unit-tested in isolation.
export function consoleBasePath(pathname: string): string {
  const wsMatch = pathname.match(/^\/console\/w\/([^/]+)/);
  return wsMatch ? `/console/w/${wsMatch[1]}` : "/app";
}
