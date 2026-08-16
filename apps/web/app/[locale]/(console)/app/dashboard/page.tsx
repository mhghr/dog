import { redirect } from "@/i18n/navigation";
import { getAuth } from "@/platform/auth/server";
import { serverApiRequest } from "@/shared/api/server";
import type { Workspace } from "@/entities/workspace/model/types";

// Server-side entry into the console. Resolves the authenticated user and the
// first workspace from the request and redirects immediately — no client-side
// "Loading..." screen, no Login → Console flash.
export default async function DashboardRedirect({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;

  const { isSignedIn } = await getAuth();
  if (!isSignedIn) {
    redirect({ href: "/login", locale });
  }

  let workspaces: Workspace[] = [];
  try {
    const data =
      await serverApiRequest<{ items: Workspace[] }>("/api/workspaces");
    workspaces = data.items ?? [];
  } catch {
    workspaces = [];
  }

  const first = workspaces[0];
  if (first) {
    redirect({ href: `/console/w/${first.slug}/dashboard`, locale });
  }

  redirect({ href: "/login", locale });
}
