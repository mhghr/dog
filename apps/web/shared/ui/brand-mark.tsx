import { Pulse } from "@/shared/ui/icons";

import { cn } from "@/shared/utils/cn";

export function BrandMark({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "grid size-6 shrink-0 place-items-center rounded-md bg-primary text-primary-foreground dark:shadow-[0_0_10px_-2px_var(--primary)_/_40%]",
        className,
      )}
      aria-hidden
    >
      <Pulse className="size-3.5" />
    </span>
  );
}
