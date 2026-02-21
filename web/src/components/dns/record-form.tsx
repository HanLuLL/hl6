import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useCreateRecord, useUpdateRecord } from "@/hooks/use-dns-records";
import type { DNSRecord } from "@/types";
import { toast } from "sonner";

interface RecordFormProps {
  subdomainId: number;
  record?: DNSRecord | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function RecordForm({ subdomainId, record, open, onOpenChange }: RecordFormProps) {
  const [type, setType] = useState<string>(record?.type || "A");
  const [content, setContent] = useState(record?.content || "");
  const [ttl, setTtl] = useState(String(record?.ttl || 1));
  const [proxied, setProxied] = useState(record?.proxied || false);

  const create = useCreateRecord(subdomainId);
  const update = useUpdateRecord(subdomainId);
  const isEdit = !!record;

  const handleSubmit = () => {
    if (!content.trim()) return;
    const data = {
      content: content.trim(),
      ttl: parseInt(ttl) || 1,
      proxied,
    };

    if (isEdit && record) {
      update.mutate(
        { recordId: record.id, ...data },
        {
          onSuccess: () => {
            toast.success("Record updated");
            onOpenChange(false);
          },
          onError: (err) => toast.error(err.message),
        }
      );
    } else {
      create.mutate(
        { type, ...data },
        {
          onSuccess: () => {
            toast.success("Record created");
            setContent("");
            onOpenChange(false);
          },
          onError: (err) => toast.error(err.message),
        }
      );
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit" : "Add"} DNS Record</DialogTitle>
          <DialogDescription>
            {isEdit ? "Update the record values." : "Create a new DNS record for this subdomain."}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          {!isEdit && (
            <div className="space-y-2">
              <Label>Type</Label>
              <Select value={type} onValueChange={setType}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="A">A (IPv4)</SelectItem>
                  <SelectItem value="AAAA">AAAA (IPv6)</SelectItem>
                  <SelectItem value="CNAME">CNAME</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}
          <div className="space-y-2">
            <Label>Content</Label>
            <Input
              placeholder={type === "A" ? "1.2.3.4" : type === "AAAA" ? "2001:db8::1" : "example.com"}
              value={content}
              onChange={(e) => setContent(e.target.value)}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>TTL</Label>
              <Select value={ttl} onValueChange={setTtl}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">Auto</SelectItem>
                  <SelectItem value="60">1 min</SelectItem>
                  <SelectItem value="300">5 min</SelectItem>
                  <SelectItem value="3600">1 hour</SelectItem>
                  <SelectItem value="86400">1 day</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Proxied</Label>
              <div className="flex items-center h-9">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={proxied}
                    onChange={(e) => setProxied(e.target.checked)}
                    className="rounded"
                  />
                  <span className="text-sm">{proxied ? "On" : "Off"}</span>
                </label>
              </div>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={!content.trim() || create.isPending || update.isPending}>
            {create.isPending || update.isPending ? "Saving..." : isEdit ? "Update" : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
