import Editor, { type OnMount } from "@monaco-editor/react";
import { useEffect, useState } from "react";
import { getSource } from "../lib/api";
import type { FlowStep, SourceView } from "../lib/schema";

export function CodePanel({
  reportId,
  step,
  refIndex,
  theme,
  onRefIndex,
}: {
  reportId: string;
  step: FlowStep;
  refIndex: number;
  theme: "dark" | "light";
  onRefIndex: (index: number) => void;
}) {
  const safeIndex = Math.min(Math.max(refIndex, 0), Math.max(step.codeRefs.length - 1, 0));
  const ref = step.codeRefs[safeIndex];
  const [source, setSource] = useState<SourceView | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!ref) {
      return;
    }
    let cancelled = false;
    setLoading(true);
    setError(null);
    getSource(reportId, step.id, safeIndex)
      .then((view) => {
        if (!cancelled) {
          setSource(view);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Could not load source");
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [reportId, step.id, safeIndex, ref?.path, ref?.startLine, ref?.endLine]);

  const verified = source && !source.missing && !source.stale;

  return (
    <section className="col code" aria-label="Code">
      <header className="col-head">
        <div>
          <p className="eyebrow">Code</p>
          <h2>{ref ? ref.path : "No source reference"}</h2>
          {ref ? (
            <p className="muted">
              Lines {ref.startLine}–{ref.endLine}
              {ref.label ? ` · ${ref.label}` : ""}
            </p>
          ) : null}
        </div>
        <span className={`badge ${verified ? "ok" : "warn"}`}>
          {source?.missing ? "Source file unavailable" : source?.stale ? "Source changed since this report was created" : "Source reference verified"}
        </span>
      </header>
      {step.codeRefs.length > 1 ? (
        <div className="tabs" role="tablist" aria-label="Code references">
          {step.codeRefs.map((item, index) => (
            <button
              key={`${item.path}-${item.startLine}-${index}`}
              type="button"
              role="tab"
              aria-selected={index === safeIndex}
              className={index === safeIndex ? "active" : ""}
              onClick={() => onRefIndex(index)}
            >
              {item.label || `${item.path}:${item.startLine}`}
            </button>
          ))}
        </div>
      ) : null}
      {source?.message ? <p className={`banner ${source.missing ? "danger" : "warn"}`}>{source.message}</p> : null}
      {error ? <p className="banner danger">{error}</p> : null}
      <div className="code-editor">
        {loading ? <p className="muted pad">Loading source…</p> : null}
        {!loading && source && !source.missing && source.context ? (
          <ReadOnlyCode source={source} theme={theme} />
        ) : null}
        {!loading && source?.missing ? <p className="muted pad">The recorded location is kept, but the file could not be read.</p> : null}
      </div>
    </section>
  );
}

function ReadOnlyCode({ source, theme }: { source: SourceView; theme: "dark" | "light" }) {
  const onMount: OnMount = (editor, monaco) => {
    const start = Math.max(1, source.startLine - source.contextStart + 1);
    const end = Math.max(start, source.endLine - source.contextStart + 1);
    editor.createDecorationsCollection([
      {
        range: new monaco.Range(start, 1, end, 1),
        options: { isWholeLine: true, className: "codepath-span" },
      },
    ]);
    editor.revealLineInCenter(start);
  };

  return (
    <Editor
      key={`${source.path}:${source.startLine}:${source.endLine}:${source.currentSha256 ?? "missing"}`}
      language="go"
      theme={theme === "dark" ? "vs-dark" : "light"}
      value={source.context}
      onMount={onMount}
      options={{
        readOnly: true,
        domReadOnly: true,
        lineNumbers: (line) => String(source.contextStart + line - 1),
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        wordWrap: "on",
        fontSize: 13,
        fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
        padding: { top: 8, bottom: 8 },
        renderLineHighlight: "none",
        overviewRulerLanes: 0,
        folding: false,
        contextmenu: false,
      }}
    />
  );
}
