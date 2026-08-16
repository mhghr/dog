import { setRequestLocale } from "next-intl/server";

import { redirect } from "@/i18n/navigation";
import { getAuth } from "@/platform/auth/server";
import { WorkspaceProvider } from "@/widgets/console-shell/workspace-provider";
import { ConsoleShell } from "@/widgets/console-shell/console-shell";

export default async function WorkspaceLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string; workspaceSlug: string }>;
}) {
  const { locale, workspaceSlug } = await params;
  setRequestLocale(locale);

  // Server-side gate: the console only renders for an authenticated request.
  // Unauthenticated users are redirected before any client hydration, so the
  // console is never rendered to a signed-out visitor.
  const { isSignedIn } = await getAuth();
  if (!isSignedIn) {
    redirect({ href: "/login", locale });
  }

  return (
    <WorkspaceProvider slug={workspaceSlug}>
      <ConsoleShell>{children}</ConsoleShell>
    </WorkspaceProvider>
  );
}
