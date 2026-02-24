import { useState, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { NotificationEditor } from "@/components/notification/notification-editor";
import {
  useAdminNotifications,
  useAdminCreateNotification,
  useAdminDeleteNotification,
} from "@/hooks/use-notifications";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

function stripHTMLForCount(html: string): number {
  const tmp = document.createElement("div");
  tmp.innerHTML = html;
  return (tmp.textContent || "").length;
}

function TypeBadge({ type }: { type: string }) {
  const { t } = useTranslation();
  const variants: Record<string, "destructive" | "default" | "secondary"> = {
    urgent: "destructive",
    pinned: "default",
    normal: "secondary",
  };
  return <Badge variant={variants[type] || "secondary"}>{t(`notifications.type_${type}`)}</Badge>;
}

function TargetBadge({ targetType }: { targetType: string }) {
  const { t } = useTranslation();
  return <Badge variant="outline">{t(`adminNotifications.target_${targetType}`)}</Badge>;
}

export default function AdminNotificationsPage() {
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; title: string } | null>(null);
  const { t } = useTranslation();

  const { data, isLoading } = useAdminNotifications(page);
  const createMutation = useAdminCreateNotification();
  const deleteMutation = useAdminDeleteNotification();

  // Create form state
  const [title, setTitle] = useState("");
  const [content, setContent] = useState("");
  const [type, setType] = useState("normal");
  const [targetType, setTargetType] = useState("all");
  const [selectedUserIds, setSelectedUserIds] = useState<number[]>([]);
  const [selectedGroupIds, setSelectedGroupIds] = useState<number[]>([]);
  const [visibleToNew, setVisibleToNew] = useState(false);
  const [userSearch, setUserSearch] = useState("");
  const [debouncedUserSearch, setDebouncedUserSearch] = useState("");

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedUserSearch(userSearch), 300);
    return () => clearTimeout(timer);
  }, [userSearch]);

  const contentCharCount = useMemo(() => stripHTMLForCount(content), [content]);

  // Load users for user selection
  const { data: usersData } = useQuery({
    queryKey: ["admin-users-search", debouncedUserSearch],
    queryFn: async () => {
      const res = await api.adminListUsers(1, 20, debouncedUserSearch);
      return res.data;
    },
    enabled: targetType === "users",
    staleTime: 30_000,
  });

  // Load groups for group selection
  const { data: groupsData } = useQuery({
    queryKey: ["admin-groups"],
    queryFn: async () => {
      const res = await api.adminListGroups();
      return res.data;
    },
    enabled: targetType === "groups",
    staleTime: 30_000,
  });

  const resetForm = () => {
    setTitle("");
    setContent("");
    setType("normal");
    setTargetType("all");
    setSelectedUserIds([]);
    setSelectedGroupIds([]);
    setVisibleToNew(false);
    setUserSearch("");
  };

  const handleCreate = () => {
    const payload: Parameters<typeof createMutation.mutate>[0] = {
      title,
      content,
      type,
      target_type: targetType,
    };

    if (targetType === "users") {
      payload.target_ids = selectedUserIds;
    } else if (targetType === "groups") {
      payload.target_ids = selectedGroupIds;
      payload.visible_to_new = visibleToNew;
    } else {
      payload.visible_to_new = visibleToNew;
    }

    createMutation.mutate(payload, {
      onSuccess: () => {
        setCreateOpen(false);
        resetForm();
      },
    });
  };

  const canCreate =
    title.trim() &&
    content.trim() &&
    contentCharCount <= 1024 &&
    title.length <= 50 &&
    (targetType === "all" || (targetType === "users" && selectedUserIds.length > 0) || (targetType === "groups" && selectedGroupIds.length > 0));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("adminNotifications.title")}</h1>
        <p className="text-muted-foreground">{t("adminNotifications.subtitle")}</p>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          {isLoading ? (
            <Skeleton className="h-4 w-28" />
          ) : (
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {t("adminNotifications.totalEntries", { count: data?.total ?? 0 })}
            </CardTitle>
          )}
          <Dialog open={createOpen} onOpenChange={(open) => { setCreateOpen(open); if (!open) resetForm(); }}>
            <DialogTrigger asChild>
              <Button size="sm">{t("adminNotifications.create")}</Button>
            </DialogTrigger>
            <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
              <DialogHeader>
                <DialogTitle>{t("adminNotifications.createTitle")}</DialogTitle>
              </DialogHeader>
              <div className="space-y-4">
                {/* Target Type */}
                <div className="space-y-2">
                  <Label>{t("adminNotifications.targetType")}</Label>
                  <Select value={targetType} onValueChange={(v) => { setTargetType(v); setSelectedUserIds([]); setSelectedGroupIds([]); }}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">{t("adminNotifications.target_all")}</SelectItem>
                      <SelectItem value="users">{t("adminNotifications.target_users")}</SelectItem>
                      <SelectItem value="groups">{t("adminNotifications.target_groups")}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {/* User selection */}
                {targetType === "users" && (
                  <div className="space-y-2">
                    <Label>{t("adminNotifications.selectUsers")}</Label>
                    <Input
                      placeholder={t("adminUsers.searchPlaceholder")}
                      value={userSearch}
                      onChange={(e) => setUserSearch(e.target.value)}
                    />
                    <div className="max-h-32 overflow-y-auto border rounded-md">
                      {usersData?.map((user) => (
                        <label
                          key={user.id}
                          className="flex items-center gap-2 px-3 py-1.5 hover:bg-accent cursor-pointer text-sm"
                        >
                          <input
                            type="checkbox"
                            checked={selectedUserIds.includes(user.id)}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedUserIds((prev) => [...prev, user.id]);
                              } else {
                                setSelectedUserIds((prev) => prev.filter((id) => id !== user.id));
                              }
                            }}
                          />
                          <span>{user.name}</span>
                          <span className="text-muted-foreground text-xs">{user.email}</span>
                        </label>
                      ))}
                    </div>
                    {selectedUserIds.length > 0 && (
                      <p className="text-xs text-muted-foreground">
                        {t("adminNotifications.selectedCount", { count: selectedUserIds.length })}
                      </p>
                    )}
                  </div>
                )}

                {/* Group selection */}
                {targetType === "groups" && (
                  <div className="space-y-2">
                    <Label>{t("adminNotifications.selectGroups")}</Label>
                    <div className="max-h-32 overflow-y-auto border rounded-md">
                      {groupsData?.map((group) => (
                        <label
                          key={group.id}
                          className="flex items-center gap-2 px-3 py-1.5 hover:bg-accent cursor-pointer text-sm"
                        >
                          <input
                            type="checkbox"
                            checked={selectedGroupIds.includes(group.id)}
                            onChange={(e) => {
                              if (e.target.checked) {
                                setSelectedGroupIds((prev) => [...prev, group.id]);
                              } else {
                                setSelectedGroupIds((prev) => prev.filter((id) => id !== group.id));
                              }
                            }}
                          />
                          <span>{group.name}</span>
                          {group.user_count !== undefined && (
                            <span className="text-muted-foreground text-xs">({group.user_count})</span>
                          )}
                        </label>
                      ))}
                    </div>
                  </div>
                )}

                {/* Visible to new users */}
                {(targetType === "all" || targetType === "groups") && (
                  <div className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="visibleToNew"
                      checked={visibleToNew}
                      onChange={(e) => setVisibleToNew(e.target.checked)}
                    />
                    <Label htmlFor="visibleToNew">{t("adminNotifications.visibleToNew")}</Label>
                  </div>
                )}

                {/* Notification type */}
                <div className="space-y-2">
                  <Label>{t("adminNotifications.notificationType")}</Label>
                  <Select value={type} onValueChange={setType}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="normal">{t("notifications.type_normal")}</SelectItem>
                      <SelectItem value="urgent">{t("notifications.type_urgent")}</SelectItem>
                      <SelectItem value="pinned">{t("notifications.type_pinned")}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>

                {/* Title */}
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label>{t("adminNotifications.titleLabel")}</Label>
                    <span className={`text-xs ${title.length > 50 ? "text-destructive" : "text-muted-foreground"}`}>
                      {title.length} / 50
                    </span>
                  </div>
                  <Input
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    maxLength={50}
                  />
                </div>

                {/* Content (TipTap editor) */}
                <div className="space-y-2">
                  <Label>{t("adminNotifications.contentLabel")}</Label>
                  <NotificationEditor
                    content={content}
                    onChange={setContent}
                    charCount={contentCharCount}
                    maxChars={1024}
                  />
                </div>

                <div className="flex justify-end gap-2">
                  <Button variant="outline" onClick={() => { setCreateOpen(false); resetForm(); }}>
                    {t("common.cancel")}
                  </Button>
                  <Button onClick={handleCreate} disabled={!canCreate || createMutation.isPending}>
                    {createMutation.isPending ? t("common.creating") : t("common.create")}
                  </Button>
                </div>
              </div>
            </DialogContent>
          </Dialog>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("adminNotifications.titleLabel")}</TableHead>
                <TableHead>{t("adminNotifications.notificationType")}</TableHead>
                <TableHead>{t("adminNotifications.targetType")}</TableHead>
                <TableHead>{t("adminNotifications.sender")}</TableHead>
                <TableHead>{t("auditLogs.time")}</TableHead>
                <TableHead>{t("adminNotifications.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                [...Array(5)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-7 w-14" /></TableCell>
                  </TableRow>
                ))
              ) : (
                data?.data?.map((notif) => (
                  <TableRow key={notif.id}>
                    <TableCell className="text-sm font-medium max-w-[200px] truncate">{notif.title}</TableCell>
                    <TableCell><TypeBadge type={notif.type} /></TableCell>
                    <TableCell><TargetBadge targetType={notif.target_type} /></TableCell>
                    <TableCell className="text-sm text-muted-foreground">{notif.creator?.name ?? `#${notif.created_by}`}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{new Date(notif.created_at).toLocaleString()}</TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive hover:text-destructive"
                        onClick={() => setDeleteTarget({ id: notif.id, title: notif.title })}
                      >
                        {t("common.delete")}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {data && data.total > 15 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>{t("common.previous")}</Button>
          <span className="flex items-center text-sm text-muted-foreground">{t("common.pageOf", { page, total: Math.ceil(data.total / 15) })}</span>
          <Button variant="outline" size="sm" disabled={page >= Math.ceil(data.total / 15)} onClick={() => setPage((p) => p + 1)}>{t("common.next")}</Button>
        </div>
      )}

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
        title={t("adminNotifications.deleteTitle")}
        description={t("adminNotifications.deleteConfirm", { title: deleteTarget?.title })}
        confirmText={t("common.delete")}
        onConfirm={() => {
          if (deleteTarget) {
            deleteMutation.mutate(deleteTarget.id, { onSuccess: () => setDeleteTarget(null) });
          }
        }}
        destructive
        loading={deleteMutation.isPending}
      />
    </div>
  );
}
