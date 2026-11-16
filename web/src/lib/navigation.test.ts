import { describe, expect, it } from "vitest";
import { adjacentStep, stepPosition } from "./navigation";

const path = ["login", "lookup", "check", "done"];

describe("scenario navigation", () => {
  it("reports the position on the selected path", () => {
    expect(stepPosition(path, "check")).toBe(2);
    expect(stepPosition(path, "other")).toBe(-1);
  });

  it("moves only along the scenario path", () => {
    expect(adjacentStep(path, "lookup", 1)).toBe("check");
    expect(adjacentStep(path, "lookup", -1)).toBe("login");
    expect(adjacentStep(path, "login", -1)).toBeNull();
    expect(adjacentStep(path, "done", 1)).toBeNull();
    expect(adjacentStep(path, "off-path", 1)).toBeNull();
  });
});
