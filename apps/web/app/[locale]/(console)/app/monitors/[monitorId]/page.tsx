"use client";

import { use } from "react";

import { MonitorDetailScreen } from "@/features/monitors/detail/monitor-detail-screen";

export default function MonitorDetailPage({
  params,
}: {
  params: Promise<{ monitorId: string }>;
}) {
  const { monitorId } = use(params);
  return <MonitorDetailScreen monitorId={monitorId} />;
}
