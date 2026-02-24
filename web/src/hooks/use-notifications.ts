import { useQuery, useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { api, getErrorMessage } from "@/lib/api";

export function useNotifications() {
  return useInfiniteQuery({
    queryKey: ["notifications"],
    queryFn: async ({ pageParam = 0 }) => {
      const res = await api.listNotifications(pageParam, 20);
      return res;
    },
    getNextPageParam: (lastPage) => {
      const nextOffset = lastPage.offset + lastPage.limit;
      if (nextOffset >= lastPage.total) return undefined;
      return nextOffset;
    },
    initialPageParam: 0,
    staleTime: 30_000,
  });
}

export function useNotification(id: number) {
  return useQuery({
    queryKey: ["notification", id],
    queryFn: async () => {
      const res = await api.getNotification(id);
      return res.data;
    },
    enabled: !!id,
    staleTime: 30_000,
  });
}

export function useUnreadStatus() {
  return useQuery({
    queryKey: ["notifications-unread"],
    queryFn: async () => {
      const res = await api.getUnreadStatus();
      return res.data.has_unread;
    },
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

export function useMarkRead() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (id: number) => api.markNotificationRead(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
      queryClient.invalidateQueries({ queryKey: ["notifications-unread"] });
    },
    onError: (err) => {
      toast.error(getErrorMessage(err, t));
    },
  });
}

export function useAdminNotifications(page: number, perPage = 15) {
  return useQuery({
    queryKey: ["admin-notifications", page, perPage],
    queryFn: async () => {
      const res = await api.adminListNotifications(page, perPage);
      return { data: res.data, total: res.total, page: res.page, per_page: res.per_page };
    },
    staleTime: 30_000,
  });
}

export function useAdminCreateNotification() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (data: Parameters<typeof api.adminCreateNotification>[0]) =>
      api.adminCreateNotification(data),
    onSuccess: () => {
      toast.success(t("adminNotifications.created"));
      queryClient.invalidateQueries({ queryKey: ["admin-notifications"] });
    },
    onError: (err) => {
      toast.error(getErrorMessage(err, t));
    },
  });
}

export function useAdminDeleteNotification() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  return useMutation({
    mutationFn: (id: number) => api.adminDeleteNotification(id),
    onSuccess: () => {
      toast.success(t("adminNotifications.deleted"));
      queryClient.invalidateQueries({ queryKey: ["admin-notifications"] });
    },
    onError: (err) => {
      toast.error(getErrorMessage(err, t));
    },
  });
}
