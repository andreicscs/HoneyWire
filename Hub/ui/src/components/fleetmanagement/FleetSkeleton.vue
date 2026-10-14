<script setup lang="ts">
import { fleetSkeletonStyles } from './fleetmanagementStyles.ts'
</script>

<template>
    <div data-testid="fleet-skeleton" :class="fleetSkeletonStyles.grid">
        <div v-for="i in 4" :key="i" data-testid="skeleton-card" :class="fleetSkeletonStyles.card">
            <!-- Header -->
            <div class="px-4 py-3">
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2.5">
                        <div class="w-2.5 h-2.5 rounded-full bg-bg-inset animate-pulse"></div>
                        <div class="h-4 w-28" :class="fleetSkeletonStyles.pulse"></div>
                    </div>
                    <div class="h-6 w-6" :class="fleetSkeletonStyles.pulse"></div>
                </div>
            </div>
            <!-- Body -->
            <div class="flex-1 px-4 py-3 flex flex-col gap-4">
                <div class="grid grid-cols-2 gap-y-2 gap-x-4 text-sm">
                    <div v-for="side in ['left', 'right']" :key="side" class="flex items-center gap-1.5">
                        <div class="w-3.5 h-3.5" :class="fleetSkeletonStyles.pulse"></div>
                        <div class="h-3 w-20" :class="fleetSkeletonStyles.pulse"></div>
                    </div>
                </div>
                <div class="flex gap-1.5">
                    <div v-for="width in ['w-12', 'w-16', 'w-8']" :key="width" class="h-5" :class="[fleetSkeletonStyles.pulse, width, 'rounded-md']"></div>
                </div>
                <div class="mt-auto bg-bg-inset/30 border border-border-default/50 rounded-lg p-3">
                    <div class="flex items-center justify-between mb-2">
                        <div class="h-3 w-24" :class="fleetSkeletonStyles.pulse"></div>
                        <div class="h-3 w-16" :class="fleetSkeletonStyles.pulse"></div>
                    </div>
                    <div class="flex gap-1.5 mt-1">
                        <div class="h-6 w-20" :class="fleetSkeletonStyles.pulse"></div>
                        <div class="h-6 w-16" :class="fleetSkeletonStyles.pulse"></div>
                    </div>
                </div>
            </div>
            <!-- Footer -->
            <div class="border-t border-border-default/50 px-4 py-3">
                <div class="h-3 w-32 mx-auto" :class="fleetSkeletonStyles.pulse"></div>
            </div>
        </div>
    </div>
</template>
