// Types mirroring the Go API's JSON shapes (see backend/internal/user/user.go,
// backend/internal/invite/invite.go). Timestamps arrive as RFC3339 strings.

export type Role = 'admin' | 'member';

export interface User {
  id: string;
  email: string;
  name: string;
  role: Role;
  createdAt: string;
  updatedAt: string;
}

export interface Invite {
  id: string;
  token: string;
  email?: string;
  role: Role;
  createdBy: string;
  expiresAt: string;
  redeemedAt?: string;
  redeemedBy?: string;
  createdAt: string;
}

export interface Version {
  version: string;
  env: string;
}

export interface LoginInput {
  email: string;
  password: string;
}

export interface RedeemInput {
  token: string;
  email: string;
  name: string;
  password: string;
}

export interface CreateInviteInput {
  email?: string;
  role?: Role;
  ttlHours?: number;
}
