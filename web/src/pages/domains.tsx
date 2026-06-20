import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useDomains } from "@/hooks/use-subdomains";
import { SubdomainCard } from "@/components/domain/subdomain-card";
import { ClaimDialog } from "@/components/domain/claim-dialog";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { Domain } from "@/types";
import { useDocumentTitle } from "@/hooks/use-document-title";

export default function DomainsPage() {
  const { data: domains, isLoading } = useDomains();
  const [claimDomain, setClaimDomain] = useState<Domain | null>(null);
  const { t } = useTranslation();
  useDocumentTitle(t("domains.title"));

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("domains.title")}</h1>
        <p className="text-muted-foreground">{t("domains.subtitle")}</p>
      </div>

      {isLoading ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <Card key={i} className="transition-shadow">
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
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" className="text-muted-foreground/50 mb-4"><path d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"/></svg>
          <p className="text-muted-foreground">{t("domains.noDomains")}</p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {domains.map((domain) => (
            <SubdomainCard key={domain.id} domain={domain} onClaim={setClaimDomain} />
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
