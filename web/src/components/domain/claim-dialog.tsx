import { useState } from "react";
import { useTranslation, Trans } from "react-i18next";
import { useNavigate } from "react-router-dom";
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
import { useAuth } from "@/hooks/use-auth";

interface ClaimDialogProps {
  domain: Domain | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ClaimDialog({ domain, open, onOpenChange }: ClaimDialogProps) {
  const [name, setName] = useState("");
  const claim = useClaimSubdomain();
  const { data: subdomainSettings } = useSubdomainSettings();
  const { user } = useAuth();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const minLength = subdomainSettings?.min_length ?? 1;
  const maxLength = subdomainSettings?.max_length ?? 63;
  const normalizedName = name.trim();
  const hasLengthError =
    normalizedName.length > 0 &&
    (normalizedName.length < minLength || normalizedName.length > maxLength);

  // 实名认证检查：域名开启实名要求且用户未认证时，禁用提交并显示警告。
  const requireRealname = !!domain?.require_realname;
  const userVerified = user?.realname_status === "verified";
  const blockedByRealname = requireRealname && !userVerified;

  const handleSubmit = () => {
    if (!domain || !normalizedName || hasLengthError || blockedByRealname) return;
    claim.mutate(
      { domain_id: domain.id, name: normalizedName.toLowerCase() },
      {
        onSuccess: (res) => {
          setName("");
          onOpenChange(false);
          navigate(`/subdomains/${res.data.id}`);
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
          {requireRealname && (
            <div className={`rounded-md border p-3 text-sm ${userVerified ? "border-emerald-500/40 bg-emerald-500/5" : "border-amber-500/40 bg-amber-500/5"}`}>
              {userVerified ? (
                <p className="text-emerald-700 dark:text-emerald-400">
                  {t("claimDialog.realnameVerified")}
                </p>
              ) : (
                <div className="space-y-1">
                  <p className="font-medium text-amber-700 dark:text-amber-400">
                    {t("claimDialog.realnameRequiredTitle")}
                  </p>
                  <p className="text-amber-700/80 dark:text-amber-400/80">
                    {t("claimDialog.realnameRequiredHint")}{" "}
                    <button
                      type="button"
                      className="underline font-medium"
                      onClick={() => {
                        onOpenChange(false);
                        navigate("/account/security");
                      }}
                    >
                      {t("claimDialog.realnameGoVerify")}
                    </button>
                  </p>
                </div>
              )}
            </div>
          )}
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
          <Button onClick={handleSubmit} disabled={!normalizedName || hasLengthError || claim.isPending || blockedByRealname} data-dialog-primary="true">
            {claim.isPending ? t("claimDialog.claiming") : t("claimDialog.claimFor", { count: domain.credit_cost, cost: domain.credit_cost })}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
