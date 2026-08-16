import { Skeleton } from "@/shared/ui/skeleton";

// SSR-safe loading shell for resource pages. Mirrors the final resource
// detail layout (toolbar, stat panels, title, tab bar, monitor cards) so the
// transition is skeleton → content of the same shape, not a layout change.
export default function ResourceDetailLoading() {
  return (
    <div className="space-y-6" aria-busy="true">
      <div className="space-y-4">
        <div className="flex justify-end">
          <Skeleton className="size-8 rounded-lg" />
        </div>
        <div className="flex flex-wrap items-start justify-between gap-6">
          <div className="grid shrink-0 grid-cols-2 gap-3 xl:grid-cols-4 xl:w-[36rem]">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="panel px-4 py-3">
                <Skeleton className="h-3 w-16" />
                <Skeleton className="mt-2 h-6 w-20" />
              </div>
            ))}
          </div>
          <div className="flex min-w-0 items-center gap-4">
            <Skeleton className="size-14 shrink-0 rounded-xl" />
            <div className="space-y-2">
              <Skeleton className="h-6 w-48" />
              <Skeleton className="h-4 w-64" />
            </div>
          </div>
        </div>
      </div>

      <div className="flex gap-1 rounded-[10px] bg-muted/70 p-1">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-9 w-28 rounded-lg" />
        ))}
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-36 rounded-xl" />
        ))}
      </div>
      <Skeleton className="h-72 rounded-xl" />
    </div>
  );
}
