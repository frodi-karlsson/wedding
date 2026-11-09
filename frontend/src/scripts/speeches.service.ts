import type { SpeechResponse, SpeechTextResponse } from './types.gen';

export type TextStatus = 'missing' | 'draft' | 'done';

export function textStatus(text: SpeechTextResponse): TextStatus {
  if (text.body.trim() === '') return 'missing';
  return text.done ? 'done' : 'draft';
}

/**
 * One language's text as it is being edited. `saved` is the last version the
 * server confirmed, which is what a save is checked against. `conflict` holds
 * someone else's newer version after a save was rejected.
 */
export interface TextDraft {
  body: string;
  done: boolean;
  saved: SpeechTextResponse;
  conflict?: SpeechTextResponse;
}

export function draftFrom(text: SpeechTextResponse): TextDraft {
  return { body: text.body, done: text.done, saved: text };
}

export function isDirty(draft: TextDraft): boolean {
  return draft.body !== draft.saved.body || draft.done !== draft.saved.done;
}

export function afterSave(draft: TextDraft, saved: SpeechTextResponse): TextDraft {
  return { body: draft.body, done: draft.done, saved };
}

export function withConflict(draft: TextDraft, current: SpeechTextResponse): TextDraft {
  return { ...draft, conflict: current };
}

/** Keep my edit, now based on their version, so saving again replaces theirs. */
export function keepMine(draft: TextDraft): TextDraft {
  return { body: draft.body, done: draft.done, saved: draft.conflict ?? draft.saved };
}

export function useTheirs(draft: TextDraft): TextDraft {
  return draftFrom(draft.conflict ?? draft.saved);
}

export function langProgress(speeches: SpeechResponse[], lang: string): { done: number; total: number } {
  const done = speeches.filter((s) => s.texts.some((t) => t.lang === lang && textStatus(t) === 'done')).length;
  return { done, total: speeches.length };
}
