import type { Block } from './booklet';

const tagFor: Record<Block['kind'], string> = { heading: 'h2', subheading: 'p', paragraph: 'p' };

/** The element a block renders as, shared by measuring and the final pages. */
export function blockTag(block: Block): string {
  return tagFor[block.kind];
}

export function blockClass(block: Block): string {
  return block.continues ? `booklet-${block.kind} booklet-${block.kind}--continues` : `booklet-${block.kind}`;
}

// A paragraph split across pages keeps at least this many lines on each side,
// so no page starts or ends with a lone line.
const MIN_LINES = 2;

function lineCount(el: HTMLElement): number {
  const lineHeight = Number.parseFloat(getComputedStyle(el).lineHeight);
  return Math.round(el.getBoundingClientRect().height / lineHeight);
}

function blockElement(block: Block): HTMLElement {
  const el = document.createElement(blockTag(block));
  el.className = blockClass(block);
  el.textContent = block.text;
  return el;
}

/**
 * Flows blocks onto pages by measuring them in `body`, an element styled and
 * sized exactly like a page's text area. A block that would overflow moves to
 * the next page; a paragraph too long for an empty page is split between
 * words. Headings never end a page: they move on with the text they introduce.
 */
export function paginate(body: HTMLElement, blocks: Block[]): Block[][] {
  const pages: Block[][] = [];
  let current: Block[] = [];
  const queue = [...blocks];
  const fits = () => body.scrollHeight <= body.clientHeight;

  function newPage() {
    if (current.length > 0) pages.push(current);
    current = [];
    body.replaceChildren();
  }

  function place(block: Block) {
    current.push(block);
    body.append(blockElement(block));
  }

  while (queue.length > 0) {
    const block = queue.shift() as Block;
    if (block.startsPage && current.length > 0) newPage();

    const el = blockElement(block);
    body.append(el);
    if (fits()) {
      current.push(block);
      continue;
    }
    const totalLines = lineCount(el);
    el.remove();

    if (block.kind === 'paragraph' && current.at(-1)?.kind === 'paragraph') {
      let split = splitToFit(body, block);
      if (totalLines - split.headLines < MIN_LINES) {
        // Too little would carry over: leave more for the next page instead.
        split = splitToFit(body, block, 0, totalLines - MIN_LINES);
      }
      if (split.headLines >= MIN_LINES && totalLines - split.headLines >= MIN_LINES) {
        if (split.head) place(split.head);
        newPage();
        if (split.tail) queue.unshift(split.tail);
        continue;
      }
    }

    const onlyHeadings = current.every((b) => b.kind !== 'paragraph');
    if (onlyHeadings) {
      if (block.kind !== 'paragraph') {
        // A heading taller than a page has nowhere better to go.
        place(block);
        newPage();
        continue;
      }
      // The page holds no text yet, so moving on would not help: start the
      // paragraph here with at least one word.
      const { head, tail } = splitToFit(body, block, 1);
      if (head) place(head);
      newPage();
      if (tail) queue.unshift(tail);
      continue;
    }

    const carried: Block[] = [];
    while (current.length > 0 && current.at(-1)?.kind !== 'paragraph') {
      carried.unshift(current.pop() as Block);
    }
    newPage();
    queue.unshift(...carried.map((b) => ({ ...b, startsPage: false })), { ...block, startsPage: false });
  }
  newPage();
  return pages;
}

/**
 * Splits a paragraph into the longest run of words that still fits on the page
 * and the rest, reporting how many lines the fitting part takes. `minWords`
 * forces a floor, so a single enormous word cannot stall the layout, and
 * `maxLines` caps the part that stays on this page.
 */
function splitToFit(
  body: HTMLElement,
  block: Block,
  minWords = 0,
  maxLines = Number.POSITIVE_INFINITY,
): { head: Block | null; tail: Block | null; headLines: number } {
  const words = block.text.match(/\S+\s*/g) ?? [];
  const probe = blockElement({ ...block, text: '' });
  body.append(probe);
  let lo = 0;
  let hi = words.length;
  while (lo < hi) {
    const mid = Math.ceil((lo + hi) / 2);
    probe.textContent = words.slice(0, mid).join('');
    if (body.scrollHeight <= body.clientHeight && lineCount(probe) <= maxLines) lo = mid;
    else hi = mid - 1;
  }
  const keep = Math.max(lo, minWords);
  probe.textContent = words.slice(0, keep).join('');
  const headLines = keep > 0 ? lineCount(probe) : 0;
  probe.remove();
  const head = words.slice(0, keep).join('').trimEnd();
  const tail = words.slice(keep).join('');
  return {
    head: head ? { kind: 'paragraph', text: head, continues: tail.length > 0 } : null,
    tail: tail ? { kind: 'paragraph', text: tail } : null,
    headLines,
  };
}
