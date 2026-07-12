import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "sonner";
import { getErrorMessage } from "@/lib/api";
import {
  useAdminFriendLinks,
  useCreateFriendLink,
  useUpdateFriendLink,
  useDeleteFriendLink,
} from "@/hooks/use-friend-links";
import type { FriendLink } from "@/types";

export function FriendLinksContent() {
  const { t } = useTranslation();
  const { data: links, isLoading } = useAdminFriendLinks();

  const createMutation = useCreateFriendLink();
  const updateMutation = useUpdateFriendLink();
  const deleteMutation = useDeleteFriendLink();

  const [showAdd, setShowAdd] = useState(false);
  const [addForm, setAddForm] = useState({
    name: "",
    url: "",
    description: "",
    logo_url: "",
    sort_order: 0,
    is_active: true,
  });

  const [editTarget, setEditTarget] = useState<FriendLink | null>(null);
  const [editForm, setEditForm] = useState({
    name: "",
    url: "",
    description: "",
    logo_url: "",
    sort_order: 0,
    is_active: true,
  });

  const [deleteTarget, setDeleteTarget] = useState<FriendLink | null>(null);

  const handleAdd = () => {
    if (!addForm.name.trim() || !addForm.url.trim()) {
      toast.error(t("friendLinks.nameUrlRequired"));
      return;
    }
    createMutation.mutate(
      {
        name: addForm.name.trim(),
        url: addForm.url.trim(),
        description: addForm.description.trim(),
        logo_url: addForm.logo_url.trim(),
        sort_order: addForm.sort_order,
        is_active: addForm.is_active,
      },
      {
        onSuccess: () => {
          toast.success(t("friendLinks.created"));
          setShowAdd(false);
          setAddForm({ name: "", url: "", description: "", logo_url: "", sort_order: 0, is_active: true });
        },
        onError: (err) => toast.error(getErrorMessage(err, t)),
      },
    );
  };

  const openEdit = (link: FriendLink) => {
    setEditTarget(link);
    setEditForm({
      name: link.name,
      url: link.url,
      description: link.description,
      logo_url: link.logo_url,
      sort_order: link.sort_order,
      is_active: link.is_active,
    });
  };

  const handleEdit = () => {
    if (!editTarget) return;
    if (!editForm.name.trim() || !editForm.url.trim()) {
      toast.error(t("friendLinks.nameUrlRequired"));
      return;
    }
    updateMutation.mutate(
      {
        id: editTarget.id,
        data: {
          name: editForm.name.trim(),
          url: editForm.url.trim(),
          description: editForm.description.trim(),
          logo_url: editForm.logo_url.trim(),
          sort_order: editForm.sort_order,
          is_active: editForm.is_active,
        },
      },
      {
        onSuccess: () => {
          toast.success(t("friendLinks.updated"));
          setEditTarget(null);
        },
        onError: (err) => toast.error(getErrorMessage(err, t)),
      },
    );
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => {
        toast.success(t("friendLinks.deleted"));
        setDeleteTarget(null);
      },
      onError: (err) => toast.error(getErrorMessage(err, t)),
    });
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex justify-end">
          <Skeleton className="h-9 w-24" />
        </div>
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("friendLinks.colName")}</TableHead>
                  <TableHead>{t("friendLinks.colUrl")}</TableHead>
                  <TableHead>{t("friendLinks.colSort")}</TableHead>
                  <TableHead>{t("friendLinks.colStatus")}</TableHead>
                  <TableHead className="text-right">{t("adminUsers.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(3)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-32" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-8" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-14 rounded-full" /></TableCell>
                    <TableCell className="text-right"><Skeleton className="h-8 w-28 ml-auto" /></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setShowAdd(true)}>{t("friendLinks.add")}</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("friendLinks.colName")}</TableHead>
                <TableHead>{t("friendLinks.colUrl")}</TableHead>
                <TableHead>{t("friendLinks.colSort")}</TableHead>
                <TableHead>{t("friendLinks.colStatus")}</TableHead>
                <TableHead className="text-right">{t("adminUsers.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {!links || links.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                    {t("friendLinks.noLinks")}
                  </TableCell>
                </TableRow>
              ) : (
                links.map((link) => (
                  <TableRow key={link.id}>
                    <TableCell className="font-medium">
                      <div className="flex items-center gap-2">
                        {link.logo_url ? (
                          <img
                            src={link.logo_url}
                            alt=""
                            className="h-6 w-6 rounded object-cover"
                            onError={(e) => ((e.target as HTMLImageElement).style.display = "none")}
                          />
                        ) : null}
                        {link.name}
                      </div>
                    </TableCell>
                    <TableCell className="text-muted-foreground max-w-xs truncate">{link.url}</TableCell>
                    <TableCell>{link.sort_order}</TableCell>
                    <TableCell>
                      <Badge variant={link.is_active ? "default" : "secondary"}>
                        {link.is_active ? t("common.active") : t("common.inactive")}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" onClick={() => openEdit(link)}>
                        {t("common.edit")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive"
                        onClick={() => setDeleteTarget(link)}
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

      {/* 新建对话框 */}
      <Dialog open={showAdd} onOpenChange={setShowAdd}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("friendLinks.addTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("friendLinks.colName")}</Label>
              <Input
                value={addForm.name}
                onChange={(e) => setAddForm({ ...addForm, name: e.target.value })}
                placeholder={t("friendLinks.namePlaceholder")}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("friendLinks.colUrl")}</Label>
              <Input
                value={addForm.url}
                onChange={(e) => setAddForm({ ...addForm, url: e.target.value })}
                placeholder="https://example.com"
              />
            </div>
            <div className="space-y-2">
              <Label>{t("friendLinks.colDesc")}</Label>
              <Textarea
                value={addForm.description}
                onChange={(e) => setAddForm({ ...addForm, description: e.target.value })}
                placeholder={t("friendLinks.descPlaceholder")}
                rows={2}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("friendLinks.colLogo")}</Label>
              <Input
                value={addForm.logo_url}
                onChange={(e) => setAddForm({ ...addForm, logo_url: e.target.value })}
                placeholder="https://example.com/logo.png"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("friendLinks.colSort")}</Label>
                <Input
                  type="number"
                  value={addForm.sort_order}
                  onChange={(e) => setAddForm({ ...addForm, sort_order: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div className="flex items-center justify-between pt-6">
                <Label>{t("friendLinks.colStatus")}</Label>
                <Switch
                  checked={addForm.is_active}
                  onCheckedChange={(v) => setAddForm({ ...addForm, is_active: v })}
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleAdd} disabled={createMutation.isPending}>
              {createMutation.isPending ? t("common.creating") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 编辑对话框 */}
      <Dialog open={!!editTarget} onOpenChange={(open) => !open && setEditTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("friendLinks.editTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("friendLinks.colName")}</Label>
              <Input
                value={editForm.name}
                onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("friendLinks.colUrl")}</Label>
              <Input
                value={editForm.url}
                onChange={(e) => setEditForm({ ...editForm, url: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("friendLinks.colDesc")}</Label>
              <Textarea
                value={editForm.description}
                onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                rows={2}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("friendLinks.colLogo")}</Label>
              <Input
                value={editForm.logo_url}
                onChange={(e) => setEditForm({ ...editForm, logo_url: e.target.value })}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("friendLinks.colSort")}</Label>
                <Input
                  type="number"
                  value={editForm.sort_order}
                  onChange={(e) => setEditForm({ ...editForm, sort_order: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div className="flex items-center justify-between pt-6">
                <Label>{t("friendLinks.colStatus")}</Label>
                <Switch
                  checked={editForm.is_active}
                  onCheckedChange={(v) => setEditForm({ ...editForm, is_active: v })}
                />
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTarget(null)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleEdit} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认 */}
      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("friendLinks.deleteTitle")}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            {t("friendLinks.deleteConfirm", { name: deleteTarget?.name ?? "" })}
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleteMutation.isPending}>
              {deleteMutation.isPending ? t("common.deleting") : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export default function AdminFriendLinksPage() {
  const { t } = useTranslation();
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("friendLinks.adminTitle")}</h1>
        <p className="text-muted-foreground">{t("friendLinks.adminSubtitle")}</p>
      </div>
      <FriendLinksContent />
    </div>
  );
}
