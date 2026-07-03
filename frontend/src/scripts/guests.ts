/**
 * Order a guest list so primary invitees come first, preserving the relative
 * order of the rest. Works for any shape carrying an `is_primary` flag
 * (both `GuestInput` and `GuestResponse`). Returns a new array; the input is
 * not mutated.
 */
export function sortPrimaryFirst<T extends { is_primary: boolean }>(guests: T[]): T[] {
  return guests.slice().sort((a, b) => Number(b.is_primary) - Number(a.is_primary));
}
