import { createSignal, For, Index, Show, untrack, type JSX } from 'solid-js';
import { translate, LOCALE_CODES, type Lang } from '../../scripts/i18n';
import type { InviteForm } from '../../scripts/admin.service';

interface AdminFormProps {
  lang: Lang;
  form: InviteForm;
  isCreate: boolean;
  formError?: string;
  onSubmit: (form: InviteForm) => void;
  onCancel: () => void;
}

export function AdminForm(props: AdminFormProps): JSX.Element {
  // Local mutable copy of the form. The parent owns the canonical state.
  // The component is unmounted/remounted on every view transition (create → dashboard,
  // edit → dashboard), so local signals initialized from props.form are correct and
  // never need to resync mid-lifecycle.
  const form = untrack(() => props.form);

  const [name, setName] = createSignal(form.guest_names[0] ?? '');
  const [minPlus, setMinPlus] = createSignal(String(form.min_plus));
  const [maxPlus, setMaxPlus] = createSignal(String(form.max_plus));
  const [guestNames, setGuestNames] = createSignal<string[]>(form.guest_names.slice(1));
  const [linkLang, setLinkLang] = createSignal<Lang>(form.link_lang);
  const [countError, setCountError] = createSignal('');
  const [group, setGroup] = createSignal(form.group);

  // Standard invites: preset names are "plus" guests, capped to [min, max].
  // Group invites: the names are co-primaries with no plus allowance, so the
  // add/remove controls are unconstrained.
  const canAddName = () => {
    if (group()) return true;
    const max = Number(maxPlus());
    return !Number.isNaN(max) && guestNames().length < max;
  };
  const canRemoveName = () => {
    if (group()) return true;
    const min = Number(minPlus());
    return guestNames().length > (Number.isNaN(min) ? 0 : min);
  };

  const heading = () =>
    translate(props.isCreate ? 'admin_new_invite' : 'admin_edit', props.lang);
  const submitLabel = () =>
    translate(props.isCreate ? 'admin_create' : 'admin_save', props.lang);

  function addName() {
    if (!canAddName()) return;
    setCountError('');
    setGuestNames((prev) => [...prev, '']);
  }

  function removeName(index: number) {
    if (!canRemoveName()) return;
    setCountError('');
    setGuestNames((prev) => prev.filter((_, i) => i !== index));
  }

  function updateName(index: number, value: string) {
    setGuestNames((prev) => prev.map((n, i) => (i === index ? value : n)));
  }

  function handleSubmit(e: Event) {
    e.preventDefault();
    const trimmedName = name().trim();
    if (!trimmedName) return;
    const allNames = [trimmedName, ...guestNames().filter((n) => n.trim().length > 0)];

    // Group invite: co-primaries, no primary/plus distinction, no min/max. The
    // server derives the display name from the joined co-primary names.
    if (group()) {
      setCountError('');
      props.onSubmit({
        id: props.form.id,
        name: trimmedName,
        min_plus: 0,
        max_plus: 0,
        guest_names: allNames,
        link_lang: linkLang(),
        group: true,
      });
      return;
    }

    const min = Number(minPlus());
    const max = Number(maxPlus());
    if (Number.isNaN(min) || Number.isNaN(max) || min > max) return;

    const plusCount = allNames.length - 1;
    if (plusCount < min || plusCount > max) {
      setCountError(
        translate('admin_guest_count_error', props.lang)
          .replace('{min}', String(min))
          .replace('{max}', String(max)),
      );
      return;
    }
    setCountError('');

    props.onSubmit({
      id: props.form.id,
      name: trimmedName,
      min_plus: min,
      max_plus: max,
      guest_names: allNames,
      link_lang: linkLang(),
      group: false,
    });
  }

  return (
    <form class="admin-form card" onSubmit={handleSubmit}>
      <h2 class="heading heading--md">{heading()}</h2>
      <label class="admin-group-toggle">
        <input
          type="checkbox"
          name="group"
          checked={group()}
          onChange={(e) => setGroup(e.currentTarget.checked)}
        />
        <span>{translate('admin_group_invite', props.lang)}</span>
      </label>
      <label>
        <span>{translate(group() ? 'admin_coprimary_name_label' : 'admin_name_label', props.lang)}</span>
        <input
          type="text"
          name="name"
          value={name()}
          required
          onInput={(e) => setName(e.currentTarget.value)}
        />
      </label>
      <Show when={!group()}>
        <label>
          <span>{translate('admin_min_label', props.lang)}</span>
          <input
            type="number"
            name="min_plus"
            min="0"
            value={minPlus()}
            required
            onInput={(e) => setMinPlus(e.currentTarget.value)}
          />
        </label>
        <label>
          <span>{translate('admin_max_label', props.lang)}</span>
          <input
            type="number"
            name="max_plus"
            min="0"
            value={maxPlus()}
            required
            onInput={(e) => setMaxPlus(e.currentTarget.value)}
          />
        </label>
      </Show>
      <div class="guest-names">
        <span>{translate(group() ? 'admin_coprimary_names_label' : 'admin_guest_names_label', props.lang)}</span>
        <div class="guest-names-list">
          <Index each={guestNames()}>
            {(gname, index) => (
              <div class="guest-name-row">
                <input
                  type="text"
                  class="guest-name"
                  data-index={index}
                  value={gname()}
                  onInput={(e) => updateName(index, e.currentTarget.value)}
                />
                <button
                  type="button"
                  class="btn btn--ghost btn--sm"
                  disabled={!canRemoveName()}
                  onClick={() => removeName(index)}
                >
                  {translate('admin_remove_name', props.lang)}
                </button>
              </div>
            )}
          </Index>
        </div>
        <button
          type="button"
          class="btn btn--secondary btn--sm"
          disabled={!canAddName()}
          onClick={addName}
        >
          {translate('admin_add_name', props.lang)}
        </button>
      </div>
      {props.isCreate && (
        <label>
          <span>{translate('admin_link_lang_label', props.lang)}</span>
          <select name="link_lang" onChange={(e) => setLinkLang(e.currentTarget.value as Lang)}>
            <For each={LOCALE_CODES}>
              {(l) => <option value={l} selected={l === linkLang()}>{l}</option>}
            </For>
          </select>
        </label>
      )}
      {(countError() || props.formError) && (
        <p class="error">{countError() || props.formError}</p>
      )}
      <div class="form-actions">
        <button type="submit" class="btn btn--primary btn--md">{submitLabel()}</button>
        <button type="button" class="btn btn--ghost btn--md" onClick={() => props.onCancel()}>
          {translate('admin_cancel', props.lang)}
        </button>
      </div>
    </form>
  );
}
