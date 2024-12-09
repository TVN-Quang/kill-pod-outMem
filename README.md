Các vấn đề:
- Trước khi thực hiện kiểm tra xem có đang có pod nào bị lỗi không.
- Khi tăng pod mới và xóa pod cũ dùng cơ chế nào phù hợp. Nếu tăng replica của deployment thì sẽ bị down nếu HPA thấy dư resource cần.
- Nếu tăng minRep trong hpa thì sẽ tăng 1 pod mới và sau khi xóa pod cũ sẽ giảm minReplica về lại, nhưng làm sao để biết pod mới đã lên chưa?
- Phải dùng label app để lấy các pods, hpa.


nếu thất bại thì phải cho vào hàng đợi + gửi mail.
Thêm lưu trữ để lưu lại các pod mà lỗi nếu số lượng nhiều thì không thực hiện nữa. Mà alert critical.
Lưu trữ các pod đã được tìm thấy trước đó để tìm kiếm 
Node có đủ resource để thêm pod không?