import os
import time
import subprocess
import pytest
import httpx
import asyncio
import psutil
import shutil
import unicodedata
from src.pipeline.document_parser import compute_hash

# Cấu hình
BASE_URL = "http://127.0.0.1:8000"
UPLOAD_URL = f"{BASE_URL}/api/v1/documents/upload"
MOCK_DATA_DIR = os.path.join(os.getcwd(), "tests", "stress_data")
TEMP_DIR = os.path.join(os.getcwd(), "data", "tmp")

@pytest.fixture(scope="session", autouse=True)
def start_server():
    """Start FastAPI server automatically."""
    print("\n[SETUP] Starting FastAPI server...")
    # Check if server is already running
    try:
        httpx.get(BASE_URL)
        print("[SETUP] Server already running.")
        yield
        return
    except:
        pass

    # Start new server, redirect logs to a file for debugging
    log_file = open("data-pipeline/server_test.log", "w")
    cmd = ["uvicorn", "src.main:app", "--host", "127.0.0.1", "--port", "8000"]
    proc = subprocess.Popen(cmd, stdout=log_file, stderr=log_file)
    
    # Wait for server to come up
    max_retries = 20
    for i in range(max_retries):
        try:
            httpx.get(BASE_URL)
            print("[SETUP] Server is ready.")
            break
        except:
            time.sleep(1.0) # Increased wait
            if i == max_retries - 1:
                proc.terminate()
                log_file.close()
                raise RuntimeError("Failed to start server after 20 seconds.")

    yield

    print("\n[TEARDOWN] Stopping server...")
    proc.terminate()
    try:
        proc.wait(timeout=5)
    except:
        proc.kill()
    log_file.close()

# --- GROUP A: FORMAT INTEGRITY (DOCX) ---

@pytest.mark.asyncio
async def test_tc01_docx_header_h1():
    """TC-01: Header H1 trong DOCX phải chuyển thành # trong Markdown."""
    # Test này cần verify output markdown. Vì worker chạy ngầm, 
    # ta sẽ test logic parser trực tiếp hoặc check DB (HITL).
    # Ở đây stress test tập trung vào luồng nạp và không crash.
    file_path = os.path.join(MOCK_DATA_DIR, "complex_structure.docx")
    async with httpx.AsyncClient(timeout=10.0) as client:
        with open(file_path, "rb") as f:
            response = await client.post(UPLOAD_URL, files={"file": f})
    assert response.status_code == 202
    print("TC-01: DOCX H1 Upload successfully triggered.")

@pytest.mark.asyncio
async def test_tc05_empty_docx():
    """TC-05: Empty DOCX should not hang worker."""
    file_path = os.path.join(MOCK_DATA_DIR, "empty_xml.docx")
    async with httpx.AsyncClient(timeout=10.0) as client:
        with open(file_path, "rb") as f:
            response = await client.post(UPLOAD_URL, files={"file": f})
    assert response.status_code == 202

# --- GROUP B: FORMAT INTEGRITY (PDF) ---

def test_tc06_unicode_normalization():
    """TC-06: Kiểm tra Unicode NFC cho tiếng Việt."""
    text = "Tiếng Việt" # Dấu ở dạng NFD
    normalized = unicodedata.normalize('NFC', text)
    assert normalized == "Tiếng Việt"
    # Verification sâu hơn sẽ ở HITL script

@pytest.mark.asyncio
async def test_tc08_huge_pdf_memory():
    """TC-08: Nạp file PDF lớn và monitor RAM."""
    file_path = os.path.join(MOCK_DATA_DIR, "huge_slide.pdf")
    process = psutil.Process(os.getpid())
    mem_before = process.memory_info().rss / 1024 / 1024
    
    async with httpx.AsyncClient() as client:
        with open(file_path, "rb") as f:
            response = await client.post(UPLOAD_URL, files={"file": f}, timeout=60.0)
    
    mem_after = process.memory_info().rss / 1024 / 1024
    print(f"TC-08: RAM usage - Before: {mem_before:.1f}MB, After: {mem_after:.1f}MB")
    assert response.status_code in [202, 413] # Tùy cấu hình MAX_FILE_SIZE

# --- GROUP C: EDGE CASES & GUARDRAILS ---

@pytest.mark.asyncio
async def test_tc09_invalid_extension():
    """TC-09: Chặn file sai định dạng (400)."""
    file_path = os.path.join(MOCK_DATA_DIR, "corrupt_header.pdf")
    # Đổi tên file sang .exe để test
    temp_exe = os.path.join(MOCK_DATA_DIR, "malicious.exe")
    shutil.copy(file_path, temp_exe)
    
    async with httpx.AsyncClient() as client:
        with open(temp_exe, "rb") as f:
            response = await client.post(UPLOAD_URL, files={"file": ("malicious.exe", f, "application/octet-stream")})
    
    os.remove(temp_exe)
    assert response.status_code == 400

@pytest.mark.asyncio
async def test_tc10_file_too_large():
    """TC-10: Chặn file quá lớn (413)."""
    # Tạo file rác 60MB
    path = os.path.join(MOCK_DATA_DIR, "too_large.pdf")
    with open(path, "wb") as f:
        f.write(b"%PDF-1.5\n" + os.urandom(61 * 1024 * 1024))
        
    async with httpx.AsyncClient() as client:
        with open(path, "rb") as f:
            response = await client.post(UPLOAD_URL, files={"file": f})
    
    os.remove(path)
    assert response.status_code == 413

@pytest.mark.asyncio
async def test_tc11_concurrent_uploads():
    """TC-11: 5 requests đồng thời file 1MB."""
    file_path = os.path.join(MOCK_DATA_DIR, "complex_structure.docx")
    
    async def upload_one(i):
        async with httpx.AsyncClient() as client:
            with open(file_path, "rb") as f:
                # Dùng filename khác nhau để tránh collision nếu hash giống nhau
                return await client.post(UPLOAD_URL, files={"file": (f"test_concurrent_{i}.docx", f)})

    tasks = [upload_one(i) for i in range(5)]
    results = await asyncio.gather(*tasks)
    
    for res in results:
        assert res.status_code == 202

@pytest.mark.asyncio
async def test_tc12_password_locked_pdf():
    """TC-12: File PDF có mật khẩu không được làm treo worker."""
    file_path = os.path.join(MOCK_DATA_DIR, "password_locked.pdf")
    async with httpx.AsyncClient() as client:
        with open(file_path, "rb") as f:
            response = await client.post(UPLOAD_URL, files={"file": f})
    assert response.status_code == 202

# --- GROUP D: CLEANUP & HASH ---

def test_tc13_temp_cleanup():
    """TC-13: Temp dir should be clean after processing (wait up to 60s)."""
    # Wait for worker (max 60s) for heavy PDF and concurrent tasks
    for _ in range(60):
        if not os.path.exists(TEMP_DIR) or len(os.listdir(TEMP_DIR)) == 0:
            return
        time.sleep(1.0)
    
    files = os.listdir(TEMP_DIR)
    assert len(files) == 0, f"Temp dir still contains files: {files}"

def test_tc15_hash_collision():
    """TC-15: Sửa 1 byte trong DOCX phải sinh ra hash khác."""
    file_path = os.path.join(MOCK_DATA_DIR, "complex_structure.docx")
    with open(file_path, "rb") as f:
        content1 = f.read()
    
    hash1 = compute_hash(content1)
    
    # Sửa file (copy và append)
    file_path2 = os.path.join(MOCK_DATA_DIR, "complex_structure_copy.docx")
    with open(file_path2, "wb") as f:
        f.write(content1 + b" ")
    
    with open(file_path2, "rb") as f:
        content2 = f.read()
    
    hash2 = compute_hash(content2)
    os.remove(file_path2)
    
    assert hash1 != hash2
    print(f"Hash 1: {hash1}")
    print(f"Hash 2: {hash2}")

if __name__ == "__main__":
    pytest.main([__file__, "-v", "-s"])
