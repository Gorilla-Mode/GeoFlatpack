import { useState } from 'react';

type HeaderTreeProps = {
  value: unknown;
  label?: string;
  initialOpen?: boolean;
};

export default function HeaderTree({ value, label, initialOpen = false }: HeaderTreeProps) {
  const [open, setOpen] = useState(initialOpen);
  const prefix = label === undefined ? '' : `${label}: `;
  const displayValue = value instanceof Float64Array ? Array.from(value) : value;

  if (displayValue === null || typeof displayValue !== 'object') {
    return (
      <div className="header-tree-value">
        {prefix}{JSON.stringify(displayValue)}
      </div>
    );
  }

  const entries = Object.entries(displayValue);
  const description = Array.isArray(displayValue)
    ? `Array[${displayValue.length}]`
    : 'Object';

  return (
    <details open={open} onToggle={event => setOpen(event.currentTarget.open)}>
      <summary>{prefix}{description}</summary>
      <div className="header-tree-children">
        {open && entries.map(([key, child]) => (
          <HeaderTree key={key} label={key} value={child} />
        ))}
      </div>
    </details>
  );
}
