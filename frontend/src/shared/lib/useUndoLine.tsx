"use client";

import { useCallback, useState, type ReactNode } from "react";
import { UndoLine } from "@/shared/ui";

/**
 * Dòng "Đã đánh vắng · Hoàn tác" tự biến sau 5 s (thay toast, UX.md). Dùng:
 *   const undo = useUndoLine();  …  undo.push("Đã đánh vắng Trần Văn A", () => revert());  …  {undo.node}
 * Mỗi lần `push` thay dòng cũ.
 */
export function useUndoLine() {
  const [line, setLine] = useState<{ id: number; message: ReactNode; onUndo?: () => void } | null>(null);
  const push = useCallback((message: ReactNode, onUndo?: () => void) => setLine({ id: Date.now() + Math.random(), message, onUndo }), []);
  const clear = useCallback(() => setLine(null), []);
  const node = line ? <UndoLine key={line.id} message={line.message} onUndo={line.onUndo} onDone={clear} /> : null;
  return { node, push, clear };
}
