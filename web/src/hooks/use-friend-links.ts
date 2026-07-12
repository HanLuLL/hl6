import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { FriendLinkInput } from "@/types";

// 前台公开友链列表
export function useFriendLinks() {
  return useQuery({
    queryKey: ["friend-links"],
    queryFn: async () => {
      const res = await api.getFriendLinks();
      return res.data;
    },
    staleTime: 60_000,
  });
}

// 后台全部友链列表
export function useAdminFriendLinks() {
  return useQuery({
    queryKey: ["admin-friend-links"],
    queryFn: async () => {
      const res = await api.adminListFriendLinks();
      return res.data;
    },
    staleTime: 30_000,
  });
}

export function useCreateFriendLink() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: FriendLinkInput) => api.adminCreateFriendLink(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-friend-links"] });
      queryClient.invalidateQueries({ queryKey: ["friend-links"] });
    },
  });
}

export function useUpdateFriendLink() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<FriendLinkInput> }) =>
      api.adminUpdateFriendLink(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-friend-links"] });
      queryClient.invalidateQueries({ queryKey: ["friend-links"] });
    },
  });
}

export function useDeleteFriendLink() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => api.adminDeleteFriendLink(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-friend-links"] });
      queryClient.invalidateQueries({ queryKey: ["friend-links"] });
    },
  });
}
