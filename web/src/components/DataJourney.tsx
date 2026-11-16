import type { StepExample } from "../lib/schema";

export function DataJourney({ example }: { example: StepExample | null }) {
  if (!example) {
    return (
      <section className="journey">
        <h3>Data journey</h3>
        <p className="muted">This step is not on the selected scenario path, so there is no example data for it.</p>
      </section>
    );
  }

  return (
    <section className="journey">
      <div className="journey-head">
        <h3>Data journey</h3>
        <p className="banner">Illustrative / synthetic — not captured from a real execution</p>
      </div>
      <ValueBlock title="Input" format={example.input.format} content={example.input.content} />
      <div className="transform">
        <h4>Transformation</h4>
        <p>{example.transformation}</p>
      </div>
      <ValueBlock title="Output" format={example.output.format} content={example.output.content} />
      {example.fieldChanges && example.fieldChanges.length > 0 ? (
        <div className="changes">
          <h4>Field changes</h4>
          <ul>
            {example.fieldChanges.map((change, index) => (
              <li key={`${change.field}-${index}`}>
                <code>{change.field}</code>
                {(change.before || change.after) && (
                  <span className="change-values">
                    {change.before ? <span>{change.before}</span> : null}
                    {change.before && change.after ? <span aria-hidden="true"> → </span> : null}
                    {change.after ? <span>{change.after}</span> : null}
                  </span>
                )}
                <span className="muted">{change.why}</span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}

function ValueBlock({ title, format, content }: { title: string; format: string; content: string }) {
  return (
    <div className="value-block">
      <div className="value-head">
        <h4>{title}</h4>
        <span className="format">{format}</span>
      </div>
      <pre>{content}</pre>
    </div>
  );
}
