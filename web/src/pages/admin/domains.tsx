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
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { toast } from "sonner";
import type { Domain } from "@/types";

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
  const [form, setForm] = useState({ name: "", cloudflare_zone_id: "", credit_cost: "1", description: "" });

  const createMutation = useMutation({
    mutationFn: api.adminCreateDomain,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-domains"] });
      toast.success("Domain created");
      setShowAdd(false);
      setForm({ name: "", cloudflare_zone_id: "", credit_cost: "1", description: "" });
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
      <Dialog open={showAdd} onOpenChange={setShowAdd}>
        <DialogContent>
          <DialogHeader><DialogTitle>Add Domain</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Domain Name</Label>
              <Input placeholder="example.com" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label>Cloudflare Zone ID</Label>
              <Input placeholder="abc123..." value={form.cloudflare_zone_id} onChange={(e) => setForm({ ...form, cloudflare_zone_id: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label>Credit Cost</Label>
              <Input type="number" min="1" value={form.credit_cost} onChange={(e) => setForm({ ...form, credit_cost: e.target.value })} />
            </div>
            <div className="space-y-2">
              <Label>Description</Label>
              <Textarea placeholder="Optional description" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAdd(false)}>Cancel</Button>
            <Button
              onClick={() => createMutation.mutate({
                name: form.name,
                cloudflare_zone_id: form.cloudflare_zone_id,
                credit_cost: parseInt(form.credit_cost) || 1,
                description: form.description,
              })}
              disabled={!form.name || !form.cloudflare_zone_id || createMutation.isPending}
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
