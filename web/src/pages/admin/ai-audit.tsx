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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDocumentTitle } from "@/hooks/use-document-title";
import {
  useAdminAIModels,
  useAdminCreateAIModel,
  useAdminUpdateAIModel,
  useAdminDeleteAIModel,
  useAdminPromptTemplates,
  useAdminCreatePromptTemplate,
  useAdminUpdatePromptTemplate,
  useAdminDeletePromptTemplate,
  useAdminAIReviews,
  useAdminReviewAIReview,
  useAdminAppeals,
  useAdminReviewAppeal,
} from "@/hooks/use-ai-audit";
import type {
  AIModelConfig,
  AIModelConfigInput,
  AuditPromptTemplate,
  PromptTemplateInput,
  AuditAIReview,
  UserAppeal,
} from "@/types/ai-audit";

// ---------- AI Model Config Tab ----------

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

function AIModelConfigTab() {
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

// ---------- Prompt Template Tab ----------

const emptyTemplateForm: PromptTemplateInput = {
  name: "",
  system_prompt: "",
  user_prompt: "",
  description: "",
  sort_order: 0,
  is_default: false,
  is_enabled: true,
};

function PromptTemplateTab() {
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

// ---------- AI Review Records Tab ----------

function judgmentBadge(judgment: string, t: (key: string) => string) {
  switch (judgment) {
    case "clean":
      return <Badge className="bg-green-600 hover:bg-green-700">{t("aiAudit.judgmentClean")}</Badge>;
    case "violation":
      return <Badge variant="destructive">{t("aiAudit.judgmentViolation")}</Badge>;
    case "error":
      return <Badge variant="secondary">{t("aiAudit.judgmentError")}</Badge>;
    default:
      return <Badge variant="outline">{judgment}</Badge>;
  }
}

function reviewStatusBadge(status: string, t: (key: string) => string) {
  switch (status) {
    case "pending":
      return <Badge className="bg-yellow-600 hover:bg-yellow-700">{t("aiAudit.reviewStatusPending")}</Badge>;
    case "confirmed":
      return <Badge variant="destructive">{t("aiAudit.reviewStatusConfirmed")}</Badge>;
    case "overturned":
      return <Badge className="bg-green-600 hover:bg-green-700">{t("aiAudit.reviewStatusOverturned")}</Badge>;
    case "dismissed":
      return <Badge variant="secondary">{t("aiAudit.reviewStatusDismissed")}</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

function AIReviewTab() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const params = new URLSearchParams({ page: String(page), page_size: "15" });
  const { data, isLoading } = useAdminAIReviews(params.toString());
  const reviewMutation = useAdminReviewAIReview();

  const [reviewTarget, setReviewTarget] = useState<AuditAIReview | null>(null);
  const [reviewStatus, setReviewStatus] = useState<string>("confirmed");
  const [reviewNote, setReviewNote] = useState("");

  const reviews = (data?.data ?? []) as AuditAIReview[];
  const total = (data?.total ?? 0) as number;
  const totalPages = Math.ceil(total / 15);

  const handleReview = () => {
    if (!reviewTarget) return;
    reviewMutation.mutate(
      { id: reviewTarget.id, status: reviewStatus, note: reviewNote.trim() || undefined },
      { onSuccess: () => {
        setReviewTarget(null);
        setReviewStatus("confirmed");
        setReviewNote("");
      }},
    );
  };

  const openReview = (r: AuditAIReview) => {
    setReviewTarget(r);
    setReviewStatus("confirmed");
    setReviewNote("");
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("aiAudit.colFqdn")}</TableHead>
                  <TableHead>{t("aiAudit.colJudgment")}</TableHead>
                  <TableHead>{t("aiAudit.colConfidence")}</TableHead>
                  <TableHead>{t("aiAudit.colReviewStatus")}</TableHead>
                  <TableHead>{t("aiAudit.colCreatedAt")}</TableHead>
                  <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(5)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-12" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell className="text-right"><Skeleton className="h-8 w-20 ml-auto" /></TableCell>
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
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("aiAudit.colFqdn")}</TableHead>
                <TableHead>{t("aiAudit.colJudgment")}</TableHead>
                <TableHead>{t("aiAudit.colConfidence")}</TableHead>
                <TableHead>{t("aiAudit.colViolationTypes")}</TableHead>
                <TableHead>{t("aiAudit.colReviewStatus")}</TableHead>
                <TableHead>{t("aiAudit.colCreatedAt")}</TableHead>
                <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {reviews.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center text-muted-foreground py-8">
                    {t("aiAudit.noReviews")}
                  </TableCell>
                </TableRow>
              ) : (
                reviews.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell className="font-medium font-mono text-xs">{r.fqdn}</TableCell>
                    <TableCell>{judgmentBadge(r.ai_judgment, t)}</TableCell>
                    <TableCell className="text-muted-foreground">{(r.ai_confidence * 100).toFixed(1)}%</TableCell>
                    <TableCell className="text-xs text-muted-foreground max-w-[200px] truncate">
                      {r.violation_types?.join(", ") || "-"}
                    </TableCell>
                    <TableCell>{reviewStatusBadge(r.admin_review_status, t)}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(r.created_at).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" onClick={() => openReview(r)}>
                        {t("aiAudit.review")}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {totalPages > 1 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            {t("common.previous")}
          </Button>
          <span className="flex items-center text-sm text-muted-foreground">
            {t("common.pageOf", { page, total: totalPages })}
          </span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
            {t("common.next")}
          </Button>
        </div>
      )}

      {/* Admin Review Dialog */}
      <Dialog open={!!reviewTarget} onOpenChange={(open) => !open && setReviewTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("aiAudit.adminReviewTitle")}</DialogTitle>
          </DialogHeader>
          {reviewTarget && (
            <div className="space-y-4">
              <div className="rounded-md border p-3 space-y-2 text-sm">
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">{t("aiAudit.colFqdn")}:</span>
                  <span className="font-mono">{reviewTarget.fqdn}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">{t("aiAudit.colJudgment")}:</span>
                  {judgmentBadge(reviewTarget.ai_judgment, t)}
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">{t("aiAudit.colConfidence")}:</span>
                  <span>{(reviewTarget.ai_confidence * 100).toFixed(1)}%</span>
                </div>
                {reviewTarget.ai_suggested_action && (
                  <div className="flex gap-2">
                    <span className="text-muted-foreground shrink-0">{t("aiAudit.colSuggestedAction")}:</span>
                    <span>{reviewTarget.ai_suggested_action}</span>
                  </div>
                )}
                {reviewTarget.ai_response && (
                  <div className="flex gap-2">
                    <span className="text-muted-foreground shrink-0">{t("aiAudit.colAiResponse")}:</span>
                    <span className="text-xs break-all">{reviewTarget.ai_response}</span>
                  </div>
                )}
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colReviewStatus")}</Label>
                <Select value={reviewStatus} onValueChange={setReviewStatus}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="confirmed">{t("aiAudit.reviewStatusConfirmed")}</SelectItem>
                    <SelectItem value="overturned">{t("aiAudit.reviewStatusOverturned")}</SelectItem>
                    <SelectItem value="dismissed">{t("aiAudit.reviewStatusDismissed")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colAdminNote")}</Label>
                <Textarea
                  value={reviewNote}
                  onChange={(e) => setReviewNote(e.target.value)}
                  placeholder={t("aiAudit.adminNotePlaceholder")}
                  rows={3}
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setReviewTarget(null)}>{t("common.cancel")}</Button>
            <Button onClick={handleReview} disabled={reviewMutation.isPending}>
              {reviewMutation.isPending ? t("common.saving") : t("common.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ---------- Appeal Management Tab ----------

function appealStatusBadge(status: string, t: (key: string) => string) {
  switch (status) {
    case "pending":
      return <Badge className="bg-yellow-600 hover:bg-yellow-700">{t("aiAudit.appealStatusPending")}</Badge>;
    case "approved":
      return <Badge className="bg-green-600 hover:bg-green-700">{t("aiAudit.appealStatusApproved")}</Badge>;
    case "rejected":
      return <Badge variant="destructive">{t("aiAudit.appealStatusRejected")}</Badge>;
    default:
      return <Badge variant="outline">{status}</Badge>;
  }
}

function AppealTab() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const params = new URLSearchParams({ page: String(page), page_size: "15" });
  const { data, isLoading } = useAdminAppeals(params.toString());
  const reviewMutation = useAdminReviewAppeal();

  const [appealTarget, setAppealTarget] = useState<UserAppeal | null>(null);
  const [appealStatus, setAppealStatus] = useState<string>("approved");
  const [appealReply, setAppealReply] = useState("");

  const appeals = (data?.data ?? []) as UserAppeal[];
  const total = (data?.total ?? 0) as number;
  const totalPages = Math.ceil(total / 15);

  const handleReview = () => {
    if (!appealTarget) return;
    reviewMutation.mutate(
      { id: appealTarget.id, status: appealStatus, reply: appealReply.trim() || undefined },
      { onSuccess: () => {
        setAppealTarget(null);
        setAppealStatus("approved");
        setAppealReply("");
      }},
    );
  };

  const openReview = (a: UserAppeal) => {
    setAppealTarget(a);
    setAppealStatus("approved");
    setAppealReply("");
  };

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Card>
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("aiAudit.colUserId")}</TableHead>
                  <TableHead>{t("aiAudit.colContent")}</TableHead>
                  <TableHead>{t("aiAudit.colAppealStatus")}</TableHead>
                  <TableHead>{t("aiAudit.colCreatedAt")}</TableHead>
                  <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {[...Array(5)].map((_, i) => (
                  <TableRow key={i}>
                    <TableCell><Skeleton className="h-4 w-16" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-40" /></TableCell>
                    <TableCell><Skeleton className="h-5 w-16 rounded-full" /></TableCell>
                    <TableCell><Skeleton className="h-4 w-28" /></TableCell>
                    <TableCell className="text-right"><Skeleton className="h-8 w-20 ml-auto" /></TableCell>
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
      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("aiAudit.colUserId")}</TableHead>
                <TableHead>{t("aiAudit.colContent")}</TableHead>
                <TableHead>{t("aiAudit.colAppealStatus")}</TableHead>
                <TableHead>{t("aiAudit.colReply")}</TableHead>
                <TableHead>{t("aiAudit.colCreatedAt")}</TableHead>
                <TableHead className="text-right">{t("aiAudit.colActions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {appeals.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} className="text-center text-muted-foreground py-8">
                    {t("aiAudit.noAppeals")}
                  </TableCell>
                </TableRow>
              ) : (
                appeals.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell className="text-muted-foreground">#{a.user_id}</TableCell>
                    <TableCell className="max-w-xs truncate">{a.content}</TableCell>
                    <TableCell>{appealStatusBadge(a.status, t)}</TableCell>
                    <TableCell className="max-w-xs truncate text-muted-foreground">{a.reply || "-"}</TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(a.created_at).toLocaleString()}
                    </TableCell>
                    <TableCell className="text-right">
                      <Button variant="ghost" size="sm" onClick={() => openReview(a)}>
                        {t("aiAudit.review")}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {totalPages > 1 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            {t("common.previous")}
          </Button>
          <span className="flex items-center text-sm text-muted-foreground">
            {t("common.pageOf", { page, total: totalPages })}
          </span>
          <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
            {t("common.next")}
          </Button>
        </div>
      )}

      {/* Appeal Review Dialog */}
      <Dialog open={!!appealTarget} onOpenChange={(open) => !open && setAppealTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("aiAudit.appealReviewTitle")}</DialogTitle>
          </DialogHeader>
          {appealTarget && (
            <div className="space-y-4">
              <div className="rounded-md border p-3 space-y-2 text-sm">
                <div className="flex items-center gap-2">
                  <span className="text-muted-foreground">{t("aiAudit.colUserId")}:</span>
                  <span>#{appealTarget.user_id}</span>
                </div>
                <div className="flex gap-2">
                  <span className="text-muted-foreground shrink-0">{t("aiAudit.colContent")}:</span>
                  <span>{appealTarget.content}</span>
                </div>
                {appealTarget.review_id && (
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground">{t("aiAudit.colReviewId")}:</span>
                    <span>#{appealTarget.review_id}</span>
                  </div>
                )}
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colAppealStatus")}</Label>
                <Select value={appealStatus} onValueChange={setAppealStatus}>
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="approved">{t("aiAudit.appealStatusApproved")}</SelectItem>
                    <SelectItem value="rejected">{t("aiAudit.appealStatusRejected")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>{t("aiAudit.colReply")}</Label>
                <Textarea
                  value={appealReply}
                  onChange={(e) => setAppealReply(e.target.value)}
                  placeholder={t("aiAudit.appealReplyPlaceholder")}
                  rows={3}
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setAppealTarget(null)}>{t("common.cancel")}</Button>
            <Button onClick={handleReview} disabled={reviewMutation.isPending}>
              {reviewMutation.isPending ? t("common.saving") : t("common.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ---------- Main Page ----------

export default function AdminAIAuditPage() {
  const { t } = useTranslation();
  useDocumentTitle(t("aiAudit.pageTitle"));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("aiAudit.pageTitle")}</h1>
        <p className="text-muted-foreground">{t("aiAudit.pageSubtitle")}</p>
      </div>

      <Tabs defaultValue="models">
        <TabsList>
          <TabsTrigger value="models">{t("aiAudit.tabModels")}</TabsTrigger>
          <TabsTrigger value="templates">{t("aiAudit.tabTemplates")}</TabsTrigger>
          <TabsTrigger value="reviews">{t("aiAudit.tabReviews")}</TabsTrigger>
          <TabsTrigger value="appeals">{t("aiAudit.tabAppeals")}</TabsTrigger>
        </TabsList>
        <TabsContent value="models">
          <AIModelConfigTab />
        </TabsContent>
        <TabsContent value="templates">
          <PromptTemplateTab />
        </TabsContent>
        <TabsContent value="reviews">
          <AIReviewTab />
        </TabsContent>
        <TabsContent value="appeals">
          <AppealTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}