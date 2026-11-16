import { useState } from "react";
import { ErdDiagram } from "./ErdDiagram";
import { SequenceDiagram } from "./SequenceDiagram";
import type { FlowStep, Scenario, WalkthroughReport } from "../lib/schema";

type DiagramTab = "sequence" | "erd";

export function SummaryPanel({
  report,
  onExplore,
  onSelectStep,
}: {
  report: WalkthroughReport;
  onExplore: () => void;
  onSelectStep: (stepId: string) => void;
}) {
  const steps = orderedSteps(report.steps, report.scenarios[0]);
  const erd = report.overview.erd;
  const [diagramTab, setDiagramTab] = useState<DiagramTab>("sequence");

  return (
    <section className="summary-view" aria-label="Summary">
      <div className="summary-hero">
        <p className="eyebrow">Summary</p>
        <h2>{report.title}</h2>
        <p className="summary-lede">{report.summary}</p>
        <div className="summary-actions">
          <button type="button" className="primary" onClick={onExplore}>
            Explore walkthrough
          </button>
          <p className="muted">Open the step-by-step flow, source, and data journey.</p>
        </div>
      </div>

      <div className="summary-grid">
        <section className="summary-steps" aria-label="Steps">
          <h3>Steps</h3>
          <ol>
            {steps.map((step, index) => (
              <li key={step.id}>
                <button type="button" className="summary-step" onClick={() => onSelectStep(step.id)}>
                  <span className="summary-step-index">{index + 1}</span>
                  <span className="summary-step-copy">
                    <strong>{step.title}</strong>
                    <span>{step.summary}</span>
                  </span>
                </button>
              </li>
            ))}
          </ol>
        </section>

        <section className="summary-diagrams" aria-label="Overview diagrams">
          <div className="summary-diagram-head">
            {erd ? (
              <div className="diagram-tabs" role="tablist" aria-label="Diagram">
                <button
                  type="button"
                  role="tab"
                  aria-selected={diagramTab === "sequence"}
                  className={diagramTab === "sequence" ? "active" : ""}
                  onClick={() => setDiagramTab("sequence")}
                >
                  Sequence
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={diagramTab === "erd"}
                  className={diagramTab === "erd" ? "active" : ""}
                  onClick={() => setDiagramTab("erd")}
                >
                  Data model
                </button>
              </div>
            ) : (
              <h3>Sequence</h3>
            )}
            <p className="muted">
              {diagramTab === "erd" && erd
                ? "Entities and relationships in this flow."
                : "Scroll to zoom, drag to pan. Click a linked message to open that step."}
            </p>
          </div>
          <div className="summary-diagram-body">
            {diagramTab === "erd" && erd ? (
              <ErdDiagram erd={erd} />
            ) : (
              <SequenceDiagram sequence={report.overview.sequence} onSelectStep={onSelectStep} />
            )}
          </div>
        </section>
      </div>
    </section>
  );
}

function orderedSteps(steps: FlowStep[], scenario: Scenario | undefined): FlowStep[] {
  const byId = new Map(steps.map((step) => [step.id, step]));
  const ordered: FlowStep[] = [];
  const seen = new Set<string>();
  for (const id of scenario?.path ?? []) {
    const step = byId.get(id);
    if (step && !seen.has(id)) {
      ordered.push(step);
      seen.add(id);
    }
  }
  for (const step of steps) {
    if (!seen.has(step.id)) {
      ordered.push(step);
    }
  }
  return ordered;
}
