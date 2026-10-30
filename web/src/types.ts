export interface Order {
  id: string;
  name: string;
  phone: string;
  address: string;
  province: string; // 省份全称（34 省级行政区枚举；识别失败为空串）
  items: string;
  note: string;
}

// 吴饮（吾饮良品）订单行：在 Order 基础上增加仓库标签与群聊发送人
export interface WuyinOrder {
  id: string; // 预览行用 VID，历史订单行用数据库 id
  dbId: number; // 数据库主键；预览行为 0
  name: string;
  phone: string;
  address: string;
  province: string;
  items: string; // 【香水柠檬1件】形态
  note: string;
  tag: string;
  sender: string;
  createdAt?: string;
}
