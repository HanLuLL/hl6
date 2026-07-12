import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";
import type { AIModelConfigInput, PromptTemplateInput } from "@/types/ai-audit";

// ---- Public hooks ----

export function useAIAuditStats() {
  return useQuery({
    queryKey: ["ai-audit-stats"],
    queryFn: async () => {
      const res = await api.getAIAuditStats();
      return res.data;
    },
    staleTime: 30_000,
  });
}

export function useBanInfo() {
  return useQuery({
    queryKey: ["ban-info"],
    queryFn: async () => {
      const res = await api.getBanInfo();
      return res.data;
    },
    staleTime: 10_000,
  });
}

export function useMyAppeals() {
  return useQuery({
    queryKey: ["my-appeals"],
    queryFn: async () => {
      const res = await api.listMyAppeals();
      return res.data;
    },
    staleTime: 30_000,
  });
}

export function useCreateAppeal() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (content: string) => api.createAppeal(content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["my-appeals"] });
      toast.success(t("aiAudit.appealCreated"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

// ---- Admin hooks ----

export function useAdminAIModels() {
  return useQuery({
    queryKey: ["admin-ai-models"],
    queryFn: async () => {
      const res = await api.adminListAIModels();
      return res.data;
    },
    staleTime: 30_000,
  });
}

export function useAdminCreateAIModel() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (data: AIModelConfigInput) => api.adminCreateAIModel(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-ai-models"] });
      toast.success(t("aiAudit.modelCreated"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

export function useAdminUpdateAIModel() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<AIModelConfigInput> }) =>
      api.adminUpdateAIModel(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-ai-models"] });
      toast.success(t("aiAudit.modelUpdated"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

export function useAdminDeleteAIModel() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (id: number) => api.adminDeleteAIModel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-ai-models"] });
      toast.success(t("aiAudit.modelDeleted"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

export function useAdminPromptTemplates() {
  return useQuery({
    queryKey: ["admin-prompt-templates"],
    queryFn: async () => {
      const res = await api.adminListPromptTemplates();
      return res.data;
    },
    staleTime: 30_000,
  });
}

export function useAdminCreatePromptTemplate() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (data: PromptTemplateInput) => api.adminCreatePromptTemplate(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-prompt-templates"] });
      toast.success(t("aiAudit.templateCreated"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

export function useAdminUpdatePromptTemplate() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<PromptTemplateInput> }) =>
      api.adminUpdatePromptTemplate(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-prompt-templates"] });
      toast.success(t("aiAudit.templateUpdated"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

export function useAdminDeletePromptTemplate() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (id: number) => api.adminDeletePromptTemplate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-prompt-templates"] });
      toast.success(t("aiAudit.templateDeleted"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

export function useAdminAIReviews(params?: string) {
  return useQuery({
    queryKey: ["admin-ai-reviews", params],
    queryFn: async () => {
      const res = await api.adminListAIReviews(params);
      return res;
    },
    staleTime: 15_000,
  });
}

export function useAdminReviewAIReview() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: ({ id, status, note }: { id: number; status: string; note?: string }) =>
      api.adminReviewAIReview(id, status, note),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-ai-reviews"] });
      toast.success(t("aiAudit.reviewUpdated"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

export function useAdminAppeals(params?: string) {
  return useQuery({
    queryKey: ["admin-appeals", params],
    queryFn: async () => {
      const res = await api.adminListAppeals(params);
      return res;
    },
    staleTime: 15_000,
  });
}

export function useAdminReviewAppeal() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: ({ id, status, reply }: { id: number; status: string; reply?: string }) =>
      api.adminReviewAppeal(id, status, reply),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-appeals"] });
      toast.success(t("aiAudit.appealReviewed"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}
