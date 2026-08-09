"use client";

import { useEffect } from "react";
import { useRouter } from "@/i18n/navigation";
import { useWorkspaces } from "@/entities/workspace/hooks/use-workspace";

export default function DashboardRedirect() {
  const router = useRouter();
  const { data: workspacesData } = useWorkspaces();

  useEffect(() => {
    if (!workspacesData) return;
    const workspaces = workspacesData.items ?? [];
    if (workspaces.length > 0) {
      router.replace(`/console/w/${workspaces[0].slug}/dashboard`);
    } else {
      router.replace("/login");
    }
  }, [workspacesData, router]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p className="text-sm text-muted-foreground">Loading...</p>
    </div>
  );
}
