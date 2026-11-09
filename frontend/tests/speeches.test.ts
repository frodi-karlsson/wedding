import { test, expect } from 'vitest';
import {
  textStatus,
  draftFrom,
  isDirty,
  afterSave,
  withConflict,
  keepMine,
  useTheirs,
  langProgress,
} from '../src/scripts/speeches.service';
import type { SpeechResponse, SpeechTextResponse } from '../src/scripts/types.gen';

function text(overrides: Partial<SpeechTextResponse> = {}): SpeechTextResponse {
  return { lang: 'is', body: '', done: false, version: 1, ...overrides };
}

function speech(texts: SpeechTextResponse[]): SpeechResponse {
  return { id: 1, position: 1, speaker: 'Anna', title: '', original_lang: 'sv', version: 1, texts };
}

test('should call an empty text missing', () => {
  const status = textStatus(text({ body: '   ' }));

  expect(status).toBe('missing');
});

test('should call a text with content a draft until it is marked done', () => {
  const draft = textStatus(text({ body: 'Kæru vinir' }));
  const done = textStatus(text({ body: 'Kæru vinir', done: true }));

  expect(draft).toBe('draft');
  expect(done).toBe('done');
});

test('should not be dirty straight after loading', () => {
  const draft = draftFrom(text({ body: 'Hæ' }));

  expect(isDirty(draft)).toBe(false);
});

test('should be dirty when the body or done flag changes', () => {
  const draft = draftFrom(text({ body: 'Hæ' }));

  const edited = isDirty({ ...draft, body: 'Halló' });
  const finished = isDirty({ ...draft, done: true });

  expect(edited).toBe(true);
  expect(finished).toBe(true);
});

test('should adopt the saved text and version after a save', () => {
  const draft = { ...draftFrom(text({ body: 'Hæ' })), body: 'Halló' };

  const result = afterSave(draft, text({ body: 'Halló', version: 2 }));

  expect(result).toEqual({ body: 'Halló', done: false, saved: text({ body: 'Halló', version: 2 }) });
  expect(isDirty(result)).toBe(false);
});

test('should keep the user\'s edit alongside the conflicting text', () => {
  const draft = { ...draftFrom(text({ body: 'Hæ' })), body: 'Mine' };

  const result = withConflict(draft, text({ body: 'Theirs', version: 3 }));

  expect(result.body).toBe('Mine');
  expect(result.conflict).toEqual(text({ body: 'Theirs', version: 3 }));
});

test('should keep mine on top of their version so the next save goes through', () => {
  const conflicted = withConflict({ ...draftFrom(text({ body: 'Hæ' })), body: 'Mine' }, text({ body: 'Theirs', version: 3 }));

  const result = keepMine(conflicted);

  expect(result).toEqual({ body: 'Mine', done: false, saved: text({ body: 'Theirs', version: 3 }) });
  expect(isDirty(result)).toBe(true);
});

test('should replace my edit with theirs when using theirs', () => {
  const conflicted = withConflict({ ...draftFrom(text({ body: 'Hæ' })), body: 'Mine' }, text({ body: 'Theirs', done: true, version: 3 }));

  const result = useTheirs(conflicted);

  expect(result).toEqual(draftFrom(text({ body: 'Theirs', done: true, version: 3 })));
});

test('should count finished translations per language', () => {
  const speeches = [
    speech([text({ lang: 'is', body: 'a', done: true }), text({ lang: 'en', body: 'b' })]),
    speech([text({ lang: 'is', body: 'c' }), text({ lang: 'en', body: 'd', done: true })]),
    speech([text({ lang: 'is', body: 'e', done: true }), text({ lang: 'en' })]),
  ];

  const progress = langProgress(speeches, 'is');

  expect(progress).toEqual({ done: 2, total: 3 });
});
