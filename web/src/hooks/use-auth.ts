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
    signIn: (ref?: string) => {
      const url = new URL(buildApiUrl("/auth/login"), window.location.origin);
      if (ref) url.searchParams.set("ref", ref);
      window.location.href = url.toString();
    },
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
