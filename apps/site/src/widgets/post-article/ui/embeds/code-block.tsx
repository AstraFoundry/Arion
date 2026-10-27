"use client";

import { Check, Copy } from "lucide-react";
import { type ComponentPropsWithoutRef, useEffect, useRef, useState } from "react";
import { useCopy } from "@/shared/i18n";

const COPIED_MS = 1600;

/**
 * A highlighted code block (Shiki ran at build time) with a copy button. The
 * Shiki theme colors are CSS variables on each token; prose.css picks the
 * light or dark one, so the inline background Shiki sets is dropped.
 */
export function CodeBlock({ children, className }: ComponentPropsWithoutRef<"pre">) {
  const { blog } = useCopy();
  const pre = useRef<HTMLPreElement>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const timer = window.setTimeout(() => setCopied(false), COPIED_MS);
    return () => window.clearTimeout(timer);
  }, [copied]);

  const copy = () => {
    const text = pre.current?.textContent ?? "";
    void navigator.clipboard.writeText(text.replace(/\n$/, "")).then(() => setCopied(true));
  };

  return (
    <div className="post-code">
      <pre ref={pre} className={className} tabIndex={0}>
        {children}
      </pre>
      <button
        type="button"
        onClick={copy}
        aria-label={copied ? blog.copied : blog.copyCode}
        className="post-code-copy"
      >
        {copied ? <Check aria-hidden="true" /> : <Copy aria-hidden="true" />}
      </button>
    </div>
  );
}
