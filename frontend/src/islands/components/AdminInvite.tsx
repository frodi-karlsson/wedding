import { createSignal, createResource, For, Show, type JSX } from 'solid-js';
import QRCode from 'qrcode';
import { translate, LOCALE_CODES, type Lang } from '../../scripts/i18n';
import type { InviteResponse } from '../../scripts/types.gen';
import { buildShareLink } from '../../scripts/admin.service';

interface AdminInviteProps {
  lang: Lang;
  invite: InviteResponse;
  initialLang: Lang;
  onBack: () => void;
}

export function AdminInvite(props: AdminInviteProps): JSX.Element {
  const [linkLang, setLinkLang] = createSignal<Lang>(props.initialLang);
  const link = (): string => buildShareLink(globalThis.location.origin, props.invite.id, linkLang());

  // QR is generated locally (no network) so it is safe for a private site.
  // A PNG data URL serves both the on-screen <img> and the download. Failures
  // (e.g. no canvas in a test environment) resolve to '' so the link stays usable.
  const [qrPng] = createResource(link, (l) =>
    QRCode.toDataURL(l, { margin: 1, width: 512 }).catch(() => ''),
  );

  let copyBtn: HTMLButtonElement | undefined;
  async function onCopy(): Promise<void> {
    try {
      await navigator.clipboard.writeText(link());
      if (copyBtn) {
        const original = copyBtn.textContent ?? '';
        copyBtn.textContent = translate('admin_copied', props.lang);
        setTimeout(() => {
          if (copyBtn) copyBtn.textContent = original;
        }, 1500);
      }
    } catch {
      // ignore clipboard errors
    }
  }

  const downloadName = (): string =>
    `invite-${props.invite.name.replace(/\s+/g, '-').toLowerCase()}-${linkLang()}.png`;

  return (
    <div class="admin-invite">
      <h2 class="heading heading--md admin-heading">{props.invite.name}</h2>

      <label class="admin-invite__field">
        <span>{translate('admin_link_lang_label', props.lang)}</span>
        <select value={linkLang()} onChange={(e) => setLinkLang(e.currentTarget.value as Lang)}>
          <For each={LOCALE_CODES}>
            {(l) => <option value={l} selected={l === linkLang()}>{l}</option>}
          </For>
        </select>
      </label>

      <label class="admin-invite__field">
        <span>{translate('admin_invite_link', props.lang)}</span>
        <input
          type="text"
          readonly
          value={link()}
          onFocus={(e) => e.currentTarget.select()}
        />
      </label>

      <button ref={copyBtn} type="button" class="btn btn--secondary btn--sm" onClick={onCopy}>
        {translate('admin_copy_link', props.lang)}
      </button>

      <div class="admin-invite__qr">
        <span class="admin-invite__qr-label">{translate('admin_qr', props.lang)}</span>
        <Show when={qrPng()}>
          <div class="admin-invite__qr-img">
            <img src={qrPng()} alt={translate('admin_qr', props.lang)} />
          </div>
          <a class="btn btn--ghost btn--sm" href={qrPng()} download={downloadName()}>
            {translate('admin_download_qr', props.lang)}
          </a>
        </Show>
      </div>

      <div class="form-actions">
        <button type="button" class="btn btn--ghost btn--md" onClick={() => props.onBack()}>
          {translate('admin_back', props.lang)}
        </button>
      </div>
    </div>
  );
}
