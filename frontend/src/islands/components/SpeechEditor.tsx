import { createSignal, For, Show, onCleanup, untrack, type JSX } from 'solid-js';
import { LOCALES, translate, type Lang } from '../../scripts/i18n';
import { api, isConflict } from '../../scripts/api';
import {
  afterSave,
  draftFrom,
  isDirty,
  keepMine,
  textStatus,
  useTheirs,
  withConflict,
  type TextDraft,
} from '../../scripts/speeches.service';
import type { SpeechResponse } from '../../scripts/types.gen';

interface SpeechEditorProps {
  lang: Lang;
  speech: SpeechResponse;
  onBack: () => void;
}

type SaveState = 'idle' | 'saving' | 'saved' | 'error';

function langLabel(code: string): string {
  return LOCALES.find((l) => l.code === code)?.label ?? code;
}

export function SpeechEditor(props: SpeechEditorProps): JSX.Element {
  // The editor is remounted for every speech it opens, so it owns its copy.
  const initial = untrack(() => props.speech);
  const lang = untrack(() => props.lang);
  const t = (key: string) => translate(key, lang);

  const [speaker, setSpeaker] = createSignal(initial.speaker);
  const [title, setTitle] = createSignal(initial.title);
  const [originalLang, setOriginalLang] = createSignal(initial.original_lang);
  const [detailsVersion, setDetailsVersion] = createSignal(initial.version);
  const [detailsState, setDetailsState] = createSignal<SaveState>('idle');
  const [detailsConflict, setDetailsConflict] = createSignal(false);

  const [drafts, setDrafts] = createSignal<Record<string, TextDraft>>(
    Object.fromEntries(initial.texts.map((text) => [text.lang, draftFrom(text)])),
  );
  const [saveStates, setSaveStates] = createSignal<Record<string, SaveState>>({});
  const [selected, setSelected] = createSignal(initial.original_lang);

  // The original language comes first, then the rest in site order.
  const tabs = () => [originalLang(), ...LOCALES.map((l) => l.code).filter((c) => c !== originalLang())];
  const draft = (code: string) => drafts()[code];
  const anyDirty = () => Object.values(drafts()).some(isDirty);

  function updateDraft(code: string, change: (d: TextDraft) => TextDraft) {
    setDrafts((prev) => ({ ...prev, [code]: change(prev[code]) }));
  }

  function setSaveState(code: string, state: SaveState) {
    setSaveStates((prev) => ({ ...prev, [code]: state }));
  }

  function warnBeforeLeaving(e: BeforeUnloadEvent) {
    if (anyDirty()) e.preventDefault();
  }
  globalThis.addEventListener('beforeunload', warnBeforeLeaving);
  onCleanup(() => globalThis.removeEventListener('beforeunload', warnBeforeLeaving));

  async function saveText(code: string): Promise<void> {
    const current = draft(code);
    setSaveState(code, 'saving');
    try {
      const saved = await api
        .saveSpeechText(initial.id, code, { body: current.body, done: current.done, version: current.saved.version })
        .run();
      updateDraft(code, (d) => afterSave(d, saved));
      setSaveState(code, 'saved');
    } catch (error) {
      if (!isConflict(error)) {
        setSaveState(code, 'error');
        return;
      }
      const latest = await loadLatest();
      const theirs = latest?.texts.find((text) => text.lang === code);
      if (theirs) updateDraft(code, (d) => withConflict(d, theirs));
      setSaveState(code, theirs ? 'idle' : 'error');
    }
  }

  async function loadLatest(): Promise<SpeechResponse | undefined> {
    try {
      const list = await api.listSpeeches().run();
      return list.speeches.find((s) => s.id === initial.id);
    } catch {
      return undefined;
    }
  }

  async function saveDetails(e: Event): Promise<void> {
    e.preventDefault();
    setDetailsState('saving');
    setDetailsConflict(false);
    try {
      const saved = await api
        .updateSpeech(initial.id, {
          speaker: speaker(),
          title: title(),
          original_lang: originalLang(),
          version: detailsVersion(),
        })
        .run();
      setDetailsVersion(saved.version);
      setDetailsState('saved');
    } catch (error) {
      const latest = isConflict(error) ? await loadLatest() : undefined;
      if (!latest) {
        setDetailsState('error');
        return;
      }
      setSpeaker(latest.speaker);
      setTitle(latest.title);
      setOriginalLang(latest.original_lang);
      setDetailsVersion(latest.version);
      setDetailsConflict(true);
      setDetailsState('idle');
    }
  }

  function back() {
    if (anyDirty() && !globalThis.confirm(t('speeches_discard_confirm'))) return;
    props.onBack();
  }

  function statusLabel(code: string): string {
    const state = saveStates()[code];
    if (state === 'saving') return t('speeches_saving');
    if (state === 'error') return t('speeches_save_error');
    if (isDirty(draft(code))) return t('speeches_unsaved');
    if (state === 'saved') return t('speeches_saved');
    return '';
  }

  return (
    <div class="speech-editor">
      <button type="button" class="btn btn--ghost btn--sm" onClick={back}>
        {t('admin_back')}
      </button>

      <form class="speech-details card" onSubmit={saveDetails}>
        <h2 class="heading heading--md">{t('speeches_details')}</h2>
        <label>
          <span>{t('speeches_speaker_label')}</span>
          <input type="text" required value={speaker()} onInput={(e) => setSpeaker(e.currentTarget.value)} />
        </label>
        <label>
          <span>{t('speeches_title_label')}</span>
          <input type="text" value={title()} onInput={(e) => setTitle(e.currentTarget.value)} />
        </label>
        <label>
          <span>{t('speeches_original_label')}</span>
          <select value={originalLang()} onChange={(e) => setOriginalLang(e.currentTarget.value)}>
            <For each={LOCALES}>{(l) => <option value={l.code}>{l.label}</option>}</For>
          </select>
        </label>
        <Show when={detailsConflict()}>
          <p class="error">{t('speeches_details_conflict')}</p>
        </Show>
        <div class="form-actions">
          <button type="submit" class="btn btn--secondary btn--sm" disabled={detailsState() === 'saving'}>
            {t('speeches_save_details')}
          </button>
          <span class="save-state">
            {detailsState() === 'saved' ? t('speeches_saved') : detailsState() === 'error' ? t('speeches_save_error') : ''}
          </span>
        </div>
      </form>

      <div class="speech-tabs" role="tablist">
        <For each={tabs()}>
          {(code) => (
            <button
              type="button"
              role="tab"
              class={`speech-tab speech-tab--${textStatus(draft(code).saved)}`}
              aria-selected={selected() === code}
              onClick={() => setSelected(code)}
            >
              {langLabel(code)}
              {code === originalLang() ? ` (${t('speeches_original')})` : ''}
              {isDirty(draft(code)) ? ' •' : ''}
            </button>
          )}
        </For>
      </div>

      <div class="speech-panes">
        <Show when={selected() !== originalLang()}>
          <section class="speech-pane speech-pane--reference">
            <h3>{`${t('speeches_original')}: ${langLabel(originalLang())}`}</h3>
            <p class="speech-reference">{draft(originalLang()).saved.body}</p>
          </section>
        </Show>
        <section class="speech-pane card">
          <label>
            <span>{`${t('speeches_text_label')}: ${langLabel(selected())}`}</span>
            <textarea
              rows="16"
              value={draft(selected()).body}
              onInput={(e) => {
                const value = e.currentTarget.value;
                updateDraft(selected(), (d) => ({ ...d, body: value }));
              }}
            />
          </label>
          <label class="checkbox">
            <input
              type="checkbox"
              checked={draft(selected()).done}
              onChange={(e) => {
                const value = e.currentTarget.checked;
                updateDraft(selected(), (d) => ({ ...d, done: value }));
              }}
            />
            <span>{t('speeches_done_label')}</span>
          </label>
          <Show when={draft(selected()).conflict}>
            {(theirs) => (
              <div class="speech-conflict">
                <p>{t('speeches_conflict')}</p>
                <p class="speech-reference">{theirs().body}</p>
                <div class="form-actions">
                  <button type="button" class="btn btn--secondary btn--sm" onClick={() => updateDraft(selected(), keepMine)}>
                    {t('speeches_keep_mine')}
                  </button>
                  <button type="button" class="btn btn--ghost btn--sm" onClick={() => updateDraft(selected(), useTheirs)}>
                    {t('speeches_use_theirs')}
                  </button>
                </div>
              </div>
            )}
          </Show>
          <div class="form-actions">
            <button
              type="button"
              class="btn btn--primary btn--md"
              disabled={!isDirty(draft(selected())) || Boolean(draft(selected()).conflict) || saveStates()[selected()] === 'saving'}
              onClick={() => saveText(selected())}
            >
              {t('admin_save')}
            </button>
            <span class="save-state">{statusLabel(selected())}</span>
          </div>
        </section>
      </div>
    </div>
  );
}
