import { useEffect, useRef, useState } from 'react';
import type { DragEvent } from 'react';
import type { SourceLayer } from './dataset';

export type LayerPlacement = 'before' | 'after';
type DropTarget = { id: string; placement: LayerPlacement };
type LayerOrderControlProps = {
  layers: SourceLayer[];
  displayErrors: Record<string, string>;
  onMove: (layerId: string, targetId: string, placement: LayerPlacement) => void;
};
const dragType = 'application/x-geoflatpack-layer';

export default function LayerOrderControl({ layers, displayErrors, onMove }: LayerOrderControlProps) {
  const [open, setOpen] = useState(false);
  const [draggedId, setDraggedId] = useState<string | null>(null);
  const [dropTarget, setDropTarget] = useState<DropTarget | null>(null);
  const [announcement, setAnnouncement] = useState('');
  const toggle = useRef<HTMLButtonElement>(null);
  const rows = useRef<Record<string, HTMLLIElement | null>>({});
  const pendingFocus = useRef<string | null>(null);
  // MapLibre draws the last dataset on top; show that dataset first in the list.
  const orderedLayers = layers.filter(layer => layer.visible).reverse();

  useEffect(() => {
    if (!pendingFocus.current) return;
    const row = rows.current[pendingFocus.current];
    const focused = document.activeElement;
    if (row && (!row.contains(focused) || (focused instanceof HTMLButtonElement && focused.disabled))) {
      (row.querySelector<HTMLButtonElement>('button:not(:disabled)') ?? row).focus();
    }
    row?.scrollIntoView({ block: 'nearest' });
    pendingFocus.current = null;
  }, [layers]);

  function clearDrag() {
    setDraggedId(null);
    setDropTarget(null);
  }

  function move(layerId: string, targetId: string, placement: LayerPlacement, keepFocus = false) {
    if (layerId === targetId) return;
    const from = orderedLayers.findIndex(layer => layer.id === layerId);
    const remaining = orderedLayers.filter(layer => layer.id !== layerId);
    const target = remaining.findIndex(layer => layer.id === targetId);
    if (from < 0 || target < 0) return;
    const to = target + (placement === 'after' ? 1 : 0);
    if (from === to) return;
    if (keepFocus) pendingFocus.current = layerId;
    onMove(layerId, targetId, placement);
    setAnnouncement(`${orderedLayers[from].label} moved to position ${to + 1} of ${orderedLayers.length}.`);
  }

  function placementAt(event: DragEvent<HTMLLIElement>): LayerPlacement {
    const bounds = event.currentTarget.getBoundingClientRect();
    return event.clientY < bounds.top + bounds.height / 2 ? 'before' : 'after';
  }

  return (
    <div className="layer-order-control" onKeyDown={event => {
      if (event.key === 'Escape' && open) {
        event.preventDefault();
        clearDrag();
        setOpen(false);
        toggle.current?.focus();
      }
    }}>
      <button ref={toggle} type="button" className="inspector-toggle layer-order-toggle"
        aria-expanded={open} aria-controls="layer-order-panel" onClick={() => {
          clearDrag();
          setOpen(current => !current);
        }}>
        Layer order <span aria-hidden="true">{open ? '▾' : '▴'}</span>
      </button>
      {open && (
        <section id="layer-order-panel" className="layer-order-panel" aria-labelledby="layer-order-title">
          <h2 id="layer-order-title" className="header-panel-title">Layer order</h2>
          <p id="layer-order-help">Active sources only. Top layers draw above lower layers. Drag a handle or use the arrows to reorder.</p>
          {orderedLayers.length === 0 && <p>No active sources. Enable a source in Select sources.</p>}
          <ol className="layer-order-list" aria-describedby="layer-order-help" onDragLeave={event => {
            if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setDropTarget(null);
          }}>
            {orderedLayers.map((layer, index) => (
              <li key={layer.id} ref={node => { rows.current[layer.id] = node; }} tabIndex={-1}
                className="layer-order-row" data-dragging={draggedId === layer.id}
                data-drop={dropTarget?.id === layer.id ? dropTarget.placement : undefined}
                onDragOver={event => {
                  if (!draggedId || !event.dataTransfer.types.includes(dragType)) return;
                  event.preventDefault();
                  event.dataTransfer.dropEffect = 'move';
                  setDropTarget(draggedId === layer.id ? null : { id: layer.id, placement: placementAt(event) });
                }}
                onDrop={event => {
                  if (!draggedId || event.dataTransfer.getData(dragType) !== draggedId) return;
                  event.preventDefault();
                  move(draggedId, layer.id, placementAt(event));
                  clearDrag();
                }}>
                <span className="layer-order-handle" draggable aria-hidden="true" title={`Drag ${layer.label}`}
                  onDragStart={event => {
                    event.dataTransfer.effectAllowed = 'move';
                    event.dataTransfer.setData(dragType, layer.id);
                    setDraggedId(layer.id);
                    setDropTarget(null);
                  }} onDragEnd={clearDrag}>⠿</span>
                <div className="layer-order-label">
                  <span title={layer.filename}>{layer.label}</span>
                  <small>{layer.error || displayErrors[layer.id] ? 'Error' : layer.loading ? 'Loading…' : 'Visible'}</small>
                </div>
                <div className="layer-order-actions">
                  <button type="button" className="inspector-toggle" disabled={index === 0}
                    aria-label={`Move ${layer.label} up`} title="Move up"
                    onClick={() => move(layer.id, orderedLayers[index - 1].id, 'before', true)}>↑</button>
                  <button type="button" className="inspector-toggle" disabled={index === orderedLayers.length - 1}
                    aria-label={`Move ${layer.label} down`} title="Move down"
                    onClick={() => move(layer.id, orderedLayers[index + 1].id, 'after', true)}>↓</button>
                </div>
              </li>
            ))}
          </ol>
        </section>
      )}
      <span className="visually-hidden" role="status" aria-live="polite" aria-atomic="true">{announcement}</span>
    </div>
  );
}
