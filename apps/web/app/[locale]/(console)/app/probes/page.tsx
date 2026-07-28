"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAgents, useAgentMutation, useAgentStatusTransition, useCreateEnrollmentToken } from "@/hooks/use-agents";
import { useLocations } from "@/hooks/use-locations";
import { useCreateLocation } from "@/hooks/use-location-mutations";
import { ApiError } from "@/lib/api-client";
import { cn } from "@/lib/utils";
import type { AgentStatus } from "@/types/agent";
import { RelativeTime } from "@/components/common/relative-time";
import { Plus, MapPin } from "lucide-react";

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

function AgentActions({ agentId, status }: { agentId: string; status: AgentStatus }) {
  const t = useTranslations("agents");
  const mutations = useAgentMutation();
  const transitions = useAgentStatusTransition(status);

  const actionLabels: Record<string, string> = {
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
      if (error instanceof ApiError) toast.error(error.message);
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
          {actionLabels[action] ?? action}
        </Button>
      ))}
    </div>
  );
}

function LocationName({ locationId }: { locationId: string }) {
  const locationsQuery = useLocations();
  if (locationsQuery.isPending) return <span className="text-muted-foreground">…</span>;
  const location = locationsQuery.data?.items.find((l) => l.id === locationId);
  return <>{location?.name ?? locationId}</>;
}

function CreateLocationDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("locations");
  const tCommon = useTranslations("common");
  const tValidation = useTranslations("validation");
  const createLocation = useCreateLocation();
  const [name, setName] = useState("");
  const [code, setCode] = useState("");

  const handleCreate = async () => {
    try {
      await createLocation.mutateAsync({ name: name.trim(), code: code.trim().toLowerCase() });
      toast.success(t("createSuccess"));
      onOpenChange(false);
      setName("");
      setCode("");
    } catch (error) {
      if (error instanceof ApiError && error.code === "duplicate") {
        toast.error(t("codeTaken"));
      } else {
        toast.error(tValidation("genericError"));
      }
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("newLocation")}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="loc-name">{t("name")}</Label>
            <Input id="loc-name" value={name} onChange={(e) => setName(e.target.value)} placeholder={t("namePlaceholder")} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="loc-code">{t("code")}</Label>
            <Input id="loc-code" dir="ltr" value={code} onChange={(e) => setCode(e.target.value)} placeholder={t("codePlaceholder")} />
            <p className="text-xs text-muted-foreground">{t("codeHint")}</p>
          </div>
          <Button onClick={() => void handleCreate()} disabled={createLocation.isPending || name.trim().length < 2 || code.trim().length < 2}>
            {tCommon("create")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function TokenDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useTranslations("agents");
  const tValidation = useTranslations("validation");
  const tCommon = useTranslations("common");
  const createToken = useCreateEnrollmentToken();
  const [ttlMinutes, setTtlMinutes] = useState("60");
  const [token, setToken] = useState<string | null>(null);

  const handleSubmit = async () => {
    try {
      const result = await createToken.mutateAsync({
        location_code: "",
        ttl_minutes: parseInt(ttlMinutes, 10) || 60,
      });
      setToken(result.token);
      toast.success(t("tokenCreated"));
    } catch (error) {
      if (error instanceof ApiError) toast.error(error.message);
      else toast.error(tValidation("genericError"));
    }
  };

  const handleClose = () => {
    onOpenChange(false);
    setToken(null);
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
            <div dir="ltr" className="select-all break-all rounded-lg border border-border bg-muted px-3 py-2 font-mono text-xs text-emerald-600">
              {token}
            </div>
            <p className="text-xs text-muted-foreground">{t("tokenHint")}</p>
            <Button onClick={handleClose} variant="outline">{tCommon("close")}</Button>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="token-ttl">{t("ttlMinutes")}</Label>
              <Input id="token-ttl" type="number" dir="ltr" min={1} max={1440} value={ttlMinutes} onChange={(e) => setTtlMinutes(e.target.value)} />
            </div>
            <Button onClick={() => void handleSubmit()} disabled={createToken.isPending}>
              {tCommon("create")}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

export default function ProbesPage() {
  const t = useTranslations("agents");
  const tCommon = useTranslations("common");
  const agentsQuery = useAgents();
  const [tokenOpen, setTokenOpen] = useState(false);
  const [locationOpen, setLocationOpen] = useState(false);

  return (
    <div>
      <TokenDialog open={tokenOpen} onOpenChange={setTokenOpen} />
      <CreateLocationDialog open={locationOpen} onOpenChange={setLocationOpen} />

      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-lg font-semibold tracking-tight">{t("title")}</h1>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={() => setLocationOpen(true)}>
            <MapPin className="size-3.5" />
            {t("locationsTab")}
          </Button>
          <Button size="sm" onClick={() => setTokenOpen(true)}>
            <Plus className="size-3.5" />
            {t("newToken")}
          </Button>
        </div>
      </div>

      {agentsQuery.isPending ? (
        <Skeleton className="h-[300px] w-full rounded-xl" />
      ) : agentsQuery.isError ? (
        <div className="flex h-[300px] items-center justify-center rounded-xl border border-border bg-card">
          <p className="text-sm text-muted-foreground">{tCommon("errorTitle")}</p>
        </div>
      ) : agentsQuery.data.items.length === 0 ? (
        <div className="flex h-[300px] flex-col items-center justify-center gap-3 rounded-xl border border-border bg-card">
          <p className="text-sm text-muted-foreground">{t("empty")}</p>
          <Button size="sm" onClick={() => setTokenOpen(true)}>{t("newToken")}</Button>
        </div>
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
                  <TableCell className="max-w-[180px] truncate font-medium">{agent.name}</TableCell>
                  <TableCell dir="ltr" className="font-mono text-xs">{agent.hostname}</TableCell>
                  <TableCell><LocationName locationId={agent.location_id} /></TableCell>
                  <TableCell dir="ltr" className="font-mono text-xs">{agent.version || "—"}</TableCell>
                  <TableCell>
                    <Badge className={cn("pointer-events-none", STATUS_COLORS[agent.status] ?? "")} variant="outline">
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
