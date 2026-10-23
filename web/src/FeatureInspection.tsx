import { useState } from 'react';
import type { Feature } from 'geojson';
import HeaderTree from './HeaderTree';
import { createFeatureInspection } from './inspection';
import { featureTitle } from './featureProperties';

function FeatureRaw({ feature }: { feature: Feature }) {
  const [open, setOpen] = useState(false);
  return <details onToggle={event => setOpen(event.currentTarget.open)}>
    <summary>Geometry and properties</summary>
    {open && <HeaderTree value={createFeatureInspection(feature)} initialOpen />}
  </details>;
}

export default function FeatureInspection({ features, onSelect }: {
  features: Feature[];
  onSelect: (index: number) => void;
}) {
  const [open, setOpen] = useState(false);
  return <details className="feature-list" onToggle={event => setOpen(event.currentTarget.open)}>
    <summary>Objects ({features.length})</summary>
    {open && <ol>{features.map((feature, index) => <li key={index}>
      <button type="button" className="feature-select" aria-label={`Show object details: ${featureTitle(feature, index)}, object ${index + 1}`}
        onClick={() => onSelect(index)}>{featureTitle(feature, index)}<span aria-hidden="true"> ↗</span>
        <small>Object {index + 1} · {feature.geometry?.type ?? 'No geometry'}</small></button>
      <FeatureRaw feature={feature} />
    </li>)}</ol>}
  </details>;
}
