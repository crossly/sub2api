import { watch } from "vue";

import type { OpenAIOverLimitSettingsForm } from "./openaiOverLimitFields";
import {
  hydrateOpenAIOverLimitSettings,
  normalizeOpenAIOverLimitCooldownSeconds,
  serializeOpenAIOverLimitSettings,
} from "./openaiOverLimitFields";

type OpenAIOverLimitFormLike = OpenAIOverLimitSettingsForm &
  Record<string, unknown>;

export function applyOpenAIOverLimitSettingsToForm<
  T extends OpenAIOverLimitFormLike,
>(form: T, settings: Partial<OpenAIOverLimitSettingsForm>) {
  Object.assign(
    form,
    hydrateOpenAIOverLimitSettings({
      openai_over_limit_mode_enabled:
        settings.openai_over_limit_mode_enabled ??
        form.openai_over_limit_mode_enabled,
      openai_over_limit_cooldown_seconds:
        settings.openai_over_limit_cooldown_seconds ??
        form.openai_over_limit_cooldown_seconds,
    }),
  );
}

export function useOpenAIOverLimitSettings<T extends OpenAIOverLimitFormLike>(
  form: T,
) {
  const normalizeOpenAIOverLimitForm = () => {
    if (!form.openai_over_limit_mode_enabled) {
      return;
    }

    form.openai_over_limit_cooldown_seconds =
      normalizeOpenAIOverLimitCooldownSeconds(
        form.openai_over_limit_cooldown_seconds,
      );
  };

  watch(
    () => form.openai_over_limit_mode_enabled,
    (enabled) => {
      if (!enabled) {
        return;
      }

      normalizeOpenAIOverLimitForm();
    },
  );

  return {
    normalizeOpenAIOverLimitForm,
    buildOpenAIOverLimitUpdateRequest: () =>
      serializeOpenAIOverLimitSettings(form),
  };
}
