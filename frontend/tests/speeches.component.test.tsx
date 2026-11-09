import { test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { SpeechEditor } from '../src/islands/components/SpeechEditor';
import { SpeechesPanel } from '../src/islands/SpeechesPanel';
import { api } from '../src/scripts/api';
import type { SpeechResponse, SpeechTextResponse } from '../src/scripts/types.gen';
import { mockTask, mockFailingTask, settleTasks } from './tasks';

function text(lang: string, overrides: Partial<SpeechTextResponse> = {}): SpeechTextResponse {
  return { lang, body: '', done: false, version: 1, ...overrides };
}

function speech(texts: Partial<Record<string, Partial<SpeechTextResponse>>> = {}): SpeechResponse {
  return {
    id: 1,
    position: 1,
    speaker: 'Anna',
    title: 'Toast',
    original_lang: 'sv',
    version: 1,
    texts: ['sv', 'is', 'en', 'de'].map((lang) => text(lang, texts[lang])),
  };
}

const conflictError = Object.assign(new Error('Conflict'), { details: { status: 409 } });

beforeEach(() => {
  vi.stubGlobal('location', { origin: 'https://example.com', pathname: '/en/speeches', search: '', hash: '' });
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

function renderEditor(sp: SpeechResponse = speech()): void {
  render(() => <SpeechEditor lang="en" speech={sp} onBack={vi.fn()} />);
}

function textArea(langLabel: string): HTMLTextAreaElement {
  return screen.getByRole('textbox', { name: `Text: ${langLabel}` }) as HTMLTextAreaElement;
}

test('should show the original next to the translation being edited', () => {
  renderEditor(speech({ sv: { body: 'Kära vänner' } }));

  fireEvent.click(screen.getByRole('tab', { name: 'Íslenska' }));

  expect(screen.getByText('Kära vänner')).toBeInTheDocument();
  expect(textArea('Íslenska')).toBeInTheDocument();
});

test('should save a text with the version it was loaded at', async () => {
  vi.spyOn(api, 'saveSpeechText').mockReturnValueOnce(mockTask(text('sv', { body: 'Kära vänner', version: 2 })));
  renderEditor();
  fireEvent.input(textArea('Svenska'), { target: { value: 'Kära vänner' } });

  fireEvent.click(screen.getByRole('button', { name: 'Save' }));
  await settleTasks();

  expect(api.saveSpeechText).toHaveBeenCalledWith(1, 'sv', { body: 'Kära vänner', done: false, version: 1 });
  expect(screen.getByText('Saved')).toBeInTheDocument();
});

test('should show their text when someone else saved first', async () => {
  vi.spyOn(api, 'saveSpeechText').mockReturnValueOnce(mockFailingTask(conflictError));
  vi.spyOn(api, 'listSpeeches').mockReturnValueOnce(
    mockTask({ speeches: [speech({ sv: { body: 'Theirs', version: 2 } })] }),
  );
  renderEditor();
  fireEvent.input(textArea('Svenska'), { target: { value: 'Mine' } });

  fireEvent.click(screen.getByRole('button', { name: 'Save' }));
  await settleTasks();

  expect(screen.getByText(/Someone else saved this/)).toBeInTheDocument();
  expect(screen.getByText('Theirs')).toBeInTheDocument();
  expect(textArea('Svenska').value).toBe('Mine');
});

test('should save mine over theirs after choosing keep mine', async () => {
  vi.spyOn(api, 'saveSpeechText')
    .mockReturnValueOnce(mockFailingTask(conflictError))
    .mockReturnValueOnce(mockTask(text('sv', { body: 'Mine', version: 3 })));
  vi.spyOn(api, 'listSpeeches').mockReturnValueOnce(
    mockTask({ speeches: [speech({ sv: { body: 'Theirs', version: 2 } })] }),
  );
  renderEditor();
  fireEvent.input(textArea('Svenska'), { target: { value: 'Mine' } });
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));
  await settleTasks();

  fireEvent.click(screen.getByRole('button', { name: 'Keep mine' }));
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));
  await settleTasks();

  expect(api.saveSpeechText).toHaveBeenLastCalledWith(1, 'sv', { body: 'Mine', done: false, version: 2 });
  expect(screen.queryByText('Theirs')).not.toBeInTheDocument();
});

test('should replace my text with theirs after choosing use theirs', async () => {
  vi.spyOn(api, 'saveSpeechText').mockReturnValueOnce(mockFailingTask(conflictError));
  vi.spyOn(api, 'listSpeeches').mockReturnValueOnce(
    mockTask({ speeches: [speech({ sv: { body: 'Theirs', version: 2 } })] }),
  );
  renderEditor();
  fireEvent.input(textArea('Svenska'), { target: { value: 'Mine' } });
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));
  await settleTasks();

  fireEvent.click(screen.getByRole('button', { name: 'Use theirs' }));

  expect(textArea('Svenska').value).toBe('Theirs');
  expect(screen.queryByText(/Someone else saved this/)).not.toBeInTheDocument();
});

test('should show the login form without a session', async () => {
  vi.spyOn(api, 'session').mockReturnValueOnce(mockFailingTask(new Error('Unauthorized')));
  render(() => <SpeechesPanel lang="en" />);

  await settleTasks();

  expect(screen.getByText('Log in')).toBeInTheDocument();
});

test('should list speeches with each language\'s status', async () => {
  vi.spyOn(api, 'session').mockReturnValueOnce(mockTask({ role: 'translator' }));
  vi.spyOn(api, 'listSpeeches').mockReturnValueOnce(
    mockTask({ speeches: [speech({ sv: { body: 'Hej', done: true }, is: { body: 'Hæ' } })] }),
  );
  render(() => <SpeechesPanel lang="en" />);

  await settleTasks();

  expect(screen.getByText('Anna')).toBeInTheDocument();
  const chips = [...document.querySelectorAll('.status-chip')].map((c) => c.textContent);
  expect(chips).toEqual(['Done', 'Missing', 'Draft', 'Missing']);
});

test('should add a speech from the new speech form', async () => {
  vi.spyOn(api, 'session').mockReturnValueOnce(mockTask({ role: 'translator' }));
  vi.spyOn(api, 'listSpeeches').mockReturnValueOnce(mockTask({ speeches: [] }));
  vi.spyOn(api, 'createSpeech').mockReturnValueOnce(mockTask(speech()));
  render(() => <SpeechesPanel lang="en" />);
  await settleTasks();
  fireEvent.click(screen.getByText('+ New speech'));
  fireEvent.input(screen.getByRole('textbox', { name: 'Speaker' }), { target: { value: 'Anna' } });

  fireEvent.submit(document.querySelector('form.speech-create') as HTMLFormElement);
  await settleTasks();

  expect(api.createSpeech).toHaveBeenCalledWith({ speaker: 'Anna', title: '', original_lang: 'en' });
  expect(screen.getByText('Anna')).toBeInTheDocument();
});
