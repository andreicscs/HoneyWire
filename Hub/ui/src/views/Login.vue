<script setup>
import { ref } from 'vue'

const emit = defineEmits(['login-success'])
const password = ref('')
const loading = ref(false)
const error = ref(false)

const doLogin = async () => {
    loading.value = true
    error.value = false
    
    try {
        const res = await fetch('/login', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({password: password.value})
        })
        
        if (res.ok) {
            emit('login-success')
        } else {
            error.value = true
            password.value = ''
        }
    } catch (err) {
        error.value = true
    } finally {
        loading.value = false
    }
}
</script>

<template>
    <div class="h-screen flex flex-col items-center justify-center bg-grid transition-colors duration-200 p-6">
        <div class="w-full max-w-[400px] space-y-8">
            
            <div class="flex flex-col items-center">
                <div class="w-12 h-12 flex items-center justify-center rounded-lg bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 shadow-sm text-2xl mb-4">🕸️</div>
                <h1 class="text-xl font-bold text-slate-900 dark:text-white tracking-tight">HoneyWire Sentinel</h1>
                <p class="text-sm text-slate-500 dark:text-zinc-500 mt-1">Authorized personnel only</p>
            </div>

            <div class="bg-white dark:bg-zinc-900 border border-slate-200 dark:border-zinc-800 rounded-lg shadow-sm p-8 transition-colors duration-200">
                <form @submit.prevent="doLogin" class="space-y-6">
                    <div class="space-y-2">
                        <label for="pwd" class="text-xs font-semibold text-slate-700 dark:text-zinc-400">Authentication Key</label>
                        <input type="password" id="pwd" v-model="password" placeholder="••••••••••••"
                            class="w-full px-3 py-2 rounded-md bg-slate-50 dark:bg-zinc-950 border border-slate-300 dark:border-zinc-800 text-sm mono text-slate-900 dark:text-zinc-200 focus:outline-none focus:ring-2 focus:ring-slate-400 dark:focus:ring-zinc-600 focus:border-transparent transition-all placeholder-slate-300 dark:placeholder-zinc-700" required>
                    </div>
                    
                    <button type="submit" :disabled="loading"
                            class="w-full py-2 rounded-md text-sm font-semibold transition-all shadow-sm"
                            :class="loading ? 'bg-slate-400 dark:bg-zinc-700 text-slate-100 cursor-not-allowed' : 'bg-slate-900 dark:bg-zinc-100 text-white dark:text-zinc-900 hover:bg-slate-800 dark:hover:bg-white'">
                        {{ loading ? 'Authenticating...' : 'Sign in' }}
                    </button>
                </form>

                <div v-if="error" class="mt-6 p-3 rounded-md bg-rose-50 dark:bg-rose-900/20 border border-rose-200 dark:border-rose-800/30 text-center">
                    <p class="text-xs font-medium text-rose-700 dark:text-rose-400">Access Denied: Invalid Key</p>
                </div>
            </div>
        </div>
    </div>
</template>