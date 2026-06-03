<template>
    <div class="grid grid-cols-1 lg:grid-cols-3 2xl:grid-cols-4 gap-5 auto-rows-max">
        <div v-for="i in 4" :key="i" class="bg-bg-surface border border-border-default rounded-lg overflow-hidden shadow-sm flex flex-col min-h-[280px]">
            <!-- Header -->
            <div class="px-4 py-3">
                <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2.5">
                        <div class="w-2.5 h-2.5 rounded-full bg-bg-inset animate-pulse"></div>
                        <div class="h-4 w-28 bg-bg-inset rounded animate-pulse"></div>
                    </div>
                    <div class="h-6 w-6 bg-bg-inset rounded animate-pulse"></div>
                </div>
            </div>
            <!-- Body -->
            <div class="flex-1 px-4 py-3 flex flex-col gap-4">
                <div class="grid grid-cols-2 gap-y-2 gap-x-4 text-sm">
                    <div class="flex items-center gap-1.5">
                        <div class="w-3.5 h-3.5 bg-bg-inset rounded animate-pulse"></div>
                        <div class="h-3 w-20 bg-bg-inset rounded animate-pulse"></div>
                    </div>
                    <div class="flex items-center gap-1.5">
                        <div class="w-3.5 h-3.5 bg-bg-inset rounded animate-pulse"></div>
                        <div class="h-3 w-20 bg-bg-inset rounded animate-pulse"></div>
                    </div>
                </div>
                <div class="flex gap-1.5">
                    <div class="h-5 w-12 bg-bg-inset rounded-md animate-pulse"></div>
                    <div class="h-5 w-16 bg-bg-inset rounded-md animate-pulse"></div>
                    <div class="h-5 w-8 bg-bg-inset rounded-md animate-pulse"></div>
                </div>
                <div class="mt-auto bg-bg-inset/30 border border-border-default/50 rounded-lg p-3">
                    <div class="flex items-center justify-between mb-2">
                        <div class="h-3 w-24 bg-bg-inset rounded animate-pulse"></div>
                        <div class="h-3 w-16 bg-bg-inset rounded animate-pulse"></div>
                    </div>
                    <div class="flex gap-1.5 mt-1">
                        <div class="h-6 w-20 bg-bg-inset rounded-md animate-pulse"></div>
                        <div class="h-6 w-16 bg-bg-inset rounded-md animate-pulse"></div>
                    </div>
                </div>
            </div>
            <!-- Footer -->
            <div class="border-t border-border-default/50 px-4 py-3">
                <div class="h-3 w-32 mx-auto bg-bg-inset rounded animate-pulse"></div>
            </div>
        </div>
    </div>
</template>