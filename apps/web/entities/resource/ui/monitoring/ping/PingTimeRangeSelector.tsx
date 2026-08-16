"use client";

import { useLocale } from "next-intl";

import { Button } from "@/shared/ui/button";
import { cn } from "@/shared/utils/cn";
import type { MetricsRange } from "@/entities/resource/hooks/use-resource";

export const PING_RANGES: MetricsRange[] = ["15m", "1h", "6h", "24h", "7d", "30d"];

const RANGE_LABEL: Record<MetricsRange, { en: string; fa: string }> = {
  "15m": { en: "15m", fa: "۱۵ دقیقه" },
  "1h": { en: "1h", fa: "۱ ساعت" },
  "6h": { en: "6h", fa: "۶ ساعت" },
  "24h": { en: "24h", fa: "۲۴ ساعت" },
  "7d": { en: "7d", fa: "۷ روز" },
  "30d": { en: "30d", fa: "۳۰ روز" },
};

export function PingTimeRangeSelector({
  range,
  onChange,
}: {
  range: MetricsRange;
  onChange: (range: MetricsRange) => void;
}) {
  const locale = useLocale();
  const isFa = locale === "fa";

  return (
    <div className="flex w-fit items-center gap-0.5">
      {PING_RANGES.map((r) => {
        const active = range === r;
        return (
          <Button
            key={r}
            type="button"
            variant="ghost"
            size="sm"
            className={cn(
              "h-7 rounded-full px-2.5 text-xs font-medium transition-colors",
              active
                ? "bg-muted text-foreground"
                : "text-muted-foreground hover:bg-transparent hover:text-foreground",
            )}
            onClick={() => onChange(r)}
          >
            {isFa ? RANGE_LABEL[r].fa : RANGE_LABEL[r].en}
          </Button>
        );
      })}
    </div>
  );
}
