import { describe, expect, it } from "vitest";

import {
  OPENAI_OVER_LIMIT_DEFAULTS,
  hydrateOpenAIOverLimitSettings,
  serializeOpenAIOverLimitSettings,
} from "../openaiOverLimitFields";

describe("openaiOverLimitFields", () => {
  it("hydrates missing over-limit settings with stable defaults", () => {
    expect(hydrateOpenAIOverLimitSettings({})).toEqual(
      OPENAI_OVER_LIMIT_DEFAULTS,
    );
  });

  it("serializes only over-limit keys for the flat admin payload", () => {
    expect(
      serializeOpenAIOverLimitSettings({
        openai_over_limit_mode_enabled: true,
        openai_over_limit_cooldown_seconds: 0,
      }),
    ).toEqual({
      openai_over_limit_mode_enabled: true,
      openai_over_limit_cooldown_seconds: 10,
    });
  });
});
