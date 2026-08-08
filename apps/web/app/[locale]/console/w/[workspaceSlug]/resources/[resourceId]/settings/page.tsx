import { ResourceSettingsView } from "@/features/observability/resource-settings/resource-settings-view";

export default async function ResourceSettingsPage({
  params,
}: {
  params: Promise<{ locale: string; workspaceSlug: string; resourceId: string }>;
}) {
  const { resourceId } = await params;
  return <ResourceSettingsView resourceId={resourceId} />;
}
