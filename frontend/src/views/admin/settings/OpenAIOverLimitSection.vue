<template>
  <div class="border-t border-gray-100 pt-5 dark:border-dark-700">
    <div class="flex items-center justify-between">
      <div>
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t("admin.settings.openaiOverLimitMode.title") }}
        </label>
        <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
          {{ t("admin.settings.openaiOverLimitMode.description") }}
        </p>
      </div>
      <Toggle
        :model-value="Boolean(form.openai_over_limit_mode_enabled)"
        @update:model-value="
          (value) =>
            emit(
              'update:field',
              'openai_over_limit_mode_enabled',
              Boolean(value),
            )
        "
      />
    </div>

    <div v-if="form.openai_over_limit_mode_enabled" class="mt-4 space-y-2">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t("admin.settings.openaiOverLimitMode.cooldownSeconds") }}
      </label>
      <input
        :value="form.openai_over_limit_cooldown_seconds"
        type="number"
        :min="OPENAI_OVER_LIMIT_MIN_COOLDOWN_SECONDS"
        :max="OPENAI_OVER_LIMIT_MAX_COOLDOWN_SECONDS"
        class="input w-32"
        @input="emitNumberUpdate($event)"
      />
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t("admin.settings.openaiOverLimitMode.cooldownSecondsHint") }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";

import Toggle from "@/components/common/Toggle.vue";

import {
  OPENAI_OVER_LIMIT_MAX_COOLDOWN_SECONDS,
  OPENAI_OVER_LIMIT_MIN_COOLDOWN_SECONDS,
  type OpenAIOverLimitFieldKey,
  type OpenAIOverLimitSettingsForm,
} from "./openaiOverLimitFields";

defineProps<{
  form: OpenAIOverLimitSettingsForm;
}>();

const emit = defineEmits<{
  (
    event: "update:field",
    key: OpenAIOverLimitFieldKey,
    value: boolean | number,
  ): void;
}>();

const { t } = useI18n();

function emitNumberUpdate(event: Event) {
  emit(
    "update:field",
    "openai_over_limit_cooldown_seconds",
    Number((event.target as HTMLInputElement).value),
  );
}
</script>
