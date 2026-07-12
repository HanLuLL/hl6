import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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
import { Switch } from "@/components/ui/switch";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import { toast } from "sonner";
import type { UserGroup } from "@/types";
import { Skeleton } from "@/components/ui/skeleton";

export function GroupsContent() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const { data: groups, isLoading } = useQuery({
    queryKey: ["admin-groups"],
    queryFn: async () => {
      const res = await api.adminListGroups();
      return res.data;
    },
    staleTime: 30_000,
  });

  const [showAdd, setShowAdd] = useState(false);
  const [addName, setAddName] = useState("");
  const [addIsDefault, setAddIsDefault] = useState(false);
  const [addIsAdmin, setAddIsAdmin] = useState(false);

  const [editGroup, setEditGroup] = useState<UserGroup | null>(null);
  const [editName, setEditName] = useState("");
  const [editIsDefault, setEditIsDefault] = useState(false);
  const [editIsAdmin, setEditIsAdmin] = useState(false);

  const [deleteGroup, setDeleteGroup] = useState<UserGroup | null>(null);
  const [migrateTo, setMigrateTo] = useState<string>("");

  const createMutation = useMutation({
    mutationFn: api.adminCreateGroup,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-groups"] });
      toast.success(t("adminGroups.groupCreated"));
      setShowAdd(false);
      setAddName("");
      setAddIsDefault(false);
      setAddIsAdmin(false);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: number; name?: string; is_default?: boolean; is_admin?: boolean }) =>
      api.adminUpdateGroup(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-groups"] });
      toast.success(t("adminGroups.groupUpdated"));
      setEditGroup(null);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const deleteMutation = useMutation({
    mutationFn: ({ id, migrateTo }: { id: number; migrateTo: number }) =>
      api.adminDeleteGroup(id, migrateTo),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-groups"] });
      queryClient.invalidateQueries({ queryKey: ["admin-users"] });
      toast.success(t("adminGroups.groupDeleted"));
      setDeleteGroup(null);
      setMigrateTo("");
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

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
                  <TableHead>{t("adminGroups.groupName")}</TableHead>
                  <TableHead>{t("adminGroups.userCount")}</TableHead>
                  <TableHead>{t("adminGroups.status")}</TableHead>
                  <TableHead className="text-right">{t("adminGroups.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(3)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-8" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-14 rounded-full" /></TableCell>
                    <TableCell className="text-right"><Skeleton className="h-8 w-36 ml-auto" /></TableCell>
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
        <Button onClick={() => setShowAdd(true)}>{t("adminGroups.addGroup")}</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("adminGroups.groupName")}</TableHead>
                <TableHead>{t("adminGroups.userCount")}</TableHead>
                <TableHead>{t("adminGroups.status")}</TableHead>
                <TableHead className="text-right">{t("adminGroups.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {groups?.map((group) => (
                <TableRow key={group.id}>
                  <TableCell className="font-medium">{group.name}</TableCell>
                  <TableCell>{group.user_count ?? 0}</TableCell>
                  <TableCell className="space-x-1">
                    {group.is_default && <Badge>{t("adminGroups.default")}</Badge>}
                    {group.is_admin && <Badge variant="secondary">{t("adminGroups.adminGroup")}</Badge>}
                  </TableCell>
                  <TableCell className="text-right space-x-1">
                    <Button variant="ghost" size="sm" onClick={() => {
                      setEditGroup(group);
                      setEditName(group.name);
                      setEditIsDefault(group.is_default);
                      setEditIsAdmin(group.is_admin);
                    }}>{t("common.edit")}</Button>
                    {!group.is_default && (
                      <Button variant="ghost" size="sm" onClick={() => {
                        updateMutation.mutate({ id: group.id, is_default: true });
                      }}>{t("adminGroups.setAsDefault")}</Button>
                    )}
                    <Button variant="ghost" size="sm" className="text-destructive" onClick={() => {
                      setDeleteGroup(group);
                      setMigrateTo("");
                    }}>{t("common.delete")}</Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Add Dialog */}
      <Dialog open={showAdd} onOpenChange={(open) => {
        setShowAdd(open);
        if (!open) { setAddName(""); setAddIsDefault(false); setAddIsAdmin(false); }
      }}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("adminGroups.addGroup")}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("adminGroups.groupName")}</Label>
              <Input value={addName} onChange={(e) => setAddName(e.target.value)} placeholder={t("adminGroups.groupName")} required />
            </div>
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <Label>{t("adminGroups.adminGroup")}</Label>
                <p className="text-xs text-muted-foreground">{t("adminGroups.adminGroupHint")}</p>
              </div>
              <Switch checked={addIsAdmin} onCheckedChange={setAddIsAdmin} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => createMutation.mutate({ name: addName, is_default: addIsDefault, is_admin: addIsAdmin })}
              disabled={!addName.trim() || createMutation.isPending}
              data-dialog-primary="true"
            >
              {createMutation.isPending ? t("common.creating") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editGroup} onOpenChange={(open) => !open && setEditGroup(null)}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("common.edit")}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("adminGroups.groupName")}</Label>
              <Input value={editName} onChange={(e) => setEditName(e.target.value)} required />
            </div>
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <Label>{t("adminGroups.adminGroup")}</Label>
                <p className="text-xs text-muted-foreground">{t("adminGroups.adminGroupHint")}</p>
              </div>
              <Switch checked={editIsAdmin} onCheckedChange={setEditIsAdmin} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditGroup(null)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => editGroup && updateMutation.mutate({
                id: editGroup.id,
                name: editName,
                is_default: editIsDefault,
                is_admin: editIsAdmin,
              })}
              disabled={!editName.trim() || updateMutation.isPending}
              data-dialog-primary="true"
            >
              {updateMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog open={!!deleteGroup} onOpenChange={(open) => {
        if (!open) { setDeleteGroup(null); setMigrateTo(""); }
      }}>
        <DialogContent aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("adminGroups.deleteGroup")}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <p className="text-sm text-muted-foreground">
              {t("adminGroups.deleteConfirm", { name: deleteGroup?.name, count: deleteGroup?.user_count ?? 0 })}
            </p>
            <div className="space-y-2">
              <Label>{t("adminGroups.migrateTarget")}</Label>
              <Select value={migrateTo} onValueChange={setMigrateTo}>
                <SelectTrigger
                  data-hotkey-required="true"
                  data-hotkey-filled={migrateTo ? "true" : "false"}
                >
                  <SelectValue placeholder={t("adminGroups.migrateTarget")} />
                </SelectTrigger>
                <SelectContent>
                  {groups?.filter((g) => g.id !== deleteGroup?.id).map((g) => (
                    <SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteGroup(null)}>{t("common.cancel")}</Button>
            <Button
              variant="destructive"
              onClick={() => deleteGroup && migrateTo && deleteMutation.mutate({
                id: deleteGroup.id,
                migrateTo: parseInt(migrateTo),
              })}
              disabled={!migrateTo || deleteMutation.isPending}
              data-dialog-primary="true"
            >
              {deleteMutation.isPending ? t("common.delete") + "..." : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
