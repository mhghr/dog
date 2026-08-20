"use client";

import { useState } from "react";
import { ChevronDown } from "lucide-react";

import { cn } from "@/shared/utils/cn";

interface SettingsSectionProps {
  title: string;
  /** rendered on the right of the title row (e.g. a badge) */
  action?: React.ReactNode;
  /** render the title as a collapse toggle (used for Advanced) */
  collapsible?: boolean;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

// A section inside the single monitoring settings card. Each section has its
// own header band (independent row) and content separated by divider lines.
export function SettingsSection({
  title,
  action,
  collapsible = false,
  defaultOpen = true,
  children,
}: SettingsSectionProps) {
  const [open, setOpen] = useState(defaultOpen);

  const titleRow = (
    <div className={cn("flex items-center gap-3", collapsible && "cursor-pointer select-none")}>
      <span className="flex min-w-0 items-center gap-2 text-[13px] font-semibold tracking-tight text-foreground">
        <span className="size-1.5 shrink-0 rounded-full bg-primary/70 shadow-[0_0_8px_1px_var(--primary)/40]" aria-hidden />
        <span className="truncate">{title}</span>
      </span>
      <span className="ms-1 min-w-0 flex-1" />
      {action}
      {collapsible && (
        <ChevronDown
          className={cn(
            "size-4 shrink-0 text-muted-foreground transition-transform duration-200",
            open ? "rotate-180" : "",
          )}
          aria-hidden
        />
      )}
    </div>
  );

  return (
    <section className="border-t border-border/50">
      <div className="bg-muted/15 px-7 py-2.5">
        {collapsible ? (
          <button
            type="button"
            onClick={() => setOpen((o) => !o)}
            aria-expanded={open}
            className="w-full text-start"
          >
            {titleRow}
          </button>
        ) : (
          titleRow
        )}
      </div>
      {open && <div className="px-7 py-4">{children}</div>}
    </section>
  );
}
