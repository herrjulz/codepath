import {
  reportSchema,
  sourceSchema,
  summarySchema,
  type ReportSummary,
  type SourceView,
  type WalkthroughReport,
} from "./schema";

async function errorMessage(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as {
      message?: string;
      details?: { path: string; message: string }[];
    };
    if (body.details && body.details.length > 0) {
      return body.details.map((detail) => `${detail.path}: ${detail.message}`).join("\n");
    }
    if (body.message) {
      return body.message;
    }
  } catch {
    // The body was not JSON.
  }
  return `${res.status} ${res.statusText}`;
}

export async function listReports(): Promise<ReportSummary[]> {
  const res = await fetch("/api/reports");
  if (!res.ok) {
    throw new Error(await errorMessage(res));
  }
  return summarySchema.array().parse(await res.json());
}

export async function getReport(id: string): Promise<WalkthroughReport> {
  const res = await fetch(`/api/reports/${encodeURIComponent(id)}`);
  if (!res.ok) {
    throw new Error(await errorMessage(res));
  }
  return reportSchema.parse(await res.json());
}

export async function getSource(id: string, stepId: string, refIndex: number): Promise<SourceView> {
  const params = new URLSearchParams({ stepId, refIndex: String(refIndex) });
  const res = await fetch(`/api/reports/${encodeURIComponent(id)}/source?${params}`, {
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(await errorMessage(res));
  }
  return sourceSchema.parse(await res.json());
}
