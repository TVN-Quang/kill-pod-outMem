# Sử dụng image Node.js chính thức làm base image
FROM node:18-slim

# Tạo thư mục ứng dụng trong container
WORKDIR /usr/src/app

# Copy toàn bộ mã nguồn vào container
COPY . .

# Mở port 3000 (hoặc port bạn muốn ứng dụng chạy trên container)
EXPOSE 3000

# Chạy ứng dụng khi container khởi động
CMD ["node", "app.js"]
