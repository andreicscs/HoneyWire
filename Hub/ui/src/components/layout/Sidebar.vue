<script setup>
import { storeToRefs } from 'pinia'
import { useAppStore } from '../../stores/System/app'
import { useEventsStore } from '../../stores/Events/events'
import { useFleetStore } from '../../stores/Fleet/fleet'

const appStore = useAppStore()
const eventsStore = useEventsStore()
const fleetStore = useFleetStore()

const { currentView, sidebarOpen, viewingArchive, version } = storeToRefs(appStore)

const clearLogs = async () => {
    try {
        const dryRes = await fetch('/api/v1/events?dryrun=true', { method: 'DELETE' })
        if (!dryRes.ok) throw new Error("Failed to fetch dryrun data")
        
        const dryData = await dryRes.json()
        const count = dryData.would_delete || 0

        if (count === 0) {
            alert("The database is already empty.")
            return
        }

        if (confirm(`Confirm Database Purge?\n\nThis will permanently delete ${count} active and archived event logs.\n\nThis action cannot be undone.`)) {
            const response = await fetch('/api/v1/events?dryrun=false', {
                method: 'DELETE',
                headers: { 'Content-Type': 'application/json' }
            })

            if (!response.ok) throw new Error(`Server error: ${response.status}`)
            
            eventsStore.purgeEvents()
            fleetStore.fetchFleet() // Refresh node & sensor stats (e.g., Event Volume 24h)
            
            alert("Database purged successfully.")
        }
    } catch (error) {
        console.error("Failed to purge logs:", error)
        alert("Error purging logs.")
    }
}

const goToDashboard = () => {
    fleetStore.clearSelection()
    appStore.setView('dashboard')
}
</script>

<template>
    <aside class="flex flex-col bg-bg-base border-r border-border-default --z-modal transition-all duration-fast ease-in-out shrink-0"
           :class="sidebarOpen ? 'w-[240px]' : 'w-[68px]'">
        
        <div class="h-14 flex items-center px-[18px] shrink-0 mb-2">
            <button @click="appStore.toggleSidebar()" 
                    type="button"
                    class="p-1.5 rounded-md text-secondary-text text-text-h hover:bg-secondary-hover transition-colors outline-none">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path></svg>
            </button>
        </div>

        <nav class="flex-1 space-y-2 overflow-y-auto custom-scroll overflow-x-hidden px-3">
            <div class="px-3 text-xs  text-text-m transition-all duration-fast overflow-hidden whitespace-nowrap"
                 :class="sidebarOpen ? 'max-h-6 opacity-100 mb-1' : 'max-h-0 opacity-0 mb-0'">Menu</div>
            
            <button @click="goToDashboard" 
                    type="button"
                    class="w-full flex items-center px-3 py-2.5 rounded-md text-base text-text-h transition-all border outline-none"
                    :class="currentView === 'dashboard' ? 'bg-secondary-selected  shadow-sm border-secondary-border' : 'border-transparent text-secondary-text  hover:bg-secondary-hover hover:text-text-h'"
                    :title="!sidebarOpen ? 'Dashboard' : ''">
                <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"></path></svg>
                <div class="overflow-hidden transition-all duration-fast ease-in-out whitespace-nowrap flex items-center"
                     :class="sidebarOpen ? 'max-w-[150px] ml-3 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>Dashboard</span>
                </div>
            </button>
            
            <button @click="appStore.setView('fleet')" 
                    type="button"
                    class="w-full flex items-center px-3 py-2.5 rounded-md text-base text-text-h transition-all border outline-none"
                    :class="currentView === 'fleet' ? 'bg-secondary-selected shadow-sm border-secondary-border' : 'border-transparent text-secondary-text  hover:bg-secondary-hover hover:text-text-h'"
                    :title="!sidebarOpen ? 'Fleet Management' : ''">
                <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"></path></svg>
                <div class="overflow-hidden transition-all duration-fast ease-in-out whitespace-nowrap flex items-center"
                     :class="sidebarOpen ? 'max-w-[150px] ml-3 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>Fleet Management</span>
                </div>
            </button>

            <button @click="appStore.setView('settings')" 
                    type="button"
                    class="w-full flex items-center px-3 py-2.5 rounded-md text-base text-text-h transition-all border outline-none"
                    :class="currentView === 'settings' ? 'bg-secondary-selected shadow-sm border-secondary-border' : 'border-transparent text-secondary-text  hover:bg-secondary-hover hover:text-text-h'"
                    :title="!sidebarOpen ? 'Settings' : ''">
                <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path></svg>
                <div class="overflow-hidden transition-all duration-fast ease-in-out whitespace-nowrap flex items-center"
                     :class="sidebarOpen ? 'max-w-[150px] ml-3 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>Settings</span>
                </div>
            </button>
        </nav>

        <div class="relative p-3 shrink-0 space-y-2">
            
            <div class="absolute top-0 left-0 w-full h-px flex justify-center">
                <div class="h-full bg-border-default transition-all duration-fast ease-in-out"
                     :class="sidebarOpen ? 'w-full opacity-100' : 'w-0 opacity-0'"></div>
            </div>
            
            <button @click="appStore.toggleArchive()" 
                    type="button"
                    class="w-full flex items-center justify-center py-2 px-3 rounded-md text-xs  transition-all border shadow-sm outline-none"
                    :class="viewingArchive ? 'bg-archive-bg text-archive-text border-archive-border hover:bg-archive-hover' : 'bg-secondary-main text-secondary-text border-secondary-border hover:bg-archive-bg hover:text-archive-text'">
            <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path v-if="viewingArchive" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 7V13M12 13L14 11M12 13L10 11M4 14H6.67452C7.1637 14 7.40829 14 7.63846 14.0553C7.84254 14.1043 8.03763 14.1851 8.21657 14.2947C8.4184 14.4184 8.59136 14.5914 8.93726 14.9373L9.06274 15.0627C9.40865 15.4086 9.5816 15.5816 9.78343 15.7053C9.96237 15.8149 10.1575 15.8957 10.3615 15.9447C10.5917 16 10.8363 16 11.3255 16H12.6745C13.1637 16 13.4083 16 13.6385 15.9447C13.8425 15.8957 14.0376 15.8149 14.2166 15.7053C14.4184 15.5816 14.5914 15.4086 14.9373 15.0627L15.0627 14.9373C15.4086 14.5914 15.5816 14.4184 15.7834 14.2947C15.9624 14.1851 16.1575 14.1043 16.3615 14.0553C16.5917 14 16.8363 14 17.3255 14H20M7.2 4H16.8C17.9201 4 18.4802 4 18.908 4.21799C19.2843 4.40973 19.5903 4.71569 19.782 5.09202C20 5.51984 20 6.07989 20 7.2V16.8C20 17.9201 20 18.4802 19.782 18.908C19.5903 19.2843 19.2843 19.5903 18.908 19.782C18.4802 20 17.9201 20 16.8 20H7.2C6.0799 20 5.51984 20 5.09202 19.782C4.71569 19.5903 4.40973 19.2843 4.21799 18.908C4 18.4802 4 17.9201 4 16.8V7.2C4 6.0799 4 5.51984 4.21799 5.09202C4.40973 4.71569 4.71569 4.40973 5.09202 4.21799C5.51984 4 6.0799 4 7.2 4Z" />
                <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
            </svg>
                <div class="overflow-hidden transition-all duration-fast ease-in-out whitespace-nowrap flex items-center"
                    :class="sidebarOpen ? 'max-w-[120px] ml-2 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>{{ viewingArchive ? 'Active Events' : 'Event Archive' }}</span>
                </div>
            </button>

            <button @click="clearLogs" 
                    type="button"
                    class="w-full flex items-center justify-center py-2 px-3 rounded-md text-xs  transition-all shadow-sm border outline-none bg-danger-bg text-danger-text border-danger-border hover:bg-danger-hover"
                    :title="!sidebarOpen ? 'Purge Logs' : ''">
                <svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                <div class="overflow-hidden transition-all duration-fast ease-in-out whitespace-nowrap flex items-center"
                    :class="sidebarOpen ? 'max-w-[120px] ml-2 opacity-100' : 'max-w-0 ml-0 opacity-0'">
                    <span>Purge System Logs</span>
                </div>
            </button>
        </div>
    </aside>
</template>