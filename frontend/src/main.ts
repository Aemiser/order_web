import { createApp } from 'vue';
import App from './App.vue';
import './index.css';
import ElementPlus from 'element-plus';
import 'element-plus/dist/index.css';
import zhCn from 'element-plus/es/locale/lang/zh-cn';

const app = createApp(App);
app.use(ElementPlus, {
  locale: zhCn
});
app.mount('#root');
