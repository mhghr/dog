import { MonitorDetailScreen } from "@/entities/monitor/ui/monitor-detail-screen";

interface PageProps {
  params: Promise<{ monitorId: string }>;
}

export default async function MonitorDetailPage({ params }: PageProps) {
  const { monitorId } = await params;
  return <MonitorDetailScreen monitorId={monitorId} />;
}
