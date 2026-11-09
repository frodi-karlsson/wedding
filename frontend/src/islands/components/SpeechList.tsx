import { createSignal, For, Show, untrack, type JSX } from 'solid-js';
import { LOCALES, localePrefix, translate, type Lang } from '../../scripts/i18n';
import { langProgress, textStatus } from '../../scripts/speeches.service';
import type { CreateSpeechRequest, SpeechResponse } from '../../scripts/types.gen';

interface SpeechListProps {
  lang: Lang;
  speeches: SpeechResponse[];
  error?: string;
  onCreate: (details: CreateSpeechRequest) => void;
  onOpen: (id: number) => void;
  onMove: (id: number, delta: -1 | 1) => void;
  onDelete: (id: number) => void;
  onLogout: () => void;
}

export function SpeechList(props: SpeechListProps): JSX.Element {
  const lang = untrack(() => props.lang);
  const t = (key: string) => translate(key, lang);

  const [creating, setCreating] = createSignal(false);
  const [speaker, setSpeaker] = createSignal('');
  const [title, setTitle] = createSignal('');
  const [originalLang, setOriginalLang] = createSignal<string>(lang);

  function create(e: Event) {
    e.preventDefault();
    props.onCreate({ speaker: speaker().trim(), title: title().trim(), original_lang: originalLang() });
    setCreating(false);
    setSpeaker('');
    setTitle('');
  }

  function progressLabel(code: string): string {
    const { done, total } = langProgress(props.speeches, code);
    return t('speeches_progress').replace('{done}', String(done)).replace('{total}', String(total));
  }

  return (
    <div class="admin-dashboard speech-list">
      <div class="toolbar">
        <button type="button" class="btn btn--primary btn--md" onClick={() => setCreating(true)}>
          {t('speeches_new')}
        </button>
        <button type="button" class="btn btn--ghost btn--md" onClick={() => props.onLogout()}>
          {t('admin_logout')}
        </button>
      </div>

      <Show when={creating()}>
        <form class="admin-form card speech-create" onSubmit={create}>
          <label>
            <span>{t('speeches_speaker_label')}</span>
            <input type="text" name="speaker" required value={speaker()} onInput={(e) => setSpeaker(e.currentTarget.value)} />
          </label>
          <label>
            <span>{t('speeches_title_label')}</span>
            <input type="text" name="title" value={title()} onInput={(e) => setTitle(e.currentTarget.value)} />
          </label>
          <label>
            <span>{t('speeches_original_label')}</span>
            <select name="original_lang" value={originalLang()} onChange={(e) => setOriginalLang(e.currentTarget.value)}>
              <For each={LOCALES}>{(l) => <option value={l.code}>{l.label}</option>}</For>
            </select>
          </label>
          <div class="form-actions">
            <button type="submit" class="btn btn--primary btn--md">{t('speeches_create')}</button>
            <button type="button" class="btn btn--ghost btn--md" onClick={() => setCreating(false)}>
              {t('admin_cancel')}
            </button>
          </div>
        </form>
      </Show>

      {props.error && <p class="error">{props.error}</p>}

      <ul class="speech-progress">
        <For each={LOCALES}>
          {(l) => (
            <li>
              <strong>{l.label}</strong> {progressLabel(l.code)}{' '}
              <a href={`${localePrefix(lang)}/speeches/booklet?lang=${l.code}`}>{t('speeches_booklet')} ↗</a>
            </li>
          )}
        </For>
      </ul>

      <Show when={props.speeches.length > 0} fallback={<p>{t('speeches_empty')}</p>}>
        <div class="admin-table-wrap">
          <table class="admin-table">
            <thead>
              <tr>
                <th>#</th>
                <th>{t('speeches_speaker_label')}</th>
                <th>{t('speeches_original_label')}</th>
                <For each={LOCALES}>{(l) => <th>{l.code.toUpperCase()}</th>}</For>
                <th>{t('admin_actions')}</th>
              </tr>
            </thead>
            <tbody>
              <For each={props.speeches}>
                {(speech, index) => (
                  <tr>
                    <td>{index() + 1}</td>
                    <td>
                      {speech.speaker}
                      {speech.title ? <span class="speech-subtitle">{speech.title}</span> : null}
                    </td>
                    <td>{speech.original_lang.toUpperCase()}</td>
                    <For each={LOCALES}>
                      {(l) => {
                        const text = speech.texts.find((x) => x.lang === l.code);
                        const status = text ? textStatus(text) : 'missing';
                        return (
                          <td>
                            <span class={`status-chip status-chip--${status}`}>{t(`speeches_status_${status}`)}</span>
                          </td>
                        );
                      }}
                    </For>
                    <td class="actions">
                      <div class="row-actions">
                        <button type="button" class="btn btn--ghost btn--sm" onClick={() => props.onOpen(speech.id)}>
                          {t('speeches_open')}
                        </button>
                        <button
                          type="button"
                          class="btn btn--ghost btn--sm"
                          aria-label={t('speeches_move_up')}
                          disabled={index() === 0}
                          onClick={() => props.onMove(speech.id, -1)}
                        >
                          ↑
                        </button>
                        <button
                          type="button"
                          class="btn btn--ghost btn--sm"
                          aria-label={t('speeches_move_down')}
                          disabled={index() === props.speeches.length - 1}
                          onClick={() => props.onMove(speech.id, 1)}
                        >
                          ↓
                        </button>
                        <button type="button" class="btn btn--ghost btn--sm" onClick={() => props.onDelete(speech.id)}>
                          {t('admin_delete')}
                        </button>
                      </div>
                    </td>
                  </tr>
                )}
              </For>
            </tbody>
          </table>
        </div>
      </Show>
    </div>
  );
}
