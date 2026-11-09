import { test, expect, type Page } from '@playwright/test';

const sentence = 'Kæru vinir, það er mér sannur heiður að fá að standa hér í dag og segja nokkur orð um brúðhjónin. ';

function speech(id: number, speaker: string, title: string, isBody: string) {
  return {
    id,
    position: id,
    speaker,
    title,
    original_lang: 'is',
    version: 1,
    texts: ['sv', 'is', 'en', 'de'].map((lang) => ({ lang, body: lang === 'is' ? isBody : '', done: true, version: 1 })),
  };
}

const speeches = [
  speech(1, 'Anna', 'Ræða brúðarinnar', Array.from({ length: 6 }, () => sentence.repeat(8)).join('\n\n')),
  speech(2, 'Björn', '', 'Skál fyrir brúðhjónunum!'),
  speech(3, 'Carla', '', ''),
];

async function openBooklet(page: Page) {
  await page.route('**/*', (route) => {
    const request = route.request();
    if (request.resourceType() !== 'fetch') return route.continue();
    if (new URL(request.url()).pathname === '/speeches') return route.fulfill({ json: { speeches } });
    return route.fulfill({ status: 404, json: { error: 'not found' } });
  });
  await page.goto('/en/speeches/booklet?lang=is');
  await expect(page.locator('.booklet-preview .booklet-page[data-page]').first()).toBeVisible();
}

function previewPages(page: Page) {
  return page.locator('.booklet-preview .booklet-page[data-page]');
}

test('booklet lays speeches out on A5 pages padded to whole sheets', async ({ page }) => {
  await openBooklet(page);

  const count = await previewPages(page).count();

  expect(count % 4).toBe(0);
  expect(count).toBeGreaterThanOrEqual(8);
  await expect(page.locator('.booklet-preview .booklet-cover__title')).toHaveText('Ræður');
});

test('booklet text never overflows a page and each speech starts a page', async ({ page }) => {
  await openBooklet(page);

  const bodies = await page.locator('.booklet-preview .booklet-page__body').evaluateAll((els) =>
    els.map((el) => ({
      overflows: el.scrollHeight > el.clientHeight + 1,
      headingNotFirst: [...el.querySelectorAll('.booklet-heading')].some((h) => h !== el.firstElementChild),
      endsOnHeading: el.lastElementChild?.classList.contains('booklet-heading') && el.children.length > 1,
    })),
  );

  expect(bodies.length).toBeGreaterThan(2);
  expect(bodies.filter((b) => b.overflows)).toEqual([]);
  expect(bodies.filter((b) => b.headingNotFirst)).toEqual([]);
  expect(bodies.filter((b) => b.endsOnHeading)).toEqual([]);
});

test('booklet warns about speeches with no text in its language', async ({ page }) => {
  await openBooklet(page);

  await expect(page.getByText(/no text in this language: Carla/)).toBeVisible();
});

test('booklet imposes pages onto A4 sheets for saddle stitching', async ({ page }) => {
  await openBooklet(page);
  const count = await previewPages(page).count();

  const sheets = await page.locator('.booklet-print .booklet-sheet--a4').evaluateAll((els) =>
    els.map((el) => [...el.querySelectorAll('.booklet-page')].map((p) => Number(p.getAttribute('data-page')))),
  );

  expect(sheets).toHaveLength(count / 2);
  expect(sheets[0]).toEqual([count, 1]);
  expect(sheets[1]).toEqual([2, count - 1]);
});

test('booklet prints only the sheets', async ({ page }) => {
  await openBooklet(page);

  await page.emulateMedia({ media: 'print' });

  await expect(page.locator('.booklet-preview')).toBeHidden();
  await expect(page.locator('.booklet-print .booklet-sheet').first()).toBeVisible();
});
