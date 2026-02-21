import { useLogto } from "@logto/react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { api, setTokenGetter } from "@/lib/api";

export function useAuth() {
  const { isAuthenticated, isLoading, signIn, signOut, getAccessToken, fetchUserInfo } = useLogto();
  const queryClient = useQueryClient();

  useEffect(() => {
    if (isAuthenticated) {
      const resource = import.meta.env.VITE_LOGTO_API_RESOURCE;
      setTokenGetter(async () => {
        const token = await getAccessToken(resource);
        return token ?? "";
      });
    }
  }, [isAuthenticated, getAccessToken]);

  const { data: meData, isLoading: isMeLoading } = useQuery({
    queryKey: ["me"],
    queryFn: () => api.getMe(),
    enabled: isAuthenticated,
  });

  const syncMutation = useMutation({
    mutationFn: api.syncUser,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["me"] }),
  });

  useEffect(() => {
    if (isAuthenticated && meData?.code === -1) {
      fetchUserInfo().then((info) => {
        if (!info) return;
        syncMutation.mutate({
          email: info.email ?? "",
          name: info.name ?? info.username ?? "",
          avatar_url: info.picture ?? "",
        });
      });
    }
  }, [isAuthenticated, meData]);

  return {
    isAuthenticated,
    isLoading: isLoading || (isAuthenticated && isMeLoading),
    user: meData?.data?.user ?? null,
    credits: meData?.data?.credits ?? 0,
    signIn: () => signIn(window.location.origin + "/callback"),
    signOut: () => signOut(window.location.origin),
    syncUser: syncMutation.mutate,
  };
}
