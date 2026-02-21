import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";

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
  return useMutation({
    mutationFn: ({ id }: { id: number; fqdn: string }) => api.releaseSubdomain(id),
    onSuccess: (_data, variables) => {
      toast.success(t("subdomains.released", { fqdn: variables.fqdn }));
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
