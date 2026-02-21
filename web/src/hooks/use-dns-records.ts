import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

export function useDNSRecords(subdomainId: number) {
  return useQuery({
    queryKey: ["dns-records", subdomainId],
    queryFn: async () => {
      const res = await api.listRecords(subdomainId);
      return res.data;
    },
    enabled: !!subdomainId,
  });
}

export function useCreateRecord(subdomainId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { type: string; content: string; ttl?: number; proxied?: boolean }) =>
      api.createRecord(subdomainId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["dns-records", subdomainId] });
      queryClient.invalidateQueries({ queryKey: ["subdomain", subdomainId] });
    },
  });
}

export function useUpdateRecord(subdomainId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ recordId, ...data }: { recordId: number; content: string; ttl?: number; proxied?: boolean }) =>
      api.updateRecord(subdomainId, recordId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["dns-records", subdomainId] });
    },
  });
}

export function useDeleteRecord(subdomainId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (recordId: number) => api.deleteRecord(subdomainId, recordId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["dns-records", subdomainId] });
      queryClient.invalidateQueries({ queryKey: ["subdomain", subdomainId] });
    },
  });
}
