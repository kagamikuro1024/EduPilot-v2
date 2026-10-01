import asyncio
import sys
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from src.database.connection import get_db_pool, init_db_pool, close_db_pool

# Fix UnicodeEncodeError for Windows console
if sys.platform == "win32":
    sys.stdout.reconfigure(encoding='utf-8')

console = Console()

async def get_random_chunks(limit=10):
    """Lấy các chunk ngẫu nhiên từ database."""
    pool = get_db_pool()
    if not pool:
        console.print("[red]Lỗi: Không thể kết nối Database Pool.[/red]")
        return []
        
    async with pool.acquire() as conn:
        rows = await conn.fetch(
            """
            SELECT c.id, d.filename, c.content, c.metadata, d.source_uri
            FROM document_chunks c
            JOIN documents d ON c.document_id = d.id
            ORDER BY RANDOM()
            LIMIT $1
            """,
            limit
        )
        return rows

def check_quality(content):
    """Kiểm tra sơ bộ chất lượng text."""
    warnings = []
    if "\u00a0" in content:
        warnings.append("Phát hiện NBSP (khoảng trắng lạ)")
    if "|" in content and "---" not in content and content.count("|") > 4:
        # Giả định nếu có nhiều pipe mà không có separator thì có thể bảng bị vỡ
        pass 
    if len(content) < 50:
        warnings.append("Chunk quá ngắn")
    return warnings

async def main():
    console.print(Panel(
        "[bold blue]TIP-105: HUMAN-IN-THE-LOOP QUALITY VERIFICATION[/bold blue]", 
        subtitle="Phòng Thí Nghiệm Độc Hại AIK-024",
        expand=False
    ))

    try:
        # Khởi tạo DB Pool cho process này
        await init_db_pool()
        chunks = await get_random_chunks()
    except Exception as e:
        console.print(f"[red]Lỗi khi truy vấn DB: {e}[/red]")
        return
    finally:
        await close_db_pool()

    if not chunks:
        console.print("[yellow]Cảnh báo: Không tìm thấy dữ liệu trong bảng document_chunks.[/yellow]")
        return

    table = Table(title="10 Chunks Ngẫu Nhiên từ Hệ Thống", show_lines=True)
    table.add_column("ID", style="dim", width=8)
    table.add_column("Filename", style="cyan")
    table.add_column("Content Preview (200 chars)", width=60)
    table.add_column("Quality Check", style="bold red")

    for row in chunks:
        content = row['content']
        warnings = check_quality(content)
        warning_str = "\n".join(warnings) if warnings else "[green]OK[/green]"
        
        # Format content preview
        preview = content[:200].replace("\n", " ").strip() + "..."
        
        table.add_row(
            str(row['id']),
            row['filename'],
            preview,
            warning_str
        )

    console.print(table)

    console.print("\n[bold green]NHIỆM VỤ CỦA THỢ (BUILDER):[/bold green]")
    console.print("1. [ ] Kiểm tra bảng biểu trong DOCX có bị biến thành 'đống chữ nát' không.")
    console.print("2. [ ] Kiểm tra tiêu đề #, ## có đúng vị trí không.")
    console.print("3. [ ] Kiểm tra PDF có bị mất chữ hoặc Unicode sai lỗi dấu không.")
    console.print("4. [ ] Nếu mọi thứ OK, hãy đánh dấu hoàn thành TIP-105.")

if __name__ == "__main__":
    asyncio.run(main())
