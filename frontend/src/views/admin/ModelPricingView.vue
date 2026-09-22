<template>
  <AppLayout>
    <TablePageLayout>
      <template #header>
        <PageHeader
          :title="t('admin.modelPricing.title')"
          :description="t('admin.modelPricing.description')"
        />
      </template>

      <template #actions>

        <button
          type="button"
          class="btn btn-secondary"
          :title="t('admin.modelPricing.defaults.open')"
          @click="openDefaultsDialog"
        >
          <Icon name="dollar" size="md" />
          <span class="hidden md:inline">{{ t('admin.modelPricing.defaults.open') }}</span>
          <span
            v-if="defaultsTunedCount > 0"
            class="tag tag-primary ml-1"
            :title="t('admin.modelPricing.defaults.tunedCount', { count: defaultsTunedCount })"
          >
            {{ defaultsTunedCount }}
          </span>
        </button>
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="loading"
          :title="t('admin.modelPricing.refresh')"
          @click="handleRefresh"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          <span class="hidden md:inline">{{ t('admin.modelPricing.refresh') }}</span>
        </button>

        <button
          type="button"
          class="btn btn-secondary"
          :disabled="exporting"
          :title="exporting ? t('admin.modelPricing.overrides.exporting') : t('admin.modelPricing.overrides.export')"
          @click="handleExportOverrides"
        >
          <Icon name="download" size="md" :class="exporting ? 'animate-pulse' : ''" />
          <span class="hidden md:inline">{{ t('admin.modelPricing.overrides.export') }}</span>
        </button>

        <button
          type="button"
          class="btn btn-secondary"
          :disabled="overridesLoading"
          :title="t('admin.modelPricing.overrides.import')"
          @click="openImportDialog"
        >
          <Icon name="upload" size="md" />
          <span class="hidden md:inline">{{ t('admin.modelPricing.overrides.import') }}</span>
        </button>

        <button
          type="button"
          class="btn btn-primary"
          :disabled="refreshing"
          :title="t('admin.modelPricing.overrides.fetchLatest')"
          @click="openRefreshDialog"
        >
          <Icon name="sync" size="md" :class="refreshing ? 'animate-spin' : ''" />
          <span class="hidden md:inline">{{ t('admin.modelPricing.overrides.fetchLatest') }}</span>
        </button>
      </template>

      <template #filters>
        <div class="filter-bar flex flex-col justify-between gap-4 lg:flex-row lg:items-start">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-64">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="filters.q"
                type="text"
                class="input pl-10"
                :placeholder="t('admin.modelPricing.searchPlaceholder')"
                :aria-label="t('admin.modelPricing.searchPlaceholder')"
                @input="handleSearchInput"
              />
            </div>

            <Select
              v-model="filters.provider"
              :options="providerOptions"
              :placeholder="t('admin.modelPricing.allProviders')"
              :aria-label="t('admin.modelPricing.providerFilter')"
              :loading="providersLoading"
              class="w-56"
              @change="applyFilters"
            />

            <Select
              v-model="filters.source"
              :options="sourceOptions"
              :placeholder="t('admin.modelPricing.sourceFilter')"
              :aria-label="t('admin.modelPricing.sourceFilter')"
              class="w-44"
              @change="applyFilters"
            />

            <div class="flex items-center gap-2">
              <Toggle
                :model-value="filters.timeDependent"
                :aria-label="t('admin.modelPricing.onlyTimeDependent')"
                @update:model-value="setTimeDependent"
              />
              <span class="text-control text-gray-700 dark:text-dark-300">
                {{ t('admin.modelPricing.onlyTimeDependent') }}
              </span>
            </div>

            <div class="flex items-center gap-2">
              <Toggle
                :model-value="filters.hasExpr"
                :aria-label="t('admin.modelPricing.onlyWithExpr')"
                @update:model-value="setHasExpr"
              />
              <span class="text-control text-gray-700 dark:text-dark-300">
                {{ t('admin.modelPricing.onlyWithExpr') }}
              </span>
            </div>
          </div>

          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <button
              v-if="hasActiveFilters"
              type="button"
              class="btn btn-ghost"
              @click="clearFilters"
            >
              <Icon name="x" size="md" class="mr-2" />
              {{ t('admin.modelPricing.clearFilters') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <!-- Load failed with nothing to show: explain instead of implying an empty catalog. -->
        <div v-if="loadError && items.length === 0" class="p-6">
          <EmptyState
            :title="t('admin.modelPricing.loadError')"
            :description="loadError"
            :action-text="t('admin.modelPricing.retry')"
            @action="handleRefresh"
          />
        </div>

        <template v-else>
          <!-- Stale data stays visible when a manual refresh fails. -->
          <div
            v-if="loadError"
            role="alert"
            class="flex flex-wrap items-center justify-between gap-3 border-b border-red-200 bg-red-50 px-4 py-2 text-caption text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400"
          >
            <span>{{ t('admin.modelPricing.loadError') }}：{{ loadError }}</span>
            <button type="button" class="btn btn-secondary" @click="handleRefresh">
              {{ t('admin.modelPricing.retry') }}
            </button>
          </div>

          <!-- The price catalog loaded; a failed override list only disables tuning. -->
          <div
            v-if="overridesError"
            role="alert"
            class="flex flex-wrap items-center justify-between gap-3 border-b border-amber-200 bg-amber-50 px-4 py-2 text-caption text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300"
          >
            <span>{{ t('admin.modelPricing.overrides.loadError') }}：{{ overridesError }}</span>
            <button type="button" class="btn btn-secondary" @click="loadOverrides()">
              {{ t('admin.modelPricing.overrides.loadRetry') }}
            </button>
          </div>

          <DataTable
            :columns="columns"
            :data="items"
            :loading="loading"
            row-key="model"
            :estimate-row-height="64"
          >
            <template #header-input>
              <span class="flex flex-col leading-tight">
                <span>{{ t('admin.modelPricing.columns.input') }}</span>
                <span class="text-2xs font-normal text-gray-500 dark:text-dark-400">
                  {{ t('admin.modelPricing.unitHint') }}
                </span>
              </span>
              <HelpTooltip :content="t('admin.modelPricing.baselineHeaderTooltip')" />
            </template>

            <template #cell-model="{ row }">
              <div class="min-w-0 max-w-[240px] whitespace-normal">
                <span class="font-medium text-gray-900 dark:text-white">{{ row.model }}</span>
                <span v-if="row.mode" class="ml-1.5 text-caption text-gray-500 dark:text-dark-400">
                  {{ row.mode }}
                </span>
                <span
                  v-if="isTuned(row.model)"
                  class="tag tag-primary ml-1.5 align-middle"
                  :title="t('admin.modelPricing.overrides.tunedBadgeTooltip')"
                >
                  <Icon name="edit" size="xs" />
                  {{ t('admin.modelPricing.overrides.tunedBadge') }}
                </span>
              </div>
            </template>

            <template #cell-provider="{ row }">
              <span class="text-caption text-gray-600 dark:text-dark-400">
                {{ row.provider || '-' }}
              </span>
            </template>

            <template #cell-source="{ row }">
              <span :class="['tag', row.source === 'builtin' ? 'tag-warning' : 'tag-gray']">
                {{ sourceLabel(row.source) }}
              </span>
            </template>

            <template #cell-input="{ row }">
              <span class="inline-flex items-center gap-1.5">
                <span
                  class="tabular-nums"
                  :class="
                    row.token_pricing_absent ? 'text-gray-400 dark:text-dark-500' : 'text-gray-900 dark:text-dark-300'
                  "
                >
                  {{ priceCell(row, 'input_price_per_million') }}
                </span>
                <span
                  v-if="row.billing_expr && !row.token_pricing_absent"
                  class="tag tag-outline"
                  :title="t('admin.modelPricing.baselineTooltip')"
                >
                  {{ t('admin.modelPricing.baseline') }}
                </span>
              </span>
            </template>

            <template #cell-output="{ row }">
              <span class="inline-flex items-center gap-1.5">
                <span
                  class="tabular-nums"
                  :class="
                    row.token_pricing_absent ? 'text-gray-400 dark:text-dark-500' : 'text-gray-900 dark:text-dark-300'
                  "
                >
                  {{ priceCell(row, 'output_price_per_million') }}
                </span>
                <span
                  v-if="row.billing_expr && !row.token_pricing_absent"
                  class="tag tag-outline"
                  :title="t('admin.modelPricing.baselineTooltip')"
                >
                  {{ t('admin.modelPricing.baseline') }}
                </span>
              </span>
            </template>

            <template #cell-cache_read="{ row }">
              <span
                class="tabular-nums"
                :class="
                  row.token_pricing_absent ? 'text-gray-400 dark:text-dark-500' : 'text-gray-900 dark:text-dark-300'
                "
              >
                {{ priceCell(row, 'cache_read_price_per_million') }}
              </span>
            </template>

            <template #cell-cache_write="{ row }">
              <span
                class="tabular-nums"
                :class="
                  row.token_pricing_absent ? 'text-gray-400 dark:text-dark-500' : 'text-gray-900 dark:text-dark-300'
                "
              >
                {{ priceCell(row, 'cache_creation_price_per_million') }}
              </span>
            </template>

            <template #cell-billing="{ row }">
              <div class="min-w-0 max-w-[420px] whitespace-normal text-left">
                <!-- Recognised tiers: one compact line per tier. -->
                <template v-if="visibleTiers(row).length">
                  <div
                    v-for="tier in visibleTiers(row)"
                    :key="tierKey(tier)"
                    class="leading-5"
                  >
                    <span class="font-medium text-gray-900 dark:text-dark-300">
                      {{ tierLabel(tier.name) }}
                    </span>
                    <span
                      v-if="tierConditionText(tier)"
                      class="text-caption text-gray-500 dark:text-dark-400"
                    >
                      （{{ tierConditionText(tier) }}）
                    </span>
                    <span class="text-gray-500 dark:text-dark-400">：</span>
                    <span
                      v-for="(segment, index) in tierChargeSegments(tier)"
                      :key="segment.label"
                      class="text-caption text-gray-700 dark:text-dark-300"
                    >
                      <span v-if="index > 0" class="mx-0.5 text-gray-300 dark:text-dark-600">/</span>
                      {{ segment.label }} {{ segment.value }}
                    </span>
                  </div>
                  <div v-if="hiddenTierCount(row) > 0" class="text-caption text-gray-500 dark:text-dark-400">
                    {{ t('admin.modelPricing.billing.moreTiers', { count: hiddenTierCount(row) }) }}
                  </div>
                </template>

                <!-- Expression exists but is not a recognised tier shape: show the raw text. -->
                <template v-else-if="row.billing_expr">
                  <code
                    class="line-clamp-3 block break-all font-mono text-caption text-gray-700 dark:text-dark-300"
                  >
                    {{ row.billing_expr }}
                  </code>
                  <p class="text-caption text-amber-700 dark:text-amber-400">
                    {{ t('admin.modelPricing.billing.unrecognized') }}
                  </p>
                </template>

                <!-- No expression: the base rates are the price. -->
                <template v-else>
                  <span class="text-caption text-gray-600 dark:text-dark-400">
                    {{ t('admin.modelPricing.billing.byTable') }}
                  </span>
                  <HelpTooltip :content="t('admin.modelPricing.billing.byTableTooltip')" />
                </template>

                <button
                  v-if="row.billing_expr"
                  type="button"
                  class="mt-1 inline-flex items-center gap-1 text-caption font-medium text-primary-600 transition-colors duration-150 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                  @click="openExprDialog(row)"
                >
                  <Icon name="terminal" size="xs" />
                  {{ t('admin.modelPricing.billing.viewExpr') }}
                </button>
              </div>
            </template>

            <template #cell-flags="{ row }">
              <div class="flex flex-wrap items-center gap-1">
                <span v-if="row.time_dependent" class="tag tag-primary">
                  <Icon name="clock" size="xs" />
                  {{ t('admin.modelPricing.flags.timeDependent') }}
                </span>
                <span v-if="row.fallback_only" class="tag tag-gray">
                  {{ t('admin.modelPricing.flags.builtinOnly') }}
                </span>
                <span v-if="row.token_pricing_absent" class="tag tag-warning">
                  {{ t('admin.modelPricing.flags.noTokenPricing') }}
                </span>
                <span
                  v-if="!row.time_dependent && !row.fallback_only && !row.token_pricing_absent"
                  class="text-caption text-gray-400 dark:text-dark-500"
                >
                  -
                </span>
              </div>
            </template>

            <template #cell-actions="{ row }">
              <button
                type="button"
                class="btn btn-sm"
                :class="isTuned(row.model) ? 'btn-secondary' : 'btn-ghost'"
                :aria-label="t('admin.modelPricing.overrides.tuneTitle', { model: row.model })"
                @click="openTuneDialog(row)"
              >
                <Icon name="edit" size="sm" />
                {{ t('admin.modelPricing.overrides.tune') }}
              </button>
            </template>

            <template #empty>
              <EmptyState
                v-if="hasActiveFilters"
                :title="t('admin.modelPricing.emptyFilteredTitle')"
                :description="t('admin.modelPricing.emptyFilteredDescription')"
                :action-text="t('admin.modelPricing.clearFilters')"
                @action="clearFilters"
              />
              <EmptyState
                v-else
                :title="t('admin.modelPricing.emptyTitle')"
                :description="t('admin.modelPricing.emptyDescription')"
              />
            </template>
          </DataTable>

          <!-- Precedence contract: tuning never outranks explicit group/channel pricing. -->
          <p
            class="flex flex-wrap items-center gap-x-2 gap-y-1 border-t border-gray-200 px-4 py-2.5 text-caption text-gray-500 dark:border-dark-700 dark:text-dark-400"
          >
            <Icon name="infoCircle" size="xs" class="shrink-0" />
            <span>{{ t('admin.modelPricing.overrides.priorityNote') }}</span>
            <span
              v-if="overridesPath"
              class="truncate font-mono text-2xs text-gray-400 dark:text-dark-500"
              :title="t('admin.modelPricing.overrides.pathLabel', { path: overridesPath })"
            >
              {{ t('admin.modelPricing.overrides.pathLabel', { path: overridesPath }) }}
            </span>
          </p>
        </template>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <!-- Full expression + tier detail: the table cell cannot hold it without breaking the grid. -->
    <BaseDialog
      :show="dialogItem !== null"
      :title="dialogTitle"
      width="wide"
      @close="closeExprDialog"
    >
      <div v-if="dialogItem" class="space-y-5">
        <div class="flex flex-wrap items-center gap-2">
          <span class="tag tag-gray">{{ sourceLabel(dialogItem.source) }}</span>
          <span v-if="dialogItem.provider" class="tag tag-outline">{{ dialogItem.provider }}</span>
          <span v-if="dialogItem.time_dependent" class="tag tag-primary">
            <Icon name="clock" size="xs" />
            {{ t('admin.modelPricing.flags.timeDependent') }}
          </span>
          <span v-if="dialogItem.fallback_only" class="tag tag-warning">
            {{ t('admin.modelPricing.flags.builtinOnly') }}
          </span>
          <span v-if="dialogItem.token_pricing_absent" class="tag tag-warning">
            {{ t('admin.modelPricing.flags.noTokenPricing') }}
          </span>
        </div>

        <p v-if="dialogItem.token_pricing_absent" class="text-caption text-amber-700 dark:text-amber-400">
          {{ t('admin.modelPricing.billing.noTokenPricingHint') }}
        </p>
        <p
          v-else-if="dialogItem.billing_expr"
          class="text-caption text-gray-500 dark:text-dark-400"
        >
          {{ t('admin.modelPricing.baselineTooltip') }}
        </p>
        <p v-if="dialogItem.fallback_only" class="text-caption text-amber-700 dark:text-amber-400">
          {{ t('admin.modelPricing.billing.fallbackHint') }}
        </p>

        <section v-if="dialogItem.billing_expr" class="space-y-2">
          <h4 class="border-b border-gray-200 pb-1.5 text-control font-medium text-gray-900 dark:border-dark-700 dark:text-white">
            {{ t('admin.modelPricing.billing.exprSection') }}
          </h4>
          <pre
            class="whitespace-pre-wrap break-all rounded-lg bg-gray-100 p-3 font-mono text-caption text-gray-700 dark:bg-dark-800 dark:text-dark-300"
          >{{ dialogItem.billing_expr }}</pre>
          <p class="text-caption text-gray-500 dark:text-dark-400">
            {{ t('admin.modelPricing.billing.exprSource') }}：{{ exprSourceLabel(dialogItem.billing_expr_source) }}
          </p>
        </section>

        <section class="space-y-3">
          <h4 class="border-b border-gray-200 pb-1.5 text-control font-medium text-gray-900 dark:border-dark-700 dark:text-white">
            {{ dialogItem.tiers && dialogItem.tiers.length ? t('admin.modelPricing.billing.tierSection') : t('admin.modelPricing.billing.noTierSection') }}
          </h4>

          <p v-if="!(dialogItem.tiers && dialogItem.tiers.length)" class="text-caption text-amber-700 dark:text-amber-400">
            {{ t('admin.modelPricing.billing.unrecognized') }}
          </p>

          <div
            v-for="tier in dialogItem.tiers || []"
            :key="tierKey(tier)"
            class="surface-flat p-3"
          >
            <div class="flex flex-wrap items-center gap-2">
              <span class="text-control font-medium text-gray-900 dark:text-white">
                {{ tierLabel(tier.name) }}
              </span>
              <span class="tag tag-gray">{{ unitLabel(tier.unit) }}</span>
            </div>

            <dl class="mt-2 space-y-1 text-caption">
              <div class="flex gap-2">
                <dt class="w-20 shrink-0 text-gray-500 dark:text-dark-400">
                  {{ t('admin.modelPricing.billing.timeWindowLabel') }}
                </dt>
                <dd class="min-w-0 text-gray-700 dark:text-dark-300">
                  {{ tierWindowText(tier) || '-' }}
                </dd>
              </div>
              <div v-if="tier.timezone" class="flex gap-2">
                <dt class="w-20 shrink-0 text-gray-500 dark:text-dark-400">
                  {{ t('admin.modelPricing.billing.timezoneLabel') }}
                </dt>
                <dd class="min-w-0 text-gray-700 dark:text-dark-300">{{ tier.timezone }}</dd>
              </div>
              <div class="flex gap-2">
                <dt class="w-20 shrink-0 text-gray-500 dark:text-dark-400">
                  {{ t('admin.modelPricing.billing.conditionLabel') }}
                </dt>
                <dd class="min-w-0 break-all font-mono text-gray-700 dark:text-dark-300">
                  {{ tier.condition || t('admin.modelPricing.billing.noCondition') }}
                </dd>
              </div>
              <div v-if="tier.constant" class="flex gap-2">
                <dt class="w-20 shrink-0 text-gray-500 dark:text-dark-400">
                  {{ t('admin.modelPricing.billing.constantLabel') }}
                </dt>
                <dd class="min-w-0 tabular-nums text-gray-700 dark:text-dark-300">
                  {{ t('admin.modelPricing.billing.constantValue', { value: `$${formatMoney(tier.constant)}` }) }}
                </dd>
              </div>
            </dl>

            <div class="mt-2">
              <p class="text-caption text-gray-500 dark:text-dark-400">
                {{ t('admin.modelPricing.billing.variablesLabel') }}
              </p>
              <div class="mt-1 grid grid-cols-2 gap-1 sm:grid-cols-3">
                <div
                  v-for="segment in tierChargeSegments(tier)"
                  :key="segment.label"
                  class="flex items-center justify-between gap-2 border-b border-gray-100 py-1 text-caption dark:border-dark-800"
                >
                  <span class="text-gray-600 dark:text-dark-400">{{ segment.label }}</span>
                  <span class="tabular-nums text-gray-900 dark:text-dark-300">{{ segment.value }}</span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section v-if="!dialogItem.token_pricing_absent" class="space-y-2">
          <h4 class="border-b border-gray-200 pb-1.5 text-control font-medium text-gray-900 dark:border-dark-700 dark:text-white">
            {{ t('admin.modelPricing.billing.baselineSection') }}
          </h4>
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <div
              v-for="cell in baselineCells(dialogItem)"
              :key="cell.label"
              class="surface-flat px-3 py-2"
            >
              <p class="text-caption text-gray-500 dark:text-dark-400">{{ cell.label }}</p>
              <p class="tabular-nums text-control text-gray-900 dark:text-dark-300">{{ cell.value }}</p>
            </div>
          </div>
        </section>
      </div>

      <template #footer>
        <button type="button" class="btn btn-secondary" @click="closeExprDialog">
          {{ t('common.close') }}
        </button>
      </template>
    </BaseDialog>

    <!--
      Manual tuning. The override file stores USD per token while the page shows
      USD / 1M tokens, so the form converts on open (×1e6) and on submit (÷1e6).
    -->
    <BaseDialog
      :show="tuneTarget !== null"
      :title="tuneTitle"
      width="normal"
      @close="closeTuneDialog"
    >
      <form v-if="tuneTarget" id="tune-pricing-form" class="space-y-5" @submit.prevent="submitTune">
        <section class="space-y-2">
          <div class="flex items-center gap-1.5">
            <h4 class="text-control font-medium text-gray-900 dark:text-white">
              {{ t('admin.modelPricing.overrides.currentPriceTitle') }}
            </h4>
            <HelpTooltip :content="t('admin.modelPricing.overrides.currentPriceTooltip')" />
          </div>
          <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <div
              v-for="cell in baselineCells(tuneTarget)"
              :key="cell.label"
              class="surface-flat px-3 py-2"
            >
              <p class="text-caption text-gray-500 dark:text-dark-400">{{ cell.label }}</p>
              <p class="tabular-nums text-control text-gray-900 dark:text-dark-300">{{ cell.value }}</p>
            </div>
          </div>
          <p v-if="tuneEntry?.updated_at" class="text-caption text-gray-500 dark:text-dark-400">
            {{ t('admin.modelPricing.overrides.updatedAt', { date: formatDateTime(tuneEntry.updated_at) }) }}
          </p>
        </section>

        <section class="space-y-3">
          <h4 class="border-b border-gray-200 pb-1.5 text-control font-medium text-gray-900 dark:border-dark-700 dark:text-white">
            {{ t('admin.modelPricing.overrides.fieldsSection') }}
          </h4>
          <p class="text-caption text-gray-500 dark:text-dark-400">
            {{ t('admin.modelPricing.overrides.fieldsUnitHint') }}
          </p>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div v-for="spec in tuneFieldSpecs" :key="spec.key">
              <label
                class="input-label flex items-center justify-between gap-2"
                :for="`tune-field-${spec.key}`"
              >
                <span>{{ t(spec.titleKey) }}</span>
                <span class="text-caption font-normal text-gray-500 dark:text-dark-400">
                  {{ t(spec.unitKey) }}
                </span>
              </label>
              <input
                :id="`tune-field-${spec.key}`"
                v-model="tuneForm[spec.key]"
                type="text"
                inputmode="decimal"
                autocomplete="off"
                class="input h-control tabular-nums text-control"
                :disabled="tuneBusy"
                @keydown.enter.prevent="submitTune"
              />
              <p v-if="spec.perMillionKey" class="input-hint">
                {{
                  t('admin.modelPricing.overrides.cardValue', {
                    value: priceCell(tuneTarget, spec.perMillionKey)
                  })
                }}
              </p>
              <p v-else-if="spec.hintKey" class="input-hint">
                {{ t(spec.hintKey) }}
              </p>
            </div>
          </div>
          <p class="text-caption text-gray-500 dark:text-dark-400">
            {{ t('admin.modelPricing.overrides.imageHint') }}
          </p>
        </section>

        <section class="space-y-2">
          <h4 class="border-b border-gray-200 pb-1.5 text-control font-medium text-gray-900 dark:border-dark-700 dark:text-white">
            {{ t('admin.modelPricing.overrides.exprLabel') }}
          </h4>
          <textarea
            v-model="tuneForm.expr"
            rows="3"
            class="input min-h-[72px] resize-y font-mono text-caption leading-relaxed"
            :aria-label="t('admin.modelPricing.overrides.exprLabel')"
            :placeholder="t('admin.modelPricing.overrides.exprPlaceholder')"
            :disabled="tuneBusy"
          ></textarea>
          <p class="input-hint">{{ t('admin.modelPricing.overrides.exprHint') }}</p>
          <p v-if="tuneExprCleared" class="text-caption text-amber-700 dark:text-amber-400">
            {{ t('admin.modelPricing.overrides.exprWillClear') }}
          </p>
          <p
            v-else-if="!tuneExprDirty && tuneOriginalExpr"
            class="text-caption text-gray-500 dark:text-dark-400"
          >
            {{ t('admin.modelPricing.overrides.exprUnchanged') }}
          </p>
        </section>

        <div
          v-if="restoreConfirm"
          class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-caption text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
        >
          <span>{{ t('admin.modelPricing.overrides.restoreConfirm') }}</span>
          <span class="flex items-center gap-2">
            <button
              type="button"
              class="btn btn-sm btn-secondary"
              :disabled="tuneBusy"
              @click="restoreConfirm = false"
            >
              {{ t('common.cancel') }}
            </button>
            <button type="button" class="btn btn-sm btn-danger" :disabled="tuneBusy" @click="submitRestore">
              {{
                tuneRestoring
                  ? t('admin.modelPricing.overrides.restoring')
                  : t('admin.modelPricing.overrides.restoreConfirmAction')
              }}
            </button>
          </span>
        </div>

        <p v-if="tuneError" role="alert" class="text-caption text-red-600 dark:text-red-400">
          {{ tuneError }}
        </p>
      </form>

      <template #footer>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-2">
            <button
              v-if="tuneEntry"
              type="button"
              class="btn btn-sm btn-danger-soft"
              :disabled="tuneBusy"
              @click="restoreConfirm = true"
            >
              {{ t('admin.modelPricing.overrides.restore') }}
            </button>
            <button
              v-if="tuneExprPresent"
              type="button"
              class="btn btn-sm btn-secondary"
              :disabled="tuneBusy"
              @click="clearTuneExpr"
            >
              {{ t('admin.modelPricing.overrides.exprClear') }}
            </button>
            <button
              v-else-if="tuneOriginalExpr"
              type="button"
              class="btn btn-sm btn-secondary"
              :disabled="tuneBusy"
              @click="restoreTuneExpr"
            >
              {{ t('admin.modelPricing.overrides.exprRestore') }}
            </button>
          </div>
          <div class="flex items-center gap-2">
            <button type="button" class="btn btn-secondary" :disabled="tuneBusy" @click="closeTuneDialog">
              {{ t('common.cancel') }}
            </button>
            <button type="submit" form="tune-pricing-form" class="btn btn-primary" :disabled="tuneBusy">
              {{ tuneSaving ? t('admin.modelPricing.overrides.saving') : t('admin.modelPricing.overrides.save') }}
            </button>
          </div>
        </div>
      </template>
    </BaseDialog>

    <!-- Latest-price fetch: the only choice that matters is what happens to tuned entries. -->
    <BaseDialog
      :show="refreshDialogOpen"
      :title="t('admin.modelPricing.overrides.refreshTitle')"
      width="normal"
      @close="closeRefreshDialog"
    >
      <div class="space-y-4">
        <p class="text-control text-gray-700 dark:text-dark-300">
          {{ t('admin.modelPricing.overrides.refreshIntro') }}
        </p>
        <p class="text-caption text-gray-500 dark:text-dark-400">
          {{
            t('admin.modelPricing.overrides.entrySummary', {
              count: overridesCount,
              tuned: tunedOverrideCount
            })
          }}
        </p>

        <div class="surface-flat space-y-1.5 p-3">
          <p class="flex items-center gap-1.5 text-control font-medium text-gray-900 dark:text-white">
            <Icon name="copy" size="sm" class="text-amber-600 dark:text-amber-400" />
            {{ t('admin.modelPricing.overrides.refreshOverwrite') }}
          </p>
          <p class="text-caption text-gray-600 dark:text-dark-400">
            {{ t('admin.modelPricing.overrides.refreshOverwriteDescription') }}
          </p>
          <p class="text-caption text-amber-700 dark:text-amber-400">
            {{
              t('admin.modelPricing.overrides.refreshOverwriteWarning', { count: tunedOverrideCount })
            }}
          </p>
          <button
            type="button"
            class="btn btn-sm btn-danger-soft mt-1"
            :disabled="refreshing"
            @click="runRefresh(true)"
          >
            <Icon name="copy" size="sm" />
            {{ t('admin.modelPricing.overrides.refreshOverwrite') }}
          </button>
        </div>

        <div class="surface-flat space-y-1.5 p-3">
          <p class="flex items-center gap-1.5 text-control font-medium text-gray-900 dark:text-white">
            <Icon name="shield" size="sm" class="text-primary-600 dark:text-primary-400" />
            {{ t('admin.modelPricing.overrides.refreshKeep') }}
          </p>
          <p class="text-caption text-gray-600 dark:text-dark-400">
            {{ t('admin.modelPricing.overrides.refreshKeepDescription') }}
          </p>
          <p class="text-caption text-gray-500 dark:text-dark-400">
            {{ t('admin.modelPricing.overrides.refreshKeepWarning', { count: tunedOverrideCount }) }}
          </p>
          <button
            type="button"
            class="btn btn-sm btn-primary mt-1"
            :disabled="refreshing"
            @click="runRefresh(false)"
          >
            <Icon name="shield" size="sm" />
            {{ t('admin.modelPricing.overrides.refreshKeep') }}
          </button>
        </div>

        <p class="text-caption text-gray-500 dark:text-dark-400">
          {{ t('admin.modelPricing.overrides.priorityNote') }}
        </p>
      </div>

      <template #footer>
        <div class="flex items-center justify-end gap-3">
          <button type="button" class="btn btn-secondary" :disabled="refreshing" @click="closeRefreshDialog">
            {{ refreshing ? t('admin.modelPricing.overrides.refreshRunning') : t('common.cancel') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Import the override file verbatim: the JSON body is the file itself. -->
    <BaseDialog
      :show="importDialogOpen"
      :title="t('admin.modelPricing.overrides.importTitle')"
      width="normal"
      @close="closeImportDialog"
    >
      <form id="import-overrides-form" class="space-y-4" @submit.prevent="submitImport">
        <p class="text-control text-gray-700 dark:text-dark-300">
          {{ t('admin.modelPricing.overrides.importIntro') }}
        </p>

        <div>
          <span class="input-label">{{ t('admin.modelPricing.overrides.importFile') }}</span>
          <div
            class="flex items-center justify-between gap-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="min-w-0">
              <div class="truncate text-control text-gray-700 dark:text-dark-200">
                {{ importFileName || t('admin.modelPricing.overrides.importSelectFile') }}
              </div>
              <div class="text-caption text-gray-500 dark:text-dark-400">
                {{ t('admin.modelPricing.overrides.importFileType') }}
              </div>
            </div>
            <button
              type="button"
              class="btn btn-secondary shrink-0"
              :disabled="importing"
              @click="openImportFilePicker"
            >
              {{ t('common.chooseFile') }}
            </button>
          </div>
          <input
            ref="importFileInput"
            type="file"
            class="hidden"
            accept="application/json,.json"
            :aria-label="t('admin.modelPricing.overrides.importFile')"
            @change="handleImportFileChange"
          />
        </div>

        <fieldset>
          <legend class="input-label">{{ t('admin.modelPricing.overrides.importMode') }}</legend>
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label
              v-for="option in importModeOptions"
              :key="option.value"
              class="flex cursor-pointer items-start gap-3 rounded-lg border p-3 transition-colors duration-150"
              :class="
                importMode === option.value
                  ? 'border-primary-300 bg-primary-50/60 dark:border-primary-500/40 dark:bg-primary-500/10'
                  : 'border-gray-200 hover:border-gray-300 dark:border-dark-700 dark:hover:border-dark-600'
              "
            >
              <input
                v-model="importMode"
                type="radio"
                name="override-import-mode"
                class="radio mt-0.5"
                :value="option.value"
                :disabled="importing"
              />
              <span class="min-w-0">
                <span class="block text-control font-medium text-gray-900 dark:text-dark-300">
                  {{ option.label }}
                </span>
                <span class="mt-0.5 block text-caption text-gray-500 dark:text-dark-400">
                  {{ option.hint }}
                </span>
              </span>
            </label>
          </div>
        </fieldset>

        <p class="text-caption text-amber-700 dark:text-amber-400">
          {{ t('admin.modelPricing.overrides.importWarning') }}
        </p>
        <p v-if="importError" role="alert" class="text-caption text-red-600 dark:text-red-400">
          {{ importError }}
        </p>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" :disabled="importing" @click="closeImportDialog">
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            form="import-overrides-form"
            class="btn btn-primary"
            :disabled="importing || !importFile"
          >
            {{
              importing
                ? t('admin.modelPricing.overrides.importImporting')
                : t('admin.modelPricing.overrides.importButton')
            }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!--
      Global default unit prices. Non-token prices live in one global override
      file keyed by field name; a blank input means "not tuned" (the price-card
      value applies) and is sent as `null`, never as 0.
    -->
    <BaseDialog
      :show="defaultsDialogOpen"
      :title="t('admin.modelPricing.defaults.title')"
      width="wide"
      @close="closeDefaultsDialog"
    >
      <form id="defaults-pricing-form" class="space-y-5" @submit.prevent="submitDefaults">
        <p class="text-control text-gray-700 dark:text-dark-300">
          {{ t('admin.modelPricing.defaults.description') }}
        </p>
        <p class="text-caption text-gray-500 dark:text-dark-400">
          {{ t('admin.modelPricing.defaults.unitHint') }}
        </p>

        <div
          v-if="defaultsLoading"
          class="surface-flat flex items-center gap-2 p-3 text-caption text-gray-500 dark:text-dark-400"
        >
          <Icon name="refresh" size="sm" class="animate-spin" />
          {{ t('common.loading') }}
        </div>

        <div
          v-else-if="defaultsError"
          role="alert"
          class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-caption text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-400"
        >
          <span>{{ t('admin.modelPricing.defaults.loadFailed') }}：{{ defaultsError }}</span>
          <button type="button" class="btn btn-sm btn-secondary" @click="loadDefaults()">
            {{ t('admin.modelPricing.defaults.loadRetry') }}
          </button>
        </div>

        <template v-else>
          <p v-if="defaultsTunedCount === 0" class="text-caption text-gray-500 dark:text-dark-400">
            {{ t('admin.modelPricing.defaults.noneTuned') }}
          </p>

          <section v-for="group in DEFAULT_FIELD_GROUPS" :key="group.key" class="space-y-2">
            <h4
              class="border-b border-gray-200 pb-1.5 text-control font-medium text-gray-900 dark:border-dark-700 dark:text-white"
            >
              {{ t(group.titleKey) }}
            </h4>
            <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <div v-for="spec in group.specs" :key="spec.key">
                <label
                  class="input-label flex items-center justify-between gap-2"
                  :for="`defaults-field-${spec.key}`"
                >
                  <span class="truncate">{{ t(spec.titleKey) }}</span>
                  <span class="shrink-0 text-caption font-normal text-gray-500 dark:text-dark-400">
                    {{ t(spec.unitKey) }}
                  </span>
                </label>
                <div class="flex items-center gap-2">
                  <input
                    :id="`defaults-field-${spec.key}`"
                    v-model="defaultsForm[spec.key]"
                    type="text"
                    inputmode="decimal"
                    autocomplete="off"
                    class="input h-control min-w-0 flex-1 tabular-nums text-control"
                    :placeholder="t('admin.modelPricing.defaults.fieldPlaceholder')"
                    :disabled="defaultsBusy"
                    @input="markDefaultDirty(spec.key)"
                  />
                  <span
                    v-if="isDefaultTuned(spec.key) && !defaultsClearedKeys.has(spec.key)"
                    class="tag tag-primary shrink-0"
                  >
                    <Icon name="edit" size="xs" />
                    {{ t('admin.modelPricing.defaults.tunedBadge') }}
                  </span>
                  <button
                    v-if="isDefaultTuned(spec.key)"
                    type="button"
                    class="btn btn-sm btn-ghost shrink-0"
                    :disabled="defaultsBusy || defaultsClearedKeys.has(spec.key)"
                    :title="t('admin.modelPricing.defaults.clearFieldTitle', { field: t(spec.titleKey) })"
                    :aria-label="t('admin.modelPricing.defaults.clearFieldTitle', { field: t(spec.titleKey) })"
                    @click="clearDefaultField(spec.key)"
                  >
                    <Icon name="x" size="xs" />
                    <span class="hidden sm:inline">{{ t('admin.modelPricing.defaults.clearField') }}</span>
                  </button>
                </div>
                <p class="input-hint">
                  {{
                    isDefaultTuned(spec.key)
                      ? t('admin.modelPricing.defaults.clearFieldHint')
                      : t('admin.modelPricing.defaults.unitHint')
                  }}
                </p>
              </div>
            </div>
            <p v-if="group.key === 'image'" class="text-caption text-gray-500 dark:text-dark-400">
              {{ t('admin.modelPricing.defaults.imageHint') }}
            </p>
          </section>
        </template>

        <div
          v-if="defaultsClearConfirm"
          class="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-caption text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
        >
          <span>{{ t('admin.modelPricing.defaults.clearAllConfirm') }}</span>
          <span class="flex items-center gap-2">
            <button
              type="button"
              class="btn btn-sm btn-secondary"
              :disabled="defaultsBusy"
              @click="defaultsClearConfirm = false"
            >
              {{ t('common.cancel') }}
            </button>
            <button type="button" class="btn btn-sm btn-danger" :disabled="defaultsBusy" @click="submitClearAllDefaults">
              {{
                defaultsClearing
                  ? t('admin.modelPricing.defaults.clearing')
                  : t('admin.modelPricing.defaults.clearAllConfirmAction')
              }}
            </button>
          </span>
        </div>

        <p v-if="defaultsFormError" role="alert" class="text-caption text-red-600 dark:text-red-400">
          {{ defaultsFormError }}
        </p>
        <p v-if="defaultsPath" class="font-mono text-2xs text-gray-400 dark:text-dark-500">
          {{ t('admin.modelPricing.defaults.pathLabel', { path: defaultsPath }) }}
        </p>
      </form>

      <template #footer>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <button
            v-if="defaultsTunedCount > 0"
            type="button"
            class="btn btn-sm btn-danger-soft"
            :disabled="defaultsBusy || defaultsLoading"
            @click="defaultsClearConfirm = true"
          >
            {{ t('admin.modelPricing.defaults.clearAll') }}
          </button>
          <span v-else></span>
          <div class="flex items-center gap-2">
            <button type="button" class="btn btn-secondary" :disabled="defaultsBusy" @click="closeDefaultsDialog">
              {{ t('common.close') }}
            </button>
            <button
              type="submit"
              form="defaults-pricing-form"
              class="btn btn-primary"
              :disabled="defaultsBusy || defaultsLoading || defaultsError !== ''"
            >
              {{ defaultsSaving ? t('admin.modelPricing.defaults.saving') : t('admin.modelPricing.defaults.save') }}
            </button>
          </div>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'
import { saveAs } from 'file-saver'
import { adminAPI } from '@/api/admin'
import {
  PRICING_OVERRIDES_EXPORT_FILENAME,
  type ModelPricingCatalogItem,
  type ModelPricingCatalogTier,
  type ModelPricingOverrideEntry,
  type ModelPricingOverrideField,
  type ModelPricingOverrideImportMode,
  type ModelPricingOverrideValue
} from '@/api/admin/channels'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import { PageHeader } from '@/components/layout'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'

const { t, te } = useI18n()
const appStore = useAppStore()

/** Variable keys the billing expression prices, in canonical display order. */
const PRICED_VARIABLES = ['p', 'c', 'cr', 'cc', 'cc1h', 'img', 'img_cr'] as const
const MAX_VISIBLE_TIERS = 3
const SEARCH_DEBOUNCE_MS = 300

const items = ref<ModelPricingCatalogItem[]>([])
const loading = ref(false)
const loadError = ref('')
const providers = ref<string[]>([])
const providersLoading = ref(false)
const providersLoaded = ref(false)
const dialogItem = ref<ModelPricingCatalogItem | null>(null)

// ── Manual price overrides ──────────────────────────────────────────
/**
 * Unit boundary for tuning: the override file keeps **USD per token**
 * (1.5e-7) while this page shows **USD / 1M token** (0.15).
 * The two conversions live in `fillTuneForm` (×1e6) and `buildTuneFields` (÷1e6).
 */
const PER_MILLION_FACTOR = 1e6

type TuneFieldKey = 'input' | 'output' | 'cacheRead' | 'cacheWrite' | 'imageOutput'

type PerMillionKey =
  | 'input_price_per_million'
  | 'output_price_per_million'
  | 'cache_read_price_per_million'
  | 'cache_creation_price_per_million'

interface TuneFieldSpec {
  key: TuneFieldKey
  /** Key written to the override file (per-token, or per-image, value). */
  field: ModelPricingOverrideField
  /** Input units per stored unit: token fields pay per token but are shown per million (×1e6); per-image fields are ×1. */
  scale: number
  /** Column the dialog reads the price-card reference from (token fields only). */
  perMillionKey?: PerMillionKey
  /** Per-field unit label; the unit lives in the label, never only in a tooltip. */
  unitKey: string
  /** Optional extra hint under the field (e.g. the per-image cross-reference). */
  hintKey?: string
  /** i18n key for the field label; named `titleKey` so the locale checker sees it. */
  titleKey: string
}

const overrides = ref<Map<string, ModelPricingOverrideEntry>>(new Map())
const overridesPath = ref('')
const overridesLoading = ref(false)
const overridesError = ref('')

const tuneTarget = ref<ModelPricingCatalogItem | null>(null)
const tuneForm = reactive<Record<TuneFieldKey, string> & { expr: string }>({
  input: '',
  output: '',
  cacheRead: '',
  cacheWrite: '',
  imageOutput: '',
  expr: ''
})
const tuneOriginalExpr = ref<string | null>(null)
const tuneError = ref('')
const tuneSaving = ref(false)
const tuneRestoring = ref(false)
const restoreConfirm = ref(false)

const refreshDialogOpen = ref(false)
const refreshing = ref(false)

const importDialogOpen = ref(false)
const importFile = ref<File | null>(null)
const importFileInput = ref<HTMLInputElement | null>(null)
const importMode = ref<string>('merge')
const importError = ref('')
const importing = ref(false)

const exporting = ref(false)

// ── Global default unit prices (non-token) ─────────────────────────
/**
 * Non-token prices the price card has no per-model column for. Keys match the
 * override file exactly; the unit lives in each field's label. A blank input
 * means "not tuned" — it never means 0.
 */
type DefaultFieldGroup = 'image' | 'video' | 'search' | 'audio' | 'perRequest'

interface DefaultFieldSpec {
  key: string
  titleKey: string
  unitKey: string
}

interface DefaultFieldGroupSpec {
  key: DefaultFieldGroup
  titleKey: string
  specs: DefaultFieldSpec[]
}

const DEFAULT_FIELD_GROUPS: DefaultFieldGroupSpec[] = [
  {
    key: 'image',
    titleKey: 'admin.modelPricing.defaults.groups.image',
    specs: [
      {
        key: 'image_price_1k',
        titleKey: 'admin.modelPricing.defaults.fields.image_price_1k',
        unitKey: 'admin.modelPricing.defaults.units.perImage'
      },
      {
        key: 'image_price_2k',
        titleKey: 'admin.modelPricing.defaults.fields.image_price_2k',
        unitKey: 'admin.modelPricing.defaults.units.perImage'
      },
      {
        key: 'image_price_4k',
        titleKey: 'admin.modelPricing.defaults.fields.image_price_4k',
        unitKey: 'admin.modelPricing.defaults.units.perImage'
      }
    ]
  },
  {
    key: 'video',
    titleKey: 'admin.modelPricing.defaults.groups.video',
    specs: [
      {
        key: 'video_price_480p',
        titleKey: 'admin.modelPricing.defaults.fields.video_price_480p',
        unitKey: 'admin.modelPricing.defaults.units.perSecond'
      },
      {
        key: 'video_price_720p',
        titleKey: 'admin.modelPricing.defaults.fields.video_price_720p',
        unitKey: 'admin.modelPricing.defaults.units.perSecond'
      },
      {
        key: 'video_price_1080p',
        titleKey: 'admin.modelPricing.defaults.fields.video_price_1080p',
        unitKey: 'admin.modelPricing.defaults.units.perSecond'
      }
    ]
  },
  {
    key: 'search',
    titleKey: 'admin.modelPricing.defaults.groups.search',
    specs: [
      {
        key: 'web_search_price_per_call',
        titleKey: 'admin.modelPricing.defaults.fields.web_search_price_per_call',
        unitKey: 'admin.modelPricing.defaults.units.perCall'
      },
      {
        key: 'search_price_per_1k',
        titleKey: 'admin.modelPricing.defaults.fields.search_price_per_1k',
        unitKey: 'admin.modelPricing.defaults.units.perKiloCall'
      }
    ]
  },
  {
    key: 'audio',
    titleKey: 'admin.modelPricing.defaults.groups.audio',
    specs: [
      {
        key: 'audio_realtime_price_per_min',
        titleKey: 'admin.modelPricing.defaults.fields.audio_realtime_price_per_min',
        unitKey: 'admin.modelPricing.defaults.units.perMinute'
      },
      {
        key: 'audio_tts_price_per_million_chars',
        titleKey: 'admin.modelPricing.defaults.fields.audio_tts_price_per_million_chars',
        unitKey: 'admin.modelPricing.defaults.units.perMillionChars'
      },
      {
        key: 'audio_stt_price_per_hour',
        titleKey: 'admin.modelPricing.defaults.fields.audio_stt_price_per_hour',
        unitKey: 'admin.modelPricing.defaults.units.perHour'
      }
    ]
  },
  {
    key: 'perRequest',
    titleKey: 'admin.modelPricing.defaults.groups.perRequest',
    specs: [
      {
        key: 'per_request_price',
        titleKey: 'admin.modelPricing.defaults.fields.per_request_price',
        unitKey: 'admin.modelPricing.defaults.units.perCall'
      }
    ]
  }
]

/** Every default-field spec, flattened — the form and the PUT body walk this. */
const DEFAULT_FIELD_SPECS: DefaultFieldSpec[] = DEFAULT_FIELD_GROUPS.flatMap((group) => group.specs)

const defaults = ref<Map<string, number>>(new Map())
const defaultsPath = ref('')
const defaultsLoading = ref(false)
const defaultsError = ref('')
const defaultsDialogOpen = ref(false)
const defaultsSaving = ref(false)
const defaultsClearing = ref(false)
const defaultsClearConfirm = ref(false)
const defaultsFormError = ref('')
const defaultsClearedKeys = ref<Set<string>>(new Set())
const defaultsForm = reactive<Record<string, string>>(
  Object.fromEntries(DEFAULT_FIELD_SPECS.map((spec) => [spec.key, '']))
)

const defaultsBusy = computed(() => defaultsSaving.value || defaultsClearing.value)
const defaultsTunedCount = computed(() => defaults.value.size)

function isDefaultTuned(key: string): boolean {
  return defaults.value.has(key)
}

/** Preview of the backend value, in a form the input accepts (1e-7 stays valid). */
function numberToInput(value: number): string {
  if (!Number.isFinite(value)) return ''
  if (value === 0) return '0'
  return String(value)
}

function fillDefaultsForm() {
  defaultsClearedKeys.value = new Set()
  for (const spec of DEFAULT_FIELD_SPECS) {
    const value = defaults.value.get(spec.key)
    defaultsForm[spec.key] = value === undefined ? '' : numberToInput(value)
  }
}

async function loadDefaults() {
  defaultsLoading.value = true
  defaultsError.value = ''
  try {
    const response = await adminAPI.channels.getPricingDefaults()
    const next = new Map<string, number>()
    for (const [key, value] of Object.entries(response.fields ?? {})) {
      if (typeof value === 'number' && Number.isFinite(value)) next.set(key, value)
    }
    defaults.value = next
    defaultsPath.value = response.path ?? ''
    fillDefaultsForm()
  } catch (error) {
    defaultsError.value = extractApiErrorMessage(error, t('admin.modelPricing.defaults.loadFailed'))
  } finally {
    defaultsLoading.value = false
  }
}

function openDefaultsDialog() {
  defaultsDialogOpen.value = true
  defaultsClearConfirm.value = false
  defaultsFormError.value = ''
  void loadDefaults()
}

function closeDefaultsDialog() {
  if (defaultsBusy.value) return
  defaultsDialogOpen.value = false
  defaultsClearConfirm.value = false
  defaultsFormError.value = ''
}

/** Blanking a tuned field asks the backend to drop the key (`null`). */
function clearDefaultField(key: string) {
  defaultsForm[key] = ''
  defaultsClearedKeys.value.add(key)
}

function markDefaultDirty(key: string) {
  if (defaultsClearedKeys.value.has(key)) defaultsClearedKeys.value.delete(key)
  defaultsFormError.value = ''
}

/**
 * The only place the defaults PUT body is built. A blank tuned field becomes
 * `null` (untune); a blank untuned field is omitted. Non-numbers are rejected
 * before the request so the admin sees the field name, not a raw 400.
 */
function buildDefaultsFields():
  | { fields: Record<string, number | null> }
  | { error: string } {
  const fields: Record<string, number | null> = {}

  for (const spec of DEFAULT_FIELD_SPECS) {
    const raw = (defaultsForm[spec.key] ?? '').trim()
    if (raw === '') {
      if (defaults.value.has(spec.key)) fields[spec.key] = null
      continue
    }
    const value = Number(raw)
    if (!Number.isFinite(value) || value < 0) {
      return {
        error: t('admin.modelPricing.defaults.invalidNumber', { field: t(spec.titleKey) })
      }
    }
    fields[spec.key] = value
  }

  // An empty body is a no-op; a body of only `null`s does untune, which is the
  // same effect as the block-level delete, so it is allowed here.
  if (Object.keys(fields).length === 0) {
    return { error: t('admin.modelPricing.defaults.needOneValue') }
  }
  return { fields }
}

async function submitDefaults() {
  if (defaultsBusy.value || defaultsLoading.value) return
  const built = buildDefaultsFields()
  if ('error' in built) {
    defaultsFormError.value = built.error
    return
  }

  defaultsFormError.value = ''
  defaultsSaving.value = true
  try {
    await adminAPI.channels.putPricingDefaults(built.fields)
    appStore.showSuccess(t('admin.modelPricing.defaults.saved'))
    // Re-fetch so the 「已微调」 marks reflect what the backend actually stored.
    await loadDefaults()
  } catch (error) {
    defaultsFormError.value = extractApiErrorMessage(error, t('admin.modelPricing.defaults.saveFailed'))
  } finally {
    defaultsSaving.value = false
  }
}

async function submitClearAllDefaults() {
  if (defaultsBusy.value) return
  defaultsFormError.value = ''
  defaultsClearing.value = true
  try {
    await adminAPI.channels.deletePricingDefaults()
    appStore.showSuccess(t('admin.modelPricing.defaults.clearAllDone'))
    defaultsClearConfirm.value = false
    await loadDefaults()
  } catch (error) {
    defaultsFormError.value = extractApiErrorMessage(error, t('admin.modelPricing.defaults.clearAllFailed'))
  } finally {
    defaultsClearing.value = false
  }
}

const filters = reactive({
  q: '',
  provider: '',
  source: '',
  timeDependent: false,
  hasExpr: false
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0
})

let abortController: AbortController | null = null
let searchTimer: ReturnType<typeof setTimeout> | null = null

// ── Price formatting ────────────────────────────────────────────────
// Prices span many orders of magnitude — 0.003 cache reads sit next to 15.0
// outputs — so a fixed 2-decimal format would print 0.003 as "0.00".
// Rule: 4 significant digits (at most 8 decimals), never fewer than 2 decimals,
// and trailing zeros beyond those two are trimmed.
function formatMoney(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '-'
  if (value === 0) return '0'
  const abs = Math.abs(value)
  const significantDecimals = 3 - Math.floor(Math.log10(abs))
  const decimals = Math.min(8, Math.max(2, significantDecimals))
  const [integerPart, fractionPart = ''] = value.toFixed(decimals).split('.')
  const trimmedFraction = fractionPart.replace(/0+$/, '')
  const fraction = trimmedFraction.length >= 2 ? trimmedFraction : fractionPart.slice(0, 2)
  return `${integerPart}.${fraction}`
}

function formatUsd(value: number | null | undefined): string {
  const text = formatMoney(value)
  return text === '-' ? text : `$${text}`
}

function priceCell(
  row: ModelPricingCatalogItem,
  key:
    | 'input_price_per_million'
    | 'output_price_per_million'
    | 'cache_read_price_per_million'
    | 'cache_creation_price_per_million'
): string {
  if (row.token_pricing_absent) return '-'
  return formatUsd(row[key])
}

// ── Billing column ──────────────────────────────────────────────────
function variableLabel(key: string): string {
  if ((PRICED_VARIABLES as readonly string[]).includes(key)) {
    return t(`admin.modelPricing.billing.variables.${key}`)
  }
  return key
}

function tierLabel(name: string | undefined): string {
  if (!name) return t('admin.modelPricing.billing.tierNameUnknown')
  const key = `admin.modelPricing.billing.tiers.${name}`
  return te(key) ? t(key) : name
}

/** Human-readable time windows win over the raw condition; both are shown in the dialog. */
function tierWindowText(tier: ModelPricingCatalogTier): string {
  return (tier.time_windows ?? []).map((window) => window.trim()).filter(Boolean).join('；')
}

function tierConditionText(tier: ModelPricingCatalogTier): string {
  return tierWindowText(tier) || (tier.condition ?? '').trim()
}

function tierKey(tier: ModelPricingCatalogTier): string {
  return `${tier.name}-${tier.condition ?? ''}-${tier.unit}`
}

interface TierSegment {
  label: string
  value: string
}

/** Coefficients keyed by priced variable plus the per-request constant, ready to render. */
function tierChargeSegments(tier: ModelPricingCatalogTier): TierSegment[] {
  const coefficients = tier.coefficients ?? {}
  const keys = (tier.variables && tier.variables.length
    ? tier.variables
    : Object.keys(coefficients)
  ).filter((key) => coefficients[key] !== undefined)

  const segments: TierSegment[] = keys.map((key) => ({
    label: variableLabel(key),
    value: formatUsd(coefficients[key])
  }))

  if (tier.constant) {
    segments.push({
      label: t('admin.modelPricing.billing.perRequest'),
      value: `${formatUsd(tier.constant)} ${t('admin.modelPricing.billing.perRequestSuffix').trim()}`
    })
  }

  return segments
}

function unitLabel(unit: string): string {
  switch (unit) {
    case 'per_request':
      return t('admin.modelPricing.billing.unitPerRequest')
    case 'per_million_tokens_plus_request':
      return t('admin.modelPricing.billing.unitMixed')
    default:
      return t('admin.modelPricing.billing.unitPerMillionTokens')
  }
}

function baselineCells(row: ModelPricingCatalogItem): TierSegment[] {
  return [
    { label: t('admin.modelPricing.columns.input'), value: formatUsd(row.input_price_per_million) },
    { label: t('admin.modelPricing.columns.output'), value: formatUsd(row.output_price_per_million) },
    { label: t('admin.modelPricing.columns.cacheRead'), value: formatUsd(row.cache_read_price_per_million) },
    { label: t('admin.modelPricing.columns.cacheWrite'), value: formatUsd(row.cache_creation_price_per_million) }
  ]
}

function visibleTiers(row: ModelPricingCatalogItem): ModelPricingCatalogTier[] {
  return (row.tiers ?? []).slice(0, MAX_VISIBLE_TIERS)
}

function hiddenTierCount(row: ModelPricingCatalogItem): number {
  return Math.max(0, (row.tiers?.length ?? 0) - MAX_VISIBLE_TIERS)
}

function sourceLabel(source: string): string {
  return source === 'builtin'
    ? t('admin.modelPricing.sourceBuiltin')
    : t('admin.modelPricing.sourceCatalog')
}

function exprSourceLabel(source: string | undefined): string {
  return source === 'builtin'
    ? t('admin.modelPricing.sourceBuiltin')
    : t('admin.modelPricing.sourceCatalog')
}

// ── Filters ─────────────────────────────────────────────────────────
const providerOptions = computed(() => [
  { value: '', label: t('admin.modelPricing.allProviders') },
  ...providers.value.map((provider) => ({ value: provider, label: provider }))
])

const sourceOptions = computed(() => [
  { value: '', label: t('admin.modelPricing.allSources') },
  { value: 'catalog', label: t('admin.modelPricing.sourceCatalog') },
  { value: 'builtin', label: t('admin.modelPricing.sourceBuiltin') }
])

const hasActiveFilters = computed(
  () =>
    filters.q.trim() !== '' ||
    filters.provider !== '' ||
    filters.source !== '' ||
    filters.timeDependent ||
    filters.hasExpr
)

const columns = computed<Column[]>(() => [
  { key: 'model', label: t('admin.modelPricing.columns.model') },
  { key: 'provider', label: t('admin.modelPricing.columns.provider') },
  { key: 'source', label: t('admin.modelPricing.columns.source') },
  { key: 'input', label: t('admin.modelPricing.columns.input'), class: 'text-right' },
  { key: 'output', label: t('admin.modelPricing.columns.output'), class: 'text-right' },
  { key: 'cache_read', label: t('admin.modelPricing.columns.cacheRead'), class: 'text-right' },
  { key: 'cache_write', label: t('admin.modelPricing.columns.cacheWrite'), class: 'text-right' },
  { key: 'billing', label: t('admin.modelPricing.columns.billing') },
  { key: 'flags', label: t('admin.modelPricing.columns.flags') },
  { key: 'actions', label: t('admin.modelPricing.columns.actions'), class: 'text-right' }
])

// ── Data loading ────────────────────────────────────────────────────
function buildQuery(page: number, pageSize: number) {
  return {
    q: filters.q.trim() || undefined,
    provider: filters.provider || undefined,
    source: filters.source || undefined,
    time_dependent: filters.timeDependent || undefined,
    has_expr: filters.hasExpr || undefined,
    page,
    page_size: pageSize
  }
}

async function loadCatalog(options: { notifySuccess?: boolean; allowPageReset?: boolean } = {}) {
  abortController?.abort()
  const controller = new AbortController()
  abortController = controller
  loading.value = true
  loadError.value = ''

  try {
    const response = await adminAPI.channels.listModelPricingCatalog(
      buildQuery(pagination.page, pagination.page_size),
      { signal: controller.signal }
    )
    if (controller.signal.aborted || abortController !== controller) return

    items.value = response.items ?? []
    pagination.total = response.total ?? 0

    // A filter change can leave the cursor past the last page; the backend clamps
    // the page but still returns an empty slice, so step back to page 1 once.
    if (items.value.length === 0 && pagination.total > 0 && pagination.page > 1 && options.allowPageReset) {
      pagination.page = 1
      await loadCatalog({ allowPageReset: false })
      return
    }

    if (options.notifySuccess) appStore.showSuccess(t('admin.modelPricing.refreshed'))
  } catch (error: unknown) {
    const e = error as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    loadError.value = extractApiErrorMessage(error, t('admin.modelPricing.loadError'))
  } finally {
    if (abortController === controller) {
      loading.value = false
      abortController = null
    }
  }
}

/**
 * Provider list is derived from the catalog itself (page_size=0 returns every row),
 * so the dropdown offers exactly the providers the backend can filter on.
 */
async function loadProviders() {
  if (providersLoaded.value || providersLoading.value) return
  providersLoading.value = true
  try {
    const response = await adminAPI.channels.listModelPricingCatalog({ page: 1, page_size: 0 })
    const unique = new Set<string>()
    for (const item of response.items ?? []) {
      if (item.provider) unique.add(item.provider)
    }
    providers.value = [...unique].sort((a, b) => a.localeCompare(b))
    providersLoaded.value = true
  } catch (error) {
    // The dropdown is a convenience; the table already reported the failure.
    console.warn('Failed to load model pricing providers:', error)
  } finally {
    providersLoading.value = false
  }
}

// ── Interactions ────────────────────────────────────────────────────
function applyFilters() {
  pagination.page = 1
  void loadCatalog({ allowPageReset: true })
}

function handleSearchInput() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    searchTimer = null
    applyFilters()
  }, SEARCH_DEBOUNCE_MS)
}

function setTimeDependent(value: boolean) {
  filters.timeDependent = value
  applyFilters()
}

function setHasExpr(value: boolean) {
  filters.hasExpr = value
  applyFilters()
}

function clearFilters() {
  filters.q = ''
  filters.provider = ''
  filters.source = ''
  filters.timeDependent = false
  filters.hasExpr = false
  applyFilters()
}

function handleRefresh() {
  void loadCatalog({ notifySuccess: true, allowPageReset: true })
  void loadProviders()
  void loadOverrides()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadCatalog({ allowPageReset: false })
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadCatalog({ allowPageReset: false })
}

function openExprDialog(row: ModelPricingCatalogItem) {
  dialogItem.value = row
}

function closeExprDialog() {
  dialogItem.value = null
}


async function loadOverrides() {
  overridesLoading.value = true
  overridesError.value = ''
  try {
    const response = await adminAPI.channels.listPricingOverrides()
    const next = new Map<string, ModelPricingOverrideEntry>()
    for (const entry of response.items ?? []) next.set(entry.model, entry)
    overrides.value = next
    overridesPath.value = response.path ?? ''
  } catch (error) {
    // The catalog is still usable, so a failed override list is reported inline
    // instead of replacing the table with an error state.
    overridesError.value = extractApiErrorMessage(error, t('admin.modelPricing.overrides.loadError'))
  } finally {
    overridesLoading.value = false
  }
}

/** A write to the override file changes both the tuned badges and the effective prices. */
async function reloadPricingData() {
  await Promise.all([loadOverrides(), loadCatalog({ allowPageReset: true })])
}

// ── Tuning form ⇄ override file ─────────────────────────────────────
/** The rates this dialog tunes, in the order the dialog shows them. */
const tuneFieldSpecs: TuneFieldSpec[] = [
  {
    key: 'input',
    field: 'input_cost_per_token',
    scale: PER_MILLION_FACTOR,
    perMillionKey: 'input_price_per_million',
    unitKey: 'admin.modelPricing.overrides.units.perMillionTokens',
    titleKey: 'admin.modelPricing.overrides.fields.input'
  },
  {
    key: 'output',
    field: 'output_cost_per_token',
    scale: PER_MILLION_FACTOR,
    perMillionKey: 'output_price_per_million',
    unitKey: 'admin.modelPricing.overrides.units.perMillionTokens',
    titleKey: 'admin.modelPricing.overrides.fields.output'
  },
  {
    key: 'cacheRead',
    field: 'cache_read_input_token_cost',
    scale: PER_MILLION_FACTOR,
    perMillionKey: 'cache_read_price_per_million',
    unitKey: 'admin.modelPricing.overrides.units.perMillionTokens',
    titleKey: 'admin.modelPricing.overrides.fields.cacheRead'
  },
  {
    key: 'cacheWrite',
    field: 'cache_creation_input_token_cost',
    scale: PER_MILLION_FACTOR,
    perMillionKey: 'cache_creation_price_per_million',
    unitKey: 'admin.modelPricing.overrides.units.perMillionTokens',
    titleKey: 'admin.modelPricing.overrides.fields.cacheWrite'
  },
  {
    // The catalog field the image parser actually reads (USD per image, no scaling).
    key: 'imageOutput',
    field: 'output_cost_per_image',
    scale: 1,
    unitKey: 'admin.modelPricing.overrides.units.perImage',
    hintKey: 'admin.modelPricing.overrides.imageHint',
    titleKey: 'admin.modelPricing.overrides.fields.imageOutput'
  }
]

function overrideFor(model: string): ModelPricingOverrideEntry | null {
  return overrides.value.get(model) ?? null
}

/**
 * Only the manual flag earns the badge: entries the price-card refresh wrote are
 * not something the admin chose.
 */
function isTuned(model: string): boolean {
  return overrideFor(model)?.tuned === true
}

const overridesCount = computed(() => overrides.value.size)
const tunedOverrideCount = computed(
  () => [...overrides.value.values()].filter((entry) => entry.tuned).length
)

const tuneEntry = computed(() => (tuneTarget.value ? overrideFor(tuneTarget.value.model) : null))
const tuneTitle = computed(() =>
  tuneTarget.value
    ? t('admin.modelPricing.overrides.tuneTitle', { model: tuneTarget.value.model })
    : ''
)
const tuneBusy = computed(() => tuneSaving.value || tuneRestoring.value)
const tuneExprDirty = computed(() => tuneForm.expr.trim() !== (tuneOriginalExpr.value ?? ''))
const tuneExprCleared = computed(() => tuneExprDirty.value && tuneForm.expr.trim() === '')
const tuneExprPresent = computed(() => tuneForm.expr.trim() !== '')

/**
 * Stored value → dialog input. Token fields store USD per token while the page
 * shows USD per 1M tokens (×1e6); unit prices such as per-image stay as-is (×1).
 * Values live around 1e-7, so `toFixed(12)` keeps `1.5e-7` readable as `0.15`
 * instead of leaking scientific notation; trailing zeros are trimmed afterwards.
 */
function storedToInput(value: ModelPricingOverrideValue | undefined, scale: number): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return ''
  const shown = value * scale
  if (shown === 0) return '0'
  return shown.toFixed(12).replace(/\.?0+$/, '') || '0'
}

/**
 * The whole per-million → per-token conversion, and the only place the PUT body
 * is built. A blank input means "follow the price card": the field is dropped
 * (sent as `null`) only when the override file currently carries it.
 */
function buildTuneFields(): { fields: Record<string, ModelPricingOverrideValue> } | { error: string } {
  const target = tuneTarget.value
  if (!target) return { error: t('admin.modelPricing.overrides.saveFailed') }

  const entry = overrideFor(target.model)
  const fields: Record<string, ModelPricingOverrideValue> = {}
  let filledPrices = 0

  for (const spec of tuneFieldSpecs) {
    const raw = tuneForm[spec.key].trim()
    if (raw === '') {
      const existing = entry?.fields?.[spec.field]
      if (existing !== undefined && existing !== null) fields[spec.field] = null
      continue
    }
    const amount = Number(raw)
    if (!Number.isFinite(amount) || amount < 0) {
      return { error: t('admin.modelPricing.overrides.invalidNumber', { field: t(spec.titleKey) }) }
    }
    // The dialog shows per-million (tokens) or per-image; the file stores per-token / per-image.
    fields[spec.field] = amount / spec.scale
    filledPrices += 1
  }

  if (tuneExprDirty.value) {
    const expr = tuneForm.expr.trim()
    fields.billing_expr = expr === '' ? null : expr
  }

  const values = Object.values(fields)
  if (values.length === 0) return { error: t('admin.modelPricing.overrides.needOneField') }
  if (filledPrices === 0 && values.every((value) => value === null)) {
    // Every change is a deletion — that is what 「恢复价卡价」 is for.
    return { error: t('admin.modelPricing.overrides.emptyAll') }
  }
  return { fields }
}

function openTuneDialog(row: ModelPricingCatalogItem) {
  const entry = overrideFor(row.model)
  for (const spec of tuneFieldSpecs) {
    tuneForm[spec.key] = storedToInput(entry?.fields?.[spec.field], spec.scale)
  }
  const expr = typeof entry?.fields?.billing_expr === 'string' ? entry.fields.billing_expr : ''
  tuneForm.expr = expr
  tuneOriginalExpr.value = expr === '' ? null : expr
  tuneError.value = ''
  restoreConfirm.value = false
  tuneTarget.value = row
}

function closeTuneDialog() {
  if (tuneBusy.value) return
  tuneTarget.value = null
  tuneOriginalExpr.value = null
  tuneError.value = ''
  restoreConfirm.value = false
}

function clearTuneExpr() {
  tuneForm.expr = ''
}

function restoreTuneExpr() {
  tuneForm.expr = tuneOriginalExpr.value ?? ''
}

async function submitTune() {
  const target = tuneTarget.value
  if (!target || tuneBusy.value) return
  const built = buildTuneFields()
  if ('error' in built) {
    tuneError.value = built.error
    return
  }

  tuneError.value = ''
  tuneSaving.value = true
  try {
    await adminAPI.channels.putPricingOverride(target.model, built.fields)
    appStore.showSuccess(t('admin.modelPricing.overrides.saved', { model: target.model }))
    tuneTarget.value = null
    await reloadPricingData()
  } catch (error) {
    tuneError.value = extractApiErrorMessage(error, t('admin.modelPricing.overrides.saveFailed'))
  } finally {
    tuneSaving.value = false
  }
}

async function submitRestore() {
  const target = tuneTarget.value
  if (!target || tuneBusy.value) return

  tuneError.value = ''
  tuneRestoring.value = true
  try {
    await adminAPI.channels.deletePricingOverride(target.model)
    appStore.showSuccess(t('admin.modelPricing.overrides.restored', { model: target.model }))
    tuneTarget.value = null
    restoreConfirm.value = false
    await reloadPricingData()
  } catch (error) {
    tuneError.value = extractApiErrorMessage(error, t('admin.modelPricing.overrides.restoreFailed'))
  } finally {
    tuneRestoring.value = false
  }
}

// ── Latest-price fetch, import and export ───────────────────────────
function openRefreshDialog() {
  refreshDialogOpen.value = true
}

function closeRefreshDialog() {
  if (refreshing.value) return
  refreshDialogOpen.value = false
}

async function runRefresh(overwriteTuned: boolean) {
  if (refreshing.value) return
  refreshing.value = true
  try {
    const result = await adminAPI.channels.refreshPricingOverrides(overwriteTuned)
    appStore.showSuccess(
      t('admin.modelPricing.overrides.refreshResult', {
        added: result.added,
        updated: result.updated,
        kept: result.kept,
        total: result.total
      })
    )
    refreshDialogOpen.value = false
    await reloadPricingData()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelPricing.overrides.refreshFailed')))
  } finally {
    refreshing.value = false
  }
}

const importModeOptions = computed(() => [
  {
    value: 'replace',
    label: t('admin.modelPricing.overrides.importModeReplace'),
    hint: t('admin.modelPricing.overrides.importModeReplaceHint')
  },
  {
    value: 'merge',
    label: t('admin.modelPricing.overrides.importModeMerge'),
    hint: t('admin.modelPricing.overrides.importModeMergeHint')
  }
])

const importFileName = computed(() => importFile.value?.name ?? '')

function openImportDialog() {
  importError.value = ''
  importFile.value = null
  importMode.value = 'merge'
  if (importFileInput.value) importFileInput.value.value = ''
  importDialogOpen.value = true
}

function closeImportDialog() {
  if (importing.value) return
  importDialogOpen.value = false
}

function openImportFilePicker() {
  importFileInput.value?.click()
}

function handleImportFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  importFile.value = target.files?.[0] ?? null
  importError.value = ''
}

async function readFileText(file: File): Promise<string> {
  if (typeof file.text === 'function') return file.text()
  if (typeof file.arrayBuffer === 'function') {
    return new TextDecoder().decode(await file.arrayBuffer())
  }
  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error ?? new Error(t('common.fileReadFailed')))
    reader.readAsText(file)
  })
}

async function submitImport() {
  const file = importFile.value
  if (!file || importing.value) return

  importing.value = true
  importError.value = ''
  try {
    const text = await readFileText(file)
    let payload: unknown
    try {
      payload = JSON.parse(text)
    } catch {
      importError.value = t('admin.modelPricing.overrides.importParseFailed')
      return
    }
    if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
      importError.value = t('admin.modelPricing.overrides.importShapeInvalid')
      return
    }

    const mode = importMode.value as ModelPricingOverrideImportMode
    const result = await adminAPI.channels.importPricingOverrides(
      payload as Record<string, unknown>,
      mode
    )
    appStore.showSuccess(
      t('admin.modelPricing.overrides.importResult', {
        count: result.imported,
        mode: t(
          mode === 'replace'
            ? 'admin.modelPricing.overrides.importModeReplace'
            : 'admin.modelPricing.overrides.importModeMerge'
        )
      })
    )
    importDialogOpen.value = false
    await reloadPricingData()
  } catch (error) {
    importError.value = extractApiErrorMessage(error, t('admin.modelPricing.overrides.importFailed'))
  } finally {
    importing.value = false
  }
}

async function handleExportOverrides() {
  if (exporting.value) return
  exporting.value = true
  try {
    const blob = await adminAPI.channels.exportPricingOverrides()
    saveAs(blob, PRICING_OVERRIDES_EXPORT_FILENAME)
    appStore.showSuccess(t('admin.modelPricing.overrides.exportDone'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelPricing.overrides.exportFailed')))
  } finally {
    exporting.value = false
  }
}

const dialogTitle = computed(() =>
  dialogItem.value
    ? t('admin.modelPricing.billing.dialogTitle', { model: dialogItem.value.model })
    : ''
)

onMounted(() => {
  void loadCatalog({ allowPageReset: true })
  void loadOverrides()
  void loadProviders()
  void loadDefaults()
})

onUnmounted(() => {
  abortController?.abort()
  abortController = null
  if (searchTimer) {
    clearTimeout(searchTimer)
    searchTimer = null
  }
})
</script>
