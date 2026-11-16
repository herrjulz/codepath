import type { Scenario } from "../lib/schema";

export function ScenarioSelector({
  scenarios,
  value,
  onChange,
}: {
  scenarios: Scenario[];
  value: string;
  onChange: (id: string) => void;
}) {
  return (
    <label className="scenario">
      <span>Scenario</span>
      <select value={value} onChange={(event) => onChange(event.target.value)} aria-label="Scenario">
        {scenarios.map((scenario) => (
          <option key={scenario.id} value={scenario.id}>
            {scenario.name}
          </option>
        ))}
      </select>
    </label>
  );
}
