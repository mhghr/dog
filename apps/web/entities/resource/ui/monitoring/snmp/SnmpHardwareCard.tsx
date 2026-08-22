"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { cn } from "@/shared/utils/cn";
import type { SnmpSensorInfo } from "./snmp-metrics";

const SENSOR_ICON: Record<string, string> = {
  temperature: "🌡",
  fan: "🌀",
  power: "⚡",
};

function sensorTone(status: string | undefined): string {
  switch (status) {
    case "critical":
      return "bg-destructive/10 text-destructive border-destructive/25";
    case "warning":
      return "bg-warning/10 text-warning border-warning/30";
    case "ok":
      return "bg-emerald-500/10 text-emerald-500 border-emerald-500/25";
    default:
      return "bg-muted/40 text-muted-foreground border-border/40";
  }
}

export function SnmpHardwareCard({
  sensors,
  isFa,
}: {
  sensors: SnmpSensorInfo[];
  isFa: boolean;
}) {
  const t = (en: string, fa: string) => (isFa ? fa : en);
  const healthy = sensors.every((s) => s.status !== "critical");

  return (
    <Card variant="bordered" className="h-full shadow-subtle">
      <CardHeader className="px-5 pt-4">
        <CardTitle className="text-sm font-semibold text-foreground">
          {t("Hardware Health", "سلامت سخت‌افزار")}
        </CardTitle>
      </CardHeader>
      <CardContent className="px-4 pb-4 pt-1">
        {sensors.length === 0 ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            {t("No environmental sensors reported", "هیچ سنسور محیطی گزارش نشده است")}
          </p>
        ) : (
          <div className="flex flex-col gap-2">
            {sensors.map((sensor) => (
              <div
                key={sensor.name}
                className="flex items-center justify-between gap-2 rounded-lg border border-border/50 px-3 py-2"
              >
                <span className="flex min-w-0 items-center gap-2 text-sm text-foreground">
                  <span className="text-base leading-none">{SENSOR_ICON[sensor.sensor_type] ?? "•"}</span>
                  <span className="truncate" dir="auto">
                    {sensor.name}
                  </span>
                </span>
                <span className="flex shrink-0 items-center gap-2">
                  <span className="text-xs tabular-nums text-muted-foreground" dir="ltr">
                    {sensor.unit === "celsius" ? `${Math.round(sensor.value)}°C` : sensor.value}
                  </span>
                  <span
                    className={cn(
                      "rounded-full border px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide",
                      sensorTone(sensor.status),
                    )}
                  >
                    {sensor.status || "unknown"}
                  </span>
                </span>
              </div>
            ))}
            <p className="mt-1 text-xs text-muted-foreground">
              {healthy
                ? t("All hardware sensors nominal", "همه سنسورهای سخت‌افزار در وضعیت عادی هستند")
                : t("Hardware alarm — check device LEDs", "هشدار سخت‌افزار — وضعیت LED دستگاه را بررسی کنید")}
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
