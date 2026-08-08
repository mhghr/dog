import { ResourceMonitorSection } from "@/features/observability/monitoring/resource-monitor-section";

export default async function ResourceMonitoringPage({
  params,
}: {
  params: Promise<{ locale: string; workspaceSlug: string; resourceId: string }>;
}) {
  const { resourceId } = await params;
  return <ResourceMonitorSection resourceId={resourceId} />;
}
