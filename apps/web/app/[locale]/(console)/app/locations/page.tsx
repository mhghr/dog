"use client";

import { useState } from "react";
import { MapPin } from "lucide-react";
import { useTranslations } from "next-intl";

import { EmptyState } from "@/components/common/empty-state";
import { ErrorState } from "@/components/common/error-state";
import { PageHeader } from "@/components/common/page-header";
import { RelativeTime } from "@/components/common/relative-time";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAgents, useAgentMutation } from "@/hooks/use-agents";
import { useLocations } from "@/hooks/use-locations";
import { useCreateLocation } from "@/hooks/use-location-mutations";
import { ApiError } from "@/lib/api-client";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

function AddLocationDialog({
  open,
  onOpenChange,
  tCommon,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tCommon: ReturnType<typeof useTranslations<"common">>;
}) {
  const t = useTranslations("locations");
  const tValidation = useTranslations("validation");

  const [name, setName] = useState("");
  const [code, setCode] = useState("");

  const createLocation = useCreateLocation();

  const handleSubmit = async () => {
    try {
      await createLocation.mutateAsync({
        name: name.trim(),
        code: code.trim().toLowerCase(),
      });
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
            <Input
              id="loc-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("namePlaceholder")}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="loc-code">{t("code")}</Label>
            <Input
              id="loc-code"
              dir="ltr"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder={t("codePlaceholder")}
            />
            <p className="text-xs text-muted-foreground">{t("codeHint")}</p>
          </div>
          <Button
            onClick={() => void handleSubmit()}
            disabled={
              createLocation.isPending || name.trim().length < 2 || code.trim().length < 2
            }
          >
            {tCommon("create")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function PendingAgentsSection() {
  const t = useTranslations("agents");
  const tCommon = useTranslations("common");
  const agentsQuery = useAgents("pending");
  const mutations = useAgentMutation();

  if (agentsQuery.isPending || agentsQuery.isError) {
    return null;
  }

  if (agentsQuery.data.items.length === 0) {
    return null;
  }

  const statusColors = "bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-400";

  return (
    <div className="mt-8">
      <h2 className="mb-4 text-lg font-semibold tracking-tight">
        {t("pendingAgents")}
      </h2>
      <div className="overflow-hidden rounded-xl border border-border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("name") ?? "Name"}</TableHead>
              <TableHead>{t("hostname")}</TableHead>
              <TableHead>{t("version")}</TableHead>
              <TableHead>{t("status")}</TableHead>
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
                <TableCell dir="ltr" className="font-mono text-xs">
                  {agent.version || "—"}
                </TableCell>
                <TableCell>
                  <Badge className={cn("pointer-events-none", statusColors)} variant="outline">
                    {agent.status}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-1">
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-7 text-xs"
                      disabled={mutations.approve.isPending || mutations.reject.isPending}
                      onClick={() =>
                        mutations.approve.mutateAsync(agent.id).then(
                          () => toast.success(t("approved")),
                          (err) => {
                            if (err instanceof ApiError) toast.error(err.message);
                          },
                        )
                      }
                    >
                      {t("approve")}
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      className="h-7 text-xs"
                      disabled={mutations.approve.isPending || mutations.reject.isPending}
                      onClick={() =>
                        mutations.reject.mutateAsync(agent.id).then(
                          () => toast.success(t("rejected")),
                          (err) => {
                            if (err instanceof ApiError) toast.error(err.message);
                          },
                        )
                      }
                    >
                      {t("reject")}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}

export default function LocationsPage() {
  const t = useTranslations("locations");
  const tCommon = useTranslations("common");

  const locationsQuery = useLocations();
  const [createOpen, setCreateOpen] = useState(false);

  return (
    <div>
      <AddLocationDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        tCommon={tCommon}
      />

      <PageHeader
        title={t("title")}
        subtitle={t("subtitle")}
        actions={
          <Button size="sm" onClick={() => setCreateOpen(true)}>
            {t("newLocation")}
          </Button>
        }
      />

      {locationsQuery.isPending ? (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <Skeleton className="h-[228px] w-full rounded-none" />
        </div>
      ) : locationsQuery.isError ? (
        <ErrorState onRetry={() => void locationsQuery.refetch()} />
      ) : locationsQuery.data.items.length === 0 ? (
        <EmptyState title={t("empty")} icon={MapPin} />
      ) : (
        <div className="overflow-hidden rounded-xl border border-border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("name")}</TableHead>
                <TableHead>{t("code")}</TableHead>
                <TableHead>{t("enabled")}</TableHead>
                <TableHead>{t("createdAt")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {locationsQuery.data.items.map((location) => (
                <TableRow key={location.id}>
                  <TableCell className="max-w-[200px] truncate font-medium">{location.name}</TableCell>
                  <TableCell dir="ltr" className="font-mono text-xs tabular-nums">
                    {location.code}
                  </TableCell>
                  <TableCell>
                    <Badge variant={location.enabled ? "secondary" : "outline"}>
                      {location.enabled ? tCommon("yes") : tCommon("no")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    <RelativeTime value={location.created_at} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <PendingAgentsSection />
    </div>
  );
}
