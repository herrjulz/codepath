import { useEffect, useRef, useState, type ReactNode, type PointerEvent as ReactPointerEvent } from "react";

const minZoom = 0.35;
const maxZoom = 2.5;

export function PanZoomViewport({
  children,
  contentWidth,
  contentHeight,
  label,
}: {
  children: ReactNode;
  contentWidth: number;
  contentHeight: number;
  label: string;
}) {
  const viewportRef = useRef<HTMLDivElement>(null);
  const [zoom, setZoom] = useState(1);
  const [pan, setPan] = useState({ x: 0, y: 0 });
  const drag = useRef<{ x: number; y: number; panX: number; panY: number } | null>(null);

  const transform = useRef({ zoom: 1, pan: { x: 0, y: 0 } });
  transform.current = { zoom, pan };

  useEffect(() => {
    const el = viewportRef.current;
    if (!el) {
      return;
    }
    // Defer until layout has a real size.
    const frame = window.requestAnimationFrame(() => resetToFit(el, contentWidth, contentHeight, setZoom, setPan));
    const onWheelNative = (event: WheelEvent) => {
      event.preventDefault();
      const { zoom: currentZoom, pan: currentPan } = transform.current;
      const rect = el.getBoundingClientRect();
      const mx = event.clientX - rect.left;
      const my = event.clientY - rect.top;
      const factor = event.deltaY < 0 ? 1.08 : 1 / 1.08;
      const next = clamp(currentZoom * factor, minZoom, maxZoom);
      const ratio = next / currentZoom;
      setZoom(next);
      setPan({
        x: mx - (mx - currentPan.x) * ratio,
        y: my - (my - currentPan.y) * ratio,
      });
    };
    el.addEventListener("wheel", onWheelNative, { passive: false });
    return () => {
      window.cancelAnimationFrame(frame);
      el.removeEventListener("wheel", onWheelNative);
    };
  }, [contentWidth, contentHeight]);

  function onPointerDown(event: ReactPointerEvent<HTMLDivElement>) {
    if (event.button !== 0) {
      return;
    }
    const target = event.target as HTMLElement;
    if (target.closest(".sequence-msg.clickable, button, a")) {
      return;
    }
    drag.current = { x: event.clientX, y: event.clientY, panX: pan.x, panY: pan.y };
    event.currentTarget.setPointerCapture(event.pointerId);
  }

  function onPointerMove(event: ReactPointerEvent<HTMLDivElement>) {
    if (!drag.current) {
      return;
    }
    setPan({
      x: drag.current.panX + (event.clientX - drag.current.x),
      y: drag.current.panY + (event.clientY - drag.current.y),
    });
  }

  function onPointerUp(event: ReactPointerEvent<HTMLDivElement>) {
    if (!drag.current) {
      return;
    }
    drag.current = null;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
  }

  function zoomBy(factor: number) {
    const el = viewportRef.current;
    if (!el) {
      return;
    }
    const mx = el.clientWidth / 2;
    const my = el.clientHeight / 2;
    const next = clamp(zoom * factor, minZoom, maxZoom);
    const ratio = next / zoom;
    setZoom(next);
    setPan({
      x: mx - (mx - pan.x) * ratio,
      y: my - (my - pan.y) * ratio,
    });
  }

  function resetView() {
    const el = viewportRef.current;
    if (!el) {
      return;
    }
    resetToFit(el, contentWidth, contentHeight, setZoom, setPan);
  }

  return (
    <div className="panzoom" aria-label={label}>
      <div className="panzoom-tools">
        <button type="button" onClick={() => zoomBy(1.15)} aria-label="Zoom in">
          +
        </button>
        <button type="button" onClick={() => zoomBy(1 / 1.15)} aria-label="Zoom out">
          −
        </button>
        <button type="button" onClick={resetView} aria-label="Fit view">
          Fit
        </button>
        <span className="panzoom-zoom">{Math.round(zoom * 100)}%</span>
      </div>
      <div
        ref={viewportRef}
        className="panzoom-viewport"
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
      >
        <div
          className="panzoom-canvas"
          style={{
            width: contentWidth,
            height: contentHeight,
            transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})`,
          }}
        >
          {children}
        </div>
      </div>
    </div>
  );
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function resetToFit(
  el: HTMLDivElement,
  contentWidth: number,
  contentHeight: number,
  setZoom: (value: number) => void,
  setPan: (value: { x: number; y: number }) => void,
) {
  const pad = 24;
  const scale = Math.min(
    (Math.max(el.clientWidth, 1) - pad) / contentWidth,
    (Math.max(el.clientHeight, 1) - pad) / contentHeight,
    1,
  );
  const next = Math.max(minZoom, Math.min(maxZoom, scale));
  setZoom(next);
  setPan({
    x: (el.clientWidth - contentWidth * next) / 2,
    y: Math.max(12, (el.clientHeight - contentHeight * next) / 4),
  });
}
