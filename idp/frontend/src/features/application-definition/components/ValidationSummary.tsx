import { forwardRef } from 'react';
import type { Problem } from '../api/types';
import { fieldDomId } from '../../../shared/ui/Field';

interface Props {
  problems: Problem[];
  title: string;
  onSelectField?: (field: string) => void;
}

function focusField(field: string) {
  // Inputs use the field path as id; row-level problems target the row.
  const target = document.getElementById(fieldDomId(field)) ?? document.getElementById(fieldDomId(field.split('.')[0] ?? field));
  target?.focus();
  target?.scrollIntoView?.({ block: 'center' });
}

/** Lists every problem; a problem tied to an input moves focus to it. */
export const ValidationSummary = forwardRef<HTMLDivElement, Props>(function ValidationSummary({ problems, title, onSelectField }, ref) {
  if (problems.length === 0) return null;
  return (
    <div className="banner banner-error" role="alert" tabIndex={-1} ref={ref}>
      <h2>{title}</h2>
      <ul>
        {problems.map((p, i) => (
          <li key={i}>
            {p.field ? (
              <button
                type="button"
                className="link"
                onClick={() => {
                  const field = p.field ?? '';
                  onSelectField?.(field);
                  window.setTimeout(() => focusField(field), 0);
                }}
              >
                {p.message}
              </button>
            ) : (
              p.message
            )}{' '}
            <code>{p.code}</code>
          </li>
        ))}
      </ul>
    </div>
  );
});
