"""
Module: src.database.models
Mô tả: Định nghĩa Pydantic models cho các bảng trong Database.
       Các model này đóng vai trò Data Transfer Object (DTO) — đại diện
       cho cấu trúc dữ liệu khi truyền giữa các tầng trong ứng dụng.

Tại sao dùng Pydantic thay vì SQLAlchemy ORM?
    - FastAPI tích hợp sẵn Pydantic để validate request/response.
    - Dự án dùng asyncpg (raw async SQL) nên không cần ORM nặng.
    - Pydantic model nhẹ, nhanh, phù hợp với kiến trúc async.

Các bảng:
    1. documents      - Bảng mục lục tài liệu (file PDF, slide...)
    2. document_chunks - Bảng lưu từng đoạn văn bản đã cắt nhỏ + vector embedding
"""

import uuid
from datetime import datetime
from enum import Enum
from typing import Any

from pydantic import BaseModel, Field


# -----------------------------------------------------------------------
# Enum trạng thái xử lý tài liệu
# Giải thích:
#   - PROCESSING: Tài liệu đang được hệ thống xử lý (cắt chunk, tạo embedding)
#   - READY:      Xử lý xong, tài liệu sẵn sàng để tìm kiếm (search)
#   - FAILED:     Xử lý thất bại (file lỗi, API embedding timeout, v.v.)
# -----------------------------------------------------------------------
class DocumentStatus(str, Enum):
    PROCESSING = "PROCESSING"
    READY = "READY"
    FAILED = "FAILED"


class DocumentType(str, Enum):
    """Aligned with PostgreSQL `documents.document_type` / Java `DocumentType`."""

    COURSE_MATERIAL = "COURSE_MATERIAL"
    REGULATION = "REGULATION"


# -----------------------------------------------------------------------
# Model: Document (Bảng documents)
# Vai trò: Mục lục - lưu thông tin tổng quan của 1 tài liệu.
# VD: Một file "Bai3_VongLap.pdf" thuộc môn "Nhập môn CNTT", tuần 3.
# -----------------------------------------------------------------------
class Document(BaseModel):
    """
    Schema cho bảng `documents`.
    Mỗi record đại diện cho 1 file tài liệu đã upload vào hệ thống.
    """
    # UUID tự sinh, đảm bảo unique trên mọi hệ thống phân tán
    id: uuid.UUID = Field(default_factory=uuid.uuid4)

    # Đường dẫn hoặc URI đến tài liệu gốc
    source_uri: str

    # Tên file hiển thị (VD: "Bai3_VongLap.pdf")
    filename: str

    # Metadata dạng JSON linh hoạt
    metadata: dict[str, Any] = Field(default_factory=dict)

    # Mã băm nội dung để chống duplicate
    content_hash: str

    # Phiên bản tài liệu
    version: int = 1

    # Trạng thái xử lý hiện tại của tài liệu
    status: DocumentStatus = DocumentStatus.PROCESSING

    # Phân loại RAG: tài liệu môn vs quy chế (cột `document_type` sau migration V21)
    document_type: DocumentType = DocumentType.COURSE_MATERIAL

    # Thời điểm tài liệu được thêm vào hệ thống
    created_at: datetime = Field(default_factory=datetime.utcnow)


# -----------------------------------------------------------------------
# Model: DocumentChunk (Bảng document_chunks)
# Vai trò: Lưu từng "đoạn văn bản nhỏ" (chunk) của tài liệu gốc.
#
# Tại sao phải cắt nhỏ (chunking)?
#   - Mô hình embedding (VD: OpenAI text-embedding-3-small) có giới hạn
#     số token đầu vào (~8191 tokens).
#   - Khi tìm kiếm, so sánh vector của câu hỏi với vector của đoạn nhỏ
#     cho kết quả chính xác hơn so sánh với toàn bộ file.
#   - Mỗi chunk thường có 500-1000 ký tự, overlap 100-200 ký tự để
#     không bị mất ngữ cảnh ở ranh giới cắt.
# -----------------------------------------------------------------------
class DocumentChunk(BaseModel):
    """
    Schema cho bảng `document_chunks`.
    Mỗi record là 1 đoạn văn bản đã cắt nhỏ từ tài liệu gốc,
    kèm theo vector embedding 1536 chiều.
    """
    # UUID cho mỗi chunk
    id: uuid.UUID = Field(default_factory=uuid.uuid4)

    # Foreign Key liên kết về tài liệu gốc trong bảng `documents`
    # Khi xóa document gốc → tất cả chunk liên quan cũng bị xóa (CASCADE)
    document_id: uuid.UUID

    # Nội dung văn bản của đoạn chunk này
    # VD: "Vòng lặp For trong Python dùng để duyệt qua các phần tử..."
    content: str

    # Vector embedding 1536 chiều (tương thích OpenAI text-embedding-3-small)
    # Đây là mảng số thực biểu diễn "ý nghĩa ngữ nghĩa" của đoạn văn bản.
    # Khi SV hỏi câu hỏi, hệ thống sẽ chuyển câu hỏi thành vector,
    # rồi dùng phép tính khoảng cách cosine để tìm chunk giống nhất.
    # Lưu ý: Trường này là list[float] trong Python, nhưng trong DB
    # sẽ được lưu ở kiểu VECTOR(1536) nhờ extension pgvector.
    embedding: list[float] | None = None

    # Thứ tự của chunk trong tài liệu gốc (bắt đầu từ 0)
    chunk_index: int

    # Mã băm nội dung của đoạn chunk
    content_hash: str
