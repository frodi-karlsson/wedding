import { type JSX } from 'solid-js';
import type { Lang } from '../../scripts/i18n';
import { translate } from '../../scripts/i18n';

interface AdminLoginProps {
  lang: Lang;
  error?: string;
  onLogin: (password: string) => void;
}

export function AdminLogin(props: AdminLoginProps): JSX.Element {
  let passwordInput: HTMLInputElement | undefined;

  function onSubmit(e: Event) {
    e.preventDefault();
    props.onLogin(passwordInput?.value ?? '');
  }

  return (
    <form class="admin-login card" onSubmit={onSubmit}>
      <label>
        <span>{translate('admin_password_label', props.lang)}</span>
        <input type="password" name="password" required ref={(el) => { passwordInput = el; }} />
      </label>
      <button type="submit" class="btn btn--primary btn--md">
        {translate('admin_login', props.lang)}
      </button>
      {props.error && <p class="error">{props.error}</p>}
    </form>
  );
}
