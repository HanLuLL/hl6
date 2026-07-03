import { useState, useEffect, useRef, useCallback, useLayoutEffect } from "react";
import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { api } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { ChevronDown, Check, X, ArrowRight } from "lucide-react";

type CheckState = "idle" | "checking" | "available" | "taken" | "error";

const TYPE_CLASS = "text-2xl sm:text-3xl font-semibold tracking-tight";
const DEFAULT_DOMAIN_NAME = "hlcloud";
const FALLBACK_SUFFIX = "houlang.cloud";

function sanitize(v: string): string {
  return v.replace(/[^\p{Script=Han}\p{N}a-zA-Z-]/gu, "").toLowerCase();
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
  const [dropdownPosition, setDropdownPosition] = useState<{
    top: number;
    right: number;
    maxHeight: number;
  } | null>(null);

  // Simple fade transition for auto-switch
  const [isTransitioning, setIsTransitioning] = useState(false);

  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const transitionTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const closeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Close dropdown with fade-out animation
  const closeDropdown = useCallback(() => {
    setDropdownOpen(false);
    if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    closeTimerRef.current = setTimeout(() => {
      setDropdownVisible(false);
    }, 300);
  }, []);

  const updateDropdownPosition = useCallback(() => {
    if (!dropdownRef.current) return;

    const rect = dropdownRef.current.getBoundingClientRect();
    const gap = 12;
    const viewportPadding = 16;
    const preferredMaxHeight = 448;
    const minHeight = 160;
    const availableBelow = window.innerHeight - rect.bottom - gap - viewportPadding;
    const availableAbove = rect.top - gap - viewportPadding;
    const openAbove = availableBelow < minHeight && availableAbove > availableBelow;
    const availableHeight = openAbove ? availableAbove : availableBelow;
    const maxHeight = Math.max(minHeight, Math.min(preferredMaxHeight, availableHeight));

    setDropdownPosition({
      top: openAbove ? Math.max(viewportPadding, rect.top - gap - maxHeight) : rect.bottom + gap,
      right: Math.max(viewportPadding, window.innerWidth - rect.right),
      maxHeight,
    });
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

  const preferredDomainIndex = activeDomains.findIndex((d) => d.name === DEFAULT_DOMAIN_NAME);

  // Default suffix to .hlcloud when available
  useEffect(() => {
    if (manualSelect || prefix || activeDomains.length === 0 || preferredDomainIndex < 0) return;
    setDisplayIndex(preferredDomainIndex);
  }, [activeDomains.length, manualSelect, prefix, preferredDomainIndex]);

  // Auto-switch with simple opacity fade transition (skip when hlcloud is the default)
  useEffect(() => {
    if (manualSelect || prefix || activeDomains.length <= 1 || preferredDomainIndex >= 0) return;

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
  }, [activeDomains.length, manualSelect, prefix, preferredDomainIndex]);

  // Cleanup close timer on unmount
  useEffect(() => {
    return () => {
      if (closeTimerRef.current) clearTimeout(closeTimerRef.current);
    };
  }, []);

  useLayoutEffect(() => {
    if (!dropdownVisible) return;

    updateDropdownPosition();
    window.addEventListener("resize", updateDropdownPosition);
    window.addEventListener("scroll", updateDropdownPosition, true);
    return () => {
      window.removeEventListener("resize", updateDropdownPosition);
      window.removeEventListener("scroll", updateDropdownPosition, true);
    };
  }, [dropdownVisible, updateDropdownPosition]);

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
  }, [dropdownOpen, closeDropdown]);

  const scrollDropdownList = useCallback((deltaY: number, deltaMode: number) => {
    const el = listRef.current;
    if (!el || deltaY === 0) return;
    const delta =
      deltaMode === WheelEvent.DOM_DELTA_LINE
        ? deltaY * 16
        : deltaMode === WheelEvent.DOM_DELTA_PAGE
          ? deltaY * el.clientHeight
          : deltaY;
    el.scrollTop += delta;
  }, []);

  // Lock page scroll while open; route all wheel/touch to the domain list
  useEffect(() => {
    if (!dropdownOpen) return;

    const scrollY = window.scrollY;
    const { style } = document.body;
    const prev = {
      position: style.position,
      top: style.top,
      width: style.width,
      overflow: style.overflow,
    };
    style.position = "fixed";
    style.top = `-${scrollY}px`;
    style.width = "100%";
    style.overflow = "hidden";

    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      scrollDropdownList(e.deltaY, e.deltaMode);
    };
    let touchStartY = 0;
    const onTouchStart = (e: TouchEvent) => {
      touchStartY = e.touches[0]?.clientY ?? 0;
    };
    const onTouchMove = (e: TouchEvent) => {
      const touchY = e.touches[0]?.clientY ?? touchStartY;
      e.preventDefault();
      scrollDropdownList(touchStartY - touchY, WheelEvent.DOM_DELTA_PIXEL);
      touchStartY = touchY;
    };

    window.addEventListener("wheel", onWheel, { passive: false, capture: true });
    window.addEventListener("touchstart", onTouchStart, { passive: true, capture: true });
    window.addEventListener("touchmove", onTouchMove, { passive: false, capture: true });

    return () => {
      style.position = prev.position;
      style.top = prev.top;
      style.width = prev.width;
      style.overflow = prev.overflow;
      window.scrollTo(0, scrollY);
      window.removeEventListener("wheel", onWheel, { capture: true });
      window.removeEventListener("touchstart", onTouchStart, { capture: true });
      window.removeEventListener("touchmove", onTouchMove, { capture: true });
    };
  }, [dropdownOpen, scrollDropdownList]);

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

  const handleDropdownWheel = useCallback((e: React.WheelEvent<HTMLDivElement>) => {
    if (e.deltaY === 0) return;
    e.preventDefault();
    e.stopPropagation();
    scrollDropdownList(e.deltaY, e.deltaMode);
  }, [scrollDropdownList]);

  const displaySuffix = currentDomain?.name ?? FALLBACK_SUFFIX;
  const placeholderText = t("landing.searchPlaceholder");

  useLayoutEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.scrollLeft = el.scrollWidth;
  }, [prefix]);

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
      <div className={cn("flex items-baseline gap-3", dropdownVisible && "relative z-50")}>
        <div
          className={cn(
            "flex min-w-0 flex-1 items-baseline gap-2 border-b-2 pb-2.5 transition-colors duration-300",
            underlineColor,
          )}
        >
          {/* Prefix — fills remaining space, scrolls when long */}
          <div
            className="min-w-0 flex-1 overflow-hidden"
            onClick={() => inputRef.current?.focus()}
          >
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
              placeholder={placeholderText}
              spellCheck={false}
              autoComplete="off"
              aria-label={t("landing.searchKicker")}
              className={cn(
                TYPE_CLASS,
                "w-full min-w-0 border-0 bg-transparent p-0 text-foreground caret-brand outline-none overflow-x-auto overflow-y-hidden whitespace-nowrap placeholder:text-muted-foreground/40 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
              )}
            />
          </div>

          {/* Domain suffix selector */}
          <div className="relative shrink-0" ref={dropdownRef}>
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
                ref={listRef}
                role="listbox"
                aria-label={t("landing.searchKicker")}
                onWheel={handleDropdownWheel}
                style={
                  dropdownPosition
                    ? {
                        top: dropdownPosition.top,
                        right: dropdownPosition.right,
                        maxHeight: dropdownPosition.maxHeight,
                      }
                    : undefined
                }
                className={cn(
                  "fixed z-50",
                  "origin-top-right overflow-y-auto overscroll-contain touch-pan-y [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden",
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
                      "block w-full px-1 py-1.5 text-right transition-colors duration-150 relative whitespace-nowrap",
                      currentDomain?.id === d.id
                        ? "text-foreground"
                        : "text-muted-foreground/50 hover:text-muted-foreground/90",
                    )}
                  >

                    .{d.name}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {checkState === "available" ? (
          isAuthenticated ? (
            <Button
              size="lg"
              asChild
              className="shrink-0 bg-emerald-600 text-white hover:bg-emerald-700 dark:bg-emerald-600 dark:hover:bg-emerald-500"
            >
              <Link to="/domains" onClick={(e) => e.stopPropagation()}>
                {t("landing.searchClaim")}
              </Link>
            </Button>
          ) : (
            <Button
              size="lg"
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                signIn();
              }}
              className="shrink-0 bg-emerald-600 text-white hover:bg-emerald-700 dark:bg-emerald-600 dark:hover:bg-emerald-500"
            >
              {t("landing.searchClaim")}
            </Button>
          )
        ) : isAuthenticated ? (
          <Button
            size="lg"
            asChild
            className="shrink-0 bg-brand hover:bg-brand/90 text-brand-foreground"
          >
            <Link to="/domains" onClick={(e) => e.stopPropagation()}>
              {t("landing.browseDomains")}
              <ArrowRight className="ml-2 h-4 w-4" />
            </Link>
          </Button>
        ) : (
          <Button
            size="lg"
            type="button"
            className="shrink-0 bg-brand hover:bg-brand/90 text-brand-foreground"
            onClick={(e) => {
              e.stopPropagation();
              signIn();
            }}
          >
            {t("landing.browseDomains")}
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        )}
      </div>

      {/* Status line */}
      <div className="mt-3 min-h-[1.5rem] text-sm">
        {checkState === "available" && (
          <span className="inline-flex items-center gap-1.5 text-emerald-600 dark:text-emerald-400">
            <Check className="h-4 w-4" />
            <span className="font-medium">{t("landing.searchAvailable")}</span>
          </span>
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
