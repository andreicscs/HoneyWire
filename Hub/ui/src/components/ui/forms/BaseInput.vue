<script setup>
import { ref, onMounted, nextTick } from 'vue'

const props = defineProps({
    modelValue: { type: [String, Number, Boolean], default: '' },
    label: { type: String, default: '' },
    description: { type: String, default: '' },
    type: { type: String, default: 'text' }, // text, number, boolean
    placeholder: { type: String, default: '' },
    required: { type: Boolean, default: false },
    disabled: { type: Boolean, default: false },
    defaultFallback: { type: [String, Number, Boolean], default: '' },
    autofocus: { type: Boolean, default: false }
})

const emit = defineEmits(['update:modelValue', 'focus', 'blur'])

const inputRef = ref(null)

onMounted(() => {
    if (props.autofocus) {
        nextTick(() => {
            if (inputRef.value) inputRef.value.focus()
        })
    }
})

const isOpen = ref(false)

const toggleDropdown = () => {
    if (!props.disabled) {
        isOpen.value = !isOpen.value
    }
}

const selectBoolean = (val) => {
    if (!props.disabled) {
        emit('update:modelValue', val)
        isOpen.value = false
    }
}

const increment = () => {
    if (props.disabled) return
    const current = props.modelValue !== undefined && props.modelValue !== '' ? props.modelValue : props.defaultFallback
    emit('update:modelValue', String(Number(current || 0) + 1))
}

const decrement = () => {
    if (props.disabled) return
    const current = props.modelValue !== undefined && props.modelValue !== '' ? props.modelValue : props.defaultFallback
    emit('update:modelValue', String(Number(current || 0) - 1))
}
</script>

<template>
    <div class="w-full relative">
        <label v-if="label" class="block text-sm text-text-h mb-0.5">
            {{ label }} <span v-if="required" class="text-danger-text">*</span>
        </label>
        
        <!-- Boolean Input (Dropdown) -->
        <div v-if="type === 'boolean'" class="relative w-full">
            <div @click="toggleDropdown"
                 @focus="$emit('focus', $event)"
                 @blur="$emit('blur', $event)"
                 ref="inputRef"
                 tabindex="0"
                 class="w-full px-3 py-2 text-sm bg-input-bg border rounded-md cursor-pointer flex justify-between items-center transition-all duration-[var(--duration-fast)] shadow-sm select-none outline-none config-input"
                 :class="disabled ? 'bg-disabled-bg border-disabled-border text-disabled-text cursor-not-allowed' : (isOpen ? 'border-primary-main ring-1 ring-focus-ring' : 'border-input-border hover:border-border-default focus:border-primary-main focus:ring-1 focus:ring-focus-ring')">
                <span v-if="String(modelValue !== undefined && modelValue !== '' ? modelValue : defaultFallback) === 'true'" class="text-success-main font-medium flex items-center gap-2">
                    <span class="w-2 h-2 rounded-full bg-success-main shrink-0"></span>true
                </span>
                <span v-else-if="String(modelValue !== undefined && modelValue !== '' ? modelValue : defaultFallback) === 'false'" class="text-danger-main font-medium flex items-center gap-2">
                    <span class="w-2 h-2 rounded-full bg-danger-main shrink-0"></span>false
                </span>
                <span v-else class="text-text-m font-medium flex items-center gap-2">
                    <span class="w-2 h-2 rounded-full bg-border-default shrink-0"></span>Select...
                </span>
                <svg class="w-4 h-4 text-text-m transition-transform duration-200 shrink-0" :class="isOpen ? 'rotate-180' : ''" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path></svg>
            </div>

            <div v-if="isOpen" @click="isOpen = false" class="fixed inset-0 z-[var(--z-elevated)]"></div>

            <transition enter-active-class="transition duration-100 ease-out" enter-from-class="transform scale-95 opacity-0" enter-to-class="transform scale-100 opacity-100" leave-active-class="transition duration-75 ease-in" leave-from-class="transform scale-100 opacity-100" leave-to-class="transform scale-95 opacity-0">
                <ul v-if="isOpen" class="absolute top-full left-0 z-[var(--z-dropdown)] w-full mt-1 bg-bg-surface border border-border-default rounded-md shadow-lg py-1 overflow-hidden">
                    <li @click="selectBoolean('true')" class="px-3 py-2 cursor-pointer transition-colors text-sm font-medium duration-[var(--duration-fast)] flex items-center gap-2 text-success-main hover:bg-success-bg">
                        <span class="w-2 h-2 rounded-full bg-success-main shrink-0"></span>true
                    </li>
                    <li @click="selectBoolean('false')" class="px-3 py-2 cursor-pointer transition-colors text-sm font-medium duration-[var(--duration-fast)] flex items-center gap-2 text-danger-main hover:bg-danger-bg">
                        <span class="w-2 h-2 rounded-full bg-danger-main shrink-0"></span>false
                    </li>
                </ul>
            </transition>
        </div>

        <!-- Number Input -->
        <div v-else-if="type === 'number'" class="relative w-full flex items-center">
            <input
                ref="inputRef"
                type="number"
                :value="modelValue"
                @input="$emit('update:modelValue', $event.target.value)"
                @focus="$emit('focus', $event)"
                @blur="$emit('blur', $event)"
                :placeholder="placeholder"
                :required="required"
                :disabled="disabled"
                class="w-full pl-3 pr-10 py-2 rounded-md text-sm text-text-h transition-colors duration-[var(--duration-fast)] shadow-sm outline-none disabled:cursor-not-allowed [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none config-input"
                :class="disabled
                    ? 'bg-disabled-bg border border-disabled-border text-disabled-text'
                    : 'bg-input-bg border border-input-border focus:border-primary-main focus:ring-1 focus:ring-focus-ring placeholder:text-text-m/50'"
            />
            <div v-if="!disabled" class="absolute right-1 top-1 bottom-1 flex flex-col border-l border-input-border w-7">
                <button type="button" tabindex="-1" @click.prevent="increment" class="flex-1 flex items-center justify-center text-text-m hover:text-text-h hover:bg-secondary-hover transition-colors rounded-tr-md outline-none border-b border-input-border"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M5 15l7-7 7 7"></path></svg></button>
                <button type="button" tabindex="-1" @click.prevent="decrement" class="flex-1 flex items-center justify-center text-text-m hover:text-text-h hover:bg-secondary-hover transition-colors rounded-br-md outline-none"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2.5"><path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7"></path></svg></button>
            </div>
        </div>

        <!-- Standard Text Inputs -->
        <input v-else
            ref="inputRef"
            :type="type"
            :value="modelValue"
            @input="$emit('update:modelValue', $event.target.value)"
            @focus="$emit('focus', $event)"
            @blur="$emit('blur', $event)"
            :placeholder="placeholder"
            :required="required"
            :disabled="disabled"
            class="w-full px-3 py-2 rounded-md text-sm text-text-h transition-colors duration-[var(--duration-fast)] shadow-sm outline-none disabled:cursor-not-allowed config-input"
            :class="disabled
                ? 'bg-disabled-bg border border-disabled-border text-disabled-text'
                : 'bg-input-bg border border-input-border focus:border-primary-main focus:ring-1 focus:ring-focus-ring placeholder:text-text-m/50'"
        />

        <p v-if="description" class="text-sm text-text-m mt-1">{{ description }}</p>
    </div>
</template>