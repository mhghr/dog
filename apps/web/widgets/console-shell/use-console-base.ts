"use client";

import { usePathname } from "@/i18n/navigation";

// useConsoleBase returns the workspace-scoped console prefix, e.g.
// "/console/w/acme". Navigation links inside the console must go through
// this so every link stays inside the active workspace.
export function useConsoleBase(): string {
  const pathname = usePathname();
  const match = pathname.match(/^\/console\/w\/([^/]+)/);
  return match ? `/console/w/${match[1]}` : "";
}

export function consoleHref(base: string, path: string): string {
  return `${base}${path}`;
}
