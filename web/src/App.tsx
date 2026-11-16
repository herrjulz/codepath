import { useCallback, useEffect, useState, type PointerEvent as ReactPointerEvent, type ReactNode } from "react";
import { CodePanel } from "./components/CodePanel";
import { ExplanationPanel } from "./components/ExplanationPanel";
import { FlowPanel } from "./components/FlowPanel";
import { SummaryPanel } from "./components/SummaryPanel";
import { listReports, getReport } from "./lib/api";
import { adjacentStep } from "./lib/navigation";
import type { ReportSummary, WalkthroughReport } from "./lib/schema";

type Tab = "flow" | "code" | "explanation";
type Mode = "summary" | "detail";

export function App() {
  const route = useRoute();
  const theme = useTheme();
  if (route.kind === "home") {
    return <Home />;
  }
  return <Viewer id={route.id} theme={theme} />;
}

function Home() {
  const [reports, setReports] = useState<ReportSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    listReports()
      .then((items) => {
        if (!cancelled) {
          setReports(items);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Could not list reports");
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <main className="home">
      <header className="home-head">
        <p className="eyebrow">Local walkthroughs</p>
        <h1>CodePath</h1>
        <p>Follow a path through your repository. Explanations are AI-generated. Source references are read from the files on this machine.</p>
      </header>
      {error ? <p className="banner danger">{error}</p> : null}
      {reports && reports.length === 0 ? (
        <p className="empty">
          No reports yet. Seed the demo with <code>codepath serve --repo "$PWD" --seed examples/authentication-flow.json</code>, or ask OpenCode to call <code>create_walkthrough</code>.
        </p>
      ) : null}
      <ul className="report-list">
        {reports?.map((report) => (
          <li key={report.id}>
            <a href={`/reports/${report.id}`}>
              <strong>{report.title}</strong>
              <span>{report.summary}</span>
              <small>
                {report.repository}
                {report.revision ? ` · ${report.revision.slice(0, 8)}` : ""} · {report.createdAt}
              </small>
            </a>
          </li>
        ))}
      </ul>
    </main>
  );
}

function Viewer({ id, theme }: { id: string; theme: "dark" | "light" }) {
  const [report, setReport] = useState<WalkthroughReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [scenarioId, setScenarioId] = useState("");
  const [stepId, setStepId] = useState("");
  const [refIndex, setRefIndex] = useState(0);
  const [tab, setTab] = useState<Tab>("flow");
  const [mode, setMode] = useState<Mode>("summary");
  const [widths, setWidths] = useState<[number, number, number]>([27, 38, 35]);
  const narrow = useNarrow();

  useEffect(() => {
    let cancelled = false;
    getReport(id)
      .then((next) => {
        if (cancelled) {
          return;
        }
        setReport(next);
        const scenario = next.scenarios[0];
        setScenarioId(scenario?.id ?? "");
        setStepId(scenario?.path[0] ?? next.steps[0]?.id ?? "");
        setRefIndex(0);
        setMode("summary");
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Could not load the report");
        }
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  const scenario = report?.scenarios.find((item) => item.id === scenarioId) ?? report?.scenarios[0];
  const step = report?.steps.find((item) => item.id === stepId) ?? report?.steps[0];

  const selectStep = useCallback((next: string) => {
    setStepId(next);
    setRefIndex(0);
  }, []);

  const openDetail = useCallback(
    (nextStepId?: string) => {
      if (nextStepId) {
        setStepId(nextStepId);
        setRefIndex(0);
      }
      setMode("detail");
    },
    [],
  );

  const selectScenario = useCallback(
    (nextId: string) => {
      const next = report?.scenarios.find((item) => item.id === nextId);
      setScenarioId(nextId);
      setStepId(next?.path[0] ?? "");
      setRefIndex(0);
    },
    [report],
  );

  useEffect(() => {
    if (!scenario || mode !== "detail") {
      return;
    }
    const path = scenario.path;
    function onKey(event: KeyboardEvent) {
      if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") {
        return;
      }
      const target = event.target;
      if (target instanceof HTMLElement) {
        if (target.closest("input, textarea, select, .monaco-editor") || target.isContentEditable) {
          return;
        }
      }
      const delta = event.key === "ArrowRight" ? 1 : -1;
      const next = adjacentStep(path, stepId, delta);
      if (!next) {
        return;
      }
      event.preventDefault();
      selectStep(next);
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [scenario, stepId, selectStep, mode]);

  function onDrag(index: 0 | 1, event: ReactPointerEvent<HTMLDivElement>) {
    const box = event.currentTarget.parentElement?.getBoundingClientRect();
    if (!box) {
      return;
    }
    const startX = event.clientX;
    const start: [number, number, number] = [widths[0], widths[1], widths[2]];
    const move = (ev: PointerEvent) => {
      const dx = ((ev.clientX - startX) / box.width) * 100;
      const next: [number, number, number] = [start[0], start[1], start[2]];
      const min = 16;
      let left = start[index] + dx;
      let right = start[index + 1] - dx;
      if (left < min) {
        right -= min - left;
        left = min;
      }
      if (right < min) {
        left -= min - right;
        right = min;
      }
      next[index] = left;
      next[index + 1] = right;
      setWidths(next);
    };
    const up = () => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
  }

  if (error) {
    return (
      <main className="home">
        <p className="banner danger">{error}</p>
        <p><a href="/">Back to reports</a></p>
      </main>
    );
  }
  if (!report || !scenario || !step) {
    return (
      <main className="home">
        <p>Loading walkthrough…</p>
      </main>
    );
  }

  const example = scenario.examples.find((item) => item.stepId === step.id) ?? null;
  const onPath = scenario.path.includes(step.id);
  const flow = (
    <FlowPanel
      steps={report.steps}
      edges={report.edges}
      scenarios={report.scenarios}
      scenario={scenario}
      stepId={step.id}
      onScenario={selectScenario}
      onSelect={selectStep}
    />
  );
  const code = (
    <CodePanel
      reportId={report.id}
      step={step}
      refIndex={refIndex}
      theme={theme}
      onRefIndex={setRefIndex}
    />
  );
  const explanation = <ExplanationPanel step={step} example={onPath ? example : null} onPath={onPath} />;

  return (
    <main className="app">
      <header className="topbar">
        <div>
          <a className="brand" href="/">CodePath</a>
          <h1>{report.title}</h1>
        </div>
        <div className="top-meta">
          {mode === "detail" ? (
            <button type="button" onClick={() => setMode("summary")}>
              Back to summary
            </button>
          ) : (
            <button type="button" className="primary" onClick={() => openDetail()}>
              Explore walkthrough
            </button>
          )}
          <span>{report.repository.displayName}</span>
          <span>{report.repository.revision ? report.repository.revision.slice(0, 8) : "no revision"}</span>
          {mode === "detail" ? <span className="kbd">← →</span> : null}
        </div>
      </header>
      {mode === "summary" ? (
        <SummaryPanel report={report} onExplore={() => openDetail()} onSelectStep={(next) => openDetail(next)} />
      ) : (
        <>
          {narrow ? (
            <div className="pane-tabs" role="tablist">
              {(["flow", "code", "explanation"] as Tab[]).map((item) => (
                <button key={item} type="button" role="tab" aria-selected={tab === item} className={tab === item ? "active" : ""} onClick={() => setTab(item)}>
                  {item === "flow" ? "Flow" : item === "code" ? "Code" : "Explanation"}
                </button>
              ))}
            </div>
          ) : null}
          <Columns narrow={narrow} tab={tab} widths={widths} onDrag={onDrag} flow={flow} code={code} explanation={explanation} />
        </>
      )}
    </main>
  );
}

function Columns({
  narrow,
  tab,
  widths,
  onDrag,
  flow,
  code,
  explanation,
}: {
  narrow: boolean;
  tab: Tab;
  widths: [number, number, number];
  onDrag: (index: 0 | 1, event: ReactPointerEvent<HTMLDivElement>) => void;
  flow: ReactNode;
  code: ReactNode;
  explanation: ReactNode;
}) {
  if (narrow) {
    return <div className="columns narrow">{tab === "flow" ? flow : tab === "code" ? code : explanation}</div>;
  }
  return (
    <div className="columns" style={{ gridTemplateColumns: `${widths[0]}fr 6px ${widths[1]}fr 6px ${widths[2]}fr` }}>
      {flow}
      <div className="handle" role="separator" aria-orientation="vertical" onPointerDown={(event) => onDrag(0, event)} />
      {code}
      <div className="handle" role="separator" aria-orientation="vertical" onPointerDown={(event) => onDrag(1, event)} />
      {explanation}
    </div>
  );
}

function useRoute(): { kind: "home" } | { kind: "report"; id: string } {
  const [path, setPath] = useState(window.location.pathname);
  useEffect(() => {
    const onPop = () => setPath(window.location.pathname);
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);
  const match = path.match(/^\/reports\/([^/]+)\/?$/);
  if (match) {
    return { kind: "report", id: decodeURIComponent(match[1]) };
  }
  return { kind: "home" };
}

function useNarrow() {
  const query = "(max-width: 900px)";
  const [narrow, setNarrow] = useState(() => window.matchMedia(query).matches);
  useEffect(() => {
    const media = window.matchMedia(query);
    const apply = () => setNarrow(media.matches);
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, []);
  return narrow;
}

function useTheme(): "dark" | "light" {
  const query = "(prefers-color-scheme: light)";
  const [theme, setTheme] = useState<"dark" | "light">(() => (window.matchMedia(query).matches ? "light" : "dark"));
  useEffect(() => {
    const media = window.matchMedia(query);
    const apply = () => setTheme(media.matches ? "light" : "dark");
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, []);
  return theme;
}
