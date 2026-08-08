import { cn } from "@/shared/utils/cn";

export function SectionHeader({
  eyebrow,
  title,
  subtitle,
}: {
  eyebrow: string;
  title: string;
  subtitle?: string;
}) {
  return (
    <div className="max-w-2xl">
      <p className="text-sm font-semibold text-primary">{eyebrow}</p>
      <h2 className="mt-2 text-balance text-2xl font-bold tracking-tight sm:text-3xl">
        {title}
      </h2>
      {subtitle ? (
        <p className="mt-3 text-pretty leading-relaxed text-muted-foreground sm:text-lg">
          {subtitle}
        </p>
      ) : null}
    </div>
  );
}

export function BentoTile({
  title,
  body,
  className,
  children,
}: {
  title: string;
  body: string;
  className?: string;
  children?: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex flex-col rounded-xl border border-border bg-card p-6",
        className,
      )}
    >
      <h3 className="font-semibold">{title}</h3>
      <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{body}</p>
      {children ? <div className="mt-6 flex flex-1 items-end">{children}</div> : null}
    </div>
  );
}

export function EyebrowLabel({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "font-mono text-caption font-medium tracking-wider text-muted-foreground",
        className,
      )}
    >
      {children}
    </span>
  );
}

export function StatusChip({
  status,
  upLabel = "UP",
  downLabel = "DOWN",
}: {
  status: "up" | "down";
  upLabel?: string;
  downLabel?: string;
}) {
  return (
    <span
      className={cn(
        "rounded-full px-2 py-0.5 font-mono text-caption-sm font-medium",
        status === "up"
          ? "bg-success/10 text-success"
          : "bg-destructive/10 text-destructive",
      )}
    >
      {status === "up" ? upLabel : downLabel}
    </span>
  );
}

export function MockCard({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      aria-hidden
      dir="ltr"
      className={cn(
        "overflow-hidden rounded-xl border border-border bg-card p-4 shadow-sm",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function MockListRow({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex items-center justify-between rounded-lg border border-border/70 bg-muted/40 px-3 py-2.5",
        className,
      )}
    >
      {children}
    </div>
  );
}
