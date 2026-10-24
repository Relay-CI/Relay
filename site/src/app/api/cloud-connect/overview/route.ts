import { NextResponse } from "next/server";
import { getCloudConnectOverview } from "@/lib/cloud-connect";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function GET() {
  const overview = await getCloudConnectOverview();
  return NextResponse.json(overview, {
    headers: {
      "Cache-Control": "no-store",
    },
  });
}
