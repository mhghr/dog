import { setRequestLocale } from "next-intl/server";

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

  return (
    <WorkspaceProvider slug={workspaceSlug}>
      <ConsoleShell>{children}</ConsoleShell>
    </WorkspaceProvider>
  );
}
