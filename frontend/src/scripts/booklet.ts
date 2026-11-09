import type { SpeechResponse } from './types.gen';

/**
 * A run of booklet content. Speeches become blocks, and the layout step flows
 * blocks onto A5 pages. `startsPage` forces a fresh page, so each speech begins
 * at the top of one.
 */
export interface Block {
  kind: 'heading' | 'subheading' | 'paragraph';
  text: string;
  startsPage?: boolean;
  /** The paragraph carries on at the top of the next page. */
  continues?: boolean;
}

/** A booklet folded from sheets holds four pages per sheet. */
export function paddedPageCount(pages: number): number {
  return Math.max(4, Math.ceil(pages / 4) * 4);
}

/** Page numbers printed on one A4 sheet, each side as [left half, right half]. */
export interface Sheet {
  front: [number, number];
  back: [number, number];
}

/**
 * Saddle-stitch imposition: the order pages must be printed in so that the
 * stacked, folded sheets read 1..n. Sheet i carries the i-th page pair from
 * each end of the booklet. Print double-sided, flipping on the short edge.
 */
export function impose(pageCount: number): Sheet[] {
  const n = paddedPageCount(pageCount);
  return Array.from({ length: n / 4 }, (_, i) => ({
    front: [n - 2 * i, 2 * i + 1],
    back: [2 * i + 2, n - 2 * i - 1],
  }));
}

/** Reading-order spreads for the preview: the covers stand alone. */
export function spreads(pageCount: number): [number | null, number | null][] {
  const n = paddedPageCount(pageCount);
  const result: [number | null, number | null][] = [[null, 1]];
  for (let left = 2; left < n; left += 2) {
    result.push([left, left + 1]);
  }
  result.push([n, null]);
  return result;
}

function paragraphs(body: string): string[] {
  return body
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .filter((p) => p.length > 0);
}

export function bookletBlocks(
  speeches: SpeechResponse[],
  lang: string,
): { blocks: Block[]; missing: string[] } {
  const blocks: Block[] = [];
  const missing: string[] = [];
  for (const speech of speeches) {
    const body = speech.texts.find((t) => t.lang === lang)?.body ?? '';
    const paras = paragraphs(body);
    if (paras.length === 0) {
      missing.push(speech.speaker);
      continue;
    }
    blocks.push({ kind: 'heading', text: speech.speaker, startsPage: true });
    if (speech.title) blocks.push({ kind: 'subheading', text: speech.title });
    for (const text of paras) blocks.push({ kind: 'paragraph', text });
  }
  return { blocks, missing };
}
