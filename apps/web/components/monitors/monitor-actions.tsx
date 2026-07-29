"use client";

import { useState } from "react";
import { MoreHorizontal, Pause, Pencil, Play, Trash2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  useDeleteMonitor,
  usePauseMonitor,
  useResumeMonitor,
} from "@/hooks/use-monitor-mutations";
import { useRouter } from "@/i18n/navigation";
import type { Monitor } from "@/types/monitor";

export function MonitorActions({
  monitor,
  afterDeleteHref,
}: {
  monitor: Monitor;
  afterDeleteHref?: string;
}) {
  const t = useTranslations("monitors");
  const tCommon = useTranslations("common");
  const tValidation = useTranslations("validation");
  const router = useRouter();

  const [deleteOpen, setDeleteOpen] = useState(false);

  const pauseMutation = usePauseMonitor();
  const resumeMutation = useResumeMonitor();
  const deleteMutation = useDeleteMonitor();

  const handlePauseToggle = async () => {
    try {
      if (monitor.enabled) {
        await pauseMutation.mutateAsync(monitor.id);
        toast.success(tValidation("pauseSuccess"));
      } else {
        await resumeMutation.mutateAsync(monitor.id);
        toast.success(tValidation("resumeSuccess"));
      }
    } catch {
      toast.error(tValidation("genericError"));
    }
  };

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(monitor.id);
      toast.success(tValidation("deleteSuccess"));
      setDeleteOpen(false);
      if (afterDeleteHref) {
        router.push(afterDeleteHref);
      }
    } catch {
      toast.error(tValidation("genericError"));
    }
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            aria-label={`${tCommon("actions")}: ${monitor.name}`}
          >
            <MoreHorizontal className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem
            onSelect={() => void handlePauseToggle()}
            disabled={pauseMutation.isPending || resumeMutation.isPending}
          >
            {monitor.enabled ? (
              <>
                <Pause className="size-4" aria-hidden /> {t("pause")}
              </>
            ) : (
              <>
                <Play className="size-4" aria-hidden /> {t("resume")}
              </>
            )}
          </DropdownMenuItem>
          <DropdownMenuItem
            onSelect={() => router.push(`/app/nodes/${monitor.id}/edit`)}
          >
            <Pencil className="size-4" aria-hidden /> {tCommon("edit")}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            variant="destructive"
            onSelect={() => setDeleteOpen(true)}
          >
            <Trash2 className="size-4" aria-hidden /> {tCommon("delete")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("deleteBody", { name: monitor.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon("cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={(event) => {
                event.preventDefault();
                void handleDelete();
              }}
              disabled={deleteMutation.isPending}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              {tCommon("delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
