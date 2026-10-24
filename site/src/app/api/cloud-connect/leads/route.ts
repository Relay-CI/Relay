import { NextResponse } from "next/server";
import { createCloudLead } from "@/lib/cloud-connect";

export const runtime = "nodejs";

function redirectWithStatus(request: Request, status: string) {
  const url = new URL(`/cloud?submitted=${status}#interest`, request.url);
  return NextResponse.redirect(url, { status: 303 });
}

function redirectToCustomerPanel(
  request: Request,
  token: string,
  mode: "ok" | "demo",
) {
  const url = new URL(`/cloud/customer/${token}?submitted=${mode}`, request.url);
  return NextResponse.redirect(url, { status: 303 });
}

export async function POST(request: Request) {
  const formData = await request.formData();

  const email = String(formData.get("email") ?? "").trim().toLowerCase();
  const teamName = String(formData.get("teamName") ?? "").trim();
  const fleetSizeRaw = String(formData.get("fleetSize") ?? "").trim();
  const needs = String(formData.get("needs") ?? "").trim();

  if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    return redirectWithStatus(request, "invalid");
  }

  const fleetSize = fleetSizeRaw ? Number.parseInt(fleetSizeRaw, 10) : null;
  if (fleetSizeRaw && Number.isNaN(fleetSize)) {
    return redirectWithStatus(request, "invalid");
  }

  try {
    const result = await createCloudLead({
      email,
      teamName: teamName || null,
      fleetSize,
      needs: needs || null,
    });

    return redirectToCustomerPanel(request, result.token, result.stored ? "ok" : "demo");
  } catch (error) {
    console.error("Failed to store Cloud Connect lead", error);
    return redirectWithStatus(request, "error");
  }
}
