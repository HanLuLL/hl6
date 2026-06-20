import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { Input } from "@/components/ui/input";
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from "@/components/ui/select";

export function AuditTableFilters({
  search,
  onSearchChange,
  searchPlaceholder,
  domainFilter,
  onDomainFilterChange,
  groupFilter,
  onGroupFilterChange,
  children,
}: {
  search: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder: string;
  domainFilter: string;
  onDomainFilterChange: (value: string) => void;
  groupFilter: string;
  onGroupFilterChange: (value: string) => void;
  children?: React.ReactNode;
}) {
  const { t } = useTranslation();

  const { data: domains } = useQuery({
    queryKey: ["admin-domains-list"],
    queryFn: async () => (await api.adminListDomainsFull()).data,
    staleTime: 60_000,
  });

  const { data: groups } = useQuery({
    queryKey: ["admin-groups"],
    queryFn: async () => (await api.adminListGroups()).data,
    staleTime: 60_000,
  });

  return (
    <div className="flex min-w-0 flex-nowrap items-center gap-2 overflow-x-auto pb-0.5">
      <Input
        className="h-9 w-52 shrink-0"
        placeholder={searchPlaceholder}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
      />
      <Select
        value={domainFilter || "all"}
        onValueChange={(v) => onDomainFilterChange(v === "all" ? "" : v)}
      >
        <SelectTrigger className="h-9 w-40 shrink-0">
          <SelectValue placeholder={t("adminDnsRecords.filterByDomain")} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">{t("adminDnsRecords.filterByDomain")}</SelectItem>
          {domains?.map((d) => (
            <SelectItem key={d.id} value={String(d.id)}>{d.name}</SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        value={groupFilter || "all"}
        onValueChange={(v) => onGroupFilterChange(v === "all" ? "" : v)}
      >
        <SelectTrigger className="h-9 w-40 shrink-0">
          <SelectValue placeholder={t("adminDnsRecords.filterByGroup")} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">{t("adminDnsRecords.filterByGroup")}</SelectItem>
          {groups?.map((g) => (
            <SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>
          ))}
        </SelectContent>
      </Select>
      {children}
    </div>
  );
}
