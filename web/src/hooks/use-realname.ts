import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import type {
  RealnameApplication,
  RealnameApplicationFull,
  RealnameStats,
  RealnameStatusResponse,
  SubmitRealnameResponse,
} from "@/types";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

// 用户当前实名状态缓存键
export const realnameStatusKey = ["realname", "status"] as const;
const realnameHistoryKey = (page: number, perPage: number) =>
  ["realname", "history", page, perPage] as const;

// 管理员审核列表缓存键
const adminRealnameListKey = (params: {
  page?: number;
  per_page?: number;
  status?: string;
  provider?: string;
  user_id?: number;
  from?: string;
  to?: string;
}) => ["admin", "realname", "applications", params] as const;

const adminRealnameStatsKey = ["admin", "realname", "stats"] as const;

// 获取当前用户实名状态
export function useRealnameStatus() {
  return useQuery({
    queryKey: realnameStatusKey,
    queryFn: async () => {
      const res = await api.getRealnameStatus();
      return res.data;
    },
    staleTime: 30_000,
  });
}

// 获取历史申请记录
export function useRealnameHistory(page = 1, perPage = 10) {
  return useQuery({
    queryKey: realnameHistoryKey(page, perPage),
    queryFn: async () => {
      const res = await api.getRealnameHistory(page, perPage);
      return res;
    },
    staleTime: 30_000,
  });
}

// 提交实名认证申请
export function useSubmitRealname() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: api.submitRealnameAuth,
    onSuccess: (res: { data: SubmitRealnameResponse }) => {
      queryClient.invalidateQueries({ queryKey: realnameStatusKey });
      queryClient.invalidateQueries({ queryKey: ["realname", "history"] });
      const data = res.data;
      // 需要付费时跳转支付链接
      if (data.need_pay && data.pay_url) {
        toast.success(t("realname.toast.redirectingToPay"));
        window.location.href = data.pay_url;
        return;
      }
      if (data.verified) {
        toast.success(t("realname.toast.verified"));
      } else {
        toast.info(data.message || t("realname.toast.submitted"));
      }
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

// 重试实名验证
export function useRetryRealname() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: api.retryRealnameVerification,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: realnameStatusKey });
      queryClient.invalidateQueries({ queryKey: ["realname", "history"] });
      toast.success(t("realname.toast.retryStarted"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

// 管理员：获取申请列表
export function useAdminRealnameApplications(params: {
  page?: number;
  per_page?: number;
  status?: string;
  provider?: string;
  user_id?: number;
  from?: string;
  to?: string;
}) {
  return useQuery({
    queryKey: adminRealnameListKey(params),
    queryFn: async () => {
      const res = await api.adminListRealnameApplications(params);
      return res;
    },
    staleTime: 15_000,
  });
}

// 管理员：获取统计
export function useAdminRealnameStats() {
  return useQuery({
    queryKey: adminRealnameStatsKey,
    queryFn: async () => {
      const res = await api.adminGetRealnameStats();
      return res.data;
    },
    staleTime: 30_000,
  });
}

// 管理员：审核申请
export function useAdminReviewRealname() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: ({ id, approved, reason }: { id: number; approved: boolean; reason: string }) =>
      api.adminReviewRealname(id, { approved, reason }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "realname"] });
      toast.success(t("realname.toast.reviewed"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

// 管理员：重试验证
export function useAdminRetryRealname() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (id: number) => api.adminRetryRealnameVerification(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin", "realname"] });
      toast.success(t("realname.toast.retryStarted"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

// 管理员：按需查看明文实名信息（每次调用后端强制写入审计日志）
export function useAdminViewRealnameFull() {
  const { t } = useTranslation();
  return useMutation({
    mutationFn: ({ id, reason }: { id: number; reason: string }) =>
      api.adminGetRealnameApplicationFull(id, { reason }),
    onSuccess: () => {
      toast.success(t("realname.toast.viewFullReady"));
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });
}

export type { RealnameApplication, RealnameApplicationFull, RealnameStats, RealnameStatusResponse };
