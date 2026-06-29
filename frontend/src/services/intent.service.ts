import javaClient from "./javaClient";

export type ChannelPreference = "PUBLIC" | "PRIVATE";

export interface IntentSuggestion {
  suggestedChannel: ChannelPreference;
  confidence: number;
  reasoning: string;
}

interface IntentRequest {
  content: string;
  channelHint: ChannelPreference;
}

interface IntentResponseWire {
  suggestedChannel: string;
  confidence: number;
  reasoning: string;
}

export const intentService = {
  async classify(content: string, channelHint: ChannelPreference): Promise<IntentSuggestion> {
    const body: IntentRequest = { content, channelHint };
    const response = await javaClient.post<IntentResponseWire>("/api/v1/classify-intent", body);
    const suggested = response.data?.suggestedChannel?.toUpperCase() === "PRIVATE" ? "PRIVATE" : "PUBLIC";
    return {
      suggestedChannel: suggested,
      confidence: Number(response.data?.confidence ?? 0),
      reasoning: String(response.data?.reasoning ?? ""),
    };
  },
};
