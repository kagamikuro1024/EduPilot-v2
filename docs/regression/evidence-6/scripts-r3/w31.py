from lib import *
import unicodedata
q=json.load(open('/tmp/qc3e/p8q.json')); DQ=q['DQ']; A1=json.load(open('/tmp/qc3e/p8c.json'))['A']
one=psql("select vn_bigram_query('học')"); four=psql("select vn_bigram_query('canh bao hoc vu')")
cnt=psql(f"select count(*) from content_chunks where tsv @@ vn_bigram_query('canh bao hoc vu') and document_id='{DQ}'")
nfd=unicodedata.normalize('NFD','cảnh báo học vụ đặc biệt QC3NFD').replace("'","''"); nfc=unicodedata.normalize('NFC','cảnh báo học vụ đặc biệt QC3NFC')
psql(f"insert into content_chunks(document_id,course_ids,audience,ord,text) values ('{DQ}',array['{A1}'::uuid],'ALL',9001,'{nfd}')")
psql(f"insert into content_chunks(document_id,course_ids,audience,ord,text) values ('{DQ}',array['{A1}'::uuid],'ALL',9002,'{nfc}')")
r1=psql(f"select count(*) from content_chunks where ord in (9001,9002) and document_id='{DQ}' and tsv @@ vn_bigram_query('cảnh báo học vụ')")  # NFC query
nfdq=unicodedata.normalize('NFD','cảnh báo học vụ')
r2=psql(f"select count(*) from content_chunks where ord in (9001,9002) and document_id='{DQ}' and tsv @@ vn_bigram_query('{nfdq}')")
top=psql(f"select c.ord||'|'||left(c.text,80) from content_chunks c where c.document_id='{DQ}' and c.ord<9000 and c.tsv @@ vn_bigram_query('điều kiện bị cảnh báo học vụ là gì') order by ts_rank_cd(c.tsv,vn_bigram_query('điều kiện bị cảnh báo học vụ là gì')) desc limit 1")
has=psql(f"select count(*) from content_chunks where document_id='{DQ}' and ord<9000 and text ilike '%cảnh báo học vụ%'")
topord=top.split('|')[0]; ok_top=psql(f"select count(*) from content_chunks where document_id='{DQ}' and ord={topord} and text ilike '%cảnh báo học vụ%'")
psql(f"delete from content_chunks where document_id='{DQ}' and ord>=9000")
rec('TC-P8-01-46','a' not in ''  and one=="'hoc'" and '<->' in four and int(cnt)>=1 and r1=='2' and r2=='2' and ok_top=='1',f'vn_bigram_query("canh bao hoc vu") = {four} (OR các cặp âm tiết liền nhau, đúng SRS 195); một âm tiết «học» → {one}; tsv @@ khớp {cnt} đoạn của Quyche; chèn NFD + NFC rồi truy vấn bằng NFC → {r1}/2, bằng NFD → {r2}/2; câu «điều kiện bị cảnh báo học vụ là gì» → đoạn hạng 1 theo ts_rank_cd có «cảnh báo học vụ»={ok_top=="1"} (ord {topord}); đã xoá 2 đoạn thử')
