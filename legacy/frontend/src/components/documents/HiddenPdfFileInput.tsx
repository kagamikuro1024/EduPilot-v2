import type { ChangeEvent, RefObject } from "react";

export interface HiddenPdfFileInputProps {
  inputRef: RefObject<HTMLInputElement | null>;
  onChange: (event: ChangeEvent<HTMLInputElement>) => void;
}

export function HiddenPdfFileInput({ inputRef, onChange }: HiddenPdfFileInputProps) {
  return (
    <input ref={inputRef} type="file" accept="application/pdf" className="hidden" onChange={onChange} />
  );
}
