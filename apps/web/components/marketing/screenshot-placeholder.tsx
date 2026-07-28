import { cn } from "@/lib/utils";

export function ScreenshotPlaceholder({
  className,
  children,
}: {
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "relative mx-auto w-full max-w-6xl overflow-hidden rounded-2xl border-2 border-dashed border-border/60 bg-muted/30 p-8 shadow-lg shadow-zinc-950/10",
        className,
      )}
    >
      <div className="flex min-h-[320px] items-center justify-center">
        {children ?? (
          <span className="font-mono text-sm text-muted-foreground/60">
            Product Screenshot
          </span>
        )}
      </div>
    </div>
  );
}
