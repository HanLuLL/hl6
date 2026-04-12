import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";

// ---- Query Hooks ----

export function useDomainMigrations(domainId: number, params?: { status?: string; page?: number; per_page?: number }) {
  return useQuery({
    queryKey: ["domain-migrations", domainId, params],
    queryFn: async () => (await api.adminListDomainMigrations(domainId, params)).data,
    enabled: domainId > 0,
    staleTime: 10_000,
  });
}

export function useDomainMigration(domainId: number, taskId: number, params?: { page?: number; per_page?: number }) {
  return useQuery({
    queryKey: ["domain-migration", domainId, taskId, params],
    queryFn: async () => (await api.adminGetDomainMigration(domainId, taskId, params)).data,
    enabled: domainId > 0 && taskId > 0,
    staleTime: 5_000,
  });
}

// ---- Mutation Hooks ----

export function useCreateMigration() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      domainId,
      data,
    }: {
      domainId: number;
      data: { target_provider_account_id: number; target_provider_zone_id: string; reason?: string };
    }) => api.adminCreateDomainMigration(domainId, data),
    onSuccess: (_result, { domainId }) => {
      queryClient.invalidateQueries({ queryKey: ["domain-migrations", domainId] });
      queryClient.invalidateQueries({ queryKey: ["admin-domains"] });
    },
  });
}

export function useRetryMigrationFailures() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ domainId, taskId }: { domainId: number; taskId: number }) =>
      api.adminRetryMigrationFailures(domainId, taskId),
    onSuccess: (_result, { domainId, taskId }) => {
      queryClient.invalidateQueries({ queryKey: ["domain-migration", domainId, taskId] });
      queryClient.invalidateQueries({ queryKey: ["domain-migrations", domainId] });
    },
  });
}

export function useCleanupMigrationSource() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({
      domainId,
      taskId,
      data,
    }: {
      domainId: number;
      taskId: number;
      data: { confirm_domain_name: string; confirm_phrase: string };
    }) => api.adminCleanupMigrationSource(domainId, taskId, data),
    onSuccess: (_result, { domainId, taskId }) => {
      queryClient.invalidateQueries({ queryKey: ["domain-migration", domainId, taskId] });
    },
  });
}
