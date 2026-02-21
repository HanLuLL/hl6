import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

export function useCredits() {
  return useQuery({
    queryKey: ["credits"],
    queryFn: async () => {
      const res = await api.getCredits();
      return res.data;
    },
  });
}

export function useTransactions(page = 1, perPage = 20) {
  return useQuery({
    queryKey: ["transactions", page, perPage],
    queryFn: async () => {
      const res = await api.listTransactions(page, perPage);
      return { data: res.data, total: res.total, page: res.page, perPage: res.per_page };
    },
  });
}
