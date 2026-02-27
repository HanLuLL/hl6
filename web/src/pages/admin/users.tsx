import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import { toast } from "sonner";
import { Skeleton } from "@/components/ui/skeleton";
import { GroupsContent } from "./groups";
import { NotificationsContent } from "./notifications";

function UsersContent() {
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const queryClient = useQueryClient();
  const { t } = useTranslation();

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearch(search);
      setPage(1);
    }, 300);
    return () => clearTimeout(timer);
  }, [search]);

  const { data, isLoading } = useQuery({
    queryKey: ["admin-users", page, debouncedSearch],
    queryFn: async () => {
      const res = await api.adminListUsers(page, 50, debouncedSearch);
      return { users: res.data, total: res.total };
    },
    staleTime: 30_000,
  });

  const { data: groups } = useQuery({
    queryKey: ["admin-groups"],
    queryFn: async () => {
      const res = await api.adminListGroups();
      return res.data;
    },
    staleTime: 30_000,
  });

  const [grantUserId, setGrantUserId] = useState<number | null>(null);
  const [grantAmount, setGrantAmount] = useState("10");
  const [grantDesc, setGrantDesc] = useState("");

  const [changeGroupUserId, setChangeGroupUserId] = useState<number | null>(null);
  const [selectedGroupId, setSelectedGroupId] = useState<string>("");

  const grantMutation = useMutation({
    mutationFn: api.adminGrantCredits,
    onSuccess: (res) => {
      toast.success(t("adminUsers.grantSuccess", { amount: res.data.granted, balance: res.data.balance }));
      setGrantUserId(null);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const changeGroupMutation = useMutation({
    mutationFn: ({ userId, groupId }: { userId: number; groupId: number }) =>
      api.adminUpdateUserGroup(userId, groupId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-users"] });
      queryClient.invalidateQueries({ queryKey: ["admin-groups"] });
      toast.success(t("adminUsers.groupChanged"));
      setChangeGroupUserId(null);
      setSelectedGroupId("");
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          {isLoading ? (
            <Skeleton className="h-4 w-32" />
          ) : (
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t("adminUsers.totalUsers", { count: data?.total ?? 0 })}
            </CardTitle>
          )}
          <Input
            placeholder={t("adminUsers.searchPlaceholder")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="max-w-xs"
          />
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("adminUsers.name")}</TableHead>
                <TableHead>{t("adminUsers.email")}</TableHead>
                <TableHead>{t("adminUsers.group")}</TableHead>
                <TableHead>{t("adminUsers.role")}</TableHead>
                <TableHead>{t("adminUsers.joined")}</TableHead>
                <TableHead className="text-right">{t("adminUsers.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                [...Array(5)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-36" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-14 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell className="text-right"><Skeleton className="h-8 w-32 ml-auto" /></TableCell>
                  </TableRow>
                ))
              ) : (
                data?.users?.map((user) => (
                  <TableRow key={user.id}>
                    <TableCell className="font-medium">{user.name}</TableCell>
                    <TableCell className="text-muted-foreground">{user.email}</TableCell>
                    <TableCell>
                      <Badge variant="outline">{user.group?.name ?? "-"}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={user.role === "admin" ? "default" : "secondary"}>{user.role}</Badge>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(user.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell className="text-right space-x-1">
                      <Button variant="ghost" size="sm" onClick={() => {
                        setChangeGroupUserId(user.id);
                        setSelectedGroupId(user.group_id ? String(user.group_id) : "");
                      }}>
                        {t("adminUsers.changeGroup")}
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => setGrantUserId(user.id)}>
                        {t("adminUsers.grantCredits")}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {data && data.total > 50 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>{t("common.previous")}</Button>
          <span className="flex items-center text-sm text-muted-foreground">{t("common.pageOf", { page, total: Math.ceil(data.total / 50) })}</span>
          <Button variant="outline" size="sm" disabled={page >= Math.ceil(data.total / 50)} onClick={() => setPage((p) => p + 1)}>{t("common.next")}</Button>
        </div>
      )}

      {/* Grant Credits Dialog */}
      <Dialog open={grantUserId !== null} onOpenChange={(open) => !open && setGrantUserId(null)}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("adminUsers.grantCredits")}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("adminUsers.amount")}</Label>
              <Input type="number" min="0" step="any" value={grantAmount} onChange={(e) => setGrantAmount(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label>{t("adminUsers.descriptionOptional")}</Label>
              <Input value={grantDesc} onChange={(e) => setGrantDesc(e.target.value)} placeholder={t("adminUsers.adminGrant")} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setGrantUserId(null)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => grantMutation.mutate({
                user_id: grantUserId!,
                amount: parseFloat(grantAmount) || 1,
                description: grantDesc || t("adminUsers.adminGrant"),
              })}
              disabled={grantMutation.isPending}
            >
              {grantMutation.isPending ? t("adminUsers.granting") : t("credits.grant")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Change Group Dialog */}
      <Dialog open={changeGroupUserId !== null} onOpenChange={(open) => {
        if (!open) { setChangeGroupUserId(null); setSelectedGroupId(""); }
      }}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("adminUsers.changeGroup")}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("adminUsers.selectGroup")}</Label>
              <Select value={selectedGroupId} onValueChange={setSelectedGroupId}>
                <SelectTrigger>
                  <SelectValue placeholder={t("adminUsers.selectGroup")} />
                </SelectTrigger>
                <SelectContent>
                  {groups?.map((g) => (
                    <SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setChangeGroupUserId(null)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => changeGroupUserId && selectedGroupId && changeGroupMutation.mutate({
                userId: changeGroupUserId,
                groupId: parseInt(selectedGroupId),
              })}
              disabled={!selectedGroupId || changeGroupMutation.isPending}
            >
              {changeGroupMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export default function AdminUsersPage() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const currentTab = searchParams.get("tab") || "users";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("adminUsers.pageTitle")}</h1>
        <p className="text-muted-foreground">{t("adminUsers.pageSubtitle")}</p>
      </div>

      <Tabs
        value={currentTab}
        onValueChange={(value) => {
          if (value === "users") {
            setSearchParams({});
          } else {
            setSearchParams({ tab: value });
          }
        }}
      >
        <TabsList variant="line">
          <TabsTrigger value="users">{t("adminUsers.tabUsers")}</TabsTrigger>
          <TabsTrigger value="groups">{t("adminUsers.tabGroups")}</TabsTrigger>
          <TabsTrigger value="notifications">{t("adminUsers.tabNotifications")}</TabsTrigger>
        </TabsList>
        <TabsContent value="users" className="space-y-6 mt-4">
          <UsersContent />
        </TabsContent>
        <TabsContent value="groups" className="mt-4">
          <GroupsContent />
        </TabsContent>
        <TabsContent value="notifications" className="mt-4">
          <NotificationsContent />
        </TabsContent>
      </Tabs>
    </div>
  );
}
