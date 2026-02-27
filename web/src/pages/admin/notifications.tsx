import { useState, useEffect, useMemo, useRef } from "react";
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
import { TypeBadge } from "@/components/notification/type-badge";
import { NotificationDetailDialog } from "@/components/notification/notification-detail-dialog";
import {
  useAdminNotifications,
  useAdminCreateNotification,
  useAdminUpdateNotification,
  useAdminDeleteNotification,
} from "@/hooks/use-notifications";
import { useQuery } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import { toast } from "sonner";
import type { Notification } from "@/types";

function stripHTMLForCount(html: string): number {
  const tmp = document.createElement("div");
  tmp.innerHTML = html;
  return (tmp.textContent || "").length;
}

function collectLocalImageSources(html: string): string[] {
  const tmp = document.createElement("div");
  tmp.innerHTML = html;
  const localSources = new Set<string>();

  tmp.querySelectorAll("img[src]").forEach((node) => {
    const src = node.getAttribute("src");
    if (src?.startsWith("blob:")) {
      localSources.add(src);
    }
  });

  return Array.from(localSources);
}

function TargetBadge({ targetType }: { targetType: string }) {
  const { t } = useTranslation();
  return <Badge variant="outline">{t(`adminNotifications.target_${targetType}`)}</Badge>;
}

export function NotificationsContent() {
  const [page, setPage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<Notification | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [editTitle, setEditTitle] = useState("");
  const [editContent, setEditContent] = useState("");
  const [editType, setEditType] = useState("normal");
  const [editProgress, setEditProgress] = useState<number | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; title: string } | null>(null);
  const [detailNotification, setDetailNotification] = useState<Notification | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [publishProgress, setPublishProgress] = useState<number | null>(null);
  const { t } = useTranslation();
  const pendingImagesRef = useRef(new Map<string, File>());
  const pendingImageURLsRef = useRef(new Set<string>());

  const { data, isLoading } = useAdminNotifications(page);
  const createMutation = useAdminCreateNotification();
  const updateMutation = useAdminUpdateNotification();
  const deleteMutation = useAdminDeleteNotification();
  const isPublishing = publishProgress !== null || createMutation.isPending;
  const isEditPublishing = editProgress !== null || updateMutation.isPending;

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

  const clearPendingImages = () => {
    pendingImageURLsRef.current.forEach((url) => URL.revokeObjectURL(url));
    pendingImageURLsRef.current.clear();
    pendingImagesRef.current.clear();
  };

  const registerPendingImage = (file: File): string => {
    const localURL = URL.createObjectURL(file);
    pendingImagesRef.current.set(localURL, file);
    pendingImageURLsRef.current.add(localURL);
    return localURL;
  };

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedUserSearch(userSearch), 300);
    return () => clearTimeout(timer);
  }, [userSearch]);

  useEffect(() => {
    return () => {
      pendingImageURLsRef.current.forEach((url) => URL.revokeObjectURL(url));
      pendingImageURLsRef.current.clear();
      pendingImagesRef.current.clear();
    };
  }, []);

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
    setPublishProgress(null);
    clearPendingImages();
  };

  const handleCreate = async () => {
    if (isPublishing) return;

    const localImageSources = collectLocalImageSources(content);
    const totalSteps = localImageSources.length + 1;
    let completedSteps = 0;
    let contentWithUploadedImages = content;
    let createRequestStarted = false;
    setPublishProgress(0);

    try {
      for (const localSource of localImageSources) {
        const file = pendingImagesRef.current.get(localSource);
        if (!file) {
          throw new Error(t("adminNotifications.uploadFailed"));
        }
        const res = await api.adminUploadNotificationImage(file);
        const uploadedURL = res.data?.url;
        if (!uploadedURL) {
          throw new Error(t("adminNotifications.uploadFailed"));
        }
        contentWithUploadedImages = contentWithUploadedImages.split(localSource).join(uploadedURL);
        completedSteps += 1;
        setPublishProgress(Math.round((completedSteps / totalSteps) * 100));
      }

      const payload: Parameters<typeof createMutation.mutate>[0] = {
        title,
        content: contentWithUploadedImages,
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

      createRequestStarted = true;
      await createMutation.mutateAsync(payload);
      setPublishProgress(100);
      setCreateOpen(false);
      resetForm();
    } catch (err) {
      if (!createRequestStarted) {
        toast.error(getErrorMessage(err, t));
      }
      setPublishProgress(null);
    }
  };

  const canCreate =
    title.trim() &&
    content.trim() &&
    contentCharCount <= 1024 &&
    title.length <= 50 &&
    (targetType === "all" || (targetType === "users" && selectedUserIds.length > 0) || (targetType === "groups" && selectedGroupIds.length > 0));

  const editContentCharCount = useMemo(() => stripHTMLForCount(editContent), [editContent]);

  const canEdit =
    editTitle.trim() &&
    editContent.trim() &&
    editContentCharCount <= 1024 &&
    editTitle.length <= 50;

  const openEditDialog = (notif: Notification) => {
    setEditTarget(notif);
    setEditTitle(notif.title);
    setEditContent(notif.content);
    setEditType(notif.type);
    setEditProgress(null);
    clearPendingImages();
    setEditOpen(true);
  };

  const resetEditForm = () => {
    setEditTarget(null);
    setEditTitle("");
    setEditContent("");
    setEditType("normal");
    setEditProgress(null);
    clearPendingImages();
  };

  const handleEdit = async () => {
    if (!editTarget || isEditPublishing) return;

    const localImageSources = collectLocalImageSources(editContent);
    const totalSteps = localImageSources.length + 1;
    let completedSteps = 0;
    let contentWithUploadedImages = editContent;
    let updateRequestStarted = false;
    setEditProgress(0);

    try {
      for (const localSource of localImageSources) {
        const file = pendingImagesRef.current.get(localSource);
        if (!file) {
          throw new Error(t("adminNotifications.uploadFailed"));
        }
        const res = await api.adminUploadNotificationImage(file);
        const uploadedURL = res.data?.url;
        if (!uploadedURL) {
          throw new Error(t("adminNotifications.uploadFailed"));
        }
        contentWithUploadedImages = contentWithUploadedImages.split(localSource).join(uploadedURL);
        completedSteps += 1;
        setEditProgress(Math.round((completedSteps / totalSteps) * 100));
      }

      updateRequestStarted = true;
      await updateMutation.mutateAsync({
        id: editTarget.id,
        data: { title: editTitle, content: contentWithUploadedImages, type: editType },
      });
      setEditProgress(100);
      setEditOpen(false);
      resetEditForm();
    } catch (err) {
      if (!updateRequestStarted) {
        toast.error(getErrorMessage(err, t));
      }
      setEditProgress(null);
    }
  };

  return (
    <div className="space-y-6">
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
            <DialogContent className="max-w-lg">
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
                    onAddImage={registerPendingImage}
                    charCount={contentCharCount}
                    maxChars={1024}
                  />
                </div>

                {publishProgress !== null && (
                  <div className="space-y-1">
                    <div className="h-2 w-full overflow-hidden rounded bg-muted">
                      <div
                        className="h-full bg-primary transition-all duration-200"
                        style={{ width: `${publishProgress}%` }}
                      />
                    </div>
                    <p className="text-right text-xs text-muted-foreground">{publishProgress}%</p>
                  </div>
                )}

                <div className="flex justify-end gap-2">
                  <Button variant="outline" onClick={() => { setCreateOpen(false); resetForm(); }}>
                    {t("common.cancel")}
                  </Button>
                  <Button onClick={handleCreate} disabled={!canCreate || isPublishing}>
                    {isPublishing ? t("common.creating") : t("common.create")}
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
                <TableHead>{t("adminNotifications.readCount")}</TableHead>
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
                    <TableCell><Skeleton className="h-4 w-10" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-7 w-24" /></TableCell>
                  </TableRow>
                ))
              ) : (
                data?.data?.map((notif) => (
                  <TableRow key={notif.id} className="cursor-pointer" onClick={() => { setDetailNotification(notif); setDetailOpen(true); }}>
                    <TableCell className="text-sm font-medium max-w-[200px] truncate">{notif.title}</TableCell>
                    <TableCell><TypeBadge type={notif.type} /></TableCell>
                    <TableCell><TargetBadge targetType={notif.target_type} /></TableCell>
                    <TableCell className="text-sm text-muted-foreground">{notif.read_count ?? 0}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{notif.creator?.name ?? `#${notif.created_by}`}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">{new Date(notif.created_at).toLocaleString()}</TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => { e.stopPropagation(); openEditDialog(notif); }}
                        >
                          {t("adminNotifications.edit")}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive"
                          onClick={(e) => { e.stopPropagation(); setDeleteTarget({ id: notif.id, title: notif.title }); }}
                        >
                          {t("common.delete")}
                        </Button>
                      </div>
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

      <NotificationDetailDialog
        notification={detailNotification}
        open={detailOpen}
        onOpenChange={setDetailOpen}
        showMarkRead={false}
      />

      <Dialog open={editOpen} onOpenChange={(open) => { setEditOpen(open); if (!open) resetEditForm(); }}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("adminNotifications.editTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            {/* Notification type */}
            <div className="space-y-2">
              <Label>{t("adminNotifications.notificationType")}</Label>
              <Select value={editType} onValueChange={setEditType}>
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
                <span className={`text-xs ${editTitle.length > 50 ? "text-destructive" : "text-muted-foreground"}`}>
                  {editTitle.length} / 50
                </span>
              </div>
              <Input
                value={editTitle}
                onChange={(e) => setEditTitle(e.target.value)}
                maxLength={50}
              />
            </div>

            {/* Content (TipTap editor) */}
            <div className="space-y-2">
              <Label>{t("adminNotifications.contentLabel")}</Label>
              <NotificationEditor
                content={editContent}
                onChange={setEditContent}
                onAddImage={registerPendingImage}
                charCount={editContentCharCount}
                maxChars={1024}
              />
            </div>

            {editProgress !== null && (
              <div className="space-y-1">
                <div className="h-2 w-full overflow-hidden rounded bg-muted">
                  <div
                    className="h-full bg-primary transition-all duration-200"
                    style={{ width: `${editProgress}%` }}
                  />
                </div>
                <p className="text-right text-xs text-muted-foreground">{editProgress}%</p>
              </div>
            )}

            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => { setEditOpen(false); resetEditForm(); }}>
                {t("common.cancel")}
              </Button>
              <Button onClick={handleEdit} disabled={!canEdit || isEditPublishing}>
                {isEditPublishing ? t("common.saving") : t("common.save")}
              </Button>
            </div>
          </div>
          </DialogContent>
      </Dialog>
    </div>
  );
}

export default function AdminNotificationsPage() {
  const { t } = useTranslation();
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("adminNotifications.title")}</h1>
        <p className="text-muted-foreground">{t("adminNotifications.subtitle")}</p>
      </div>
      <NotificationsContent />
    </div>
  );
}
