import { ResourceMetricsView } from "@/features/observability/metrics/resource-metrics-view";

export default async function ResourceMetricsPage({
  params,
}: {
  params: Promise<{ locale: string; workspaceSlug: string; resourceId: string }>;
}) {
  const { resourceId } = await params;
  return <ResourceMetricsView resourceId={resourceId} />;
}
