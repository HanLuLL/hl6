import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useSubdomain, useReleaseSubdomain } from "@/hooks/use-subdomains";
import { useDNSRecords } from "@/hooks/use-dns-records";
import { RecordTable } from "@/components/dns/record-table";
import { RecordForm } from "@/components/dns/record-form";
import { toast } from "sonner";

export default function SubdomainDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const subdomainId = parseInt(id || "0");
  const { data: subdomain, isLoading } = useSubdomain(subdomainId);
  const { data: records } = useDNSRecords(subdomainId);
  const release = useReleaseSubdomain();
  const [showAddRecord, setShowAddRecord] = useState(false);

  const handleRelease = () => {
    if (!subdomain) return;
    if (!confirm(`Release ${subdomain.fqdn}? All DNS records will be deleted.`)) return;
    release.mutate(subdomain.id, {
      onSuccess: () => {
        toast.success(`Released ${subdomain.fqdn}`);
        navigate("/subdomains");
      },
      onError: (err) => toast.error(err.message),
    });
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
      </div>
    );
  }

  if (!subdomain) {
    return (
      <div className="flex flex-col items-center justify-center py-16">
        <p className="text-muted-foreground">Subdomain not found</p>
        <Button className="mt-4" asChild>
          <Link to="/subdomains">Back to list</Link>
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link to="/subdomains" className="hover:text-foreground">My Subdomains</Link>
        <span>/</span>
        <span className="text-foreground">{subdomain.fqdn}</span>
      </div>

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{subdomain.fqdn}</h1>
          <div className="flex items-center gap-2 mt-1">
            <Badge variant="outline">{subdomain.domain.name}</Badge>
            <span className="text-sm text-muted-foreground">
              Claimed {new Date(subdomain.created_at).toLocaleDateString()}
            </span>
          </div>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => setShowAddRecord(true)}>Add Record</Button>
          <Button variant="destructive" onClick={handleRelease} disabled={release.isPending}>
            Release
          </Button>
        </div>
      </div>

      {/* Records */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-lg">DNS Records</CardTitle>
          <Badge variant="secondary">{records?.length ?? 0} total</Badge>
        </CardHeader>
        <CardContent>
          <RecordTable subdomainId={subdomainId} records={records || []} />
        </CardContent>
      </Card>

      <RecordForm
        subdomainId={subdomainId}
        open={showAddRecord}
        onOpenChange={setShowAddRecord}
      />
    </div>
  );
}
