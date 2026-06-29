export interface RAGDocument {
  id: string;
  title: string;
  pageNumber: number;
  snippet: string;
}

interface SseCitationMeta {
  source_file?: string;
  page_number?: number | string;
  document_id?: string;
  source_uri?: string;
  chunk_id?: string;
  chunk_index?: number | string;
  snippet?: string;
}

interface SseFinishMetadata {
  citations?: SseCitationMeta[];
}

export function mapSseMetadataToCitations(metadata: unknown): RAGDocument[] {
  let normalized: unknown = metadata;
  if (typeof normalized === "string") {
    try {
      normalized = JSON.parse(normalized);
    } catch {
      return [];
    }
  }
  if (!normalized || typeof normalized !== "object") return [];
  const maybe = normalized as SseFinishMetadata;
  if (!Array.isArray(maybe.citations)) return [];

  const seen = new Set<string>();
  const docs: RAGDocument[] = [];

  maybe.citations.forEach((c, index) => {
      const title = typeof c?.source_file === "string" ? c.source_file.trim() : "";
      if (!title) return;
      const pageRaw = c.page_number;
      const pageNumber = typeof pageRaw === "number" ? pageRaw : Number(pageRaw);
      const safePage = Number.isFinite(pageNumber) ? pageNumber : 0;
      const chunkIndex = typeof c.chunk_index === "number" ? c.chunk_index : Number(c.chunk_index);
      const normalizedChunkIndex = Number.isFinite(chunkIndex) ? chunkIndex : -1;
      const normalizedDocumentId = typeof c.document_id === "string" ? c.document_id.trim() : "";
      const normalizedChunkId = typeof c.chunk_id === "string" ? c.chunk_id.trim() : "";
      const stableId =
        [normalizedDocumentId, normalizedChunkId, Number.isFinite(chunkIndex) ? chunkIndex : undefined]
          .filter((part) => typeof part === "string" ? part.trim() : part !== undefined)
          .join("-") || `${title.toLowerCase()}-${safePage}-${index}`;
      // Deduplicate by document title + page (not by chunk, since multiple chunks
      // from the same page should produce only one citation card)
      const dedupeKey = `${title.toLowerCase()}::${String(safePage)}`;
      if (seen.has(dedupeKey)) return;
      seen.add(dedupeKey);

      docs.push({
        id: stableId,
        title,
        pageNumber: safePage,
        snippet: typeof c.snippet === "string" ? c.snippet : "",
      } satisfies RAGDocument);
  });

  return docs;
}

export function mergeRagDocs(prev: RAGDocument[], next: RAGDocument[]): RAGDocument[] {
  const merged = [...next, ...prev];
  const deduped = new Map<string, RAGDocument>();
  for (const doc of merged) {
    const key = `${doc.title.toLowerCase()}::${doc.pageNumber}`;
    if (!deduped.has(key)) deduped.set(key, doc);
  }
  return Array.from(deduped.values()).slice(0, 6);
}
