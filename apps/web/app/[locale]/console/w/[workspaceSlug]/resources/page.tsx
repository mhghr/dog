import { dehydrate, HydrationBoundary } from "@tanstack/react-query";

import { createQueryClient } from "@/shared/data/query-client";
import { serverApiRequest } from "@/shared/api/server";
import { endpoints } from "@/shared/api/endpoints";
import { resourceListQueryString } from "@/entities/resource/hooks/resource-query";
import type { Resource, ResourceType } from "@/entities/resource/model/types";
import { ResourceList } from "@/entities/resource/ui/resource-list";

export default async function ResourcesPage() {
  // Server resolves the data the page renders before sending HTML, so the
  // initial grid is the final state — no client-side skeleton → content swap.
  const queryClient = createQueryClient();
  const listParams = resourceListQueryString({ page: 1, pageSize: 60 });

  await Promise.all([
    queryClient.prefetchQuery({
      queryKey: ["resources", "list", listParams],
      queryFn: () =>
        serverApiRequest<{ items: Resource[] }>(
          `${endpoints.resource.list}?${listParams}`,
          undefined,
          { refreshOn401: true },
        ),
      staleTime: 15_000,
    }),
    queryClient.prefetchQuery({
      queryKey: ["resource-types"],
      queryFn: () =>
        serverApiRequest<{ items: ResourceType[] }>(
          endpoints.resource.types,
          undefined,
          { refreshOn401: true },
        ),
      staleTime: 300_000,
    }),
  ]);

  return (
    <HydrationBoundary state={dehydrate(queryClient)}>
      <ResourceList />
    </HydrationBoundary>
  );
}
