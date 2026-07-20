"use client";

import { useState } from "react";
import { ExternalLink, MoreHorizontal, Pencil, Trash2 } from "lucide-react";
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
import { useDeleteStatusPage } from "@/hooks/use-status-pages";
import { Link } from "@/i18n/navigation";
import type { StatusPage } from "@/types/status-page";

export function StatusPageActions({
  statusPage,
  onEdit,
}: {
  statusPage: StatusPage;
  onEdit: () => void;
}) {
  const t = useTranslations("statusPages");
  const tCommon = useTranslations("common");

  const [deleteOpen, setDeleteOpen] = useState(false);

  const deleteMutation = useDeleteStatusPage();

  const handleDelete = async () => {
    try {
      await deleteMutation.mutateAsync(statusPage.id);
      toast.success(t("deleteSuccess"));
      setDeleteOpen(false);
    } catch {
      // error already logged by mutation
    }
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            aria-label={`${tCommon("actions")}: ${statusPage.name}`}
          >
            <MoreHorizontal className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem asChild>
            <Link href={`/status/${statusPage.slug}`}>
              <ExternalLink className="size-4" aria-hidden />
              {t("viewPublic")}
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={() => onEdit()}>
            <Pencil className="size-4" aria-hidden />
            {tCommon("edit")}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            variant="destructive"
            onSelect={() => setDeleteOpen(true)}
          >
            <Trash2 className="size-4" aria-hidden />
            {tCommon("delete")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("deleteBody", { name: statusPage.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon("cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => {
                e.preventDefault();
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
