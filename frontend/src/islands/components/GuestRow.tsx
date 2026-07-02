import { type JSX } from 'solid-js';
import type { Lang } from '../../scripts/i18n';
import { translate } from '../../scripts/i18n';
import type { GuestInput } from '../../scripts/types.gen';

interface GuestRowProps {
  guest: GuestInput;
  index: number;
  lang: Lang;
  onRemove: (index: number) => void;
  onUpdate: (index: number, patch: Partial<GuestInput>) => void;
}

export function GuestRow(props: GuestRowProps): JSX.Element {
  const legend = () =>
    props.guest.is_primary
      ? translate('name_you_label', props.lang)
      : `${translate('guest_label', props.lang)} ${props.index}`;

  const nameLabel = () =>
    translate(props.guest.is_primary ? 'name_you_label' : 'name_label', props.lang);

  function onNameInput(e: InputEvent & { currentTarget: HTMLInputElement }) {
    props.onUpdate(props.index, { name: e.currentTarget.value });
  }

  function onDietaryInput(e: InputEvent & { currentTarget: HTMLInputElement }) {
    props.onUpdate(props.index, { dietary_preference: e.currentTarget.value });
  }

  function onAlcoholFreeChange(e: Event & { currentTarget: HTMLInputElement }) {
    props.onUpdate(props.index, { alcohol_free: e.currentTarget.checked });
  }

  function onRemoveClick() {
    props.onRemove(props.index);
  }

  return (
    <fieldset class="guest-row" data-index={props.index}>
      <legend>{legend()}</legend>
      <label>
        <span>{nameLabel()}</span>
        <input type="text" value={props.guest.name} required onInput={onNameInput} />
      </label>
      <label>
        <span>{translate('dietary_label', props.lang)}</span>
        <input
          type="text"
          value={props.guest.dietary_preference}
          onInput={onDietaryInput}
        />
      </label>
      <label class="checkbox">
        <input
          type="checkbox"
          checked={props.guest.alcohol_free}
          onChange={onAlcoholFreeChange}
        />
        <span>{translate('alcohol_free_label', props.lang)}</span>
      </label>
      {!props.guest.is_primary && (
        <button
          type="button"
          class="btn btn--ghost btn--sm remove"
          onClick={onRemoveClick}
        >
          {translate('remove_guest', props.lang)}
        </button>
      )}
    </fieldset>
  );
}
