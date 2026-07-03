import { For, type JSX } from 'solid-js';
import type { Lang } from '../../scripts/i18n';
import { translate } from '../../scripts/i18n';
import type { InviteResponse } from '../../scripts/types.gen';
import { buildShareLink } from '../../scripts/admin.service';

interface AdminDashboardProps {
  lang: Lang;
  invites: InviteResponse[];
  error?: string;
  onNewInvite: () => void;
  onLogout: () => void;
  onView: (id: string) => void;
  onEdit: (id: string) => void;
  onDelete: (id: string) => void;
  onGetInvite: (id: string) => void;
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
      {props.error && <p class="error">{props.error}</p>}
      <div class="admin-table-wrap">
      <table class="admin-table">
        <thead>
          <tr>
            <th>{translate('admin_id', props.lang)}</th>
            <th>{translate('admin_name_label', props.lang)}</th>
            <th>{translate('admin_min_label', props.lang)}</th>
            <th>{translate('admin_max_label', props.lang)}</th>
            <th>{translate('admin_submitted', props.lang)}</th>
            <th>{translate('admin_actions', props.lang)}</th>
          </tr>
        </thead>
        <tbody>
          <For each={props.invites}>
            {(invite) => (
              <tr>
                <td>
                  <a class="invite-id" href={buildShareLink(globalThis.location.origin, invite.id, props.lang)} title={invite.id}>{invite.id.slice(0, 8)}…</a>
                </td>
                <td>{invite.name}</td>
                <td>{invite.min_plus}</td>
                <td>{invite.max_plus}</td>
                <td>{invite.submitted ? '✓' : '—'}</td>
                <td class="actions">
                  <button
                    type="button"
                    class="btn btn--ghost btn--sm"
                    onClick={() => props.onGetInvite(invite.id)}
                  >
                    {translate('admin_get_invite', props.lang)}
                  </button>
                  <button
                    type="button"
                    class="btn btn--ghost btn--sm"
                    onClick={() => props.onView(invite.id)}
                  >
                    {translate('admin_view', props.lang)}
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
    </div>
  );
}
