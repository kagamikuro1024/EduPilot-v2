import { useState, useEffect } from "react";
import axios from "axios";

export function usePiiDebounce(text: string, delay: number = 500) {
  const [isPii, setIsPii] = useState(false);
  const [isChecking, setIsChecking] = useState(false);

  useEffect(() => {
    if (!text.trim()) {
      setIsPii(false);
      setIsChecking(false);
      return;
    }

    const controller = new AbortController();

    const handler = setTimeout(async () => {
      setIsChecking(true);
      try {
        const response = await axios.post("/api/v1/classify-intent", { text }, {
          signal: controller.signal
        });
        setIsPii(response.data.hasPii);
      } catch (error) {
        if (axios.isCancel(error)) {
          // Silent abort
          return;
        }
        console.error("PII Check failed:", error);
        setIsPii(false);
      } finally {
        setIsChecking(false);
      }
    }, delay);

    return () => {
      clearTimeout(handler);
      controller.abort();
    };
  }, [text, delay]);

  return { isPii, isChecking };
}
