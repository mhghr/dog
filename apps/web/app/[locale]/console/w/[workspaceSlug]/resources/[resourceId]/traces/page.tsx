import { ResourceTracesView } from "@/features/observability/traces/resource-traces-view";

export default async function ResourceTracesPage({
  params,
}: {
  params: Promise<{ locale: string; workspaceSlug: string; resourceId: string }>;
}) {
  const { resourceId } = await params;
  return <ResourceTracesView resourceId={resourceId} />;
}
