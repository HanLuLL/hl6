import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { ZoomIn, ZoomOut, RotateCcw, RotateCw, X } from "lucide-react";
import {
  Dialog,
  DialogContent,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";

interface ImageLightboxProps {
  src: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ImageLightbox({ src, open, onOpenChange }: ImageLightboxProps) {
  const { t } = useTranslation();
  const [zoom, setZoom] = useState(1);
  const [rotation, setRotation] = useState(0);

  useEffect(() => {
    if (open) {
      setZoom(1);
      setRotation(0);
    }
  }, [open]);

  const zoomIn = () => setZoom((z) => Math.min(z + 0.25, 5));
  const zoomOut = () => setZoom((z) => Math.max(z - 0.25, 0.25));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        showCloseButton={false}
        className="fixed inset-0 z-[60] flex max-w-none translate-x-0 translate-y-0 top-0 left-0 items-center justify-center border-none bg-black/90 p-0 rounded-none sm:max-w-none"
      >
        {src && (
          <img
            src={src}
            alt=""
            className="max-h-[85vh] max-w-[90vw] object-contain transition-transform duration-200"
            style={{ transform: `scale(${zoom}) rotate(${rotation}deg)` }}
            draggable={false}
          />
        )}

        <div className="fixed bottom-6 left-1/2 -translate-x-1/2 flex items-center gap-1 rounded-lg bg-black/70 px-3 py-2 backdrop-blur-sm">
          <Button
            variant="ghost"
            size="icon"
            className="size-8 text-white hover:bg-white/20"
            onClick={zoomOut}
            aria-label={t("notifications.zoomOut")}
          >
            <ZoomOut className="size-4" />
          </Button>
          <span className="min-w-[3.5rem] text-center text-sm text-white">
            {Math.round(zoom * 100)}%
          </span>
          <Button
            variant="ghost"
            size="icon"
            className="size-8 text-white hover:bg-white/20"
            onClick={zoomIn}
            aria-label={t("notifications.zoomIn")}
          >
            <ZoomIn className="size-4" />
          </Button>

          <Separator orientation="vertical" className="mx-1 h-5 bg-white/30" />

          <Button
            variant="ghost"
            size="icon"
            className="size-8 text-white hover:bg-white/20"
            onClick={() => setRotation((r) => r - 90)}
            aria-label={t("notifications.rotateLeft")}
          >
            <RotateCcw className="size-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="size-8 text-white hover:bg-white/20"
            onClick={() => setRotation((r) => r + 90)}
            aria-label={t("notifications.rotateRight")}
          >
            <RotateCw className="size-4" />
          </Button>

          <Separator orientation="vertical" className="mx-1 h-5 bg-white/30" />

          <Button
            variant="ghost"
            size="icon"
            className="size-8 text-white hover:bg-white/20"
            onClick={() => onOpenChange(false)}
            aria-label={t("notifications.close")}
          >
            <X className="size-4" />
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
