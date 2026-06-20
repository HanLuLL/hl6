import { toast } from "sonner";
import type { TFunction } from "i18next";
import { ApiError } from "@/lib/api";

export type DnsBulkJobAction = "ban" | "delete" | "release" | "operation";

const ERROR_KEYS: Record<DnsBulkJobAction, string> = {
  ban: "errors.dnsBulkJobQueuedBan",
  delete: "errors.dnsBulkJobQueuedDelete",
  release: "errors.dnsBulkJobQueuedRelease",
  operation: "errors.dnsBulkJobQueuedOperation",
};

export function toastDnsBulkJobQueued(t: TFunction, action: DnsBulkJobAction, jobID: number): void {
  toast.error(t(ERROR_KEYS[action], { jobID }));
}

export function isDnsBulkJobError(err: unknown): err is ApiError & { data: { bulk_job_id: number } } {
  return (
    err instanceof ApiError &&
    err.data != null &&
    typeof err.data === "object" &&
    "bulk_job_id" in err.data &&
    typeof (err.data as { bulk_job_id: unknown }).bulk_job_id === "number"
  );
}

export function handleDnsBulkJobError(
  err: unknown,
  t: TFunction,
  action: DnsBulkJobAction,
  onOtherError: (err: unknown) => void,
): void {
  if (isDnsBulkJobError(err)) {
    toastDnsBulkJobQueued(t, action, err.data.bulk_job_id);
    return;
  }
  onOtherError(err);
}
