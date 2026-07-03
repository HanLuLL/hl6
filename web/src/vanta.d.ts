// three 0.185 不再自带类型；此处仅把 THREE 原样传给 Vanta，无需完整类型，声明为宽松模块即可。
declare module "three";

declare module "vanta/dist/vanta.fog.min" {
  interface VantaEffect {
    destroy: () => void;
    resize: () => void;
    setOptions: (options: Record<string, unknown>) => void;
  }
  interface VantaFogOptions {
    el: HTMLElement | string;
    THREE?: unknown;
    mouseControls?: boolean;
    touchControls?: boolean;
    gyroControls?: boolean;
    minHeight?: number;
    minWidth?: number;
    highlightColor?: number;
    midtoneColor?: number;
    lowlightColor?: number;
    baseColor?: number;
    blurFactor?: number;
    speed?: number;
    zoom?: number;
  }
  const FOG: (options: VantaFogOptions) => VantaEffect;
  export default FOG;
}
