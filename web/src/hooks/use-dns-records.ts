import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";

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
  return useMutation({
    mutationFn: (data: { type: string; content: string; ttl?: number; proxied?: boolean }) =>
      api.createRecord(subdomainId, data),
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
}

export function useUpdateRecord(subdomainId: number) {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: ({ recordId, ...data }: { recordId: number; content: string; ttl?: number; proxied?: boolean }) =>
      api.updateRecord(subdomainId, recordId, data),
    onSuccess: () => {
      toast.success(t("recordForm.recordUpdated"));
      queryClient.invalidateQueries({ queryKey: ["dns-records", subdomainId] });
    },
    onError: (err) => {
      toast.error(getErrorMessage(err, t));
    },
  });
}

export function useDeleteRecord(subdomainId: number) {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (recordId: number) => api.deleteRecord(subdomainId, recordId),
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
}
