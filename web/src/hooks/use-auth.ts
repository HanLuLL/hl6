import { useQuery } from "@tanstack/react-query";
import { api, clearBrowserSessionToken, resetBannedSessionState } from "@/lib/api";
import { isNativeClient } from "@/lib/client-runtime";
import { signOutNativeClient } from "@/lib/native-client";

export function useAuth() {
  const { data, isLoading, error } = useQuery({
    queryKey: ["me"],
    queryFn: () => api.getMe(),
    staleTime: 30_000,
    retry: false,
  });

  const isAuthenticated = !error && !!data?.data?.user;

  return {
    isAuthenticated,
    isLoading,
    error,
    user: data?.data?.user ?? null,
    credits: data?.data?.credits ?? 0,
    signIn: (ref?: string) => {
      const query = ref ? `?ref=${encodeURIComponent(ref)}` : "";
      window.location.assign(`/login${query}`);
    },
    signOut: async () => {
      // 重置封禁跳转标志，确保下次登录若被封禁能正常跳转
      resetBannedSessionState();
      if (isNativeClient) {
        await signOutNativeClient();
        return;
      }
      // Browser mode: call logout API first, then clear local state
      // HttpOnly cookie will be cleared by the server
      try {
        const res = await api.logout();
        const logoutUrl = res?.data?.logout_url;
        // Clear any stale session markers
        clearBrowserSessionToken();
        sessionStorage.removeItem("hl6_401_count");
        sessionStorage.removeItem("hl6_401_time");
        sessionStorage.removeItem("hl6_kicked_out");
        if (logoutUrl) {
          window.location.href = logoutUrl;
        } else {
          window.location.href = "/";
        }
      } catch {
        // Even if API fails, clear local state and redirect
        clearBrowserSessionToken();
        window.location.href = "/";
      }
    },
  };
}
