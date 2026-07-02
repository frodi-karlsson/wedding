import { Show, type JSX } from 'solid-js';
import { hasInviteId } from '../scripts/nav';

interface InviteeViewProps {
  children: JSX.Element;
}

export function InviteeView(props: InviteeViewProps): JSX.Element {
  return <Show when={hasInviteId(globalThis.location)}>{props.children}</Show>;
}

export default InviteeView;
