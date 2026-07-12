import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Domain } from "@/types";

interface SubdomainCardProps {
  domain: Domain;
  onClaim: (domain: Domain) => void;
}

export function SubdomainCard({ domain, onClaim }: SubdomainCardProps) {
  const { t } = useTranslation();

  return (
    <Card className="group transition-shadow hover:shadow-md">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{domain.name}</CardTitle>
          <Badge variant="secondary">{t("domains.creditCost", { count: domain.credit_cost })}</Badge>
        </div>
        {domain.description && (
          <div className="text-sm text-muted-foreground whitespace-pre-line break-words">
            {domain.description}
          </div>
        )}
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          <code className="text-sm text-muted-foreground">*.{domain.name}</code>
          <Button size="sm" onClick={() => onClaim(domain)}>
            {t("domains.claim")}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
