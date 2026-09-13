import { createApp } from 'vue';
import App from './App.vue';
import './index.css';
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import zhCn from 'element-plus/es/locale/lang/zh-cn';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { triggerUpdateDialog } from './stores/update';

EventsOn('update_available', (version: string, name: string) => {
  triggerUpdateDialog(version, name);
});

const app = createApp(App);
app.use(ElementPlus, {
  locale: zhCn
});
app.mount('#root');
