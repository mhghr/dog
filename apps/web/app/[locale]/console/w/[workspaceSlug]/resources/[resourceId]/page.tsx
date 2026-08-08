import { ResourceDetailScreen } from "@/entities/resource/ui/resource-detail-screen";

export default async function ResourceDetailPage({
  params,
}: {
  params: Promise<{ locale: string; resourceId: string }>;
}) {
  const { resourceId } = await params;
  return <ResourceDetailScreen resourceId={resourceId} />;
}
