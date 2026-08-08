import { ResourceLogsView } from "@/features/observability/logs/resource-logs-view";

export default async function ResourceLogsPage({
  params,
}: {
  params: Promise<{ locale: string; workspaceSlug: string; resourceId: string }>;
}) {
  const { resourceId } = await params;
  return <ResourceLogsView resourceId={resourceId} />;
}
