export function stepPosition(path: string[], stepId: string): number {
  return path.indexOf(stepId);
}

export function adjacentStep(path: string[], stepId: string, delta: -1 | 1): string | null {
  const index = path.indexOf(stepId);
  if (index < 0) {
    return null;
  }
  const next = index + delta;
  if (next < 0 || next >= path.length) {
    return null;
  }
  return path[next];
}
