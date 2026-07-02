import { createSignal, Switch, Match, Show, onMount, untrack, type JSX } from 'solid-js';
import type { Lang } from '../scripts/i18n';
import { translate } from '../scripts/i18n';
import { api } from '../scripts/api';
import {
  createEmptyForm,
  formFromInvite,
  buildShareLink,
  type AdminState,
  type InviteForm,
} from '../scripts/admin.service';
import { AdminLogin } from './components/AdminLogin';
import { AdminDashboard } from './components/AdminDashboard';
import { AdminForm } from './components/AdminForm';

export function AdminPanel(props: { lang: Lang }): JSX.Element {
  const lang = untrack(() => props.lang);

  const [state, setState] = createSignal<AdminState>({
    view: 'login',
    lang,
    invites: [],
  });

  let rootRef: HTMLDivElement | undefined;

  async function refreshDashboard(): Promise<void> {
    try {
      const response = await api.listInvites().run();
      setState((prev) => ({
        ...prev,
        view: 'dashboard',
        invites: response.invites,
        error: undefined,
        formError: undefined,
      }));
    } catch {
      setState((prev) => ({ ...prev, view: 'login', error: undefined, formError: undefined }));
    }
  }

  onMount(() => {
    rootRef?.classList.add('fade-in');
    refreshDashboard();
  });

  async function onLogin(password: string): Promise<void> {
    try {
      await api.adminLogin(password).run();
      await refreshDashboard();
    } catch {
      setState((prev) => ({ ...prev, error: translate('admin_login_error', prev.lang) }));
    }
  }

  function onNewInvite(): void {
    setState((prev) => ({
      ...prev,
      view: 'form',
      form: createEmptyForm(prev.lang),
      formError: undefined,
    }));
  }

  async function onLogout(): Promise<void> {
    try {
      await api.adminLogout().run();
    } catch {
      // ignore logout errors
    }
    setState((prev) => ({ ...prev, view: 'login', invites: [], error: undefined }));
  }

  async function onEdit(id: string): Promise<void> {
    try {
      const response = await api.getAdminInvite(id).run();
      setState((prev) => ({
        ...prev,
        view: 'form',
        form: formFromInvite(response.invite, response.guests, prev.lang),
        formError: undefined,
      }));
    } catch {
      // ignore load error
    }
  }

  async function onDelete(id: string): Promise<void> {
    if (!globalThis.confirm(translate('admin_delete_confirm', state().lang))) return;
    try {
      await api.deleteInvite(id).run();
    } catch {
      // ignore delete error
    }
    await refreshDashboard();
  }

  async function onCopyLink(id: string, linkLang: Lang, button: HTMLButtonElement): Promise<void> {
    const link = buildShareLink(id, linkLang);
    try {
      await navigator.clipboard.writeText(link);
      const original = button.textContent ?? '';
      button.textContent = translate('admin_copied', state().lang);
      setTimeout(() => {
        button.textContent = original;
      }, 1500);
    } catch {
      // ignore clipboard errors
    }
  }

  async function onFormSubmit(form: InviteForm): Promise<void> {
    const body = {
      name: form.name,
      min_plus: form.min_plus,
      max_plus: form.max_plus,
      guest_names: form.guest_names,
    };

    try {
      if (form.id === undefined) {
        const response = await api.createInvite(body).run();
        const shareLink = buildShareLink(response.invite.id, form.link_lang);
        globalThis.alert(shareLink);
      } else {
        await api.updateInvite(form.id, body).run();
      }
      await refreshDashboard();
    } catch {
      setState((prev) => ({ ...prev, formError: translate('admin_error', prev.lang) }));
    }
  }

  async function onFormCancel(): Promise<void> {
    await refreshDashboard();
  }

  return (
    <div id="admin-root" ref={(el) => { rootRef = el; }}>
      <Switch>
        <Match when={state().view === 'login'}>
          <AdminLogin lang={state().lang} error={state().error} onLogin={onLogin} />
        </Match>
        <Match when={state().view === 'dashboard'}>
          <AdminDashboard
            lang={state().lang}
            invites={state().invites}
            onNewInvite={onNewInvite}
            onLogout={onLogout}
            onEdit={onEdit}
            onDelete={onDelete}
            onCopyLink={onCopyLink}
          />
        </Match>
        <Match when={state().view === 'form'}>
          <Show when={state().form}>
            {(form) => (
              <AdminForm
                lang={state().lang}
                form={form()}
                isCreate={form().id === undefined}
                formError={state().formError}
                onSubmit={onFormSubmit}
                onCancel={onFormCancel}
              />
            )}
          </Show>
        </Match>
      </Switch>
    </div>
  );
}

export default AdminPanel;
