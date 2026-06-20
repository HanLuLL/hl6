import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { ApiError, api, createIdempotencyKey, isRetryableMutationError } from "@/lib/api";
import { useErrorToast } from "@/hooks/use-error-toast";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";

export function ReleaseSubdomainDialog({
  subdomainId,
  open,
  onOpenChange,
  onSuccess,
}: {
  subdomainId: number | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}) {
  const { t } = useTranslation();
  const showError = useErrorToast();
  const queryClient = useQueryClient();
  const [sendNotify, setSendNotify] = useState(false);
  const [reason, setReason] = useState("");

  const releaseMutation = useMutation({
    mutationFn: async (opts: { notify: boolean; reason?: string }) => {
      const idempotencyKey = createIdempotencyKey();
      try {
        return await api.adminReleaseAuditSubdomain(subdomainId!, opts, { idempotencyKey, timeoutMs: 3000 });
      } catch (err) {
        if (isRetryableMutationError(err)) {
          return api.adminReleaseAuditSubdomain(subdomainId!, opts, { idempotencyKey, timeoutMs: 3000 });
        }
        throw err;
      }
    },
    onSuccess: () => {
      toast.success(t("audit.detail.released"));
      onOpenChange(false);
      setSendNotify(false);
      setReason("");
      queryClient.invalidateQueries({ queryKey: ["admin-audit-cases"] });
      queryClient.invalidateQueries({ queryKey: ["admin-audit-detail", subdomainId] });
      onSuccess?.();
    },
    onError: (err: unknown) => {
      if (err instanceof ApiError && err.data && typeof err.data === "object" && "bulk_job_id" in err.data) {
        toast.error(t("audit.detail.releaseQueued"));
        return;
      }
      showError(err);
    },
  });

  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setSendNotify(false);
      setReason("");
    }
    onOpenChange(next);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("audit.actions.release")}</DialogTitle>
          <DialogDescription>{t("audit.detail.releaseConfirm")}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={sendNotify} onChange={(e) => setSendNotify(e.target.checked)} />
            {t("audit.detail.sendNotify")}
          </label>
          {sendNotify && (
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder={t("audit.detail.reasonPlaceholder")}
            />
          )}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>{t("common.cancel")}</Button>
          <Button
            variant="destructive"
            disabled={subdomainId == null || releaseMutation.isPending}
            onClick={() => releaseMutation.mutate({ notify: sendNotify, reason: sendNotify ? reason : undefined })}
          >
            {t("audit.actions.release")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
