import type { FlowStep, StepExample } from "../lib/schema";
import { DataJourney } from "./DataJourney";

export function ExplanationPanel({
  step,
  example,
  onPath,
}: {
  step: FlowStep;
  example: StepExample | null;
  onPath: boolean;
}) {
  return (
    <section className="col explain" aria-label="Explanation">
      <header className="col-head">
        <div>
          <p className="eyebrow">Explanation</p>
          <h2>{step.title}</h2>
        </div>
        <span className="badge ai">AI-generated interpretation</span>
      </header>
      <div className="explain-scroll">
        {!onPath ? <p className="banner warn">This step is outside the selected scenario. Previous and Next stay on that path.</p> : null}
        <p className="lede">{step.summary}</p>
        <p className="prose">{step.explanation}</p>
        <DataJourney example={example} />
        {step.notes && step.notes.length > 0 ? (
          <section className="notes">
            <h3>Notes</h3>
            <ul>
              {step.notes.map((note, index) => (
                <li key={index}>{note}</li>
              ))}
            </ul>
          </section>
        ) : null}
      </div>
    </section>
  );
}
