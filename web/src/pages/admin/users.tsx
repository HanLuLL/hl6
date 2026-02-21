import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { useQuery, useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { toast } from "sonner";

export default function AdminUsersPage() {
  const [page, setPage] = useState(1);
  const { t } = useTranslation();
  const { data, isLoading } = useQuery({
    queryKey: ["admin-users", page],
    queryFn: async () => {
      const res = await api.adminListUsers(page, 20);
      return { users: res.data, total: res.total };
    },
  });

  const [grantUserId, setGrantUserId] = useState<number | null>(null);
  const [grantAmount, setGrantAmount] = useState("10");
  const [grantDesc, setGrantDesc] = useState("");

  const grantMutation = useMutation({
    mutationFn: api.adminGrantCredits,
    onSuccess: (res) => {
      toast.success(t("adminUsers.grantSuccess", { amount: res.data.granted, balance: res.data.balance }));
      setGrantUserId(null);
    },
    onError: (err) => toast.error(err.message),
  });

  if (isLoading) {
    return <div className="flex items-center justify-center py-12"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" /></div>;
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">{t("adminUsers.title")}</h1>
        <p className="text-muted-foreground">{t("adminUsers.subtitle")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium text-muted-foreground">
            {t("adminUsers.totalUsers", { count: data?.total ?? 0 })}
          </CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("adminUsers.name")}</TableHead>
                <TableHead>{t("adminUsers.email")}</TableHead>
                <TableHead>{t("adminUsers.role")}</TableHead>
                <TableHead>{t("adminUsers.joined")}</TableHead>
                <TableHead className="text-right">{t("adminUsers.actions")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data?.users?.map((user) => (
                <TableRow key={user.id}>
                  <TableCell className="font-medium">{user.name}</TableCell>
                  <TableCell className="text-muted-foreground">{user.email}</TableCell>
                  <TableCell>
                    <Badge variant={user.role === "admin" ? "default" : "secondary"}>{user.role}</Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {new Date(user.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button variant="ghost" size="sm" onClick={() => setGrantUserId(user.id)}>
                      {t("adminUsers.grantCredits")}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {data && data.total > 20 && (
        <div className="flex justify-center gap-2">
          <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>{t("common.previous")}</Button>
          <span className="flex items-center text-sm text-muted-foreground">{t("common.page", { page })}</span>
          <Button variant="outline" size="sm" disabled={page >= Math.ceil(data.total / 20)} onClick={() => setPage((p) => p + 1)}>{t("common.next")}</Button>
        </div>
      )}

      <Dialog open={grantUserId !== null} onOpenChange={(open) => !open && setGrantUserId(null)}>
        <DialogContent>
          <DialogHeader><DialogTitle>{t("adminUsers.grantCredits")}</DialogTitle></DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>{t("adminUsers.amount")}</Label>
              <Input type="number" min="1" value={grantAmount} onChange={(e) => setGrantAmount(e.target.value)} />
            </div>
            <div className="space-y-2">
              <Label>{t("adminUsers.descriptionOptional")}</Label>
              <Input value={grantDesc} onChange={(e) => setGrantDesc(e.target.value)} placeholder={t("adminUsers.adminGrant")} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setGrantUserId(null)}>{t("common.cancel")}</Button>
            <Button
              onClick={() => grantMutation.mutate({
                user_id: grantUserId!,
                amount: parseInt(grantAmount) || 1,
                description: grantDesc || t("adminUsers.adminGrant"),
              })}
              disabled={grantMutation.isPending}
            >
              {grantMutation.isPending ? t("adminUsers.granting") : t("credits.grant")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
