import { create } from "zustand";

interface PrivacyState {
  pendingPrivateText: string;
  setPendingPrivateText: (text: string) => void;
  clearPendingPrivateText: () => void;
}

export const usePrivacyStore = create<PrivacyState>((set) => ({
  pendingPrivateText: "",
  setPendingPrivateText: (text) => set({ pendingPrivateText: text }),
  clearPendingPrivateText: () => set({ pendingPrivateText: "" }),
}));
