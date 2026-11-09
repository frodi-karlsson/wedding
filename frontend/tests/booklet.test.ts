import { test, expect } from 'vitest';
import { bookletBlocks, impose, paddedPageCount, spreads } from '../src/scripts/booklet';
import type { SpeechResponse } from '../src/scripts/types.gen';

function speech(id: number, speaker: string, title: string, bodies: Record<string, string>): SpeechResponse {
  return {
    id,
    position: id,
    speaker,
    title,
    original_lang: 'sv',
    version: 1,
    texts: Object.entries(bodies).map(([lang, body]) => ({ lang, body, done: true, version: 1 })),
  };
}

test('should pad the page count up to a multiple of four', () => {
  const counts = [1, 4, 5, 8, 9].map(paddedPageCount);

  expect(counts).toEqual([4, 4, 8, 8, 12]);
});

test('should impose an eight page booklet onto two sheets', () => {
  const sheets = impose(8);

  expect(sheets).toEqual([
    { front: [8, 1], back: [2, 7] },
    { front: [6, 3], back: [4, 5] },
  ]);
});

test('should put every page on exactly one sheet side', () => {
  const pages = impose(12).flatMap((s) => [...s.front, ...s.back]);

  expect(pages.toSorted((a, b) => a - b)).toEqual(Array.from({ length: 12 }, (_, i) => i + 1));
});

test('should pair pages into spreads with the covers on their own', () => {
  const result = spreads(8);

  expect(result).toEqual([
    [null, 1],
    [2, 3],
    [4, 5],
    [6, 7],
    [8, null],
  ]);
});

test('should turn each speech into a page-starting heading and its paragraphs', () => {
  const speeches = [speech(1, 'Anna', 'Toast', { is: 'Kæru vinir.\n\nSkál!' })];

  const { blocks } = bookletBlocks(speeches, 'is');

  expect(blocks).toEqual([
    { kind: 'heading', text: 'Anna', startsPage: true },
    { kind: 'subheading', text: 'Toast' },
    { kind: 'paragraph', text: 'Kæru vinir.' },
    { kind: 'paragraph', text: 'Skál!' },
  ]);
});

test('should leave out and report speeches with no text in the language', () => {
  const speeches = [
    speech(1, 'Anna', '', { is: 'Hæ' }),
    speech(2, 'Björn', '', { is: '  ', sv: 'Hej' }),
  ];

  const { blocks, missing } = bookletBlocks(speeches, 'is');

  expect(blocks.map((b) => b.text)).toEqual(['Anna', 'Hæ']);
  expect(missing).toEqual(['Björn']);
});
