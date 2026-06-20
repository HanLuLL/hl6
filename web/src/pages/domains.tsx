import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useDomains } from "@/hooks/use-subdomains";
import { SubdomainCard } from "@/components/domain/subdomain-card";
import { ClaimDialog } from "@/components/domain/claim-dialog";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { Domain } from "@/types";
import { useDocumentTitle } from "@/hooks/use-document-title";
import { Globe } from "lucide-react";

export default function DomainsPage() {
  const { data: domains, isLoading } = useDomains();
  const [claimDomain, setClaimDomain] = useState<Domain | null>(null);
  const { t } = useTranslation();
  useDocumentTitle(t("domains.title"));

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{t("domains.title")}</h1>
          <p className="mt-1 text-sm text-muted-foreground">{t("domains.subtitle")}</p>
        </div>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <Card key={i}>
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <Skeleton className="h-5 w-32" />
                  <Skeleton className="h-5 w-16" />
                </div>
                <Skeleton className="h-4 w-full mt-2" />
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between">
                  <Skeleton className="h-4 w-24" />
                  <Skeleton className="h-8 w-16" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : !domains || domains.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-center">
          <div className="rounded-xl bg-muted p-4 mb-4">
            <Globe className="h-8 w-8 text-muted-foreground/50" />
          </div>
          <p className="text-muted-foreground text-sm">{t("domains.noDomains")}</p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {domains.map((domain) => (
            <div key={domain.id} className="hover:-translate-y-px transition-transform duration-150">
              <SubdomainCard domain={domain} onClaim={setClaimDomain} />
            </div>
          ))}
        </div>
      )}

      <ClaimDialog
        domain={claimDomain}
        open={!!claimDomain}
        onOpenChange={(open) => !open && setClaimDomain(null)}
      />
    </div>
  );
}
