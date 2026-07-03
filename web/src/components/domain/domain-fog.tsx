import { useEffect, useRef, useState } from "react";
import * as THREE from "three";
import FOG from "vanta/dist/vanta.fog.min";
import { cn } from "@/lib/utils";

type VantaEffect = { destroy: () => void };

/**
 * Vanta FOG 背景层。组件本身只负责一个填满父容器的 div 与 Vanta 生命周期，
 * 定位（absolute inset-0 等）由调用方通过 className 控制，便于用作 hero 背景。
 */
export function DomainFog({ className }: { className?: string }) {
  const elRef = useRef<HTMLDivElement>(null);
  const [isVisible, setIsVisible] = useState(false);

  useEffect(() => {
    if (!elRef.current) return;
    let effect: VantaEffect | null = null;
    let frame = 0;
    try {
      effect = FOG({
        el: elRef.current,
        THREE,
        mouseControls: true,
        touchControls: true,
        gyroControls: false,
        minHeight: 200.0,
        minWidth: 200.0,
        highlightColor: 0x0059ff,
        midtoneColor: 0x00a4ff,
        lowlightColor: 0x00a4ff,
        baseColor: 0xffebeb,
        blurFactor: 0.32,
        speed: 0.5,
        zoom: 0.8,
      });
      frame = requestAnimationFrame(() => setIsVisible(true));
    } catch {
      effect = null;
    }
    return () => {
      cancelAnimationFrame(frame);
      effect?.destroy();
      setIsVisible(false);
    };
  }, []);

  return (
    <div
      ref={elRef}
      className={cn(
        "h-full w-full opacity-0 transition-opacity duration-700 ease-out will-change-opacity",
        isVisible && "opacity-100",
        className,
      )}
    />
  );
}
