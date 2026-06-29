export type UserRole = "STUDENT" | "TA" | "ADMIN";

export interface AuthRequest {
  email: string;
  password?: string;
}

export interface AuthResponse {
  token: string;
  role: UserRole;
  fullName?: string;
}

export interface RegisterRequest {
  fullName: string;
  email: string;
  password?: string;
  role: UserRole;
}
