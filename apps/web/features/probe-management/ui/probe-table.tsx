import {
  Badge,
} from "@/shared/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/shared/ui/table";
import { cn } from "@/shared/utils/cn";
import type { ProbeAgent } from "@/entities/agent/model/types";

export type AgentColumnKey =
  | "name"
  | "hostname"
  | "location"
  | "publicIp"
  | "status"
  | "runningJobs"
  | "version"
  | "lastSeen";

export interface AgentColumn<T = unknown> {
  key: AgentColumnKey;
  header: string;
  cell: (agent: ProbeAgent, ctx?: T) => React.ReactNode;
  className?: string;
}

const STATUS_COLORS: Record<string, string> = {
  pending: "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400",
  approved: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400",
  active: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/30 dark:text-emerald-400",
  offline: "bg-muted text-muted-foreground",
  disabled: "bg-muted text-muted-foreground",
  rejected: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  revoked: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  draining: "bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-400",
  updating: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
};

export function ProbeTable<T>({
  agents,
  columns,
  ctx,
}: {
  agents: ProbeAgent[];
  columns: AgentColumn<T>[];
  ctx?: T;
}) {
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((col) => (
              <TableHead key={col.key}>{col.header}</TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {agents.map((agent) => (
            <TableRow key={agent.id}>
              {columns.map((col) => (
                <TableCell key={col.key} className={col.className}>
                  {col.cell(agent, ctx)}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

export function AgentStatusBadge({ status }: { status: string }) {
  return (
    <Badge
      className={cn("pointer-events-none", STATUS_COLORS[status] ?? "")}
      variant="outline"
    >
      {status}
    </Badge>
  );
}
