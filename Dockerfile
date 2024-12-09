# Bước 1: Sử dụng image Golang để xây dựng ứng dụng
FROM golang:1.23.4-alpine AS builder

# Thiết lập thư mục làm việc trong container
WORKDIR /app

# Sao chép go mod và go sum (nếu có) vào container
COPY go.mod go.sum ./

# Tải các dependencies của Go
RUN go mod tidy

# Sao chép mã nguồn của ứng dụng vào container
COPY . .

# Biên dịch ứng dụng Go
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o app .

# Bước 2: Chạy ứng dụng trên image distroless
FROM gcr.io/distroless/base-debian10

# Thiết lập thư mục làm việc trong container
WORKDIR /root/

# Sao chép tệp đã build từ bước trước vào container distroless
COPY --from=builder  /app/app .

# Chạy ứng dụng khi container bắt đầu
CMD ["/root/app"]
