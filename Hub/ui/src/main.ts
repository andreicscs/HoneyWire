import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './assets/style.css'
import App from './App.vue'
import router from './index'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
