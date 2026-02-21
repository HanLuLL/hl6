import { useState } from "react";
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
import { api } from "@/lib/api";
import { toast } from "sonner";
import { Check, ChevronsUpDown } from "lucide-react";
import { cn } from "@/lib/utils";
import type { Domain, CloudflareZone } from "@/types";

export default function AdminDomainsPage() {
  const queryClient = useQueryClient();
  const { data: domains, isLoading } = useQuery({
    queryKey: ["admin-domains"],
    queryFn: async () => {
      const res = await api.listDomains();
      return res.data;
    },
  });

  const [showAdd, setShowAdd] = useState(false);
  const [editDomain, setEditDomain] = useState<Domain | null>(null);
  const [selectedZone, setSelectedZone] = useState<CloudflareZone | null>(null);
  const [creditCost, setCreditCost] = useState("1");
  const [description, setDescription] = useState("");

  const createMutation = useMutation({
    mutationFn: api.adminCreateDomain,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-domains"] });
      toast.success("Domain created");
      setShowAdd(false);
      setSelectedZone(null);
      setCreditCost("1");
      setDescription("");
    },
    onError: (err) => toast.error(err.message),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, ...data }: { id: number; credit_cost?: number; is_active?: boolean; description?: string }) =>
      api.adminUpdateDomain(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-domains"] });
      toast.success("Domain updated");
      setEditDomain(null);
    },
    onError: (err) => toast.error(err.message),
  });

  if (isLoading) {
    return <div className="flex items-center justify-center py-12"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" /></div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Manage Domains</h1>
          <p className="text-muted-foreground">Add and configure root domains</p>
        </div>
        <Button onClick={() => setShowAdd(true)}>Add Domain</Button>
      </div>

      <Card>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Domain</TableHead>
                <TableHead>Zone ID</TableHead>
                <TableHead>Cost</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {domains?.map((domain) => (
                <TableRow key={domain.id}>
                  <TableCell className="font-medium">{domain.name}</TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{domain.cloudflare_zone_id.slice(0, 12)}...</TableCell>
                  <TableCell>{domain.credit_cost}</TableCell>
                  <TableCell>
                    <Badge variant={domain.is_active ? "default" : "secondary"}>
                      {domain.is_active ? "Active" : "Inactive"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => setEditDomain(domain)}>Edit</Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => updateMutation.mutate({ id: domain.id, is_active: !domain.is_active })}
                    >
                      {domain.is_active ? "Disable" : "Enable"}
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
          setCreditCost("1");
          setDescription("");
        }
      }}>
        <DialogContent>
          <DialogHeader><DialogTitle>Add Domain</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Domain</Label>
              <ZoneCombobox
                value={selectedZone}
                onSelect={setSelectedZone}
                enabled={showAdd}
              />
            </div>
            <div className="space-y-2">
              <Label>Credit Cost</Label>
              <Input type="number" min="1" value={creditCost} onChange={(e) => setCreditCost(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label>Description</Label>
              <Textarea placeholder="Optional description" value={description} onChange={(e) => setDescription(e.target.value)} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>Cancel</Button>
            <Button
              onClick={() => {
                if (!selectedZone) return;
                createMutation.mutate({
                  name: selectedZone.name,
                  cloudflare_zone_id: selectedZone.id,
                  credit_cost: parseInt(creditCost) || 1,
                  description,
                });
              }}
              disabled={!selectedZone || createMutation.isPending}
            >
              {createMutation.isPending ? "Creating..." : "Create"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={!!editDomain} onOpenChange={(open) => !open && setEditDomain(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>Edit Domain: {editDomain?.name}</DialogTitle></DialogHeader>
          {editDomain && (
            <EditDomainForm
              domain={editDomain}
              onSave={(data) => updateMutation.mutate({ id: editDomain.id, ...data })}
              isPending={updateMutation.isPending}
            />
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ZoneCombobox({ value, onSelect, enabled }: {
  value: CloudflareZone | null;
  onSelect: (zone: CloudflareZone | null) => void;
  enabled: boolean;
}) {
  const [open, setOpen] = useState(false);

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
          {value ? value.name : "Select a domain..."}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0" align="start" onWheel={(e) => e.stopPropagation()}>
        <Command>
          <CommandInput placeholder="Search domains..." />
          <CommandList>
            <CommandEmpty>
              {isLoading ? "Loading..." : "No domains found."}
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

function EditDomainForm({ domain, onSave, isPending }: {
  domain: Domain;
  onSave: (data: { credit_cost: number; description: string }) => void;
  isPending: boolean;
}) {
  const [cost, setCost] = useState(String(domain.credit_cost));
  const [desc, setDesc] = useState(domain.description);

  return (
    <>
      <div className="space-y-4 py-4">
        <div className="space-y-2">
          <Label>Credit Cost</Label>
          <Input type="number" min="1" value={cost} onChange={(e) => setCost(e.target.value)} />
        </div>
        <div className="space-y-2">
          <Label>Description</Label>
          <Textarea value={desc} onChange={(e) => setDesc(e.target.value)} />
        </div>
      </div>
      <DialogFooter>
        <Button onClick={() => onSave({ credit_cost: parseInt(cost) || 1, description: desc })} disabled={isPending}>
          {isPending ? "Saving..." : "Save"}
        </Button>
      </DialogFooter>
    </>
  );
}
