import { useState } from "react";
import { useTranslation } from "react-i18next";
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
import { getErrorMessage } from "@/lib/api";

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
  const { t } = useTranslation();

  const create = useCreateRecord(subdomainId);
  const update = useUpdateRecord(subdomainId);
  const isEdit = !!record;

  const handleSubmit = () => {
    if (!content.trim()) return;
    const data = {
      content: content.trim(),
      ttl: parseInt(ttl) || 1,
      proxied: type === "TXT" ? false : proxied,
    };

    if (isEdit && record) {
      update.mutate(
        { recordId: record.id, ...data },
        {
          onSuccess: () => {
            toast.success(t("recordForm.recordUpdated"));
            onOpenChange(false);
          },
          onError: (err) => toast.error(getErrorMessage(err, t)),
        }
      );
    } else {
      create.mutate(
        { type, ...data },
        {
          onSuccess: () => {
            toast.success(t("recordForm.recordCreated"));
            setContent("");
            onOpenChange(false);
          },
          onError: (err) => toast.error(getErrorMessage(err, t)),
        }
      );
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? t("recordForm.editTitle") : t("recordForm.addTitle")}</DialogTitle>
          <DialogDescription>
            {isEdit ? t("recordForm.editDesc") : t("recordForm.addDesc")}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          {!isEdit && (
            <div className="space-y-2">
              <Label>{t("recordForm.type")}</Label>
              <Select value={type} onValueChange={setType}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="A">{t("recordForm.typeA")}</SelectItem>
                  <SelectItem value="AAAA">{t("recordForm.typeAAAA")}</SelectItem>
                  <SelectItem value="CNAME">{t("recordForm.typeCNAME")}</SelectItem>
                  <SelectItem value="TXT">{t("recordForm.typeTXT")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}
          <div className="space-y-2">
            <Label>{t("recordForm.content")}</Label>
            <Input
              placeholder={type === "A" ? "1.2.3.4" : type === "AAAA" ? "2001:db8::1" : type === "TXT" ? "v=spf1 include:example.com ~all" : "example.com"}
              value={content}
              onChange={(e) => setContent(e.target.value)}
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>{t("recordForm.ttl")}</Label>
              <Select value={ttl} onValueChange={setTtl}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">{t("common.auto")}</SelectItem>
                  <SelectItem value="60">{t("recordForm.ttl1min")}</SelectItem>
                  <SelectItem value="300">{t("recordForm.ttl5min")}</SelectItem>
                  <SelectItem value="3600">{t("recordForm.ttl1hour")}</SelectItem>
                  <SelectItem value="86400">{t("recordForm.ttl1day")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>{t("recordForm.proxied")}</Label>
              <div className="flex items-center h-9">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={type === "TXT" ? false : proxied}
                    onChange={(e) => setProxied(e.target.checked)}
                    disabled={type === "TXT"}
                    className="rounded"
                  />
                  <span className="text-sm">{proxied ? t("common.on") : t("common.off")}</span>
                </label>
              </div>
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={!content.trim() || create.isPending || update.isPending}>
            {create.isPending || update.isPending ? t("common.saving") : isEdit ? t("recordForm.update") : t("common.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
