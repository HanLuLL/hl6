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
import type { Domain } from "@/types";
import { useClaimSubdomain } from "@/hooks/use-subdomains";
import { toast } from "sonner";

interface ClaimDialogProps {
  domain: Domain | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ClaimDialog({ domain, open, onOpenChange }: ClaimDialogProps) {
  const [name, setName] = useState("");
  const claim = useClaimSubdomain();

  const handleSubmit = () => {
    if (!domain || !name.trim()) return;
    claim.mutate(
      { domain_id: domain.id, name: name.trim().toLowerCase() },
      {
        onSuccess: () => {
          toast.success(`Successfully claimed ${name}.${domain.name}`);
          setName("");
          onOpenChange(false);
        },
        onError: (err) => {
          toast.error(err.message);
        },
      }
    );
  };

  if (!domain) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Claim Subdomain</DialogTitle>
          <DialogDescription>
            Choose a subdomain name for <strong>{domain.name}</strong>. This will cost{" "}
            <strong>{domain.credit_cost}</strong> credit{domain.credit_cost > 1 ? "s" : ""}.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="subdomain">Subdomain name</Label>
            <div className="flex items-center gap-2">
              <Input
                id="subdomain"
                placeholder="mysite"
                value={name}
                onChange={(e) => setName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
              />
              <span className="text-sm text-muted-foreground whitespace-nowrap">.{domain.name}</span>
            </div>
            {name && (
              <p className="text-xs text-muted-foreground">
                Result: <strong>{name}.{domain.name}</strong>
              </p>
            )}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSubmit} disabled={!name.trim() || claim.isPending}>
            {claim.isPending ? "Claiming..." : `Claim for ${domain.credit_cost} credit${domain.credit_cost > 1 ? "s" : ""}`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
