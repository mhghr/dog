"use client";

import { useTranslations } from "next-intl";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type {
  NotificationChannel,
  NotificationPolicy,
  NotificationTrigger,
} from "@/types/health";

interface NotificationBuilderProps {
  policies: NotificationPolicy[];
  channels: NotificationChannel[];
  onAdd: () => void;
  onUpdate: (policy: NotificationPolicy) => void;
  onRemove: (policyId: string) => void;
}

const TRIGGER_OPTIONS: NotificationTrigger[] = [
  "STATUS_ENTERED_WARNING",
  "STATUS_ENTERED_ERROR",
  "STATUS_ENTERED_UNKNOWN",
  "RECOVERED_TO_OK",
  "DEGRADED_FROM_ERROR_TO_WARNING",
  "REPEATED_WARNING",
  "REPEATED_ERROR",
  "NO_DATA",
  "FLAPPING_DETECTED",
];

const TRIGGER_LABELS: Record<NotificationTrigger, string> = {
  STATUS_ENTERED_WARNING: "Enters Warning",
  STATUS_ENTERED_ERROR: "Enters Error",
  STATUS_ENTERED_UNKNOWN: "Enters Unknown",
  RECOVERED_TO_OK: "Recovers to OK",
  DEGRADED_FROM_ERROR_TO_WARNING: "Degrades from Error to Warning",
  REPEATED_WARNING: "Repeated Warning",
  REPEATED_ERROR: "Repeated Error",
  NO_DATA: "No Data",
  FLAPPING_DETECTED: "Flapping Detected",
};

export function NotificationBuilder({
  policies,
  channels,
  onAdd,
  onUpdate,
  onRemove,
}: NotificationBuilderProps) {
  const t = useTranslations("health");

  return (
    <div className="flex flex-col gap-4">
      {policies.map((policy) => {
        const channel = channels.find((c) => c.id === policy.channel_id);

        return (
          <div
            key={policy.id}
            className="rounded-lg border border-border p-3 space-y-3"
          >
            <div className="flex items-center justify-between">
              <Label className="text-sm font-medium">
                {channel?.name ?? policy.channel_id}
              </Label>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => policy.id && onRemove(policy.id)}
              >
                <Trash2 className="size-4" />
              </Button>
            </div>

            <div>
              <Label className="mb-2 block text-xs text-muted-foreground">
                {t("triggers")}
              </Label>
              <div className="grid grid-cols-2 gap-1.5">
                {TRIGGER_OPTIONS.map((trigger) => (
                  <label
                    key={trigger}
                    className="flex items-center gap-1.5 text-xs"
                  >
                    <Checkbox
                      checked={policy.triggers.includes(trigger)}
                      onCheckedChange={(checked) => {
                        const next = checked
                          ? [...policy.triggers, trigger]
                          : policy.triggers.filter((t) => t !== trigger);
                        onUpdate({ ...policy, triggers: next });
                      }}
                    />
                    {TRIGGER_LABELS[trigger]}
                  </label>
                ))}
              </div>
            </div>

            <div className="grid grid-cols-3 gap-3">
              <div className="flex flex-col gap-1">
                <Label className="text-xs text-muted-foreground">
                  {t("delay")} (s)
                </Label>
                <Input
                  type="number"
                  min={0}
                  value={policy.delay_seconds}
                  onChange={(e) =>
                    onUpdate({
                      ...policy,
                      delay_seconds: Number(e.target.value),
                    })
                  }
                />
              </div>
              <div className="flex flex-col gap-1">
                <Label className="text-xs text-muted-foreground">
                  {t("repeat")} (s)
                </Label>
                <Input
                  type="number"
                  min={0}
                  value={policy.repeat_interval_seconds}
                  onChange={(e) =>
                    onUpdate({
                      ...policy,
                      repeat_interval_seconds: Number(e.target.value),
                    })
                  }
                />
              </div>
              <div className="flex flex-col gap-1">
                <Label className="text-xs text-muted-foreground">
                  {t("cooldown")} (s)
                </Label>
                <Input
                  type="number"
                  min={0}
                  value={policy.cooldown_seconds}
                  onChange={(e) =>
                    onUpdate({
                      ...policy,
                      cooldown_seconds: Number(e.target.value),
                    })
                  }
                />
              </div>
            </div>
          </div>
        );
      })}

      <Button variant="outline" size="sm" onClick={onAdd} className="w-full">
        <Plus className="size-4" />
        {t("addChannel")}
      </Button>
    </div>
  );
}
