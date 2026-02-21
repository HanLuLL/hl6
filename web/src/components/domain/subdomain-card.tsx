import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import type { Domain } from "@/types";

interface SubdomainCardProps {
  domain: Domain;
  onClaim: (domain: Domain) => void;
}

export function SubdomainCard({ domain, onClaim }: SubdomainCardProps) {
  return (
    <Card className="group transition-shadow hover:shadow-md">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">{domain.name}</CardTitle>
          <Badge variant="secondary">{domain.credit_cost} credit{domain.credit_cost > 1 ? "s" : ""}</Badge>
        </div>
        {domain.description && (
          <CardDescription>{domain.description}</CardDescription>
        )}
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          <code className="text-sm text-muted-foreground">*.{domain.name}</code>
          <Button size="sm" onClick={() => onClaim(domain)}>
            Claim
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
