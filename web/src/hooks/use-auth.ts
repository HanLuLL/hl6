import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { api, buildApiUrl, getErrorMessage } from "@/lib/api";
import { isNativeClient } from "@/lib/client-runtime";
import { signOutNativeClient, startNativeSignIn } from "@/lib/native-client";

export function useAuth() {
  const { t } = useTranslation();
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
      if (isNativeClient) {
        void startNativeSignIn(ref).catch((err) => toast.error(getErrorMessage(err, t)));
        return;
      }
      const url = new URL(buildApiUrl("/auth/login"), window.location.origin);
      if (ref) url.searchParams.set("ref", ref);
      window.location.href = url.toString();
    },
    signOut: async () => {
      if (isNativeClient) {
        await signOutNativeClient();
        return;
      }
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
