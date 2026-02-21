import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

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
  return useMutation({
    mutationFn: api.claimSubdomain,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["subdomains"] });
      queryClient.invalidateQueries({ queryKey: ["domains"] });
      queryClient.invalidateQueries({ queryKey: ["me"] });
      queryClient.invalidateQueries({ queryKey: ["credits"] });
    },
  });
}

export function useReleaseSubdomain() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: api.releaseSubdomain,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["subdomains"] });
      queryClient.invalidateQueries({ queryKey: ["domains"] });
      queryClient.invalidateQueries({ queryKey: ["me"] });
      queryClient.invalidateQueries({ queryKey: ["credits"] });
    },
  });
}
