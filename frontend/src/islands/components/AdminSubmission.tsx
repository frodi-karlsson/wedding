import { For, Show, type JSX } from 'solid-js';
import type { Lang } from '../../scripts/i18n';
import { translate } from '../../scripts/i18n';
import { sortPrimaryFirst } from '../../scripts/guests';
import type { GuestResponse, InviteResponse } from '../../scripts/types.gen';

interface AdminSubmissionProps {
  lang: Lang;
  invite: InviteResponse;
  guests: GuestResponse[];
  onBack: () => void;
}

export function AdminSubmission(props: AdminSubmissionProps): JSX.Element {
  const sortedGuests = () => sortPrimaryFirst(props.guests);

  return (
    <div class="admin-submission card">
      <h2 class="heading heading--md">{props.invite.name}</h2>
      <p class="submission-status">
        {props.invite.submitted ? '✓ ' : ''}
        {props.invite.submitted
          ? translate('admin_submitted', props.lang)
          : translate('admin_not_submitted', props.lang)}
      </p>

      <div class="submission-guests">
        <For each={sortedGuests()}>
          {(guest) => (
            <div class="submission-guest">
              <p class="submission-guest__name">
                {guest.name}
                <Show when={guest.is_primary}>
                  <span class="submission-tag">{translate('you_label', props.lang)}</span>
                </Show>
              </p>
              <p class="submission-guest__detail">
                {translate('dietary_label', props.lang)}: {guest.dietary_preference || '—'}
              </p>
              <p class="submission-guest__detail">
                {translate('alcohol_free_label', props.lang)}: {guest.alcohol_free ? '✓' : '—'}
              </p>
            </div>
          )}
        </For>
      </div>

      <div class="submission-message">
        <span class="submission-message__label">{translate('message_label', props.lang)}</span>
        <p>{props.invite.message || translate('admin_no_message', props.lang)}</p>
      </div>

      <button type="button" class="btn btn--ghost btn--md" onClick={() => props.onBack()}>
        {translate('admin_back', props.lang)}
      </button>
    </div>
  );
}
