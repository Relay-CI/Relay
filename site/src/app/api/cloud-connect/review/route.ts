import { NextResponse } from "next/server";
import type { LeadPriority, LeadStatus } from "@/lib/cloud-connect";
import { updateCloudLeadReview } from "@/lib/cloud-connect";

export const runtime = "nodejs";

const allowedStatuses: LeadStatus[] = [
  "new",
  "reviewing",
  "qualified",
  "pilot",
  "won",
  "rejected",
];

const allowedPriorities: LeadPriority[] = [
  "low",
  "medium",
  "high",
  "urgent",
];

function redirectToAdmin(request: Request, leadId: string, result: string) {
  const url = new URL(`/cloud/admin?updated=${result}#lead-${leadId}`, request.url);
  return NextResponse.redirect(url, { status: 303 });
}

export async function POST(request: Request) {
  const formData = await request.formData();

  const leadId = String(formData.get("leadId") ?? "").trim();
  const status = String(formData.get("status") ?? "").trim() as LeadStatus;
  const priority = String(formData.get("priority") ?? "").trim() as LeadPriority;
  const ownerName = String(formData.get("ownerName") ?? "").trim();
  const reviewNotes = String(formData.get("reviewNotes") ?? "").trim();

  if (
    !leadId ||
    !allowedStatuses.includes(status) ||
    !allowedPriorities.includes(priority)
  ) {
    return redirectToAdmin(request, leadId || "invalid", "invalid");
  }

  try {
    const result = await updateCloudLeadReview({
      leadId,
      status,
      priority,
      ownerName: ownerName || null,
      reviewNotes: reviewNotes || null,
    });

    return redirectToAdmin(request, leadId, result.updated ? "ok" : "demo");
  } catch (error) {
    console.error("Failed to update Cloud Connect lead", error);
    return redirectToAdmin(request, leadId, "error");
  }
}
