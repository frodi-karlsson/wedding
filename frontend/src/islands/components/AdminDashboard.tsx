import { For, type JSX } from 'solid-js';
import type { Lang } from '../../scripts/i18n';
import { translate } from '../../scripts/i18n';
import type { InviteResponse } from '../../scripts/types.gen';
import { buildShareLink } from '../../scripts/admin.service';

const ALL_LANGS: Lang[] = ['en', 'is', 'de', 'sv'];

interface AdminDashboardProps {
  lang: Lang;
  invites: InviteResponse[];
  onNewInvite: () => void;
  onLogout: () => void;
  onEdit: (id: string) => void;
  onDelete: (id: string) => void;
  onCopyLink: (id: string, lang: Lang, button: HTMLButtonElement) => void;
}

export function AdminDashboard(props: AdminDashboardProps): JSX.Element {
  return (
    <div class="admin-dashboard">
      <div class="toolbar">
        <button type="button" class="btn btn--primary btn--md" onClick={() => props.onNewInvite()}>
          {translate('admin_new_invite', props.lang)}
        </button>
        <button type="button" class="btn btn--ghost btn--md" onClick={() => props.onLogout()}>
          {translate('admin_logout', props.lang)}
        </button>
      </div>
      <table class="admin-table">
        <thead>
          <tr>
            <th>{translate('admin_id', props.lang)}</th>
            <th>{translate('admin_name_label', props.lang)}</th>
            <th>{translate('admin_min_label', props.lang)}</th>
            <th>{translate('admin_max_label', props.lang)}</th>
            <th>{translate('admin_submitted', props.lang)}</th>
            <th>{translate('admin_link', props.lang)}</th>
            <th>{translate('admin_actions', props.lang)}</th>
          </tr>
        </thead>
        <tbody>
          <For each={props.invites}>
            {(invite) => (
              <tr>
                <td>{invite.id}</td>
                <td>{invite.name}</td>
                <td>{invite.min_plus}</td>
                <td>{invite.max_plus}</td>
                <td>{invite.submitted ? '✓' : '—'}</td>
                <td>
                  <a href={buildShareLink(invite.id, props.lang)}>{buildShareLink(invite.id, props.lang)}</a>
                </td>
                <td class="actions">
                  <select class="link-lang" data-id={invite.id}>
                    <For each={ALL_LANGS}>
                      {(l) => <option value={l} selected={l === props.lang}>{l}</option>}
                    </For>
                  </select>
                  <button
                    type="button"
                    class="btn btn--ghost btn--sm"
                    onClick={(e) => {
                      const select = e.currentTarget
                        .closest('tr')
                        ?.querySelector<HTMLSelectElement>(`select.link-lang[data-id="${invite.id}"]`);
                      const linkLang = (select?.value as Lang) ?? props.lang;
                      props.onCopyLink(invite.id, linkLang, e.currentTarget);
                    }}
                  >
                    {translate('admin_copy_link', props.lang)}
                  </button>
                  <button
                    type="button"
                    class="btn btn--ghost btn--sm"
                    onClick={() => props.onEdit(invite.id)}
                  >
                    {translate('admin_edit', props.lang)}
                  </button>
                  <button
                    type="button"
                    class="btn btn--ghost btn--sm"
                    onClick={() => props.onDelete(invite.id)}
                  >
                    {translate('admin_delete', props.lang)}
                  </button>
                </td>
              </tr>
            )}
          </For>
        </tbody>
      </table>
    </div>
  );
}
