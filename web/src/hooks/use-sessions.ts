import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { toast } from "sonner";
import { useTranslation } from "react-i18next";

export function useSessions() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["sessions"],
    queryFn: () => api.getSessions(),
    staleTime: 60_000,
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.deleteSession(id),
    onSuccess: () => {
      toast.success(t("sessions.terminated"));
      queryClient.invalidateQueries({ queryKey: ["sessions"] });
    },
    onError: (err) => {
      toast.error(t("sessions.terminateFailed"));
      console.error("Failed to terminate session:", err);
    },
  });

  const logoutAllMutation = useMutation({
    mutationFn: () => api.logoutAll(),
    onSuccess: () => {
      toast.success(t("sessions.allTerminated"));
      queryClient.invalidateQueries({ queryKey: ["sessions"] });
      // 刷新当前页面，因为当前会话也会失效
      window.location.href = "/login";
    },
    onError: (err) => {
      toast.error(t("sessions.terminateAllFailed"));
      console.error("Failed to terminate all sessions:", err);
    },
  });

  return {
    sessions: data?.data ?? [],
    isLoading,
    error,
    terminateSession: deleteMutation.mutate,
    isTerminating: deleteMutation.isPending,
    terminateAll: logoutAllMutation.mutate,
    isTerminatingAll: logoutAllMutation.isPending,
  };
}
