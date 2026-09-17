import type { ReactNode } from 'react';
import type { Problem } from '../api/types';

/** DOM id of the input for a field path such as workloads.<id>.name. */
export function fieldDomId(field: string): string {
  return 'field-' + field.replace(/[^A-Za-z0-9_-]/g, '-');
}

export type ProblemsByField = Map<string, Problem[]>;

export function groupProblems(problems: Problem[]): ProblemsByField {
  const map: ProblemsByField = new Map();
  for (const p of problems) {
    if (!p.field) continue;
    map.set(p.field, [...(map.get(p.field) ?? []), p]);
  }
  return map;
}

function FieldErrors({ id, problems }: { id: string; problems: Problem[] }) {
  if (problems.length === 0) return null;
  return (
    <ul className="field-errors" id={id}>
      {problems.map((p, i) => (
        <li key={i}>{p.message}</li>
      ))}
    </ul>
  );
}

interface TextFieldProps {
  field: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  problems: ProblemsByField;
  hint?: ReactNode;
  required?: boolean;
  multiline?: boolean;
  inputMode?: 'text' | 'numeric';
  list?: string;
  placeholder?: string;
  className?: string;
}

export function TextField(props: TextFieldProps) {
  const id = fieldDomId(props.field);
  const errors = props.problems.get(props.field) ?? [];
  const hintId = props.hint ? `${id}-hint` : undefined;
  const errorId = errors.length > 0 ? `${id}-errors` : undefined;
  const common = {
    id,
    value: props.value,
    'aria-invalid': errors.length > 0 || undefined,
    'aria-describedby': [hintId, errorId].filter(Boolean).join(' ') || undefined,
    'aria-required': props.required || undefined,
    placeholder: props.placeholder,
  };
  return (
    <div className={`field ${props.className ?? ''}`}>
      <label htmlFor={id}>
        {props.label}
        {props.required && <span aria-hidden="true"> *</span>}
      </label>
      {props.multiline ? (
        <textarea {...common} rows={2} onChange={(e) => props.onChange(e.target.value)} />
      ) : (
        <input {...common} type="text" inputMode={props.inputMode} list={props.list} autoComplete="off" spellCheck={false} onChange={(e) => props.onChange(e.target.value)} />
      )}
      {props.hint && (
        <p className="hint" id={hintId}>
          {props.hint}
        </p>
      )}
      <FieldErrors id={`${id}-errors`} problems={errors} />
    </div>
  );
}

interface SelectFieldProps {
  field: string;
  label: string;
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
  problems: ProblemsByField;
  placeholder: string;
}

export function SelectField(props: SelectFieldProps) {
  const id = fieldDomId(props.field) + '-' + props.label.toLowerCase().replace(/\W+/g, '-');
  return (
    <div className="field">
      <label htmlFor={id}>{props.label}</label>
      <select id={id} value={props.value} onChange={(e) => props.onChange(e.target.value)}>
        <option value="">{props.placeholder}</option>
        {props.options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  );
}

/** Errors for a whole row (for example a dependency) rather than one input. */
export function RowErrors({ field, problems }: { field: string; problems: ProblemsByField }) {
  return <FieldErrors id={fieldDomId(field) + '-errors'} problems={problems.get(field) ?? []} />;
}
