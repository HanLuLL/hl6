import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
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
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from "@/components/ui/command";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, getErrorMessage } from "@/lib/api";
import { toast } from "sonner";
import { Check, ChevronsUpDown, Plus, X } from "lucide-react";
import { cn } from "@/lib/utils";
import type { CloudflareZone, DomainWithGroupAccess, UserGroup } from "@/types";

interface GroupAccessEntry {
  group_id: number;
  credit_cost: number;
}

export default function AdminDomainsPage() {
  const queryClient = useQueryClient();
  const { t } = useTranslation();
  const { data: domains, isLoading } = useQuery({
    queryKey: ["admin-domains"],
    queryFn: async () => {
      const res = await api.adminListDomainsFull();
      return res.data;
    },
  });

  const { data: groups } = useQuery({
    queryKey: ["admin-groups"],
    queryFn: async () => {
      const res = await api.adminListGroups();
      return res.data;
    },
  });

  const [showAdd, setShowAdd] = useState(false);
  const [editDomain, setEditDomain] = useState<DomainWithGroupAccess | null>(null);
  const [selectedZone, setSelectedZone] = useState<CloudflareZone | null>(null);
  const [description, setDescription] = useState("");
  const [groupAccess, setGroupAccess] = useState<GroupAccessEntry[]>([]);

  const createMutation = useMutation({
    mutationFn: api.adminCreateDomain,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-domains"] });
      toast.success(t("adminDomains.domainCreated"));
      setShowAdd(false);
      setSelectedZone(null);
      setDescription("");
      setGroupAccess([]);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: number; is_active?: boolean; description?: string; group_access?: GroupAccessEntry[] }) =>
      api.adminUpdateDomain(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-domains"] });
      queryClient.invalidateQueries({ queryKey: ["domains"] });
      toast.success(t("adminDomains.domainUpdated"));
      setEditDomain(null);
    },
    onError: (err) => toast.error(getErrorMessage(err, t)),
  });

  if (isLoading) {
    return <div className="flex items-center justify-center py-12"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" /></div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("adminDomains.title")}</h1>
          <p className="text-muted-foreground">{t("adminDomains.subtitle")}</p>
        </div>
        <Button onClick={() => {
          setGroupAccess([]);
          setShowAdd(true);
        }}>{t("adminDomains.addDomain")}</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("adminDomains.domain")}</TableHead>
                <TableHead>{t("adminDomains.zoneId")}</TableHead>
                <TableHead>{t("adminDomains.groupAccess")}</TableHead>
                <TableHead>{t("adminDomains.status")}</TableHead>
                <TableHead className="text-right">{t("adminDomains.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {domains?.map((domain) => (
                <TableRow key={domain.id}>
                  <TableCell className="font-medium">{domain.name}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{domain.cloudflare_zone_id.slice(0, 12)}...</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {domain.group_access?.map((ga) => (
                        <Badge key={ga.group_id} variant="outline" className="text-xs">
                          {ga.group?.name ?? `#${ga.group_id}`}: {ga.credit_cost}
                        </Badge>
                      ))}
                      {(!domain.group_access || domain.group_access.length === 0) && (
                        <span className="text-xs text-muted-foreground">-</span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={domain.is_active ? "default" : "secondary"}>
                      {domain.is_active ? t("common.active") : t("common.inactive")}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => {
                      setEditDomain(domain);
                    }}>{t("common.edit")}</Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => updateMutation.mutate({ id: domain.id, is_active: !domain.is_active })}
                    >
                      {domain.is_active ? t("common.disable") : t("common.enable")}
                    </Button>
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
        if (!open) {
          setSelectedZone(null);
          setDescription("");
          setGroupAccess([]);
        }
      }}>
        <DialogContent className="max-w-lg" aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("adminDomains.addDomain")}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("adminDomains.domain")}</Label>
              <ZoneCombobox
                value={selectedZone}
                onSelect={setSelectedZone}
                enabled={showAdd}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("adminDomains.description")}</Label>
              <Textarea placeholder={t("adminDomains.optionalDescription")} value={description} onChange={(e) => setDescription(e.target.value)} />
            </div>
            <GroupAccessEditor
              groups={groups ?? []}
              value={groupAccess}
              onChange={setGroupAccess}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => {
                if (!selectedZone) return;
                createMutation.mutate({
                  name: selectedZone.name,
                  cloudflare_zone_id: selectedZone.id,
                  description,
                  group_access: groupAccess,
                });
              }}
              disabled={!selectedZone || createMutation.isPending}
            >
              {createMutation.isPending ? t("common.creating") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editDomain} onOpenChange={(open) => !open && setEditDomain(null)}>
        <DialogContent className="max-w-lg" aria-describedby={undefined}>
          <DialogHeader><DialogTitle>{t("adminDomains.editDomain", { name: editDomain?.name })}</DialogTitle></DialogHeader>
          {editDomain && (
            <EditDomainForm
              domain={editDomain}
              groups={groups ?? []}
              onSave={(data) => updateMutation.mutate({ id: editDomain.id, ...data })}
              isPending={updateMutation.isPending}
            />
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function GroupAccessEditor({ groups, value, onChange }: {
  groups: UserGroup[];
  value: GroupAccessEntry[];
  onChange: (v: GroupAccessEntry[]) => void;
}) {
  const { t } = useTranslation();
  const [bulkCost, setBulkCost] = useState("1");
  const usedGroupIds = new Set(value.map((v) => v.group_id));
  const availableGroups = groups.filter((g) => !usedGroupIds.has(g.id));

  return (
    <div className="space-y-2">
      <Label>{t("adminDomains.groupAccess")}</Label>
      {/* Bulk price row */}
      <div className="flex items-center gap-2 rounded-md border border-dashed p-2">
        <span className="text-sm min-w-24 font-medium">{t("adminDomains.allGroups")}</span>
        <Input
          type="number"
          step="any"
          className="w-24"
          value={bulkCost}
          onChange={(e) => {
            setBulkCost(e.target.value);
            const cost = parseFloat(e.target.value) || 0;
            if (value.length > 0) {
              onChange(value.map((entry) => ({ ...entry, credit_cost: cost })));
            }
          }}
        />
        <span className="text-xs text-muted-foreground">{t("adminDomains.creditCost")}</span>
        {availableGroups.length > 0 && (
          <Button
            variant="outline"
            size="sm"
            className="ml-auto shrink-0"
            onClick={() => {
              const cost = parseFloat(bulkCost) || 1;
              const newEntries = availableGroups.map((g) => ({ group_id: g.id, credit_cost: cost }));
              onChange([...value, ...newEntries]);
            }}
          >
            <Plus className="h-3 w-3 mr-1" />
            {t("adminDomains.addAllGroups")}
          </Button>
        )}
      </div>
      {/* Per-group rows */}
      <div className="space-y-2">
        {value.map((entry, idx) => {
          const group = groups.find((g) => g.id === entry.group_id);
          return (
            <div key={entry.group_id} className="flex items-center gap-2 flex-wrap">
              <span className="text-sm min-w-24 truncate">{group?.name ?? `#${entry.group_id}`}</span>
              <Input
                type="number"
                step="any"
                className="w-24"
                value={entry.credit_cost}
                onChange={(e) => {
                  const next = [...value];
                  next[idx] = { ...entry, credit_cost: parseFloat(e.target.value) || 0 };
                  onChange(next);
                }}
              />
              <span className="text-xs text-muted-foreground">{t("adminDomains.creditCost")}</span>
              <Button variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={() => {
                onChange(value.filter((_, i) => i !== idx));
              }}>
                <X className="h-4 w-4" />
              </Button>
              {entry.credit_cost < 0 && (
                <p className="text-xs text-green-600 basis-full">{t("adminDomains.negativeCostHint")}</p>
              )}
            </div>
          );
        })}
      </div>
      {availableGroups.length > 0 && (
        <Select onValueChange={(v) => {
          const groupId = parseInt(v);
          const cost = parseFloat(bulkCost) || 1;
          onChange([...value, { group_id: groupId, credit_cost: cost }]);
        }}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder={t("adminDomains.addGroupAccess")} />
          </SelectTrigger>
          <SelectContent>
            {availableGroups.map((g) => (
              <SelectItem key={g.id} value={String(g.id)}>
                <div className="flex items-center gap-2">
                  <Plus className="h-3 w-3" />
                  {g.name}
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )}
      {value.length === 0 && (
        <p className="text-xs text-muted-foreground">{t("adminDomains.noGroupAccess")}</p>
      )}
    </div>
  );
}

function ZoneCombobox({ value, onSelect, enabled }: {
  value: CloudflareZone | null;
  onSelect: (zone: CloudflareZone | null) => void;
  enabled: boolean;
}) {
  const [open, setOpen] = useState(false);
  const { t } = useTranslation();

  const { data: zones, isLoading } = useQuery({
    queryKey: ["admin-cloudflare-zones"],
    queryFn: async () => {
      const res = await api.adminListCloudflareZones();
      return res.data;
    },
    enabled,
  });

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between font-normal"
        >
          {value ? value.name : t("adminDomains.selectDomain")}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start" onWheel={(e) => e.stopPropagation()}>
        <Command>
          <CommandInput placeholder={t("adminDomains.searchDomains")} />
          <CommandList>
            <CommandEmpty>
              {isLoading ? t("common.loading") : t("adminDomains.noDomainsFound")}
            </CommandEmpty>
            <CommandGroup>
              {zones?.map((zone) => (
                <CommandItem
                  key={zone.id}
                  value={zone.name}
                  onSelect={() => {
                    onSelect(value?.id === zone.id ? null : zone);
                    setOpen(false);
                  }}
                >
                  <Check className={cn("mr-2 h-4 w-4", value?.id === zone.id ? "opacity-100" : "opacity-0")} />
                  <span className="flex-1">{zone.name}</span>
                  <Badge variant="secondary" className="ml-2 text-xs">{zone.status}</Badge>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

function EditDomainForm({ domain, groups, onSave, isPending }: {
  domain: DomainWithGroupAccess;
  groups: UserGroup[];
  onSave: (data: { description: string; group_access: GroupAccessEntry[] }) => void;
  isPending: boolean;
}) {
  const [desc, setDesc] = useState(domain.description);
  const [access, setAccess] = useState<GroupAccessEntry[]>(
    domain.group_access?.map((ga) => ({ group_id: ga.group_id, credit_cost: ga.credit_cost })) ?? []
  );
  const { t } = useTranslation();

  return (
    <>
      <div className="space-y-4 py-4">
        <div className="space-y-2">
          <Label>{t("adminDomains.description")}</Label>
          <Textarea value={desc} onChange={(e) => setDesc(e.target.value)} />
        </div>
        <GroupAccessEditor
          groups={groups}
          value={access}
          onChange={setAccess}
        />
      </div>
      <DialogFooter>
        <Button onClick={() => onSave({ description: desc, group_access: access })} disabled={isPending}>
          {isPending ? t("common.saving") : t("common.save")}
        </Button>
      </DialogFooter>
    </>
  );
}
