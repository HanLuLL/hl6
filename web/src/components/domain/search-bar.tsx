import { useState, useEffect, useRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { cn } from "@/lib/utils";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { ChevronDown, Check, X, ArrowRight, Loader2 } from "lucide-react";

type CheckState = "idle" | "checking" | "available" | "taken" | "error";

const TYPE_CLASS = "text-2xl sm:text-3xl font-semibold tracking-tight";

function sanitize(v: string): string {
  return v.toLowerCase().replace(/[^a-z0-9-]/g, "");
}

export function DomainSearchBar() {
  const { t } = useTranslation();
  const { isAuthenticated, signIn } = useAuth();

  const [prefix, setPrefix] = useState("");
  const [selectedDomainId, setSelectedDomainId] = useState<number | null>(null);
  const [popoverOpen, setPopoverOpen] = useState(false);
  const [checkState, setCheckState] = useState<CheckState>("idle");
  const [displayIndex, setDisplayIndex] = useState(0);
  const [isTransitioning, setIsTransitioning] = useState(false);
  const [manualSelect, setManualSelect] = useState(false);
  const [focused, setFocused] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const { data: domains } = useQuery({
    queryKey: ["public-domains"],
    queryFn: async () => {
      const res = await api.listPublicDomains();
      return res.data;
    },
    staleTime: 60_000,
  });

  const activeDomains = domains ?? [];

  // 后缀自动轮播 —— 一旦用户开始输入或手动选择，即锁定当前后缀
  useEffect(() => {
    if (manualSelect || prefix || activeDomains.length <= 1) return;
    intervalRef.current = setInterval(() => {
      setIsTransitioning(true);
      setTimeout(() => {
        setDisplayIndex((prev) => (prev + 1) % activeDomains.length);
        setIsTransitioning(false);
      }, 250);
    }, 2800);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [activeDomains.length, manualSelect, prefix]);

  const currentDomain = manualSelect
    ? activeDomains.find((d) => d.id === selectedDomainId)
    : activeDomains[displayIndex];

  const runCheck = useCallback(async (name: string, domainId: number) => {
    setCheckState("checking");
    try {
      const res = await api.checkSubdomainAvailable(name, domainId);
      setCheckState(res.data.available ? "available" : "taken");
    } catch {
      setCheckState("error");
    }
  }, []);

  // 边打字边校验（debounce），实时感。清空/切换后的即时“idle”态由事件处理器负责，
  // 这里只在异步回调里 setState，避免 effect 内同步 setState 触发级联渲染。
  const currentDomainId = currentDomain?.id;
  useEffect(() => {
    const name = sanitize(prefix).trim();
    if (!name || !currentDomainId) return;
    let cancelled = false;
    const timer = setTimeout(async () => {
      setCheckState("checking");
      try {
        const res = await api.checkSubdomainAvailable(name, currentDomainId);
        if (!cancelled) setCheckState(res.data.available ? "available" : "taken");
      } catch {
        if (!cancelled) setCheckState("error");
      }
    }, 500);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [prefix, currentDomainId]);

  const handleSelectDomain = useCallback((domainId: number) => {
    setSelectedDomainId(domainId);
    setManualSelect(true);
    setPopoverOpen(false);
    setCheckState("idle");
  }, []);

  const handleImmediateCheck = useCallback(() => {
    const name = sanitize(prefix).trim();
    if (name && currentDomainId) runCheck(name, currentDomainId);
  }, [prefix, currentDomainId, runCheck]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") handleImmediateCheck();
    },
    [handleImmediateCheck],
  );

  const displaySuffix = currentDomain?.name ?? "—";
  const measure = prefix || t("landing.searchPlaceholder");

  const underlineColor =
    checkState === "available"
      ? "border-emerald-500"
      : checkState === "taken" || checkState === "error"
        ? "border-destructive"
        : focused
          ? "border-brand"
          : "border-border";

  return (
    <div className="w-full max-w-xl">


      {/* 内容化大标题：整行无框，唯一的“控件”是底部随状态变色的下划线 */}
      <div
        className={cn(
          "flex items-baseline gap-1 border-b-2 pb-2.5 transition-colors duration-300",
          underlineColor,
        )}
        onClick={() => inputRef.current?.focus()}
      >
        {/* 前缀输入 —— 透明背景、宽度随内容 */}
        <span className="relative inline-flex min-w-[1ch] max-w-[62%] items-baseline">
          <span aria-hidden className={cn(TYPE_CLASS, "invisible whitespace-pre")}>
            {measure}
          </span>
          <input
            ref={inputRef}
            value={prefix}
            onChange={(e) => {
              setPrefix(sanitize(e.target.value));
              setCheckState("idle");
            }}
            onKeyDown={handleKeyDown}
            onFocus={() => setFocused(true)}
            onBlur={() => setFocused(false)}
            placeholder={t("landing.searchPlaceholder")}
            spellCheck={false}
            autoComplete="off"
            aria-label={t("landing.searchKicker")}
            className={cn(
              TYPE_CLASS,
              "absolute inset-0 w-full border-0 bg-transparent p-0 text-foreground caret-brand outline-none placeholder:text-muted-foreground/40",
            )}
          />
        </span>

        <div className="ml-auto flex items-center gap-1">
          {/* 后缀 —— inline 可点切换 */}
          <Popover open={popoverOpen} onOpenChange={setPopoverOpen}>
            <PopoverTrigger asChild>
              <button
                type="button"
                onClick={(e) => e.stopPropagation()}
                className={cn(
                  TYPE_CLASS,
                  "group inline-flex shrink-0 items-baseline gap-1 text-muted-foreground transition-colors hover:text-foreground",
                )}
              >
                <span
                  className={cn(
                    "transition-opacity duration-200",
                    isTransitioning ? "opacity-0" : "opacity-100",
                  )}
                >
                  .{displaySuffix}
                </span>
                <ChevronDown className="h-4 w-4 shrink-0 translate-y-[-0.15em] text-muted-foreground/50 transition-transform group-hover:text-muted-foreground" />
              </button>
            </PopoverTrigger>
            <PopoverContent
              align="start"
              className="w-56 p-1"
              onOpenAutoFocus={(e) => e.preventDefault()}
            >

              {activeDomains.map((d) => (
                <button
                  key={d.id}
                  type="button"
                  onClick={() => handleSelectDomain(d.id)}
                  className={cn(
                    "w-full rounded-md px-2 py-1.5 text-left text-sm transition-colors",
                    currentDomain?.id === d.id
                      ? "bg-brand-muted font-medium text-brand"
                      : "hover:bg-muted",
                  )}
                >
                  {d.name}
                </button>
              ))}
            </PopoverContent>
          </Popover>


        </div>
      </div>

      {/* 结果 / 提示 —— 内联无卡片 */}
      <div className="mt-3 min-h-[1.5rem] text-sm">
        {checkState === "available" && (
          <div className="flex flex-wrap items-center gap-2">
            <span className="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400">
              <Check className="h-4 w-4" />
              <span className="font-medium">{t("landing.searchAvailable")}</span>
            </span>
           {/**  <code className="rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
              {prefix.trim()}.{displaySuffix}
            </code>*/}
            {isAuthenticated ? (
              <Link
                to="/domains"
                className="ml-auto inline-flex items-center gap-1 text-xs font-medium text-brand hover:underline"
              >
                {t("landing.searchClaim")}
                <ArrowRight className="h-3 w-3" />
              </Link>
            ) : (
              <button
                type="button"
                onClick={() => signIn()}
                className="ml-auto inline-flex items-center gap-1 text-xs font-medium text-brand hover:underline"
              >
                {t("landing.searchLogin")}
                <ArrowRight className="h-3 w-3" />
              </button>
            )}
          </div>
        )}

        {checkState === "taken" && (
          <div className="flex items-center gap-1.5 text-destructive">
            <X className="h-4 w-4" />
            <span>{t("landing.searchTaken")}</span>
           {/** <code className="ml-1 rounded bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
              {prefix.trim()}.{displaySuffix}
            </code> */}
          </div>
        )}


      </div>
    </div>
  );
}
