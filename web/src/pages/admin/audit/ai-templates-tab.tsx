import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
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
import {
  useAdminPromptTemplates,
  useAdminCreatePromptTemplate,
  useAdminUpdatePromptTemplate,
  useAdminDeletePromptTemplate,
} from "@/hooks/use-ai-audit";
import type { AuditPromptTemplate, PromptTemplateInput } from "@/types/ai-audit";

const emptyTemplateForm: PromptTemplateInput = {
  name: "",
  system_prompt: "",
  user_prompt: "",
  description: "",
  sort_order: 0,
  is_default: false,
  is_enabled: true,
};

export function AITemplatesTab() {
  const { t } = useTranslation();
  const { data: templates, isLoading } = useAdminPromptTemplates();
  const createMutation = useAdminCreatePromptTemplate();
  const updateMutation = useAdminUpdatePromptTemplate();
  const deleteMutation = useAdminDeletePromptTemplate();

  const [showAdd, setShowAdd] = useState(false);
  const [addForm, setAddForm] = useState<PromptTemplateInput>({ ...emptyTemplateForm });
  const [editTarget, setEditTarget] = useState<AuditPromptTemplate | null>(null);
  const [editForm, setEditForm] = useState<PromptTemplateInput>({ ...emptyTemplateForm });
  const [deleteTarget, setDeleteTarget] = useState<AuditPromptTemplate | null>(null);

  const handleAdd = () => {
    if (!addForm.name.trim() || !addForm.system_prompt.trim()) return;
    createMutation.mutate(addForm, {
      onSuccess: () => {
        setShowAdd(false);
        setAddForm({ ...emptyTemplateForm });
      },
    });
  };

  const openEdit = (tp: AuditPromptTemplate) => {
    setEditTarget(tp);
    setEditForm({
      name: tp.name,
      system_prompt: tp.system_prompt,
      user_prompt: tp.user_prompt,
      description: tp.description,
      sort_order: tp.sort_order,
      is_default: tp.is_default,
      is_enabled: tp.is_enabled,
    });
  };

  const handleEdit = () => {
    if (!editTarget) return;
    updateMutation.mutate({ id: editTarget.id, data: editForm }, { onSuccess: () => setEditTarget(null) });
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteMutation.mutate(deleteTarget.id, { onSuccess: () => setDeleteTarget(null) });
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="flex justify-end"><Skeleton className="h-9 w-24" /></div>
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("aiAudit.colName")}</TableHead>
                  <TableHead>{t("aiAudit.colSortOrder")}</TableHead>
                  <TableHead>{t("aiAudit.colStatus")}</TableHead>
                  <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(3)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
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
        <Button onClick={() => setShowAdd(true)}>{t("aiAudit.addTemplate")}</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("aiAudit.colName")}</TableHead>
                <TableHead>{t("aiAudit.colDescription")}</TableHead>
                <TableHead>{t("aiAudit.colSortOrder")}</TableHead>
                <TableHead>{t("aiAudit.colDefault")}</TableHead>
                <TableHead>{t("aiAudit.colStatus")}</TableHead>
                <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {!templates || templates.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                    {t("aiAudit.noTemplates")}
                  </TableCell>
                </TableRow>
              ) : (
                templates.map((tp) => (
                  <TableRow key={tp.id}>
                    <TableCell className="font-medium">{tp.name}</TableCell>
                    <TableCell className="text-muted-foreground max-w-xs truncate">{tp.description}</TableCell>
                    <TableCell>{tp.sort_order}</TableCell>
                    <TableCell>
                      {tp.is_default && <Badge>{t("aiAudit.default")}</Badge>}
                    </TableCell>
                    <TableCell>
                      <Badge variant={tp.is_enabled ? "default" : "secondary"}>
                        {tp.is_enabled ? t("common.active") : t("common.inactive")}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right space-x-1">
                      <Button variant="ghost" size="sm" onClick={() => openEdit(tp)}>
                        {t("common.edit")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive"
                        onClick={() => setDeleteTarget(tp)}
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

      {/* Add Template Dialog */}
      <Dialog open={showAdd} onOpenChange={setShowAdd}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("aiAudit.addTemplateTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("aiAudit.colName")}</Label>
              <Input
                value={addForm.name}
                onChange={(e) => setAddForm({ ...addForm, name: e.target.value })}
                placeholder={t("aiAudit.templateNamePlaceholder")}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("aiAudit.colSystemPrompt")}</Label>
              <Textarea
                value={addForm.system_prompt}
                onChange={(e) => setAddForm({ ...addForm, system_prompt: e.target.value })}
                placeholder={t("aiAudit.systemPromptPlaceholder")}
                rows={4}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("aiAudit.colUserPrompt")}</Label>
              <Textarea
                value={addForm.user_prompt}
                onChange={(e) => setAddForm({ ...addForm, user_prompt: e.target.value })}
                placeholder={t("aiAudit.userPromptPlaceholder")}
                rows={4}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("aiAudit.colDescription")}</Label>
              <Textarea
                value={addForm.description}
                onChange={(e) => setAddForm({ ...addForm, description: e.target.value })}
                placeholder={t("aiAudit.descriptionPlaceholder")}
                rows={2}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("aiAudit.colSortOrder")}</Label>
                <Input
                  type="number"
                  value={addForm.sort_order}
                  onChange={(e) => setAddForm({ ...addForm, sort_order: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div className="flex items-center justify-between pt-6">
                <Label>{t("aiAudit.colDefault")}</Label>
                <Switch
                  checked={addForm.is_default}
                  onCheckedChange={(v) => setAddForm({ ...addForm, is_default: v })}
                />
              </div>
            </div>
            <div className="flex items-center justify-between">
              <Label>{t("aiAudit.colEnabled")}</Label>
              <Switch
                checked={addForm.is_enabled}
                onCheckedChange={(v) => setAddForm({ ...addForm, is_enabled: v })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>{t("common.cancel")}</Button>
            <Button
              onClick={handleAdd}
              disabled={!addForm.name.trim() || !addForm.system_prompt.trim() || createMutation.isPending}
            >
              {createMutation.isPending ? t("common.creating") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Template Dialog */}
      <Dialog open={!!editTarget} onOpenChange={(open) => !open && setEditTarget(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("aiAudit.editTemplateTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("aiAudit.colName")}</Label>
              <Input
                value={editForm.name}
                onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("aiAudit.colSystemPrompt")}</Label>
              <Textarea
                value={editForm.system_prompt}
                onChange={(e) => setEditForm({ ...editForm, system_prompt: e.target.value })}
                rows={4}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("aiAudit.colUserPrompt")}</Label>
              <Textarea
                value={editForm.user_prompt}
                onChange={(e) => setEditForm({ ...editForm, user_prompt: e.target.value })}
                rows={4}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("aiAudit.colDescription")}</Label>
              <Textarea
                value={editForm.description}
                onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                rows={2}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("aiAudit.colSortOrder")}</Label>
                <Input
                  type="number"
                  value={editForm.sort_order}
                  onChange={(e) => setEditForm({ ...editForm, sort_order: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div className="flex items-center justify-between pt-6">
                <Label>{t("aiAudit.colDefault")}</Label>
                <Switch
                  checked={editForm.is_default}
                  onCheckedChange={(v) => setEditForm({ ...editForm, is_default: v })}
                />
              </div>
            </div>
            <div className="flex items-center justify-between">
              <Label>{t("aiAudit.colEnabled")}</Label>
              <Switch
                checked={editForm.is_enabled}
                onCheckedChange={(v) => setEditForm({ ...editForm, is_enabled: v })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTarget(null)}>{t("common.cancel")}</Button>
            <Button onClick={handleEdit} disabled={updateMutation.isPending}>
              {updateMutation.isPending ? t("common.saving") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Template Dialog */}
      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("aiAudit.deleteTemplateTitle")}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            {t("aiAudit.deleteTemplateConfirm", { name: deleteTarget?.name ?? "" })}
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>{t("common.cancel")}</Button>
            <Button variant="destructive" onClick={handleDelete} disabled={deleteMutation.isPending}>
              {deleteMutation.isPending ? t("common.deleting") : t("common.delete")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
