import type { SystemSettings, UpdateSettingsRequest } from "@/api/admin/settings";

export type OpenAIOverLimitSettingsForm = Required<
  Pick<
    SystemSettings,
    | "openai_over_limit_mode_enabled"
    | "openai_over_limit_cooldown_seconds"
  >
>;

export type OpenAIOverLimitFieldKey = keyof OpenAIOverLimitSettingsForm;

export const OPENAI_OVER_LIMIT_MIN_COOLDOWN_SECONDS = 10;
export const OPENAI_OVER_LIMIT_MAX_COOLDOWN_SECONDS = 300;

export const OPENAI_OVER_LIMIT_DEFAULTS: OpenAIOverLimitSettingsForm = {
  openai_over_limit_mode_enabled: false,
  openai_over_limit_cooldown_seconds: OPENAI_OVER_LIMIT_MIN_COOLDOWN_SECONDS,
};

export function normalizeOpenAIOverLimitCooldownSeconds(value: unknown): number {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return OPENAI_OVER_LIMIT_MIN_COOLDOWN_SECONDS;
  }

  const integerValue = Math.trunc(parsed);
  if (integerValue < OPENAI_OVER_LIMIT_MIN_COOLDOWN_SECONDS) {
    return OPENAI_OVER_LIMIT_MIN_COOLDOWN_SECONDS;
  }
  if (integerValue > OPENAI_OVER_LIMIT_MAX_COOLDOWN_SECONDS) {
    return OPENAI_OVER_LIMIT_MAX_COOLDOWN_SECONDS;
  }

  return integerValue;
}

export function hydrateOpenAIOverLimitSettings(
  settings: Partial<OpenAIOverLimitSettingsForm>,
): OpenAIOverLimitSettingsForm {
  return {
    openai_over_limit_mode_enabled: Boolean(
      settings.openai_over_limit_mode_enabled ??
        OPENAI_OVER_LIMIT_DEFAULTS.openai_over_limit_mode_enabled,
    ),
    openai_over_limit_cooldown_seconds: normalizeOpenAIOverLimitCooldownSeconds(
      settings.openai_over_limit_cooldown_seconds ??
        OPENAI_OVER_LIMIT_DEFAULTS.openai_over_limit_cooldown_seconds,
    ),
  };
}

export function serializeOpenAIOverLimitSettings(
  form: OpenAIOverLimitSettingsForm,
): Pick<
  UpdateSettingsRequest,
  | "openai_over_limit_mode_enabled"
  | "openai_over_limit_cooldown_seconds"
> {
  return {
    openai_over_limit_mode_enabled: Boolean(form.openai_over_limit_mode_enabled),
    openai_over_limit_cooldown_seconds: form.openai_over_limit_mode_enabled
      ? normalizeOpenAIOverLimitCooldownSeconds(
          form.openai_over_limit_cooldown_seconds,
        )
      : 0,
  };
}
