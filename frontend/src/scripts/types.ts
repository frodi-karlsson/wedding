export interface Invite {
  id: number;
  name: string;
  min_plus: number;
  max_plus: number;
  submitted: boolean;
}

export interface Guest {
  id: number;
  name: string;
  dietary_preference: string;
  alcohol_free: boolean;
  is_primary: boolean;
}

export interface InviteWithGuestsResponse {
  invite: Invite;
  guests: Guest[];
}

export interface GuestInput {
  name: string;
  dietary_preference: string;
  alcohol_free: boolean;
  is_primary: boolean;
}

export interface RSVPRequest {
  guests: GuestInput[];
}

export interface CreateInviteRequest {
  name: string;
  min_plus: number;
  max_plus: number;
  guest_names: string[];
}

export interface ListInvitesResponse {
  invites: Invite[];
}

export interface AdminAuthResponse {
  status: string;
}
