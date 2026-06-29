import { describe, expect, it } from "vitest";
import { mapSseMetadataToCitations, mergeRagDocs } from "./ragCitations";

describe("mapSseMetadataToCitations", () => {
  it("maps source file and page into relevant docs", () => {
    const docs = mapSseMetadataToCitations({
      citations: [
        {
          source_file: "week4-pointers.pdf",
          page_number: 12,
          document_id: "doc-1",
          chunk_id: "chunk-a",
          chunk_index: 3,
          snippet: "A pointer stores a memory address.",
        },
      ],
    });

    expect(docs).toHaveLength(1);
    expect(docs[0]).toMatchObject({
      id: "doc-1-chunk-a-3",
      title: "week4-pointers.pdf",
      pageNumber: 12,
      snippet: "A pointer stores a memory address.",
    });
  });

  it("returns empty for missing or invalid metadata", () => {
    expect(mapSseMetadataToCitations(undefined)).toEqual([]);
    expect(mapSseMetadataToCitations({ citations: [{ page_number: 9 }] })).toEqual([]);
  });

  it("parses metadata when SSE payload provides JSON string", () => {
    const docs = mapSseMetadataToCitations(
      JSON.stringify({
        citations: [{ source_file: "Saga pattern.pdf", page_number: 4 }],
      }),
    );
    expect(docs).toHaveLength(1);
    expect(docs[0].title).toBe("Saga pattern.pdf");
    expect(docs[0].pageNumber).toBe(4);
  });

  it("deduplicates repeated citations in the same SSE payload", () => {
    const docs = mapSseMetadataToCitations({
      citations: [
        {
          source_file: "week4-pointers.pdf",
          page_number: 12,
          document_id: "doc-1",
          chunk_id: "chunk-a",
          chunk_index: 3,
          snippet: "A pointer stores a memory address.",
        },
        {
          source_file: "week4-pointers.pdf",
          page_number: 12,
          document_id: "doc-1",
          chunk_id: "chunk-a",
          chunk_index: 3,
          snippet: "A pointer stores a memory address.",
        },
      ],
    });

    expect(docs).toHaveLength(1);
    expect(docs[0].id).toBe("doc-1-chunk-a-3");
  });
});

describe("mergeRagDocs", () => {
  it("deduplicates by title and page", () => {
    const merged = mergeRagDocs(
      [{ id: "1", title: "intro.pdf", pageNumber: 1, snippet: "" }],
      [
        { id: "2", title: "intro.pdf", pageNumber: 1, snippet: "" },
        { id: "3", title: "lab.pdf", pageNumber: 3, snippet: "" },
      ],
    );

    expect(merged).toHaveLength(2);
    expect(merged.map((x) => `${x.title}:${x.pageNumber}`)).toEqual(["intro.pdf:1", "lab.pdf:3"]);
  });
});
