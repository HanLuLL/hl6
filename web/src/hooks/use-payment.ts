import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

export function usePaymentProducts() {
  return useQuery({
    queryKey: ["payment-products"],
    queryFn: async () => {
      const res = await api.getPaymentProducts();
      return res.data;
    },
    staleTime: 60_000,
  });
}

export function usePaymentMethods() {
  return useQuery({
    queryKey: ["payment-methods"],
    queryFn: async () => {
      const res = await api.getPaymentMethods();
      return res.data;
    },
    staleTime: 60_000,
  });
}

export function usePaymentOrders() {
  return useQuery({
    queryKey: ["payment-orders"],
    queryFn: async () => {
      const res = await api.getPaymentOrders();
      return res.data;
    },
    staleTime: 30_000,
  });
}
