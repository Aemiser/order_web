export interface Order {
  id: string;
  name: string;
  phone: string;
  address: string;
  items: string;
  note: string;
}

// 声明window.go对象的类型
declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          OrderRecognition?: (context: string) => Promise<any>;
        };
      };
    };
  }
}
