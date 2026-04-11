import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { api, createIdempotencyKey, getErrorMessage, isRetryableMutationError } from "@/lib/api";

export function useDNSRecords(subdomainId: number) {
  return useQuery({
    queryKey: ["dns-records", subdomainId],
    queryFn: async () => {
      const res = await api.listRecords(subdomainId);
      return res.data;
    },
    enabled: !!subdomainId,
    staleTime: 30_000,
  });
}

export function useCreateRecord(subdomainId: number) {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const [isRetrying, setIsRetrying] = useState(false);
  const mutation = useMutation({
    mutationFn: async (data: { type: string; content: string; ttl?: number; proxied?: boolean }) => {
      const idempotencyKey = createIdempotencyKey();
      try {
        return await api.createRecord(subdomainId, data, { idempotencyKey, timeoutMs: 3000 });
      } catch (err) {
        if (!isRetryableMutationError(err)) {
          throw err;
        }
        setIsRetrying(true);
        return api.createRecord(subdomainId, data, { idempotencyKey, timeoutMs: 3000 });
      } finally {
        setIsRetrying(false);
      }
    },
    onSuccess: () => {
      toast.success(t("recordForm.recordCreated"));
      queryClient.invalidateQueries({ queryKey: ["dns-records", subdomainId] });
      queryClient.invalidateQueries({ queryKey: ["subdomain", subdomainId] });
      queryClient.invalidateQueries({ queryKey: ["subdomains"] });
    },
    onError: (err) => {
      toast.error(getErrorMessage(err, t));
    },
  });
  return { ...mutation, isRetrying };
}

export function useUpdateRecord(subdomainId: number) {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const [isRetrying, setIsRetrying] = useState(false);
  const mutation = useMutation({
    mutationFn: async ({ recordId, ...data }: { recordId: number; content: string; ttl?: number; proxied?: boolean }) => {
      const idempotencyKey = createIdempotencyKey();
      try {
        return await api.updateRecord(subdomainId, recordId, data, { idempotencyKey, timeoutMs: 3000 });
      } catch (err) {
        if (!isRetryableMutationError(err)) {
          throw err;
        }
        setIsRetrying(true);
        return api.updateRecord(subdomainId, recordId, data, { idempotencyKey, timeoutMs: 3000 });
      } finally {
        setIsRetrying(false);
      }
    },
    onSuccess: () => {
      toast.success(t("recordForm.recordUpdated"));
      queryClient.invalidateQueries({ queryKey: ["dns-records", subdomainId] });
    },
    onError: (err) => {
      toast.error(getErrorMessage(err, t));
    },
  });
  return { ...mutation, isRetrying };
}

export function useDeleteRecord(subdomainId: number) {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const [isRetrying, setIsRetrying] = useState(false);
  const mutation = useMutation({
    mutationFn: async (recordId: number) => {
      const idempotencyKey = createIdempotencyKey();
      try {
        return await api.deleteRecord(subdomainId, recordId, { idempotencyKey, timeoutMs: 3000 });
      } catch (err) {
        if (!isRetryableMutationError(err)) {
          throw err;
        }
        setIsRetrying(true);
        return api.deleteRecord(subdomainId, recordId, { idempotencyKey, timeoutMs: 3000 });
      } finally {
        setIsRetrying(false);
      }
    },
    onSuccess: () => {
      toast.success(t("recordTable.recordDeleted"));
      queryClient.invalidateQueries({ queryKey: ["dns-records", subdomainId] });
      queryClient.invalidateQueries({ queryKey: ["subdomain", subdomainId] });
      queryClient.invalidateQueries({ queryKey: ["subdomains"] });
    },
    onError: (err) => {
      toast.error(getErrorMessage(err, t));
    },
  });
  return { ...mutation, isRetrying };
}
