import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "./button";

interface PaginatorProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

export function Paginator({ page, totalPages, onPageChange }: PaginatorProps) {
  if (totalPages <= 1) return null;

  const prev = () => onPageChange(Math.max(1, page - 1));
  const next = () => onPageChange(Math.min(totalPages, page + 1));

  return (
    <div className="flex items-center gap-2">
      <Button variant="outline" size="sm" disabled={page <= 1} onClick={prev}>
        <ChevronLeft className="h-4 w-4" />
        <span className="sr-only">Previous</span>
      </Button>
      <span className="text-sm text-muted-foreground">
        {page} / {totalPages}
      </span>
      <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={next}>
        <ChevronRight className="h-4 w-4" />
        <span className="sr-only">Next</span>
      </Button>
    </div>
  );
}
