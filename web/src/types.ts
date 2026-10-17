export interface Order {
  id: string;
  name: string;
  phone: string;
  address: string;
  province: string; // 省份全称（34 省级行政区枚举；识别失败为空串）
  items: string;
  note: string;
}
