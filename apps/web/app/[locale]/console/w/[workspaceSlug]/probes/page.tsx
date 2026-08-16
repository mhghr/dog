"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Check, Copy, MonitorCheck, Plus } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Skeleton } from "@/shared/ui/skeleton";
import { PageHeader } from "@/design-system/patterns/page-header";
import { EmptyState } from "@/design-system/patterns/empty-state";
import { ErrorState } from "@/design-system/patterns/error-state";
import { MetricCard } from "@/design-system/components/metric-card";
import { Badge } from "@/shared/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui/dialog";
import { Label } from "@/shared/ui/label";
import { ApiError, API_BASE_URL } from "@/shared/api";
import {
  useAgents,
  useAgentMutation,
  useAgentStatusTransition,
  useCreateEnrollmentToken,
  useUnusedTokens,
} from "@/entities/agent/hooks/use-agent";
import type {
  AgentStatus,
  ProbeAgent,
  UnusedToken,
} from "@/entities/agent/model/types";
import { ProbeTable, type AgentColumn } from "@/features/probe-management/ui/probe-table";
import { getAgentColumns, isAgentConnected } from "@/features/probe-management/lib/columns";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui/table";
import { Broadcast, Gauge, Pulse } from "@/shared/ui/icons";

const INSTALL_SCRIPT_URL =
  "https://raw.githubusercontent.com/mhghr/dog/main/scripts/install-agent.sh";

const OPERATIONAL_STATUSES = new Set(["active", "draining", "updating"]);

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
    delete: t("delete"),
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
        case "delete": await mutations.deleteAgent.mutateAsync(agentId); toast.success(t("deleted")); break;
      }
    } catch (error) {
      if (error instanceof ApiError) toast.error(error.message);
    }
  };

  const isPending = Object.values(mutations).some((m) => m.isPending);

  return (
    <div className="flex items-center justify-end gap-1">
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
      <Button
        size="sm"
        variant="destructive"
        disabled={isPending}
        onClick={() => void handleAction("delete")}
        className="h-7 text-xs"
      >
        {t("delete")}
      </Button>
    </div>
  );
}

function AddProbeDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (token: string) => void;
}) {
  const t = useTranslations("agents");
  const tCommon = useTranslations("common");
  const createToken = useCreateEnrollmentToken();
  const [token, setToken] = useState<string | null>(null);

  const handleGenerate = async () => {
    try {
      const result = await createToken.mutateAsync({
        location_code: "",
        ttl_minutes: 60,
      });
      setToken(result.token);
      onCreated(result.token);
    } catch (error) {
      toast.error(error instanceof ApiError ? error.message : t("tokenCreated"));
    }
  };

  const handleClose = () => {
    onOpenChange(false);
    setToken(null);
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("addProbe")}</DialogTitle>
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
            <p className="text-sm text-muted-foreground">{t("emptyDesc")}</p>
            <Button onClick={() => void handleGenerate()} disabled={createToken.isPending}>
              {createToken.isPending ? tCommon("creating") : t("newToken")}
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function ProbeStats({ agents }: { agents: ProbeAgent[] }) {
  const t = useTranslations("agents");
  const total = agents.length;
  const active = agents.filter((a) => OPERATIONAL_STATUSES.has(a.status)).length;
  const connected = agents.filter((a) => isAgentConnected(a)).length;
  const jobs = agents.reduce((sum, a) => sum + (a.running_jobs ?? 0), 0);

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      <MetricCard title={t("total")} value={total} status="unknown" icon={<Broadcast className="size-4" />} />
      <MetricCard
        title={t("active")}
        value={active}
        status={active > 0 ? "healthy" : "unknown"}
        icon={<MonitorCheck className="size-4" />}
      />
      <MetricCard
        title={t("connected")}
        value={connected}
        status={connected > 0 ? "healthy" : "unknown"}
        icon={<Pulse className="size-4" />}
      />
      <MetricCard title={t("runningJobs")} value={jobs} status="unknown" icon={<Gauge className="size-4" />} />
    </div>
  );
}

function InstallGuide({
  token,
  onCopyCommand,
  commandCopied,
}: {
  token: string | null;
  onCopyCommand: () => void;
  commandCopied: boolean;
}) {
  const t = useTranslations("agents");
  const command = `curl -fsSL ${INSTALL_SCRIPT_URL} | sudo bash -s -- ${token ?? "<TOKEN>"} ${API_BASE_URL}`;

  return (
    <div className="rounded-xl border border-border bg-card">
      <div className="border-b border-border px-5 py-4">
        <h2 className="text-sm font-semibold">{t("installTitle")}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">{t("installDesc")}</p>
      </div>
      <div className="flex flex-col gap-4 px-5 py-4">
        <ol className="flex flex-col gap-3 text-sm">
          <li className="flex items-start gap-3">
            <span className="grid size-6 shrink-0 place-items-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
              1
            </span>
            <span className="pt-0.5">{t("installStep1")}</span>
          </li>
          <li className="flex flex-col gap-2">
            <div className="flex items-start gap-3">
              <span className="grid size-6 shrink-0 place-items-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                2
              </span>
              <span className="pt-0.5">{t("installStep2")}</span>
            </div>
            <div className="flex items-center gap-2 ps-9">
              <code
                dir="ltr"
                className="flex-1 select-all break-all rounded-lg border border-border bg-muted px-3 py-2 font-mono text-xs text-foreground"
              >
                {command}
              </code>
              <Button size="icon" variant="outline" onClick={onCopyCommand} className="shrink-0">
                {commandCopied ? <Check className="size-4 text-emerald-600" /> : <Copy className="size-4" />}
              </Button>
            </div>
          </li>
          <li className="flex items-start gap-3">
            <span className="grid size-6 shrink-0 place-items-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
              3
            </span>
            <span className="pt-0.5">{t("installStep3")}</span>
          </li>
        </ol>
      </div>
    </div>
  );
}

export default function ProbesPage() {
  const t = useTranslations("agents");
  const tNav = useTranslations("navigation");
  const tCommon = useTranslations("common");
  const agentsQuery = useAgents();
  const tokensQuery = useUnusedTokens();
  const [lastToken, setLastToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [commandCopied, setCommandCopied] = useState(false);
  const [dialogOpen, setDialogOpen] = useState(false);

  const baseColumns = useMemo(() => getAgentColumns(t), [t]);
  const columns = useMemo<AgentColumn[]>(
    () => [
      ...baseColumns,
      {
        key: "lastSeen" as AgentColumn["key"],
        header: tCommon("actions"),
        cell: (agent: ProbeAgent) => (
          <AgentActions agentId={agent.id} status={agent.status} />
        ),
      },
    ],
    [baseColumns, tCommon],
  );

  const agents = agentsQuery.data?.items ?? [];

  const handleCopyToken = async () => {
    if (!lastToken) return;
    await navigator.clipboard.writeText(lastToken);
    setCopied(true);
    toast.success(t("commandCopied"));
    setTimeout(() => setCopied(false), 2000);
  };

  const handleCopyCommand = async () => {
    await navigator.clipboard.writeText(
      `curl -fsSL ${INSTALL_SCRIPT_URL} | sudo bash -s -- ${lastToken ?? "<TOKEN>"} ${API_BASE_URL}`,
    );
    setCommandCopied(true);
    toast.success(t("commandCopied"));
    setTimeout(() => setCommandCopied(false), 2000);
  };

  const isLoading = agentsQuery.isPending && tokensQuery.isPending;
  const tokenRows = tokensQuery.data?.items ?? [];

  return (
    <div className="flex flex-col gap-6">
      <AddProbeDialog open={dialogOpen} onOpenChange={setDialogOpen} onCreated={setLastToken} />

      <PageHeader
        title={tNav("probes")}
        subtitle={t("subtitle")}
        actions={
          <Button size="sm" onClick={() => setDialogOpen(true)}>
            <Plus className="size-4" />
            {t("addProbe")}
          </Button>
        }
      />

      {isLoading ? (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-24 rounded-xl" />
            ))}
          </div>
          <Skeleton className="h-[300px] w-full rounded-xl" />
        </div>
      ) : agentsQuery.isError ? (
        <ErrorState onRetry={() => void agentsQuery.refetch()} />
      ) : (
        <>
          <ProbeStats agents={agents} />

          <InstallGuide
            token={lastToken}
            onCopyCommand={handleCopyCommand}
            commandCopied={commandCopied}
          />

          {lastToken && (
            <div className="flex items-center justify-between gap-3 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 dark:border-emerald-800 dark:bg-emerald-950/30">
              <div className="min-w-0 flex-1">
                <div dir="ltr" className="select-all break-all font-mono text-sm text-emerald-700 dark:text-emerald-300">
                  {lastToken}
                </div>
                <p className="mt-0.5 text-xs text-emerald-600 dark:text-emerald-400">اعتبار: ۱ ساعت</p>
              </div>
              <Button size="icon" variant="ghost" onClick={handleCopyToken} className="shrink-0">
                {copied ? <Check className="size-4 text-emerald-600" /> : <Copy className="size-4" />}
              </Button>
            </div>
          )}

          {agents.length === 0 && tokenRows.length === 0 ? (
            <EmptyState title={tNav("probes")} description={t("emptyDesc")} icon={MonitorCheck} />
          ) : (
            <div className="flex flex-col gap-6">
              <ProbeTable agents={agents} columns={columns} />

              {tokenRows.length > 0 && (
                <section>
                  <h3 className="mb-3 text-sm font-semibold text-muted-foreground">توکن‌های استفاده نشده</h3>
                  <div className="overflow-hidden rounded-xl border border-border bg-card">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>توکن</TableHead>
                          <TableHead>وضعیت</TableHead>
                          <TableHead>تاریخ ساخت</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {tokenRows.map((token: UnusedToken) => (
                          <TableRow key={`token-${token.id}`} className="bg-amber-50/50 dark:bg-amber-950/10">
                            <TableCell>
                              <span dir="ltr" className="font-mono text-xs text-amber-700 dark:text-amber-300">
                                {token.token_label}
                              </span>
                            </TableCell>
                            <TableCell>
                              <Badge className="pointer-events-none bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400" variant="outline">
                                در انتظار
                              </Badge>
                            </TableCell>
                            <TableCell className="text-muted-foreground">{token.created_at}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                </section>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
