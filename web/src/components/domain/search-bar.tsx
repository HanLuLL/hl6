import { useState, useEffect, useRef, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { cn } from "@/lib/utils";
import { ChevronDown, Check, X, ArrowRight } from "lucide-react";

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
  const [dropdownOpen, setDropdownOpen] = useState(false);
  const [dropdownVisible, setDropdownVisible] = useState(false); // stays true during close animation
  const [checkState, setCheckState] = useState<CheckState>("idle");
  const [displayIndex, setDisplayIndex] = useState(0);
  const [manualSelect, setManualSelect] = useState(false);
  const [focused, setFocused] = useState(false);

  // Simple fade transition for auto-switch
  const [isTransitioning, setIsTransitioning] = useState(false);

  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const transitionTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Close dropdown with fade-out animation
  const closeDropdown = useCallback(() => {
    setDropdownOpen(false);
    if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    closeTimerRef.current = setTimeout(() => {
      setDropdownVisible(false);
    }, 300);
  }, []);

  const { data: domains } = useQuery({
    queryKey: ["public-domains"],
    queryFn: async () => {
      const res = await api.listPublicDomains();
      return res.data;
    },
    staleTime: 60_000,
  });

  const activeDomains = domains ?? [];

  // Auto-switch with simple opacity fade transition
  useEffect(() => {
    if (manualSelect || prefix || activeDomains.length <= 1) return;

    intervalRef.current = setInterval(() => {
      // Fade out
      setIsTransitioning(true);

      // At midpoint, swap text then fade in
      transitionTimerRef.current = setTimeout(() => {
        setDisplayIndex((prev) => (prev + 1) % activeDomains.length);
        setIsTransitioning(false);
      }, 400);
    }, 2800);

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
      if (transitionTimerRef.current) clearTimeout(transitionTimerRef.current);
      setIsTransitioning(false);
    };
  }, [activeDomains.length, manualSelect, prefix]);

  // Cleanup close timer on unmount
  useEffect(() => {
    return () => {
      if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    };
  }, []);

  // Close dropdown on outside click / Escape
  useEffect(() => {
    if (!dropdownOpen) return;

    const onMouseDown = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        closeDropdown();
      }
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") closeDropdown();
    };

    document.addEventListener("mousedown", onMouseDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("mousedown", onMouseDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [dropdownOpen]);

  const currentDomain = manualSelect
    ? activeDomains.find((d) => d.id === selectedDomainId)
    : activeDomains[displayIndex];

  const currentDomainId = currentDomain?.id;

  // Debounced domain check
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
    setCheckState("idle");
    closeDropdown();
  }, [closeDropdown]);

  const handleToggleDropdown = useCallback(() => {
    if (activeDomains.length <= 1) return;
    if (dropdownVisible) {
      closeDropdown();
    } else {
      // Opening dropdown → stop auto-switch permanently
      setManualSelect(true);
      if (!manualSelect && currentDomain) {
        setSelectedDomainId(currentDomain.id);
      }
      setDropdownVisible(true);
      setDropdownOpen(true);
    }
  }, [activeDomains.length, dropdownVisible, manualSelect, currentDomain, closeDropdown]);

  const handleImmediateCheck = useCallback(async () => {
    const name = sanitize(prefix).trim();
    if (!name || !currentDomainId) return;
    setCheckState("checking");
    try {
      const res = await api.checkSubdomainAvailable(name, currentDomainId);
      setCheckState(res.data.available ? "available" : "taken");
    } catch {
      setCheckState("error");
    }
  }, [prefix, currentDomainId]);

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
      {/* Global blur overlay when dropdown is open */}
      {dropdownVisible && (
        <div
          className={cn(
            "fixed inset-0 z-40 bg-background/30 backdrop-blur-md transition-opacity duration-300",
            dropdownOpen ? "opacity-100" : "opacity-0",
          )}
          aria-hidden
        />
      )}

      {/* Input row */}
      <div
        className={cn(
          "flex items-baseline gap-1 border-b-2 pb-2.5 transition-colors duration-300 relative",
          underlineColor,
          dropdownVisible && "z-50",
        )}
        onClick={() => inputRef.current?.focus()}
      >
        {/* Prefix input */}
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
              if (dropdownVisible) closeDropdown();
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

        {/* Domain suffix selector */}
        <div className="ml-auto flex items-center relative" ref={dropdownRef}>
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              handleToggleDropdown();
            }}
            className={cn(
              TYPE_CLASS,
              "group inline-flex shrink-0 items-baseline gap-1 text-muted-foreground transition-colors hover:text-foreground",
            )}
            aria-expanded={dropdownVisible}
            aria-haspopup="listbox"
          >
            {/* Simple opacity fade transition */}
            <span
              className={cn(
                TYPE_CLASS,
                "transition-opacity duration-500",
                isTransitioning ? "opacity-0" : "opacity-100",
              )}
            >
              .{displaySuffix}
            </span>

            {activeDomains.length > 1 && (
              <ChevronDown
                className={cn(
                  "h-4 w-4 shrink-0 translate-y-[-0.15em] text-muted-foreground/50 transition-transform duration-300",
                  dropdownVisible && "rotate-180",
                  "group-hover:text-muted-foreground",
                )}
              />
            )}
          </button>

          {/* Custom dropdown — pure floating text */}
          {dropdownVisible && (
            <div
              role="listbox"
              aria-label={t("landing.searchKicker")}
              className={cn(
                "absolute right-0 top-full mt-3 z-50",
                "origin-top-right",
                dropdownOpen
                  ? "animate-in fade-in-0 zoom-in-95 slide-in-from-top-2 duration-300"
                  : "animate-out fade-out-0 zoom-out-95 slide-out-to-top-2 duration-300",
              )}
            >
              {/* All domains in original order, current one highlighted in-place */}
              {activeDomains.map((d) => (
                <button
                  key={d.id}
                  type="button"
                  role="option"
                  aria-selected={currentDomain?.id === d.id}
                  onClick={(e) => {
                    e.stopPropagation();
                    if (currentDomain?.id === d.id) {
                      closeDropdown();
                    } else {
                      handleSelectDomain(d.id);
                    }
                  }}
                  className={cn(
                    TYPE_CLASS,
                    "block w-full px-5 py-1.5 text-right transition-colors duration-150 relative whitespace-nowrap",
                    currentDomain?.id === d.id
                      ? "text-foreground"
                      : "text-muted-foreground/50 hover:text-muted-foreground/90",
                  )}
                >
                  {currentDomain?.id === d.id && (
                    <span className="absolute inset-y-1.5 right-0 w-0.5 rounded-l-full bg-brand/60" />
                  )}
                  .{d.name}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Status line */}
      <div className="mt-3 min-h-[1.5rem] text-sm">
        {checkState === "available" && (
          <div className="flex flex-wrap items-center gap-2">
            <span className="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400">
              <Check className="h-4 w-4" />
              <span className="font-medium">{t("landing.searchAvailable")}</span>
            </span>
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
          </div>
        )}
      </div>
    </div>
  );
}
