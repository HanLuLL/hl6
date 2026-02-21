import { useLocation } from "react-router-dom";
import { useLayoutEffect, useRef } from "react";

export function PageTransition({ children }: { children: React.ReactNode }) {
  const { pathname } = useLocation();
  const ref = useRef<HTMLDivElement>(null);
  const prevPathname = useRef(pathname);

  useLayoutEffect(() => {
    if (pathname === prevPathname.current) return;
    prevPathname.current = pathname;

    const el = ref.current;
    if (!el) return;

    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;

    // Synchronously hide before browser paints — no blank frame
    el.style.transition = "none";
    el.style.opacity = "0";

    // Next frame: fade in with smooth transition
    requestAnimationFrame(() => {
      el.style.transition = "opacity 150ms cubic-bezier(0.4, 0, 0.2, 1)";
      el.style.opacity = "1";
    });
  }, [pathname]);

  return <div ref={ref}>{children}</div>;
}
