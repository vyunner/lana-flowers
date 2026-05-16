import { createApp } from 'vue'
import App from './App.vue'
import { initTelegram } from './telegram'
import './style.css'

initTelegram()

createApp(App).mount('#app')
