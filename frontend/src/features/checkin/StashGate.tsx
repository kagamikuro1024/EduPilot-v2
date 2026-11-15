"use client";

import { useLayoutEffect } from "react";
import { stashFromLocation } from "./stash";

/** Chạy TRƯỚC `AuthGate` (đứng trước nó trong layout): cất token rồi xoá fragment. */
export function StashGate() {
  useLayoutEffect(() => stashFromLocation(), []);
  return null;
}
