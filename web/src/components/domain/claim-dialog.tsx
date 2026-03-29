import { useState } from "react";
import { useTranslation, Trans } from "react-i18next";
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
import { useClaimSubdomain, useSubdomainSettings } from "@/hooks/use-subdomains";

interface ClaimDialogProps {
  domain: Domain | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ClaimDialog({ domain, open, onOpenChange }: ClaimDialogProps) {
  const [name, setName] = useState("");
  const claim = useClaimSubdomain();
  const { data: subdomainSettings } = useSubdomainSettings();
  const { t } = useTranslation();
  const minLength = subdomainSettings?.min_length ?? 1;
  const maxLength = subdomainSettings?.max_length ?? 63;
  const normalizedName = name.trim();
  const hasLengthError =
    normalizedName.length > 0 &&
    (normalizedName.length < minLength || normalizedName.length > maxLength);

  const handleSubmit = () => {
    if (!domain || !normalizedName || hasLengthError) return;
    claim.mutate(
      { domain_id: domain.id, name: normalizedName.toLowerCase() },
      {
        onSuccess: () => {
          setName("");
          onOpenChange(false);
        },
      }
    );
  };

  if (!domain) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("claimDialog.title")}</DialogTitle>
          <DialogDescription>
            <Trans
              i18nKey="claimDialog.description"
              count={domain.credit_cost}
              values={{ domain: domain.name, cost: domain.credit_cost }}
              components={{ strong: <strong /> }}
            />
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="subdomain">{t("claimDialog.subdomainName")}</Label>
            <div className="flex items-center gap-2">
              <Input
                id="subdomain"
                placeholder={t("claimDialog.placeholder")}
                value={name}
                onChange={(e) => setName(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                maxLength={maxLength}
                aria-invalid={hasLengthError}
                onKeyDown={(e) => {
                  if (e.key !== "Enter") return;
                  if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
                  handleSubmit();
                }}
                required
              />
              <span className="text-sm text-muted-foreground whitespace-nowrap">.{domain.name}</span>
            </div>
            <p className="text-xs text-muted-foreground">
              {t("claimDialog.lengthHint", { min: minLength, max: maxLength })}
            </p>
            {hasLengthError && (
              <p className="text-xs text-destructive">
                {t("claimDialog.invalidLength", { min: minLength, max: maxLength })}
              </p>
            )}
            {normalizedName && (
              <p className="text-xs text-muted-foreground">
                <Trans
                  i18nKey="claimDialog.result"
                  values={{ fqdn: `${normalizedName}.${domain.name}` }}
                  components={{ strong: <strong /> }}
                />
              </p>
            )}
            {domain.credit_cost < 0 && (
              <p className="text-xs text-green-600 dark:text-green-400">
                {t("claimDialog.earnCredits", { gain: -domain.credit_cost })}
              </p>
            )}
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={!normalizedName || hasLengthError || claim.isPending} data-dialog-primary="true">
            {claim.isPending ? t("claimDialog.claiming") : t("claimDialog.claimFor", { count: domain.credit_cost, cost: domain.credit_cost })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
