// Shared types mirroring the Go backend's JSON shapes.

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
  createdAt: string;
}

export interface VersionInfo {
  version: string;
  env: string;
}
