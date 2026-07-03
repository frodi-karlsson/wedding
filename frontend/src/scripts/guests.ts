/**
 * Order a guest list so primary invitees come first, preserving the relative
 * order of the rest. Works for any shape carrying an `is_primary` flag
 * (both `GuestInput` and `GuestResponse`). Returns a new array; the input is
 * not mutated.
 */
export function sortPrimaryFirst<T extends { is_primary: boolean }>(guests: T[]): T[] {
  return guests.slice().sort((a, b) => Number(b.is_primary) - Number(a.is_primary));
}

/** Render a list of names as "A", "A & B", or "A, B & C". */
export function joinNames(names: string[]): string {
  const clean = names.map((n) => n.trim()).filter((n) => n.length > 0);
  if (clean.length <= 1) return clean[0] ?? '';
  return `${clean.slice(0, -1).join(', ')} & ${clean[clean.length - 1]}`;
}
