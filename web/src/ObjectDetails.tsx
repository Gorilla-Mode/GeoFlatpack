import { useState } from 'react';
import type { Feature } from 'geojson';
import { propertyGroups, scalarValue } from './featureProperties';

function PropertyValue({ value }: { value: unknown }) {
  const [open, setOpen] = useState(false);
  if (value !== null && typeof value === 'object') {
    const entries = Object.entries(value);
    return <details className="nested-properties" onToggle={event => setOpen(event.currentTarget.open)}>
      <summary>{Array.isArray(value) ? 'Array' : 'Object'} · {entries.length} {Array.isArray(value) ? 'items' : 'fields'}</summary>
      {open && <dl>{entries.map(([key, child]) => <div className="nested-property" key={key}>
        <dt>{key}</dt><dd><PropertyValue value={child} /></dd>
      </div>)}</dl>}
    </details>;
  }
  const displayed = scalarValue(value);
  return <><span className="property-value">{displayed.text}</span><span className="value-type">{displayed.type}</span></>;
}

export default function ObjectDetails({ feature, layerLabel }: { feature: Feature; layerLabel: string }) {
  const [rawOpen, setRawOpen] = useState(false);
  const groups = propertyGroups(feature.properties);
  return <div className="object-details">
    <p className="object-source"><span aria-hidden="true">◇</span> {layerLabel}</p>
    {!groups.length && <p className="empty-state">{feature.properties === null ? 'Properties are null.' : 'This object has no properties.'}</p>}
    {groups.map(group => <section className="property-group" key={group.title} aria-label={group.title}>
      <h3>{group.title}</h3>
      <dl>{group.entries.map(([key, value]) => <div className="property-row" key={key}>
        <dt>{key === 'name' ? <><span>Name</span><code>name</code></> : <span>{key}</span>}</dt>
        <dd><PropertyValue value={value} /></dd>
      </div>)}</dl>
    </section>)}
    <details className="raw-properties" onToggle={event => setRawOpen(event.currentTarget.open)}>
      <summary>Raw properties JSON</summary>
      {rawOpen && <pre tabIndex={0} aria-label="Raw properties JSON">{JSON.stringify(feature.properties, null, 2)}</pre>}
    </details>
  </div>;
}
