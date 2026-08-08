"use client";

import { useLocale } from "next-intl";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip";
import { formatDateTime, formatRelativeTime } from "@/shared/utils/formatters";

// RelativeTime shows a relative label with the absolute timestamp available
// on hover/focus, per the design requirements.
export function RelativeTime({ value }: { value: string | null | undefined }) {
  const locale = useLocale();

  if (!value) {
    return <span className="text-muted-foreground">—</span>;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0} className="cursor-default whitespace-nowrap">
          {formatRelativeTime(value, locale)}
        </span>
      </TooltipTrigger>
      <TooltipContent dir="ltr">{formatDateTime(value, locale)}</TooltipContent>
    </Tooltip>
  );
}
