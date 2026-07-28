"use client";

import { useEffect, useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Check, Copy } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAgents, useAgentMutation, useAgentStatusTransition, useCreateEnrollmentToken, useUnusedTokens } from "@/hooks/use-agents";
import { useLocations } from "@/hooks/use-locations";
import { ApiError } from "@/lib/api-client";
import type { AgentStatus, ProbeAgent, UnusedToken } from "@/types/agent";
import { RelativeTime } from "@/components/common/relative-time";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";

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

function maskToken(token: string): string {
  if (token.length <= 16) return token;
  return token.slice(0, 8) + "..." + token.slice(-8);
}

function PublicIPCell({ agentId, publicIP }: { agentId: string; publicIP: string }) {
  const mutations = useAgentMutation();
  const [editing, setEditing] = useState(false);
  const [value, setValue] = useState(publicIP || "");

  const handleSave = () => {
    setEditing(false);
    const trimmed = value.trim();
    if (trimmed && trimmed !== (publicIP || "")) {
      mutations.updatePublicIP.mutate({ agentId, publicIP: trimmed });
    }
    if (!trimmed) setValue(publicIP || "");
  };

  if (editing) {
    return (
      <Input
        dir="ltr"
        className="h-7 w-36 font-mono text-xs"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onBlur={handleSave}
        onKeyDown={(e) => { if (e.key === "Enter") handleSave(); if (e.key === "Escape") { setEditing(false); setValue(publicIP || ""); } }}
        autoFocus
      />
    );
  }

  return (
    <button
      type="button"
      dir="ltr"
      className="font-mono text-xs text-muted-foreground hover:text-foreground cursor-text text-left"
      onClick={() => setEditing(true)}
      title="Click to edit public IP"
    >
      {publicIP || "—"}
    </button>
  );
}

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

type MergedRow =
  | { kind: "agent"; agent: ProbeAgent }
  | { kind: "token"; token: UnusedToken };

export default function ProbesPage() {
  const t = useTranslations("agents");
  const tCommon = useTranslations("common");
  const agentsQuery = useAgents();
  const tokensQuery = useUnusedTokens();
  const createToken = useCreateEnrollmentToken();
  const [lastToken, setLastToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const handleCreateToken = async () => {
    try {
      setLastToken(null);
      const result = await createToken.mutateAsync({
        location_code: "",
        ttl_minutes: 60,
      });
      setLastToken(result.token);
    } catch (error) {
      toast.error(error instanceof ApiError ? error.message : "خطا در ساخت توکن");
    }
  };

  const handleCopy = async () => {
    if (!lastToken) return;
    await navigator.clipboard.writeText(lastToken);
    setCopied(true);
    toast.success("توکن کپی شد");
    setTimeout(() => setCopied(false), 2000);
  };

  const mergedRows = useMemo<MergedRow[]>(() => {
    const rows: MergedRow[] = [];

    const agentIDs = new Set<string>();
    for (const agent of agentsQuery.data?.items ?? []) {
      rows.push({ kind: "agent", agent });
      agentIDs.add(agent.id);
    }

    for (const token of tokensQuery.data?.items ?? []) {
      rows.push({ kind: "token", token });
    }

    rows.sort((a, b) => {
      const aDate = a.kind === "agent" ? a.agent.created_at : a.token.created_at;
      const bDate = b.kind === "agent" ? b.agent.created_at : b.token.created_at;
      return bDate.localeCompare(aDate);
    });

    return rows;
  }, [agentsQuery.data, tokensQuery.data]);

  const isLoading = agentsQuery.isPending && tokensQuery.isPending;

  return (
    <div>
      <div className="mb-4 flex items-start gap-3">
        <Button size="sm" onClick={() => void handleCreateToken()} disabled={createToken.isPending}>
          {createToken.isPending ? (
            <span className="flex items-center gap-2">
              <span className="size-3 animate-spin rounded-full border-2 border-current border-t-transparent" />
              در حال ساخت...
            </span>
          ) : (
            "ساخت توکن"
          )}
        </Button>
      </div>

      {lastToken && (
        <div className="mb-4 rounded-xl border border-emerald-200 bg-emerald-50 p-4 dark:border-emerald-800 dark:bg-emerald-950/30">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <div dir="ltr" className="select-all break-all font-mono text-sm text-emerald-700 dark:text-emerald-300">
                {lastToken}
              </div>
              <p className="mt-1 text-xs text-emerald-600 dark:text-emerald-400">اعتبار: ۱ ساعت</p>
            </div>
            <Button size="icon" variant="ghost" onClick={handleCopy} className="shrink-0">
              {copied ? <Check className="size-4 text-emerald-600" /> : <Copy className="size-4" />}
            </Button>
          </div>
        </div>
      )}

      {isLoading ? (
        <Skeleton className="h-[300px] w-full rounded-xl" />
      ) : agentsQuery.isError ? (
        <div className="flex h-[300px] items-center justify-center rounded-xl border border-border bg-card">
          <p className="text-sm text-muted-foreground">{tCommon("errorTitle")}</p>
        </div>
      ) : mergedRows.length === 0 ? (
        <div className="flex h-[300px] items-center justify-center rounded-xl border border-border bg-card">
          <p className="text-sm text-muted-foreground">{t("empty")}</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("name")}</TableHead>
                <TableHead>{t("hostname")}</TableHead>
                <TableHead>آیپی</TableHead>
                <TableHead>توکن</TableHead>
                <TableHead>{t("location")}</TableHead>
                <TableHead>{t("version")}</TableHead>
                <TableHead>{t("status")}</TableHead>
                <TableHead>{t("lastSeen")}</TableHead>
                <TableHead>{tCommon("actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {mergedRows.map((row, idx) =>
                row.kind === "token" ? (
                  <TableRow key={`token-${row.token.id}`} className="bg-amber-50/50 dark:bg-amber-950/10">
                    <TableCell className="text-muted-foreground">—</TableCell>
                    <TableCell className="text-muted-foreground">—</TableCell>
                    <TableCell className="text-muted-foreground">—</TableCell>
                    <TableCell>
                      <span dir="ltr" className="font-mono text-xs text-amber-700 dark:text-amber-300">
                        {row.token.token_label}
                      </span>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      <LocationName locationId={row.token.location_id} />
                    </TableCell>
                    <TableCell className="text-muted-foreground">—</TableCell>
                    <TableCell>
                      <Badge className="pointer-events-none bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400" variant="outline">
                        در انتظار
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">—</TableCell>
                    <TableCell className="text-muted-foreground">—</TableCell>
                  </TableRow>
                ) : (
                  <TableRow key={`agent-${row.agent.id}`}>
                    <TableCell className="max-w-[180px] truncate font-medium">{row.agent.name}</TableCell>
                    <TableCell dir="ltr" className="font-mono text-xs">{row.agent.hostname}</TableCell>
                    <TableCell>
                      <PublicIPCell agentId={row.agent.id} publicIP={row.agent.public_ip || ""} />
                    </TableCell>
                    <TableCell>
                      <span className="text-xs text-muted-foreground">—</span>
                    </TableCell>
                    <TableCell><LocationName locationId={row.agent.location_id} /></TableCell>
                    <TableCell dir="ltr" className="font-mono text-xs">{row.agent.version || "—"}</TableCell>
                    <TableCell>
                      <Badge className={cn("pointer-events-none", STATUS_COLORS[row.agent.status] ?? "")} variant="outline">
                        {row.agent.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      <RelativeTime value={row.agent.last_seen_at} />
                    </TableCell>
                    <TableCell>
                      <AgentActions agentId={row.agent.id} status={row.agent.status} />
                    </TableCell>
                  </TableRow>
                )
              )}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
