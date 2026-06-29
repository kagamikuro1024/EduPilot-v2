import javaClient from "./javaClient";
import { AuthRequest, AuthResponse, RegisterRequest } from "@/types/auth";

/**
 * Service to handle Authentication with the Java Backend.
 */
export const authService = {
  /**
   * Log in a user and return a JWT token.
   */
  login: async (credentials: AuthRequest): Promise<AuthResponse> => {
    // javaClient has baseURL set to the API host (e.g. http://localhost:8080)
    const response = await javaClient.post<AuthResponse>("/api/v1/auth/login", credentials);
    return response.data;
  },

  /**
   * Register a new user and return a JWT token.
   */
  register: async (userData: RegisterRequest): Promise<AuthResponse> => {
    const response = await javaClient.post<AuthResponse>("/api/v1/auth/register", userData);
    return response.data;
  },
};
