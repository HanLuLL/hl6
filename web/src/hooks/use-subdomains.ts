import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { ApiError, api, createIdempotencyKey, getErrorMessage, isRetryableMutationError } from "@/lib/api";

const DEFAULT_SUBDOMAIN_SETTINGS = {
  min_length: 1,
  max_length: 63,
};

export function useDomains() {
  return useQuery({
    queryKey: ["domains"],
    queryFn: async () => {
      const res = await api.listDomains();
      return res.data;
    },
    staleTime: 30_000,
  });
}

export function useSubdomains() {
  return useQuery({
    queryKey: ["subdomains"],
    queryFn: async () => {
      const res = await api.listSubdomains();
      return res.data;
    },
    staleTime: 30_000,
  });
}

export function useSubdomainSettings() {
  return useQuery({
    queryKey: ["subdomain-settings"],
    queryFn: async () => {
      const res = await api.getSubdomainSettings();
      return res.data;
    },
    staleTime: 30_000,
    initialData: DEFAULT_SUBDOMAIN_SETTINGS,
  });
}

export function useSubdomain(id: number) {
  return useQuery({
    queryKey: ["subdomain", id],
    queryFn: async () => {
      const res = await api.getSubdomain(id);
      return res.data;
    },
    enabled: !!id,
    staleTime: 30_000,
  });
}

export function useClaimSubdomain() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: api.claimSubdomain,
    onSuccess: (data) => {
      toast.success(t("claimDialog.success", { fqdn: data.data.fqdn }));
      queryClient.invalidateQueries({ queryKey: ["subdomains"] });
      queryClient.invalidateQueries({ queryKey: ["domains"] });
      queryClient.invalidateQueries({ queryKey: ["me"] });
      queryClient.invalidateQueries({ queryKey: ["credits"] });
    },
    onError: (err) => {
      toast.error(getErrorMessage(err, t));
    },
  });
}

export function useReleaseSubdomain() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const [isRetrying, setIsRetrying] = useState(false);
  const mutation = useMutation({
    mutationFn: async ({ id }: { id: number; fqdn: string }) => {
      const idempotencyKey = createIdempotencyKey();
      try {
        return await api.releaseSubdomain(id, { idempotencyKey, timeoutMs: 3000 });
      } catch (err) {
        if (!isRetryableMutationError(err)) {
          throw err;
        }
        setIsRetrying(true);
        toast(`${t("common.retry")}...`);
        return api.releaseSubdomain(id, { idempotencyKey, timeoutMs: 3000 });
      } finally {
        setIsRetrying(false);
      }
    },
    onSuccess: (_data, variables) => {
      toast.success(t("subdomains.released", { fqdn: variables.fqdn }));
      queryClient.invalidateQueries({ queryKey: ["subdomains"] });
      queryClient.invalidateQueries({ queryKey: ["domains"] });
      queryClient.invalidateQueries({ queryKey: ["me"] });
      queryClient.invalidateQueries({ queryKey: ["credits"] });
    },
    onError: (err) => {
      if (err instanceof ApiError && err.data && typeof err.data === "object" && "bulk_job_id" in err.data) {
        const jobID = (err.data as { bulk_job_id: number }).bulk_job_id;
        toast.error(`DNS 批量任务已排队（Job #${jobID}），请等待完成后重试释放`);
        return;
      }
      toast.error(getErrorMessage(err, t));
    },
  });
  return { ...mutation, isRetrying };
}
