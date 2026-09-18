/** Human label for a component, used in aria labels and dependency options. */
export function labelOf(kind: string, name: string) {
  return name ? `${kind} ${name}` : `unnamed ${kind.toLowerCase()}`;
}
