import { NextResponse } from "next/server";

export async function POST(request: Request) {
  try {
    const { text } = await request.json();
    
    // Simple heuristic for PII in the mock
    // Patterns for MSSV (Student ID), Phone numbers (09...), or Keywords like "Điểm"
    const piiPatterns = [
      /\b\d{8,10}\b/, // MSSV typically 8-10 digits
      /0[35789]\d{8}/, // Vietnamese phone numbers
      /điểm/i,
      /mssv/i,
      /số điện thoại/i,
      /cmnd/i,
      /cccd/i
    ];

    const hasPii = piiPatterns.some(pattern => pattern.test(text));

    // Simulate small latency
    await new Promise(resolve => setTimeout(resolve, 100));

    return NextResponse.json({ hasPii });
  } catch (error) {
    return NextResponse.json({ error: "Invalid request" }, { status: 400 });
  }
}
