import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

export function useReferrals(page = 1, perPage = 20) {
  return useQuery({
    queryKey: ["referrals", page, perPage],
    queryFn: async () => {
      const res = await api.getReferrals(page, perPage);
      return res.data;
    },
    staleTime: 30_000,
  });
}
