"use client";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

interface MarkdownRendererProps {
  content: string;
}

export default function MarkdownRenderer({ content }: MarkdownRendererProps) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      components={{
        p: ({ children }) => <p className="mb-3 leading-relaxed last:mb-0">{children}</p>,
        strong: ({ children }) => <strong className="font-bold text-slate-950 dark:text-white">{children}</strong>,
        em: ({ children }) => <em className="italic text-slate-800 dark:text-slate-200">{children}</em>,
        ul: ({ children }) => <ul className="mb-4 ml-6 list-disc space-y-1 text-slate-700 dark:text-slate-300">{children}</ul>,
        ol: ({ children }) => <ol className="mb-4 ml-6 list-decimal space-y-1 text-slate-700 dark:text-slate-300">{children}</ol>,
        li: ({ children }) => <li className="pl-1">{children}</li>,
        code: ({ children }) => (
          <code className="rounded bg-slate-100 dark:bg-slate-800 px-1.5 py-0.5 font-mono text-xs text-slate-800 dark:text-slate-200">
            {children}
          </code>
        ),
        pre: ({ children }) => (
          <pre className="mb-4 overflow-x-auto rounded-xl bg-slate-100 dark:bg-zinc-900 p-4 font-mono text-xs shadow-inner">
            {children}
          </pre>
        ),
        a: ({ children, href }) => (
          <a
            href={href}
            className="font-medium text-indigo-600 dark:text-indigo-400 underline decoration-indigo-200 dark:decoration-indigo-900 underline-offset-2 hover:decoration-indigo-600 dark:hover:decoration-indigo-400 transition-colors"
            target="_blank"
            rel="noopener noreferrer"
          >
            {children}
          </a>
        ),
        h1: ({ children }) => <h1 className="mb-4 mt-6 text-xl font-bold text-slate-900 dark:text-slate-100">{children}</h1>,
        h2: ({ children }) => <h2 className="mb-3 mt-5 text-lg font-bold text-slate-900 dark:text-slate-100">{children}</h2>,
        h3: ({ children }) => <h3 className="mb-2 mt-4 text-base font-bold text-slate-900 dark:text-slate-100">{children}</h3>,
        blockquote: ({ children }) => <blockquote className="border-l-4 border-slate-300 dark:border-slate-700 pl-4 italic my-4 text-slate-600 dark:text-slate-400">{children}</blockquote>,
      }}
    >
      {content}
    </ReactMarkdown>
  );
}
