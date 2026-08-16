import { Skeleton } from "@/shared/ui/skeleton";

// SSR-safe loading shell for the resources list. Mirrors the final layout
// (search bar + 4-column card grid) so the transition is skeleton → content
// of the same shape.
export default function ResourcesLoading() {
  return (
    <div aria-busy="true">
      <div dir="ltr" className="mb-5 flex items-center gap-3">
        <div className="w-32" />
        <div className="mx-auto w-full max-w-sm">
          <Skeleton className="h-9 w-full rounded-lg" />
        </div>
        <div className="flex w-32 justify-end">
          <Skeleton className="h-9 w-24 rounded-lg" />
        </div>
      </div>

      <div className="mb-5 border-b border-border" />

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {Array.from({ length: 8 }).map((_, i) => (
          <div key={i} className="h-44 rounded-xl border border-border bg-card p-4">
            <Skeleton className="size-11 rounded-xl" />
            <Skeleton className="mt-3 h-4 w-2/3" />
            <Skeleton className="mt-2 h-3 w-1/2" />
          </div>
        ))}
      </div>
    </div>
  );
}
