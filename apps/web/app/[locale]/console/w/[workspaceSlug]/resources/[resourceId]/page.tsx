import { dehydrate, HydrationBoundary } from "@tanstack/react-query";
import { notFound } from "next/navigation";

import { createQueryClient } from "@/shared/data/query-client";
import { ResourceDetailScreen } from "@/entities/resource/ui/resource-detail-screen";
import {
  prefetchResourceDetail,
  ResourceNotFoundError,
} from "@/entities/resource/hooks/prefetch";

export default async function ResourceDetailPage({
  params,
}: {
  params: Promise<{
    locale: string;
    workspaceSlug: string;
    resourceId: string;
  }>;
}) {
  const { resourceId } = await params;

  // Server resolves the data the page layout depends on before rendering, so
  // the initial HTML (and the first client render after hydration) is the
  // final page state — no placeholder → data cascade.
  const queryClient = createQueryClient();

  try {
    await prefetchResourceDetail(queryClient, resourceId);
  } catch (error) {
    if (error instanceof ResourceNotFoundError) {
      notFound();
    }
    throw error;
  }

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <ResourceDetailScreen resourceId={resourceId} />
    </HydrationBoundary>
  );
}
