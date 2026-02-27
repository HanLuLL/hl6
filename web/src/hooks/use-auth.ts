import { useQuery } from "@tanstack/react-query";
import { api, buildApiUrl } from "@/lib/api";

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
    user: data?.data?.user ?? null,
    credits: data?.data?.credits ?? 0,
    signIn: () => { window.location.href = buildApiUrl("/auth/login"); },
    signOut: async () => {
      try {
        const res = await api.logout();
        const logoutUrl = res?.data?.logout_url;
        if (logoutUrl) {
          window.location.href = logoutUrl;
        } else {
          window.location.href = "/";
        }
      } catch {
        window.location.href = "/";
      }
    },
  };
}
