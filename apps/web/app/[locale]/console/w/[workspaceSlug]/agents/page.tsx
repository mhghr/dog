"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";

import { EmptyState } from "@/design-system/patterns/empty-state";
import { ErrorState } from "@/design-system/patterns/error-state";
import { PageHeader } from "@/design-system/patterns/page-header";
import { RelativeTime } from "@/shared/ui/relative-time";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog";
import { Input } from "@/shared/ui/input";
import { Label } from "@/shared/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/shared/ui/select";
import { Skeleton } from "@/shared/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui/table";
import { useAgents, useAgentMutation, useAgentStatusTransition, useCreateEnrollmentToken } from "@/entities/agent/hooks/use-agent";
import { useLocations } from "@/entities/probe/hooks/use-location";
import { ApiError } from "@/shared/api";
import { cn } from "@/shared/utils/cn";
import { MonitorCheck } from "lucide-react";
import { toast } from "sonner";
import type { AgentStatus } from "@/entities/agent/model/types";

const STATUS_COLORS: Record<AgentStatus, string> = {
  pending: "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400",
  approved: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400",
  active: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400",
  offline: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
  disabled: "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400",
  rejected: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  revoked: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  draining: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400",
  updating: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
};

function AgentTableSkeleton() {
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <Skeleton className="h-[228px] w-full rounded-none" />
    </div>
  );
}

function CreateTokenDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("agents");
  const tValidation = useTranslations("validation");
  const tCommon = useTranslations("common");

  const locationsQuery = useLocations();
  const [locationCode, setLocationCode] = useState("");
  const [ttlMinutes, setTtlMinutes] = useState("60");
  const [token, setToken] = useState<string | null>(null);

  const createToken = useCreateEnrollmentToken();

  const handleSubmit = async () => {
    try {
      const result = await createToken.mutateAsync({
        location_code: locationCode,
        ttl_minutes: parseInt(ttlMinutes, 10) || 60,
      });
      setToken(result.token);
      toast.success(t("tokenCreated"));
    } catch (error) {
      if (error instanceof ApiError) {
        toast.error(error.message);
      } else {
        toast.error(tValidation("genericError"));
      }
    }
  };

  const handleClose = () => {
    onOpenChange(false);
    setToken(null);
    setLocationCode("");
    setTtlMinutes("60");
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("newToken")}</DialogTitle>
        </DialogHeader>

        {token ? (
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label>{t("tokenCreated")}</Label>
              <div
                dir="ltr"
                className="select-all break-all rounded-lg border border-border bg-muted px-3 py-2 font-mono text-xs"
              >
                {token}
              </div>
            </div>
            <Button onClick={handleClose} variant="outline">
              {tCommon("close")}
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="token-location">{t("location")}</Label>
              <Select value={locationCode} onValueChange={setLocationCode}>
                <SelectTrigger id="token-location" className="w-full">
                  <SelectValue placeholder={t("locationCode")} />
                </SelectTrigger>
                <SelectContent>
                  {locationsQuery.data?.items
                    .filter((l) => l.enabled)
                    .map((loc) => (
                      <SelectItem key={loc.id} value={loc.code}>
                        {loc.name} ({loc.code})
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </div>

            <div className="flex flex-col gap-1.5">
              <Label htmlFor="token-ttl">{t("ttlMinutes")}</Label>
              <Input
                id="token-ttl"
                type="number"
                dir="ltr"
                min={1}
                max={1440}
                value={ttlMinutes}
                onChange={(e) => setTtlMinutes(e.target.value)}
              />
            </div>

            <Button
              onClick={() => void handleSubmit()}
              disabled={createToken.isPending || !locationCode}
            >
              {tCommon("create")}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function AgentActions({ agentId, status }: { agentId: string; status: AgentStatus }) {
  const t = useTranslations("agents");
  const mutations = useAgentMutation();

  const transitions = useAgentStatusTransition(status);

  const ACTION_LABELS: Record<string, string> = {
    approve: t("approve"),
    reject: t("reject"),
    disable: t("disable"),
    enable: t("enable"),
    revoke: t("revoke"),
    drain: t("drain"),
  };

  const handleAction = async (action: string) => {
    try {
      switch (action) {
        case "approve": await mutations.approve.mutateAsync(agentId); toast.success(t("approved")); break;
        case "reject": await mutations.reject.mutateAsync(agentId); toast.success(t("rejected")); break;
        case "disable": await mutations.disable.mutateAsync(agentId); toast.success(t("disabled")); break;
        case "enable": await mutations.enable.mutateAsync(agentId); toast.success(t("enabled")); break;
        case "revoke": await mutations.revoke.mutateAsync(agentId); toast.success(t("revoked")); break;
        case "drain": await mutations.drain.mutateAsync(agentId); toast.success(t("draining")); break;
      }
    } catch (error) {
      if (error instanceof ApiError) {
        toast.error(error.message);
      }
    }
  };

  const isPending = Object.values(mutations).some((m) => m.isPending);

  return (
    <div className="flex items-center gap-1">
      {transitions.map((action) => (
        <Button
          key={action}
          size="sm"
          variant={action === "reject" || action === "revoke" ? "destructive" : "outline"}
          disabled={isPending}
          onClick={() => void handleAction(action)}
          className="h-7 text-xs"
        >
          {ACTION_LABELS[action] ?? action}
        </Button>
      ))}
    </div>
  );
}

function LocationName({ locationId }: { locationId: string }) {
  const locationsQuery = useLocations();
  if (locationsQuery.isPending) {
    return <span className="text-muted-foreground">…</span>;
  }
  const location = locationsQuery.data?.items.find((l) => l.id === locationId);
  return <>{location?.name ?? locationId}</>;
}

export default function ProbeAgentsPage() {
  const t = useTranslations("agents");
  const tCommon = useTranslations("common");
  const agentsQuery = useAgents();
  const [createOpen, setCreateOpen] = useState(false);

  return (
    <div>
      <CreateTokenDialog open={createOpen} onOpenChange={setCreateOpen} />

      <PageHeader
        title={t("title")}
        subtitle={t("subtitle")}
        actions={
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            {t("newToken")}
          </Button>
        }
      />

      {agentsQuery.isPending ? (
        <AgentTableSkeleton />
      ) : agentsQuery.isError ? (
        <ErrorState onRetry={() => void agentsQuery.refetch()} />
      ) : agentsQuery.data.items.length === 0 ? (
        <EmptyState
          title={t("title")}
          description={t("subtitle")}
          icon={MonitorCheck}
        />
      ) : (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("name")}</TableHead>
                <TableHead>{t("hostname")}</TableHead>
                <TableHead>{t("location")}</TableHead>
                <TableHead>{t("version")}</TableHead>
                <TableHead>{t("status")}</TableHead>
                <TableHead>{t("lastSeen")}</TableHead>
                <TableHead>{tCommon("actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {agentsQuery.data.items.map((agent) => (
                <TableRow key={agent.id}>
                  <TableCell className="max-w-[180px] truncate font-medium">
                    {agent.name}
                  </TableCell>
                  <TableCell dir="ltr" className="font-mono text-xs">
                    {agent.hostname}
                  </TableCell>
                  <TableCell>
                    <LocationName locationId={agent.location_id} />
                  </TableCell>
                  <TableCell dir="ltr" className="font-mono text-xs">
                    {agent.version || "—"}
                  </TableCell>
                  <TableCell>
                    <Badge
                      className={cn(
                        "pointer-events-none",
                        STATUS_COLORS[agent.status] ?? "",
                      )}
                      variant="outline"
                    >
                      {agent.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    <RelativeTime value={agent.last_seen_at} />
                  </TableCell>
                  <TableCell>
                    <AgentActions agentId={agent.id} status={agent.status} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
