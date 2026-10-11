# docling-serve cho đọc tài liệu (US-P8-01, proposals #7): image chính thức + dữ liệu OCR tiếng Việt (Tesseract `vie`, tessdata_fast, Apache-2.0).
# RapidOCR mặc định mất dấu tiếng Việt; EasyOCR tốn 4,3 GiB RAM — Tesseract `vie` giữ đúng dấu và thứ tự từ (docs/research/2026-10-10-docling-arm64.md).
FROM ghcr.io/docling-project/docling-serve-cpu:v1.36.0@sha256:225c8586e20d5d0fc6811a9e0e044fa602bcc4393f00389009bad42d6787b58f
COPY vie.traineddata /usr/share/tesseract/tessdata/vie.traineddata
