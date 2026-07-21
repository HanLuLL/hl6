import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import {
  useAdminAIModels,
  useAdminCreateAIModel,
  useAdminUpdateAIModel,
  useAdminDeleteAIModel,
} from "@/hooks/use-ai-audit";
import type { AIModelConfig, AIModelConfigInput } from "@/types/ai-audit";

const emptyModelForm: AIModelConfigInput = {
  name: "",
  provider: "openai",
  api_base_url: "",
  api_key: "",
  model_name: "",
  is_default: false,
  is_enabled: true,
  max_tokens: 4096,
  temperature: 0.7,
  rate_limit_rpm: 60,
};

export function AIModelsTab() {
  const { t } = useTranslation();
  const { data: models, isLoading } = useAdminAIModels();
  const createMutation = useAdminCreateAIModel();
  const updateMutation = useAdminUpdateAIModel();
  const deleteMutation = useAdminDeleteAIModel();

  const [showAdd, setShowAdd] = useState(false);
  const [addForm, setAddForm] = useState<AIModelConfigInput>({ ...emptyModelForm });
  const [editTarget, setEditTarget] = useState<AIModelConfig | null>(null);
  const [editForm, setEditForm] = useState<AIModelConfigInput>({ ...emptyModelForm });
  const [deleteTarget, setDeleteTarget] = useState<AIModelConfig | null>(null);

  const handleAdd = () => {
    if (!addForm.name.trim() || !addForm.api_base_url.trim() || !addForm.model_name.trim()) {
      return;
    }
    createMutation.mutate(addForm, {
      onSuccess: () => {
        setShowAdd(false);
        setAddForm({ ...emptyModelForm });
      },
    });
  };

  const openEdit = (m: AIModelConfig) => {
    setEditTarget(m);
    setEditForm({
      name: m.name,
      provider: m.provider,
      api_base_url: m.api_base_url,
      api_key: "",
      model_name: m.model_name,
      is_default: m.is_default,
      is_enabled: m.is_enabled,
      max_tokens: m.max_tokens,
      temperature: m.temperature,
      rate_limit_rpm: m.rate_limit_rpm,
    });
  };

  const handleEdit = () => {
    if (!editTarget) return;
    const data: Partial<AIModelConfigInput> = { ...editForm };
    if (!data.api_key) delete data.api_key;
    updateMutation.mutate({ id: editTarget.id, data }, { onSuccess: () => setEditTarget(null) });
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
                  <TableHead>{t("aiAudit.colProvider")}</TableHead>
                  <TableHead>{t("aiAudit.colModelName")}</TableHead>
                  <TableHead>{t("aiAudit.colStatus")}</TableHead>
                  <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(3)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-24" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-20" /></TableCell>
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
        <Button onClick={() => setShowAdd(true)}>{t("aiAudit.addModel")}</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("aiAudit.colName")}</TableHead>
                <TableHead>{t("aiAudit.colProvider")}</TableHead>
                <TableHead>{t("aiAudit.colModelName")}</TableHead>
                <TableHead>{t("aiAudit.colDefault")}</TableHead>
                <TableHead>{t("aiAudit.colStatus")}</TableHead>
                <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {!models || models.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                    {t("aiAudit.noModels")}
                  </TableCell>
                </TableRow>
              ) : (
                models.map((m) => (
                  <TableRow key={m.id}>
                    <TableCell className="font-medium">{m.name}</TableCell>
                    <TableCell className="text-muted-foreground">{m.provider}</TableCell>
                    <TableCell className="text-muted-foreground font-mono text-xs">{m.model_name}</TableCell>
                    <TableCell>
                      {m.is_default && <Badge>{t("aiAudit.default")}</Badge>}
                    </TableCell>
                    <TableCell>
                      <Badge variant={m.is_enabled ? "default" : "secondary"}>
                        {m.is_enabled ? t("common.active") : t("common.inactive")}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right space-x-1">
                      <Button variant="ghost" size="sm" onClick={() => openEdit(m)}>
                        {t("common.edit")}
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-destructive"
                        onClick={() => setDeleteTarget(m)}
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

      {/* Add Model Dialog */}
      <Dialog open={showAdd} onOpenChange={setShowAdd}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("aiAudit.addModelTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("aiAudit.colName")}</Label>
                <Input
                  value={addForm.name}
                  onChange={(e) => setAddForm({ ...addForm, name: e.target.value })}
                  placeholder="GPT-4o"
                />
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colProvider")}</Label>
                <Select
                  value={addForm.provider}
                  onValueChange={(v) => setAddForm({ ...addForm, provider: v })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="openai">OpenAI</SelectItem>
                    <SelectItem value="custom">Custom</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label>{t("aiAudit.colApiBaseUrl")}</Label>
              <Input
                value={addForm.api_base_url}
                onChange={(e) => setAddForm({ ...addForm, api_base_url: e.target.value })}
                placeholder="https://api.openai.com/v1"
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("aiAudit.colApiKey")}</Label>
                <Input
                  type="password"
                  value={addForm.api_key}
                  onChange={(e) => setAddForm({ ...addForm, api_key: e.target.value })}
                  placeholder="sk-..."
                />
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colModelName")}</Label>
                <Input
                  value={addForm.model_name}
                  onChange={(e) => setAddForm({ ...addForm, model_name: e.target.value })}
                  placeholder="gpt-4o"
                />
              </div>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label>{t("aiAudit.colMaxTokens")}</Label>
                <Input
                  type="number"
                  value={addForm.max_tokens}
                  onChange={(e) => setAddForm({ ...addForm, max_tokens: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colTemperature")}</Label>
                <Input
                  type="number"
                  step="0.1"
                  value={addForm.temperature}
                  onChange={(e) => setAddForm({ ...addForm, temperature: parseFloat(e.target.value) || 0 })}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colRateLimitRpm")}</Label>
                <Input
                  type="number"
                  value={addForm.rate_limit_rpm}
                  onChange={(e) => setAddForm({ ...addForm, rate_limit_rpm: parseInt(e.target.value) || 0 })}
                />
              </div>
            </div>
            <div className="flex items-center justify-between">
              <Label>{t("aiAudit.colDefault")}</Label>
              <Switch
                checked={addForm.is_default}
                onCheckedChange={(v) => setAddForm({ ...addForm, is_default: v })}
              />
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
              disabled={!addForm.name.trim() || !addForm.api_base_url.trim() || !addForm.model_name.trim() || createMutation.isPending}
            >
              {createMutation.isPending ? t("common.creating") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Model Dialog */}
      <Dialog open={!!editTarget} onOpenChange={(open) => !open && setEditTarget(null)}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("aiAudit.editModelTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("aiAudit.colName")}</Label>
                <Input
                  value={editForm.name}
                  onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colProvider")}</Label>
                <Select
                  value={editForm.provider}
                  onValueChange={(v) => setEditForm({ ...editForm, provider: v })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="openai">OpenAI</SelectItem>
                    <SelectItem value="custom">Custom</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-2">
              <Label>{t("aiAudit.colApiBaseUrl")}</Label>
              <Input
                value={editForm.api_base_url}
                onChange={(e) => setEditForm({ ...editForm, api_base_url: e.target.value })}
              />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label>{t("aiAudit.colApiKey")}</Label>
                <Input
                  type="password"
                  value={editForm.api_key}
                  onChange={(e) => setEditForm({ ...editForm, api_key: e.target.value })}
                  placeholder={t("aiAudit.apiKeyPlaceholder")}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colModelName")}</Label>
                <Input
                  value={editForm.model_name}
                  onChange={(e) => setEditForm({ ...editForm, model_name: e.target.value })}
                />
              </div>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div className="space-y-2">
                <Label>{t("aiAudit.colMaxTokens")}</Label>
                <Input
                  type="number"
                  value={editForm.max_tokens}
                  onChange={(e) => setEditForm({ ...editForm, max_tokens: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colTemperature")}</Label>
                <Input
                  type="number"
                  step="0.1"
                  value={editForm.temperature}
                  onChange={(e) => setEditForm({ ...editForm, temperature: parseFloat(e.target.value) || 0 })}
                />
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colRateLimitRpm")}</Label>
                <Input
                  type="number"
                  value={editForm.rate_limit_rpm}
                  onChange={(e) => setEditForm({ ...editForm, rate_limit_rpm: parseInt(e.target.value) || 0 })}
                />
              </div>
            </div>
            <div className="flex items-center justify-between">
              <Label>{t("aiAudit.colDefault")}</Label>
              <Switch
                checked={editForm.is_default}
                onCheckedChange={(v) => setEditForm({ ...editForm, is_default: v })}
              />
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

      {/* Delete Model Dialog */}
      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("aiAudit.deleteModelTitle")}</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            {t("aiAudit.deleteModelConfirm", { name: deleteTarget?.name ?? "" })}
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
