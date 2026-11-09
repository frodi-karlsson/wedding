import { createSignal, Match, onMount, Show, Switch, untrack, type JSX } from 'solid-js';
import { translate, type Lang } from '../scripts/i18n';
import { api } from '../scripts/api';
import type { CreateSpeechRequest, SpeechResponse } from '../scripts/types.gen';
import { AdminLogin } from './components/AdminLogin';
import { SpeechList } from './components/SpeechList';
import { SpeechEditor } from './components/SpeechEditor';

export function SpeechesPanel(props: { lang: Lang }): JSX.Element {
  const lang = untrack(() => props.lang);
  const t = (key: string) => translate(key, lang);

  // undefined until the session check resolves, so login never flashes first.
  const [authed, setAuthed] = createSignal<boolean | undefined>(undefined);
  const [speeches, setSpeeches] = createSignal<SpeechResponse[]>([]);
  const [openId, setOpenId] = createSignal<number | undefined>(undefined);
  const [error, setError] = createSignal<string | undefined>(undefined);
  const [loginError, setLoginError] = createSignal<string | undefined>(undefined);

  const openSpeech = () => speeches().find((s) => s.id === openId());

  async function load(): Promise<void> {
    try {
      setSpeeches((await api.listSpeeches().run()).speeches);
      setError(undefined);
    } catch {
      setError(t('speeches_load_error'));
    }
  }

  onMount(() => {
    api
      .session()
      .run()
      .then(async () => {
        await load();
        setAuthed(true);
      })
      .catch(() => setAuthed(false));
  });

  async function onLogin(password: string): Promise<void> {
    try {
      await api.login(password).run();
    } catch {
      setLoginError(t('admin_login_error'));
      return;
    }
    await load();
    setAuthed(true);
  }

  async function onLogout(): Promise<void> {
    try {
      await api.logout().run();
    } finally {
      setAuthed(false);
    }
  }

  async function onCreate(details: CreateSpeechRequest): Promise<void> {
    try {
      const created = await api.createSpeech(details).run();
      setSpeeches((prev) => [...prev, created]);
    } catch {
      setError(t('speeches_save_error'));
    }
  }

  async function onMove(id: number, delta: -1 | 1): Promise<void> {
    try {
      setSpeeches((await api.moveSpeech(id, delta).run()).speeches);
    } catch {
      setError(t('speeches_save_error'));
    }
  }

  async function onDelete(id: number): Promise<void> {
    if (!globalThis.confirm(t('speeches_delete_confirm'))) return;
    try {
      await api.deleteSpeech(id).run();
      setSpeeches((prev) => prev.filter((s) => s.id !== id));
    } catch {
      setError(t('speeches_save_error'));
    }
  }

  async function onBack(): Promise<void> {
    setOpenId(undefined);
    await load();
  }

  return (
    <div id="speeches-root">
      <Show when={authed() !== undefined}>
        <Show when={authed()} fallback={<AdminLogin lang={lang} error={loginError()} onLogin={onLogin} />}>
          <Switch>
            <Match when={openSpeech()}>
              {(speech) => <SpeechEditor lang={lang} speech={speech()} onBack={onBack} />}
            </Match>
            <Match when={openId() === undefined}>
              <SpeechList
                lang={lang}
                speeches={speeches()}
                error={error()}
                onCreate={onCreate}
                onOpen={setOpenId}
                onMove={onMove}
                onDelete={onDelete}
                onLogout={onLogout}
              />
            </Match>
          </Switch>
        </Show>
      </Show>
    </div>
  );
}

export default SpeechesPanel;
