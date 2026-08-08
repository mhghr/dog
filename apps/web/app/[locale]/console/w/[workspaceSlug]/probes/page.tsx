"use client";

import { useMemo, useState } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { Check, Copy } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Skeleton } from "@/shared/ui/skeleton";
import { useAgents, useCreateEnrollmentToken, useUnusedTokens } from "@/entities/agent/hooks/use-agent";
import { useLocations } from "@/entities/probe/hooks/use-location";
import { ApiError } from "@/shared/api";
import type { ProbeAgent, UnusedToken } from "@/entities/agent/model/types";
import { ProbeTable } from "@/features/probe-management/ui/probe-table";
import { getAgentColumns } from "@/features/probe-management/lib/columns";
import { Badge } from "@/shared/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui/table";
import { cn } from "@/shared/utils/cn";

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

  const columns = useMemo(() => getAgentColumns(t), [t]);

  const agents = agentsQuery.data?.items ?? [];

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
    for (const agent of agents) {
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
  }, [agents, tokensQuery.data]);

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
      ) : agents.length === 0 && (tokensQuery.data?.items ?? []).length === 0 ? (
        <div className="flex h-[300px] items-center justify-center rounded-xl border border-border bg-card">
          <p className="text-sm text-muted-foreground">{t("empty")}</p>
        </div>
      ) : (
        <>
          <ProbeTable agents={agents} columns={columns} />

          {mergedRows.filter((r) => r.kind === "token").length > 0 && (
            <>
              <h3 className="mb-3 mt-6 text-sm font-semibold text-muted-foreground">توکن‌های استفاده نشده</h3>
              <div className="overflow-hidden rounded-xl border border-border bg-card">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>توکن</TableHead>
                      <TableHead>موقعیت</TableHead>
                      <TableHead>وضعیت</TableHead>
                      <TableHead>تاریخ ساخت</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {mergedRows.filter((r) => r.kind === "token").map((row) => (
                      <TableRow key={`token-${(row as { kind: "token"; token: UnusedToken }).token.id}`} className="bg-amber-50/50 dark:bg-amber-950/10">
                        <TableCell>
                          <span dir="ltr" className="font-mono text-xs text-amber-700 dark:text-amber-300">
                            {(row as { kind: "token"; token: UnusedToken }).token.token_label}
                          </span>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          <TokenLocationCell locationId={(row as { kind: "token"; token: UnusedToken }).token.location_id} />
                        </TableCell>
                        <TableCell>
                          <Badge className="pointer-events-none bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400" variant="outline">
                            در انتظار
                          </Badge>
                        </TableCell>
                        <TableCell className="text-muted-foreground">{(row as { kind: "token"; token: UnusedToken }).token.created_at}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </>
          )}
        </>
      )}
    </div>
  );
}

function TokenLocationCell({ locationId }: { locationId: string }) {
  const locationsQuery = useLocations();
  if (locationsQuery.isPending) return <span className="text-muted-foreground">…</span>;
  const location = locationsQuery.data?.items.find((l) => l.id === locationId);
  return <>{location?.name ?? locationId}</>;
}
